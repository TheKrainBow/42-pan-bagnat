# Image used by the admin "Claude" tab of a module (see backend/srcs/core/claude.go).
# Runs the official `claude` CLI inside tmux, on the module repo mounted at the
# same absolute path as on the host. Each user logs in with their own Claude
# account (/login); credentials live in a per-user home volume, never in PB.
# The container never gets the real docker socket: DOCKER_HOST points to a
# per-module proxy (core/shell_docker_proxy.go) scoped to the module's compose
# project, which cannot create containers nor build images: builds and deploys
# go through PB with the `pb` command (logged in the module logs).
FROM node:22-bookworm-slim

# Pinned on purpose: bump it deliberately, the auto-updater is disabled.
ARG CLAUDE_CODE_VERSION=2.1.312
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl gnupg git openssh-client tmux jq ripgrep procps less vim-tiny \
    make python3 locales \
  && rm -rf /var/lib/apt/lists/* \
  && sed -i 's/^# *en_US.UTF-8/en_US.UTF-8/' /etc/locale.gen && locale-gen

# Docker CLI + compose/buildx plugins (no daemon)
RUN install -m 0755 -d /etc/apt/keyrings \
  && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
  && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian bookworm stable" \
     > /etc/apt/sources.list.d/docker.list \
  && apt-get update && apt-get install -y --no-install-recommends docker-ce-cli docker-compose-plugin docker-buildx-plugin \
  && rm -rf /var/lib/apt/lists/*

RUN npm install -g "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" && npm cache clean --force

# The container runs as the owner of the module repo, whose UID is only known at
# runtime: the entrypoint registers it in /etc/passwd (ssh needs an entry), and
# the home (a per-user volume) is world-writable with the sticky bit.
RUN chmod 0666 /etc/passwd /etc/group \
  && mkdir -p /home/claude && chmod 1777 /home/claude

COPY pb/ /opt/pb/
COPY managed-settings.json /etc/claude-code/managed-settings.json
COPY prod-notice.md /etc/pb/prod-notice.md
# Managed CLAUDE.md: cannot be overridden by the repo or the user.
RUN sed 's/`<slug>`/courant (variable d'"'"'environnement `PB_MODULE`)/' /etc/pb/prod-notice.md > /etc/claude-code/CLAUDE.md \
  && chmod 0755 /opt/pb/*.sh /opt/pb/pb && ln -s /opt/pb/pb /usr/local/bin/pb \
  && chmod 0644 /etc/claude-code/* /etc/pb/prod-notice.md

ENV LANG=en_US.UTF-8 \
    TERM=xterm-256color \
    HOME=/home/claude \
    DISABLE_AUTOUPDATER=1 \
    PB_ENV=production

ENTRYPOINT ["/opt/pb/entrypoint.sh"]
