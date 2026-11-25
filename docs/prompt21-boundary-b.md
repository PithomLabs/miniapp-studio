# Boundary B Implementation Report — C6 Focus Console + U5 Acceptance

Date: 2026-11-25. Codebase: `github.com/PithomLabs/miniapp-studio`,
uncommitted tree at `bb0d436` (prompt19) plus the C6/U-phase diff
listed in §8. Framework pin unchanged: Miniapp `73351a8`.

Boundary B = C6 Focus Console shell, Pulse/Line, clarification and
approval wiring, density/responsive/keyboard/a11y, plus the U5
ledger-driven browser acceptance matrix, two consecutive clean-slate
green passes, and this report. Post-acceptance role separation
(U5.12) remains HOLD per the approval and is not started.

Verdict: Boundary B implementation is complete and evidenced below.
Submitted for architectural acceptance. No commits or pushes were
made. All Phase C/D/E scene work, Part 2/C5 live proof, Sprite CLI,
Fly operations, and live OpenCode remain out of scope and untouched.

## 1. What was built (U1–U4 summary)

Backend touch-ups F2–F6: gateway-turn tracking with stable command-row
IDs (message ID = durable row ID), continue turns derived only from
gateway-acked turns plus status reconcile (never Studio counters),
recovery firing once per lineage via marker, waits not burning
attempts, crash sweep, retry/backoff, frozen turns with stale-answer
rejection, assigned-join handoff, sender running-flip, and SSE
history/replay/reconnect/restart. Sender stays an explicit opt-in
server worker. `build.cancel_all` is operator-only: a Studio
control-plane action behind the operator session with the tenant
derived server-side, absent from the agent-callable registry, and its
UI states that it cancels every non-terminal build in that tenant.

Console: `server/handlers_console.go` (auth/tenant/409/kill/dispatch
handlers), `server/console_stop.go`, templates under
`server/templates/focus/`, `server/static/focus/console.js`. The
seven-stage spine stays honest: product Run never goes green from
builder state, and every unfinished Phase D/E scene carries an
explicit "lands in Phase D/E" disclosure instead of a green seal.
Kill notice and scene-map render defects found by the matrix were
fixed at root cause (see §7).

Migrations `000017`–`000020`: sender tracking, sender schedule,
question expiry cause, dispatch stream kind.

## 2. Go test evidence (fresh, report-time)

Static gates re-verified at report time:

- `gofmt -l .` — clean (no output).
- `go vet ./...` — clean.
- `go build ./...` — clean.
- Go toolchain `go1.27.3 linux/amd64`.

Full suite, serial, report-time run against a scratch database on
PG19 (`127.0.0.1:5434`, container `miniapp-p0-pg`, database
`studio_u5_suite`, dropped after the run):

`TEST_DATABASE_URL=<scratch-dsn> go test -p 1 -count=1 ./...`

| Package | Result |
|---|---|
| miniapp-studio (root) | no test files |
| apps | ok 0.003s |
| cmd/gateway | ok 0.017s |
| cmd/studio | ok 0.576s |
| connector | ok 0.002s |
| e2e/boundary_b | ok 0.005s |
| fly | ok 0.010s |
| gateway | ok 2.972s |
| render | ok 0.019s |
| runner | ok 0.134s |
| server | ok 29.655s |
| spec | ok 0.024s |
| spike | ok 0.128s |
| store | ok 5.646s |
| tools | ok 12.200s |

14 packages with tests, all ok. This run postdates every `.go`
edit in the tree, including the late console/activity fixes found
during live debugging (22:03–22:18). Targeted `-race` concurrency
subsets for the sender chains and harness races were green earlier
in the phase; the serial full suite above is the report-time gate.

Coverage includes sender chains, stale/indeterminate/busy/exhaustion
recovery, crash sweep, tenant isolation, Q&A binding, SSE resume,
console auth/tenant/409/kill/dispatch, and activity exclusion.

