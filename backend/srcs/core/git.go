package core

import (
    "backend/database"
    "backend/websocket"
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

func DeleteModuleRepoDir(module Module) error {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos"
	}
	targetDir := filepath.Join(baseRepoPath, module.Slug)

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to delete repo folder %s: %w", targetDir, err)
	}

	log.Printf("✅ Deleted repo folder %s\n", targetDir)
	return nil
}

func CloneModuleRepo(module Module) error {
    LogModule(module.ID, "INFO", fmt.Sprintf("Cloning repo %s in repos/%s", module.GitURL, module.Slug), nil, nil)
    baseRepoPath := os.Getenv("REPO_BASE_PATH")
    if baseRepoPath == "" {
        baseRepoPath = "../../repos" // fallback for local dev
    }
    targetDir := filepath.Join(baseRepoPath, module.Slug)

    // Prepare a temporary SSH key for this clone
    sshCommand, cleanup, err := tempSSHForModule(module)
    if err != nil {
        return LogModule(module.ID, "ERROR", "failed to prepare temp ssh key", nil, err)
    }
    defer cleanup()
    cmd := exec.Command(
        "git", "clone",
        "-b", module.GitBranch,
        module.GitURL,
        targetDir,
    )
    cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)

	err = runAndLog(module.ID, cmd)
	if err != nil {
		return LogModule(
			module.ID,
			"ERROR",
			"git clone failed",
			nil,
			err,
		)
	}

	newStatus := "disabled"
	_, err = database.PatchModule(database.ModulePatch{
		ID:     module.ID,
		Status: &newStatus,
	})
	if err != nil {
		return LogModule(module.ID, "ERROR", "error while updating status to database", nil, err)
    }

    _ = ensureSafeDirectory(module, targetDir)

    composePath := filepath.Join(targetDir, "docker-compose.yml")
    if _, statErr := os.Stat(composePath); statErr != nil {
        LogModule(module.ID, "WARN", "No docker-compose.yml found; Pan Bagnat modules require Docker Compose", nil, nil)
    }

	// Initialize git metadata in DB (timestamps, commits, behind count)
	now := time.Now().UTC()
	_ = updateGitComputed(module, &now, &now)
	return nil
}

