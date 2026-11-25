The next work order should complete **P4 fix-forward corrections first, then Studio tools and registry, then the six screens**. It must explicitly prevent the four MAJOR findings from being forgotten or accidentally worked around.

I’ve incorporated the adversarial report's findings, including the F2 prerequisite for Service invocation, F3 composite foreign keys, F7 pool reset, and the F1/F4 restrictions.&#x20;

adv_review_prompt5(1).md

adv_review_prompt5(1).md



# Miniapp Studio — P4 Continuation and Completion Work Order

## Role

Act as the principal Go architect, PostgreSQL security engineer, and implementation lead for Miniapp Studio (`github.com/PithomLabs/miniapp-studio`).

P0, P1, P2, P3, and the P4 server/security foundation are complete and accepted.

Your assignment is to:

1. Correct the three immediate P4 foundation findings F2, F3, and F7.
2. Implement Studio tools and static registry through `runtime.Service`.
3. Complete the six Studio screens over real, verified backend handlers.
4. Run the complete P4 acceptance and security test suite.
5. Deliver a final P4 implementation report.

This is an implementation work order, not another architecture-review exercise.

**Governing principle:** investigate mechanisms and solve root causes. Do not patch symptoms, bypass authorization, weaken RLS, or introduce Studio-only workarounds for framework defects.

## 1. Authoritative inputs and accepted state

Read:

- `plans/miniapp-studio-implementation-plan-consolidated.md` — governing plan, Revision 2.
- `plans/adv_review_prompt5(1).md` — P4 foundation adversarial review and findings F1–F13.
- `docs/p0-evidence.md` — completed P0 evidence.
- `docs/p3-evidence.md` — completed P3 evidence.
- `docs/architecture/decisions/001-url-topology.md`.
- `docs/architecture/decisions/002-postgres.md`.
- All existing source under `spec/`, `runner/`, `render/`, `connector/`, `fly/`, `apps/`, `store/`, `server/`, and `cmd/studio`.

Frozen documents must remain untouched:

- `plans/plan4.md`
- `plans/plan5.1.md`
- `plans/plan6.md`

### Accepted foundation

- P0 passed on framework `73351a8`, with PostgreSQL 19, required restart tests executed, race tests, and Class A invocation/approval-resumption spike.
- P1 MiniappSpec and validator accepted.
- P2 generic runner, interpreter, renderer, and fixture dry-run accepted.
- P3 Sprite-per-app topology proven live.
- MPG Developer on PostgreSQL 17 is the selected database target, subject to its deployment-time smoke test.
- P4 server, sessions, three-role database model, and embedded migrations implemented and adversarially reviewed.
- P4 foundation verdict: `APPROVE P4 FOUNDATION`.

All work must remain uncommitted. Do not modify the Miniapp framework unless a concrete §14 finding establishes that a general-purpose framework correction is necessary and its approval requirements are satisfied.

## 2. Findings and mandatory dispositions

### Immediate correction batch

**F2 — Framework schema grants are ineffective**

The migration grants table privileges on framework `app.*` objects but omitted `USAGE ON SCHEMA app`.

This causes Service invocations and framework-owned persistence operations through the application pool to fail with SQLSTATE `42501`.

Required correction:

- Add a forward-only migration, named consistently with the current migration sequence (the review proposed `000005_app_usage`).
- Grant `USAGE ON SCHEMA app` to `studio_app`.
- Do not change unrelated framework grants.
- Do not use `SET ROLE`, a second privileged application pool, or direct access to private framework internals.
- Prove framework RLS works through the actual executor-owned connections.

Required regression tests:

1. `TestFrameworkTablesReachableAsApp`: application role can access the specifically granted framework tables without `42501`.
2. `TestServiceInvocationTenantScoped`: a Class A Service invocation inside the supported tenant transaction establishes correctly tenant-scoped framework rows.
3. A bare Service invocation without tenant context fails closed.
4. Cross-tenant invocation and persistence attempts remain denied.

