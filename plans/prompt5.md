I recommend a **focused P4 Foundation Security Review**, rather than reviewing P4 tools and screens before they exist.

There is one particularly important attack to test: whether the application database role can invoke the `session.create` SECURITY DEFINER function directly and mint a valid session without passing the server's configured operator-credential verification. The normal login handler working correctly does not prove that database boundary is secure.

Here is the review prompt.

# Adversarial Security Review — Miniapp Studio P4 Foundation

## 1. Role and objective

Act as a principal application-security architect, PostgreSQL security specialist, and senior Go reviewer.

Conduct an adversarial, evidence-driven review of the currently implemented P4 foundation in the Miniapp Studio repository.

Your task is to determine whether the server, session authentication, three-role database security model, embedded migrations, tenant isolation, and startup configuration are safe foundations for the remaining P4 implementation.

This is a READ-ONLY review.

Do not modify source files, migrations, tests, configuration, dependencies, or documentation. Do not commit anything.

The objective is to discover material security defects before the Studio tools, registry, and six screens are built on this foundation.

## 2. Established project state

P0, P1, P2, and P3 are complete and accepted.

P4 foundation has been implemented, but Studio tools and the six screens are still pending.

### P4 implementation reported

**Database (`store/`)**

- Four embedded migrations.
- Six tenant-scoped tables.
- RLS enabled and forced.
- Tenant-scoped unique constraints.
- Sessions and operators inaccessible directly to the application role.
- Three SECURITY DEFINER functions: lookup, create, revoke.
- Functions owned by a dedicated non-login lookup role.
- PUBLIC execution revoked.
- Operator provisioning limited to pre-provisioned records and bounded expiries.
- Regression tests for direct-access denial, invalid sessions, role attributes, RLS, tenant isolation, rollback, and idempotent provisioning.

**Server (`server/`, `cmd/studio`)**

- Fail-closed configuration.
- Secret loading through `*_FILE`.
- Conflicting secret sources rejected.
- Fly credential pair validated together.
- Server wiring following the framework's production construction pattern.
- Session middleware: cookie → SHA-256 token hash → lookup → tenant context.
- Single-operator bootstrap credential from server configuration.
- Login/logout and secure cookie flags.
- Constant-time credential comparison.
- Redirect validation.
- Health and readiness endpoints.
- Structured errors and redacted logging.

**Reported verification**

- 13 store tests.
- 13 server tests.
- PostgreSQL-backed testing.
- Serial database test execution.
- Disposable restricted roles for privilege and RLS tests.
- All work remains uncommitted.
- Miniapp remains untouched at `73351a8`.

Treat these as claims to verify, not facts to assume.

## 3. Critical attack surface A — database role separation

Inspect every migration and the actual PostgreSQL catalogs.

Identify the precise roles used for:

1. Migration/table ownership.
2. SECURITY DEFINER function ownership and restricted session lookup.
3. Application runtime access.

Do not assume their names or attributes from comments. Derive their actual properties from PostgreSQL.

### A1. Migration/owner role

Verify:

- It is used only for migrations and administrative provisioning.
- It is not the application's runtime connection.
- Its credentials are not exposed to normal HTTP handlers.
- The application role cannot assume it through direct or transitive membership.
- Runtime privileges cannot create or replace trusted functions, policies, schemas, tables, or roles.
- Its ownership and RLS behavior are correctly understood.

### A2. Lookup/function-owner role

Verify:

- `NOLOGIN`.
- `NOBYPASSRLS`.
- It does not own the sessions table unless that is an explicitly justified and secured design.
- It owns only the intended SECURITY DEFINER functions.
- It has only the privileges needed by those functions.
- It cannot be assumed by the application role.
- It has no unexpected membership or privilege inheritance.
- It cannot change schemas, alter protected tables, or create arbitrary executable objects.

### A3. Application role

Verify:

- `NOBYPASSRLS`.
- Does not own protected tables.
- Is not a member, directly or indirectly, of either privileged role.
- Cannot `SET ROLE` to a more privileged identity.
- Cannot read or write sessions directly.
- Cannot read operator credential records.
- Cannot disable RLS or alter policies.
- Cannot modify the SECURITY DEFINER functions or their schemas.