func PullModuleRepo(module Module) error {
    baseRepoPath := os.Getenv("REPO_BASE_PATH")
    if baseRepoPath == "" {
        baseRepoPath = "../../repos" // fallback for local dev
    }
    targetDir := filepath.Join(baseRepoPath, module.Slug)
    // Ensure safe.directory before issuing git commands
    _ = ensureSafeDirectory(module, targetDir)

    // Prepare a temporary SSH key for this pull session
    sshCommand, cleanup, err := tempSSHForModule(module)
    if err != nil {
        return LogModule(module.ID, "ERROR", "failed to prepare temp ssh key", nil, err)
    }
    defer cleanup()

    // If working tree is dirty, auto-stash (including untracked), pull, then try to pop
    dirty := false
    {
        cmd := exec.Command("git", "-C", targetDir, "status", "--porcelain")
		out, _ := cmd.CombinedOutput()
		if len(bytesTrimSpace(out)) > 0 {
			dirty = true
		}
	}

	stashed := false
	if dirty {
		LogModule(module.ID, "INFO", "Working tree dirty: auto-stashing before pull", nil, nil)
		msg := fmt.Sprintf("pan-bagnat auto-stash %s", time.Now().UTC().Format(time.RFC3339))
		cmdStash := exec.Command("git", "-C", targetDir, "stash", "push", "--include-untracked", "-m", msg)
		if err := runAndLog(module.ID, cmdStash); err == nil {
			stashed = true
		} else {
			LogModule(module.ID, "WARN", "git stash failed; proceeding with pull may fail", nil, err)
		}
	}

	// Always fetch all remotes and prune before pulling
    cmdFetchAll := exec.Command("git", "-C", targetDir, "fetch", "--all", "--prune")
    cmdFetchAll.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)
    if e := runAndLog(module.ID, cmdFetchAll); e != nil {
        LogModule(module.ID, "WARN", "git fetch --all failed before pull", nil, e)
    }

	// Ensure current branch has an upstream; if not, try to set it to origin/<branch> if it exists
	curBranch := ""
	{
		cmdCur := exec.Command("git", "-C", targetDir, "rev-parse", "--abbrev-ref", "HEAD")
		out, _ := cmdCur.CombinedOutput()
		curBranch = string(bytesTrimSpace(out))
	}
	if curBranch != "" && curBranch != "HEAD" {
		cmdUp := exec.Command("git", "-C", targetDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
		if upOut, upErr := cmdUp.CombinedOutput(); upErr != nil || len(bytesTrimSpace(upOut)) == 0 {
			// Check if origin/<branch> exists
            cmdLs := exec.Command("git", "-C", targetDir, "ls-remote", "--heads", "origin", curBranch)
            cmdLs.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)
            lsOut, _ := cmdLs.CombinedOutput()
			if len(bytesTrimSpace(lsOut)) > 0 {
				setUp := exec.Command("git", "-C", targetDir, "branch", "--set-upstream-to=origin/"+curBranch)
				if e := runAndLog(module.ID, setUp); e != nil {
					LogModule(module.ID, "WARN", "failed to set upstream to origin/<branch>", map[string]any{"branch": curBranch}, e)
				}
			} else {
				// No remote branch; skip pull gracefully
				LogModule(module.ID, "INFO", "No upstream remote branch found; skipping pull", map[string]any{"branch": curBranch}, nil)
				if stashed {
					_ = runAndLog(module.ID, exec.Command("git", "-C", targetDir, "stash", "pop"))
				}
				now := time.Now().UTC()
				_, _ = database.PatchModule(database.ModulePatch{ID: module.ID, LastUpdate: &now, GitLastPull: &now})
				broadcastGitStatus(module)
				return nil
			}
		}
	}

    // Pull in merge mode (no rebase)
    cmd := exec.Command("git", "-C", targetDir, "-c", "pull.rebase=false", "pull")
    cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)
    err = runAndLog(module.ID, cmd)
    if err != nil {
        return LogModule(module.ID, "ERROR", "git pull failed", nil, err)
    }

	if stashed {
		LogModule(module.ID, "INFO", "Re-applying stashed changes (stash pop)", nil, nil)
		cmdPop := exec.Command("git", "-C", targetDir, "stash", "pop")
		if err := runAndLog(module.ID, cmdPop); err != nil {
			// Conflicts likely; keep going. Status endpoint will reflect conflicts.
			LogModule(module.ID, "WARN", "stash pop completed with issues. Resolve conflicts if any.", nil, err)
		}
	}

	// Record last pull in DB and notify
	// Persist HEAD immediately then recompute full snapshot
	_ = persistHeadCommit(module)
	now := time.Now().UTC()
	_ = updateGitComputed(module, &now, &now)
	broadcastGitStatus(module)
	return LogModule(module.ID, "INFO", fmt.Sprintf("Pulled module from URL %s", module.GitURL), nil, nil)
}

func UpdateModuleGitRemote(moduleID, moduleSlug, newGitURL string) error {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos"
	}
	targetDir := filepath.Join(baseRepoPath, moduleSlug)

	_ = ensureSafeDirectory(Module{ID: moduleID, Slug: moduleSlug}, targetDir)
	cmd := exec.Command("git", "-C", targetDir, "remote", "set-url", "origin", newGitURL)
	err := runAndLog(moduleID, cmd)
	if err != nil {
		return LogModule(moduleID, "ERROR", "failed to update remote url", nil, err)
	}

	_, err = database.PatchModule(database.ModulePatch{
		ID:     moduleID,
		GitURL: &newGitURL,
	})
	if err != nil {
		log.Printf("error while updating git_url to database: %s\n", err.Error())
		return fmt.Errorf("error while updating git_url to database: %w", err)
	}
	return nil
}

// ---- Git helpers for fetch/status/merge ----

type GitStatus struct {
    IsMerging bool     `json:"is_merging"`
    Conflicts []string `json:"conflicts"`
    Branch    string   `json:"branch"`
    Head      string   `json:"head"`
    HeadSubject string `json:"head_subject"`
    Modified  []string `json:"modified"`
    LastFetch string   `json:"last_fetch"`
    LastPull  string   `json:"last_pull"`
    LatestHash    string `json:"latest_hash"`
    LatestSubject string `json:"latest_subject"`
    Behind        int    `json:"behind"`
}

