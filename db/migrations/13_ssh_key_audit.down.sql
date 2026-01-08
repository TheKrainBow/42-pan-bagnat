BEGIN;

DROP TABLE IF EXISTS ssh_key_events;
ALTER TABLE ssh_keys
    DROP COLUMN IF EXISTS created_by_user_id,
    DROP COLUMN IF EXISTS created_by_module_id;

COMMIT;
