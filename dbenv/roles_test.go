package dbenv

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// roleFakeQuerier answers the capability diagnostic with fixed
// booleans and records the bound role argument.
type roleFakeQuerier struct {
	vals []any
	role any
}

func (f *roleFakeQuerier) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	if len(args) > 0 {
		f.role = args[0]
	}
	return fakeRow{vals: f.vals}
}

func (f *roleFakeQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return &fakeRows{}, nil
}

func TestCollectRoleReport(t *testing.T) {
	fq := &roleFakeQuerier{vals: []any{true, false, false, false, true, true}}
	r, err := CollectRoleReport(context.Background(), fq, "studio_lookup")
	if err != nil {
		t.Fatal(err)
	}
	if fq.role != "studio_lookup" {
		t.Errorf("role bound as %v, want studio_lookup (must be parameterized, never interpolated)", fq.role)
	}
	if !r.IsMember || r.CanSetRole || r.InheritsPrivileges || r.DirectSetOption || !r.DirectAdminOption || !r.CurrentMatches {
		t.Errorf("booleans mis-scanned: %+v", r)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"is_member", "can_set_role", "inherits_privileges", "direct_set_option", "direct_admin_option", "current_matches_session"} {
		if !strings.Contains(string(raw), `"`+key+`"`) {
			t.Errorf("report JSON lacks %q: %s", key, raw)
		}
	}
}
