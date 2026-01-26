# Docker — Pan Bagnat

This doc explains the containers that make up Pan Bagnat, how they connect, and how to work with them in development.

Contents
- Services (containers)
- Networks
- Volumes and repo mounts
- Common workflows (bring up, rebuild, prune)

## Services (containers)

Defined in `docker-compose.yml`:

- `frontend` (container: `pan-bagnat-frontend`)
  - Vite dev/build container for the React app.
  - Exposes port `3000` to the internal network; served by Nginx in dev/prod.

- `backend` (container: `pan-bagnat-backend`)
  - Go API server exposing port `8080` internally.
  - Environment: `POSTGRES_URL`, 42 OAuth creds, `REPO_BASE_PATH`, etc. (see `.env.example`).
  - Mounts docker.sock read‑only to manage module containers and `./repos` to `/data/repos` for cloning modules.

- `nginx`
  - Fronts the stack and terminates TLS.
  - Proxies:
    - `/` → frontend
    - `/api`, `/auth`, `/ws` → backend
    - `*.modules.<domain>` → proxy-service (module iframes via subdomains)
  - Reads config from `nginx/nginx.conf` and certs from `nginx/ssl`.

- `proxy-service` (container: `pan-bagnat-proxy-service`)
  - Lightweight Go HTTP proxy (no Docker socket) that serves `*.modules.<domain>` traffic.
  - Looks up module pages in Postgres, enforces `IframeOnly`/`NeedAuth` flags, shares the `session_id` cookie with the SPA, and forwards to the corresponding gateway container on the shared proxy network.
  - If a user is not authenticated and the browser accepts HTML, it redirects to `MODULES_LOGIN_URL` with the module URL in `next`, otherwise it returns JSON `{code:"unauthorized"}`. After login the SPA pre-warms the module cookie via `/api/v1/modules/pages/{slug}/session`.
  - Connected to both `pan-bagnat-core` (for DB/HTTP ingress) and `pan-bagnat-proxy-net` (to reach gateways). Configuration keys: `MODULES_PROXY_ALLOWED_DOMAINS`, `MODULES_IFRAME_ALLOWED_HOSTS`, `MODULES_SESSION_SECRET`, `MODULES_LOGIN_URL`, etc.

- `net-controller` (container: `pan-bagnat-net-controller`)
  - Reconciliation loop with Docker socket access; ensures a `gateway-<slug>` container exists per module page and reflects changes in `target_container` / `target_port` in real time.
  - Connects each gateway to the shared proxy network + the selected module network, rendering a per-page nginx reverse proxy pointed at the container/port defined in `module_page`. Gateways preserve the browser host header while telling modules which upstream container served the request via `X-Upstream-Host`.
  - Exposes an internal HTTP API consumed by `proxy-service` for status/reattach actions and surfaces per-page connectivity info in the admin UI.

- `db` (container: `pan-bagnat-db`)
  - PostgreSQL 16 with healthcheck.
  - Data persisted in the `postgres_data` volume.

- `migrator`
  - One‑shot `migrate/migrate` container mounting `db/migrations` to apply schema on startup.
  - Depends on `db` being healthy.

## Networks

- `pan-bagnat-core` (bridge, internal: true)
  - Private network for core services (`frontend`, `backend`, `db`, `nginx`, `migrator`).
  - Used by Nginx to route to backend/frontend.

- `pan-bagnat-net` (bridge, external: true)
  - External network used by module containers spawned via `docker compose` inside module repos.
  - Lets the core backend communicate with module containers and enables cross‑container DNS by name.

- `pan-bagnat-host` (bridge)
  - Additional network for Nginx with `extra_hosts: host.docker.internal` convenience.

- `pan-bagnat-proxy-net` (bridge, internal: true)
  - Shared “modules” network used by `proxy-service` and every generated `gateway-<slug>` container.
  - Provides a collision-free DNS space so `proxy-service` can hit `gateway-<slug>` without joining module networks.

## Volumes and repo mounts

- `postgres_data` — persistent Postgres data volume.
  - `./repos` → `/data/repos` (on `backend`) — where module Git repos are checked out. Each module uses its own `docker-compose.yml` and containers attach to `pan-bagnat-net`.

## Common workflows

Bring up core services
- `make up` — ensures the `pan-bagnat-net` exists, then `docker compose up -d`.

Build images
- `make build` — build all; `make build-back`; `make build-front`.

Tear down/prune
- `make prune` — stop core containers and prune images. Removes `pan-bagnat-net` if unused.
- `make fprune` — stronger cleanup: stops/removes volumes and orphans across the repo, prunes system resources, and clears `./repos`.

Modules up/down
- `make up-modules` / `make down-modules` — iterate repos in `./repos` and run `docker compose up|down` when `docker-compose.yml` exists.

Notes
- The backend mounts `/var/run/docker.sock` read‑only to perform `docker` operations for modules (build/up/down, container logs, etc.).
- Module modules run inside their own docker-compose projects and attach to the external `pan-bagnat-net`. Gateways bridge those networks to the shared proxy net so we never join module stacks directly to the public edge.
- Local wildcard DNS for dev is handled by dnsmasq (see `localDNS/README.md`). Production relies on public DNS entries such as `*.modules.panbagnat.42nice.fr`.
- Ensure your host user can access the Docker socket or run Docker Desktop with appropriate sharing.

Links
- Root overview: ./README.md
- Backend details: ./backend/README.md
- Proxy stack details: ./modules-proxy/README.md
