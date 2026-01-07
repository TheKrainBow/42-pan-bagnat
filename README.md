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
- Expose module pages through a reverse-proxied `/module-page/{slug}` system
- Manage users and roles, including rule-based assignments
- Authenticate via 42 OAuth
- Provide real-time updates through WebSockets

────────────────────────────────────────────────────────────
## Project Structure
────────────────────────────────────────────────────────────

- Backend: Go API (clean architecture, REST + WebSocket)
- Frontend: React application
- Database: PostgreSQL with migrations
- Infrastructure: Docker & Nginx

Quick links:
- Backend architecture: backend/README.md
- Frontend overview: frontend/README.md
- Docker & networks: DOCKER.md

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