Important: inspect the real Miniapp executor and transaction implementation. Do not assume an outer Studio transaction automatically propagates tenant context to another connection. Prove the actual connection and transaction choreography.

If the framework executor opens its own connection without inheriting tenant context, classify the root cause under governing plan §14. Do not bypass the framework's tenant isolation.

**F3 — Cross-tenant foreign-key references**

The current tenant tables use globally unique app IDs for references, but the foreign keys do not require tenant equality.

A child row belonging to tenant A can reference an app belonging to tenant B. Deleting the parent can cascade into another tenant's rows.

Required correction:

- Add `UNIQUE (tenant_id, id)` to `studio.apps` where necessary.
- Replace app-reference foreign keys in `app_specs`, `deployments`, and `runs` with composite foreign keys:

  `(tenant_id, app_id) REFERENCES studio.apps(tenant_id, id)`
- Preserve intended same-tenant cascading behavior.
- Inspect all other foreign keys for the same tenant-integrity defect.
- Before migration, query for existing cross-tenant references.
- If any exist, fail the migration with a clear, non-sensitive error. Never silently reassign or delete data.
- Use forward-only migrations; do not rewrite applied migration history.

Required regression tests:

1. `TestCrossTenantReferenceRejected`: inserting a tenant-A child referencing a tenant-B app fails with a foreign-key violation.
2. `TestSameTenantReferenceAccepted`: same-tenant references continue working.
3. `TestCascadeStaysInTenant`: deleting tenant B's app cannot delete tenant A's child rows.
4. Migration tests prove existing invalid data causes a safe migration refusal.

**F7 — Connection pool preserves session-level tenant settings**

The adversarial test demonstrated that `SET app.tenant_id` on a pooled connection survives release and can affect later checkouts.

Required correction:

- Implement a verified connection-reset strategy for `app.tenant_id` on the actual pinned pgxpool version.
- Use the correct lifecycle hook or supported connection wrapper.
- Inspect the exact callback semantics, context handling, connection disposal, and failure behavior before selecting the mechanism.
- If reset fails, do not return a contaminated connection to the pool.
- Apply the protection to both server and relevant test/store pools.
- Document `SET LOCAL` inside transaction-scoped tenant helpers as the only supported way to establish tenant context.
- Never use session-level `SET` in production code.
- Do not issue expensive `DISCARD ALL` as a substitute for a narrowly scoped reset.

Required regression test:

`TestPoolResetsTenantSetting`:

1. Acquire an application connection.
2. Set a session-level `app.tenant_id`.
3. Release the connection.
4. Acquire a connection again.
5. Prove an unscoped tenant-table read sees no rows.
6. Prove `InTenantTx` still works correctly and does not leak tenant context after transaction completion.

Also test reset failure behavior: a connection that cannot be sanitized must be discarded, not reused.

### Deferred MAJOR findings — explicit release boundaries

**F1 — Direct session creation without credential proof**

The app role can call `session_create` directly with a self-chosen token hash and a known provisioned tenant/binding pair, minting an HTTP-usable session without operator credential verification.

This finding is accepted as MAJOR for the current foundation review, not resolved.

Required treatment:

- Do not claim the database function independently authorizes credential issuance.
- Record the app DSN trust assumption and the exact capability it grants.
- Prepare an owner decision for a durable proof-of-issuance mechanism.
- Evaluate both proposed approaches:
  1. HMAC proof bound to tenant, binding, token hash, and expiry, verified with a key unavailable to the application role.
  2. DB-held credential verification within the SECURITY DEFINER boundary.
- Compare key management, rotation, threat model, and operational trade-offs.
- Do not select or implement a new issuance design without the required owner decision.
- Do not revoke the application's legitimate session creation path without an authorized replacement.

**Release restriction:** F1 must be resolved or explicitly risk-accepted with an owner decision before tools that deploy, spend, or perform irreversible external actions ship.

