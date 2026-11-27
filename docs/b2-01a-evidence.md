# B2-01A Evidence — Sprite lifecycle contracts, local

**Status:** Implemented, unit-tested. No live Fly contact (card
prohibition honored; the suite dials only loopback httptest doubles).

## Decision brief (SKILLS §22)

| Field | Value |
|---|---|
| Objective | Local lifecycle contract: provision/wake/sleep/connect/disconnect (+ destroy, see below) with receipts, authorization, and refusals |
| Scope | `fly` package only: 3 new sources, 3 new test files. No existing file touched, no migrations, no server wiring |
| Execution location | Local Desktop |
| Application target | None |
| Database target | None (no DB code; `fly` has zero pgx/testdb/store deps) |
| Infrastructure actions | None |
| Authorization | Task-level GO (owner "go", 2026-11-26); no cloud surface exists |
| Verification | Shared contract suite × 2 backends, wire/error/desync tests, repo gates |
| Recovery | N/A (nothing provisioned) |

Destroy is included though the card names five transitions: leaving the
destructive verb outside the authorization contract would punch an
ungated hole through it, and B2-01B's recovery already requires
destroy-scoped authorization. No other scope was added.

## Files

- `fly/lifecycle.go` — Transition/State, Authorizer + DecisionAuthorizer,
  RefusalError (12 named reasons), UnsupportedError, LifecycleReceipt,
  ReceiptRecorder/MemoryRecorder, LifecycleProvider, Clock, Lifecycle
  controller (authorize → replay → state-check → effect → receipt),
  Reconcile.
- `fly/lifecycle_fake.go` — FakeLifecycleProvider: deterministic,
  full-verb, self-guarding, call-counted, Fail-injectable.
- `fly/lifecycle_adapter.go` — AdapterLifecycleProvider over
  SpriteAdapter: provision/destroy/describe are real SDK calls;
  SetPower/Attach/Detach refuse with *UnsupportedError.
- `fly/lifecycle_contract_test.go` — shared suite (25 subtests: happy
  paths, 8 auth refusals, replay × 3, concurrency × 2, determinism,
  reconcile × 3, power/session cycles or unsupported-refusal).
- `fly/lifecycle_parity_test.go` — scripted Sprites API double (Bearer
  asserted, never logged), adapter suite run, exact wire-log, error
  mapping, outage-vs-absence, desync + reconcile.
- `fly/lifecycle_provider_test.go` — fake guards, Fail injection,
  constructor panics, gate-malfunction fail-closed, fake desync.

## Gate results (exact commands, exit 0 throughout)

- `gofmt -l .` → clean
- `go vet ./...` → clean
- `go build ./...` → clean
- `go test -p 1 ./...` → every package ok; `fly` 0 skips
- `go test -race ./fly/ -count=1` → ok
- `go test -race -p 1 ./gateway ./store ./tools ./server ./cmd/studio ./cmd/gateway` → ok
- (All with `GOCACHE=/tmp/gocache`.)

## Material finding for B2-01B

sprites-go v0.2.2 exposes **no** power or session-attach primitives
(verified in module source; no wake/sleep/suspend/resume/connect API,
and UpdateSpriteRequest carries no power field). The adapter therefore
cannot implement wake/sleep/connect/disconnect without inventing
endpoints, which the constitution forbids. B2-01B must discover the
real mechanism live (candidate: auto-sleep/warm-cold + exec-based
reachability — LastRunningAt/LastWarmingAt exist but their vocabulary
is unverified). The contract side is ready: state machine, receipts
(incl. per-transition Latency for wake-latency notes), authorization,
and the already-written (fake-green) power/session contract tests.

## Honesty notes

- Evidence tier: **unit-tested**. The double proves adapter/fake
  contract parity, not Fly behavior. Live proof is B2-01B's gate.
- Controller state is in-memory; restart drops Studio-side position.
  Reconcile re-derives absent-vs-present from the provider only.
  Durable lifecycle position/receipts are B2-01B's scope per its card
  ("DB: local PG (lifecycle receipts)"); no schema was added here.
- DB-backed gates blocked (environmental, pre-existing): PG19 at
  127.0.0.1:5434 refused connections (Docker socket permission-denied
  in this shell; per P0 precedent the owner starts it unsandboxed).
  275 suite tests skipped on unset TEST_DATABASE_URL; none are B2-01A
  tests. Re-run `go test -p 1 ./...` with the cluster up to clear.