Query the real catalog state, including:

- `pg_roles`
- `pg_auth_members`
- `pg_class`
- `pg_namespace`
- `pg_proc`
- `pg_policy`
- function ACLs and default privileges

Do not accept a test that checks only one role attribute when transitive membership or ownership can still provide the privilege.

## 4. Critical attack surface B — SECURITY DEFINER functions

Inspect all three functions individually:

- Session lookup.
- Session creation.
- Session revocation.

For each function, record:

- Exact signature.
- Owner.
- `SECURITY DEFINER` status.
- Execution grants.
- PUBLIC privilege state.
- Function search path.
- Referenced schemas and objects.
- Inputs and outputs.
- Underlying table privileges.
- Applicable RLS policies.
- Failure behavior.

Verify:

1. Fixed SQL, no dynamic SQL.
2. Fully qualified object names.
3. Hardened `search_path`.
4. No unsafe temporary-schema object substitution.
5. PUBLIC execution revoked.
6. Execution granted only to the intended role.
7. Minimal returned data.
8. No plaintext session tokens or credentials returned.
9. No caller-controlled tenant identity accepted as authority.
10. Expired and revoked sessions are never returned as active.

### Critical test: direct session-minting attack

Do not limit the review to the normal Go login handler.

Attempt to invoke the session creation function directly using only the application database role.

Use a self-selected token hash and the information available to an ordinary application-role caller.

Determine:

- Can that caller create a valid session for a pre-provisioned operator?
- Can it choose or influence the tenant or credential binding?
- Can it obtain a usable session without successful verification of the configured operator credential?
- Does the function independently enforce the intended issuance authorization, or merely assume the Go handler has already performed it?

This distinction is fundamental.

A normal HTTP login test does not prove that the database function cannot be abused directly.

If a caller with only application-role database access can mint a usable authenticated session without the required credential verification, classify it as a potential foundational blocker.

Do not prescribe a superficial SQL check. Identify the smallest durable correction that maintains the approved three-role security model or explicitly explain why the model needs an owner decision.

## 5. Critical attack surface C — session lifecycle

Trace session creation, authentication, lookup, and revocation through the real implementation.

### Token generation and storage

Verify:

- Tokens are generated with a cryptographically secure random source.
- Entropy is sufficient for an opaque bearer credential.
- Only SHA-256 token hashes are persisted.
- Plaintext tokens never enter database records, audit events, application logs, or error responses.
- Token comparison and lookup do not expose unnecessary information.

### Authentication and fixation

Attack:

- Reused pre-login session identifiers.
- Session fixation.
- Repeated login.
- Concurrent login.
- Reuse after logout.
- Reuse after revocation.
- Expiry races.
- Forged cookies.
- Malformed and oversized cookies.
- Cookie replay from a different tenant.

Verify that successful authentication does not accidentally preserve an attacker-controlled prior session.

### Cookies and HTTP behavior

Verify:

- HttpOnly.
- Secure.
- SameSite policy.
- Path and Domain scope.
- Expiry and clearing behavior.
- No credential values in URLs.
- No open redirects through return parameters.
- Correct behavior for GET versus state-changing HTTP methods.
- CSRF protection or an explicitly justified request-origin/anti-forgery strategy for cookie-authenticated state-changing endpoints.

Do not assume that using secure cookie flags alone prevents CSRF.

### Session revocation

Verify that revocation is effective on subsequent requests, not just within the current request or cached process state.

Challenge races between authentication, expiry, and revocation.

## 6. Critical attack surface D — tenant isolation and RLS

Enumerate every Studio table from the actual migrations.

For each tenant-bearing table, verify:

- `ENABLE ROW LEVEL SECURITY`.
- `FORCE ROW LEVEL SECURITY`.
- Correct tenant policy for SELECT, INSERT, UPDATE, and DELETE.
- Correct tenant-scoped unique constraints and indexes.
- No global uniqueness rule accidentally creates cross-tenant interference.
- No missing tenant predicate in application queries.
- No policy that accidentally permits access when tenant context is NULL or missing.

Test through the actual application role:

