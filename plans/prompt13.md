I inspected the actual GitHub Actions logs. The failure is in **step 8,&#x20;**`go test -p 1 ./...`, not formatting, vetting, or compilation.

The leading failure is a PostgreSQL migration collision:

```
ERROR: policy "tenant_isolation" for table "invocations" already exists
SQLSTATE 42710
```

This causes several tests to fail, followed by:

```
Dirty database version 1. Fix and force version.
```

Affected tests include `TestGatewayCommandRoundTrip`, `TestProvisionFresh`, `TestProvisionRotationAndIdempotency`, `TestServeHappyPath`, multiple runner tests, and Phase A server tests.

The logs also show a migration lookup for `public.schema_migrations` failing because that relation does not exist. That may be a separate migration bootstrap defect or a consequence of the earlier failure; the implementation agent must establish the actual causal sequence rather than assume.

**Important:** PostgreSQL 19 itself initialized successfully in the CI container. The available evidence points toward migration/test-database lifecycle and isolation, not a PostgreSQL version incompatibility. The fix should make migrations and test setup correct under repeated and fresh test execution, not weaken RLS or simply drop existing policies.

# Implementation Directive — Investigate and Fix GitHub Actions PostgreSQL Migration Failure

## 1. Role

Act as the Senior Go Systems Architect and database reliability engineer for:

`github.com/PithomLabs/miniapp-studio`

Investigate and correct the GitHub Actions failure:

