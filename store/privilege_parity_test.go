package store

import (
	"context"
	"testing"

	"github.com/PithomLabs/miniapp-studio/dbenv"
)

// TestPrivilegeParity pins U5.12 invariant 5: the test role carries
// EXACTLY the live role's privilege set on run databases — derived
// from the migration grant audit (000002/3/5/6/9/10/11/12/13/14/28/29/30), and
// enforced here against the live catalog so any future migration that
// widens studio_app without widening the mirror fails loudly.
//
// Each entry asserts has_privilege(test) == has_privilege(live) ==
// want. A new table granted to the app role but missing from
// dbenv.GrantTestPrivileges breaks parity on the test side; a grant
// the mirror invents beyond the app role breaks it on the live side.
func TestPrivilegeParity(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	live, test := AppRoleName, dbenv.TestRoleName
	check := func(kind, object, priv string, want bool) {
		t.Helper()
		var fn string
		switch kind {
		case "table":
			fn = "has_table_privilege"
		case "sequence":
			fn = "has_sequence_privilege"
		case "function":
			fn = "has_function_privilege"
		case "schema":
			fn = "has_schema_privilege"
		default:
			t.Fatalf("unknown kind %q", kind)
		}
		var gotLive, gotTest bool
		q := `SELECT ` + fn + `($1, $2, $3)`
		if err := env.owner.QueryRow(ctx, q, live, object, priv).Scan(&gotLive); err != nil {
			t.Fatalf("%s %s %s live: %v", kind, object, priv, err)
		}
		if err := env.owner.QueryRow(ctx, q, test, object, priv).Scan(&gotTest); err != nil {
			t.Fatalf("%s %s %s test: %v", kind, object, priv, err)
		}
		if gotLive != want || gotTest != want {
			t.Errorf("%s %s %s: live=%v test=%v, want %v", kind, object, priv, gotLive, gotTest, want)
		}
	}
	studioTables := []string{"apps", "app_specs", "deployments", "runs",
		"conversations", "messages", "builds", "turns", "questions",
		"commands", "events", "conversation_stream", "artifacts", "approvals",
		"artifact_blobs", "artifact_verifications", "artifact_test_runs",
		"checkpoints", "sprite_lifecycle_receipts",
		"broker_credentials", "broker_leases",
		"invitations", "capability_snapshots"}
	for _, table := range studioTables {
		for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "DELETE"} {
			check("table", "studio."+table, priv, true)
		}
		// TRUNCATE is the one DML-adjacent privilege migrations never
		// grant; parity pins the negative too.
		check("table", "studio."+table, "TRUNCATE", false)
	}
	for _, table := range []string{"operators", "sessions"} {
		for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "DELETE"} {
			check("table", "studio."+table, priv, false)
		}
	}
	for _, table := range []string{"invocations", "approvals", "audit_events", "effect_records", "kill_switches"} {
		for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "DELETE"} {
			check("table", "app."+table, priv, true)
		}
	}
	check("table", "studio_schema_migrations", "SELECT", true)
	check("table", "studio_schema_migrations", "INSERT", false)
	check("sequence", "app.audit_events_id_seq", "USAGE", true)
	check("sequence", "app.audit_events_id_seq", "SELECT", true)
	for _, fn := range []string{
		"studio.session_lookup(text)",
		"studio.session_create(text,text,text,text,timestamptz)",
		"studio.session_revoke(text)",
		"studio.invitation_accept(text)",
	} {
		check("function", fn, "EXECUTE", true)
	}
	check("function", "studio.operator_upsert(text,text,boolean)", "EXECUTE", false)
	check("schema", "studio", "USAGE", true)
	check("schema", "app", "USAGE", true)
	var liveConn, testConn bool
	if err := env.owner.QueryRow(ctx,
		`SELECT has_database_privilege($1, current_database(), 'CONNECT')`, live).Scan(&liveConn); err != nil {
		t.Fatal(err)
	}
	if err := env.owner.QueryRow(ctx,
		`SELECT has_database_privilege($1, current_database(), 'CONNECT')`, test).Scan(&testConn); err != nil {
		t.Fatal(err)
	}
	if !testConn {
		t.Error("test role lacks CONNECT on the run database")
	}
	t.Logf("live CONNECT on run database: %v (informational; tests never dial it)", liveConn)
	if t.Failed() {
		t.Logf("function ACL state:\n%s", dumpFunctionACLs(ctx, env.owner))
	}
}
