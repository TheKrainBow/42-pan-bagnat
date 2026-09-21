BEGIN;

-- MODULE ACTIVITY --
-- One row per (module, user) per minute of usage, written by the module
-- proxy on every authenticated module request (iframe or direct access).
-- Stats are derived by sessionizing these rows: a gap of 60+ minutes
-- between two consecutive rows for the same module/user starts a new
-- activity session.

CREATE TABLE module_activity (
  id          BIGSERIAL     PRIMARY KEY,
  module_id   TEXT          NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  user_id     TEXT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at  TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_module_activity_module_user_time ON module_activity (module_id, user_id, created_at DESC);
CREATE INDEX idx_module_activity_time ON module_activity (created_at DESC);

COMMIT;
