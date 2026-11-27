// Package dbenv holds the database-environment contract shared by the
// Go test suite and the `studio dev` command interface: the Neon
// endpoint policy, run-database naming, cluster roles, privilege
// mirror, and server fingerprint collectors.
//
// Trust model (Neon-first, ADR-004/006; the U5.12 loopback allowlist
// was retired by the T1 C4 redesign, not adapted):
//
//   - Known Neon branches only (endpoint policy + TLS + PG18 gate).
//     A mispointed URL is refused before any connection opens, and
//     loopback/private hosts are refused outright.
//   - The test role carries a FIXED test-only password so concurrent
//     creators converge; nothing rotates it, and this package
//     exposes no rotation API. It is valid only on non-production
//     branches holding synthetic data (ADR-015). Per-run credential
//     issuance is T2-controller work, which retires this residual.
//   - Run databases match ^studio_test_run_<pid>_<ts>_<rand>$;
//     grants, resets, and app-DSN rewrites refuse anything else.
//   - Roles are created once and never modified (duplicate_object
//     convergence for racers).
//
// Import discipline: production request paths (server, tools, runner)
// must never import this package — enforced by
// TestNoProdDbenvImport, which allows only the test harness, tests,
// and the `studio dev` command (cmd/studio/dev.go, cmd/studio/bind.go).
package dbenv

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WantPGMajor is the only accepted server major version. PG18 tracks
// the Neon project default (owner decision 2026-11-27, amending the
// earlier PG19 mandate in ADR-004); PG19-capable code paths are
// unchanged, only the gate value moved.
const WantPGMajor = 18

// CheckPGMajor verifies the connected server is PostgreSQL 18. Called
// immediately after connecting to an allowlisted endpoint and before
// any database is provisioned on it.
func CheckPGMajor(ctx context.Context, pool *pgxpool.Pool) error {
	var num int
	if err := pool.QueryRow(ctx, `SELECT current_setting('server_version_num')::int`).Scan(&num); err != nil {
		return fmt.Errorf("dbenv: server version check: %w", err)
	}
	if num/10000 != WantPGMajor {
		return fmt.Errorf("dbenv: unapproved endpoint: requires PostgreSQL %d, server reports major %d",
			WantPGMajor, num/10000)
	}
	return nil
}