func repoDirFor(module Module) string {
	baseRepoPath := os.Getenv("REPO_BASE_PATH")
	if baseRepoPath == "" {
		baseRepoPath = "../../repos"
	}
	return filepath.Join(baseRepoPath, module.Slug)
}

func GitFetchModule(module Module) error {
    repoDir := repoDirFor(module)
    _ = ensureSafeDirectory(module, repoDir)
    // Use a temporary SSH key for this fetch
    sshCommand, cleanup, err := tempSSHForModule(module)
    if err != nil {
        return LogModule(module.ID, "ERROR", "failed to prepare temp ssh key", nil, err)
    }
    defer cleanup()
    cmd := exec.Command("git", "-C", repoDir, "fetch", "--all")
    cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand)
    if err := runAndLog(module.ID, cmd); err != nil {
        return LogModule(module.ID, "ERROR", "git fetch failed", nil, err)
    }
    now := time.Now().UTC()
    _ = updateGitComputed(module, &now, nil)
    broadcastGitStatus(module)
    return nil
}

func GitStatusModule(module Module) (GitStatus, error) {
    repoDir := repoDirFor(module)
    _ = ensureSafeDirectory(module, repoDir)
    // Prepare temp SSH for any network fetch during status computation
    sshCommand, cleanup, _ := tempSSHForModule(module)
    if cleanup != nil {
        defer cleanup()
    }
    st := GitStatus{}
	// Prefer DB values when available; compute and store if missing
	if module.GitBranch != "" {
		st.Branch = module.GitBranch
	} else {
		// Fallback to runtime branch
		cmd := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD")
		out, _ := cmd.CombinedOutput()
		st.Branch = string(bytesTrimSpace(out))
		if st.Branch != "" && st.Branch != "HEAD" {
			_, _ = database.PatchModule(database.ModulePatch{ID: module.ID, GitBranch: &st.Branch})
		}
	}
    if module.CurrentCommitHash != "" {
        st.Head = module.CurrentCommitHash
    } else {
        cmd := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD")
        out, _ := cmd.CombinedOutput()
        st.Head = string(bytesTrimSpace(out))
        // Ensure commit metadata is computed and stored for later responses
        _ = updateGitComputed(module, nil, nil)
    }
    // Head subject
    if module.CurrentCommitSubject != "" {
        st.HeadSubject = module.CurrentCommitSubject
    } else {
        subjB, _ := exec.Command("git", "-C", repoDir, "log", "-1", "--pretty=%s").CombinedOutput()
        st.HeadSubject = string(bytesTrimSpace(subjB))
    }
	// Is merging?
	if _, err := os.Stat(filepath.Join(repoDir, ".git", "MERGE_HEAD")); err == nil {
		st.IsMerging = true
	} else {
		st.IsMerging = false
	}
	// Conflicts
	{
		cmd := exec.Command("git", "-C", repoDir, "diff", "--name-only", "--diff-filter=U")
		out, _ := cmd.CombinedOutput()
		lines := splitLines(string(out))
		st.Conflicts = make([]string, 0, len(lines))
		for _, l := range lines {
			if l != "" {
				st.Conflicts = append(st.Conflicts, l)
			}
		}
	}
	// Modified (tracked) files: combine staged and unstaged vs HEAD
	{
		set := map[string]struct{}{}
		cmd1 := exec.Command("git", "-C", repoDir, "diff", "--name-only")
		if out, err := cmd1.CombinedOutput(); err == nil {
			for _, l := range splitLines(string(out)) {
				if l != "" {
					set[l] = struct{}{}
				}
			}
		}
		cmd2 := exec.Command("git", "-C", repoDir, "diff", "--name-only", "--cached")
		if out, err := cmd2.CombinedOutput(); err == nil {
			for _, l := range splitLines(string(out)) {
				if l != "" {
					set[l] = struct{}{}
				}
			}
		}
		mod := make([]string, 0, len(set))
		for k := range set {
			mod = append(mod, k)
		}
    st.Modified = mod
}
    // Determine upstream and compute latest commit and behind count
    up := ""
    if b, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}").CombinedOutput(); err == nil {
        up = string(bytesTrimSpace(b))
    }
    if up == "" && module.GitBranch != "" { up = "origin/" + module.GitBranch }
    if up != "" {
        cmdFetch := exec.Command("git", "-C", repoDir, "fetch", "--all", "--prune")
        if sshCommand != "" { cmdFetch.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand) }
        _ = cmdFetch.Run()
        latestLine, _ := exec.Command("git", "-C", repoDir, "log", "-1", up, "--pretty=%H%x1f%s").CombinedOutput()
        s := string(bytesTrimSpace(latestLine))
        if s != "" {
            arr := []string{}
            cur := ""
            for i := 0; i < len(s); i++ { if s[i]==0x1f { arr = append(arr, cur); cur = "" } else { cur += string(s[i]) } }
            arr = append(arr, cur)
            if len(arr) >= 2 { st.LatestHash = arr[0]; st.LatestSubject = arr[1] }
        }
        cntB, _ := exec.Command("git", "-C", repoDir, "rev-list", "--left-right", "--count", "HEAD..."+up).CombinedOutput()
        var left, right int
        fmt.Sscanf(string(bytesTrimSpace(cntB)), "%d %d", &left, &right)
        st.Behind = right
    }
    if !module.GitLastPull.IsZero() {
        st.LastPull = module.GitLastPull.UTC().Format(time.RFC3339)
    }
    if !module.GitLastFetch.IsZero() {
        st.LastFetch = module.GitLastFetch.UTC().Format(time.RFC3339)
    }
    return st, nil
}

