## Goal

Execute `plans/prompt9.md`: the bounded Phase A closure — three mandatory
corrections (agent-content secret handling, genuine shutdown→restart
recovery, RLS ENABLE on the ten Phase A tables), plus M1–M4 where small
and isolated, with M5–M8 recorded-and-carried. Deliverable: corrected
code + regression tests + updated `docs/gateway-phase-a-evidence.md`
covering the §5 list. No Phase B work.

## Success Criteria

- Item 1: secret-shaped canaries (key-positioned AND embedded/mid-string)
  driven through real tool/callback paths no longer persist unsanitized
  in any Studio-owned agent-message surface (`events.payload`,
  `questions.body`/`.options`, and everything derived from them);
  `build.status`, `conversation.get`, `conversation.stream`/SSE, and
  rendered history all read clean; surrounding text preserved; regression
  tests assert DB rows + every read surface incl. the previously failing
  question-body scenario; `app.invocations` framework residual disclosed
  in evidence (Miniapp untouched).
- Item 2: `TestPhaseARestartRecovery` performs the ordered 8-step
  lifecycle — srv1 shut down exactly once with errors surfaced, shutdown
  confirmed before srv2 boots, parked answer applies, history + SSE
  replay verified, duplicate callback acks without reapply.
- Item 3: migration `000016` enables RLS on all ten Phase A tables (both
  catalog flags true, FORCE kept), correct down migration, extended
  catalog test red-before/green-after, behavioral app-role enforcement
  test, evidence RLS claim corrected.
- M1–M4 folded (lock order + concurrency test if reproducible; dead code
  removed; strict `Parse`; "gapless" comment fixed) or explicitly
  recorded as not completed with reasons; M5–M8 carried in evidence.
- Full gates green (fmt/vet/build/tests/race/migration round-trip) on
  PG19 + PG17 via the escalated shell; §6 boundaries held; no commits.

## Approach

Root-cause corrections at owning boundaries, red-then-green each:

1. **Secret handling — scrub at the write choke point.** All gateway
   content enters Studio tables through `build.ingest_event`; scrub the
   payload there (before `events` insert and question derivation) so
   persisted surfaces are sanitary by construction — not output filters
   over dirty tables. Extend the scrub engine beyond key/prefix matching
   to ordered substring patterns for known credential shapes
   (`sk-live`/`sk-test`/generic `sk-` + body, `AKIA…`, `xox[baprs]-…`,
   `ghp_`/`gho_`/`github_pat_…`), replacing only the matched span with
   `[REDACTED]` and preserving surrounding text. Single pure function +
   exhaustive table tests. Keep read-path scrub as defense-in-depth (also
   covers pre-closure rows) and say so. Out of scope by design:
   user/system `messages.body` (operator's own words), `agent_session_id`
   (opaque correlator, resume-critical), `commands.payload` and
   `artifacts.manifest` (no Phase A read path — carried per review).
2. **Restart — honest lifecycle.** Make `Server.Close` idempotent
   (`sync.Once`) and error-returning; update all call sites (verified
   small: `cmd/studio` serve defer, `server.New` error path, test
   cleanup — cleanup wrappers report via `t.Errorf`, serve path logs).
   Rewrite the test to the ordered 8 steps; Close's return IS the
   shutdown confirmation (both halves are synchronous).
3. **RLS — migration + two tests.** `000016` ENABLEs all ten tables
   (FORCE retained); down restores DISABLE-with-FORCE exactly (verified
   via catalog flags both ways). Catalog test: extend the table list +
   a v15-staged scratch-DB leg proving red-before (flags false at v15)
   and green-after (true at v16) in one run. Behavioral test: two
   tenants seeded as owner, app-role pool asserts tenant-filtered rows
   when armed and zero rows unarmed.

## Steps

1. Baseline: `git status`, gofmt/vet/build/tests green on the untouched
   tree (record versions).
2. Item 3 first (schema foundation): write `000016` (+ down), extend the
   catalog test with the v15/v16 scratch legs, add the behavioral
   app-role test; run red (v15 leg) then green.
3. Item 1: extend the scrub engine (new patterns + span replacement +
   pure table tests), wire into `build.ingest_event` before persistence,
   add canary regression tests through real tool + callback paths
   asserting DB rows, `build.status`, `conversation.get`,
   `conversation.stream`/SSE bytes, and rendered page HTML (incl.
   question-body + embedded cases).
4. Item 2: `Close() error` + `sync.Once`, update call sites, rewrite the
   restart test to the ordered lifecycle; run green.
5. M1: reorder `questionAnswer` locks to build → turn → question (read
   question unlocked first, re-validate after locking); attempt
   pre-fix deadlock reproduction with a concurrency loop; keep the loop
   as regression/correctness coverage either way, documented honestly.
   M2: delete `questionIDsFor` (zero callers verified). M3: trailing-data
   rejection in `gateway.Parse` + test. M4: in-place header-comment fix
   in `000014` (comment-only, zero semantic delta — disclosed as such;
   round-trip + version gates prove no drift).
6. Full gates (PG19 + PG17 + race + migration round-trip incl. v16),
   update `docs/gateway-phase-a-evidence.md` per the §5 list (incl.
   invocations residual, PG versions, M-status, M5–M8 carry list),
   final report.

## Validation Plan

- Red-green proof per item: canary tests fail pre-fix (run new tests
  against unpatched ingest by stashing the one-line wire-up — or assert
  the v15/v16 legs for RLS); restart test fails if shutdown skipped
  (assert Close called: e.g., srv1 pool refuses new work post-close).
- Gates: `gofmt -l .` empty, `go vet ./...`, `go build ./...`,
  `go test -p 1 ./...` on PG19 + PG17, `-race` on touched packages,
  down-to-v10/up-to-v16 round-trip, catalog flags true/true on all ten.
- Evidence audit: every §5 bullet present with locations + actual
  results; no "13/13"-style miscounts; no live-Sprite contact (fake
  envelope paths only); frozen docs + Miniapp untouched; no commit.

## Risks / Open Questions

- No blocking questions: scope (§1–§4), boundaries (§6), and acceptance
  (§7) are explicit. Three judgment calls are stated above for pre-GO
  objection: (a) the credential-pattern set (extendable list, documented
  false-positive stance); (b) `Close() error` signature change (3 call
  sites); (c) M4 as an in-place comment-only edit with disclosure.
  Silence = approval to proceed as planned.
- Environment: PG19 (:5434) + PG17 (:5435) via the established escalated
  pattern; any unavailability will be disclosed per §5 with no
  substitution claims. The review's PG16 sandbox deviation does not
  apply to this execution.
- M1's deadlock loop may not reproduce pre-fix (timing-dependent); the
  plan keeps it as concurrency-correctness coverage regardless and
  reports which outcome obtained.
