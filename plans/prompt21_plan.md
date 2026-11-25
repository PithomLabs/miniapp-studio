# Prompt21 Plan — Boundary B Consolidated U5 Execution

## Goal

Finish Boundary B by fixing the verification architecture first,
then proving repeatability: restructure the monolithic browser
matrix into resumable ledger-driven phases, complete all remaining
U5 acceptance cases, demonstrate two consecutive clean-slate green
passes, and file the Boundary B report. After acceptance, land the
permanent PostgreSQL role separation as a dedicated follow-up.

## Success Criteria

- A browser-matrix failure is locally diagnosable and rerunnable
  without destroying previously verified state (proven by the
  mandatory sabotage gate, not asserted).
- Two consecutive complete green passes, each on an independently
  provisioned clean database; the second is the repeatability
  evidence. One green followed by a failure is not acceptance.
- Full U5 scope stays green: evidence receipts, density, offline/
  reconnect, restart resume, black-hole retrying, keyboard kill,
  tenant-B isolation, mobile sheet, screenshots, seven scenes,
  all required UI states.
- Evidence stays visible under transport polling (query-layer
  exclusion verified + heavy-poll acceptance test).
- Provisioning latency handled by derived deadlines (30s hold +
  bounded allowance), never arbitrary sleeps or weakened asserts.
- Live server provably survives a concurrent full suite run after
  role separation (post-acceptance gate).
- No Miniapp changes, no F1/G-01/P5 changes, no Sprite/Fly/OpenCode/
  Part 2/C5/Phase D-E work, no commits without authorization.

## Approach

Sequence exactly per prompt21: restructure → sabotage gate → finish
U5 → two clean passes → report → STOP for acceptance → role
separation as a separate gated task. Causes 2 and 3 stay as
dispositioned (verify exclusion; keep polling; roadmap sketch).
The harness moves from `/tmp` into the repo as reviewable
acceptance tooling; only run-scoped runtime state (ledger,
profiles, screenshots staging) stays under `/tmp`.

## Steps

### U5.0 Harness home + runbook skeleton

Create `e2e/boundary_b/` in the repo: `phases/`, `lib/`,
`run.py` (orchestrator), `README.md`. Create
`docs/boundary-b-u5-runbook.md` with the interim
credential-mutation restriction as its header (prompt21 §2
verbatim, including the abort/verify/restart procedure). No
product-code changes in this unit.

### U5.1 Split `drive.py` into phase scripts + shared lib

Port the 14 numbered sections of `/tmp/boundary-b/drive.py`
verbatim in behavior into 7 phase scripts matching bug_review
§2.D (environment/auth; conversation+dispatch; clarification
lifecycle; approvals+evidence; console behavior; recovery+
isolation; final clean-slate orchestration entry). Factor
one shared lib: CDP client, login, psql helper, callback
signer caller, stub-gateway manager, screenshot helper.
Each phase is one file, runnable standalone:
`run.py --phase N --ledger <path> [--db <name>]`.

### U5.2 Run-scoped state ledger

One JSON ledger per run at `/tmp/boundary-b/runs/<run-id>/
ledger.json` (run ID = UTC timestamp + random suffix). Schema:
`schema_version`, `run_id`, `database` (name + cluster),
`server` (listen addr + binary sha256 + start epoch),
`operators` (tenant/binding, never tokens), `conversations`
and `builds` (phase-unique titles → IDs), `epochs` per phase,
`phases` (status + evidence refs). Atomic writes (tmp +
rename). Loader rejects missing/malformed/version-mismatched/
wrong-database ledgers with explicit errors. No credentials
or secrets in the ledger — assert this in the loader test.

### U5.3 Relative assertions

Replace every absolute assumption: kill count becomes
(live-before minus live-after) with per-ledger-build-ID
confirmation; "latest row" lookups become ledger-ID lookups;
epoch-second tags become `{phase}-{run-shortid}` titles
recorded in the ledger; event ordering asserted via stream
`seq` and gateway `message_id` correlation. Each replaced
assertion keeps equal or greater strength — narrowing an
assertion to make it pass is forbidden and will be called
out in review.

### U5.4 Prerequisite preambles

Each phase opens by verifying server health, database
identity vs. ledger, operator/session validity, and the exact
lifecycle state it needs (e.g. build in `waiting_user` with
open question Q), failing fast with `PREREQ:` diagnostics.
A phase that runs against a foreign or half-torn-down
environment must fail at line 3, not at a DOM assert.

### U5.5 Sabotage gate (mandatory)

Intentionally break one assertion in one phase; demonstrate:
only that phase fails, database and prior results preserved,
rerun of that phase alone goes green with no reset, dependent
phases report blocked (not failed). Restore, re-verify. This
demonstration is the gate — record its transcript as evidence.

