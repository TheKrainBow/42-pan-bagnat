ALTER TABLE module_page
    ADD COLUMN target_container TEXT,
    ADD COLUMN target_port INTEGER;

ALTER TABLE module_page
    DROP COLUMN url;
