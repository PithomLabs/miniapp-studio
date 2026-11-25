# Plan to implement plans/prompt11.md — Phase B Part 1 corrections

## Goal

Execute the bounded correction bundle from `plans/prompt11.md` (governing
review: `plans/adv_review_prompt10.md`): fix B1 + F2–F9 at their root
causes, disposition M1–M16, correct the evidence/runbook overstatements
(§G/§J), reinforce the Miniapp application contract (§7), and prove the
result on PostgreSQL 19 Docker only — then stop for correction
verification. Part 2 live work stays unauthorized.

## Success Criteria

- Every prompt11 §9 item holds: B1 fail-closed identity separation; F2/F3
  symlink exploits refused; F4 concurrent identical Continue executes once;
  F5 terminality from process outcome only; F6 explicit honest resume;
  F7 non-disruptive provision; F8/F9 requeue + freshness; no silent
  pump loss; evidence/runbook corrected; M1–M16 fixed or explicitly
  carried; probes P1–P4 re-run green; PG19-only gates green.
- No §8 forbidden change: no live/Sprite/OAuth/Fly mutation, no Miniapp
  or F1/G-01 change, no P5 change, no new adapters/engines, no commits.
- The correction report (§10) exists with root causes, red/green
  evidence, and an explicit Part-2-not-executed statement.

## Approach

One uninterrupted implementation pass in dependency order (§H accepted
decisions are extended, never redesigned; §I deferred items stay
deferred). For every finding: reproduce red on the original tree first,
trace the violated invariant, fix at the ownership boundary, prove
green, then run the gates. Reuse the reviewer's probes at `/tmp/b1probe`
(still present on disk) for P1–P4 reproduction and re-verification.

### Key decisions

1. **B1 mechanism: `SysProcAttr.Credential` + fail-closed-by-construction
   (no setuid helper, no invented service `user` field).** The supervisor
   gains agent-identity config and applies `Credential` on every agent
   spawn; any spawn error (including unprivileged `EPERM`) fails the
   command honestly — there is no code path that launches the agent
   without the drop, so no explicit "am I root" check can drift.
   Rejected: setuid helper (second binary, install complexity, no
   stdlib need) and shared-user fallback (explicitly forbidden).
   Credential/group code lives in a linux platform file; non-linux
   builds refuse agent spawn (gateway is linux-only). Supplementary
   groups are set explicitly; execution verifies Go's group behavior
   empirically by spawning `id` and asserting its output.
   Verification splits honestly: fake tests prove refusal
   (unprivileged + different uid → error), plumbing (same-uid
   Credential spawn works, groups applied), and mode assertions on
   every created path; the real different-uid drop is proved live by
   new runbook §7 checks (uid + key-read-denied). `setup.sh` gains the
   inbox model (gateway-owned dir, group `studio-io`, mode 2730;
   `answer.json` 0640 `studio-gw:studio-io`), staging 0600 + key
   cleanup after install, service PATH pinned to root-owned dirs, and
   uid discovery baked into the service command line
   (`--agent-uid/--agent-gid` flags on `cmd/gateway`; unset → refuse
   agent spawn). AMENDED per plan review (see A3): the gateway's
   effective identity and required spawn privilege are specified
   explicitly (privileged service identity so the Credential drop to
   the agent uid succeeds; absent privilege → agent execution
   refuses, no same-uid fallback; live identity + key-read-denied
   proof stays a Part 2 gate), and the question-file read path is
   closed: agent-owned drops must be group-readable (spawn wrapper
   pins agent umask 007 → 0660 `studio-agent:studio-io`), the
   gateway reads as a `studio-io` member, and present-but-invalid
   drops fail the run honestly instead of being silently ignored.
2. **F2/F3: `os.OpenRoot` inbox operations (stdlib-only preserved).**
   Question read: dir-relative open with `O_NOFOLLOW`, require regular
   file owned by the agent uid, size cap, then consume. Answer write:
   `O_CREAT|O_EXCL|O_NOFOLLOW`; on `EEXIST`, lstat and handle by
   ownership — agent-owned/symlink → remove + bounded retry;
   gateway-owned → atomic temp+rename replace (preserves designed
   single-slot overwrite; the sticky bit already stops the agent from
   touching gateway-owned files). `O_NOFOLLOW`/`O_EXCL` passthrough is
   verified empirically via symlink fixtures, not assumed.