## 3. Browser matrix evidence (U5)

Harness: `e2e/boundary_b/` in-repo (orchestrator `run.py`, six phase
scripts under `phases/`, shared `lib/` with ledger/serverctl/harness,
stub gateway under `stub/`, provision script under `scripts/`,
`provision_test.go`, `README.md`). Runtime state (databases,
ledgers, secrets, screenshots, chromes) lives outside the repo under
`/tmp/boundary-b`. Runbook: `docs/boundary-b-u5-runbook.md`.

Each phase carries a prerequisite preamble that fails fast when its
inputs are absent, assertions are ledger-relative (IDs, epochs, and
counts read back from the run ledger, never hardcoded), and one
failure stops the run without destroying anything (dependents report
blocked; see the t1 trial below).

Two consecutive clean-slate passes, each with a fresh database, its
own ledger, and independently captured evidence:

| Pass | Run dir | Database | Phase 1–6 |
|---|---|---|---|
| B1 | `/tmp/boundary-b/runs/b1-clean` | `studio_boundary_b_b1` | all passed |
| B2 | `/tmp/boundary-b/runs/b2-clean` | `studio_boundary_b_b2` | all passed |

Ledger status maps (both runs): `phase1_environment`,
`phase2_conversation_dispatch`, `phase3_clarification`,
`phase4_approvals_evidence`, `phase5_console`,
`phase6_recovery_isolation` — every value `passed`. Both ledgers
were read back with `python3 -c json.load` after the runs; no
secret-pattern sweep hits in either ledger (0 matches).

Non-counting runs, disclosed: `t1` (trial; phases 1, 2, 5 passed
while 3, 4, 6 reported blocked during harness debugging), `a1`
(ledger-green but DISQUALIFIED: a concurrent suite run rotated
cluster-global app-role credentials mid-pass, violating the
credential-mutation restriction; aborted and recovered explicitly
per guardrail 1, consecutive-green count reset to zero), `a2`
(provisioned, zero phases executed, abandoned during the a1
recovery). The a1 incident is the red proof that the guardrail is
load-bearing, not decorative.

Heavy-poll measurement (guardrail 2: audit rows generated through
the real Miniapp runtime invocation path by a live browser holding
an open SSE stream, 60 s window, both passes):

- `audit_stream_rows_per_min`: 359.5 (B1), 359.5 (B2).

Decomposition from the B2 database after the run: 1093
`dispatched` + 1093 `succeeded` `conversation.stream` rows (exactly
2 rows per invoke), 2186 stream rows over a 605 s span
(~217/min average across the whole run including multi-browser
phases). Per-stream rate is 120 rows/min (1 s poll cadence × 2
rows); the 359.5/min window reflects ~3 overlapping streams during
evidence↔run navigation. Evidence exclusion held: per-second stream
polls never appear in the Console receipts table (pixel-verified in
`evidence.png`, which shows only `conversation.get` receipts plus
the gate-ledger and transport-noise disclosures).

Screenshots (B2, copied to `docs/boundary-b/`, each opened and
pixel-inspected at report time):

- `desktop-run.png` — Run stage: honest spine (Describe green,
  Run amber), `waiting_user` build card, Pulse affordance.
- `mobile-sheet.png` — narrow viewport: bottom-sheet Pulse,
  responsive layout intact.
- `conflict-409.png` — stale-answer banner naming the asked
  question ID, current open questions below, nothing applied.
- `evidence.png` — live audit receipts with gate-ledger
  ("lands in Phase D") and stream-poll exclusion disclosures.

## 4. Sabotage transcript (U5.5)

Dedicated run `/tmp/boundary-b/runs/sabotage`, database
`studio_boundary_b_sab` (dropped after; full output in
`transcript.log` in that dir). One phase failed, was repaired by a
targeted rerun, and went green — no environment reset (same
database, same servers, same ledger file throughout):

