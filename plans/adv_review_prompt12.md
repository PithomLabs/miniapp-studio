# Adversarial Review — Phase B Part 1 Corrections (prompt12)

Read-only verification of the prompt11 correction pass against
`plans/adv_review_prompt10.md`. No code, docs, or configuration modified.
No live operations performed. Part 2 remains unauthorized.

Note: prompt12 §9 names this deliverable
`docs/adv-review-prompt11-correction.md`; it was ordered as
`plans/adv_review_prompt12.md`. This file is that report.

## 1. Executive verdict

**ACCEPTED WITH BOUNDED RESIDUALS.**

The correction pass is genuine root-cause work, not symptom patching. I
independently traced every B1/F2–F9 correction through its execution
paths, re-ran the byte-identical P1–P4 probes (red in my prompt10
session, green today with zero probe changes), reproduced all runnable
gates green, recounted all 349 tests exactly, and verified each M- and
G-disposition against source. The B1 privilege drop, OpenRoot file-drop
discipline, durable indeterminate claims, honest terminality, F6 resume,
non-disruptive provisioning, and re-signing deliverer are all correctly
designed and faithfully implemented, with adversarial tests that assert
real contracts (live processes, symlink fixtures, reopen crash
simulation, cross-session DB coordination).

What keeps this from unqualified ACCEPTED: (a) the database gates are
**blocked by environment, not by the code** — PG19 at 127.0.0.1:5434
refused connections, the Docker socket denied access, and no credentials
exist in this session, so F7/M14/M15 execution evidence and the full
DB-backed suite could not be re-run here (per AGENTS.md §8.3 and the
prompt12 standard, no substitute was used and no DB result is claimed);
(b) the B1 deployment specification is incoherent on one point — the
service privilege requirement is unstated and runbook §7b's file
ownership expectations contradict root operation (new finding N1,
documentation + small hardening); (c) ten Low findings (N2–N10) and the
M5 identifier collision need tracking. None is a structural defect; all
are bounded below with exact corrections. Part 2 must additionally
re-verify the DB-blocked items with PG19 access — named explicitly in
§12 rather than assumed.

## 2. Scope and tree state reviewed

- Studio root `/home/chaschel/Documents/go/miniapp-studio`, all-untracked
  zero-commit tree — recorded before review (22 untracked entries, 0
  tracked modifications) and unchanged after except this report file,
  which keeps the tree all-untracked.
- Miniapp: working tree clean (`git status --porcelain` empty),
  `go.mod` pin unchanged
  (`miniapp v0.0.0-20261123181255-73351a801fc5`), pin verified as an
  ancestor of the checkout. No framework modification.
- Reviewed: `gateway/` (all non-test + targeted tests), `tools/`
  provisioning/upgrade, `cmd/gateway`, `cmd/studio` gateway CLI,
  `fly` sprite surface, `server` KDF wiring, `tools/provisiondata/*`,
  and all eight governing documents (all present; none missing).
- Frozen scope held: no F1/G-01/P5 movement ( Builder instructions are
  prompt reinforcement only — no admission pipeline); tool surface adds
  exactly one tool (`build.upgrade`, F7-authorized); no new modules
  (`go.mod` requires unchanged); `gateway/` production code remains
  stdlib-only; no commits, pushes, stages, or history changes.

## 3. Documents and source areas inspected

Governing docs (all read in full): `plans/adv_review_prompt10.md`,
`plans/prompt11.md`, `plans/prompt11_plan_review.md` (A1/A2/A3),
`plans/prompt11_plan.md`, `docs/prompt11-correction-report.md`,
`docs/gateway-phase-b-evidence.md` (v1 + correction record + matrix
v2), `docs/gateway-phase-b-live-runbook.md` (all sections),
`docs/opencode-run-v1.19.6.md`, and
`plans/miniapp-studio-implementation-agent-guidelines.md` (§§4–5
applied as the engineering bar; the correction visibly follows them:
OpenRoot + O_NOFOLLOW/O_EXCL + fstat checks per §4.1, durable claims
per §5.2, indeterminate rule per §5.3, re-sign-with-stable-id per
§5.4, explicit requeue per §5.5).

