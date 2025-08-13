BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_sessions_last_seen;
DROP INDEX IF EXISTS idx_sessions_login_useragent;
DROP INDEX IF EXISTS idx_sessions_login_expires;

-- Remove added columns
ALTER TABLE sessions
  DROP COLUMN IF EXISTS last_seen,
  DROP COLUMN IF EXISTS device_label,
  DROP COLUMN IF EXISTS ip,
  DROP COLUMN IF EXISTS user_agent;

COMMIT;
