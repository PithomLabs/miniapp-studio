-- 000006: sequence grant completing F2. audit_events.id is the only
-- sequence-backed column in the framework schema (BIGSERIAL); table
-- privileges alone cannot advance it, so audit writes failed with 42501 even
-- after schema USAGE. Least privilege: exactly this sequence, nothing else.

GRANT USAGE, SELECT ON SEQUENCE app.audit_events_id_seq TO studio_app;
