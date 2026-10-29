-- Sessions of the module "Claude" tab (claude CLI running in a per-module
-- container). Conversation transcripts live in the user's home volume; this
-- table only keeps metadata: owner, claude session id, title, status, dates.
CREATE TABLE claude_sessions (
  id                TEXT PRIMARY KEY, -- claude-session_ULID
  module_id         TEXT NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  claude_session_id TEXT,                 -- set by the SessionStart hook
  title             TEXT NOT NULL DEFAULT '',
  status            TEXT NOT NULL CHECK (status IN
                      ('starting','idle','working','waiting_input','stopped','aborted')),
  container_id      TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at        TIMESTAMPTZ NOT NULL DEFAULT now(), -- last time a container was started for it
  last_activity_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_attached_at  TIMESTAMPTZ,
  work_started_at   TIMESTAMPTZ,          -- start of the current working/waiting_input run
  ended_at          TIMESTAMPTZ,
  -- the owner has not seen the end of the last task yet (pastille in the UI)
  unseen_result     BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX claude_sessions_lookup
  ON claude_sessions (module_id, user_id, last_activity_at DESC);

-- At most one active container per module, even with concurrent requests.
CREATE UNIQUE INDEX claude_one_active_per_module
  ON claude_sessions (module_id)
  WHERE container_id IS NOT NULL
    AND status IN ('starting','idle','working','waiting_input');

CREATE UNIQUE INDEX claude_sessions_claude_id
  ON claude_sessions (module_id, user_id, claude_session_id)
  WHERE claude_session_id IS NOT NULL;
