BEGIN;

ALTER TABLE modules DROP COLUMN IF EXISTS ssh_public_key;
ALTER TABLE modules DROP COLUMN IF EXISTS ssh_private_key;

COMMIT;
