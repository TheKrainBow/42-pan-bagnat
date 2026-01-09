# backend/entrypoint.sh
#!/bin/sh
set -e

if [ -d /data/repos ]; then
  chown -R app:app /data/repos
fi

KNOWN_HOSTS_OUTPUT=./known_hosts
# Check .env.example to see expected input
SSH_KNOWN_HOSTS_LIST=$(printf '%s' "${SSH_KNOWN_HOSTS_LIST:-}")
> "$KNOWN_HOSTS_OUTPUT"

if [ -n "$SSH_KNOWN_HOSTS_LIST" ]; then
  printf '%s' "$SSH_KNOWN_HOSTS_LIST" | tr ',' '\n' | while IFS= read -r entry || [ -n "$entry" ]; do
    entry="$(printf '%s' "$entry" | sed 's/^[[:space:]]\+//;s/[[:space:]]\+$//')"
    [ -z "$entry" ] && continue
    case "$entry" in
      '#'* ) continue ;;
    esac

    host="${entry%%:*}"
    port="${entry#*:}"

    if [ "$port" = "$entry" ] || [ -z "$port" ]; then
      ssh-keyscan "$host" >> "$KNOWN_HOSTS_OUTPUT" || true
    else
      ssh-keyscan -p "$port" "$host" >> "$KNOWN_HOSTS_OUTPUT" || true
    fi
  done
else
  echo "[entrypoint] SSH_KNOWN_HOSTS_LIST is empty; known_hosts remains empty." >&2
fi

# exec your binary
exec ./main "$@"