1. No tenant context.
2. Tenant A context.
3. Tenant B context.
4. A attempting to read B's records.
5. A attempting to update B's records.
6. A attempting to delete B's records.
7. A attempting to create records owned by B.
8. Reuse of pooled database connections across A and B.
9. Transaction rollback after tenant context is established.
10. Session lookup followed by tenant-scoped database operations.

Verify that the session lookup boundary does not leave a privileged database session or connection state attached to subsequent requests.

The framework must establish transaction-local tenant context. Check that tenant identity cannot leak through pooled connections, non-transactional query paths, or session-level settings.

RLS and application ownership checks are independent layers. Prove both.

## 7. Critical attack surface E — embedded migrations and database operations

Inspect the embedded migration implementation and all four migrations.

Verify:

- Migration sources are independent of the process working directory.
- Embedded files match the expected migration sequence.
- Fresh-database bootstrap works.
- Existing-database upgrades work.
- Migration ordering is deterministic.
- Failed migrations do not leave partially trusted schema state.
- Fix-forward migrations actually correct the earlier privilege or RLS state.
- Rerunning migration commands does not corrupt the database.
- Application startup does not silently run migrations using application-role credentials.
- Migration failure prevents readiness or startup where required.

Use a disposable PostgreSQL 19 database.

Run a PostgreSQL 17 compatibility spot-check for the selected MPG target.

Do not use the owner's development database or persistent production resources.

## 8. Critical attack surface F — fail-closed server startup and configuration

Inspect the complete startup path, not just the config parser.

Challenge:

- Missing credentials.
- Empty secrets.
- Malformed configuration.
- Conflicting environment and `*_FILE` sources.
- Unreadable secret files.
- Unexpected credential combinations.
- Missing Fly runtime tokens.
- Invalid database configuration.
- Connection failures.
- Migration failures.
- Partial service initialization.
- Startup after a required dependency becomes unavailable.

Verify that the server validates required configuration and secrets before opening the listening socket.

There must be no degraded mode that silently turns an authentication or authorization failure into anonymous access.

Check that health and readiness have distinct semantics and expose no credentials, internal database details, or privileged configuration.

Inspect error and logging paths for secret disclosure, including errors produced by third-party libraries and PostgreSQL drivers.

Do not echo raw configuration values when reporting multiple configuration errors.

## 9. Critical attack surface G — framework integration

Inspect the actual server construction and database wiring.

Verify the intended sequence:

`configuration → files → policy → database pool → registry → executors → runtime.Service → HTTP server`

Check:

- Fail-closed pre-auth behavior.
- Correct use of public framework APIs.
- Correct Class A executor registration.
- No private framework imports or unsafe internal shortcuts.
- Correct transaction-local tenant context.
- Correct pool and connection budget configuration.
- Correct separation between migration and application roles.
- Health/readiness reflect the actual initialized dependencies.

The subsequent tools and screens have not yet been implemented. Do not invent their behavior or grade missing P4 work as a failure of this foundation review.

Identify integration contracts that the remaining P4 tools must preserve.

## 10. Test execution and evidence requirements

Run the applicable checks when the environment permits:

```
gofmt -l .
go vet ./...
go build ./...
go test -p 1 ./...
go test -race ./...
```

Run PostgreSQL-backed tests against disposable PostgreSQL 19.

Run PostgreSQL 17 compatibility tests where practical.

Use disposable roles and random credentials. Never introduce hardcoded database passwords into source or tests.

For every test, distinguish:

- Executed and passed.
- Executed and failed.
- Skipped.
- Blocked by environment.
- Not applicable.

Do not call a skipped test green.

Do not rely exclusively on fakes for privilege, RLS, function ownership, or session bootstrap claims. Those require actual PostgreSQL evidence.

Inspect existing tests and add no code: this is read-only review. You may run scratch probes against disposable databases outside the repository, but do not alter repository files.

## 11. Root-cause standard

For every finding:

1. Identify exact source file and line, migration, function, policy, or catalog evidence.
2. Reproduce the exploit or failure.
3. Trace the complete execution path.
4. Identify the underlying root cause.
5. Identify the violated invariant.
6. Propose the smallest durable correction.
7. Specify regression tests that fail before and pass after.

