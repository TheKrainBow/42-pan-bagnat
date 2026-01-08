BEGIN;

ALTER TABLE modules DROP CONSTRAINT IF EXISTS modules_ssh_key_id_fkey;
DROP INDEX IF EXISTS idx_modules_ssh_key_id;
ALTER TABLE modules DROP COLUMN IF EXISTS ssh_key_id;
DROP TABLE IF EXISTS ssh_keys;

COMMIT;
