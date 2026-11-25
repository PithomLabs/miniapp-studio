-- 000010: let the application role read the Studio schema version (F12).
-- The server's startup gate and readyz check run on the app pool, so
-- studio_app needs SELECT on the version table the migrate library owns.

GRANT SELECT ON studio_schema_migrations TO studio_app;
