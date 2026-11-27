package dbenv

// Role-capability diagnostics (SQLSTATE 42501 investigation). The
// collector reports membership capabilities as booleans only — no
// names beyond the queried role, no credentials, nothing secret.
// PostgreSQL membership is not Boolean: MEMBER, INHERIT (USAGE),
// SET, and ADMIN are independent capabilities, and the CREATEROLE
// auto-grant sets ADMIN TRUE with SET FALSE.

import (
	"context"
	"fmt"
)

// RoleReport is the six-boolean membership signature for one role as
// seen by the current session. JSON keys match the architect's
// diagnostic specification verbatim.
type RoleReport struct {
	Role               string `json:"role"`
	IsMember           bool   `json:"is_member"`
	CanSetRole         bool   `json:"can_set_role"`
	InheritsPrivileges bool   `json:"inherits_privileges"`
	DirectSetOption    bool   `json:"direct_set_option"`
	DirectAdminOption  bool   `json:"direct_admin_option"`
	CurrentMatches     bool   `json:"current_matches_session"`
}

// CollectRoleReport runs the capability diagnostic for role. The role
// name travels as a query parameter, never interpolated into SQL.
func CollectRoleReport(ctx context.Context, q Querier, role string) (RoleReport, error) {
	var r RoleReport
	r.Role = role
	err := q.QueryRow(ctx, `
SELECT
  pg_has_role(session_user, $1, 'MEMBER'),
  pg_has_role(session_user, $1, 'SET'),
  pg_has_role(session_user, $1, 'USAGE'),
  COALESCE((
    SELECT bool_or(am.set_option)
    FROM pg_auth_members am
    JOIN pg_roles target ON target.oid = am.roleid
    JOIN pg_roles member ON member.oid = am.member
    WHERE target.rolname = $1 AND member.rolname = session_user
  ), false),
  COALESCE((
    SELECT bool_or(am.admin_option)
    FROM pg_auth_members am
    JOIN pg_roles target ON target.oid = am.roleid
    JOIN pg_roles member ON member.oid = am.member
    WHERE target.rolname = $1 AND member.rolname = session_user
  ), false),
  current_user = session_user`, role).Scan(
		&r.IsMember, &r.CanSetRole, &r.InheritsPrivileges,
		&r.DirectSetOption, &r.DirectAdminOption, &r.CurrentMatches,
	)
	if err != nil {
		return RoleReport{}, fmt.Errorf("dbenv: role report: %w", err)
	}
	return r, nil
}
