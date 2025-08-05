-- ROLES --

CREATE TABLE roles (
  id TEXT PRIMARY KEY, -- role_ULID
  name TEXT NOT NULL,
  color TEXT NOT NULL,
  is_default BOOLEAN DEFAULT false
);

-- MODULES --

CREATE TABLE modules (
  id TEXT PRIMARY KEY, -- module_ULID
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  git_url TEXT,
  git_branch TEXT NOT NULL DEFAULT 'main',
  ssh_public_key TEXT NOT NULL DEFAULT '',
  ssh_private_key TEXT NOT NULL DEFAULT '',
  version TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  icon_url TEXT NOT NULL DEFAULT '',
  latest_version TEXT NOT NULL DEFAULT '',
  late_commits INT NOT NULL DEFAULT 0,
  last_update TIMESTAMP WITH TIME ZONE NOT NULL
);

-- USERS --

CREATE TABLE users (
  id TEXT PRIMARY KEY, -- user_ULID
  ft_login TEXT NOT NULL UNIQUE,
  ft_id BIGINT NOT NULL,
  ft_is_staff BOOLEAN NOT NULL,
  photo_url TEXT NOT NULL,
  last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
  is_staff BOOLEAN NOT NULL
);

CREATE TABLE sessions (
  session_id TEXT PRIMARY KEY,
  ft_login   TEXT REFERENCES users(ft_login),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMP NOT NULL
);

-- LOGS --

CREATE TABLE module_log (
  id          BIGSERIAL     PRIMARY KEY,
  module_id   TEXT          NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
  level       TEXT          NOT NULL,
  message     TEXT          NOT NULL,
  meta        JSONB
);
CREATE INDEX idx_module_log_module_time ON module_log (module_id, created_at DESC);

-- MODULE PAGES --

CREATE TABLE module_page (
  id TEXT PRIMARY KEY, -- user_ULID
  name TEXT,
  slug TEXT,
  url TEXT,
  is_public BOOLEAN NOT NULL,
  module_id TEXT NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
  UNIQUE (module_id, name)
);

-- JOIN TABLES --

CREATE TABLE user_roles (
  user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
  role_id TEXT REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE module_roles (
  module_id TEXT REFERENCES modules(id) ON DELETE CASCADE,
  role_id   TEXT REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (module_id, role_id)
);
