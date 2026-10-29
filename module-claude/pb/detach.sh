#!/usr/bin/env bash
# Detaches the tmux client started by attach.sh with the given id.
id="${1:-}"
case "$id" in
  ''|*[!a-f0-9]*) exit 1 ;;
esac
f="/tmp/pb-attach-${id}"
[ -f "$f" ] && kill -HUP "$(cat "$f")" 2>/dev/null
rm -f "$f"
exit 0