3. **F4: durable claim + in-memory serialization (no second persistence
   mechanism). AMENDED per plan review (see A1 below): at-most-once
   dispatch with explicit recovery for ambiguous outcomes — stale
   claims recover as indeterminate, never auto-drop.** The existing
   dedup store file gains claim records: `Claim` (persist
   in-progress before dispatch: message id, payload digest,
   build/session/turn identity, invocation id) → `Complete`
   (overwrite with result) / `Drop` (only when the supervisor proves
   process creation failed and no child started) / `Indeterminate`
   (recovery default). `Check` replays completed only; redelivery of
   an indeterminate id returns explicit 409 (recover via F6
   continue-as-new-turn, never a retry of the original); same id with
   a different payload digest is rejected. In-process: per-build
   mutex queues distinct commands (blocking on ctx); an in-package
   per-(build,message) call table (stdlib-only — no x/sync)
   shares one execution among digest-matching duplicates and 409s
   digest mismatches without joining. Lock order is call-table-join
   → build mutex, never reversed. Crash recovery reaps identifiable
   orphans (claim pid + starttime) first, reconciles persisted
   outbox events (stable ids; Studio dedups; no relaunch), then
   marks unprovable claims indeterminate. `Complete`-persist failure
   aborts the run honestly (cancel agent, mark failed, drop claim,
   500) instead of being ignored as today.
4. **F5: terminality from exit code + question-drop only.**
   Translator maps mid-run `error` lines to progress-with-detail;
   pump keeps its question-first ordering (a yield with exit 0 is a
   question, never `done`); nonzero exit stays honest failure.
   `TestAdapterFailureHonest` is updated to the corrected semantics.
5. **F6: continue-from-failed only, with turn fencing (M11 folded in).**
   `done`/`cancelled` stay 409; `failed` + authorized continue
   launches a fresh supervised run (same workdir, `--session` lineage)
   and emits `agent.resumed` only after verified launch. No
   auto-restart, no fabricated resume. Turn fencing: record tracks
   turn, assign sets 1, continue requires payload turn == record+1
   (400 when absent — contract tightening; operator CLI + runbook
   updated in the same phase since the CLI is the only caller).
   Recovery emits before marking with a deterministic per-build event
   id (fixes the M6 Update-then-emit window via idempotent re-emit).
6. **F7: create-if-absent + explicit upgrade operation. AMENDED per
   plan review (see A2): explicit non-destructive failure rules +
   cross-process lock.** `GetService` first: absent → create;
   healthy → preserve (still stage + idempotent setup + key refresh
   + running-verify); unhealthy or inspection error → fail closed
   (report the condition, never delete/recreate); replacement only
   via explicit `gateway upgrade`. Binary digest drift (M14: sha256
   printed + echoed in the tool result as the record — no migration)
   refuses with the named upgrade op (guarded by a live-session
   check unless forced). The per-sprite lock is a DB advisory lock
   keyed by sprite-name hash held for the provision duration — it
   coordinates genuinely concurrent invocations across processes
   (serve + operator CLI share the framework DB), not just
   in-process goroutines; proved by racing independent sessions
   against a concurrency-observing fake.
7. **F9: re-sign per attempt, original bytes immutable.**
   `protocol.Resign` rebuilds an envelope with fresh `SentAt` and the
   same MessageID/build/workspace/kind/name/payload; the deliverer
   gains keys and re-signs in memory each attempt (no extra
   persistence, no new crash window). Studio's skew check is untouched.
8. **F8: per-build signed `outbox_inspect` / `outbox_requeue` commands.**
   Tenant-safe by construction (items filtered by build; requeue
   verifies ownership, preserves MessageID, resets attempts with a
   requeue note). Pump failure: bounded retry, then durable
   degraded+failed marking surfaced via status; if the mark write
   also fails, log + stop (catastrophic-disk residual, documented).
   The fictitious §16 comment is replaced with the requeue + Phase C
   sender-reconciliation requirement. Status keeps service-wide
   outbox counts (counts only, documented).
