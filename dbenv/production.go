package dbenv

import (
	"fmt"
	"strings"
)

// productionBranchNames is the name-denylist root: exact
// production-identifying branch names, matched case-insensitively.
// Single point of maintenance; widen only by explicit owner decision
// (over-broad names brick legitimate dev topologies).
var productionBranchNames = []string{"production", "prod"}

// RefuseProductionBranch is the single production guard every dev verb
// calls after loading its branch policy and before touching Neon. Two
// independent roots, either one sufficient to refuse:
//
//  1. Platform identity (primary): the branch's own `protected` marker
//     from the Neon API, recorded at bind time. Dashboard-controlled,
//     stable across renames.
//  2. Name denylist (defense in depth): exact production-identifying
//     names, case-insensitive. Catches unprotected branches that are
//     production by convention (notably on plan tiers without branch
//     protection) — and costs nothing.
//
// Unknown identity (empty branch name) refuses: a policy that cannot
// name its branch cannot prove it is not production. Callers must
// surface the error verbatim; every error names its remedy.
//
// Staleness model (honest limitation): the binding is a point-in-time
// attestation. Toggling protection in the dashboard after binding is
// invisible until re-bind — re-bind after any protection change. There
// is deliberately no live re-check (it would need authed Neon access
// in every verb path including CI); the recorded `bound_at` marks
// attestation age.
func RefuseProductionBranch(pol EndpointPolicy) error {
	name := strings.TrimSpace(pol.BranchName)
	if name == "" {
		return fmt.Errorf("dbenv: refusing branch with unknown identity (re-run dev:bind to record it)")
	}
	for _, denied := range productionBranchNames {
		if strings.ToLower(name) == denied {
			return fmt.Errorf("dbenv: refusing denylisted branch name %q (dev tooling never touches production; bind a development branch instead)", pol.BranchName)
		}
	}
	if pol.Protected {
		branch := pol.BranchName
		if pol.ProjectName != "" {
			branch += " in project " + pol.ProjectName
		}
		return fmt.Errorf("dbenv: refusing platform-protected branch %q (dev tooling never touches production; bind a development branch instead)", branch)
	}
	return nil
}
