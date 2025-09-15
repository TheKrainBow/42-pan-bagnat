ALTER TABLE modules
  DROP COLUMN IF EXISTS current_commit_hash,
  DROP COLUMN IF EXISTS current_commit_subject,
  DROP COLUMN IF EXISTS latest_commit_hash,
  DROP COLUMN IF EXISTS latest_commit_subject;

