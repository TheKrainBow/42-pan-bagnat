-- +migrate Down

DROP TABLE IF EXISTS oidc_access_tokens;
DROP TABLE IF EXISTS oidc_authorization_codes;
DROP TABLE IF EXISTS oidc_clients;

