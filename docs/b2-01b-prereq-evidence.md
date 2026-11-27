# B2-01B Prerequisites Evidence — receipt store + Tasks-API bindings

**Status:** Local prerequisites implemented and tested. Live proof
NOT started: blocked on owner-terminal `sprite login` (see below)
and the per-operation GOs. No live contact was made.

## What this turn built (all local, no cloud surface)

- `store/migrations/000025_sprite_lifecycle.up.sql` (+ down):
  `studio.sprite_lifecycle_receipts`, composite PK (tenant_id, id),
  transition/state CHECKs mirroring the fly contract, latest-position
  index, tenant RLS (ENABLE+FORCE), grants to studio_app. No FKs:
  receipts are build-less by design.
- `testdb/testdb.go` + `store/privilege_parity_test.go`: new table
  added to both privilege-mirror lists.
- `store/lifecycle_receipts.go`: Save (23505 → ErrReceiptExists),
  ByKey (→ ErrNoReceipt), LatestPosition (restart-recovery
  position). All three run inside InTenantTx (RLS applies to reads).
- `store/lifecycle_receipts_test.go`: round-trip, duplicate-key
  (incl. cross-tenant independence), missing-key, latest-position
  ordering, tenant isolation + unarmed-zero, CHECK refusals.
- `fly/lifecycle_adapter.go`: Attach/Detach now execute the
  vendor-documented Tasks-API curl argv over exec (hold
  `studio-mgmt`, expiry `30m`, `-f` fail-loud, bounded error
  output). SetPower still refuses — pause is automatic, not an API.
- Contract suite: new `sessionLiveOnly` capability — the adapter
  run pins the wiring attempt (transport must fail, gate must
  pass, never Unsupported) instead of the outcome; argv pinned
  exactly by TestTasksHoldArgs.

## Gate results

- `gofmt -l .`, `go vet ./...`, `go build ./...` → clean.
- `go test -p 1 ./...` → exit 0, every package ok.
- `go test -race` on store/tools/server/cmd/fly → ok.
- New store tests compile and SKIP honestly (PG19 down; migration
  SQL reviewed against 000013/000024 patterns but unexecuted —
  runs at the live turn).
- `fly` suite (incl. new SessionLiveOnly + argv tests) green
  with and without `-race`.

## Blockers (exact)

1. `sprite list` fails: "no token found for organization
   isagani-mendoza … Run 'sprite login' to authenticate". The CLI
   already targets the D1 org; only owner-terminal login is
   missing. Next live step after login: task-level vocabulary
   reads per plans/plan1.md step 1.
2. Per-operation owner GOs still required: provision, connect,
   disconnect, wake, and separately-authorized destroy.

## Pre-existing failure (not a regression, not fixed)

`gateway TestAdapterStatus` fails under `-race` (TempDir cleanup:
sessions dir still being written by a background pump after Cancel)
but passes without it. Proven pre-existing: gateway references
none of the new API and imports neither store nor testdb; my only
modified files are those two lists; and the test fails identically
on a clean HEAD worktree. Timing-sensitive (it passed the B2-01A
gate run under lower load). Left untouched per scope discipline —
fixing it needs its own authorization.
