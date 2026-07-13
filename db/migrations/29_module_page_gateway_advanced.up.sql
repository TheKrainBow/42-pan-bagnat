ALTER TABLE module_page
    ADD COLUMN proxy_timeout_seconds INTEGER NOT NULL DEFAULT 60,
    ADD COLUMN rate_limit_rps INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN rate_limit_burst INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN disable_request_buffering BOOLEAN NOT NULL DEFAULT false;
