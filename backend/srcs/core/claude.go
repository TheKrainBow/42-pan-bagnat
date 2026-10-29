package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"backend/database"
	"backend/websocket"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	eventtypes "github.com/docker/docker/api/types/events"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockermount "github.com/docker/docker/api/types/mount"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
)

// The module "Claude" tab runs the official `claude` CLI in a per-module
// container (image: module-claude/), inside tmux so that it keeps working when
// no browser is attached. Each user logs in with their own Claude account: the
// credentials and transcripts live in a per-user home volume that PB never reads.
//
// Per container, the backend serves two unix sockets in a directory next to
// the module repos (bind-mounted on /run/pb): the scoped docker proxy and the
// "pb" socket used by the Claude Code hooks (status reporting) and by the `pb`
// command (deploys run by PB, so that they are logged on the module). Both are
// re-created after a backend restart, the container is left untouched.

const (
	claudeLabel        = "pb.claude"
	claudeModuleLabel  = "pb.module"
	claudeModIDLabel   = "pb.module.id"
	claudeUserLabel    = "pb.user"
	claudeHome         = "/home/claude"
	claudeRunDir       = "/run/pb"
	claudeDirName      = ".pb-claude"
	claudeHookSockName = "pb.sock"
	claudeNetwork      = "pan-bagnat-claude"
	defaultClaudeImage = "pan-bagnat-module-claude:latest"
	// A session stuck in "starting" with no container (backend died while creating it).
	claudePendingTimeout = 5 * time.Minute
	claudePendingPrefix  = "pending:"
)

var (
	ErrClaudeLimitReached = errors.New("too many claude containers running")
	ErrClaudeModuleBusy   = database.ErrClaudeModuleBusy
	ErrClaudeWorking      = errors.New("claude is working, force is required")
	ErrClaudeNotActive    = errors.New("claude session is not running")
	ErrClaudeNotResumable = errors.New("claude session cannot be resumed")

	claudeSessionIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)
)

type claudeConfig struct {
	image         string
	idleTTL       time.Duration
	maxWork       time.Duration
	retention     time.Duration
	memory        int64
	nanoCPUs      int64
	pidsLimit     int64
	maxContainers int
}

func envDuration(key string, def time.Duration, unit time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return d
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
		return time.Duration(n * float64(unit))
	}
	return def
}

func envNumber(key string, def float64) float64 {
	if n, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(key)), 64); err == nil && n > 0 {
		return n
	}
	return def
}

func loadClaudeConfig() claudeConfig {
	img := strings.TrimSpace(os.Getenv("MODULE_CLAUDE_IMAGE"))
	if img == "" {
		img = defaultClaudeImage
	}
	return claudeConfig{
		image:         img,
		idleTTL:       envDuration("CLAUDE_IDLE_TTL", 30*time.Minute, time.Second),
		maxWork:       envDuration("CLAUDE_MAX_WORK_SECONDS", 12*time.Hour, time.Second),
		retention:     envDuration("CLAUDE_SESSION_RETENTION_DAYS", 30*24*time.Hour, 24*time.Hour),
		memory:        int64(envNumber("CLAUDE_MEMORY_MB", 2048)) << 20,
		nanoCPUs:      int64(envNumber("CLAUDE_CPUS", 2) * 1e9),
		pidsLimit:     int64(envNumber("CLAUDE_PIDS_LIMIT", 512)),
		maxContainers: int(envNumber("CLAUDE_MAX_CONTAINERS", 5)),
	}
}

var claudeCfg = loadClaudeConfig()

// claudeRuntime is the backend side of a running Claude container.
type claudeRuntime struct {
	moduleID    string
	slug        string
	containerID string
	token       string
	user        string // uid:gid the container runs as
	dir         string // socket directory, as seen by this backend
	proxy       *scopedDockerProxy
	hook        *claudeHookServer

	mu       sync.Mutex
	attached int
}

var (
	claudeMu       sync.Mutex
	claudeRuntimes = map[string]*claudeRuntime{} // by module ID
	// claudeModuleOps serializes start/stop/reap of a given module.
	claudeModuleOps sync.Map
)