func GitAddPaths(module Module, paths []string) error {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	args := []string{"-C", repoDir, "add"}
	if len(paths) == 0 {
		args = append(args, "-A")
	} else {
		args = append(args, paths...)
	}
	cmd := exec.Command("git", args...)
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	broadcastGitStatus(module)
	return nil
}

func GitMergeContinue(module Module) error {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	// Commit with default merge message
	cmd := exec.Command("git", "-C", repoDir, "commit", "--no-edit")
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	now := time.Now().UTC()
	_ = updateGitComputed(module, &now, &now)
	broadcastGitStatus(module)
	return nil
}

func GitMergeAbort(module Module) error {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	cmd := exec.Command("git", "-C", repoDir, "merge", "--abort")
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	broadcastGitStatus(module)
	return nil
}

type GitCommit struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Email   string `json:"email"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}

type GitBranch struct {
	Name     string `json:"name"`
	Current  bool   `json:"current"`
	Upstream string `json:"upstream"`
}

func GitListCommits(module Module, limit int) ([]GitCommit, error) {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	// %H hash, %an author name, %ae email, %ad date ISO, %s subject; use unit-sep \x1f
	format := "%H%x1f%an%x1f%ae%x1f%ad%x1f%s"
	// Show commits for the current HEAD (branch or detached)
	cmd := exec.Command("git", "-C", repoDir, "log", "--date=iso-strict", "--pretty=format:"+format, "-n", fmt.Sprintf("%d", limit))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log error: %w", err)
	}
	lines := splitLines(string(out))
	commits := make([]GitCommit, 0, len(lines))
	for _, l := range lines {
		parts := []string{}
		cur := ""
		for i := 0; i < len(l); i++ {
			if l[i] == 0x1f {
				parts = append(parts, cur)
				cur = ""
			} else {
				cur += string(l[i])
			}
		}
		parts = append(parts, cur)
		if len(parts) >= 5 {
			commits = append(commits, GitCommit{Hash: parts[0], Author: parts[1], Email: parts[2], Date: parts[3], Subject: parts[4]})
		}
	}
	return commits, nil
}

// GitListCommitsRef returns commits for a specific ref, preferably a remote branch.
// If ref is empty, it attempts to use the upstream (e.g. origin/main). It fetches
// the ref before listing to ensure online view.
func GitListCommitsRef(module Module, ref string, limit int) ([]GitCommit, error) {
    repoDir := repoDirFor(module)
    _ = ensureSafeDirectory(module, repoDir)
    // Temp SSH for any fetches here
    sshCommand, cleanup, _ := tempSSHForModule(module)
    if cleanup != nil { defer cleanup() }

	// Determine remote ref
	remoteRef := ref
	if remoteRef == "" {
		// Resolve upstream of current branch
		if out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}").CombinedOutput(); err == nil {
			remoteRef = string(bytesTrimSpace(out))
		}
	}
	// If ref is HEAD (detached), prefer the configured DB branch
	if remoteRef == "HEAD" || remoteRef == "origin/HEAD" {
		if module.GitBranch != "" {
			remoteRef = module.GitBranch
		}
		if remoteRef == "" {
			remoteRef = "main"
		}
	}
	// Normalize to origin/<branch> if local name provided
	if remoteRef != "" && !strings.HasPrefix(remoteRef, "origin/") && !strings.Contains(remoteRef, "/") {
		remoteRef = "origin/" + remoteRef
	}

	// Fetch the specific ref if possible
    if remoteRef != "" {
        // Split remote/name
        parts := strings.SplitN(remoteRef, "/", 2)
        if len(parts) == 2 {
            c := exec.Command("git", "-C", repoDir, "fetch", parts[0], parts[1])
            if sshCommand != "" { c.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand) }
            _ = runAndLog(module.ID, c)
        } else {
            c := exec.Command("git", "-C", repoDir, "fetch", "--all", "--prune")
            if sshCommand != "" { c.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand) }
            _ = runAndLog(module.ID, c)
        }
    } else {
        c := exec.Command("git", "-C", repoDir, "fetch", "--all", "--prune")
        if sshCommand != "" { c.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand) }
        _ = runAndLog(module.ID, c)
    }

	// Build pretty format
	format := "%H%x1f%an%x1f%ae%x1f%ad%x1f%s"
	var cmd *exec.Cmd
	if remoteRef != "" {
		cmd = exec.Command("git", "-C", repoDir, "log", remoteRef, "--date=iso-strict", "--pretty=format:"+format, "-n", fmt.Sprintf("%d", limit))
	} else {
		cmd = exec.Command("git", "-C", repoDir, "log", "--date=iso-strict", "--pretty=format:"+format, "-n", fmt.Sprintf("%d", limit))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log error: %w", err)
	}
	lines := splitLines(string(out))
	commits := make([]GitCommit, 0, len(lines))
	for _, l := range lines {
		parts := []string{}
		cur := ""
		for i := 0; i < len(l); i++ {
			if l[i] == 0x1f {
				parts = append(parts, cur)
				cur = ""
			} else {
				cur += string(l[i])
			}
		}
		parts = append(parts, cur)
		if len(parts) >= 5 {
			commits = append(commits, GitCommit{Hash: parts[0], Author: parts[1], Email: parts[2], Date: parts[3], Subject: parts[4]})
		}
	}
	return commits, nil
}

func GitListBranches(module Module) ([]GitBranch, error) {
    repoDir := repoDirFor(module)
    _ = ensureSafeDirectory(module, repoDir)
    sshCommand, cleanup, _ := tempSSHForModule(module)
    if cleanup != nil { defer cleanup() }
	// Determine current upstream (e.g. origin/main); if detached, fallback to DB git_branch
	curUpstream := ""
	if out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}").CombinedOutput(); err == nil {
		curUpstream = string(bytesTrimSpace(out))
	}
	if curUpstream == "" && module.GitBranch != "" {
		curUpstream = "origin/" + module.GitBranch
	}
	// Query remote branches from origin
    cmd := exec.Command("git", "-C", repoDir, "ls-remote", "--heads", "origin")
    if sshCommand != "" { cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand) }
    out, err := cmd.CombinedOutput()
	if err != nil {
		LogModule(module.ID, "ERROR", "git ls-remote failed", map[string]any{"stderr": string(out)}, err)
		return nil, fmt.Errorf("git ls-remote error: %w", err)
	}
	lines := splitLines(string(out))
	list := make([]GitBranch, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		// <hash>\trefs/heads/<branch>
		parts := strings.Split(l, "\t")
		if len(parts) < 2 {
			continue
		}
		ref := parts[1]
		const pfx = "refs/heads/"
		if !strings.HasPrefix(ref, pfx) {
			continue
		}
		name := strings.TrimPrefix(ref, pfx)
		upstream := "origin/" + name
		list = append(list, GitBranch{Name: name, Current: curUpstream == upstream, Upstream: upstream})
	}
	return list, nil
}

// isCommitRef returns true if ref resolves to a commit object
func isCommitRef(repoDir, ref string) bool {
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "--verify", ref+"^{commit}")
	return cmd.Run() == nil
}

func GitCheckout(module Module, ref string) error {
    repoDir := repoDirFor(module)
    _ = ensureSafeDirectory(module, repoDir)
    // Temp SSH for network fetch
    sshCommand, cleanup, _ := tempSSHForModule(module)
    if cleanup != nil { defer cleanup() }

    // Fetch all remotes and prune deleted branches
    {
        c := exec.Command("git", "-C", repoDir, "fetch", "--all", "--prune")
        if sshCommand != "" { c.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand) }
        if e := runAndLog(module.ID, c); e != nil {
            return e
        }
    }

	if isCommitRef(repoDir, ref) {
		// Checkout the commit (detached HEAD initially)
		if e := runAndLog(module.ID, exec.Command("git", "-C", repoDir, "checkout", ref)); e != nil {
			return e
		}

		// Detect remote branch containing this commit
		out, err := exec.Command("git", "-C", repoDir, "branch", "--contains", ref, "-r").Output()
		if err != nil {
			return err
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")

		// Pick the first real remote branch (ignore symbolic refs like HEAD -> origin/main)
		var branch string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "origin/") && !strings.Contains(line, "->") {
				branch = strings.TrimPrefix(line, "origin/")
				break
			}
		}

		if branch == "" {
			return fmt.Errorf("commit %s is not contained in any remote branch", ref)
		}

		// Checkout the branch and reset to commit to preserve HEAD on that commit
		if e := runAndLog(module.ID, exec.Command("git", "-C", repoDir, "checkout", branch)); e != nil {
			return e
		}
		if e := runAndLog(module.ID, exec.Command("git", "-C", repoDir, "reset", "--hard", ref)); e != nil {
			return e
		}

		// Set upstream to origin/<branch>
		_ = runAndLog(module.ID, exec.Command("git", "-C", repoDir, "branch", "--set-upstream-to=origin/"+branch, branch))

		// Persist detected branch in DB
		_, _ = database.PatchModule(database.ModulePatch{ID: module.ID, GitBranch: &branch})

	} else {
		// It's a branch name
		branch := ref
		if strings.HasPrefix(ref, "origin/") {
			branch = strings.TrimPrefix(ref, "origin/")
		}

		// Check if local branch exists
		err := exec.Command("git", "-C", repoDir, "rev-parse", "--verify", "refs/heads/"+branch).Run()
		if err != nil {
			// Local branch doesn't exist, create and track remote
			if e := runAndLog(module.ID, exec.Command("git", "-C", repoDir, "checkout", "-b", branch, "--track", "origin/"+branch)); e != nil {
				return e
			}
		} else {
			// Local branch exists, just checkout
			if e := runAndLog(module.ID, exec.Command("git", "-C", repoDir, "checkout", branch)); e != nil {
				return e
			}
			_ = runAndLog(module.ID, exec.Command("git", "-C", repoDir, "branch", "--set-upstream-to=origin/"+branch, branch))
		}

		// Persist selected branch in DB
		_, _ = database.PatchModule(database.ModulePatch{ID: module.ID, GitBranch: &branch})
	}

    // Persist current HEAD immediately, then recompute full metadata
    _ = persistHeadCommit(module)
    now := time.Now().UTC()
    _ = updateGitComputed(module, &now, nil)
    broadcastGitStatus(module)
    return nil
}

// updateGitComputed recomputes commit metadata and behind counts and patches DB.
func updateGitComputed(module Module, updatedAt *time.Time, pulledAt *time.Time) error {
    repoDir := repoDirFor(module)
    _ = ensureSafeDirectory(module, repoDir)
    // Temp SSH for any fetch during computation
    sshCommand, cleanup, _ := tempSSHForModule(module)
    if cleanup != nil { defer cleanup() }
    up := ""
    if out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}").CombinedOutput(); err == nil {
        up = string(bytesTrimSpace(out))
    }
	if up == "" && module.GitBranch != "" {
		up = "origin/" + module.GitBranch
	}
    if up != "" {
        parts := strings.SplitN(up, "/", 2)
        if len(parts) == 2 {
            c := exec.Command("git", "-C", repoDir, "fetch", parts[0], parts[1])
            if sshCommand != "" { c.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand) }
            _ = runAndLog(module.ID, c)
        }
    }
	headHashB, _ := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").CombinedOutput()
	headHash := string(bytesTrimSpace(headHashB))
	headSubjB, _ := exec.Command("git", "-C", repoDir, "log", "-1", "--pretty=%s").CombinedOutput()
	headSubj := string(bytesTrimSpace(headSubjB))
	latestHash := ""
	latestSubj := ""
	if up != "" {
		latestLine, _ := exec.Command("git", "-C", repoDir, "log", "-1", up, "--pretty=%H%x1f%s").CombinedOutput()
		s := string(bytesTrimSpace(latestLine))
		if s != "" {
			arr := []string{}
			cur := ""
			for i := 0; i < len(s); i++ {
				if s[i] == 0x1f {
					arr = append(arr, cur)
					cur = ""
				} else {
					cur += string(s[i])
				}
			}
			arr = append(arr, cur)
			if len(arr) >= 2 {
				latestHash = arr[0]
				latestSubj = arr[1]
			}
		}
	}
	behind := 0
	if up != "" {
		cntB, _ := exec.Command("git", "-C", repoDir, "rev-list", "--left-right", "--count", "HEAD..."+up).CombinedOutput()
		var left, right int
		fmt.Sscanf(string(bytesTrimSpace(cntB)), "%d %d", &left, &right)
		behind = right
	}
	patch := database.ModulePatch{ID: module.ID}
	if updatedAt != nil {
		patch.LastUpdate = updatedAt
	}
	if pulledAt != nil {
		patch.GitLastPull = pulledAt
	} else if updatedAt != nil {
		patch.GitLastFetch = updatedAt
	}
	patch.LateCommits = &behind
	if headHash != "" {
		patch.CurrentCommitHash = &headHash
	}
	if headSubj != "" {
		patch.CurrentCommitSubject = &headSubj
	}
	if latestHash != "" {
		patch.LatestCommitHash = &latestHash
	}
	if latestSubj != "" {
		patch.LatestCommitSubject = &latestSubj
	}
    _, err := database.PatchModule(patch)
    return err
}

// RefreshModuleGitSnapshot recomputes commit metadata without touching fetch/pull timestamps.
// It keeps DB in sync with the working directory so reads (GetModule) are accurate.
func RefreshModuleGitSnapshot(module Module) error {
    return updateGitComputed(module, nil, nil)
}

// HostRepoDir exposes the absolute repository path for a module (for API helpers).
func HostRepoDir(module Module) string { return repoDirFor(module) }

// EnsureSafeDir wraps ensureSafeDirectory to make it callable from API layer.
func EnsureSafeDir(module Module, repoDir string) error { return ensureSafeDirectory(module, repoDir) }

// BytesTrimSpace exposes bytesTrimSpace for API helpers.
func BytesTrimSpace(b []byte) []byte { return bytesTrimSpace(b) }

// GitBehind returns how many commits HEAD is behind its upstream.
func GitBehind(module Module) (int, error) {
    repoDir := repoDirFor(module)
    if err := ensureSafeDirectory(module, repoDir); err != nil { /* ignore */ }
    sshCommand, cleanup, _ := tempSSHForModule(module)
    if cleanup != nil { defer cleanup() }
    up := ""
    if out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}").CombinedOutput(); err == nil {
        up = string(bytesTrimSpace(out))
    }
    if up == "" && module.GitBranch != "" { up = "origin/" + module.GitBranch }
    if up == "" { return 0, nil }
    // Make sure upstream is fetched
    c := exec.Command("git", "-C", repoDir, "fetch", "--all", "--prune")
    if sshCommand != "" { c.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCommand) }
    _ = c.Run()
    out, err := exec.Command("git", "-C", repoDir, "rev-list", "--left-right", "--count", "HEAD..."+up).CombinedOutput()
    if err != nil { return 0, err }
    var left, right int
    fmt.Sscanf(string(bytesTrimSpace(out)), "%d %d", &left, &right)
    return right, nil
}

// persistHeadCommit writes only the current HEAD hash/subject to DB snapshot.
func persistHeadCommit(module Module) error {
    repoDir := repoDirFor(module)
    _ = ensureSafeDirectory(module, repoDir)
    headHashB, _ := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").CombinedOutput()
    headHash := string(bytesTrimSpace(headHashB))
    headSubjB, _ := exec.Command("git", "-C", repoDir, "log", "-1", "--pretty=%s").CombinedOutput()
    headSubj := string(bytesTrimSpace(headSubjB))
    patch := database.ModulePatch{ID: module.ID}
    if headHash != "" { patch.CurrentCommitHash = &headHash }
    if headSubj != "" { patch.CurrentCommitSubject = &headSubj }
    _, err := database.PatchModule(patch)
    return err
}

func GitCheckoutFile(module Module, path, ref string) error {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	args := []string{"-C", repoDir, "checkout"}
	if ref != "" {
		args = append(args, ref)
	}
	args = append(args, "--", path)
	cmd := exec.Command("git", args...)
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	broadcastGitStatus(module)
	return nil
}

func GitResolveOurs(module Module, path string) error {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	cmd := exec.Command("git", "-C", repoDir, "checkout", "--ours", "--", path)
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	cmdAdd := exec.Command("git", "-C", repoDir, "add", "--", path)
	if err := runAndLog(module.ID, cmdAdd); err != nil {
		return err
	}
	broadcastGitStatus(module)
	return nil
}

func GitResolveTheirs(module Module, path string) error {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	cmd := exec.Command("git", "-C", repoDir, "checkout", "--theirs", "--", path)
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	cmdAdd := exec.Command("git", "-C", repoDir, "add", "--", path)
	if err := runAndLog(module.ID, cmdAdd); err != nil {
		return err
	}
	broadcastGitStatus(module)
	return nil
}

func GitCreateBranch(module Module, name, from string) error {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	args := []string{"-C", repoDir, "checkout", "-b", name}
	if from != "" {
		args = append(args, from)
	}
	cmd := exec.Command("git", args...)
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	broadcastGitStatus(module)
	return nil
}

func GitDeleteBranch(module Module, name string) error {
	repoDir := repoDirFor(module)
	_ = ensureSafeDirectory(module, repoDir)
	cmd := exec.Command("git", "-C", repoDir, "branch", "-D", name)
	if err := runAndLog(module.ID, cmd); err != nil {
		return err
	}
	broadcastGitStatus(module)
	return nil
}

// broadcastGitStatus computes the current git status and sends it over WS
func broadcastGitStatus(module Module) {
	st, err := GitStatusModule(module)
	if err != nil {
		return
	}
    payload := map[string]any{
        "is_merging": st.IsMerging,
        "conflicts":  st.Conflicts,
        "modified":   st.Modified,
        "branch":     st.Branch,
        "head":       st.Head,
        "head_subject": st.HeadSubject,
        "last_pull":  st.LastPull,
        "last_fetch": st.LastFetch,
        "latest_hash": st.LatestHash,
        "latest_subject": st.LatestSubject,
        "behind":     st.Behind,
    }
	websocket.SendGenericModuleEvent(module.ID, "git_status", payload)
}

// small helpers
func bytesTrimSpace(b []byte) []byte {
	i := 0
	j := len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') {
		j--
	}
	return b[i:j]
}

// ensureSafeDirectory adds the repository path to git's safe.directory list to
// avoid "dubious ownership" failures when running inside containers.
func ensureSafeDirectory(module Module, repoDir string) error {
    if repoDir == "" {
        return nil
    }
    // Mark as safe directory (ignore error if already set)
    _ = exec.Command("git", "config", "--global", "--add", "safe.directory", repoDir).Run()
    // Ensure auto setup of remote tracking on push for new branches
    _ = exec.Command("git", "config", "--global", "push.autoSetupRemote", "true").Run()
    return nil
}

// tempSSHForModule creates a temporary private key file from the module's
// SSHPrivateKey and returns an ssh command string and a cleanup function.
func tempSSHForModule(module Module) (string, func(), error) {
    tmpKey, err := os.CreateTemp("", "id_rsa_")
    if err != nil {
        return "", nil, err
    }
    if err := os.WriteFile(tmpKey.Name(), []byte(module.SSHPrivateKey), 0600); err != nil {
        _ = os.Remove(tmpKey.Name())
        return "", nil, err
    }
    sshCommand := "ssh -i " + tmpKey.Name() + " -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes"
    cleanup := func() { _ = os.Remove(tmpKey.Name()) }
    return sshCommand, cleanup, nil
}