9. **Minors ride with their components** (no separate churn pass):
   M1 dir-fsync helper with F4's store work; M3/M4/M5 (SIGTERM +
   `WithoutCancel` spawn + shutdown signaling; process groups; stderr
   ring) with B1's supervisor work; M6/M7 (per-record skip, deterministic
   recovery ids, argv[0]+starttime reaper with Server boot time) with
   F6; M9/M10 with F5/F2; M12 https-except-loopback in client +
   deliverer wiring with F8/F9; M13 CLI allowlist + fetch truncation
   with F8's new commands; M16 (header, `turn_id` populated from
   record turn, `shortID` hex-validate-else-hash) as hygiene;
   M15 disposition starts from the existing `TestCallbackKDFWiring`
   (which already drives KDFProvider via config — the review's
   zero-reference claim is stale) and extends only proved gaps.
10. **Evidence honesty over completeness.** Matrix uses the five
    required states; `run.ts` claims are vendored (pinned excerpt +
    digest) if fetchable during execution, else explicitly marked
    SPECIFIED-FROM-UNPINNED-SOURCE with P-G0 carrying full weight —
    never silently kept as "verified".

## Amendments from prompt11_plan_review.md

**A1 — F4 indeterminate claims (correctness flaw fix).** Dropping a
stale claim after recovery is unsound: an event may already be
persisted or delivered while `Complete` never landed, so re-dispatch
would execute the same logical work twice. Recovery rule:
- `Drop` only when the supervisor proves process creation failed
  (Go `cmd.Start` error ⇒ no child exists) and no child was started.
- Any claim that cannot prove no-child — including claims without a
  recorded pid (crash may have landed between start and pid-record,
  or before dispatch) — recovers as **indeterminate**: reap the
  identifiable orphan via claim pid + starttime (M7 identity,
  M4 group kill), reconcile persisted outbox events (stable ids
  stay queued; Studio dedups; the deterministic recovery `failed`
  callback still emits), and never relaunch that logical command.
- Deliberate recovery uses the F6 continue-from-failed path as a new
  fenced turn — never disguised as a retry of the original command.
- Same MessageID + different payload digest ⇒ 409 conflict (checked
  before joining an in-flight call, so mismatches never share a
  result). Redelivery of an indeterminate id ⇒ explicit 409 naming
  the continue path.
- Guarantee stated honestly: **at-most-once dispatch with explicit
  recovery for ambiguous outcomes.**
- New adversarial test: agent emits → outbox persists → crash
  before `Complete` → redelivery ⇒ no second launch, no duplicate
  event. Plus crash-window tests: between claim persist / process
  start / pid-record.

**A2 — F7 failure rules + cross-process lock.**
- Absent service → create. Healthy → preserve. Unhealthy service or
  `GetService` error → fail closed with the reported condition; no
  delete, no recreate. Replacement only via explicit upgrade.
- The per-sprite lock is a PG advisory lock (two-integer form, per
  the established codebase pattern) on `hashtext(sprite)` held for
  the whole provision; it coordinates serve + operator CLI because
  both transact against the same framework DB.
- Proof test races two independent DB sessions (separate pools) on
  one sprite name against a concurrency-observing fake and asserts
  max in-flight install == 1 plus attach-convergence, alongside the
  unhealthy/inspection-error fail-closed cases.

**A3 — B1 identity, privilege, and question read path.**
- Effective identities: the gateway service runs under the Sprite
  service uid (observed live in Part 2); the agent must run as
  `studio-agent` (uid/gid from `--agent-uid/--agent-gid`). The
  Credential drop to a different uid requires spawn privilege
  (root/CAP_SETUID in practice). Rule: the gateway always applies
  the drop and treats any spawn failure as refusal — if the Sprite
  cannot provide the privilege, agent execution refuses; there is
  no same-uid fallback. Live identity + key-read-denied proof
  remains a Part 2 gate (runbook §7).
- Inbox matrix: dir gateway-owned `studio-gw:studio-io` 2730;
  `answer.json` gateway-owned 0640 (agent reads via group, sticky
  bit + ownership stop replace/remove/overwrite); `question.json`
  agent-owned, required group-readable — the agent spawn wrapper
  pins umask 007 so drops land 0660 `studio-agent:studio-io`, and
  the gateway (a `studio-io` member) reads via group. Consume
  accepts iff regular file + uid == agent uid + group-readable +
  size ≤ cap; absent ⇒ normal exit-code outcome; present-but-invalid
  ⇒ honest run failure with detail (never silent).

## Steps

1. **Phase 0 — baseline (no code changes).** Confirm PG19 (`SELECT
   version(); SHOW server_version;` → 19.x) on 127.0.0.1:5434 via the
   established escalated pattern; record baseline gates green;
   reproduce P1–P4 against the original tree with `/tmp/b1probe`
   (re-derive from review §C if any probe bit-rotted). No fix lands
   without its red reproduction.
