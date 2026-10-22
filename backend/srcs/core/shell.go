package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	dockermount "github.com/docker/docker/api/types/mount"
)

// Module shells are throwaway root containers whose only host access is the
// module's cloned repo, bind-mounted on /workspace. They run without the docker
// socket, without privileges and outside the Pan Bagnat networks, so being root
// there cannot reach anything but the repo itself.

const (
	shellWorkDir      = "/workspace"
	shellDockerDir    = "/run/pb-docker"
	shellLabel        = "pan-bagnat.module-shell"
	shellModuleLabel  = "pan-bagnat.module-shell.module"
	defaultShellImage = "pan-bagnat-module-shell:latest"
	maxModuleShells   = 5
	// ShellMaxLifetime is a hard cap on a shell session, enforced by the API
	// handler, by the container itself and by a reaper (zombie safety net).
	ShellMaxLifetime = time.Hour
)

var (
	ErrShellLimitReached = errors.New("too many open shells")

	shellMu     sync.Mutex
	shellActive int
)

// ModuleShell is an attached TTY session inside a sandbox container.
type ModuleShell struct {
	containerID string
	conn        dockertypes.HijackedResponse
	proxy       *scopedDockerProxy
	closeOnce   sync.Once
}

func shellImage() string {
	if img := strings.TrimSpace(os.Getenv("MODULE_SHELL_IMAGE")); img != "" {
		return img
	}
	return defaultShellImage
}

func ensureShellImage(ctx context.Context, image string) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	if _, err := cli.ImageInspect(ctx, image); err == nil {
		return nil
	}
	rc, err := cli.ImagePull(ctx, image, dockerimage.PullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = io.Copy(io.Discard, rc)
	return err
}

// OpenModuleShell starts a sandbox container on the module repo and attaches to it.
// Callers MUST have verified the user is an admin beforehand.
func OpenModuleShell(ctx context.Context, module Module, cols, rows uint) (*ModuleShell, error) {
	// ModuleRepoRoot validates the slug and that the repo is actually cloned.
	containerRoot, err := ModuleRepoRoot(module)
	if err != nil {
		return nil, fmt.Errorf("module repository is not available: %w", err)
	}
	hostRoot, err := HostModuleRepoRoot(module)
	if err != nil {
		return nil, err
	}

	shellMu.Lock()
	if shellActive >= maxModuleShells {
		shellMu.Unlock()
		return nil, ErrShellLimitReached
	}
	shellActive++
	shellMu.Unlock()
	release := func() {
		shellMu.Lock()
		shellActive--
		shellMu.Unlock()
	}

	cli, err := getDockerClient()
	if err != nil {
		release()
		return nil, err
	}
	image := shellImage()
	if err := ensureShellImage(ctx, image); err != nil {
		release()
		return nil, fmt.Errorf("shell image unavailable: %w", err)
	}

	// Scoped docker endpoint: the shell never sees the real docker socket.
	sessionID, err := randomShellID()
	if err != nil {
		release()
		return nil, err
	}
	proxy, err := startScopedDockerProxy(module.Slug, filepath.Join(filepath.Dir(containerRoot), shellProxyDirName, sessionID))
	if err != nil {
		release()
		return nil, fmt.Errorf("start docker proxy: %w", err)
	}
	proxyHostDir := filepath.Join(filepath.Dir(hostRoot), shellProxyDirName, sessionID)

	cfg := &dockercontainer.Config{
		Image: image,
		// The container dies on its own after ShellMaxLifetime even if the backend is gone.
		Cmd:          []string{"timeout", "--foreground", "-s", "KILL", fmt.Sprint(int(ShellMaxLifetime.Seconds())), "/usr/bin/zsh"},
		User:         "0:0",
		WorkingDir:   shellWorkDir,
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"TERM=xterm-256color",
			"DOCKER_HOST=unix://" + shellDockerDir + "/" + shellProxySockName,
			"COMPOSE_PROJECT_NAME=" + module.Slug,
		},
		Labels: map[string]string{shellLabel: "true", shellModuleLabel: module.ID},
	}
	hostCfg := &dockercontainer.HostConfig{
		// Mount struct (not Binds) so the host path can't inject extra bind options.
		Mounts: []dockermount.Mount{
			{Type: dockermount.TypeBind, Source: hostRoot, Target: shellWorkDir},
			{Type: dockermount.TypeBind, Source: proxyHostDir, Target: shellDockerDir},
		},
		Privileged:     false,
		SecurityOpt:    []string{"no-new-privileges"},
		NetworkMode:    "bridge", // default bridge only: never the pan-bagnat-* networks
		AutoRemove:     false,
		ReadonlyRootfs: false,
		Resources: dockercontainer.Resources{
			Memory:    1 << 30,
			NanoCPUs:  2_000_000_000,
			PidsLimit: ptrInt64(512),
		},
	}
	created, err := cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, "")
	if err != nil {
		proxy.Close()
		release()
		return nil, fmt.Errorf("create shell container: %w", err)
	}
	removeContainer := func() {
		rctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = cli.ContainerRemove(rctx, created.ID, dockercontainer.RemoveOptions{Force: true, RemoveVolumes: true})
	}

	conn, err := cli.ContainerAttach(ctx, created.ID, dockercontainer.AttachOptions{
		Stream: true, Stdin: true, Stdout: true, Stderr: true,
	})
	if err != nil {
		removeContainer()
		proxy.Close()
		release()
		return nil, fmt.Errorf("attach shell container: %w", err)
	}
	if err := cli.ContainerStart(ctx, created.ID, dockercontainer.StartOptions{}); err != nil {
		conn.Close()
		removeContainer()
		proxy.Close()
		release()
		return nil, fmt.Errorf("start shell container: %w", err)
	}

	sh := &ModuleShell{containerID: created.ID, conn: conn, proxy: proxy}
	if cols > 0 && rows > 0 {
		_ = sh.Resize(cols, rows)
	}
	return sh, nil
}

