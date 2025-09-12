-- Add deployment tracking fields to modules
ALTER TABLE modules
  ADD COLUMN IF NOT EXISTS is_deploying BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS last_deploy TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS last_deploy_status TEXT NOT NULL DEFAULT '';