func claudeModuleLock(moduleID string) func() {
	v, _ := claudeModuleOps.LoadOrStore(moduleID, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func getClaudeRuntime(moduleID string) *claudeRuntime {
	claudeMu.Lock()
	defer claudeMu.Unlock()
	return claudeRuntimes[moduleID]
}

func (rt *claudeRuntime) close() {
	if rt.hook != nil {
		rt.hook.Close()
	}
	if rt.proxy != nil {
		rt.proxy.Close()
	}
}

// claudeDirs returns the module's socket directory as seen by this backend and by the docker host.
func claudeDirs(module Module) (local, host string, err error) {
	containerRoot, err := ModuleRepoRoot(module)
	if err != nil {
		return "", "", err
	}
	hostRoot, err := HostModuleRepoRoot(module)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(filepath.Dir(containerRoot), claudeDirName, module.Slug),
		filepath.Join(filepath.Dir(hostRoot), claudeDirName, module.Slug), nil
}

func claudeContainerName(module Module, userID string) string {
	return "pb-claude-" + module.Slug + "-" + strings.ToLower(userID)
}

func claudeHomeVolume(userID string) string {
	return "pb-claude-home-" + strings.ToLower(userID)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// repoOwner returns the uid/gid owning the module repo: the container runs as
// that user so that every file Claude creates is usable by git, the IDE and PB.
func repoOwner(root string) (int, int, error) {
	fi, err := os.Stat(root)
	if err != nil {
		return 0, 0, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, errors.New("cannot read repository owner")
	}
	return int(st.Uid), int(st.Gid), nil
}

func parseUIDGID(s string) (int, int) {
	u, g, _ := strings.Cut(s, ":")
	uid, _ := strconv.Atoi(u)
	gid, _ := strconv.Atoi(g)
	return uid, gid
}

// startClaudeRuntime prepares the socket directory (ssh key, docker proxy, pb
// socket) of a module. The directory is kept as is when it already exists: it is
// bind-mounted in a running container after a backend restart.
func startClaudeRuntime(module Module, containerID, token string, uid, gid int) (*claudeRuntime, error) {
	dir, _, err := claudeDirs(module)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "ssh"), 0o700); err != nil {
		return nil, err
	}
	_ = os.Chown(dir, uid, gid)
	_ = os.Chown(filepath.Join(dir, "ssh"), uid, gid)
	if err := writeClaudeSSHFiles(module, filepath.Join(dir, "ssh"), uid, gid); err != nil {
		return nil, err
	}
	rt := &claudeRuntime{
		moduleID:    module.ID,
		slug:        module.Slug,
		containerID: containerID,
		token:       token,
		user:        fmt.Sprintf("%d:%d", uid, gid),
		dir:         dir,
	}
	rt.proxy, err = startScopedDockerProxyWith(module.Slug, dir, dockerProxyOptions{keepDir: true, socketUID: uid, socketGID: gid})
	if err != nil {
		return nil, fmt.Errorf("start docker proxy: %w", err)
	}
	rt.hook, err = startClaudeHookServer(rt, filepath.Join(dir, claudeHookSockName), uid, gid)
	if err != nil {
		rt.proxy.Close()
		return nil, fmt.Errorf("start pb socket: %w", err)
	}
	return rt, nil
}

// writeClaudeSSHFiles gives the container the module deploy key (read-only for
// its user) and PB's known_hosts, for git fetch/pull/push.
func writeClaudeSSHFiles(module Module, dir string, uid, gid int) error {
	key := filepath.Join(dir, "id_key")
	_ = os.Remove(key)
	if strings.TrimSpace(module.SSHPrivateKey) != "" {
		if err := os.WriteFile(key, []byte(strings.TrimSpace(module.SSHPrivateKey)+"\n"), 0o400); err != nil {
			return err
		}
		_ = os.Chown(key, uid, gid)
	}
	known := filepath.Join(dir, "known_hosts")
	data := []byte{}
	if src := strings.TrimSpace(os.Getenv("GIT_SSH_KNOWN_HOSTS")); src != "" {
		data, _ = os.ReadFile(src)
	}
	if err := os.WriteFile(known, data, 0o444); err != nil {
		return err
	}
	_ = os.Chown(known, uid, gid)
	return nil
}

func ensureClaudeNetwork(ctx context.Context) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	if _, err := cli.NetworkInspect(ctx, claudeNetwork, dockernetwork.InspectOptions{}); err == nil {
		return nil
	}
	// Outbound access only (claude.ai, api.anthropic.com, git remote): no PB
	// network, and containers on it cannot talk to each other.
	_, err = cli.NetworkCreate(ctx, claudeNetwork, dockernetwork.CreateOptions{
		Driver:  "bridge",
		Options: map[string]string{"com.docker.network.bridge.enable_icc": "false"},
		Labels:  map[string]string{claudeLabel: "1"},
	})
	if err != nil && errdefs.IsConflict(err) {
		return nil
	}
	return err
}

func gitIdentity(user User) (string, string) {
	name := user.FtLogin
	email := strings.TrimSpace(user.Email)
	if email == "" {
		email = user.FtLogin + "@users.noreply.panbagnat"
	}
	return name, email
}

// ClaudeOccupant describes who currently holds a module's Claude container.
type ClaudeOccupant struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	UserLogin string    `json:"user_login"`
	Status    string    `json:"status"`
	Since     time.Time `json:"since"`
}

