CREATE TABLE module_page_roles (
    page_id TEXT REFERENCES module_page(id) ON DELETE CASCADE,
    role_id TEXT REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (page_id, role_id)
);

INSERT INTO module_page_roles (page_id, role_id)
SELECT mp.id, mr.role_id
  FROM module_page mp
  JOIN module_roles mr ON mr.module_id = mp.module_id
ON CONFLICT DO NOTHING;
