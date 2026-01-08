BEGIN;

ALTER TABLE ssh_key_events
    DROP CONSTRAINT IF EXISTS ssh_key_events_actor_module_id_fkey;

ALTER TABLE ssh_key_events
    ADD CONSTRAINT ssh_key_events_actor_module_id_fkey
        FOREIGN KEY (actor_module_id)
        REFERENCES modules(id)
        ON DELETE SET NULL;

COMMIT;