func ClaudeModuleOccupant(moduleID string) (*ClaudeOccupant, error) {
	s, err := database.GetActiveClaudeSessionByModule(moduleID)
	if err != nil || s == nil {
		return nil, err
	}
	return &ClaudeOccupant{SessionID: s.ID, UserID: s.UserID, UserLogin: s.UserLogin, Status: s.Status, Since: s.StartedAt}, nil
}

func ListClaudeSessions(moduleID, userID string) ([]database.ClaudeSession, error) {
	return database.ListClaudeSessions(moduleID, userID)
}

func GetClaudeSession(id string) (*database.ClaudeSession, error) {
	return database.GetClaudeSession(id)
}

// StartClaudeSession starts a container for a new session, or for resumeID (a
// previous session of the same user on this module).
func StartClaudeSession(ctx context.Context, user User, module Module, resumeID string) (*database.ClaudeSession, error) {
	containerRoot, err := ModuleRepoRoot(module)
	if err != nil {
		return nil, fmt.Errorf("module repository is not available: %w", err)
	}
	hostRoot, err := HostModuleRepoRoot(module)
	if err != nil {
		return nil, err
	}
	uid, gid, err := repoOwner(containerRoot)
	if err != nil {
		return nil, err
	}
	if err := ensureShellImage(ctx, claudeCfg.image); err != nil {
		return nil, fmt.Errorf("claude image unavailable: %w", err)
	}
	if err := ensureClaudeNetwork(ctx); err != nil {
		return nil, fmt.Errorf("claude network: %w", err)
	}

	unlock := claudeModuleLock(module.ID)
	defer unlock()

	active, err := database.ListActiveClaudeSessions()
	if err != nil {
		return nil, err
	}
	if len(active) >= claudeCfg.maxContainers {
		return nil, ErrClaudeLimitReached
	}

	// Take the module lock in the database first (unique index), then create the container.
	var sessionID, resumeClaudeID string
	if resumeID != "" {
		prev, err := database.GetClaudeSession(resumeID)
		if err != nil {
			return nil, err
		}
		if prev == nil || prev.ModuleID != module.ID || prev.UserID != user.ID || !claudeSessionIDRe.MatchString(prev.ClaudeSessionID) {
			return nil, ErrClaudeNotResumable
		}
		sessionID, resumeClaudeID = prev.ID, prev.ClaudeSessionID
		if err := database.ReopenClaudeSession(sessionID, claudePendingPrefix+sessionID); err != nil {
			return nil, err
		}
	} else {
		sessionID, err = GenerateULID(ClaudeSessionKind)
		if err != nil {
			return nil, err
		}
		if err := database.CreateClaudeSession(sessionID, module.ID, user.ID, claudePendingPrefix+sessionID); err != nil {
			return nil, err
		}
	}
	fail := func(err error) (*database.ClaudeSession, error) {
		if resumeID == "" {
			_ = database.DeleteClaudeSession(sessionID)
		} else {
			_ = database.EndClaudeSession(sessionID, database.ClaudeAborted)
		}
		return nil, err
	}

	token, err := randomHex(32)
	if err != nil {
		return fail(err)
	}
	cli, err := getDockerClient()
	if err != nil {
		return fail(err)
	}
	name := claudeContainerName(module, user.ID)
	// A leftover container with the same name has no session anymore (the lock was free).
	_ = cli.ContainerRemove(ctx, name, dockercontainer.RemoveOptions{Force: true})

	rt, err := startClaudeRuntime(module, "", token, uid, gid)
	if err != nil {
		return fail(err)
	}
	_, hostDir, _ := claudeDirs(module)
	gitName, gitEmail := gitIdentity(user)

	env := []string{
		"TERM=xterm-256color",
		"PB_ENV=production",
		"PB_MODULE=" + module.Slug,
		"PB_WORKDIR=" + hostRoot,
		"PB_HOOK_TOKEN=" + token,
		"PB_RESUME_ID=" + resumeClaudeID,
		"DOCKER_HOST=unix://" + claudeRunDir + "/" + shellProxySockName,
		"COMPOSE_PROJECT_NAME=" + module.Slug,
		"GIT_SSH_COMMAND=ssh -i " + claudeRunDir + "/ssh/id_key -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes -o UserKnownHostsFile=" + claudeRunDir + "/ssh/known_hosts",
		"GIT_AUTHOR_NAME=" + gitName, "GIT_AUTHOR_EMAIL=" + gitEmail,
		"GIT_COMMITTER_NAME=" + gitName, "GIT_COMMITTER_EMAIL=" + gitEmail,
		"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=safe.directory", "GIT_CONFIG_VALUE_0=" + hostRoot,
	}
	cfg := &dockercontainer.Config{
		Image:      claudeCfg.image,
		User:       rt.user,
		WorkingDir: hostRoot,
		Env:        env,
		Labels: map[string]string{
			claudeLabel:       "1",
			claudeModuleLabel: module.Slug,
			claudeModIDLabel:  module.ID,
			claudeUserLabel:   user.ID,
		},
	}
	// Everything dropped; a root repo owner (PB runs as root) keeps only what
	// it needs to edit files owned by other users in the repo.
	var capAdd []string
	if uid == 0 {
		capAdd = []string{"CHOWN", "DAC_OVERRIDE", "FOWNER", "FSETID", "KILL"}
	}
	init := true
	hostCfg := &dockercontainer.HostConfig{
		// Same absolute path as on the host: compose run from the container resolves
		// relative bind mounts client-side, and claude stores sessions per directory.
		Mounts: []dockermount.Mount{
			{Type: dockermount.TypeBind, Source: hostRoot, Target: hostRoot},
			{Type: dockermount.TypeBind, Source: hostDir, Target: claudeRunDir},
			{Type: dockermount.TypeVolume, Source: claudeHomeVolume(user.ID), Target: claudeHome},
		},
		Init:          &init,
		CapDrop:       []string{"ALL"},
		CapAdd:        capAdd,
		SecurityOpt:   []string{"no-new-privileges"},
		NetworkMode:   dockercontainer.NetworkMode(claudeNetwork),
		RestartPolicy: dockercontainer.RestartPolicy{Name: dockercontainer.RestartPolicyDisabled},
		Resources: dockercontainer.Resources{
			Memory:    claudeCfg.memory,
			NanoCPUs:  claudeCfg.nanoCPUs,
			PidsLimit: ptrInt64(claudeCfg.pidsLimit),
		},
	}
	created, err := cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, name)
	if err != nil {
		rt.close()
		return fail(fmt.Errorf("create claude container: %w", err))
	}
	rt.containerID = created.ID
	if err := database.SetClaudeSessionContainer(sessionID, created.ID); err != nil {
		_ = cli.ContainerRemove(context.Background(), created.ID, dockercontainer.RemoveOptions{Force: true})
		rt.close()
		return fail(err)
	}
	claudeMu.Lock()
	claudeRuntimes[module.ID] = rt
	claudeMu.Unlock()
	if err := cli.ContainerStart(ctx, created.ID, dockercontainer.StartOptions{}); err != nil {
		_ = finishClaudeContainer(module.ID, created.ID, database.ClaudeAborted, claudeEnd{})
		return nil, fmt.Errorf("start claude container: %w", err)
	}
	log.Printf("[claude] %s started a session on module %s (container %.12s)", user.FtLogin, module.Slug, created.ID)
	sess, err := database.GetClaudeSession(sessionID)
	if err == nil && sess != nil {
		emitClaudeStatus(*sess)
	}
	return sess, err
}

