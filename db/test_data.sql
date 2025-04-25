INSERT INTO users (id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff)
VALUES
  ('user_01HZXYZDE0420', 'heinz',    '220393', true,  'https://cdn.intra.42.fr/users/e02ff39566a64af026f137b30c18c3e4/heinz.png',    '2025-04-24 12:00:00', true),
  ('user_01HZXYZDE0430', 'ltcherep', '194037', false, 'https://cdn.intra.42.fr/users/2d49a08ed473a0705e7f7cfd5cd3ae72/ltcherep.jpg', '2002-04-16 12:00:00', false),
  ('user_01HZXYZDE0440', 'tac',      '79125',  true,  'https://cdn.intra.42.fr/users/41714d1b596f06cc352f21791ee64096/tac.png',      '2003-04-16 12:00:00', true),
  ('user_01HZXYZDE0450', 'rose',     '188547', true,  'https://cdn.intra.42.fr/users/8d185c6e0587ce836f0185e05daa5b13/rose.jpg',     '2004-04-16 12:00:00', false),
  ('user_01HZXYZDE0460', 'yoshi',    '78574',  true,  'https://cdn.intra.42.fr/users/dc17821ef0dc39f998f340e9db0fe604/yoshi.jpg',    '2001-04-16 12:00:00', true);

INSERT INTO modules (id, name, version, status, url, latest_version, late_commits, last_update)
VALUES
  ('module_01HZXYZDE0420', 'captain-hook', '1.2', 'enabled', 'https://github.com/42nice/captain-hook', '1.7', 5, '2025-04-16 12:00:00'),
  ('module_01HZXYZDE0430', 'adm-stud', '1.5', 'enabled', 'https://github.com/42nice/adm-stud', '1.5', 0, '2025-04-16 12:00:00'),
  ('module_01HZXYZDE0440', 'adm-manager', '1.0', 'enabled', 'https://github.com/42nice/adm-manager', '1.0', 0, '2025-04-16 12:00:00'),
  ('module_01HZXYZDE0450', 'student-info', '1.8', 'enabled', 'https://github.com/42nice/student-info', '1.9', 1, '2025-04-16 12:00:00');


INSERT INTO roles (id, name, color)
VALUES
  ('role_01HZXYZDE0420', 'Student', '0xFFCD70'),
  ('role_01HZXYZDE0430', 'ADM', '0x69DE7A'),
  ('role_01HZXYZDE0440', 'Pedago', '0x86B3E7'),
  ('role_01HZXYZDE0450', 'Cluster', '0x69D2DE'),
  ('role_01HZXYZDE0460', 'Stagiaire', '0x75FFF8'),
  ('role_01HZXYZDE0470', 'IT', '0xF26363');

INSERT INTO user_roles (user_id, role_id)
VALUES
  ('user_01HZXYZDE0420', 'role_01HZXYZDE0460'), -- heinz Stagiaire
  ('user_01HZXYZDE0430', 'role_01HZXYZDE0420'), -- ltcherep student
  ('user_01HZXYZDE0440', 'role_01HZXYZDE0440'), -- tac Pedago
  ('user_01HZXYZDE0440', 'role_01HZXYZDE0430'), -- tac ADM
  ('user_01HZXYZDE0440', 'role_01HZXYZDE0450'), -- tac Cluster
  ('user_01HZXYZDE0440', 'role_01HZXYZDE0470'), -- tac IT
  ('user_01HZXYZDE0460', 'role_01HZXYZDE0440'), -- Yoshi Pedago
  ('user_01HZXYZDE0460', 'role_01HZXYZDE0430'), -- Yoshi ADM
  ('user_01HZXYZDE0460', 'role_01HZXYZDE0450'), -- Yoshi Cluster
  ('user_01HZXYZDE0460', 'role_01HZXYZDE0470'), -- Yoshi IT
  ('user_01HZXYZDE0450', 'role_01HZXYZDE0430'), -- Rose ADM
  ('user_01HZXYZDE0450', 'role_01HZXYZDE0450'); -- Rose Cluster


INSERT INTO module_roles (module_id, role_id)
VALUES
  ('module_01HZXYZDE0420', 'role_01HZXYZDE0450'), -- captaine-hook IT
  ('module_01HZXYZDE0430', 'role_01HZXYZDE0420'), -- adm-stud Student
  ('module_01HZXYZDE0440', 'role_01HZXYZDE0430'), -- adm-manager ADM
  ('module_01HZXYZDE0450', 'role_01HZXYZDE0440'); -- student-info Pedago