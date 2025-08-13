BEGIN;

-- Add per-device/session metadata
ALTER TABLE sessions
  ADD COLUMN IF NOT EXISTS user_agent   TEXT      NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS ip           TEXT      NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS device_label TEXT      NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS last_seen    TIMESTAMP NOT NULL DEFAULT NOW();

-- Backfill last_seen for existing rows
UPDATE sessions SET last_seen = created_at WHERE last_seen IS NULL;

-- Helpful indexes for lookups and housekeeping
CREATE INDEX IF NOT EXISTS idx_sessions_login_expires   ON sessions (ft_login, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_login_useragent ON sessions (ft_login, user_agent);
CREATE INDEX IF NOT EXISTS idx_sessions_last_seen       ON sessions (last_seen DESC);

COMMIT;
