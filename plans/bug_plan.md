## Goal

End the circling with one concrete decisive experiment: a minimal, fast, bounded live test that reproduces
the exact incident sequence (full Up → Down to 15 → Up replay on a scratch Neon database) and either proves
the migrate stall fixed or captures a phase-attributed failure in minutes — not tens of silent minutes.

## Success Criteria

- One new test, `TestMigrationDownUpBounded` (store package), executes exactly the incident sequence and nothing
  else: no RLS assertions, no 300-test baggage.
- Owner-terminal run completes (pass or fail) with the test binary bounded by `-timeout 5m`, streaming
  progress markers — silence is never again mistaken for a stall. Total wall clock is measured via `time`
  and reported alongside, not capped.
- Pass closes the incident mechanism: bounded migrate lifecycle proven on live Neon (dirty protocol, advisory
  lock, version marks all exercised by the down/up replay).
- Fail arrives as evidence, not mystery: phase trail naming the exact operation, version/dirty state, and a
  goroutine dump locating the stall — sufficient to fix from, with no new theorizing required.
- Full-suite duration, if still slow afterward, is handled as a separate performance/observability matter with
  per-package timing — never again conflated with "the migrate hang".

## Approach

Two confounded problems masqueraded as one hang: (1) a possibly-real migrate stall, (2) guaranteed multi-minute
silence (`go test` prints per package only; first package compiles + migrates + tests before a single line).
The plan separates them with finality: a minimal repro isolates (1) with a 5-minute cap and streaming markers,
so every outcome — pass, bounded failure, or timeout panic — is decisive. No migrate-path code changes ship in
this slice unless the test fails; the test adjudicates first.

Key decisions:

- **D1 — Minimal repro, exact sequence.** Full Up → Down to v15 → Up replay on a fresh scratch DB, reusing
  `scratchDB`, `testdb.FrameworkBaseline`, `Migrate`, `migrateTo` — the identical calls the hanging test made.
  Rejected: full-suite-as-diagnostic (slow, noisy, inconclusive) and more unit tests (cannot touch server behavior).
- **D2 — Tight timeout, streaming output, preserved exit code.** Owner runs (verified end to end: task
  `--` passes through, `dev test` stops flag parsing at its own `--`, remainder reaches `go test` — proven
  by a `-- --help` probe printing go-test usage):
  `set -o pipefail; time task dev:test -- -- -run TestMigrationDownUpBounded -timeout 5m -v 2>&1 | tee test-run-cycle.log`
  (file→env bridging already built; no transcription). Without the inner `--`, `dev test`'s FlagSet would
  reject `-timeout`/`-v` as undefined flags — the separator is load-bearing. `pipefail` keeps a test failure
  from hiding behind tee's zero exit; `time` records total wall clock separately from go test's per-package
  durations. `-timeout 5m` converts any stall into a 5-minute stack dump; `-v` plus `t.Log` markers at each
  phase (up-start/up-done/down-start/down-done with versions) make progress visible continuously.
- **D3 — Version/dirty assertions at each leg.** After full Up: v25 clean. After Down: v15 clean. After replay:
  v25 clean. Any deviation fails with the recorded state attached — reconciling version truth without new surface.
- **D4 — Branch on outcome, in advance.** Pass → incident closed on mechanism; remaining work is suite-duration
  observability only. Fail → fix strictly from the captured trail + state + stack; no new hypotheses without
  new evidence. Timeout panic → the dump names the precise stall point; treat it as the new primary exhibit.
- **D5 — No scope bleed.** RLS assertions stay in their existing test; the adapter, migrations, and Taskfile are
  untouched by this slice unless the test's evidence demands a specific change.

## Steps

1. **Write the test** (`store/migrate_cycle_test.go`): scratch DB → framework baseline → `Migrate` full Up →
   assert v25 clean → `migrateTo` v15 → assert v15 clean → `Migrate` replay → assert v25 clean, with `t.Log`
   progress markers at each boundary. Skips honestly without binding (via existing helpers). Repo gates green.
2. **Owner live run** (exact block, evidence to verified-gitignored file):
   ```
   set -o pipefail
   time task dev:test -- -- -run TestMigrationDownUpBounded -timeout 5m -v 2>&1 | tee test-run-cycle.log
   ```
   then "done". `test-run-cycle.log` matches the committed `/test-run*.log` ignore rule (verified).
3. **Adjudicate from evidence.** Pass: record versions/dirty/counts/timings, close the incident, file the
   suite-duration follow-up separately. Fail/panic: fix from the trail + state + stack only.
4. **Contingency (only if the test passes but the full suite still stalls):** the hang was never in migrate —
   rerun the suite with `-v`, identify the slow/stuck package from timestamps, and open a test-phase-only
   investigation with the same bounded-evidence discipline.

## Validation Plan

- Repo gates: `gofmt`, `go vet`, full unit suite counts; new test listed as SKIP without binding (honest-skip proof).
- Live: pipeline exit status (pipefail-preserved) + `v25→v15→v25` assertion trail + phase events in the log;
  both durations recorded and distinguished: go-test per-package/test time vs `time` total wall clock (compile +
  provision + migrate + tests). Test binary bounded by `-timeout 5m` either way (pass fast, or bounded
  failure/panic — never silence); wall clock has no cap but is measured, not assumed.
- Highest-risk validation: the live replay leg through v16–v17 — the exact versions that stalled the original run.

## Risks / Open Questions

- If Neon itself stalls outside migrate (e.g., provisioning path, which has no deadlines by explicit prior scope),
  the 5-minute panic dump will show it outside adapter phases — that outcome replans provisioning bounds as its own
  slice; it does not reopen the adapter.
- A pre-existing dirty state on the shared dev branch cannot affect this test (fresh scratch DB per run; roles are
  branch-wide but the SET repair is idempotent and asserted).
- Open questions: none. The DBeaver `default`-vs-`run-pattern` answer stays useful background but gates nothing —
  the test reads version truth from the migrated database itself.