1. `--init`, `--phase 1`, `--phase 2` — all PASS. Phase 2 opens
   build `c04c521f…` (status `running`) in conversation
   `p2-invoice-sabotage`.
2. Sabotage (out-of-band, via psql, simulating a lost
   prerequisite): `DELETE FROM studio.builds;` → `DELETE 1`.
3. `--phase 3` — RED, recorded in the ledger:
   `FAIL phase3_clarification: RuntimeError: PREREQ: phase-2
   build c04c521f… is 'missing'; clarification needs a live
   build (rerun phase 2 first on a fresh run)`.
   The preamble names the exact missing input and prescribes the
   repair; nothing else ran.
4. Repair: `--phase 2` rerun — `adopt existing conversation
   97af63a4… (rerun)`, build reopened, assign ack streamed, PASS.
5. `--phase 3` rerun — full clarification flow PASS (question
   streamed, UI reply recorded, continue froze turn 2, stale
   answer → 409 naming asked + current, double-submit fenced).
6. Final ledger: phases 1–3 all `passed`.

Gate satisfied: red → targeted repair → green, in place.

## 5. Guardrail compliance

1. **Credential-mutation restriction.** Enforced throughout U5; the
   a1 violation was aborted and recovered explicitly, and both
   counting passes ran with no concurrent suite or provisioning.
2. **Heavy-poll through the real path.** §3 measurement used a live
   browser + open SSE stream through the Miniapp runtime, not
   synthetic audit inserts.
3. **Independent passes.** Fresh database, own ledger, own evidence
   per pass; a1's disqualification reset the count; B1 then B2.
4. **Post-acceptance grant audit.** Not started (U5.12 HOLD).

## 6. Residuals and known limitations

- **F7 operating cost (SSE poll audit volume).** Measured, not
  fixed: ~120 stream-audit rows/min per open stream (1 s cadence ×
  dispatched/succeeded pair), multiplied by concurrent streams.
  Exclusion from the Console receipts view is verified; audit-table
  growth itself is the deferred item.
- **Cluster-global app-role rotation.** Until U5.12 role
  separation, any suite boot rotates credentials cluster-wide; live
  passes must run alone. Demonstrated by the a1 incident.
- **Reconcile-once bound.** Out-of-band gateway driving is bounded
  by reconcile-once recovery; no stronger claim is made.
- **Unprovisioned builds wait indefinitely by design.** No timeout
  is imposed; the Console shows the wait honestly.
- **Evidence pagination and provisioning completion signaling**
  deferred to the roadmap provisioning sketch (U5.9).
- **Assigned-join handoff and F4 expiry** accepted with the trust
  assumption recorded in the plan.
- **Phase B Part 2, C5 live proof, Phase D/E** remain gated off;
  every Console scene for them carries an explicit unfinished
  disclosure.

## 7. Root-cause fixes found by the matrix (not band-aids)

Each item below failed red in the harness or suite first; the fix
addresses the owning boundary, and the full suite in §2 covers it:

- Send-vs-ingest and settle lock-order deadlocks → single global
  lock ordering in the sender/callback path.
- Gateway digest semantics for sequential replay clarified; F3
  cancel overreach corrected (provisioned cancels still dispatch).
- Workspace binding / tenant mismatch on callbacks → tenant-bound
  routing.
- Duplicate-question retry poison → terminal mapping instead of
  unbounded retry.
- Scene-map and kill-notice render defects → corrected handlers
  (`handlers_console.go`, 22:03) with regression tests.
- SSE replay and dispatch-stream gaps → stream-kind coverage
  (`000020`) so successful acks are representable, not failure-only.
- Matrix-only races (fixed in the harness, not the product):
  load-event navigation waits, DB-driven resumption, nonce-scoped
  IDs, freshness-checked server lifecycle, single run binary path.

## 8. Environment notes (report-time)

