# Modules Proxy Stack

This package builds the services that expose module frontends on dedicated
sub‑domains such as `https://<slug>.modules.panbagnat.42nice.fr` or
`http://<slug>.modules.localhost`.

It contains two binaries:

- **proxy-service** — HTTP entrypoint that terminates traffic for
  `*.modules.<domain>`, enforces iframe/auth rules, exchanges module sessions,
  and forwards to an internal gateway.
- **net-controller** — Reconciliation loop that talks to Docker and maintains a
  `gateway-<slug>` container per module page. These gateways bridge the shared
  proxy network and the module’s own network.

Both services load module page definitions from Postgres via the `module_page`
table (slug, network, target container, target port, iframe/needAuth flags) and
listen to the `MODULES_PROXY_CHANNEL` notification to refresh in near real-time.

## Request flow

```
Browser (slug.modules.domain)
        │
        ▼
Nginx wildcard vhost → proxy-service
        │
        ├─ session/iframe checks (see below)
        └─ reverse proxy to http://gateway-<slug>:8080/
                     │
                     ▼
       gateway-<slug> container (nginx)
                     │
                     ▼
         module container + port defined in module_page
```

The gateway containers exist because the main proxy must never join arbitrary
module networks. Each gateway joins the shared `pan-bagnat-proxy-net` and the
module network(s) selected in the admin UI. From the proxy’s perspective, every
module host is reachable via DNS name `gateway-<slug>` regardless of which
module docker compose file spawned the actual workloads.

## Proxy Service

Location: `cmd/proxy-service`.

Responsibilities:
- Cache module pages (`pageStore`) and handle every virtual host
  `*.modules.<allowed domains>`:
  - Validates the slug exists and, if `NeedAuth` is enabled, looks up the user
    session from the shared `session_id` cookie (`SESSION_COOKIE_DOMAIN` should
    be `.localhost` or `.panbagnat.42nice.fr`).
  - Checks iframe rules: `IframeOnly` requires the `Referer` to be Pan Bagnat or
    the module itself.
  - Redirects unauthenticated browser requests to `MODULES_LOGIN_URL` with a
    `next` parameter while still returning JSON errors for API clients.
  - Issues short-lived HMAC-signed module session tokens
    (`POST /api/v1/modules/pages/{slug}/session` in the backend). The frontend
    trades that token for a cookie by hitting `<origin>/_pb/session`.
  - Proxies the HTTP/WebSocket stream to the gateway host for the slug,
    preserving the browser Host header and forwarding the upstream container
    name via `X-Upstream-Host`.

Config knobs (see `docker-compose.yml`):
- `MODULES_PROXY_ALLOWED_DOMAINS` — comma-separated list of base domains
  (e.g. `modules.panbagnat.42nice.fr,modules.localhost`) accepted from the Host
  header. Wildcard subdomains are implied.
- `MODULES_IFRAME_ALLOWED_HOSTS` — list of trusted parent hosts (Pan Bagnat
  SPA) allowed to embed module iframes.
- `MODULES_LOGIN_URL` — absolute URL to the SPA login page for redirecting
  unauthenticated users. The login handler uses the same env list to validate
  `next` targets.
- `MODULES_SESSION_SECRET` / `MODULES_SESSION_COOKIE_TTL` — signing key and TTL
  for module session tokens.

## Net Controller

Location: `cmd/net-controller`.

Responsibilities:
- Listen to the same Postgres NOTIFY channel and fetch `module_page` rows that
  have both a container and a port.
- Ensure a gateway container exists for every slug; remove gateways whose page
  has been deleted or no longer has routing info.
- Generate the gateway configuration (small nginx template) that proxies to the
  module’s declared container/port. It keeps `$host` when the target is served
  over HTTPS/TLS but rewrites it to `localhost` when a module uses
  `.modules.localhost` certificates.
- Attach the gateway to:
  1. `MODULES_SHARED_NETWORK` → shared network with `proxy-service`.
  2. The module network stored in `module_page.network_name` so it can reach the
     real container (e.g. `piscine-monitoring-net`).
- Report health/status via `/gateways/{slug}` (GET for cached info, POST to
  force a reconcile then return status).

Important env vars:
- `MODULES_SHARED_NETWORK` — usually `pan-bagnat-proxy-net`; must already exist.
- `MODULES_GATEWAY_IMAGE` — container image for gateways (defaults to
  `nginx:alpine`).
- `MODULES_GATEWAY_PORT` — internal port exposed by gateways (proxy-service
  always talks HTTP to this port).
- `MODULES_PROXY_CHANNEL` — same NOTIFY channel as the backend.
- Requires `/var/run/docker.sock` mounted read/write: the controller spins up,
  stops, and connects containers dynamically.

## Gateway containers

- Named `gateway-<slug>` (slug sanitized).
- Labels keep track of slug, target container:port, and the module network.
- Run `nginx` with a generated config that:
  - `proxy_pass http://<target-container>:<port>/`
  - Propagates Upgrade headers for WebSockets.
  - Sets `X-Forwarded-Host` to the browser host and `X-Upstream-Host` to the raw
    Docker target (handy for debugging module logs).
  - Forces `Host localhost` when matching `.modules.localhost` because Chrome
    dev certs typically expect localhost.

Because gateways are the only containers attached to the module networks, we
avoid leaking internal services to the core stack while still allowing a stable
entrypoint for each subdomain.

## Local development

- Use the provided `localDNS/dnsmasq.conf` to resolve `*.modules.localhost`
  to `127.0.0.1`. See `../localDNS/README.md` for instructions.
- Set `SESSION_COOKIE_DOMAIN=.localhost` so the backend session cookie applies
  to both `localhost` and `*.modules.localhost`.
- Configure `MODULES_PROXY_ALLOWED_DOMAINS=modules.localhost` and
  `MODULES_IFRAME_ALLOWED_HOSTS=localhost`.

With that setup, navigating directly to
`https://example.modules.localhost` loads the module without opening Pan Bagnat
first, and the login redirect flow knows how to return the user to the module
URL after authentication.