### U5.6 Finish remaining U5 cases

Drive the restructured phases to green over the preserved
implementation: evidence receipts, OPS/DEV, offline/reconnect,
restart resume, black-hole retrying, keyboard kill, tenant B,
mobile sheet, screenshots. Offline/reconnect and restart-resume
get execution-path tracing on any failure (server log + stream
cursor + DB state), never "script mechanics" dismissal. Mobile
and keyboard failures must be classified DOM-harness vs. genuine
a11y/interaction defect with evidence.

### U5.7 Provisioning timing done right

Assertion deadlines derive from the configured 30s sender hold
(`server/sender.go:defaultSenderWait`) plus bounded scheduling
and dispatch allowance (recommend 30s + 15s, single constant,
commented). Bounded polling for persisted outcomes everywhere;
the SQL wake stays explicitly labeled test scaffolding. No
LISTEN/NOTIFY, no new machinery.

### U5.8 Evidence exclusion verification + heavy-poll test

Verify (don't rewrite) the shipped `activity.list exclude`
(query-layer, pre-LIMIT, schema-pinned, UI-disclosed). Add one
Go acceptance test: generate substantial `conversation.stream`
invocation volume, then assert business receipts remain
returned and stream polls are excluded. Measure real rates
during a matrix run and record invocations/day vs. rows/day
separately (prompt21's derived estimates: 86,400 / 172,800 per
continuous stream) in the Boundary B report's F7 residual.

### U5.9 Roadmap provisioning sketch

Two-line sketch in `plans/roadmap4.md` (durable
provisioning-completed signal/outbox + notify-to-wake with
polling retained as recovery). Deferred design record only.

### U5.10 Two clean-slate passes

Two consecutive full runs, each on an independently
provisioned clean database (fresh `studio_boundary_bN`,
migrated + operators provisioned per runbook). Capture:
per-phase PASS transcripts, screenshots, stub-gateway call
log, ledger copies, server logs. Any failure restarts the
count at zero.

### U5.11 Boundary B report + demo-path update, then STOP

File the report (prompt21 §7 contents: matrix architecture,
per-phase evidence, ledger design + sample, preamble/assertion
examples, sabotage results, both clean passes, screenshots,
four-cause postmortem, interim restriction, quantified audit
volume, provisioning timing evidence, residuals). Update
`docs/hackathon-demo-critical-path.md` with evidence
references. Then STOP and request architectural acceptance.
The role-separation unit below starts only after that
acceptance arrives.

### U5.12 Post-acceptance: permanent role separation

Split `EnsureAppRole` into create-if-missing + grants (never
touches an existing password) and an explicit rotate used
only by the provision CLI. Bootstrap `studio_app_test` once
(password set once, never rotated from test code), granted
only on test databases — audit every `GRANT ... studio_app`
in `store/migrations/000001–000013*.up.sql` to derive the
exact grant set, applied at test bootstrap, never via a
production migration. Update all four harnesses
(`server/server_test.go`, `tools/tools_test.go`,
`store/store_test.go`, `cmd/studio/studio_test.go`) to the
test role + run-specific databases (extend the existing
`studio_migcheck_<rand>` precedent). Update provisioning
docs/runbooks. Mandatory proof, live: start a server on
`studio_app`, run the full suite concurrently, demonstrate
unchanged credential, continued health, working authed
requests, test isolation from live data, and no cross-talk
between concurrent suite runs.

## Validation Plan

- Gates every unit: `gofmt -l .`, `go vet ./...`, `go build
  ./...`, full serial suite on isolated PG19 DBs (never
  concurrent with a live pass until U5.12 lands — interim
  restriction), `-race` on concurrency tests, migration
  round-trip suite.
- Sabotage gate (U5.5) must pass before any clean-slate pass.
- Clean-slate evidence: two consecutive transcripts + ledgers +
  screenshots, all quoted or attached in the report.
- Role separation (U5.12): red-before (concurrent suite kills
  live server on current code — already observed three times),
  green-after (survival proof per prompt21 §6).
- Highest-risk validation: the sabotage gate first (it proves
  the new structure is real), then offline/reconnect and
  restart-resume (likeliest genuine product defects).

## Risks / Open Questions

- Risk: hidden coupling between phases (shared Chrome profile
  state, sender timing) undermining independence — mitigated
  by per-phase preambles failing loudly and the sabotage gate.
- Risk: audit-volume measurement perturbs timing-sensitive
  steps — measure on a dedicated pass, not inside a clean-slate
  acceptance run.
- Open questions: none — all material choices resolve from
  prompt21, bug_review, or inspected code above.