// claudeEnd tells finishClaudeContainer how a session ends.
type claudeEnd struct {
	// unattended: an end the owner did not ask for (max work time, crash). If
	// Claude was in the middle of a task, the owner is told and the result is
	// flagged as unseen.
	unattended bool
	// graceful: let claude exit cleanly (SIGTERM, then SIGKILL after
	// claudeStopGrace) instead of destroying the container right away.
	graceful bool
}

const claudeStopGrace = 10 * time.Second

// removeClaudeContainer destroys a container; a container already gone is not an error.
func removeClaudeContainer(containerID string, graceful bool) error {
	if containerID == "" || strings.HasPrefix(containerID, claudePendingPrefix) {
		return nil
	}
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), claudeStopGrace+30*time.Second)
	defer cancel()
	if graceful {
		grace := int(claudeStopGrace.Seconds())
		_ = cli.ContainerStop(ctx, containerID, dockercontainer.StopOptions{Timeout: &grace})
	}
	err = cli.ContainerRemove(ctx, containerID, dockercontainer.RemoveOptions{Force: true})
	if err != nil && !errdefs.IsNotFound(err) {
		return err
	}
	return nil
}

// finishClaudeContainer removes the module's container and its runtime, and
// ends the session holding it with status. Callers hold claudeModuleLock.
// If the container cannot be removed, the session keeps the module lock (a
// claude may still be running on the repo) and the error is returned.
func finishClaudeContainer(moduleID, containerID, status string, end claudeEnd) error {
	if err := removeClaudeContainer(containerID, end.graceful); err != nil {
		log.Printf("[claude] remove container %.12s of module %s: %v", containerID, moduleID, err)
		return fmt.Errorf("remove claude container: %w", err)
	}
	claudeMu.Lock()
	rt := claudeRuntimes[moduleID]
	if rt != nil && rt.containerID == containerID {
		delete(claudeRuntimes, moduleID)
	} else {
		rt = nil
	}
	claudeMu.Unlock()
	if rt != nil {
		rt.close()
	}

	sess, _ := database.GetActiveClaudeSessionByContainer(containerID)
	if sess == nil {
		return nil
	}
	_ = database.EndClaudeSession(sess.ID, status)
	interrupted := end.unattended && (sess.Status == database.ClaudeWorking || sess.Status == database.ClaudeWaitingInput)
	if interrupted {
		_ = database.SetClaudeSessionUnseen(sess.ID, true)
	}
	if dir := claudeDirFor(sess.ModuleID); dir != "" {
		_ = os.RemoveAll(dir)
	}
	if ended, _ := database.GetClaudeSession(sess.ID); ended != nil {
		emitClaudeStatus(*ended)
		if interrupted {
			notifyClaudeUser(*ended, "error", "Claude's session ended during a task ("+status+").")
		}
	}
	log.Printf("[claude] session %s on module %s ended (%s)", sess.ID, sess.ModuleID, status)
	return nil
}