Source: `gateway/supervisor.go`, `proc_linux.go`, `proc_other.go`,
`filedrop.go`, `fs.go`, `collector.go`, `dedup.go`, `server.go` (all
881 lines), `recover.go`, `opencode.go` (all 499), `events.go`,
`session.go`, `outbox.go`, `deliver.go`, `protocol.go`, `keys.go`,
`client.go`, `cmd/gateway/main.go`, `cmd/studio/gateway.go`,
`tools/build_provision.go`, `tools/build_upgrade.go`,
`tools/provisiondata/{setup.sh,builder.md,opencode.json}`,
`fly/interfaces.go` + `fly/sprites.go` (WriteFile/perms), `server`
KDF wiring. Tests: all gateway correction tests by name + full reads
of the adversarial core (`server_f4_test` crash/concurrency tests,
filedrop symlink tests, identity-drop tests, resume e2e, deliverer
freshness tests, F7 cross-session lock test, KDF tests).

## 4. Independent gate verification

| Gate | Command | Result |
|---|---|---|
| format | `gofmt -l .` | clean (empty), exit 0 |
| vet | `go vet ./...` | exit 0 |
| build | `go build ./...` | exit 0 |
| static Linux | `CGO_ENABLED=0 GOOS=linux go build -o /tmp/studio-gateway-review ./cmd/gateway` + `file` | ELF 64-bit x86-64 statically linked ✓ |
| darwin stubs | `GOOS=darwin go build ./gateway/ ./cmd/gateway/` | ok |
| shell | `sh -n tools/provisiondata/setup.sh` | ok |
| full suite, no DB | `go test -p 1 -count=1 ./...` (no TEST_DATABASE_URL) | all ok **except** pre-existing `server/TestBareInvocationFailsClosed`, which needs a DB and predates the correction (Nov 24 file; fails only for lack of DB, not a code defect) |
| gateway + cmd/gateway | `go test -race -count=1 ./gateway/ ./cmd/gateway/` | ok (4.0s / 1.0s) |
| DB-backed suites | full `./...` + 6-package race on PG19 | **BLOCKED — see below** |
| test counts | `grep -rh '^func Test[A-Za-z0-9_]*('` per package | **349 total / 99 gateway, every per-package count exact** (gateway 99, store 23, tools 58, server 56, cmd/studio 9, cmd/gateway 3, fly 16, apps 5, connector 3, render 11, runner 29, spec 34, spike 3) |
| P1–P4 probes | unmodified `/tmp/b1probe` | all green (see §6) |

**Database blocker (exact):** `bash /dev/tcp/127.0.0.1/5434` →
`Connection refused`; `docker ps` → `permission denied ... /var/run/docker.sock`
(uid 1000, groups `chaschel,nogroup` — not in `docker` group); no
`TEST_DATABASE_URL` in the environment and no credentials available.
Per AGENTS.md §8.3 and prompt12 §5, database-backed verification was
stopped, no PG16/17 substitute was used, and no DB result is claimed.
Consequence: F7/M14/M15 execution, migration/RLS suites, and the full
DB-backed + race runs are verified by code/test-assertion inspection
only (marked accordingly in §5), and must be re-executed with PG19
access before Part 2 (§12 residual R-DB).

## 5. Finding-by-finding closure matrix

Statuses: VERIFIED CLOSED | CLOSED — EVIDENCE LIMITED |
DOCUMENTATION CORRECTION REQUIRED | CARRIED TO PART 2 | NOT ACCEPTED.
Evidence classes per prompt12 §2 (code/test/local-process/DB/live).