2. **Phase 1 — B1 + supervisor minors (M3/M4/M5).** Credential
   plumbing + fail-closed, linux/non-linux split, flags + wiring,
   `setup.sh` inbox/staging/PATH/uid work, `builder.md` false promise
   corrected. Tests: refusal, plumbing (`id` output), group
   assertion, mode assertions, umask-pin proof (spawned child
   creates 0660), shutdown-signals-runs, grandchild reaping,
   stderr-ring detail. Empirically pin Go's group behavior.
3. **Phase 2 — F2/F3 (+M10).** OpenRoot inbox ops, answer replace
   protocol. Full regression matrix: question-symlink→key refused,
   answer-symlink→victim refused, pre-created regular, oversized,
   malformed, concurrent replace, restart-during-create.
4. **Phase 3 — F4 (+M1).** Claim lifecycle with indeterminate
   recovery (A1), call table + per-build mutex, digest-conflict
   409s, abort-on-Complete-failure, dir-fsync helper. Tests:
   N identical → 1 execution; N distinct → serialized; digest
   mismatch → 409; emit-then-crash redelivery → no second launch,
   no duplicate event; claim/start/pid-record crash windows →
   indeterminate + F6 recovery; no stranded session.
5. **Phase 4 — F5 (+M9).** Translator + pump terminality, scanner
   errors. Tests: error-then-success, error-then-failure,
   question-yield (incl. with exit 0), malformed output, killed
   process, overlong line.
6. **Phase 5 — F6 (+M6/M7/M11).** Resume state machine, turn fencing,
   deterministic recovery ids, precise reaper, CLI `--turn` +
   runbook §11.5 rewrite. E2E: recover → continue → resumed →
   completed; stale-turn 409; corrupt-record skip.
7. **Phase 6 — F7 (+M14).** Create-if-absent, unhealthy/error
   fail-closed (A2), advisory per-sprite lock, explicit upgrade op,
   digest echo. Regression: second provision leaves the first live
   session untouched; cross-session concurrency proof; no
   delete/recreate on unhealthy or inspection failure.
8. **Phase 7 — F8/F9 (+M2/M12/M13, pump path).** Resign + deliverer
   keys, inspect/requeue commands + CLI, bounded-retry pump +
   degraded marking, timeout wiring, https rule, §16 comment fix,
   Phase C reconciliation note. Tests: aged redelivery accepted,
   stale-forged bytes 401, requeue drains, enqueue failure surfaced.
9. **Phase 8 — §7 contract + M15/M16 + evidence/runbook (§6).**
   `builder.md` + assignment-prompt reinforcement (no Phase D
   pipeline); KDF gap extension; hygiene; matrix rewrite in five
   states; G1–G8 corrections; runbook §6/§7/§11.5/§12 updates;
   `run.ts` vendoring attempt with honest fallback.
10. **Phase 9 — gates + probes + report.** Full gate set PG19-only,
    P1–P4 re-run green, correction report per §10, then STOP.

## Validation Plan

- Per finding: named red reproduction on original code → green after
  the fix; concurrency/crash/restart/malformed paths exercised where
  the prompt requires them (F2/F3/F4/F6/F8 matrices above).
- Final gates (PG19 Docker only, serial, escalated shell for Docker):
  `gofmt -l .` empty; `go vet ./...`; `go build ./...`;
  `go test -p 1 ./...`; `go test -race -p 1 ./gateway ./store ./tools
  ./server ./cmd/studio ./cmd/gateway`; static `cmd/gateway` ELF
  re-verify. Record PG version, exact commands, exit codes, per-package
  results, skips (expect zero), and re-run P1–P4 outcomes.
- Highest-risk validation: B1's real different-uid drop is NOT
  fake-provable (tests run unprivileged) — the plan contains it via
  refusal + plumbing + mode tests now and explicit live EACCES proofs
  in runbook §7 later, with the residual stated, not hidden.

## Risks / Open Questions

- `run.ts` vendoring may fail on network; fallback is marking the
  claims unpinned (plan step 9), not blocking.
- If PG19 :5434 is unreachable, stop DB-backed verification and
  report the exact failure per §3 — no substitution, no completion
  claim.
- None. No user-owned product choice is outstanding; all technical
  choices above carry reversible defaults with reasons.
