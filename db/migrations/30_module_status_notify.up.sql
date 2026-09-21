-- Notify modules-proxy when a module's status changes, so the gateway cache
-- (which needs to know whether a module is enabled) gets refreshed.
CREATE OR REPLACE FUNCTION notify_module_status_changed() RETURNS trigger AS $$
BEGIN
    IF NEW.status IS DISTINCT FROM OLD.status THEN
        PERFORM pg_notify('module_page_changed', NEW.slug);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS module_status_changed_notify ON modules;
CREATE TRIGGER module_status_changed_notify
AFTER UPDATE ON modules
FOR EACH ROW EXECUTE FUNCTION notify_module_status_changed();
