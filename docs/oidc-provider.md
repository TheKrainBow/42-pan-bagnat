# Pan Bagnat OIDC Provider

Pan Bagnat exposes a generic OpenID Connect provider for external modules.
Each module gets its own OIDC client, managed from the admin UI.

## Endpoints

- `GET /.well-known/openid-configuration`
- `GET /.well-known/jwks.json`
- `GET /oauth/authorize`
- `POST /oauth/token`
- `GET /oauth/userinfo`

## Environment

```env
OIDC_ISSUER=https://panbagnat.example.com
OIDC_JWT_PRIVATE_KEY_PEM=...
OIDC_JWT_PRIVATE_KEY_PATH=
OIDC_JWT_KEY_ID=panbagnat-main
OIDC_AUTH_CODE_TTL_SECONDS=120
OIDC_ACCESS_TOKEN_TTL_SECONDS=900
OIDC_ID_TOKEN_TTL_SECONDS=900
OIDC_DEFAULT_SCOPES=openid profile email roles
```

If no private key is configured, Pan Bagnat generates an in-memory RSA key at startup.

## Module client setup

- Open a module in admin.
- Switch to the `OIDC` tab.
- Add one or more redirect URIs.
- Generate or rotate the client secret.
- Copy the discovery URL and client ID into the external module.

## Generic module configuration

```text
Issuer: https://panbagnat.example.com
Discovery URL: https://panbagnat.example.com/.well-known/openid-configuration
Authorization URL: https://panbagnat.example.com/oauth/authorize
Token URL: https://panbagnat.example.com/oauth/token
UserInfo URL: https://panbagnat.example.com/oauth/userinfo
JWKS URL: https://panbagnat.example.com/.well-known/jwks.json
Scopes: openid profile email roles
```

## Claims

- `sub`
- `email`
- `email_verified`
- `name`
- `preferred_username`
- `picture`
- `module`
- `roles`
- `role_slugs`

`email` is derived from the Pan Bagnat login:

- `login@student.42nice.fr` for non-staff users
- `login@42nice.fr` for staff users

## Limits

- Authorization Code flow only
- No refresh token in this version
- Redirect URIs must be explicit and exact
- Secret values are shown only once on generation or rotation
