# backend/entrypoint.sh
#!/bin/sh
set -e

if [ -d /data/repos ]; then
  chown -R app:app /data/repos
fi

ssh-keyscan github.com > ./known_hosts
ssh-keyscan gitlab-world.42.fr >> ./known_hosts

# exec your binary
exec ./main "$@"
