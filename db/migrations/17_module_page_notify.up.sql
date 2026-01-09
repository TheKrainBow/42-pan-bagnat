-- Notify modules-proxy when module_page rows change
CREATE OR REPLACE FUNCTION notify_module_page_changed() RETURNS trigger AS $$
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

DROP TRIGGER IF EXISTS module_page_changed_notify ON module_page;
CREATE TRIGGER module_page_changed_notify
AFTER INSERT OR UPDATE OR DELETE ON module_page
FOR EACH ROW EXECUTE FUNCTION notify_module_page_changed();
