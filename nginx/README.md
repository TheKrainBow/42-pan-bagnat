# Nginx — Pan Bagnat Reverse Proxy

This README explains how Nginx fronts the stack, which routes it proxies, TLS details, headers it sets for the backend, and useful tips for local development and troubleshooting.

Contents
- Purpose and topology
- Routes and upstreams
- TLS and certificates
- Forwarded headers and cookies
- WebSockets and timeouts
- Module page proxy specifics
- Logs, reload, and troubleshooting

## Purpose and topology

Nginx sits in front of the frontend (Vite/build output) and backend (Go API) and exposes a single public entrypoint on ports 80/443. It terminates TLS, forwards requests to the correct upstream, and handles WebSocket upgrades.

- Config: `nginx/nginx.conf`
- Docker image: `nginx:alpine`
- Mounted certs: `nginx/ssl` → `/etc/ssl`

## Routes and upstreams

Defined upstreams:
- `pan-bagnat-frontend` → the Vite/React app on port 3000
- `pan-bagnat-backend` → the Go API on port 8080

Key locations:
- `/` → `pan-bagnat-frontend` (serves the SPA)
- `/api/` → `pan-bagnat-backend`
- `/auth/` → `pan-bagnat-backend`
- `/ws` → `pan-bagnat-backend` (WebSocket)
- `^~ /module-page/` → `pan-bagnat-backend` (reverse proxy of module pages)

These paths are defined in `server { … }` for the TLS vhost. Port 80 vhost only redirects to HTTPS.

## TLS and certificates

The TLS vhost is bound to:
```
listen 443 ssl;
server_name localhost heinz.42nice.fr panbagnat.42nice.fr;

ssl_certificate     /etc/ssl/fullchain.pem;
ssl_certificate_key /etc/ssl/42nice.fr.key;
```

- For local development, you can place dev certs under `nginx/ssl` that match the names above. In Docker, that folder is mounted read‑only into the container.
- The HTTP vhost redirects all traffic to HTTPS (301).

## Forwarded headers and cookies

For `/api/`, `/auth/`, `/ws`, and `/module-page/`, the proxy sets standard forwarding headers:
- `X-Forwarded-Proto`, `X-Forwarded-Host`, `X-Forwarded-For`, and `Host`.

The backend uses `X-Forwarded-Proto` to decide whether to mark the `session_id` cookie as `Secure`. Make sure TLS terminates at Nginx and that this header is present so cookies behave correctly in browsers.

## WebSockets and timeouts

The `/ws` location upgrades connections:
```
proxy_set_header   Upgrade $http_upgrade;
proxy_set_header   Connection $connection_upgrade;
proxy_read_timeout 600s;
```

This keeps WebSocket connections alive for live updates/log events.

## Module page proxy specifics

The `^~ /module-page/` location proxies module front‑ends that are served by modules behind the backend. It also:
- Hides any upstream `Access-Control-Allow-Origin` header (`proxy_hide_header`)
- Adds `Access-Control-Allow-Origin $http_origin` to mirror the caller (for iframe use in the SPA)

Authorization for module pages is enforced in the backend (`PageAccessMiddleware`) which checks if the page is public or whether the request originates from the SPA (via Referer) and user session.

## Logs, reload, and troubleshooting

Logs inside the container:
- Error log: `/var/log/nginx/error.log`
- Access log: not explicitly set in `nginx.conf`; you can enable if needed.

Validate config:
- `docker exec -it <nginx_container> nginx -t`

Reload without restart:
- `docker exec -it <nginx_container> nginx -s reload`

Common issues:
- Mixed content or cookies not sticking → ensure you’re using HTTPS and `X-Forwarded-Proto` is set to `https`.
- WebSocket disconnects → check `proxy_read_timeout` and network policies.
- 404s on SPA routes → ensure `/` is routed to the frontend and the frontend build/dev server handles client‑side routes.

Links
- Root overview: ../README.md
- Backend details: ../backend/README.md
