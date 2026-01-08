BEGIN;

ALTER TABLE modules ADD COLUMN IF NOT EXISTS ssh_public_key TEXT NOT NULL DEFAULT '';
ALTER TABLE modules ADD COLUMN IF NOT EXISTS ssh_private_key TEXT NOT NULL DEFAULT '';

UPDATE modules AS m
SET ssh_public_key = COALESCE(k.public_key, ''),
    ssh_private_key = COALESCE(k.private_key, '')
FROM ssh_keys k
WHERE m.ssh_key_id = k.id;

COMMIT;
