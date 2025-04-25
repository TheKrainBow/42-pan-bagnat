CREATE TABLE roles (
  id TEXT PRIMARY KEY, -- role_ULID
  name TEXT NOT NULL,
  color TEXT NOT NULL
);

CREATE TABLE modules (
  id TEXT PRIMARY KEY, -- module_ULID
  name TEXT NOT NULL,
  version TEXT,
  status TEXT,
  url TEXT,
  latest_version TEXT,
  late_commits INT,
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


INSERT INTO users (id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff)
VALUES
  ('user_01HZXYZDE0420', 'heinz',    '220393', true,  'https://intra.42.fr/heinz/220393',    '2001-04-16 12:00:00', true),
  ('user_01HZXYZDE0430', 'ltcherep', '194037', false, 'https://intra.42.fr/ltcherep/194037', '2000-04-16 12:00:00', false),
  ('user_01HZXYZDE0440', 'tac',      '79125',  true,  'https://intra.42.fr/tac/79125',      '2003-04-16 12:00:00', true),
  ('user_01HZXYZDE0450', 'yoshi',    '78574',  true,  'https://intra.42.fr/yoshi/78574',    '2002-04-16 12:00:00', true);

INSERT INTO modules (id, name, version, status, url, latest_version, late_commits, last_update)
VALUES
  ('module_01HZXYZDE0420', 'captain-hook', '1.2', 'enabled', 'https://github.com/42nice/captain-hook', '1.7', 5, '2025-04-16 12:00:00'),
  ('module_01HZXYZDE0430', 'adm-stud', '1.5', 'enabled', 'https://github.com/42nice/adm-stud', '1.5', 0, '2025-04-16 12:00:00'),
  ('module_01HZXYZDE0440', 'adm-manager', '1.0', 'enabled', 'https://github.com/42nice/adm-manager', '1.0', 0, '2025-04-16 12:00:00'),
  ('module_01HZXYZDE0450', 'student-info', '1.8', 'enabled', 'https://github.com/42nice/student-info', '1.9', 1, '2025-04-16 12:00:00');


INSERT INTO roles (id, name, color)
VALUES
  ('role_01HZXYZDE0420', 'Student', '0x000000'),
  ('role_01HZXYZDE0430', 'ADM', '0x00FF00'),
  ('role_01HZXYZDE0440', 'Pedago', '0xFF0000'),
  ('role_01HZXYZDE0450', 'IT', '0xFF00FF');

INSERT INTO user_roles (user_id, role_id)
VALUES
  ('user_01HZXYZDE0420', 'role_01HZXYZDE0420'), -- heinz student
  ('user_01HZXYZDE0420', 'role_01HZXYZDE0430'), -- heinz ADM
  ('user_01HZXYZDE0420', 'role_01HZXYZDE0440'), -- heinz Pedago
  ('user_01HZXYZDE0420', 'role_01HZXYZDE0450'), -- heinz IT
  ('user_01HZXYZDE0430', 'role_01HZXYZDE0420'), -- ltcherep student
  ('user_01HZXYZDE0440', 'role_01HZXYZDE0420'), -- tac student
  ('user_01HZXYZDE0440', 'role_01HZXYZDE0440'), -- tac Pedago
  ('user_01HZXYZDE0440', 'role_01HZXYZDE0450'), -- tac IT
  ('user_01HZXYZDE0450', 'role_01HZXYZDE0420'), -- yoshi student
  ('user_01HZXYZDE0450', 'role_01HZXYZDE0430'), -- yoshi ADM
  ('user_01HZXYZDE0450', 'role_01HZXYZDE0440'); -- yoshi Pedago


INSERT INTO module_roles (module_id, role_id)
VALUES
  ('module_01HZXYZDE0420', 'role_01HZXYZDE0450'), -- captaine-hook IT
  ('module_01HZXYZDE0430', 'role_01HZXYZDE0420'), -- adm-stud Student
  ('module_01HZXYZDE0440', 'role_01HZXYZDE0430'), -- adm-manager ADM
  ('module_01HZXYZDE0450', 'role_01HZXYZDE0440'); -- student-info Pedago