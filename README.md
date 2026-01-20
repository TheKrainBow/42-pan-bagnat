# Pan Bagnat

Pan Bagnat is a community-driven, source-available toolbox designed for 42 campuses.
It provides a unified platform to deploy, manage, and expose internal tools (“modules”)
behind role-based access control.

The project focuses on long-term maintainability, security, and shared infrastructure
for campus-level tooling.

────────────────────────────────────────────────────────────
## What Pan Bagnat Does
────────────────────────────────────────────────────────────

- Deploy and manage feature modules from Git repositories
- Auto-generate SSH keys and orchestrate deployments with Docker Compose
- Expose module pages on dedicated subdomains (`https://<slug>.modules.domain`) via an internal gateway mesh (see “Module Domains”)
- Manage users and roles, including rule-based assignments
- Authenticate via 42 OAuth
- Provide real-time updates through WebSockets

────────────────────────────────────────────────────────────
## Project Structure
────────────────────────────────────────────────────────────

- Backend: Go API (clean architecture, REST + WebSocket)
- Frontend: React application
- Database: PostgreSQL with migrations
- Infrastructure: Docker & Nginx + dedicated module proxy services

Quick links:
- Backend architecture: backend/README.md
- Frontend overview: frontend/README.md
- Docker & networks: DOCKER.md
- Module proxy stack (proxy-service, net-controller, gateways): modules-proxy/README.md
- Local DNS helper for `*.modules.localhost`: localDNS/README.md

────────────────────────────────────────────────────────────
## Module Domains & Gateways
────────────────────────────────────────────────────────────

Modules are exposed strictly through DNS names shaped like
`https://<slug>.modules.<base-domain>`. The base domain differs per environment
(e.g. `modules.panbagnat.42nice.fr` in production, `modules.localhost` in dev).

Key ideas:
- The backend issues a `session_id` cookie scoped to `.panbagnat.42nice.fr`
  (or `.localhost`) so both the SPA and all module subdomains share the same
  authentication state.
- When a user opens a module directly, `proxy-service` validates the session,
  enforces iframe/standalone rules defined per page, and if needed redirects to
  `/login?next=<module-url>`. After logging in, the SPA replays the `next`
  target, warms the module session via `/api/v1/modules/pages/{slug}/session`,
  and the browser lands on the module again with cookies already set.
- Each module page points to a Docker container + port and optional module
  network. The `net-controller` spawns a `gateway-<slug>` container (nginx) that
  attaches to both the shared proxy network and the module network, so traffic
  never flows directly from the public edge into module containers.
- Local development uses a tiny dnsmasq instance (`localDNS/dnsmasq.conf`) to
  resolve `*.modules.localhost` to `127.0.0.1`, mirroring production DNS.

Full details live in `modules-proxy/README.md` and `localDNS/README.md`.

────────────────────────────────────────────────────────────
## License & Governance
────────────────────────────────────────────────────────────

Pan Bagnat is **source-available**, not open-source under OSI definitions.

- The code is publicly visible
- Commercial use and SaaS offerings are strictly prohibited
- Forks may not be distributed or monetized
- The project is centrally maintained to ensure coherence and sustainability

Contributions are welcome via Pull Request and are subject to a Contributor License Agreement (CLA).

────────────────────────────────────────────────────────────
## Roadmap
────────────────────────────────────────────────────────────

Phase 1 – Core documentation:
1. backend/README.md: architecture, security, tests, Swagger
2. frontend/README.md: structure and dev workflow
3. db/README.md: database schema and migration strategy

Phase 2 – Usage documentation:
- Admin guide
- User guide

If you are new to the project, start with backend/README.md.
