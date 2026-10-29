# Module "Claude" tab

The **Claude** tab of a module page (admins only) opens a terminal running the
official `claude` CLI (Claude Code) on the module repository, in production.
Each admin logs in with **their own Claude account** (`/login` in the terminal):
Pan Bagnat never reads, stores nor proxies Claude credentials.

## How it works

```
browser ──WS──> backend ──docker exec──> tmux attach ─┐
                                                      │  container pb-claude-<module>-<user>
                     /run/pb/pb.sock  <── hooks ── claude (in tmux)
                     /run/pb/docker.sock (scoped proxy)
```

- **One container per module** (`module-claude/` image), at most one active
  session per module (unique index on `claude_sessions`). Another admin sees who
  holds it and since when.
- `claude` runs **inside tmux**: closing the tab only detaches, Claude keeps working.
- The repo is mounted **at the same absolute path as on the host** (`REPO_HOST_BASE_PATH/<slug>`).
- The container runs as the **owner of the repo** (uid/gid of its root directory),
  with every capability dropped. If that owner is root, it keeps only
  `CHOWN DAC_OVERRIDE FOWNER FSETID KILL` so it can edit every file of the repo.
- `$HOME` is the per-user volume `pb-claude-home-<user_id>` (credentials and
  transcripts), shared by that user's modules and never mounted anywhere else.
- Network: `pan-bagnat-claude` (bridge, outbound only, no container-to-container traffic).

The backend serves two unix sockets per container, in `REPO_BASE_PATH/.pb-claude/<slug>/`
(bind-mounted on `/run/pb`). Both are re-created after a backend restart:

| Socket | Used by | Purpose |
|--------|---------|---------|
| `docker.sock` | `docker` CLI | Same scoped proxy as the Shell tab: only the module's compose project (ps, logs, inspect, restart, stop, exec…). No container creation or image build. |
| `pb.sock` | `/opt/pb/hook.sh`, `pb` | Claude Code hooks (status) and `pb deploy\|rebuild\|down`, authenticated by a per-container token. |

`/run/pb/ssh/` holds the module deploy key (read-only) and PB's known_hosts, used by
`GIT_SSH_COMMAND` for `git fetch/pull/push`. Commits are signed with the admin's PB login and email.

### Builds and deploys

`docker compose up/build/down`, `docker build/run` are refused. Claude runs
`pb deploy` (build + up -d), `pb rebuild` (no cache) or `pb down` instead: the
backend runs the same code as the Deploy / Rebuild / Down buttons, so the output
lands in the **module logs**, and is streamed back to Claude.

## Statuses

| Status | Set by | Auto-removed? |
|--------|--------|---------------|
| `starting` | container created (login screen until the first `SessionStart`) | like `idle` |
| `idle` | `SessionStart`, `Stop`, `StopFailure`, idle reminder after an interrupt | after `CLAUDE_IDLE_TTL` without any browser attached |
| `working` | `UserPromptSubmit`, tool run after a permission prompt | never, below `CLAUDE_MAX_WORK_SECONDS` (then `aborted`) |
| `waiting_input` | `Notification` (permission prompt / question) during a task | never, below `CLAUDE_MAX_WORK_SECONDS` |
| `stopped` | container gone (user exit, stop button, idle TTL) | resumable |
| `aborted` | max work time reached, start failure | resumable |

The owner of a session gets a toast anywhere in PB (and a browser notification
when the tab is hidden) when Claude **waits for them** or **finishes a task**.
Until they come back, a **pastille** is shown on the module card (which then
opens the Claude tab) and on the Claude tab: orange and pulsing while Claude
waits for an answer, green when a task ended and its result has not been seen
(`unseen_result`, cleared when the owner displays the Claude tab or types a new
prompt). Only the owner sees it; other admins see the status line on the card.

While a session is open, the **IDE** and **Shell** tabs show a warning to every
admin ("Claude (login) is modifying this repository"): Claude edits the same
working tree from its container.

Only the owner of a session can end it: **Stop** lets claude exit cleanly
(10 s grace), **Kill** (on the session row) destroys the container right away,
for a session that hangs. If docker cannot remove the container, the session keeps
the module lock and the error is shown, so that two Claudes never run on one repo.

"Resume" recreates the container and runs `claude --resume <claude_session_id>`:
the transcript is in the user's home volume, under the same working directory.
Sessions older than `CLAUDE_SESSION_RETENTION_DAYS` are deleted with their transcript.

At boot, the backend reconciles: running containers are kept (sockets re-created,
last hook event read back from the container), sessions without a container are ended.

## Guardrails (in the image, not editable by users)

- `/etc/claude-code/managed-settings.json`: allow/ask/deny permission rules,
  status hooks, `PreToolUse` guard (`/opt/pb/guard.sh`), bypass mode disabled,
  only managed hooks.
- `/etc/claude-code/CLAUDE.md` and `--append-system-prompt`: production notice
  (`module-claude/prod-notice.md`).

Permission rules are not a security boundary: the boundary is the container
(scoped docker proxy, repo-only mounts, no capabilities).

## Configuration

See `.env.example` (`CLAUDE_*`, `MODULE_CLAUDE_IMAGE`). The Claude Code version is
pinned in `module-claude/Dockerfile` (`CLAUDE_CODE_VERSION`); the auto-updater is off.
