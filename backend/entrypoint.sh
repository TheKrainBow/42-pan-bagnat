# backend/entrypoint.sh
#!/bin/sh
set -e

if [ -d /data/repos ]; then
  chown -R app:app /data/repos
fi

# exec your binary
exec ./main "$@"
