ALTER TABLE module_page
    ADD COLUMN iframe_only BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN need_auth BOOLEAN NOT NULL DEFAULT false;

UPDATE module_page
   SET iframe_only = NOT is_public,
       need_auth = NOT is_public;

ALTER TABLE module_page
    DROP COLUMN is_public;
