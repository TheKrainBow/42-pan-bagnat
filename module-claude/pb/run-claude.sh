#!/usr/bin/env bash
# Runs inside the tmux session. PB_RESUME_ID resumes a previous conversation.
notice="$(sed "s/<slug>/${PB_MODULE}/g" /etc/pb/prod-notice.md)"
args=()
# claude only writes the transcript on the first message: a session stopped
# before that (e.g. right after /login) has nothing to resume, so start a fresh
# conversation under the same id (PB keeps it bound to the same session).
if [ -n "${PB_RESUME_ID:-}" ]; then
  if find "$HOME/.claude/projects" -name "${PB_RESUME_ID}.jsonl" -print -quit 2>/dev/null | grep -q .; then
    args+=(--resume "$PB_RESUME_ID")
  else
    args+=(--session-id "$PB_RESUME_ID")
  fi
fi

claude "${args[@]}" --append-system-prompt "$notice"
code=$?

# Keep the error visible instead of killing the container right away.
if [ "$code" -ne 0 ]; then
  printf '\n\033[31mclaude exited with code %d.\033[0m Press Enter to end this session.\n' "$code"
  read -r _ || true
fi
