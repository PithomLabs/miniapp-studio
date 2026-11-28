package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// testReadMigrationFile reads the on-disk migration source for the
// embed-vs-disk freshness comparison. Test-only.
func testReadMigrationFile(t *testing.T, name string) ([]byte, error) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("cannot locate test source")
	}
	return os.ReadFile(filepath.Join(filepath.Dir(thisFile), "migrations", name))
}

const locatorFixture = `-- leading comment; with semicolon
CREATE ROLE probe_a;
DO $$ BEGIN
  IF 1 = 1 THEN
    RAISE NOTICE 'semi;colon';
  END IF;
END $$;
GRANT probe_a TO SESSION_USER WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
`

func TestLocateStatement(t *testing.T) {
	src := []byte(locatorFixture)
	// Position inside statement 1 (CREATE ROLE).
	if idx, _ := locateStatement(src, int32(strings.Index(locatorFixture, "CREATE ROLE")+1)); idx != 1 {
		t.Errorf("CREATE ROLE position maps to statement %d, want 1", idx)
	}
	// Position inside the dollar-quoted DO block: still statement 2,
	// despite interior semicolons.
	if idx, _ := locateStatement(src, int32(strings.Index(locatorFixture, "RAISE NOTICE")+1)); idx != 2 {
		t.Errorf("DO-body position maps to statement %d, want 2", idx)
	}
	// Position inside statement 3.
	if idx, excerpt := locateStatement(src, int32(strings.Index(locatorFixture, "GRANT probe_a")+1)); idx != 3 {
		t.Errorf("GRANT position maps to statement %d, want 3", idx)
	} else if !strings.HasPrefix(excerpt, "GRANT probe_a") {
		t.Errorf("excerpt = %q", excerpt)
	}
	// Out-of-range positions are unmappable, not errors.
	if idx, _ := locateStatement(src, 0); idx != 0 {
		t.Errorf("position 0 maps to %d, want unmappable", idx)
	}
	if idx, _ := locateStatement(src, int32(len(src)+99)); idx != 0 {
		t.Errorf("past-end position maps to %d, want unmappable", idx)
	}
}

type stubVersioner struct{ ver uint }

func (s stubVersioner) Version() (uint, bool, error) { return s.ver, true, nil }

func TestDiagnoseMigrateErr(t *testing.T) {
	pgErr := &pgconn.PgError{
		Severity: "ERROR", Code: "42501", Message: `must be able to SET ROLE "studio_lookup"`,
		Position: 10, Routine: "check_can_set_role",
	}
	// Wrap twice: errors.As must still reach the PgError.
	err := diagnoseMigrateErr(stubVersioner{ver: 2}, "up",
		fmt.Errorf("migration failed: %w", fmt.Errorf("driver: %w", pgErr)))
	msg := err.Error()
	for _, want := range []string{"schema version 2", "dirty=true", "000002_schema.up.sql", "sha256 ",
		"sqlstate=42501", "position=10", "routine=check_can_set_role", "SET ROLE"} {
		if !strings.Contains(msg, want) {
			t.Errorf("diagnostic lacks %q: %s", want, msg)
		}
	}
	// Non-PgError failures still pin version + hash.
	plain := diagnoseMigrateErr(stubVersioner{ver: 1}, "up", errors.New("boom"))
	if !strings.Contains(plain.Error(), "000001_roles.up.sql") || !strings.Contains(plain.Error(), "boom") {
		t.Errorf("plain diagnostic = %q", plain.Error())
	}
}

func TestMigrationDigestMatchesDisk(t *testing.T) {
	// Freshness at test time: embedded bytes equal the source files,
	// so tests below assert on what actually ships.
	for _, ver := range []uint{1, 2} {
		name, sum, raw := migrationDigest(ver)
		if name == "" || sum == "" || len(raw) == 0 {
			t.Fatalf("version %d: no embedded bytes", ver)
		}
		disk, err := testReadMigrationFile(t, name)
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != string(disk) {
			t.Errorf("version %d: embedded bytes drift from source file %s", ver, name)
		}
	}
}

// TestMigrationFunctionOwnershipDiscipline pins the 000027 invariant
// for every migration that touches lookup-owned functions (live
// 42501 on the first 000031 attempt): function DDL/ACL executes
// ONLY between SET ROLE studio_lookup and RESET ROLE, while
// table-owner operations (CREATE TRIGGER, table GRANT/REVOKE)
// stay outside the role block.
func TestMigrationFunctionOwnershipDiscipline(t *testing.T) {
	up, err := testReadMigrationFile(t, "000031_offboard_correction.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(up)
	setAt := strings.Index(text, "SET ROLE studio_lookup;")
	resetAt := strings.Index(text, "RESET ROLE;")
	if setAt < 0 || resetAt < 0 || resetAt < setAt {
		t.Fatal("000031 up lacks a SET ROLE/RESET ROLE block")
	}
	inside := text[setAt:resetAt]
	outside := text[:setAt] + text[resetAt:]
	for _, want := range []string{"CREATE OR REPLACE FUNCTION studio.invitation_accept", "CREATE OR REPLACE FUNCTION studio.operator_offboard", "GRANT EXECUTE ON FUNCTION studio.invitation_accept"} {
		if !strings.Contains(inside, want) {
			t.Errorf("000031 up: %q outside the role block", want)
		}
	}
	for _, want := range []string{"CREATE TRIGGER operators_reactivation_revoke", "REVOKE DELETE ON studio.invitations", "GRANT UPDATE (revoked_at)"} {
		if !strings.Contains(outside, want) {
			t.Errorf("000031 up: %q inside the role block (table-owner op)", want)
		}
	}
	down, err := testReadMigrationFile(t, "000031_offboard_correction.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	dtext := string(down)
	dset := strings.Index(dtext, "SET ROLE studio_lookup;")
	dreset := strings.Index(dtext, "RESET ROLE;")
	if dset < 0 || dreset < 0 || dreset < dset {
		t.Fatal("000031 down lacks a SET ROLE/RESET ROLE block")
	}
	if !strings.Contains(dtext[dset:dreset], "DROP FUNCTION IF EXISTS studio.operator_offboard") {
		t.Error("000031 down: function drops outside the role block")
	}
	down29, err := testReadMigrationFile(t, "000029_invitations.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	d29 := string(down29)
	s29 := strings.Index(d29, "SET ROLE studio_lookup;")
	r29 := strings.Index(d29, "RESET ROLE;")
	if s29 < 0 || r29 < 0 || r29 < s29 || !strings.Contains(d29[s29:r29], "DROP FUNCTION IF EXISTS studio.invitation_accept") {
		t.Error("000029 down: function drop outside a role block")
	}
}