No external effects are authorized in this work order.

**F4 — Operator deactivation leaves live sessions valid**

Treat this as a production-offboarding gap.

- Document the current maximum exposure window (session TTL).
- Propose a revoke-by-operator/binding mechanism owned by the correct privileged boundary.
- Define deactivation and reactivation semantics.
- Preserve old-token invalidation after reactivation.
- Name the regression test `TestDeactivateEndsSessions`.

Do not add multi-operator administration or signup flows.

Before multi-operator production use, F4 must be fixed. Do not silently represent current deactivation as immediate session revocation.

## 3. Phase A — Correct F2, F3, F7

Implement the three corrections before building Service-invoking Studio tools.

For each finding:

1. Reproduce the issue on a disposable database using the current implementation.
2. Capture the failing behavior.
3. Identify the exact root cause.
4. Implement the smallest durable forward correction.
5. Add the named regression test.
6. Run the test before and after the fix, demonstrating failure before and success after.
7. Run the relevant full store/server and framework integration suites.
8. Record the migration and test evidence.

Do not use stash-based or destructive repository manipulation to simulate before/after states. Preserve the user's working tree and existing uncommitted P4 work. Use a safe disposable worktree, isolated copy, or scratch database when needed.

### Phase A exit

F2, F3, and F7 are complete only when:

- Framework tables are reachable using the intended app privileges.
- Class A framework operations remain tenant-scoped through actual executor connections.
- Cross-tenant foreign keys are rejected.
- Same-tenant cascades remain correct.
- Pooled tenant settings cannot leak between requests.
- All named tests execute and pass.

Do not begin the first Service-invoking Studio tool before F2 is corrected and its real Class A tenant-context integration test passes.

## 4. Phase B — Studio tools and static registry

Implement Studio tools as statically registered Miniapps.

Every consequential operation must pass through the framework's canonical `runtime.Service.Invoke` path.

Do not call tool handlers directly from HTTP routes.

Use the supported public Miniapp APIs at the approved SHA. Do not invent methods or use private internals.

### Required tool families

**A. `spec.validate`**

- Validate submitted MiniappSpec.
- Return deterministic validation results and actionable errors.
- Reject invalid specs.
- Bind accepted spec versions to their canonical content hash.
- Derive and verify ownership against the authenticated tenant.
- Never treat a populated spec returned alongside validation errors as usable.

**B. Registry tools**

Implement tenant-scoped application and specification management:

- Create/register an app.
- List apps.
- Read app details.
- Update app metadata within permitted rules.
- Create and retrieve immutable spec versions.
- Retrieve the current version and content hash.
- Validate app/spec ownership on every operation.

Preserve tenant-scoped unique constraints and composite foreign keys.

Use explicit ownership checks in addition to RLS.

Never accept the tenant identifier from browser arguments as authority.

**C. Deployment tools**

Implement the Studio deployment lifecycle through registered tools:

- Validate deployment eligibility.
- Create or update deployment records.
- Track lifecycle state and actual evidence.
- Read deployment status.
- Preserve the stable Sprite URL recorded by P3.
- Reconcile known failures accurately.

For P4:

- Use injected Fly/Sprite fakes.
- Test success, failure, timeout, and unknown outcomes.
- Do not provision real Sprites.
- Do not deploy live apps.
- Do not mutate real URLs or resources.
- Report missing production credentials or an unconfigured deployment manager honestly.

Never fabricate a successful deployment.

**D. Connector tools**

Implement connector operations through the existing typed client and `ConnectorManager` abstraction.

Support only the operations authorized by the current P4 design.

Requirements:

- Discover configured connectors from the actual injected interface.
- Distinguish listed, configured, unavailable, failed, and unknown states.
- Keep secrets server-side.
- Reject unknown connection IDs.
- Enforce connector authorization and declared capabilities.
- Exercise client request/response behavior using fakes and HTTP test transports.
- Do not perform live connector mutations during P4.
- Report production configuration honestly.

