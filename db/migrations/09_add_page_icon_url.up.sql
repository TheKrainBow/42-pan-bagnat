-- +migrate Up
-- Add an optional icon_url to module_page so a page can override its module icon.
ALTER TABLE module_page
    ADD COLUMN IF NOT EXISTS icon_url TEXT;

