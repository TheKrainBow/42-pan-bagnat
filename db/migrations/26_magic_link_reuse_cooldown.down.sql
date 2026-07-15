DROP INDEX IF EXISTS idx_magic_link_tokens_user_sent;
DROP INDEX IF EXISTS idx_magic_link_tokens_active_user;

ALTER TABLE magic_link_tokens
  DROP COLUMN IF EXISTS last_sent_at,
  DROP COLUMN IF EXISTS token_value;
