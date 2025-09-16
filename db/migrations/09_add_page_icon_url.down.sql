-- +migrate Down
ALTER TABLE module_page
    DROP COLUMN IF EXISTS icon_url;

