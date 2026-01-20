ALTER TABLE module_page
    ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT true;

UPDATE module_page
   SET is_public = NOT need_auth;

ALTER TABLE module_page
    DROP COLUMN iframe_only,
    DROP COLUMN need_auth;
