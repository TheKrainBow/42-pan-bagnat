# Database — Pan Bagnat

This guide documents the database used by Pan Bagnat, the core schema, ID conventions, and how migrations work.

Contents
- Engine and connection
- Schema overview (tables and relationships)
- ID strategy and constraints
- Migrations (how we evolve the schema)
- Test data and local tips

## Engine and connection

- Engine: PostgreSQL 16
- Container: `pan-bagnat-db` (see `docker-compose.yml`)
- Connection string: `POSTGRES_URL` (exported to backend and migration jobs). Example from `.env.example`:
  - `postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable`

## Schema overview

Baseline schema is defined in `db/migrations/01_baseline.up.sql`, then evolved by subsequent migrations.

Main tables
- `users` — 42 users known to the system
  - Columns: `id`, `ft_login`, `ft_id`, `ft_is_staff`, `photo_url`, `last_seen`
- `roles` — access control roles
  - Columns: `id`, `name`, `color`, `is_default`, plus `rules_json`, `rules_updated_at` (02), `is_protected` (03)
- `modules` — deployable feature modules
  - Columns: `id`, `name`, `slug`, `git_url`, `git_branch`, `ssh_public_key`, `ssh_private_key`, `version`, `status`, `icon_url`, `latest_version`, `late_commits`, `last_update`
- `module_page` — user‑facing pages for a module (reverse‑proxied)
  - Columns: `id`, `module_id`, `name`, `slug`, `network_name`, `target_container`, `target_port`, `iframe_only`, `need_auth`, `icon_url`; `(module_id, name)` unique
  - `target_container`/`target_port` tell the net-controller which Docker service to reach, `network_name` indicates which module network to attach to, and the boolean flags control whether the page can be opened outside Pan Bagnat (`iframe_only`) and whether proxy-service enforces authentication (`need_auth`).
- `module_log` — logs attached to a module (git/docker outputs, lifecycle messages)
  - Columns: `id`, `module_id`, `created_at`, `level`, `message`, `meta jsonb`
- `sessions` — user sessions (cookie `session_id`)
  - Baseline: `session_id`, `ft_login`, `created_at`, `expires_at`
  - (04) Adds per‑device metadata: `user_agent`, `ip`, `device_label`, `last_seen` + helpful indexes

Join tables
- `user_roles` — many‑to‑many users ↔ roles (`PRIMARY KEY (user_id, role_id)`)
- `module_roles` — many‑to‑many modules ↔ roles (`PRIMARY KEY (module_id, role_id)`)

Seeded/protected roles (03)
- `roles_blacklist` — blocks access; protected
- `roles_admin` — Pan Bagnat admin; protected
- `roles_default` — default role for new users

Staff flag migration (05)
- Migrates legacy `users.is_staff` → grants `roles_admin` to those users and removes the boolean. Admin is now strictly role‑based.

## ID strategy and constraints

- All first‑class entities use ULID with a type prefix, generated in backend core:
  - Users: `user_01…`
  - Roles: `role_01…`
  - Modules: `module_01…`
  - Pages: `page_01…`
- This keeps IDs sortable and unique across time and improves readability when debugging logs.

Other constraints
- `modules.slug` is globally unique.
- `module_page (module_id, name)` is unique (avoid duplicate page names under one module).
- Sessions are keyed by `session_id` and indexed for housekeeping queries.

## Migrations

Migration files live in `db/migrations` and follow the `NNNN_name.up.sql` / `NNNN_name.down.sql` convention.

How they run
- Docker Compose includes a one‑shot `migrator` service (image `migrate/migrate`) that mounts `db/migrations` and runs `up` against `POSTGRES_URL`.
- You can also run migrations locally via Make targets (see Makefile):
  - `make migrate-up` — apply all pending migrations
  - `make migrate-down1` — rollback the last migration
  - `make migrate-steps N=-2` — move N steps (negative to rollback)
  - `make migrate-goto V=2` — go to migration version V
  - `make migrate-version` — show current version
  - `make migrate-force V=1` — mark version (clears dirty), use with care
  - `make migrate-new NAME=my_change` — scaffold a new `000X_NAME.up.sql` and `.down.sql`

Guidelines for editing schema
- Never modify applied migration files; always add a new pair.
- Keep `up` idempotent where possible (e.g., `IF NOT EXISTS` for indexes) and ensure `down` is a safe inverse.
- When changing core tables (users, roles, modules), update the Go scan structs under `backend/srcs/database` if columns are added/removed.
- Test locally using `make migrate-up` and run backend tests (`make test-backend`).

## Test data and local tips

- Seed data: `db/test_data.sql` contains a small set you can load with `make db-test`.
- Inspect the DB:
  - `docker exec -it pan-bagnat-db-1 psql -U ${POSTGRES_USER:-admin} -d ${POSTGRES_DB:-panbagnat}`
- Prune/reset helpers (use with care): see `db-prune-safe`, `db-init-schema-safe`, `db-clear-data-safe` in the Makefile.

Links
- Root overview: ../README.md
- Backend details: ../backend/README.md
