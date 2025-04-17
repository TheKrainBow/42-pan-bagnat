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