[https://github.com/PithomLabs/miniapp-studio/actions/runs/45545786772/job/138974954451#step:8:1018](https://github.com/PithomLabs/miniapp-studio/actions/runs/45545786772/job/138974954451#step:8:1018)

Your task is to establish the underlying cause, fix the violated invariant at the correct ownership boundary, and provide durable regression coverage.

Do not merely make this particular CI run green. Make the migration and test-database lifecycle correct, deterministic, isolated, and repeatable.

The governing Miniapp Studio Implementation Agent Guidelines apply throughout this task.

## 2. Verified CI failure

The workflow executed these stages successfully:

- `gofmt -l .`
- `go vet ./...`
- `go build ./...`

The failing stage was:

```
go test -p 1 ./...
```

The relevant PostgreSQL log contains:

```
ERROR: policy "tenant_isolation" for table "apps" already exists
SQLSTATE 42710
```

The Go test output identifies:

```
FAIL: TestGatewayCommandRoundTrip
gateway_test.go:153:
migration failed: policy "tenant_isolation" for table "invocations" already exists
```

Additional failures include:

```
FAIL: TestProvisionFresh
studio_test.go:100:
Dirty database version 1. Fix and force version.
```

```
FAIL: TestProvisionRotationAndIdempotency
FAIL: TestServeHappyPath
```

Additional runner and server tests fail after the database becomes dirty:

- `TestRunnerInvokeEndToEnd`
- `TestRunnerRejectsInvalidSpec`
- `TestRunnerRejectsTenantMismatch`
- `TestPhaseAEndToEnd`
- `TestPhaseARestartRecovery`

The logs also contain:

```
ERROR: relation "public.schema_migrations" does not exist
```

and:

```
FATAL: role "root" does not exist
```

These messages must be investigated in context. Do not automatically classify them as root causes.

The CI PostgreSQL container reports PostgreSQL 19.0.

## 3. Primary hypothesis — investigate, do not assume

The immediate failure indicates that a migration attempts to create an RLS policy that already exists.

The likely underlying problem is a mismatch between migration lifecycle and database lifecycle. Possibilities include:

1. Multiple tests or test helpers applying the same framework migrations to one shared database.
2. Test setup creating a partially migrated database before invoking another migration sequence.
3. Migration version tracking being absent, misplaced, or inconsistent with the actual schema.
4. A migration being executed twice because separate provisioning paths do not coordinate.
5. Test helpers assuming a fresh database when the database is already initialized.
6. Framework migration and Studio migration ownership not being clearly separated.
7. Dirty migration state being left behind after a failed migration and reused by subsequent tests.
8. Search-path or migration-table configuration causing the migrator to inspect the wrong schema.
9. Migration files implementing non-idempotent DDL without the correct migration-version guarantees.

Determine which mechanism actually produces the collision.

Do not stop after observing that `CREATE POLICY` is not idempotent. Explain why the same migration is being applied when the policy already exists.

## 4. Repository investigation

Inspect the complete relevant implementation before changing code.

At minimum inspect:

- GitHub Actions workflow YAML.
- PostgreSQL service-container configuration.
- Environment variables and database URLs used by tests.
- `internal/store` migration code and test helpers.
- Framework migration files.
- Studio migration files.
- Migration runner configuration.
- `schema_migrations` creation, schema selection, and version tracking.
- Tests identified in the failure logs.
- Shared test database helpers.
- Any application startup or provisioning paths that apply migrations.
- RLS policy creation and database bootstrap code.

Build an execution map:

```
GitHub Actions
    → PostgreSQL container
    → test environment
    → test helper / DB bootstrap
    → framework migrations
    → Studio migrations
    → migration version tracking
    → test execution
```

Establish exactly which component owns each migration and whether that migration can execute more than once.

Inspect every call site that can initialize or migrate the test database.

Search specifically for:

- Migration runner construction.
- `Up()` and equivalent migration entry points.
- `schema_migrations`.
- `CREATE POLICY`.
- `tenant_isolation`.
- Test database initialization and teardown.
- Shared package-level test fixtures.
- Framework migration invocation.
- Provisioning functions that might run migrations implicitly.

Do not infer migration ownership from filenames alone.

## 5. Required root-cause questions

Answer each question with source evidence.

### A. Why does the policy already exist?

Establish:

- Which migration created it the first time.
- Which migration or helper attempts to create it again.
- Whether these are two distinct migration runners or repeated calls through one runner.
- Whether they use the same database, schema, and role.
- Whether migration tracking correctly records the first execution.

### B. Why does the migration system report a dirty database?

Determine:

- Which migration version is marked dirty.
- Which operation fails after the version is marked dirty.
- Whether the migration transaction actually rolls back schema changes.
- Whether earlier DDL or role operations survive the failure.
- Whether later tests incorrectly reuse the dirty database.

Explain the complete failure chain.

### C. Why is `public.schema_migrations` missing?

Determine whether:

- The migration table is intentionally located elsewhere.
- The migration runner is looking in the wrong schema.
- Test helpers initialize a different migration backend.
- The database has never initialized the migration table.
- This is an unrelated expected probe or a secondary failure.

Do not create an additional migration table simply to silence the message.

### D. Are the tests properly isolated?

Determine whether tests share the same database/schema intentionally.

Inspect behavior across:

- Fresh database.
- Already-migrated database.
- Repeated migration invocation.
- Two migration runners concurrently.
- Migration interruption.
- Test failure followed by another test.
- Repeated full-suite execution.
- Different package test order.

`-p 1` serializes package execution, but it does not prove that multiple test helpers inside a package cannot initialize the same database repeatedly.

### E. Is RLS security preserved?

Verify that any correction preserves:

- RLS enabled on the intended tables.
- FORCE ROW LEVEL SECURITY where required.
- Correct tenant isolation policies.
- Correct policy roles and predicates.
- Existing owner/application role boundaries.
- Correct behavior when tenant context is absent.
- Cross-tenant read/write denial.

Do not resolve the collision by removing tenant policies, weakening RLS, disabling FORCE, or converting a restrictive policy into an overly permissive one.

## 6. Correctness requirements

The final design must satisfy these invariants.

### Migration ownership

Every migration must have one clearly defined owner and execution path.

Framework migrations and Studio migrations must not accidentally initialize each other's schemas or replay each other's DDL.

### Version/schema consistency

The migration version table must accurately represent the migrations that have been applied.

A migration cannot be considered complete merely because its SQL ran partially.

### Repeatability

Running the intended migration initialization entry point repeatedly must produce a well-defined result.

An already-initialized database must not fail merely because its correct policies and tables exist.

Do not indiscriminately make every SQL statement idempotent to mask duplicate migration execution. First fix migration ownership and tracking. Use idempotent DDL only where it is semantically correct and compatible with migration-version guarantees.

### Failure atomicity and recovery

A failed migration must not leave an inconsistent database that subsequent tests silently reuse.

If a migration is not transactionally atomic, document and test its recovery semantics.

Do not automatically force migration versions or delete schema objects to bypass a dirty state.

### Test isolation

Tests must not contaminate one another through shared database state unless that sharing is intentional and explicitly coordinated.

A test that expects a fresh database must receive a genuinely isolated database/schema or use a correct reusable fixture.

### Concurrency

If multiple migration initializers can target the same database, use an appropriate database-level coordination mechanism.

Do not rely solely on an in-process mutex if multiple processes or separate test binaries can migrate the same database.

## 7. Adversarial regression matrix

Add regression tests that prove the underlying invariant.

At minimum:

| Scenario                                              | Expected result                                  |
| ----------------------------------------------------- | ------------------------------------------------ |
| Fresh database migration                              | All migrations complete successfully             |
| Repeated migration invocation                         | Correct no-op/already-current behavior           |
| Framework migrations followed by Studio migrations    | No policy collision                              |
| Studio migrations followed by repeated initialization | No duplicate DDL                                 |
| Previously initialized database                       | Tests do not replay completed migrations         |
| Two concurrent migration attempts                     | One coherent migration outcome                   |
| Migration fails midway                                | Dirty state is detected and recovery is explicit |
| Subsequent test after migration failure               | Does not silently reuse contaminated state       |
| RLS policies after migration                          | Exact expected policies remain present           |
| Cross-tenant reads/writes                             | Rejected under the intended policy               |
| Missing tenant context                                | Fail-closed behavior retained                    |
| Full suite executed repeatedly                        | Consistent results                               |

Where possible, include a test that reproduces the exact original CI failure before the correction and demonstrates that it passes afterward.

Do not delete, skip, weaken, or rename a failing test simply to make the suite green.

## 8. PostgreSQL 19 testing standard

The local development and verification baseline remains the user's existing PostgreSQL 19 Docker instance:

`127.0.0.1:5434`

For local database-backed verification:

1. Confirm the endpoint.
2. Verify `SHOW server_version` and `SELECT version()`.
3. Confirm PostgreSQL 19.x.
4. Use an isolated disposable database/schema.
5. Never destroy or reset the user's existing data.
6. If PG19 is inaccessible, stop database-backed tests and report the exact blocker.
7. Never substitute PostgreSQL 16, PostgreSQL 17, or another version.

**CI is a distinct environment.** The GitHub Actions service container can be configured independently, but it must use PostgreSQL 19 and explicitly verify its version before database tests begin.

Do not confuse the local endpoint with the GitHub Actions container's internal service hostname and port.

## 9. CI workflow verification

Inspect the actual workflow configuration and confirm:

- PostgreSQL image is pinned to the intended major/minor policy.
- Health checks guarantee readiness before tests.
- The intended database and role are created correctly.
- `TEST_DATABASE_URL` and `TEST_DBOS_DATABASE_URL` target the intended CI database.
- Credentials are not emitted in logs.
- Framework and Studio migrations initialize in the correct order.
- Migration state does not persist unexpectedly across test jobs.
- The job fails immediately and transparently on migration errors.

Do not add arbitrary sleeps as a replacement for database health checks.

Do not introduce a second database service unless the architecture requires separate databases and their ownership is explicit.

## 10. Required verification gates

After the correction:

```
gofmt -l .
go vet ./...
go build ./...
```

Then, after verifying the correct PostgreSQL 19 test database:

```
go test -p 1 -count=1 ./...
```

Run database-sensitive package race tests:

```
go test -race -p 1 -count=1 ./...
```

If the full race suite is materially inappropriate or too resource-intensive, run the required affected packages and explain the exact scope. Never claim a full race pass when only selected packages ran.

Run the relevant migration tests independently before the entire suite.

Then verify repeatability:

```
go test -p 1 -count=1 ./...
go test -p 1 -count=1 ./...
```

Both executions must pass against correctly isolated test databases, not because the first run leaves a lucky schema state that hides the problem.

Also verify the CI workflow itself. A local test result alone is insufficient to claim that the GitHub Actions configuration is fixed.

## 11. Scope boundaries

This is a database migration/test lifecycle correction.

Do not:

- Modify Agent Gateway behavior unless a concrete migration dependency requires it.
- Change the Miniapp application contract.
- Change F1, G-01, P5, or accepted architecture.
- Introduce a new migration framework.
- Introduce DBOS or another workflow engine.
- Weaken security policies.
- Reorganize unrelated tests.
- Perform live Fly/Sprite/OAuth operations.
- Commit or push changes.

Keep the change set as narrow as possible while fixing the actual root cause.

## 12. Evidence and deliverables

Create:

`docs/ci-postgres-migration-failure-report.md`

Include:

1. Original failing workflow and job URL.
2. Exact failed step and failure messages.
3. Migration execution map.
4. Root cause, with the full failure sequence.
5. Why the existing behavior failed.
6. Durable correction and the owning component.
7. Migration ownership and version-tracking explanation.
8. Test isolation and concurrency design.
9. RLS security verification.
10. Red/green regression test evidence.
11. PostgreSQL version and exact test commands.
12. CI workflow changes and verification.
13. Remaining limitations or unverified claims.
14. Explicit statement that no commits or pushes were made.

Update relevant existing migration documentation if necessary. Do not write a generic migration tutorial.

## 13. Final acceptance criteria

The work is complete only when:

- The duplicate-policy collision is explained and eliminated at its root cause.
- Dirty migration state cannot silently contaminate subsequent tests.
- Migration version tracking is correct.
- Framework and Studio migrations have explicit ownership.
- Repeated test initialization is safe.
- RLS policies remain correct and secure.
- The `schema_migrations` lookup behavior is explained and corrected if defective.
- The full Go test suite passes on PostgreSQL 19.
- The actual GitHub Actions workflow passes with PostgreSQL 19.
- The report contains reproducible evidence.

If a gate cannot be run, do not claim completion. Report the blocker and keep the work incomplete.

**Do not stop at making `CREATE POLICY` conditional. Solve why the migration was being replayed and why the migration tracker failed to protect the database lifecycle.**

Stop after producing the correction report and verification evidence. Do not commit, push, or perform unrelated work.

**Architectural emphasis:** the correct fix may involve migration ownership, migration tracking, or test database isolation—or a combination. The agent must prove which one caused this failure before choosing the implementation.