| ID | Status | Basis |
|---|---|---|
| B1 | DOCUMENTATION CORRECTION REQUIRED | Mechanism code-verified + fake-tested: `Credential` on every spawn (`supervisor.go:136`), no fallback path (same-uid requires test-only `AllowSameUID`, serve never sets it — verified by search), unprivileged EPERM refuses, groups pinned non-empty, umask-007 wrapper, SecureDir 2730/2750/2770 + 0640 answers, staging 0700/0600 + verified cleanup, PATH pinned via env-wrapped service, identity resolution refuses boot on failure. Tests green (refusal/plumbing/`id`/umask/modes). **But**: the service privilege requirement is unstated anywhere (§7 discovers *setup* privilege, not service runtime uid), and runbook §7b expects `studio-gw`-owned inbox files while the only working configuration (privileged service) produces service-uid-owned files — see new finding N1. Live EACCES honestly deferred to §7b. |
| F2 | VERIFIED CLOSED | OpenRoot + O_NOFOLLOW + descriptor fstat (regular/agent-owned/io-group/group-readable) + 64KB double-check + unlink-before-read; symlink/foreign-owner/non-regular refused; malformed-but-owned yields leniently (never success-fabricating). Unit + P2 probe green (symlink → honest run failure, zero exfil). |
| F3 | VERIFIED CLOSED | O_CREAT\|O_EXCL\|O_NOFOLLOW + ownership-defined EEXIST (symlink/non-regular→remove; gateway-owned→atomic temp+rename; agent-owned→remove+recreate; foreign→refuse) + 0640 + stale-temp sweep + Continue clears question slot. Unit/concurrency + P3 probe green (victim intact). |
| F4 | VERIFIED CLOSED | Durable claim-before-dispatch (id/digest/build/session/turn/invocation) in the existing store file (no second mechanism); call-table digest join + per-build sem with valid lock order (join→sem→store, nothing held across waits — traced); digest-mismatch 409; indeterminate default with 409-naming-continue; Drop only on proven no-child (Runner contract sound: all Start error paths predate process existence); Complete-failure aborts honestly; busy→retryable drop. A1 crash window traced end-to-end (§C scenario holds: no relaunch, no duplicate event). Concurrency + reopen + P1 green. Residuals: pid recorded post-capture not post-spawn (N5, Low); recovery-guidance wording (N3, Low). |
| F5 | VERIFIED CLOSED | Error lines → not-ok progress; terminality from exit code + question-drop only (drop beats exit 0; nonzero honest failure); corrected `TestAdapterFailureHonest` + translator cases assert the new contract; P4 green (`[progress progress message done]`). Vendored-source coupling claim matches the audit doc. |
| F6 | VERIFIED CLOSED | Failed-only resume (done/cancelled 409), lineage verification with stray-kill on mismatch, verified-launch-gated `resumed`, pre-resume restore on failure, turn fencing (absent→400, stale→409), CLI `--turn` + rewritten §11.5 (read: matches implementation exactly, incl. 409/500 guidance). E2E recover→continue→resumed→completed green (server+real adapter, lineage/workdir/turn asserted). |
| F7 | CLOSED — EVIDENCE LIMITED | Implementation code-verified: inspect-first (absent→create, healthy→digest-verified refresh without restart, unhealthy/inspection-error→fail closed), explicit upgrade with all-builds idle check (+force), advisory xact lock held for the whole op with consistent lock order (row→advisory in both tools), digest echo, service args pinned (boot-args gap closed), staging atomic key install. Test assertions read and sound (incl. genuine cross-session lock proof: separate pools, delayed observer, max-in-flight 1). **Cannot execute**: all F7/upgrade tests are DB-backed → blocked. Re-verify on PG19 before Part 2. |
| F8 | VERIFIED CLOSED | Per-build signed inspect/requeue (ownership-checked, identity-preserving, action recorded); pump degraded-marking + drain (no wedge, no silent loss); §16 fiction replaced with requeue + Phase C requirement; M2/M12/M13 wired. Unit + integration tests green (real HTTP stubs). |
| F9 | VERIFIED CLOSED | `Resign` preserves id/build/payload, fresh `SentAt`; deliverer re-signs per attempt from immutable stored bytes; Studio skew untouched; stale-forged bytes still 401 (negative test green); aged-redelivery test green. |
| M1 | VERIFIED CLOSED | `fsyncDir` after every atomic rename in dedup/outbox/session stores (all three verified) + helper test; green. |
| M2 | VERIFIED CLOSED | `AttemptTimeout` (30s default) bounds each POST via request ctx; hanging-stub test green. |
| M3 | VERIFIED CLOSED | SIGTERM handled, spawn on `WithoutCancel` ctx, `Shutdown` cancels live runs bounded by ctx; request-cancel-spared test green. |
| M4 | VERIFIED CLOSED | `Setpgid` + group kill with verified non-leader fallback; grandchild + fallback tests use real processes, green. |
| M5 (stderr) | VERIFIED CLOSED | Continuous drain into 32KB tail ring; `Stderr()` serves snapshots; wedge + bounded-tail tests green. Identifier collision with carried M5 (SSE) is new finding N4. |
| M6 | VERIFIED CLOSED | `ListPartial` skip-and-log, emit-before-mark with deterministic ids, already-queued proceeds to marking; skip + idempotence tests green. |
| M7 | VERIFIED CLOSED | argv[0]-basename-exact + starttime-before-self reaper; decoy/PID-reuse/parser tests green. `SelfTicks<=0` degrades loudly at boot (disclosed). |
| M8 | VERIFIED CLOSED | OpenRoot + O_NOFOLLOW + fd-pinned read + double size check; all-symlink refusal; swap-hammer test green. |
| M9 | VERIFIED CLOSED | `scanner.Err()` counted + `truncated` surfaced in status detail; overlong-line test green. |
| M10 | VERIFIED CLOSED | 64KB cap, stat + post-read double check; oversize test green. |
| M11 | VERIFIED CLOSED | Absent→400, non-integer→400, stale→409, next-turn-only; CLI `--turn` (+agreement check) + runbook rows; fencing tests green. |
| M12 | VERIFIED CLOSED | `RequireSecureTransport` enforced in client, deliverer, serve boot, provision + upgrade tools; loopback exception correctly bounded (localhost + IP loopback; fail-closed on odd schemes); refuse/allow tests green. |
| M13 | VERIFIED CLOSED | CLI allowlist (pre-config, pre-network), fetch 1KB-inline stubbing, `--output` 0600; allowlist + formatter units ran green (DB-backed CLI tests skip without DB, gracefully). |
| M14 | CLOSED — EVIDENCE LIMITED | Digest computed + echoed in provision AND upgrade results (code-verified); echo tests read and sound — but DB-backed, execution blocked. |
| M15 | CLOSED — EVIDENCE LIMITED | KDF wiring + wrong-secret-401 tests present in tree and read (correct assertions); `DeriveKey` unit green; server round-trips DB-blocked. Vintage dispute (report says wiring test "pre-existed", my prompt10 row said zero KDF test references): unresolvable — zero-commit tree, mtimes inconclusive; I own the uncertainty (see §8). Substance satisfied regardless. |
| M16 | VERIFIED CLOSED | Package header refreshed (one stale const comment remains — folded into N4); `turn_id` populated inside signed bytes (M16 mapping); `shortID` hex-validate-or-hash; tests green. |
| G1 | VERIFIED CLOSED | Independent recount exact: 349 total / 99 gateway, all 13 per-package counts match; method = top-level `func Test*(`, subtests excluded (consistent with claim). |
| G2 | VERIFIED CLOSED | Matrix v2 states single-user-proven + dual-user-live (§7b) honestly; directionality tests exist and pass. Caveat rides B1's doc correction (N1). |
| G3 | VERIFIED CLOSED | v2 uses future tense + explicit "never exercised"; "now proves" survives only inside the unmarked v1 block → folded into N4's supersede-banner requirement. |
| G4 | CLOSED — EVIDENCE LIMITED | Vendored excerpt present with blob hashes + fetch method; excerpts internally support every cited claim (denial ruleset, emit shape, six types, error/exit coupling, exit codes, --session + honest unknown-id failure); code cites the doc. Upstream hash comparison impossible here (external fetch forbidden) — one-command check for anyone with network (see §12). |
| G5 | VERIFIED CLOSED | Requeue/inspect implemented + tested; matrix + comment corrected; Phase C requirement recorded. |
| G6 | VERIFIED CLOSED | Claims + serialization implemented + tested; guarantee honestly framed as at-most-once dispatch (verified wording in code comments, report, and matrix). |
| G7 | VERIFIED CLOSED | Resume implemented + e2e-tested; §11.5 rewritten and verified line-by-line against the state machine (409/500 guidance all match code paths). |
| G8 | VERIFIED CLOSED | `flyErr` comment downgraded to capped-but-unverified with the live-inspection requirement; no secret-freedom claimed. |

