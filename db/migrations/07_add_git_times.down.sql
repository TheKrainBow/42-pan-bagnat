ALTER TABLE modules
  DROP COLUMN IF EXISTS git_last_fetch,
  DROP COLUMN IF EXISTS git_last_pull;

