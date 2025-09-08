# Pan Bagnat

The all‑in‑one 42 campus toolbox for deploying, managing, and exposing feature “modules” (apps/tools) behind role‑based access control. It ships a Go backend, a React frontend, a Postgres database, and Nginx, with first‑class Docker support.

What you can do:
- Import modules from Git, auto‑generate SSH keys, and deploy with Docker Compose
- Define module pages and expose them at `/module-page/{slug}` (reverse‑proxied)
- Manage users and roles (including rule‑based/automatic assignments)
- Authenticate through 42 OAuth; sessions via cookies; real‑time updates via WebSocket

Quick links:
- Backend: backend/README.md
- Frontend: frontend/README.md
- Docker & networks: DOCKER.md
- Database & migrations: db/README.md (to be added)

Documentation roadmap (phase 1):
1. Backend: stack, clean architecture flow (API ↔ core ↔ database), tests, security, Swagger
2. Frontend: stack, dev requirements, interaction with backend (dev flow detailed later)
3. Docker: containers and networks (purposes, local dev tips)
4. Database: schema, ID strategy (ULID), migrations and how to edit schema

Usage docs (phase 2):
- Admin guide (importing modules, deploying, roles/pages)
- User guide (navigating modules/pages)

If you’re new here, start with backend/README.md for the architecture overview.