## 6. P1–P4 reproduction matrix

The original `/tmp/b1probe` program survives on disk and ran **unmodified**
against the corrected tree (it already sent `turn: 1`; zero-value record
Turn satisfies fencing) — the cleanest possible red/green comparison.

| Probe | Red (prompt10 executed run, documented in adv_review_prompt10 F1–F5/F4) | Green (this review, executed today, same binary source) |
|---|---|---|
| P1 concurrent identical Continue | `codes=[200 200] adapter.Continue calls=2` (double execution) | `codes=[200 200] adapter.Continue calls=1` — duplicates share the leader; both 200 with one execution. Matches correction §5 verbatim. |
| P2 question symlink → key file | yielded Body contained `HMAC-KEY-MATERIAL` | `kinds=[message failed]`, zero key material — symlink refused, run fails honestly. Matches §5. |
| P3 answer symlink → victim | victim clobbered with answer JSON | victim byte-identical (`ORIGINAL-KEY` intact). Matches §5. |
| P4 mid-run error + exit 0 | `kinds=[progress failed message done]` (premature terminal) | `kinds=[progress progress message done]`, no `failed`. Matches §5. |

Per-phase reds beyond P1–P4 (F6/F7/F8 crash windows, etc.) cannot be
reproduced independently — no VCS baseline exists to restore the original
tree, and reconstructing it would mean modifying the repo. They are
**reported-but-unreproducible**; what IS independently established is that
each corresponding regression test exists, asserts the corrected contract
(read), and passes on the corrected tree (executed, except DB-backed ones
noted in §5).

