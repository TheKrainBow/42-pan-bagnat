-- +migrate Up
ALTER TABLE roles
ADD COLUMN rules_json JSONB,
ADD COLUMN rules_updated_at TIMESTAMPTZ;
