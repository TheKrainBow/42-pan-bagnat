ALTER TABLE module_page
    ADD COLUMN max_upload_body_size TEXT NOT NULL DEFAULT '1m';
