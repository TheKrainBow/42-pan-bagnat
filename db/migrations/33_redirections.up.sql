CREATE TABLE redirections (
  id TEXT PRIMARY KEY, -- redirection_ULID
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  target_url TEXT NOT NULL,
  icon_url TEXT NOT NULL DEFAULT '',
  need_auth BOOLEAN NOT NULL DEFAULT true,
  is_visible BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE redirection_roles (
    redirection_id TEXT REFERENCES redirections(id) ON DELETE CASCADE,
    role_id TEXT REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (redirection_id, role_id)
);

CREATE TABLE redirection_forbidden_roles (
    redirection_id TEXT REFERENCES redirections(id) ON DELETE CASCADE,
    role_id TEXT REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (redirection_id, role_id)
);

-- Notify modules-proxy when redirections rows change, reusing the same
-- channel/payload shape as module_page so the existing LISTEN loop picks it
-- up without any changes.
CREATE OR REPLACE FUNCTION notify_redirection_changed() RETURNS trigger AS $$
DECLARE
    payload TEXT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        payload := COALESCE(OLD.slug, OLD.id);
    ELSE
        payload := COALESCE(NEW.slug, NEW.id);
    END IF;
    PERFORM pg_notify('module_page_changed', payload);
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS redirection_changed_notify ON redirections;
CREATE TRIGGER redirection_changed_notify
AFTER INSERT OR UPDATE OR DELETE ON redirections
FOR EACH ROW EXECUTE FUNCTION notify_redirection_changed();
