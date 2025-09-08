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
    - `/api`, `/auth`, `/ws`, `/module-page/*` → backend
  - Reads config from `nginx/nginx.conf` and certs from `nginx/ssl`.

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

## Volumes and repo mounts

- `postgres_data` — persistent Postgres data volume.
- `./repos` → `/data/repos` (on `backend`) — where module Git repos are checked out. Each module can define its own `docker-compose-panbagnat.yml` and containers attach to `pan-bagnat-net`.

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
- Ensure your host user can access the Docker socket or run Docker Desktop with appropriate sharing.

Links
- Root overview: ./README.md
- Backend details: ./backend/README.md
