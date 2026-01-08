BEGIN;

CREATE TABLE ssh_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    public_key TEXT NOT NULL DEFAULT '',
    private_key TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE modules ADD COLUMN ssh_key_id TEXT;

INSERT INTO ssh_keys (id, name, public_key, private_key)
SELECT
    'ssh-key_' || substring(id from 8),
    slug,
    ssh_public_key,
    ssh_private_key
FROM modules;

UPDATE modules
SET ssh_key_id = 'ssh-key_' || substring(id from 8);

ALTER TABLE modules
    ALTER COLUMN ssh_key_id SET NOT NULL;

ALTER TABLE modules
    ADD CONSTRAINT modules_ssh_key_id_fkey FOREIGN KEY (ssh_key_id)
        REFERENCES ssh_keys(id) ON DELETE RESTRICT;

CREATE INDEX idx_modules_ssh_key_id ON modules(ssh_key_id);

COMMIT;
