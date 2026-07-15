ALTER TABLE users
  ADD COLUMN IF NOT EXISTS email TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower
  ON users (LOWER(email))
  WHERE email IS NOT NULL AND email <> '';

CREATE TABLE IF NOT EXISTS magic_link_tokens (
  token_hash TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  next_url TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_magic_link_tokens_user_created
  ON magic_link_tokens (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_magic_link_tokens_expires
  ON magic_link_tokens (expires_at);
