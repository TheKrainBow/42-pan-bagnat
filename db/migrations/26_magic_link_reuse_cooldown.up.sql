ALTER TABLE magic_link_tokens
  ADD COLUMN IF NOT EXISTS token_value TEXT,
  ADD COLUMN IF NOT EXISTS last_sent_at TIMESTAMPTZ;

UPDATE magic_link_tokens
   SET last_sent_at = created_at
 WHERE last_sent_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_magic_link_tokens_active_user
  ON magic_link_tokens (user_id, expires_at DESC)
  WHERE consumed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_magic_link_tokens_user_sent
  ON magic_link_tokens (user_id, last_sent_at DESC)
  WHERE last_sent_at IS NOT NULL;