## 7. Miniapp contract assessment

Satisfied at the required level (prompt reinforcement, not runtime
enforcement — Phase D owns admission): `builder.md` gains an explicit
"Application contract (non-negotiable)" section (Go + Miniapp +
MiniappSpec + runtime; yield-instead-of-improvise rule), and
`assignmentPrompt` (`gateway/opencode.go:130-140`) restates the
exclusive contract on every assignment with an honest comment that
instructions don't enforce. No alternative stack is referenced
anywhere as an option; no Phase D pipeline was built (tool surface
adds only F7-authorized `build.upgrade`). No contradiction introduced.

## 8. Documentation/evidence consistency findings

- **M5 identifier collision (new finding N4, Low/doc).** Traced to
  source: `plans/prompt10.md:480` (§7 carried items) defines M5 as SSE
  stream lifetime; my `plans/adv_review_prompt10.md:87` reused M5 for
  the stderr ring. Both descendants are now in-tree (runbook §15 "M5:
  SSE…" vs correction "M5 stderr FIXED"), so "M1–M16 all fixed" can be
  misread as covering SSE lifetime. The runbook backlog itself is
  correctly classified (still listed as must-close live-segment work,
  NOT represented as implemented — verified). Requires: rename +
  cross-references (N4).
- **v1 matrix not marked superseded (part of N4).** The correction
  appends v2 but leaves "Capability matrix v1 (honest states)" plus
  its deviations/findings standing unmarked; a reader landing mid-doc
  meets stale claims (e.g. present-tense "Live P-G3 now proves…")
  before any correction context. Requires the unmistakable banner
  prompt12 §I demands.
- **M15 vintage dispute: unresolvable, immaterial.** The report says
  the KDF wiring test "pre-existed" and calls my zero-reference row
  stale. Forensics: the `mutate`-param boot helper it needs lives in
  an 08:41 (pre-correction-window) file with no other non-nil callers
  — consistent with pre-existence — but mtimes prove nothing without
  VCS, and my original grep output is not retained verbatim. I cannot
  confirm or refute either direction, and state that plainly. It does
  not affect M15's disposition (tests exist now; e2e halves DB-blocked
  either way). Recommend dropping the "stale" attribution for neutral
  wording.
