#!/usr/bin/env bash
# Forwards a Claude Code hook event (JSON on stdin) to Pan Bagnat through the
# per-container unix socket. It must never block nor fail claude.
# The last event is also kept locally so that PB can recover the status of a
# session after a backend restart (events sent while it was down are lost).
echo "${1:-unknown}" > /tmp/pb-last-event 2>/dev/null
curl -sS --max-time 3 --unix-socket /run/pb/pb.sock \
  -H "Authorization: Bearer ${PB_HOOK_TOKEN:-}" \
  -H 'Content-Type: application/json' \
  --data-binary @- "http://pb/hook?event=${1:-unknown}" >/dev/null 2>&1 || true
exit 0