func claudeDirFor(moduleID string) string {
	module, err := GetModule(moduleID)
	if err != nil || module.ID == "" {
		return ""
	}
	dir, _, err := claudeDirs(module)
	if err != nil {
		return ""
	}
	return dir
}

// StopClaudeSession ends the session's container. A working session needs
// force. kill destroys the container immediately (for a session that hangs);
// otherwise claude is given claudeStopGrace to exit cleanly.
func StopClaudeSession(sess database.ClaudeSession, force, kill bool) error {
	unlock := claudeModuleLock(sess.ModuleID)
	defer unlock()
	cur, err := database.GetClaudeSession(sess.ID)
	if err != nil {
		return err
	}
	if cur == nil || !cur.Active() {
		return ErrClaudeNotActive
	}
	if !force && (cur.Status == database.ClaudeWorking || cur.Status == database.ClaudeWaitingInput) {
		return ErrClaudeWorking
	}
	return finishClaudeContainer(cur.ModuleID, cur.ContainerID, database.ClaudeStopped, claudeEnd{graceful: !kill})
}

// DeleteClaudeSession stops the session if needed, then deletes its row and its
// transcript in the user's home volume.
func DeleteClaudeSession(sess database.ClaudeSession, force bool) error {
	if sess.Active() {
		if err := StopClaudeSession(sess, force, true); err != nil && !errors.Is(err, ErrClaudeNotActive) {
			return err
		}
	}
	if err := database.DeleteClaudeSession(sess.ID); err != nil {
		return err
	}
	if claudeSessionIDRe.MatchString(sess.ClaudeSessionID) {
		go removeClaudeTranscript(sess.UserID, sess.ClaudeSessionID)
	}
	return nil
}

// removeClaudeTranscript deletes a conversation from the user's home volume with
// a throwaway container: PB itself never mounts nor reads that volume.
func removeClaudeTranscript(userID, claudeSessionID string) {
	cli, err := getDockerClient()
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	vol := claudeHomeVolume(userID)
	if _, err := cli.VolumeInspect(ctx, vol); err != nil {
		return
	}
	cfg := &dockercontainer.Config{
		Image: claudeCfg.image,
		User:  "0:0",
		Entrypoint: []string{"find", claudeHome + "/.claude/projects", "-mindepth", "2", "-maxdepth", "2",
			"(", "-name", claudeSessionID + ".jsonl", "-o", "-name", claudeSessionID, ")", "-exec", "rm", "-rf", "{}", "+"},
		Labels: map[string]string{claudeLabel + ".cleanup": "1"},
	}
	hostCfg := &dockercontainer.HostConfig{
		Mounts:      []dockermount.Mount{{Type: dockermount.TypeVolume, Source: vol, Target: claudeHome}},
		CapDrop:     []string{"ALL"},
		CapAdd:      []string{"DAC_OVERRIDE", "FOWNER"},
		SecurityOpt: []string{"no-new-privileges"},
		NetworkMode: "none",
		AutoRemove:  true,
	}
	created, err := cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, "")
	if err != nil {
		log.Printf("[claude] transcript cleanup for %s: %v", userID, err)
		return
	}
	if err := cli.ContainerStart(ctx, created.ID, dockercontainer.StartOptions{}); err != nil {
		_ = cli.ContainerRemove(context.Background(), created.ID, dockercontainer.RemoveOptions{Force: true})
		log.Printf("[claude] transcript cleanup for %s: %v", userID, err)
	}
}

// ClaudeTerminal is a tmux client attached to a session's container.
type ClaudeTerminal struct {
	rt        *claudeRuntime
	sessionID string
	execID    string
	attachID  string
	conn      dockertypes.HijackedResponse
	closeOnce sync.Once
}

