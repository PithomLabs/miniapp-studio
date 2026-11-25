// Package store owns Studio persistence: embedded migrations, session
// management through narrow SECURITY DEFINER functions, tenant-scoped
// transaction helpers, and owner-side provisioning.
//
// Privilege model: the migration owner applies DDL; the non-login
// studio_lookup role owns the session functions; the application role
// reaches tenant rows only (RLS) and sessions only through the three
// session functions. Nothing here ever sees a plaintext credential.
package store