**E. `activity.list`**

Expose actual activity records through a registered Class A tool.

Requirements:

- Retrieve real framework audit events.
- Preserve tenant-scoped access.
- Enforce pagination and bounded query parameters.
- Return honest empty states.
- Redact sensitive information.
- Never manufacture synthetic audit activity.

Inspect the actual audit schema and framework RLS policy before implementing the query.

### Registry and invocation requirements

- Static `registry.Register`.
- Explicit descriptors and bounded input/output schemas.
- Canonical `runtime.Service.Invoke`.
- Tenant derived from authenticated server context.
- Framework authorization and audit preserved.
- Idempotency keys and invocation semantics preserved.
- Tool execution must reject missing tenant context.
- No direct handler bypass or alternate privileged execution path.

### Required tool tests

For every tool:

- Positive same-tenant operation.
- Missing tenant context.
- Cross-tenant read.
- Cross-tenant write.
- Wrong owner.
- Malformed arguments.
- Unauthorized operation.
- Idempotent duplicate submission.
- Invocation/audit evidence.
- Secret absence from responses and logs.

Test at both application ownership and PostgreSQL RLS layers.

## 5. Phase C — Six Studio screens

Implement the six screens over the completed, verified backend routes and renderer.

Do not add fabricated data to fill missing backend capabilities.

Every screen must have four states:

- Loading.
- Empty.
- Error.
- Success.

### Screen 1 — Home

- Real tenant-owned application list.
- Recent real activity.
- Accurate application and deployment status.
- Create-app entry point.
- No fabricated metrics or sample activity in production.

### Screen 2 — Create App

- Conversational-builder entry point reserved for P5.
- Existing specification editor/inspector where supported.
- Real validation results through `spec.validate`.
- Fixture-backed preview using dry-run.
- Display the exact content hash and spec version.
- Do not pretend a builder agent is implemented.

### Screen 3 — Connectors

- Real connector discovery through the typed interface.
- Listed/configured/unavailable/unknown states.
- Custom API wizard against the real client interface.
- Safe secret-entry handling.
- Scope and permission explanation.
- No credentials in HTML, browser responses, logs, or specs.
- No live connector mutations or tests with real external credentials during P4.

### Screen 4 — Validate & Deploy

- Real validation results.
- Accurate deployment eligibility.
- Honest unconfigured production state.
- Injected-fake deployment behavior in tests.
- No live deployment action.
- No action enabled unless its backend route and authorization exist.

### Screen 5 — My Apps

- Real tenant-owned application records.
- Spec version and hash.
- Actual deployment records and URLs.
- Accurate lifecycle state.
- No fabricated app records or URLs.
- Cross-tenant access denied.

### Screen 6 — Activity

- Actual audit events via `activity.list`.
- Tenant-scoped filtering and pagination.
- Honest empty and error states.
- Redacted details.
- No synthetic timeline entries.

### UI and HTTP security

- Escape user-controlled content.
- Keep credentials out of browser-readable state.
- Protect state-changing cookie-authenticated routes with a documented CSRF strategy (Origin validation and/or anti-forgery tokens).
- Preserve secure cookie settings.
- Validate redirect targets using a shared safe helper.
- Do not enable controls without a fully implemented backend route and authorization check.
- Use appropriate status codes and structured errors.
- Do not expose SQL errors, stack traces, tokens, database connection strings, or provider secrets.

## 6. Phase D — Authentication and lifecycle hardening

Resolve the applicable contained findings from the adversarial report as part of P4 where scope permits:

- F5: deactivated-but-valid login must not produce a distinct credential oracle response; map issuance refusal to the normal invalid-credentials response while preserving server-side diagnostic classification.
- F6: reject control characters in redirect targets through a shared safe helper.
- F8: trim and validate Sprite token-file content consistently, without exposing its value.
- F9: make whoami session identity consistent or omit the field.
- F10: reject duplicate login-token hashes in operator configuration.
- F11: remove dead code or fix dangling documentation and keep test-only session helpers out of production paths.
- F12: readiness must verify the expected embedded migration version and report 503 when missing, dirty, or incompatible.
- F13: add the highest-value missing CLI/provisioning and error-redaction tests where practical.

Do not let optional improvements expand the scope. Apply the approval rule: correct real defects, defer nonessential polish.

F4's revoke-by-operator requirement remains a production-offboarding prerequisite; do not add multi-operator administration.

## 7. Tenant isolation and security invariants

These are mandatory across every tool and screen:

1. Tenant identity comes exclusively from the validated server-side session.
2. Browser headers, query parameters, route values, and request bodies cannot select the effective tenant.
3. Every persistent record belongs to one tenant.
4. All tenant-scoped database operations use transaction-local tenant context through the supported helper.
5. Application ownership checks supplement RLS; neither replaces the other.
6. Composite foreign keys prevent cross-tenant references.
7. Pool cleanup prevents session-level tenant state leakage.
8. Session lookup errors fail closed as unauthenticated.
9. Revoked and expired sessions are ineffective.
10. Framework invocation and audit preserve the authenticated tenant.
11. No secret reaches browser responses, Sprite state, MiniappSpec, logs, or activity output.
12. No unconfigured Fly or connector capability is presented as operational.

## 8. External-effect and deferred-work boundary

P4 is not authorized to implement external write support.

Preserve:

- G-01 cross-invocation effect ownership transfer remains unresolved.
- F1 session-issuance proof remains unresolved pending owner design.
- F4 live-session invalidation on operator deactivation remains unresolved unless explicitly implemented and tested within the authorized scope.
- No live Sprite provisioning, external deployment, connector creation/update, URL mutation, or provider writes.
- Fly/Sprite deployment tools use injected fakes.
- Production-unconfigured paths return explicit, truthful status.

Before any later external or irreversible action is authorized, F1 and G-01 must be addressed according to their required approval and test criteria.

P5 builder-agent integration and the Invoice Review Desk golden path are out of scope.

## 9. Validation and acceptance tests

Use disposable PostgreSQL databases and roles.

Required primary validation:

```
gofmt -l .
go vet ./...
go build ./...
go test -p 1 ./...
go test -race -p 1 ./...
```

Run the complete PostgreSQL-backed P4 test suite against local PostgreSQL 19.

Run PostgreSQL 17 compatibility tests for the selected MPG target where the authorized environment permits.

No database tests may be marked green because they silently skipped.

Record exact server versions, commands, test counts, failures, and skips.

### A. Database tests

- F2 framework schema grants and Service invocation.
- F3 composite FK constraints and tenant-safe cascades.
- F7 pool cleanup and tenant-local context.
- RLS enabled and forced on all Studio tenant tables.
- App and lookup role privilege boundaries remain correct.
- Session direct-read denial remains `42501`.
- Role membership and ownership remain unchanged.
- All six tenant-table isolation cases continue passing.

### B. Tool tests

For each registered Studio tool:

- Same-tenant success.
- Cross-tenant denial.
- No-context refusal.
- Correct invocation and audit record.
- Malformed input rejection.
- Authorization denial.
- Bounded pagination/results.
- Secret-redaction assertions.

### C. Runtime tests

- Invoke is the only consequential execution entry point.
- No direct handler call from HTTP routes.
- Class A execution and terminal replay.
- Tenant context reaches executor-owned connections.
- Cancellation and authorization remain enforced.
- Audit events are produced and tenant-scoped.

### D. UI tests

- All six screens render all four states.
- No fabricated records or operational statuses.
- Unavailable backend states are honest.
- Escaping and response redaction.
- State-changing actions have CSRF protection.
- Cross-tenant rendering and resource access denied.