- **"No test was deleted" (report §3): unverifiable** — no VCS baseline
  exists to diff against. Corroborating (not proving): counts grew in
  every touched package and the three named semantic corrections are
  present with corrected assertions (verified by reading
  `TestAdapterFailureHonest` + the two translator error cases).
- **B1 threat-model matrix (report §7): accurate** as far as it goes;
  verified row-by-row against code except the service-uid-dependent
  ownership cells, which inherit the N1 incoherence (matrix says
  `studio-gw`-owned inbox/answers; only true if the service runs as
  studio-gw, which cannot drop privileges — the matrix must state the
  privileged-service assumption or go uid-agnostic).
- **Runbook §7b fallback honesty: good.** The agent self-report
  fallback is explicitly labeled weaker evidence that "supplements
  but does not replace" kernel-observed denial — exactly the
  discipline prompt12 §6 requires. (It still needs the N1 ownership
  corrections to be executable.)
- **No other staleness found**: §11.5/§12/§6/§6.5 match implemented
  behavior command-for-command (verified against CLI flags, tool
  inputs, and status shapes); matrix v2's five states are applied
  correctly in every row I checked; R1–R10 and residuals are carried
  with homes, not claimed.

## 9. Residual and Part 2 live-proof register

Carried residuals (all belong to Part 2 runbook gates, none claimed
complete — verified each has a checkpoint + expected observation + stop
condition in the runbook): real different-uid drop + key EACCES (§7b);
dual-user inbox access matrix (§7b); full `serve()` boot under service
identities (§7/§7b); `setup.sh` execution (§6–§7); real OpenCode no-op +
file compliance P-G3 (§10); P-G0/P-G1 live semantics (§10 + forced
runs); warm/cold + latency (§11); OAuth P-E ladder (§8); exec privilege
+ service identity (§7 — but see N1: must also pin the *service*
privilege requirement, currently unstated); plugin/egress/tokens
(§6/§8); R1–R10 (mapped). Gaps I add: reaper-must-actually-reap proof
(§11 should record pre-restart agent PIDs + post-recovery absence —
M7's argv[0] check assumes the live process image keeps the `opencode`
argv[0], unverified); G4 one-fetch hash comparison.

## 10. New findings

Severity scale per prompt12 §8 (Critical/High/Medium/Low). All are
confirmed defects or discrepancies except where labeled residual.

