-- Migration 35 defaulted photo_updated_at to now() for existing rows, which
-- wrongly marked their already-stored (pre-bearer-requirement) avatar URLs
-- as fresh, so the TTL refresh never kicked in to fetch a working link.
-- Force every user's avatar to be re-fetched from 42 on next access.
UPDATE users SET photo_updated_at = 'epoch';
