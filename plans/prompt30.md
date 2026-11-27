# Engineering Directive — Restore Reliable Neon Test Migrations

## Objective

Restore deterministic, repeatable migration execution across the Studio development and database-test workflows.

The objective is not merely to fix the observed hang at TestRLSEnable000016.

Identify and correct the structural cause that permits migration operations to block indefinitely.

The implementation must preserve Neon-only PostgreSQL, verify-full TLS, migration ordering, advisory locking, dirty-state semantics, RLS, tenant isolation and clean-bootstrap guarantees.

## Priority 1 — Establish incident scope

Before modifying the implementation:

1. Determine whether the DBeaver observation concerns the default database or the actual run-pattern database.
2. Capture the secret-free pg_stat_activity and pg_locks diagnostics during a reproduced hang.
3. Determine whether the old migration backend still exists.
4. Determine whether a fresh connection can execute a trivial query.
5. Inspect the actual migration version and dirty state.
6. Capture the current connection lifecycle and migration phase timings.

Do not assume H1 is proven. Distinguish transport failure, server/backend stall and protocol desynchronization.

Do not modify migration SQL to address a transport-level hang.

## Priority 2 — Fix the migration execution contract

Inspect the pinned golang-migrate v4.20.1 PGX v5 driver.

The investigation confirms that:

- It acquires a dedicated database/sql connection.
- It uses context.Background for several migration operations.
- StatementTimeout applies to Run, not SetVersion.
- SetVersion starts its transaction with an unbounded BeginTx.
- Lock, Unlock, Version and other bookkeeping operations also contain unbounded calls.

Implement a small, maintained internal adaptation of the PGX v5 migration backend.

Do not fork the entire migration framework unnecessarily.

Do not run migrations in an untracked goroutine and simply return when a timer fires.

Do not claim that a context passed to store.Migrate provides cancellation until the actual database driver consumes it.

## Priority 3 — Establish universal bounded execution

Ensure the migration execution context reaches:

- Connection establishment and initialization.
- Ping.
- Dedicated database/sql connection acquisition.
- Advisory lock acquisition.
- Migration version reads.
- Dirty-version transaction.
- Migration body execution.
- Clean-version transaction.
- Transaction commit and rollback.
- Advisory lock release.
- Driver and connection cleanup.

Use explicit child deadlines for operations, bounded by the overall migration context.

Use context-aware APIs, including BeginTx, ExecContext, QueryContext and QueryRowContext.

The transaction must use the same cancellation context throughout its lifetime.

The cleanup path must have a bounded opportunity to release the advisory lock. If the connection is unusable, close it and record the cleanup failure.

A timed-out or protocol-compromised connection must never be returned to normal application use.

Keep server-side statement and lock timeouts as complementary protections.

Keep connection establishment timeout and TCP keepalive settings as additional network protections, not substitutes for query deadlines.

## Priority 4 — Preserve migration correctness

Never blindly retry a migration after an ambiguous network failure.

A timeout can occur after the server received a command but before the client received its response.

Reconcile migration version and dirty state through a fresh, independently verified connection before deciding whether another execution is safe.

Preserve golang-migrate's dirty-marker protocol.

A failure between migration execution and clean-version recording must remain visible as an incomplete migration.

Do not automatically force a dirty migration to clean.

Do not weaken advisory locking or remove its mutual-exclusion guarantee.

Do not drop databases automatically outside the established disposable run-pattern cleanup authorization.

## Priority 5 — Correct preflight and connection lifecycle

Review PGX v5 connection creation.

Ensure health verification applies to the exact dedicated connection used by the migration driver, not merely another connection acquired by DB.Ping.

Use bounded PingContext after acquiring the dedicated connection.

Record a safe application_name to distinguish migration sessions in pg_stat_activity.

Review connection reuse, maximum lifetime and idle behavior.

Do not introduce aggressive unnecessary reconnect loops.

A connection health failure should produce an actionable error, not hidden retries.

## Priority 6 — Migration instrumentation

Record structured, secret-free events for:

- Connection acquisition.
- Health checks.
- Lock acquisition and release.
- Version reads.
- Dirty marker writes.
- Migration body start and completion.
- Clean marker writes.
- Commit and rollback.
- Connection invalidation and close.

Each event should contain:

- Migration version.
- Operation/phase.
- Elapsed duration.
- Configured deadline.
- Success/failure.
- Error classification.
- Whether the connection remains reusable.

Never record credentials, DSNs, hostnames, SQL parameters or raw connection URLs.

Identify the exact operation responsible for every timeout.

Ensure instrumentation does not itself introduce a blocking logging path.

## Priority 7 — Regression testing without self-hosted PostgreSQL

Self-hosted PostgreSQL is prohibited, including Docker and Testcontainers.

Create deterministic fault-injection tests.

At minimum:

1. Simulate a driver operation that blocks until its context is canceled.
2. Verify SetVersion's BeginTx is bounded.
3. Simulate a server that stops responding after receiving a command.
4. Verify the PGX connection operation returns within its deadline.
5. Verify a timed-out connection cannot be reused.
6. Verify a migration failure does not mark the migration clean.
7. Verify advisory lock behavior remains correct.
8. Verify cleanup does not leak goroutines or hold test resources indefinitely.
9. Verify a subsequent healthy migration can succeed.
10. Verify all ordinary migration phases use bounded contexts.

Prefer an in-process PostgreSQL wire-protocol fault fixture or an appropriately isolated network fault-injection test. A fake driver is useful for unit-testing context propagation but must not be presented as proof of actual PGX network cancellation.

No local PostgreSQL server is required for these tests.

Use Neon for actual database integration and migration acceptance.

## Priority 8 — Repeatable end-to-end proof

Use the existing Neon non-production branch.

Run:

1. Clean database bootstrap.
2. Full migration Up.
3. Complete DB-dependent suite.
4. Migration Down to version 15.
5. Existing RLS enablement and assertion checks.
6. Migration Up replay through version 25.
7. Repeat the complete sequence.

Use the same persistent Neon branch while generating fresh authorized run-pattern databases.

Record exact migration versions, clean/dirty states, test counts, timings, and connection lifecycle evidence.

No unexplained skips or mock-green results.

## Acceptance criteria

The issue is resolved only when:

- All migration database operations have bounded execution.
- The original hang can no longer block the suite indefinitely.
- The failure mode is reproduced by a deterministic regression test.
- Connection cancellation and cleanup are demonstrated.
- The advisory lock remains effective.
- Dirty-state behavior remains correct.
- Repeated Neon-backed Up/Down/Up cycles complete successfully.
- RLS and tenant-isolation tests remain green.
- A full clean-bootstrap test cycle is repeatable.
- The final evidence distinguishes local fault simulation from live Neon verification.

## Governing principle

The migration engine must not depend on Go's ten-minute test timeout as its primary protection against network failure.

Restore reliable migration execution by correcting the end-to-end database operation lifecycle, not by increasing test timeouts, adding blind retries, suppressing failures, or bypassing the migration framework's safety mechanisms.