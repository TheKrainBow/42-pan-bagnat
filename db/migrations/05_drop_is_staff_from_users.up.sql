BEGIN;

-- Remove the local staff boolean; admin is now role-based.
ALTER TABLE users
  DROP COLUMN IF EXISTS is_staff;

COMMIT;