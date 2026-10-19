-- Track when a user's avatar URL was last fetched from 42, so we know when
-- to refresh it (avatars can change on intra without PB ever finding out).
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS photo_updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
