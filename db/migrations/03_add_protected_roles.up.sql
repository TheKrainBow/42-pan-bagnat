BEGIN;

-- 1) Block-deletion flag on roles
ALTER TABLE roles
  ADD COLUMN IF NOT EXISTS is_protected BOOLEAN NOT NULL DEFAULT FALSE;

-- 2) Seed built-in roles

-- Blacklist: protected, non-default, no rules
INSERT INTO roles (id, name, color, is_default, is_protected, rules_json, rules_updated_at)
VALUES
  ('roles_blacklist', 'Blacklisted', '#0d1320', FALSE, TRUE,
    '{
		"kind": "group",
		"logic": "AND",
		"rules": [
		{
			"kind": "array",
			"path": "campus",
			"quantifier": "NONE",
			"predicate": {
			"kind": "group",
			"logic": "AND",
			"rules": [
				{
				"kind": "scalar",
				"path": "id",
				"valueType": "number",
				"op": "eq",
				"value": 41
				}
			]
			}
		}
		]
	}'::jsonb,
    NOW()
  )
ON CONFLICT (id) DO NOTHING;

-- Pan Bagnat Admin: protected, rule "kind == admin"
INSERT INTO roles (id, name, color, is_default, is_protected, rules_json, rules_updated_at)
VALUES
  ('roles_admin', 'PB Admin', '#ed210e', FALSE, TRUE,
    '{
        "kind": "group",
        "logic": "AND",
        "rules": [
        {
            "kind": "scalar",
            "op": "eq",
            "path": "kind",
            "value": "admin",
            "valueType": "string"
        }
        ]
    }'::jsonb,
    NOW()
  )
ON CONFLICT (id) DO NOTHING;

-- Student: default, deletable
INSERT INTO roles (id, name, color, is_default, is_protected, rules_json, rules_updated_at)
VALUES
  ('roles_default', 'Student', '#edba55', TRUE, FALSE, NULL, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, 'roles_default'
FROM users u
LEFT JOIN user_roles ur
  ON ur.user_id = u.id AND ur.role_id = 'roles_default'
WHERE ur.user_id IS NULL
ON CONFLICT DO NOTHING;

COMMIT;
