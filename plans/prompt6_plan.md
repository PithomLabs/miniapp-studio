## Goal

Implement [prompt6.md](/home/chaschel/Documents/go/miniapp-studio/plans/prompt6.md): Phase A (F2/F3/F7 root-cause corrections with named regression tests), Phase B (Studio tools + static registry exclusively through `runtime.Service`), Phase C (six screens over real handlers), Phase D (contained F5/F6/F8–F13 fixes), with F1/F4 carried as documented release boundaries. No P5, no external writes, uncommitted, Miniapp untouched absent §14 evidence.

## Success Criteria

- F2: `USAGE ON SCHEMA app` fix-forward (`000005`); framework tables reachable as app role; Class A invocations tenant-scoped through executor-owned connections (verified, not assumed); bare invocations fail closed.
- F3: composite `(tenant_id, app_id)` FKs with same-tenant cascades; pre-migration data check refuses on existing violations; all four named tests green.
- F7: verified reset on the pinned pgx v5.13.0 (`AfterRelease` vs `PrepareConn` decided empirically; deprecated `BeforeAcquire` excluded); `SET LOCAL` documented as the only tenant mechanism; failure discards, never reuses.
- Tools: single `studio` miniapp (v1.0.0) with `spec.validate`, registry CRUD, deploy lifecycle (fakes), connectors (fakes/`httptest`), `activity.list` over real audit events — every tool with the ten-case matrix at app + RLS layers.
- Screens: all six render loading/empty/error/success with zero fabricated data; CSRF via strict Origin validation plus `SameSite=Lax`; secrets absent from responses/logs.
- Full gates green on PG19 with a PG17 spot-check; every prompt6 §9 bullet mapped to an executed test or a recorded blocker; skips explicitly counted (target: zero with DB present).

## Approach

Evidence already shapes the hard parts: `classa.go:51` arms tenant context inside executor-owned transactions from the server-resolved `req.TenantID`, so Phase A proves rather than assumes the choreography; pgx v5.13.0 offers `AfterRelease` (sync, destroy-on-false) and `PrepareConn` ( richer acquire-time semantics) — the plan tests both and picks on behavior. Audit `INSERT` under the `USING`-only policy stays an explicit Phase A branch: the F2 probe was blocked one layer early by the missing `USAGE`, so post-fix behavior gets verified live, with a `WITH CHECK` gap classified honestly if it appears. Fail-before evidence uses tests-first sequencing (write named test → run red → fix → run green), which satisfies the no-stash/no-destructive constraint without worktrees. Reversible defaults, stated plainly: single `studio` miniapp over per-family miniapps; populate (not omit) `whoami.session` via migration; Origin-check CSRF; deterministic content-hash idempotency keys for writes, random per request for reads.

## Steps

1. Phase A, per finding in F2/F3/F7 order: reproduce on a disposable scratch database, capture, root-cause, forward-only fix, named tests red-then-green, suite re-run, evidence record.
2. Phase B: register `studio` tools; implement the five families with ownership checks + RLS; ten-case matrix per tool; fakes/`httptest` for Fly/connector paths.
3. Phase C: six screens on verified handlers with the four-state contract; connector wizard and deploy flows honest about unconfigured production.
4. Phase D + F1/F4 packages: contained fixes, owner-decision comparison for F1 (HMAC proof vs DB-held verification, no implementation), F4 offboarding gap record, CSRF/secret audits.
5. Full validation (`gofmt`, `vet`, `build`, `test -p 1`, `race -p 1`, PG19 + PG17), traceability matrix, final §11 report.

## Validation Plan

- Exact prompt6 §9 commands; DB suites against local PG19 with a PG17 compat pass; every named test (`TestFrameworkTablesReachableAsApp`, `TestServiceInvocationTenantScoped`, `TestCrossTenantReferenceRejected`, `TestSameTenantReferenceAccepted`, `TestCascadeStaysInTenant`, `TestPoolResetsTenantSetting`, plus F5/F6/F8–F10/F12/F13 tests) executed and passing; skip count reported, never silent.
- Tenant/session attack matrices across every new tool and route; secret scans over responses, logs, and error paths; frozen docs, pin, G-01, and no-commit rule re-verified at the end.

## Risks / Open Questions

- If audit `INSERT` is denied post-`USAGE` (USING-only policy gap), I'll classify it as a framework/migration finding with a concrete proposal rather than working around it — flagging now so it isn't a surprise.
- F1 needs your design decision after I deliver the comparison: prefer it as a file under `plans/` or inline in the final report?
- The three stated defaults (single miniapp, populate session id, Origin-check CSRF) stand unless you redirect — say the word if any should flip before I start.