### E. Failure and security tests

- Database unavailable.
- Migration missing or dirty.
- Invalid session.
- Revoked session.
- Expired session.
- Tenant context missing.
- Pool connection contaminated.
- Fly manager unavailable.
- Connector manager unavailable.
- Deployment timeout/unknown state through fakes.
- Duplicate invocation.
- Cross-tenant CRUD and foreign-key attempts.
- Secret-bearing errors and response paths.

Every acceptance requirement must map to an executed test or an explicitly recorded environmental blocker.

## 10. Scope and change-control restrictions

Do not:

- Start P5.
- Implement a builder agent.
- Implement the Invoice Review Desk end-to-end journey.
- Provision live Sprites or deploy live generated applications.
- Perform live external connector mutations.
- Enable external writes.
- Bypass G-01.
- Introduce DBOS or background execution.
- Add signup, invitation, multi-operator UI, or complex RBAC.
- Add marketplace, billing, workflow canvas, or multi-tenant administration.
- Change the approved topology or database decision without contrary implementation evidence and owner decision.
- Modify frozen planning documents.
- Modify Miniapp casually.

If an actual Miniapp defect is discovered, use governing plan §14:

1. Reproduce the defect.
2. Explain why the capability belongs in the general framework.
3. Identify the violated invariant.
4. Propose the smallest durable framework correction.
5. Name tests that fail before and pass after.
6. Obtain required approval for foundational changes.
7. Keep framework and Studio changes separately reviewable.

No framework workaround inside Studio.

Everything remains uncommitted.

## 11. Required deliverables

### Database and security corrections

- Forward-only migrations for F2 and F3.
- Pool hygiene correction for F7.
- All named regression tests.
- Updated migration documentation and evidence.

### Studio tools

- Static tool registrations.
- `spec.validate`.
- Tenant-scoped registry CRUD.
- Deployment lifecycle tools using fakes.
- Connector tools using typed interfaces and fakes.
- `activity.list` over actual framework audit events.
- Invocation, authorization, and tenant-isolation tests.

### Studio experience

- Six operational screens.
- Real backend handlers.
- Loading/empty/error/success behavior.
- No fabricated data.
- Security and secret-redaction tests.

### Final report

Provide:

1. F2/F3/F7 root-cause corrections and regression evidence.
2. Disposition of F1 and F4, including owner decision status and release restrictions.
3. Migrations added and their safety properties.
4. Role/RLS verification.
5. Studio tools and their registered descriptors.
6. Runtime Service invocation traces.
7. Six-screen completion matrix.
8. Tenant-isolation evidence.
9. CSRF, session, and secret-handling results.
10. Exact test commands, PostgreSQL versions, counts, skips, and exit codes.
11. Known limitations and explicitly deferred P5 work.
12. Any §14 framework findings.
13. Confirmation that frozen documents are untouched, Miniapp remains untouched unless approved §14 correction is necessary, and all work is uncommitted.

## 12. Completion and approval standard

The owner has established:

**Approve if there are no foundational or structural blockers.**

Apply this strictly:

- Foundational privilege, tenant-isolation, or secret-boundary defects block the affected work.
- MAJOR but containable issues receive a named owner, correction, regression test, and explicit release restriction.
- MINOR findings are documented and deferred when safe.
- Optional improvements must not hold up implementation.

Do not request another general architecture review during implementation.

The next adversarial review is the integrated P4 review after the tools and all six screens exist.

**Final instruction:** correct F2, F3, and F7 first. Then implement Studio tools and registry exclusively through `runtime.Service`. Complete the six screens using real handlers. Preserve the read-only boundary, tenant isolation, one-operator MVP, no-P5 scope, exact framework pin, frozen plans, and no-commit rule.

**Execution priority:** F2, F3 and F7 first; then tools and registry; then six screens. F1 must be resolved or formally risk-accepted before any future external-effect capability ships.