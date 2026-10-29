#!/usr/bin/env bash
# Container main process: keeps a tmux session running `claude`. The container
# lives as long as that session does, whether or not a browser is attached.
set -euo pipefail

if ! getent passwd "$(id -u)" >/dev/null; then
  echo "claude:x:$(id -u):$(id -g):Claude:${HOME}:/bin/bash" >> /etc/passwd
fi
if ! getent group "$(id -g)" >/dev/null; then
  echo "claude:x:$(id -g):" >> /etc/group
fi

# claude refuses a temp dir it does not own (/tmp/claude-<uid> may be created
# by docker as root when the repo is mounted under /tmp).
CLAUDE_CODE_TMPDIR="$(mktemp -d /tmp/pb-claude.XXXXXX)"
export CLAUDE_CODE_TMPDIR

cd "$PB_WORKDIR"
tmux -f /opt/pb/tmux.conf new-session -d -s claude -x 200 -y 50 /opt/pb/run-claude.sh
# docker stop (the Stop button): hang up claude so that it exits cleanly
# (SessionEnd hook, transcript flushed) before the container goes away.
trap 'tmux kill-session -t claude 2>/dev/null' TERM INT
while tmux has-session -t claude 2>/dev/null; do sleep 1 & wait $!; done
