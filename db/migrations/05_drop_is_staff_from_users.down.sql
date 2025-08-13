BEGIN;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS is_staff BOOLEAN NOT NULL DEFAULT FALSE;

-- If roles still exist, mark users who have the admin role as staff again.
UPDATE users u
SET is_staff = TRUE
WHERE EXISTS (
  SELECT 1
  FROM user_roles ur
  WHERE ur.user_id = u.id
    AND ur.role_id = 'roles_admin'
);

COMMIT;
