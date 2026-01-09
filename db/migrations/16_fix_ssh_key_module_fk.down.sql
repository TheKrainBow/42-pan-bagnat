BEGIN;

ALTER TABLE ssh_keys
    DROP CONSTRAINT IF EXISTS ssh_keys_created_by_module_id_fkey;

ALTER TABLE ssh_keys
    ADD CONSTRAINT ssh_keys_created_by_module_id_fkey
        FOREIGN KEY (created_by_module_id)
        REFERENCES modules(id);

COMMIT;
