-- A single global dashboard: a staff-editable canvas (drawings, text boxes,
-- GIFs) shown to users instead of auto-opening their first module on login.
CREATE TABLE dashboard (
  id TEXT PRIMARY KEY,
  canvas_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by TEXT REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO dashboard (id, canvas_json) VALUES ('default', '{}'::jsonb);
