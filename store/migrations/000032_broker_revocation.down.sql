GRANT DELETE ON studio.broker_credentials TO studio_app;
GRANT DELETE ON studio.broker_leases TO studio_app;
ALTER TABLE studio.broker_leases DROP COLUMN credential_version;
ALTER TABLE studio.broker_credentials DROP COLUMN version;
