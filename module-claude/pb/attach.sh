#!/usr/bin/env bash
# Run by the backend (docker exec) for each terminal WebSocket. The pid file
# lets the backend detach exactly this client when the browser goes away
# (detach.sh): closing the exec stream alone does not always end it.
case "${PB_ATTACH_ID:-}" in
  ''|*[!a-f0-9]*) echo "invalid attach id" >&2; exit 1 ;;
esac
# The container may still be starting.
for _ in $(seq 1 50); do
  tmux has-session -t claude 2>/dev/null && break
  sleep 0.2
done
echo $$ > "/tmp/pb-attach-${PB_ATTACH_ID}"
exec tmux attach-session -t claude
