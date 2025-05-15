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

-- 1) Add some new users to test member counts
INSERT INTO users (id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff) VALUES
  ('user_01HZXYZDE0460', 'alice',  '300001', false, 'https://intra.42.fr/alice/300001',  '2025-01-10 09:00:00+00', false),
  ('user_01HZXYZDE0461', 'bob',    '300002', false, 'https://intra.42.fr/bob/300002',    '2025-01-11 10:30:00+00', false),
  ('user_01HZXYZDE0462', 'carol',  '300003', false, 'https://intra.42.fr/carol/300003',  '2025-01-12 11:45:00+00', false),
  ('user_01HZXYZDE0463', 'dave',   '300004', false, 'https://intra.42.fr/dave/300004',   '2025-01-13 14:20:00+00', false),
  ('user_01HZXYZDE0464', 'eve',    '300005', true,  'https://intra.42.fr/eve/300005',    '2025-01-14 15:00:00+00', true),
  ('user_01HZXYZDE0465', 'frank',  '300006', false, 'https://intra.42.fr/frank/300006',  '2025-01-15 16:30:00+00', false);

-- 2) Add a bunch of roles (we now have 24 total, enough to test pagination)
INSERT INTO roles (id, name, color) VALUES
  ('role_01HZXYZDE0460', 'Reviewer',    '0xAAAAAA'),
  ('role_01HZXYZDE0461', 'Editor',      '0xBBBBBB'),
  ('role_01HZXYZDE0462', 'Manager',     '0xCCCCCC'),
  ('role_01HZXYZDE0463', 'Support',     '0xDDDDDD'),
  ('role_01HZXYZDE0464', 'Operator',    '0xEEEEEE'),
  ('role_01HZXYZDE0465', 'Guest',       '0x111111'),
  ('role_01HZXYZDE0466', 'Developer',   '0x222222'),
  ('role_01HZXYZDE0467', 'Analyst',     '0x333333'),
  ('role_01HZXYZDE0468', 'Designer',    '0x444444'),
  ('role_01HZXYZDE0469', 'Tester',      '0x555555'),
  ('role_01HZXYZDE0470', 'Maintainer',  '0x666666'),
  ('role_01HZXYZDE0471', 'Contributor', '0x777777'),
  ('role_01HZXYZDE0472', 'Owner',       '0x888888'),
  ('role_01HZXYZDE0473', 'SuperAdmin',  '0x999999'),
  ('role_01HZXYZDE0474', 'Moderator',   '0x123456'),
  ('role_01HZXYZDE0475', 'Auditor',     '0x654321'),
  ('role_01HZXYZDE0476', 'Architect',   '0xABCDEF'),
  ('role_01HZXYZDE0477', 'Coordinator', '0xFEDCBA'),
  ('role_01HZXYZDE0478', 'Planner',     '0x0F0F0F'),
  ('role_01HZXYZDE0479', 'Strategist',  '0xF0F0F0');

-- 3) Add some new modules (applications) for testing
INSERT INTO modules (id, name, version, status, url, latest_version, late_commits, last_update) VALUES
  ('module_01HZXYZDE0460', 'role-manager',    '1.0', 'enabled', 'https://github.com/42nice/role-manager',    '1.0', 0, '2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0461', 'role-editor',     '1.1', 'enabled', 'https://github.com/42nice/role-editor',     '1.1', 2, '2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0462', 'support-tool',    '2.0', 'enabled', 'https://github.com/42nice/support-tool',    '2.0', 1, '2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0463', 'analytics',       '3.2', 'enabled', 'https://github.com/42nice/analytics',       '3.2', 4, '2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0464', 'design-proto',    '0.9', 'enabled', 'https://github.com/42nice/design-proto',    '0.9', 0, '2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0465', 'test-suite',      '5.4', 'enabled', 'https://github.com/42nice/test-suite',      '5.4', 3, '2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0466', 'deploy-automate', '1.5', 'enabled', 'https://github.com/42nice/deploy-automate', '1.5', 0, '2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0467', 'strategy-dash',   '4.0', 'enabled', 'https://github.com/42nice/strategy-dash',   '4.0', 2, '2025-04-20 12:00:00+00');

-- 4) Assign users to roles (varied member counts)
INSERT INTO user_roles (user_id, role_id) VALUES
  -- new users
  ('user_01HZXYZDE0460', 'role_01HZXYZDE0460'),
  ('user_01HZXYZDE0460', 'role_01HZXYZDE0461'),
  ('user_01HZXYZDE0461', 'role_01HZXYZDE0462'),
  ('user_01HZXYZDE0461', 'role_01HZXYZDE0463'),
  ('user_01HZXYZDE0462', 'role_01HZXYZDE0464'),
  ('user_01HZXYZDE0463', 'role_01HZXYZDE0465'),
  ('user_01HZXYZDE0463', 'role_01HZXYZDE0466'),
  ('user_01HZXYZDE0463', 'role_01HZXYZDE0467'),
  ('user_01HZXYZDE0464', 'role_01HZXYZDE0468'),
  ('user_01HZXYZDE0464', 'role_01HZXYZDE0469'),
  ('user_01HZXYZDE0464', 'role_01HZXYZDE0470'),
  ('user_01HZXYZDE0465', 'role_01HZXYZDE0471'),
  ('user_01HZXYZDE0465', 'role_01HZXYZDE0472'),
  ('user_01HZXYZDE0465', 'role_01HZXYZDE0473'),
  ('user_01HZXYZDE0465', 'role_01HZXYZDE0474'),
  ('user_01HZXYZDE0465', 'role_01HZXYZDE0475');

-- 5) Assign modules to roles (varied application counts)
INSERT INTO module_roles (module_id, role_id) VALUES
  ('module_01HZXYZDE0460', 'role_01HZXYZDE0460'),
  ('module_01HZXYZDE0461', 'role_01HZXYZDE0461'),
  ('module_01HZXYZDE0462', 'role_01HZXYZDE0462'),
  ('module_01HZXYZDE0463', 'role_01HZXYZDE0463'),
  ('module_01HZXYZDE0464', 'role_01HZXYZDE0464'),
  ('module_01HZXYZDE0465', 'role_01HZXYZDE0465'),
  ('module_01HZXYZDE0466', 'role_01HZXYZDE0466'),
  ('module_01HZXYZDE0467', 'role_01HZXYZDE0467');
