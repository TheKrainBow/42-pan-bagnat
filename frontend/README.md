# Frontend — Pan Bagnat

This doc covers the frontend stack, how to run it locally, the (upcoming) development flow, testing status, and how the SPA interacts with the backend.

Contents
- Stack and prerequisites
- Running locally
- Development flow (molecular UI — high level, detailed later)
- Testing (status and plan)
- Backend interaction (API, auth, WebSocket, module pages)

## Stack and prerequisites

- Framework: React + Vite
- Routing: React Router v6
- Styling: CSS modules and project styles (e.g., `App.css`, `Notifications.css`)
- State: Local React state + helper utilities
- Notifications: `react-toastify`
- WS client: simple wrapper (see `Global/SocketService`)
- Package manager: `pnpm`

Prereqs
- Node 18+ (LTS recommended)
- pnpm installed globally: `npm i -g pnpm`

## Running locally

Option A — via Makefile
- `make local-front` → `cd frontend && BROWSER=none pnpm dev`
- Exposes Vite dev server on port 3000 (see `docker-compose.yml`/Nginx for proxying).

Option B — manual
- `cd frontend`
- `pnpm install`
- `BROWSER=none pnpm dev`

Notes
- For authenticated flows, the backend must be running and reachable on the host used by CORS (see `HOST_NAME` in `.env.example`).
- In Docker, Nginx proxies `/` to the frontend, `/api` + `/auth` + `/ws` to the backend, `/module-page/_status` to `proxy-service`, and any `*.modules.<domain>` hostnames to the module `proxy-service`.

## Development flow (molecular UI)

We follow a “molecular” approach: small reusable atoms/molecules compose into pages.
- Global components: `src/Global/*` (icons, inputs, layout, helpers)
- Pages: `src/Pages/*` for feature areas (Users, Roles, Modules)
- Route shell: `src/App.jsx` wires the Sidebar and Routes

Initial rules
- Keep components small and cohesive; prefer composition over deep props.
- Co-locate simple styles next to components; reuse global styles where helpful.
- Keep network calls in page-level or service helpers (see `Global/utils/Auth.jsx`).

We will document conventions and folder layout in detail during the refactor.

## Testing (status and plan)

- Current status: no formal test suite.
- Planned: Vitest + React Testing Library for component tests and critical flows.
- Philosophy: start with targeted tests around business logic and critical UI (auth, navigation), then broaden.

## Backend interaction

Auth + fetch helper
- Use `fetchWithAuth` (`src/Global/utils/Auth.jsx`) for all API calls. It:
  - Sends `credentials: 'include'` to use the `session_id` cookie
  - On 401/403, shows a toast and redirects to `/login`
  - Provides friendly handling for 404/409

API endpoints
- Base paths used in the SPA:
  - `/api/v1/users/me` and `/api/v1/users/me/pages`
  - `/api/v1/admin/modules` and related CRUD
  - `/api/v1/admin/roles` and related CRUD
  - Integrations (42): `/api/v1/admin/integrations/42/users/{login}`

WebSocket
- Endpoint: `/ws`. The client subscribes per module to receive live events/logs.

Module Pages
- User-facing pages load from dedicated subdomains like `https://<slug>.modules.panbagnat.42nice.fr/`.
- `ModulePage.jsx` points the iframe to `<protocol>://<slug>.<modules-domain>/` where the domain comes from `VITE_MODULES_BASE_DOMAIN` (configured via Docker build arg or `frontend/.env` when running Vite locally).
- When the domain ends with `.nip.io` (e.g. `modules.127.0.0.1.nip.io`), the iframe automatically forces plain HTTP since no TLS cert exists; you can override the protocol explicitly via `VITE_MODULES_PROTOCOL=http|https` if needed.

Links
- Backend details: ../backend/README.md
- Root overview: ../README.md