// Read returns the raw TTY output of the shell.
func (s *ModuleShell) Read(p []byte) (int, error) { return s.conn.Reader.Read(p) }

// Write sends raw input to the shell.
func (s *ModuleShell) Write(p []byte) (int, error) { return s.conn.Conn.Write(p) }

func (s *ModuleShell) Resize(cols, rows uint) error {
	cli, err := getDockerClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return cli.ContainerResize(ctx, s.containerID, dockercontainer.ResizeOptions{Width: cols, Height: rows})
}

// Close detaches and destroys the sandbox container. Safe to call multiple times.
func (s *ModuleShell) Close() {
	s.closeOnce.Do(func() {
		s.conn.Close()
		if cli, err := getDockerClient(); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = cli.ContainerRemove(ctx, s.containerID, dockercontainer.RemoveOptions{Force: true, RemoveVolumes: true})
		}
		s.proxy.Close()
		shellMu.Lock()
		shellActive--
		shellMu.Unlock()
	})
}

// CleanupStaleModuleShells removes shell containers left behind by a previous
// backend run (crash or restart while a shell was open).
func CleanupStaleModuleShells() {
	base := os.Getenv("REPO_BASE_PATH")
	if base == "" {
		base = "../../repos"
	}
	_ = os.RemoveAll(filepath.Join(base, shellProxyDirName))
	cli, err := getDockerClient()
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	list, err := cli.ContainerList(ctx, dockercontainer.ListOptions{
		All:     true,
		Filters: dockerfilters.NewArgs(dockerfilters.Arg("label", shellLabel+"=true")),
	})
	if err != nil {
		return
	}
	for _, c := range list {
		_ = cli.ContainerRemove(ctx, c.ID, dockercontainer.RemoveOptions{Force: true, RemoveVolumes: true})
	}
}

func randomShellID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func ptrInt64(v int64) *int64 { return &v }

// StartModuleShellReaper periodically removes shell containers that outlived
// their lifetime (e.g. the backend died while a session was open).
func StartModuleShellReaper() {
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			cli, err := getDockerClient()
			if err != nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			list, err := cli.ContainerList(ctx, dockercontainer.ListOptions{
				All:     true,
				Filters: dockerfilters.NewArgs(dockerfilters.Arg("label", shellLabel+"=true")),
			})
			if err == nil {
				for _, c := range list {
					if time.Since(time.Unix(c.Created, 0)) > ShellMaxLifetime+2*time.Minute {
						_ = cli.ContainerRemove(ctx, c.ID, dockercontainer.RemoveOptions{Force: true, RemoveVolumes: true})
					}
				}
			}
			cancel()
		}
	}()
}
