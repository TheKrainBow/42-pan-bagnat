# Image used by the admin "Shell" tab of a module (see backend/srcs/core/shell.go).
# The container never gets the real docker socket: DOCKER_HOST points to a
# per-session proxy (core/shell_docker_proxy.go) scoped to the module's compose project.
FROM ubuntu:22.04

ARG GO_VERSION=1.25.5
ARG NODE_MAJOR=22
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl wget gnupg lsb-release git openssh-client \
    build-essential clang clangd clang-format lldb gdb make cmake pkg-config valgrind \
    zsh vim nano less tree jq htop procps unzip zip tar rsync file locales \
    python3 python3-pip python3-venv \
  && rm -rf /var/lib/apt/lists/* \
  && locale-gen en_US.UTF-8

# Docker CLI + compose plugin (no daemon)
RUN mkdir -p /etc/apt/keyrings \
  && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
  && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
     > /etc/apt/sources.list.d/docker.list \
  && apt-get update && apt-get install -y --no-install-recommends docker-ce-cli docker-compose-plugin docker-buildx-plugin \
  && rm -rf /var/lib/apt/lists/*

# Go
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-$(dpkg --print-architecture).tar.gz" | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"

# Node.js, npm, pnpm, vite
# Versions pinned (pnpm/vite as in frontend/package.json): an unpinned install
# breaks the build whenever a freshly published release is not served yet.
RUN curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash - \
  && apt-get install -y --no-install-recommends nodejs \
  && rm -rf /var/lib/apt/lists/* \
  && npm install -g pnpm@10.12.1 yarn@1 vite@7

# zsh + oh-my-zsh (default robbyrussell theme)
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended \
  && chsh -s /usr/bin/zsh root

ENV LANG=en_US.UTF-8 \
    TERM=xterm-256color \
    SHELL=/usr/bin/zsh

WORKDIR /workspace
CMD ["/usr/bin/zsh"]
