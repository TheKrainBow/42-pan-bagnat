CREATE TABLE roles (
  id TEXT PRIMARY KEY, -- role_ULID
  name TEXT NOT NULL,
  color TEXT NOT NULL
);

CREATE TABLE modules (
  id TEXT PRIMARY KEY, -- module_ULID
  name TEXT NOT NULL,
  url TEXT,
  ssh_public_key TEXT NOT NULL DEFAULT '',
  ssh_private_key TEXT NOT NULL DEFAULT '',
  version TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  icon_url TEXT NOT NULL DEFAULT '',
  latest_version TEXT NOT NULL DEFAULT '',
  late_commits INT NOT NULL DEFAULT 0,
  last_update TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE users (
  id TEXT PRIMARY KEY, -- user_ULID
  ft_login TEXT NOT NULL,
  ft_id INTEGER NOT NULL,
  ft_is_staff BOOLEAN NOT NULL,
  photo_url TEXT NOT NULL,
  last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
  is_staff BOOLEAN NOT NULL
);

-- join tables
CREATE TABLE user_roles (
  user_id TEXT REFERENCES users(id),
  role_id TEXT REFERENCES roles(id),
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE module_roles (
  module_id TEXT REFERENCES modules(id),
  role_id TEXT REFERENCES roles(id),
  PRIMARY KEY (module_id, role_id)
);
