ALTER TABLE module_page
    DROP COLUMN proxy_timeout_seconds,
    DROP COLUMN rate_limit_rps,
    DROP COLUMN rate_limit_burst,
    DROP COLUMN disable_request_buffering;
