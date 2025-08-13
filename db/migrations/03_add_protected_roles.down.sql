-- 03_add_protected_roles.down.sql

BEGIN;

-- Remove the seeded roles (user_roles rows will be removed via ON DELETE CASCADE)
DELETE FROM roles
WHERE id IN ('roles_blacklist', 'roles_admin', 'roles_student');

-- Drop the protection flag added by the up migration
ALTER TABLE roles
  DROP COLUMN IF EXISTS is_protected;

COMMIT;