// AttachClaudeTerminal opens a new tmux client on the session's container.
// Closing it only detaches: claude keeps running.
func AttachClaudeTerminal(ctx context.Context, sess database.ClaudeSession, cols, rows uint) (*ClaudeTerminal, error) {
	if !sess.Active() || strings.HasPrefix(sess.ContainerID, claudePendingPrefix) {
		return nil, ErrClaudeNotActive
	}
	rt := getClaudeRuntime(sess.ModuleID)
	if rt == nil || rt.containerID != sess.ContainerID {
		return nil, ErrClaudeNotActive
	}
	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	attachID, err := randomHex(8)
	if err != nil {
		return nil, err
	}
	exec, err := cli.ContainerExecCreate(ctx, rt.containerID, dockercontainer.ExecOptions{
		User:         rt.user,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{"TERM=xterm-256color", "PB_ATTACH_ID=" + attachID},
		Cmd:          []string{"/opt/pb/attach.sh"},
		ConsoleSize:  &[2]uint{max(rows, 1), max(cols, 1)},
	})
	if err != nil {
		return nil, fmt.Errorf("create terminal: %w", err)
	}
	conn, err := cli.ContainerExecAttach(ctx, exec.ID, dockercontainer.ExecAttachOptions{Tty: true})
	if err != nil {
		return nil, fmt.Errorf("attach terminal: %w", err)
	}
	rt.mu.Lock()
	rt.attached++
	rt.mu.Unlock()
	_ = database.MarkClaudeSessionAttached(sess.ID)
	t := &ClaudeTerminal{rt: rt, sessionID: sess.ID, execID: exec.ID, attachID: attachID, conn: conn}
	if cols > 0 && rows > 0 {
		_ = t.Resize(cols, rows)
	}
	return t, nil
}

func (t *ClaudeTerminal) Read(p []byte) (int, error)  { return t.conn.Reader.Read(p) }
func (t *ClaudeTerminal) Write(p []byte) (int, error) { return t.conn.Conn.Write(p) }

func (t *ClaudeTerminal) Resize(cols, rows uint) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return cli.ContainerExecResize(ctx, t.execID, dockercontainer.ResizeOptions{Width: cols, Height: rows})
}

// Close detaches this tmux client. Safe to call multiple times.
func (t *ClaudeTerminal) Close() {
	t.closeOnce.Do(func() {
		t.conn.Close()
		t.rt.mu.Lock()
		t.rt.attached--
		t.rt.mu.Unlock()
		_ = database.MarkClaudeSessionAttached(t.sessionID)
		if cli, err := getDockerClient(); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if ex, err := cli.ContainerExecCreate(ctx, t.rt.containerID, dockercontainer.ExecOptions{
				User: t.rt.user,
				Cmd:  []string{"/opt/pb/detach.sh", t.attachID},
			}); err == nil {
				_ = cli.ContainerExecStart(ctx, ex.ID, dockercontainer.ExecStartOptions{Detach: true})
			}
		}
	})
}

// --- status events ---------------------------------------------------------

func claudeSessionPayload(s database.ClaudeSession) map[string]any {
	return map[string]any{
		"id":               s.ID,
		"module_id":        s.ModuleID,
		"user_id":          s.UserID,
		"user_login":       s.UserLogin,
		"title":            s.Title,
		"status":           s.Status,
		"active":           s.Active(),
		"last_activity_at": s.LastActivityAt,
		"created_at":       s.CreatedAt,
		"unseen_result":    s.UnseenResult,
	}
}

// emitClaudeStatus pushes a session's status to the module page and to the modules list.
func emitClaudeStatus(s database.ClaudeSession) {
	p := claudeSessionPayload(s)
	websocket.SendTopicEvent("module:"+s.ModuleID, s.ModuleID, "claude_status", p)
	websocket.SendTopicEvent("claude:status", s.ModuleID, "claude_status", p)
}

// notifyClaudeUser tells the session owner, wherever they are in PB, that
// Claude needs them ("waiting") or finished its task ("done").
func notifyClaudeUser(s database.ClaudeSession, kind, message string) {
	module, _ := GetModule(s.ModuleID)
	websocket.SendTopicEvent("claude-user:"+s.UserID, s.ModuleID, "claude_notify", map[string]any{
		"kind":        kind,
		"session_id":  s.ID,
		"module_id":   s.ModuleID,
		"module_name": module.Name,
		"title":       s.Title,
		"message":     message,
	})
}

// ListClaudeActivity returns, for the modules list badges, the active session
// of every module plus the caller's ended sessions whose result is unseen.
func ListClaudeActivity(userID string) ([]map[string]any, error) {
	list, err := database.ListActiveClaudeSessions()
	if err != nil {
		return nil, err
	}
	unseen, err := database.ListUnseenClaudeSessions(userID)
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, s := range append(list, unseen...) {
		if s.Active() || s.UnseenResult {
			out = append(out, claudeSessionPayload(s))
		}
	}
	return out, nil
}

