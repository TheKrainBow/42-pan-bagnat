#!/usr/bin/env bash
# PreToolUse hook on Bash: second line of defense behind the managed permission
# rules (which are not a security boundary). Exit code 2 blocks the command and
# stderr is shown to claude as the reason. The real boundary is the container
# itself (scoped docker proxy, repo-only mounts).
cmd="$(jq -r '.tool_input.command // empty' 2>/dev/null)"
[ -z "$cmd" ] && exit 0

block() {
  echo "Pan Bagnat (production): commande bloquée — $1. Explique à l'utilisateur ce que tu voulais faire et demande-lui de le faire lui-même s'il le juge nécessaire." >&2
  exit 2
}

# One regex per rule, matched against the whole command (chained commands,
# subshells and pipes included).
w='(^|[;&|(`$[:space:]])'

grep -Eq "${w}git([[:space:]]+-[^[:space:]]+)*[[:space:]]+push([[:space:]].*)?[[:space:]](-f|--force|--force-with-lease|--mirror|--delete|-d)([[:space:]=]|$)" <<<"$cmd" && block "git push forcé ou destructif"
grep -Eq "${w}git([[:space:]]+-[^[:space:]]+)*[[:space:]]+push([[:space:]].*)?[[:space:]]\+[^[:space:]]" <<<"$cmd" && block "git push forcé (+refspec)"
grep -Eq "${w}git([[:space:]]+-[^[:space:]]+)*[[:space:]]+reset([[:space:]].*)?[[:space:]]--hard" <<<"$cmd" && block "git reset --hard"
grep -Eq "${w}git([[:space:]]+-[^[:space:]]+)*[[:space:]]+clean([[:space:]]|$)" <<<"$cmd" && block "git clean"

# Builds and deploys go through PB so that they are logged on the module.
grep -Eq "${w}docker(-compose|[[:space:]]+compose)([[:space:]].*)?[[:space:]](up|build|create|run|down)([[:space:]]|$)" <<<"$cmd" \
  && block "les build/up/down passent par PB : utilise \`pb deploy\`, \`pb rebuild\` ou \`pb down\`"
grep -Eq "${w}docker([[:space:]]+(container|image|buildx|builder))?[[:space:]]+(build|run|create)([[:space:]]|$)" <<<"$cmd" \
  && block "les build/run passent par PB : utilise \`pb deploy\` ou \`pb rebuild\`"
grep -Eq "${w}docker[[:space:]]+(volume[[:space:]]+(rm|prune)|system[[:space:]]+prune|image[[:space:]]+prune|container[[:space:]]+prune)" <<<"$cmd" && block "suppression de volumes, images ou conteneurs en masse"
grep -Eq "${w}docker[[:space:]]+(container[[:space:]]+)?rm[[:space:]]+(.*[[:space:]])?(-f|--force|-[a-z]*f[a-z]*)([[:space:]]|$)" <<<"$cmd" && block "docker rm -f"
grep -Eq -- "--privileged|--pid[= ]host|--net(work)?[= ]host|--ipc[= ]host|--userns[= ]host|--uts[= ]host|--cap-add|--device|--security-opt" <<<"$cmd" && grep -Eq "${w}docker" <<<"$cmd" && block "option docker privilégiée"
grep -Eq "docker\.sock|/var/run/docker|/run/pb/(pb|docker)\.sock" <<<"$cmd" && block "accès direct aux sockets"

# Secrets: .env files, database dumps and the module deploy key.
grep -Eq "(^|[^[:alnum:]_])\.env([.[:space:]'\"]|$)" <<<"$cmd" \
  && grep -Eq "${w}(cat|less|more|head|tail|bat|strings|xxd|od|hexdump|grep|rg|awk|sed|cut|sort|uniq|base64|cp|mv|scp|curl|nc|source|\.)[[:space:]]" <<<"$cmd" \
  && block "lecture d'un fichier .env"
grep -Eq "\.dump([[:space:]'\"]|$)" <<<"$cmd" && block "accès à un dump de base"
grep -Eq "/run/pb/ssh|id_key" <<<"$cmd" && block "accès à la clé SSH du module"

# rm -r on anything outside the module repo.
if grep -Eq "${w}rm[[:space:]]+(-[a-zA-Z]*[rR][a-zA-Z]*|--recursive)" <<<"$cmd"; then
  grep -Eq "${w}rm[[:space:]].*[[:space:]](/|~|\\\$HOME|\.\.)([[:space:]/]|$)" <<<"$cmd" && block "rm -r hors du repo"
  for arg in $(grep -Eo "[[:space:]]/[^[:space:]]*" <<<"$cmd"); do
    case "$arg" in
      "$PB_WORKDIR"|"$PB_WORKDIR"/*|/tmp/*) ;;
      *) block "rm -r sur $arg, hors du repo du module" ;;
    esac
  done
fi

exit 0
