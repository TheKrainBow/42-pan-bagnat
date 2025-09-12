-- Drop deployment tracking fields from modules
ALTER TABLE modules
  DROP COLUMN IF EXISTS is_deploying,
  DROP COLUMN IF EXISTS last_deploy,
  DROP COLUMN IF EXISTS last_deploy_status;