// MarkClaudeSeen clears the unseen results of a user on a module (Claude tab opened).
func MarkClaudeSeen(moduleID, userID string) error {
	ids, err := database.MarkClaudeSessionsSeen(moduleID, userID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if s, _ := database.GetClaudeSession(id); s != nil {
			emitClaudeStatus(*s)
		}
	}
	return nil
}

// --- lifecycle: reaper, reconciliation, retention --------------------------

// StartClaudeManager reconciles the database with the running containers, then
// starts the reaper and the docker events watcher.
func StartClaudeManager() {
	go func() {
		reconcileClaude()
		go watchClaudeContainers()
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		lastPurge := time.Time{}
		for range t.C {
			reapClaude()
			if time.Since(lastPurge) > time.Hour {
				purgeClaudeSessions()
				lastPurge = time.Now()
			}
		}
	}()
}

// reconcileClaude runs at boot: restores the runtime of every container that
// survived the restart and ends the sessions whose container is gone. It never
// touches a running container.
func reconcileClaude() {
	list, err := database.ListActiveClaudeSessions()
	if err != nil {
		log.Printf("[claude] reconcile: %v", err)
		return
	}
	cli, err := getDockerClient()
	if err != nil {
		log.Printf("[claude] reconcile: %v", err)
		return
	}
	known := map[string]bool{}
	for _, s := range list {
		known[s.ContainerID] = true
		if !s.Active() || strings.HasPrefix(s.ContainerID, claudePendingPrefix) {
			_ = database.EndClaudeSession(s.ID, database.ClaudeAborted)
			continue
		}
		if err := restoreClaudeRuntime(s); err != nil {
			log.Printf("[claude] reconcile: session %s: %v", s.ID, err)
			_ = finishClaudeContainer(s.ModuleID, s.ContainerID, database.ClaudeStopped, claudeEnd{unattended: true})
		}
	}
	// Containers without a session (lost row): stopped ones are removed, running
	// ones are left alone and reported, they may still be working.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	containers, err := cli.ContainerList(ctx, dockercontainer.ListOptions{
		All:     true,
		Filters: dockerfilters.NewArgs(dockerfilters.Arg("label", claudeLabel+"=1")),
	})
	if err != nil {
		return
	}
	for _, c := range containers {
		if known[c.ID] {
			continue
		}
		if c.State != "running" {
			_ = cli.ContainerRemove(ctx, c.ID, dockercontainer.RemoveOptions{Force: true})
		} else {
			log.Printf("[claude] container %s (%.12s) has no session, left running", strings.Join(c.Names, ","), c.ID)
		}
	}
}

func restoreClaudeRuntime(s database.ClaudeSession) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	info, err := cli.ContainerInspect(ctx, s.ContainerID)
	if err != nil {
		return err
	}
	if info.State == nil || !info.State.Running {
		return errors.New("container is not running")
	}
	token := ""
	for _, e := range info.Config.Env {
		if v, ok := strings.CutPrefix(e, "PB_HOOK_TOKEN="); ok {
			token = v
		}
	}
	if token == "" {
		return errors.New("container has no hook token")
	}
	module, err := GetModule(s.ModuleID)
	if err != nil || module.ID == "" {
		return errors.New("module not found")
	}
	uid, gid := parseUIDGID(info.Config.User)
	// An old runtime would remove the new sockets (same paths) when closed.
	claudeMu.Lock()
	old := claudeRuntimes[s.ModuleID]
	delete(claudeRuntimes, s.ModuleID)
	claudeMu.Unlock()
	if old != nil {
		old.close()
	}
	rt, err := startClaudeRuntime(module, s.ContainerID, token, uid, gid)
	if err != nil {
		return err
	}
	claudeMu.Lock()
	claudeRuntimes[s.ModuleID] = rt
	claudeMu.Unlock()

	// Hooks fired while the backend was down are lost: recover the last one.
	if status := lastClaudeEventStatus(ctx, rt); status != "" && status != s.Status {
		_ = database.SetClaudeSessionStatus(s.ID, status)
	}
	log.Printf("[claude] restored session %s on module %s", s.ID, module.Slug)
	return nil
}

