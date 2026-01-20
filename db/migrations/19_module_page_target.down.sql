ALTER TABLE module_page
    ADD COLUMN url TEXT;

UPDATE module_page
   SET url = 'http://' ||
             COALESCE(NULLIF(target_container, ''), slug, 'pending') ||
             ':' ||
             COALESCE(target_port, 80)::TEXT;

ALTER TABLE module_page
    ALTER COLUMN url SET NOT NULL;

ALTER TABLE module_page
    DROP COLUMN target_container,
    DROP COLUMN target_port;
