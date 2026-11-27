package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestStudioLookupSetRepair reproduces the PostgreSQL 16+ CREATEROLE
// auto-membership lifecycle against a scratch probe role and proves
// the canonical repair. Live-only: without a bound Neon branch it
// skips honestly via testdb.RunDSN (pending-live, never green).
//
// Scenario (architect §7): CREATE ROLE as the non-superuser owner,
// observe ADMIN TRUE / SET FALSE / INHERIT FALSE, demonstrate the
// existence-check trap, apply the explicit SET grant, prove transfer
// works, prove the app role cannot follow, and leave no trace.
func TestStudioLookupSetRepair(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	// Probe identifiers derive from fixed prefixes + PID/nanotime ints
	// only — safe for interpolation; no external input reaches them.
	probe := fmt.Sprintf("studio_probe_%d_%d", os.Getpid(), time.Now().UnixNano())
	fn := "studio.probe_transfer_fn"
	cleanup := func() {
		// The canonical repair leaves the owner holding SET (not
		// ADMIN) on the probe, so object drops run AS the probe over
		// a dedicated connection, with RESET guaranteed before the
		// connection returns to the pool. Every step surfaces its
		// error: the previous version swallowed the DROP OWNED
		// result, hiding the decisive evidence on failure.
		if _, err := env.owner.Exec(ctx, `DROP OWNED BY `+probe); err != nil {
			t.Logf("cleanup: DROP OWNED BY %s: %v", probe, err)
		}
		func() {
			conn, err := env.owner.Acquire(ctx)
			if err != nil {
				t.Errorf("cleanup acquire: %v", err)
				return
			}
			defer conn.Release()
			defer func() {
				// Never return a role-switched connection.
				if _, err := conn.Exec(ctx, `RESET ROLE`); err != nil {
					t.Errorf("cleanup RESET ROLE: %v", err)
				}
			}()
			if _, err := conn.Exec(ctx, `SET ROLE `+probe); err != nil {
				t.Errorf("cleanup SET ROLE %s: %v", probe, err)
				return
			}
			if _, err := conn.Exec(ctx, `DROP FUNCTION IF EXISTS `+fn+`()`); err != nil {
				t.Errorf("cleanup drop %s: %v", fn, err)
			}
		}()
		if _, err := env.owner.Exec(ctx, `REVOKE CREATE ON SCHEMA studio FROM `+probe); err != nil {
			t.Logf("cleanup: revoke schema CREATE: %v", err)
		}
		if _, err := env.owner.Exec(ctx, `DROP OWNED BY `+probe); err != nil {
			t.Logf("cleanup: second DROP OWNED: %v", err)
		}
		if _, err := env.owner.Exec(ctx, `DROP ROLE IF EXISTS `+probe); err != nil {
			t.Errorf("cleanup probe role: %v\nprobe dependencies:\n%s",
				err, dumpRoleDeps(ctx, env.owner, probe))
		}
	}
	t.Cleanup(cleanup)

	mustExec := func(sql, what string) {
		t.Helper()
		if _, err := env.owner.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	mustBool := func(query, what string, want bool) {
		t.Helper()
		var got bool
		if err := env.owner.QueryRow(ctx, query, probe).Scan(&got); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if got != want {
			t.Errorf("%s = %v, want %v", what, got, want)
		}
	}

	// 1. CREATE ROLE as the non-superuser migration principal.
	mustExec(`CREATE ROLE `+probe+` WITH NOLOGIN NOBYPASSRLS`, "create probe role")

	// 2. The automatic membership: ADMIN TRUE, SET FALSE, INHERIT FALSE.
	var admin, setOpt, inheritOpt bool
	err := env.owner.QueryRow(ctx, `SELECT am.admin_option, am.set_option, am.inherit_option
		FROM pg_auth_members am
		JOIN pg_roles target ON target.oid = am.roleid
		JOIN pg_roles member ON member.oid = am.member
		WHERE target.rolname = $1 AND member.rolname = session_user`, probe,
	).Scan(&admin, &setOpt, &inheritOpt)
	if err != nil {
		t.Fatalf("auto-membership row: %v", err)
	}
	if !admin || setOpt || inheritOpt {
		t.Errorf("auto-membership = admin=%v set=%v inherit=%v, want true/false/false",
			admin, setOpt, inheritOpt)
	}

	// 3. The trap: membership EXISTS while SET ROLE is prohibited.
	mustBool(`SELECT EXISTS (SELECT 1 FROM pg_auth_members am
		JOIN pg_roles r ON r.oid = am.roleid JOIN pg_roles u ON u.oid = am.member
		WHERE r.rolname = $1 AND u.rolname = session_user)`,
		"membership row exists", true)
	mustBool(`SELECT pg_has_role(session_user, $1, 'SET')`, "SET before repair", false)

	// 4-5. Canonical repair (twice: idempotent re-execution) + SET proof.
	for i := 0; i < 2; i++ {
		mustExec(`GRANT `+probe+` TO SESSION_USER WITH ADMIN FALSE, INHERIT FALSE, SET TRUE`,
			"explicit SET grant")
	}
	mustBool(`SELECT pg_has_role(session_user, $1, 'SET')`, "SET after repair", true)
	mustBool(`SELECT pg_has_role(session_user, $1, 'USAGE')`, "inherit after repair", false)

	// 6. Ownership transfer now succeeds.
	mustExec(`CREATE FUNCTION `+fn+`() RETURNS INT LANGUAGE sql AS $$ SELECT 1 $$`, "create scratch fn")
	mustExec(`GRANT CREATE ON SCHEMA studio TO `+probe, "probe schema CREATE")
	mustExec(`ALTER FUNCTION `+fn+`() OWNER TO `+probe, "transfer ownership")

	// 7. The application role cannot SET ROLE to the probe.
	var appSet bool
	if err := env.owner.QueryRow(ctx,
		`SELECT pg_has_role($1, $2, 'SET')`, AppRoleName, probe).Scan(&appSet); err != nil {
		t.Fatalf("app SET check: %v", err)
	}
	if appSet {
		t.Error("application role can SET ROLE to probe")
	}

	// 8. Probe role carries no bypass; RLS/FORCE coverage lives in the
	// dedicated suites (referenced, not duplicated here).
	mustBool(`SELECT NOT rolbypassrls FROM pg_roles WHERE rolname = $1`, "probe NOBYPASSRLS", true)
}
