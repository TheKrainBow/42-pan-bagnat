DROP INDEX IF EXISTS idx_magic_link_tokens_expires;
DROP INDEX IF EXISTS idx_magic_link_tokens_user_created;
DROP TABLE IF EXISTS magic_link_tokens;
DROP INDEX IF EXISTS idx_users_email_lower;
ALTER TABLE users
  DROP COLUMN IF EXISTS email;
