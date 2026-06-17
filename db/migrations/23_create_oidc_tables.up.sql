-- +migrate Up

CREATE TABLE oidc_clients (
  id TEXT PRIMARY KEY,
  module_id TEXT NOT NULL UNIQUE REFERENCES modules(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  client_id TEXT NOT NULL UNIQUE,
  client_secret_hash TEXT,
  client_type TEXT NOT NULL,
  allowed_redirect_uris TEXT[] NOT NULL DEFAULT '{}',
  allowed_scopes TEXT[] NOT NULL DEFAULT '{openid,profile,email,roles}',
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_secret_rotated_at TIMESTAMPTZ
);

CREATE TABLE oidc_authorization_codes (
  id TEXT PRIMARY KEY,
  code_hash TEXT NOT NULL UNIQUE,
  client_id TEXT NOT NULL REFERENCES oidc_clients(client_id) ON DELETE CASCADE,
  module_id TEXT NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  redirect_uri TEXT NOT NULL,
  scopes TEXT[] NOT NULL DEFAULT '{}',
  nonce TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE oidc_access_tokens (
  id TEXT PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE,
  client_id TEXT NOT NULL REFERENCES oidc_clients(client_id) ON DELETE CASCADE,
  module_id TEXT NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  scopes TEXT[] NOT NULL DEFAULT '{}',
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