**N1 — B1 deployment spec incoherent on service identity + file
ownership (Medium, documentation + small hardening).**
Location: `tools/provisiondata/setup.sh` (no service-identity
provisioning possible — API has no user field), `cmd/gateway/main.go`
`serve()` (no privilege/capability check), runbook §7b ownership
expectations, correction report §7 matrix. Invariant: the B1 boundary
must be deployable as specified. Root cause: the mechanism (Credential
drop) requires a privileged service, but nothing states, pins, or
verifies that requirement, while §7b + the report matrix assert
`studio-gw`-owned inbox/answer files — impossible under the only
working configuration (privileged service creates service-uid-owned
files; a studio-gw service cannot drop at all). Reproduction: code
walk (no `setcap`/chown-to-studio-gw/user-field exists anywhere —
verified by search) + §7b `stat` expectations vs `SecureDir`/
`writeFileComplete` (chgrp/chmod only, never chown owner). Impact:
live run either fails its own §7b checks or, worse, an operator
"fixes" it by weakening perms. Correction: (a) state the privilege
requirement (service must run with setuid privilege; drops fail
closed otherwise); (b) make §7b + matrix ownership assertions
uid-agnostic (mode + group + actual access probes, not literal owner
names); (c) add a boot-time drop-capability self-test (spawn `true`
as the agent uid; refuse boot — or warn loudly — on EPERM) with a
unit test. Blocks Part 1? No (code fail-closed in all cases) — but
must land before Part 2 runs §7b.

**N2 — Upgrade idle check TOCTOU + empty-URL skip (Low).**
Location: `tools/build_upgrade.go:92-136`. An assign racing the
upgrade (direct to gateway, outside the advisory lock) can start a
run after the idle scan but before delete/create; builds with empty
URL are skipped rather than refused. Impact: killed run with honest
failed callback (fail-loud, bounded). Correction: re-verify idle
immediately before delete/create; refuse (not skip) unscannable
builds. Regression: racing-assign test with fake timings; empty-URL
refusal test.

**N3 — Recovery-guidance messages misdirect (Low).**
Location: `gateway/server.go:272-273` ("safe to retry" — same id
actually 409s; must continue-as-new-turn) and the indeterminate 409
text (assumes continue works, but the no-record case needs a fresh
assign). Impact: operator confusion during incidents. Correction:
name the correct next action per case. Regression: assert message
text in the existing abort/indeterminate tests.

**N4 — M5 identifier collision + unmarked v1 matrix (Low/doc).**
Locations: runbook §15, evidence v1 block, correction M5.
`prompt10.md:480` defined M5=SSE lifetime; `adv_review_prompt10.md:87`
(I own this) reused M5 for the stderr ring; both meanings are now
in-tree, so "M1–M16 all fixed" can mislead, and v1 lacks the
unmistakable superseded banner prompt12 §I requires. Correction:
rename one lineage (suggest `M5-SSE` for the backlog + keep
correction-M5 with an explicit alias note), banner v1 as
SUPERSEDED with a pointer to v2 + the correction record, fix the
two present-tense stragglers or scope them under the banner.

**N5 — Claim pid recorded late (Low hardening).**
Location: `gateway/server.go:257-263` (RecordLaunch after launch
returns, i.e. after stream capture). A crash during agent startup
leaves a pidless claim (correctly indeterminate) with a possibly
live, unreaped, unidentifiable orphan (EPIPE usually ends it —
best-effort, not guaranteed). Correction: persist pid immediately
after `Runner.Start` returns (adapter returns pid alongside the
stream, or the claim path records it pre-capture). Regression:
kill-after-start-pre-capture simulation asserting recorded pid +
reap.

**N6 — Launch identity capture has no timeout (Low).**
Location: `gateway/opencode.go` `launch` head-read loop. A hung
binary blocks holding the build semaphore; redeliveries 503 until
crash/recovery. Correction: bounded capture timeout → kill +
honest launch failure. Regression: hanging-runner test asserting
bounded failure + semaphore release.

**N7 — Deliverer transition failures silent (Low).**
Location: `gateway/deliver.go` `pass()` (`_ =` on
MarkAcked/MarkRetry/MarkFailed). A dead disk spins the loop with no
log and no status signal (markings are safety-neutral — redelivery
dedups — but invisible). Correction: log transition errors + expose
a deliverer-error counter in status. Regression: fault-injecting
outbox test asserting log/signal.

