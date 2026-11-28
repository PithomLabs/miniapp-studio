package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var rxHexPassword = regexp.MustCompile(`^[0-9a-f]{32,}$`)

// rxRoleName admits the fixed role names this file interpolates into DDL.
var rxRoleName = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func isDuplicateObject(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42710"
}

// EnsureAppRole creates the application role when missing and is a safe
// no-op (except the idempotent CONNECT grant) when it exists. It NEVER
// changes an existing password: a concurrent test boot or a second
// provision run cannot invalidate live credentials (U5.12 invariant 1;
// the Boundary B a1 run died to prove the old behavior). Only
// RotateAppRole changes a password, and only the provision CLI calls it.
//
// The password argument is consumed solely at creation. Callers that only
// need existence (test harnesses satisfying the migrate gate) pass any
// hex value; on clusters where the role already exists the value is
// ignored. Concurrent creators racing CREATE ROLE converge: the
// duplicate_object loser treats the role as created by the winner.
//
// The role can log in but can never bypass RLS. Passwords must be
// lowercase hex (generate with `openssl rand -hex 32`) so role DDL stays
// injectable-free.
func EnsureAppRole(ctx context.Context, owner *pgxpool.Pool, password string) error {
	if !rxHexPassword.MatchString(password) {
		return fmt.Errorf("store: application role password must be lowercase hex, 32+ chars")
	}
	created, err := createRoleOnce(ctx, owner, AppRoleName, password)
	if err != nil {
		return err
	}
	_ = created
	return ensureConnect(ctx, owner, AppRoleName)
}

// RotateAppRole sets the application role password. This is the ONLY
// ordinary code path that mutates a persistent credential (U5.12
// invariant 2): an explicit administrative provisioning operation,
// reachable solely through `studio provision --app-role-password-file`
// on an already-provisioned role and the Boundary B run provisioner.
// It refuses a missing role rather than creating one, so a typo'd
// cluster never gains a half-provisioned role. Test harnesses must
// never call it; TestNoHarnessRotation scans for that.
func RotateAppRole(ctx context.Context, owner *pgxpool.Pool, password string) error {
	if !rxHexPassword.MatchString(password) {
		return fmt.Errorf("store: application role password must be lowercase hex, 32+ chars")
	}
	return rotateRolePassword(ctx, owner, AppRoleName, password)
}

// EnsureAppRoleConnect grants the application role CONNECT on the
// current database. Idempotent; used when an existing role meets a new
// database (provision without a password file, run provisioning).
func EnsureAppRoleConnect(ctx context.Context, owner *pgxpool.Pool) error {
	return ensureConnect(ctx, owner, AppRoleName)
}

// createRoleOnce creates a LOGIN NOBYPASSRLS role with the given
// password, or converges when a concurrent creator wins the race.
// It reports whether this call performed the creation.
func createRoleOnce(ctx context.Context, owner *pgxpool.Pool, role, password string) (bool, error) {
	if !rxRoleName.MatchString(role) {
		return false, fmt.Errorf("store: refusing unsafe role name")
	}
	tx, err := owner.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("store: provision begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, role,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("store: provision role check: %w", err)
	}
	if exists {
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("store: provision commit: %w", err)
		}
		return false, nil
	}
	// Hex-validated passwords and regexp-validated role names are safe
	// to interpolate: DDL takes no bind parameters.
	if _, err := tx.Exec(ctx, `CREATE ROLE `+role+` WITH LOGIN NOBYPASSRLS PASSWORD '`+password+`'`); err != nil {
		if isDuplicateObject(err) {
			// A concurrent creator won; their password stands and
			// ours is discarded. Convergence, not conflict.
			_ = tx.Rollback(ctx)
			return false, nil
		}
		return false, fmt.Errorf("store: provision create role: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		if isDuplicateObject(err) {
			return false, nil
		}
		return false, fmt.Errorf("store: provision commit: %w", err)
	}
	return true, nil
}

// rotateRolePassword sets a persistent role password. Unexported and
// generic so the mechanism is testable against scratch roles without
// ever touching a real credential in the suite; RotateAppRole is the
// thin production wrapper over AppRoleName.
func rotateRolePassword(ctx context.Context, owner *pgxpool.Pool, role, password string) error {
	if !rxHexPassword.MatchString(password) {
		return fmt.Errorf("store: role password must be lowercase hex, 32+ chars")
	}
	if !rxRoleName.MatchString(role) {
		return fmt.Errorf("store: refusing unsafe role name")
	}
	var exists bool
	if err := owner.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, role,
	).Scan(&exists); err != nil {
		return fmt.Errorf("store: rotate role check: %w", err)
	}
	if !exists {
		return fmt.Errorf("store: rotate refused: role %q does not exist", role)
	}
	if _, err := owner.Exec(ctx, `ALTER ROLE `+role+` WITH PASSWORD '`+password+`'`); err != nil {
		return fmt.Errorf("store: rotate password: %w", err)
	}
	return nil
}

func ensureConnect(ctx context.Context, owner *pgxpool.Pool, role string) error {
	if !rxRoleName.MatchString(role) {
		return fmt.Errorf("store: refusing unsafe role name")
	}
	var dbname string
	if err := owner.QueryRow(ctx, `SELECT current_database()`).Scan(&dbname); err != nil {
		return fmt.Errorf("store: provision database: %w", err)
	}
	grant := `GRANT CONNECT ON DATABASE ` + pgx.Identifier{dbname}.Sanitize() + ` TO ` + role
	if _, err := owner.Exec(ctx, grant); err != nil {
		return fmt.Errorf("store: provision connect: %w", err)
	}
	return nil
}

// UpsertOperator provisions or updates one (tenant, binding) operator row.
// It runs on an owner connection; the SECURITY DEFINER function constrains
// the write to the operators table.
func UpsertOperator(ctx context.Context, owner *pgxpool.Pool, tenantID, bindingID string, active bool) error {
	if tenantID == "" || bindingID == "" {
		return fmt.Errorf("store: tenant and binding are required")
	}
	if _, err := owner.Exec(ctx, `SELECT studio.operator_upsert($1, $2, $3)`,
		tenantID, bindingID, active); err != nil {
		return fmt.Errorf("store: upsert operator: %w", err)
	}
	return nil
}

// OffboardOperator deactivates one (tenant, binding) operator and revokes
// its outstanding invitations atomically. Owner connection only; the
// SECURITY DEFINER function owns the write set. This is the only
// offboarding path that revokes invites — raw-SQL deactivation stops
// issuance but leaves invite hygiene to the reactivation trigger.
func OffboardOperator(ctx context.Context, owner *pgxpool.Pool, tenantID, bindingID string) error {
	if tenantID == "" || bindingID == "" {
		return fmt.Errorf("store: tenant and binding are required")
	}
	if _, err := owner.Exec(ctx, `SELECT studio.operator_offboard($1, $2)`,
		tenantID, bindingID); err != nil {
		return fmt.Errorf("store: offboard operator: %w", err)
	}
	return nil
}