Reject symptom patches, privilege broadening, `BYPASSRLS`, direct session-table access, tenant IDs from browsers, and Studio-only workarounds for framework defects.

Do not recommend unnecessary abstractions or a redesign when a focused correction preserves the approved architecture.

## 12. Review boundaries

This review covers only the implemented P4 foundation:

- `store/`
- Embedded migrations
- Session functions and role grants
- `server/`
- `cmd/studio`
- Configuration and startup wiring
- Authentication and session middleware
- Health/readiness
- Structured errors and redacted logging

Do not review unimplemented Studio tools or six screens as if they already exist.

Do not implement anything.

Do not modify:

- `plans/plan4.md`
- `plans/plan5.1.md`
- `plans/plan6.md`
- The consolidated governing implementation plan
- Miniapp repository or its pinned SHA
- Dependencies or deployment infrastructure

Do not begin P5.

Do not enable external writes or bypass G-01.

Keep the review and all scratch testing uncommitted.

## 13. Approval criteria

The owner uses this principle:

**Approve if there are no foundational or structural blockers.**

Classify findings:

**BLOCKER**

- Application role can bypass authentication and mint a valid session.
- Cross-tenant reads or writes are possible.
- RLS can be bypassed through role membership, table ownership, missing policies, or connection-pool leakage.
- SECURITY DEFINER functions can be hijacked or used to cross privilege boundaries.
- Session revocation or expiry is not reliably enforced.
- Server can start in an unsafe unauthenticated or improperly configured state.
- Secrets leak through responses, logs, browser data, or database records.
- The actual runtime transaction boundary fails to preserve tenant identity.

**MAJOR**\
A meaningful weakness that can be contained without violating a foundational security guarantee. Must have an owner, correction, and regression test. It does not automatically stop unrelated implementation.

**MINOR**\
Maintainability, test breadth, documentation, or non-critical hardening. Record and defer where appropriate.

**IMPROVEMENT**\
Optional enhancement. Do not use it to delay implementation.

A finding is not a blocker merely because a design could be more elegant.

## 14. Required report

### A. Executive verdict

Choose exactly one:

- `APPROVE P4 FOUNDATION`
- `BLOCK P4 FOUNDATION`

Explain the decision in evidence-based terms.

### B. Security boundary diagram

Show the actual path:

`Browser → session middleware → session lookup → server-derived tenant → runtime/database context → RLS-protected store`

Show the privilege boundaries and function owners.

### C. Role and function scorecard

| Object/role/function | Actual privileges/owner | Verified invariant | Attack result | Verdict |
| -------------------- | ----------------------- | ------------------ | ------------- | ------- |

Include all three roles and all three SECURITY DEFINER functions.

### D. Findings register

For each finding:

- ID and severity.
- Exact source/migration/catalog location.
- Reproduction steps.
- Root cause.
- Impact.
- Durable correction.
- Named regression test.
- Whether it blocks P4 continuation.

### E. Test evidence

List exact commands, PostgreSQL versions, role identities, exit codes, and results.

Clearly distinguish prior reported evidence from tests independently executed during this review.

### F. Remaining P4 implementation contract

List only invariants that Studio tools, registry, and screens must preserve when implemented.

### G. Final decision

Apply the owner's approval standard.

If no foundational or structural blockers exist, approve the P4 foundation and recommend continuing immediately with Studio tools and registry.

Do not request another general architecture review.

## Final instruction

The goal is to protect the security foundation, not create an endless review process.

Be adversarial about real privilege and tenant-boundary failures. Be disciplined about evidence. Do not inflate minor defects into blockers.

A clean review should close the P4 foundation security checkpoint and let implementation proceed.

### **My particular concern**

Pay special attention to the **direct&#x20;**`session.create`**&#x20;function invocation test**. The database must not silently assume that because the normal Go login handler checks the operator credential, every caller of the function has also passed that check. That's the most important attack to rule out before the remaining P4 tools are built.

After this foundation review, the next adversarial review should be the integrated P4 review **after tools and all six screens exist**, not another review of this same foundation.