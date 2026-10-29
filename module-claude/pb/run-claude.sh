#!/usr/bin/env bash
# Runs inside the tmux session. PB_RESUME_ID resumes a previous conversation.
notice="$(sed "s/<slug>/${PB_MODULE}/g" /etc/pb/prod-notice.md)"
args=()
[ -n "${PB_RESUME_ID:-}" ] && args+=(--resume "$PB_RESUME_ID")

claude "${args[@]}" --append-system-prompt "$notice"
code=$?

# Keep the error visible instead of killing the container right away.
if [ "$code" -ne 0 ]; then
  printf '\n\033[31mclaude exited with code %d.\033[0m Press Enter to end this session.\n' "$code"
  read -r _ || true
fi
