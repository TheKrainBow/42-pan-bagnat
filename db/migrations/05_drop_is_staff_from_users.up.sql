BEGIN;

-- Give the PB Admin role to anyone currently flagged as staff
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, 'roles_admin'
FROM users u
WHERE u.is_staff = TRUE
  AND EXISTS (SELECT 1 FROM roles r WHERE r.id = 'roles_admin')
ON CONFLICT DO NOTHING;

-- Now remove the local staff boolean; admin is role-based
ALTER TABLE users
  DROP COLUMN IF EXISTS is_staff;

COMMIT;