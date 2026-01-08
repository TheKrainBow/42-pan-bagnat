BEGIN;

ALTER TABLE ssh_keys
    ADD COLUMN created_by_user_id TEXT REFERENCES users(id),
    ADD COLUMN created_by_module_id TEXT REFERENCES modules(id);

CREATE TABLE ssh_key_events (
    id BIGSERIAL PRIMARY KEY,
    ssh_key_id TEXT NOT NULL REFERENCES ssh_keys(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    actor_user_id TEXT REFERENCES users(id),
    actor_module_id TEXT REFERENCES modules(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

UPDATE ssh_keys k
   SET created_by_module_id = m.id
  FROM modules m
 WHERE k.id = ('ssh-key_' || substring(m.id from 8))
   AND k.created_by_user_id IS NULL
   AND k.created_by_module_id IS NULL;

COMMIT;