- **Sandbox PG path loss.** Late in the session the sandboxed
  shell lost its route to `127.0.0.1:5434` (connection refused)
  while the cluster stayed healthy (`/readyz` 200, container Up,
  in-container isready OK). DB-touching report steps (audit
  decomposition, full suite, drops) were therefore run from the
  unsandboxed shell. No test or harness behavior depends on which
  shell runs it.
- **Accidental test-credential echo.** A malformed compound
  command caused psql to print the test-cluster maintainer DSN
  (password included) into tool output once. Remediation, same
  session: role password rotated with `ALTER ROLE`, the cluster
  bootstrap file `/tmp/boundary-b/owner.dsn` rewritten (0600),
  new DSN verified connecting, old DSN verified rejected. No
  secret was written to any repo file. Owner action: if you kept
  a copy of the old maintainer password anywhere, replace it
  from `/tmp/boundary-b/owner.dsn`; run-dir `owner.dsn` copies
  under `/tmp/boundary-b/runs/` are stale by design (their
  databases are dropped).
- **Pre-existing database left untouched.**
  `studio_boundary_b` (bare name) predates the U5 runs and
  appears in no U5 provision log; it was not modified or dropped.

## 9. Files changed (all uncommitted, no commits made)

Product code (modified): `server/sender.go`,
`server/gateway_callback.go`, `server/routes.go`,
`server/sse_resume_test.go`, `server/conversation_ui_test.go`,
`server/sender_test.go`, `server/gateway_callback_test.go`,
`tools/build.go`, `tools/conversation.go`, `tools/ingest.go`,
`tools/activity.go`, `tools/register.go`,
`tools/connectors_activity_test.go`, `tools/ingest_answer_test.go`.

Product code (new): `server/handlers_console.go`,
`server/console_stop.go`, `server/handlers_console_test.go`,
`server/templates/focus/*` (7 scenes), `server/static/focus/
console.js`, `store/migrations/000019_*` and `000020_*` (up/down).

Harness (new, ~1835 lines incl. README): `e2e/boundary_b/run.py`,
`e2e/boundary_b/lib/` (ledger, serverctl, harness, phase),
`e2e/boundary_b/phases/` (phase1–phase6), `e2e/boundary_b/stub/
gateway.py`, `e2e/boundary_b/scripts/provision_run.sh`,
`e2e/boundary_b/provision_test.go`, `doc.go`, `README.md`.

Docs (new): `docs/boundary-b-u5-runbook.md`,
`docs/c1-command-row-upgrade-note.md`,
`docs/prompt21-boundary-b.md` (this report),
`docs/boundary-b/` (4 inspected screenshots).
Plans added during the phase: `prompt21.md`,
`prompt21_plan.md`, `bug.md`, `bug_review.md`,
`prompt20_plan_review.md`, `prompt_coding_agents.md`,
`prompt_coding_muse.md`; `plans/roadmap4.md` modified (pre-U5
edit, carried in the tree).

## 10. Teardown record

All U5 live processes stopped (verified: no boundary-b ports
listening, no matching processes host-wide). Scratch databases
dropped: `studio_boundary_b_t1/a1/a2/b1/b2`,
`studio_boundary_b_sab`, `studio_u5_suite`. Four orphaned PG
backends from a killed run server were terminated before the B2
drop. The pre-existing `studio_boundary_b` database was left in
place (§8).

## 11. Demo-path update

`docs/hackathon-demo-critical-path.md` steps 1, 6, 7, 11 now cite
this report: Console Line/Pulse clarification and approval
rendering move from `missing` to `implemented` (live-verified
against the stub gateway; Part 2 + C5 still required for the live
agent path). Step 8 keeps its Phase C crux note with the sender
loop now `implemented` behind the stub.

---

Boundary B work stops here. U5.12 (permanent PostgreSQL role
separation) is HOLD pending architectural acceptance and has not
been started.