**N8 — Staging re-creation window (Low).**
Location: `fly/sprites.go` `WriteFile` (0755/0644) + setup ordering.
Steady state is safe (existing 0700 dir), but if staging is ever
recreated while sibling agents run, key bytes land world-readable
until setup's chmod seconds later. Correction: restrictive modes at
write time for `keys-*` (or verify-and-fix dir mode before staging
keys). Regression: fake-fs mode assertion on staged key files.

**N9 — Adapter Continue check-then-act (Low hardening).**
Location: `gateway/opencode.go:215-217` (live-check, unlock, launch
later). Unreachable via the serialized server (per-build sem +
call table verified), but any future direct caller re-opens F4.
Correction: per-session mutex spanning check→launch-registration.
Regression: direct-adapter concurrent-continue test.

**N10 — Server IOGID zero-fallback (Low).**
Location: `gateway/server.go:462-465` (silent `os.Getegid()`
default). Masks misconfiguration; outcome is fail-closed (drops
refused) but confusing. Correction: refuse explicit misconfiguration
(tests set the field). Regression: zero-IOGID workdir creation
refuses.

## 11. Checks not run and why

- All DB-backed execution (full `./...` + 6-package race on PG19,
  F7 lock/upgrade/provision tests, M14/M15 e2e, migration/RLS suites):
  **blocked** — PG19 :5434 refused, Docker socket denied, no
  credentials; substitution forbidden. Code + test assertions for
  these were read and verified instead (marked CLOSED — EVIDENCE
  LIMITED where execution matters).
- G4 upstream hash comparison: **blocked** — external fetch forbidden;
  one-command check for anyone with network.
- Live proofs (R1–R10, EACCES, setup.sh, OpenCode, OAuth, wake):
  **unauthorized** (Part 2) — verified only that each has a runbook
  home with observation + stop conditions.
- Red-before reproduction on the original tree: **impossible** —
  zero-commit tree, no baseline to restore; P1–P4 reds are
  substantiated by my own prior executed runs (documented
  per-exploit outcomes) + byte-identical probe re-runs, not by
  re-execution.
- "No test was deleted": **unverifiable** for the same reason (no
  baseline); corroborated by growth in every touched package's count
  and the three named semantic corrections verified in-tree.

## 12. Final acceptance recommendation

**ACCEPTED WITH BOUNDED RESIDUALS.**

No in-scope blocker remains: every B1/F2–F9 correction is sound at
its ownership boundary, honestly framed (at-most-once, not
exactly-once), and covered by adversarial tests that assert real
contracts; all runnable gates pass; P1–P4 re-run green on the
identical probe code; evidence is accurate except the flagged doc
items. The following residuals require explicit tracking before the
relevant later gate (none reopens the architecture):

- **R-DOC (before Part 2):** N1 (privilege requirement + uid-agnostic
  §7b/matrix + boot self-test) and N4 (M5 rename + v1 superseded
  banner). Part 2's runbook cannot execute correctly without these.
- **R-DB (before Part 2):** re-execute on PG19 Docker with credentials:
  full `go test -p 1 ./...`, 6-package race, and specifically the F7
  lock/upgrade/provision, M14 digest, and M15 KDF suites. Record
  versions, exit codes, skips.
- **R-NET (any time, one fetch):** G4 upstream blob-hash comparison.
- **R-LOW (tracked):** N2, N3, N5–N10 fixed or explicitly carried with
  owner sign-off; none gates Part 2 alone, N2/N5/N6 nearest the live
  path should ride the next touch of those files.
- **R-LIVE (Part 2):** R1–R10 + EACCES + setup + reaper-actually-reaps
  (added: record agent PIDs pre-restart, verify absence post-recovery)
  via the runbook checkpoints, which I verified exist with
  observations and stop conditions.

Part 2 authorization additionally requires no new general review —
verification of R-DOC/R-DB/R-NET closure is sufficient.

- No Sprite/Fly/OAuth/live installation work was performed.
- Part 2 remains unauthorized.
- No code, docs, or configuration were modified by the reviewer.
- No commits or pushes were made.
