CREATE OR REPLACE FUNCTION notify_mailhole_email()
RETURNS trigger AS $$
DECLARE
    channel text;
    payload text;
BEGIN
    channel := 'mailhole_recipient_' ||
        replace(replace(NEW.recipient, '@', '_at_'), '.', '_dot_');
    payload := row_to_json(NEW)::text;
    EXECUTE format('NOTIFY %I, %L', channel, payload);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notify_mailhole_email_trigger
AFTER INSERT ON emails
FOR EACH ROW
EXECUTE FUNCTION notify_mailhole_email();
