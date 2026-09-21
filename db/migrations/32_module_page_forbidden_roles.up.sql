CREATE TABLE module_page_forbidden_roles (
    page_id TEXT REFERENCES module_page(id) ON DELETE CASCADE,
    role_id TEXT REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (page_id, role_id)
);

INSERT INTO module_page_forbidden_roles (page_id, role_id)
SELECT mp.id, 'roles_blacklist'
  FROM module_page mp
ON CONFLICT DO NOTHING;
