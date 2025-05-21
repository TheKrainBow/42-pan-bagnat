INSERT INTO users (id, ft_login, ft_id, ft_is_staff, photo_url, last_seen, is_staff)
VALUES
  ('user_01HZXYZDE0420','heinz',    220393, true,  'https://intra.42.fr/heinz/220393','2001-04-16 12:00:00+00', true),
  ('user_01HZXYZDE0430','ltcherep', 194037, false, 'https://intra.42.fr/ltcherep/194037','2000-04-16 12:00:00+00', false),
  ('user_01HZXYZDE0440','tac',      79125,  true,  'https://intra.42.fr/tac/79125',    '2003-04-16 12:00:00+00', true),
  ('user_01HZXYZDE0450','yoshi',    78574,  true,  'https://intra.42.fr/yoshi/78574',  '2002-04-16 12:00:00+00', true),
  ('user_01HZXYZDE0460','alice',    300001, false, 'https://intra.42.fr/alice/300001','2025-01-10 09:00:00+00', false),
  ('user_01HZXYZDE0461','bob',      300002, false, 'https://intra.42.fr/bob/300002',  '2025-01-11 10:30:00+00', false);

-- BASE ROLES
INSERT INTO roles (id, name, color)
VALUES
  ('role_01HZXYZDE0420','Student',     '0x000000'),
  ('role_01HZXYZDE0430','ADM',         '0x00FF00'),
  ('role_01HZXYZDE0440','Pedago',      '0xFF0000'),
  ('role_01HZXYZDE0450','IT',          '0xFF00FF'),
  ('role_01HZXYZDE0460','Reviewer',    '0xAAAAAA'),
  ('role_01HZXYZDE0461','Editor',      '0xBBBBBB'),
  ('role_01HZXYZDE0462','Manager',     '0xCCCCCC');

-- MODULES: initial 4
INSERT INTO modules (id, name, version, status, url, icon_url, latest_version, late_commits, last_update)
VALUES
  ('module_01HZXYZDE0420','captain-hook',    '1.2','enabled', 'https://github.com/42nice/captain-hook', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',    '1.7',5,'2025-04-16 12:00:00+00'),
  ('module_01HZXYZDE0430','adm-stud',        '1.5','enabled', 'https://github.com/42nice/adm-stud', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',         '1.5',0,'2025-04-16 12:00:00+00'),
  ('module_01HZXYZDE0440','adm-manager',     '1.0','enabled', 'https://github.com/42nice/adm-manager', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',      '1.0',0,'2025-04-16 12:00:00+00'),
  ('module_01HZXYZDE0450','student-info',    '1.8','enabled', 'https://github.com/42nice/student-info', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',     '1.9',1,'2025-04-16 12:00:00+00'),
  ('module_01HZXYZDE0460','role-manager',    '1.0','enabled', 'https://github.com/42nice/role-manager', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',     '1.0',0,'2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0461','role-editor',     '1.1','enabled', 'https://github.com/42nice/role-editor', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',      '1.1',2,'2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0462','support-tool',    '2.0','enabled', 'https://github.com/42nice/support-tool', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',     '2.0',1,'2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0463','analytics',       '3.2','enabled', 'https://github.com/42nice/analytics', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',        '3.2',4,'2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0464','design-proto',    '0.9','enabled', 'https://github.com/42nice/design-proto', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',     '0.9',0,'2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0465','test-suite',      '5.4','enabled', 'https://github.com/42nice/test-suite', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',       '5.4',3,'2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0466','deploy-automate', '1.5','enabled', 'https://github.com/42nice/deploy-automate', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',  '1.5',0,'2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0467','strategy-dash',   '4.0','enabled', 'https://github.com/42nice/strategy-dash', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',    '4.0',2,'2025-04-20 12:00:00+00'),
  ('module_01HZXYZDE0468','zorbi-app',       '1.0','disabled','https://example.com/zorbi', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',                  '1.0',0,'2025-04-21 08:00:00+00'),
  ('module_01HZXYZDE0469','alpha-tool',      '2.2','disabled','https://example.com/alpha', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',                  '2.2',1,'2025-04-21 08:00:00+00'),
  ('module_01HZXYZDE0470','beta-service',    '3.5','enabled', 'https://example.com/beta', 'https://cdn.intra.42.fr/users/43445ac80da38e73e2af06b5897339fd/anissa.jpg',                   '3.5',1,'2025-04-21 08:00:00+00');

-- ASSIGN some module_roles (not used by GetAllModules but for completeness)
INSERT INTO module_roles (module_id, role_id)
VALUES
  ('module_01HZXYZDE0420','role_01HZXYZDE0450'),
  ('module_01HZXYZDE0430','role_01HZXYZDE0420'),
  ('module_01HZXYZDE0440','role_01HZXYZDE0430'),
  ('module_01HZXYZDE0450','role_01HZXYZDE0440');