func lastClaudeEventStatus(ctx context.Context, rt *claudeRuntime) string {
	cli, err := getDockerClient()
	if err != nil {
		return ""
	}
	ex, err := cli.ContainerExecCreate(ctx, rt.containerID, dockercontainer.ExecOptions{
		User: rt.user, AttachStdout: true, Cmd: []string{"cat", "/tmp/pb-last-event"},
	})
	if err != nil {
		return ""
	}
	resp, err := cli.ContainerExecAttach(ctx, ex.ID, dockercontainer.ExecAttachOptions{})
	if err != nil {
		return ""
	}
	defer resp.Close()
	out, _ := io.ReadAll(io.LimitReader(resp.Reader, 512))
	ev := string(out)
	switch {
	case strings.Contains(ev, "stop"), strings.Contains(ev, "session_start"):
		return database.ClaudeIdle
	case strings.Contains(ev, "prompt"), strings.Contains(ev, "tool_done"):
		return database.ClaudeWorking
	}
	return ""
}

// reapClaude applies the lifecycle rules:
//   - a dead container ends its session (stopped);
//   - an idle (or still starting) session with no client attached for CLAUDE_IDLE_TTL is stopped;
//   - a working/waiting_input session is never touched below CLAUDE_MAX_WORK_SECONDS,
//     and aborted above it.
func reapClaude() {
	list, err := database.ListActiveClaudeSessions()
	if err != nil {
		return
	}
	cli, err := getDockerClient()
	if err != nil {
		return
	}
	for _, s := range list {
		func() {
			unlock := claudeModuleLock(s.ModuleID)
			defer unlock()
			cur, err := database.GetClaudeSession(s.ID)
			if err != nil || cur == nil || cur.ContainerID == "" {
				return
			}
			if strings.HasPrefix(cur.ContainerID, claudePendingPrefix) {
				if time.Since(cur.CreatedAt) > claudePendingTimeout {
					_ = database.EndClaudeSession(cur.ID, database.ClaudeAborted)
				}
				return
			}
			if decision := claudeReapDecision(*cur, claudeAttached(cur.ModuleID), containerRunning(cli, cur.ContainerID), time.Now()); decision != "" {
				_ = finishClaudeContainer(cur.ModuleID, cur.ContainerID, decision, claudeEnd{unattended: true})
			}
		}()
	}
}

func claudeAttached(moduleID string) int {
	rt := getClaudeRuntime(moduleID)
	if rt == nil {
		return 0
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.attached
}

func containerRunning(cli interface {
	ContainerInspect(context.Context, string) (dockercontainer.InspectResponse, error)
}, id string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		// Only a confirmed missing container counts as dead.
		return !errdefs.IsNotFound(err)
	}
	return info.State != nil && info.State.Running
}

// claudeReapDecision returns the final status to apply to s, or "" to keep it.
func claudeReapDecision(s database.ClaudeSession, attached int, running bool, now time.Time) string {
	if !running {
		return database.ClaudeStopped
	}
	switch s.Status {
	case database.ClaudeWorking, database.ClaudeWaitingInput:
		if s.WorkStartedAt != nil && now.Sub(*s.WorkStartedAt) > claudeCfg.maxWork {
			return database.ClaudeAborted
		}
	case database.ClaudeIdle, database.ClaudeStarting:
		if attached > 0 {
			return ""
		}
		last := s.LastActivityAt
		if s.LastAttachedAt != nil && s.LastAttachedAt.After(last) {
			last = *s.LastAttachedAt
		}
		if now.Sub(last) > claudeCfg.idleTTL {
			return database.ClaudeStopped
		}
	}
	return ""
}

// watchClaudeContainers ends a session as soon as its container exits
// (claude quit with /exit, crash, OOM...).
func watchClaudeContainers() {
	for {
		cli, err := getDockerClient()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		args := dockerfilters.NewArgs(
			dockerfilters.Arg("type", "container"),
			dockerfilters.Arg("label", claudeLabel+"=1"),
			dockerfilters.Arg("event", "die"),
		)
		msgs, errs := cli.Events(context.Background(), eventtypes.ListOptions{Filters: args})
	loop:
		for {
			select {
			case msg := <-msgs:
				moduleID := msg.Actor.Attributes[claudeModIDLabel]
				if moduleID == "" {
					continue
				}
				go func(moduleID, containerID string) {
					unlock := claudeModuleLock(moduleID)
					defer unlock()
					if s, _ := database.GetActiveClaudeSessionByContainer(containerID); s != nil {
						_ = finishClaudeContainer(moduleID, containerID, database.ClaudeStopped, claudeEnd{unattended: true})
					}
				}(moduleID, msg.Actor.ID)
			case <-errs:
				break loop
			}
		}
		time.Sleep(2 * time.Second)
	}
}

// purgeClaudeSessions deletes sessions (and transcripts) older than the retention.
func purgeClaudeSessions() {
	list, err := database.ListExpiredClaudeSessions(time.Now().Add(-claudeCfg.retention))
	if err != nil {
		return
	}
	for _, s := range list {
		if err := DeleteClaudeSession(s, false); err == nil {
			log.Printf("[claude] purged session %s (retention)", s.ID)
		}
	}
}
