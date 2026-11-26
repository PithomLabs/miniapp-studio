# B1-R1 Independent Adversarial Re-Review

**Scope:** verify closure of B1 findings F-01–F-11 from `plans/adv_review_prompt26.md` against the R1 implementation. Read-only review; no code, tests, migrations, config, or docs modified. No cloud operations. No commits, pushes, resets, or branch changes. B2 not started and not authorized by this review.
**Path note:** prompt26.2 §5 names this deliverable `plans/b1-r1-adversarial-re-review.md`; it was ordered as `plans/adv_review_prompt26.2.md`. This file is that report.

## 1. Executive verdict

**ACCEPT B1.** All eleven findings are verified closed under their authorized dispositions (nine fixed, two documented limitations as the owner decision brief directed). No Critical or High findings remain or were introduced. DB-backed verification is independently green on PG 19.0: full serial suite 17/17, race suites clean, migration/RLS/parity green, targeted R1 suites green with zero skips. The one observed failure (a single `TestAdapterStatus` TempDir-cleanup flake under `-race`) is pre-existing, unrelated to R1, and documented with reproduction evidence below.

## 2. Finding-by-finding closure matrix

| ID | Disposition | Verdict |
|---|---|---|
| F-01 Filesystem TOCTOU + FIFO | Fix + accepted residual (privileged parent swap) | **Verified closed** |
| F-02 assume-unchanged / skip-worktree | Fix (refuse both flags everywhere) | **Verified closed** |
| F-03 Submodule state | Fix (enumerate, gate, toplevel proof) | **Verified closed** |
| F-04 Idempotency identity | Fix (namespaces + usage check + ownership verify) | **Verified closed** |
| F-05 Ignored files | Document (owner-authorized) | **Accepted limitation** |
| F-06 Git-derived secrets | Fix (refuse identity strings; typed scrub defense-in-depth) | **Verified closed** |
| F-07 waiting_user message | Fix (shared helper + runbook) | **Verified closed** |
| F-08 repo_path identity | Document (owner-authorized) | **Accepted limitation** |
| F-09 Serialization | Fix (mf2 + frozen mf1 + per-row dispatch) | **Verified closed** |
| F-10 Config independence | Fix (pin + audit) | **Verified closed** |
| F-11 Listing honesty | Fix (skip counts surfaced + tested) | **Verified closed** |

## 3. Root-cause remediation verification

### F-01 — Filesystem TOCTOU and FIFO safety: VERIFIED CLOSED
Every checklist item in prompt26.2 §2 was verified against `server/gitworktree.go` (`secureOpen`, `hashWorktreeFile`) plus live syscalls:
- Opens are descriptor-relative from a held root fd; every component (intermediate and final) opens with `O_NOFOLLOW`; intermediates add `O_DIRECTORY`.
- Type and size come from `fstat` on the open descriptor (`unix.Fstat`), never from path stat; size re-checked after the bounded read.
- Final open adds `O_NONBLOCK`: a FIFO opens without blocking and is refused by the fstat type gate — no hang path remains. Cancellation is checked per file and all I/O is bounded (4 MiB/file, 64 MiB total).
- Check-then-open is gone: there is no `Lstat`-then-`Open` sequence anywhere on the hashing path. Failures map to deterministic entry notes (`unresolvable path`, `vanished during inspection`, `unreadable`) or explicit fatal errors for budget breaches.
- **Syscall proof** (strace on a live inspection): `openat(fd, "f.txt", O_RDONLY|O_NONBLOCK|O_NOFOLLOW|O_CLOEXEC)`; all worktree file opens are fd-relative. The only path-based opens are the root fd itself, git subprocess internals, and loader paths.
- **Race re-run**: ~1,900 hammered symlink-swap inspections, 0 outside-hash hits (previously: mechanism present; now the mechanism itself is absent — `O_NOFOLLOW` fails `ELOOP` instead of redirecting).
- Accepted residual (runbook-documented, accurate): a privileged local attacker swapping a parent directory between top-level resolution and the first open. Correctly scoped — defeating that requires OS-level confinement outside B1's authority.

### F-02 — Index flags: VERIFIED CLOSED
- `hiddenFlagPaths` parses `ls-files -v -z`, flagging lowercase tags (assume-unchanged) and `S` (skip-worktree). Tag semantics verified correct.
- Enforced in **all three** read paths: `InspectWorktree` (capture + resume validation) and `QuickStatus` (UI live state) — refusal names count + sample paths.
- **Demonstrated live**: assume-unchanged fixture and skip-worktree fixture both refuse with the explicit message (previously: stable digest + `clean=true` on diverged bytes).
- Sparse checkouts correctly out of scope (they set skip-worktree, hence refuse loudly rather than bind blindly).

### F-03 — Submodule state: VERIFIED CLOSED
- Gitlinks enumerate from the index **including status-invisible clean ones** (closes the original gap, not just the dirty case).
- Gates verified in code: initialized (toplevel proof defeats the deinit trap — an emptied dir would otherwise let git answer from the parent), clean worktree, non-nested (index scan for mode 160000), and checked-out HEAD equal to the recorded gitlink SHA. Any failure refuses with an explicit reason.
- Deinitialized submodules cannot resolve through parent traversal (the toplevel-equality check), mismatched HEADs refuse, nested submodules refuse, and the 50-submodule cap bounds `ls-files` fan-out.
- Different submodule contents cannot produce identical accepted manifests: content enters only via verified HEAD SHAs; anything unverifiable refuses instead of recording a note.
- Submodule tests green (clean/dirty/mismatch/uninitialized/nested/states).

### F-04 — Idempotency identity: VERIFIED CLOSED
- Resume owns the `resume:` event namespace (was: shared `start:`); `opkeyOpFor` spans all five namespaces (`start`, `interrupt`, `resume`, `ckpt-ev`, `attempt:` including B0's record path).
- Cross-build checkpoint reuse → 409 (ownership verified on the conflict path, not just opkey presence); cross-operation reuse → 409 via the usage check; same-op retries still replay without writing (all replay paths return before any insert — verified in each of the five operations).
- TOCTOU between check and insert is closed by the build-row lock held across check-then-insert in every op, plus `ON CONFLICT DO NOTHING` + re-read-and-verify on the checkpoint insert race path.
- Concurrency evidence: `TestConcurrentCheckpointSeq`, `TestConcurrentStartOneWins`, `TestConcurrentResumeOneWins`, `TestResumeVsCheckpointRace` all green, including under `-race`.

### F-05 — Ignored files: ACCEPTED LIMITATION (as authorized)
- Runbook boundary section verified present and accurate: ignored files are outside the manifest, reconcile-by-hand rule with digests-only (never secret values) instruction.
- Console checkpoints card carries the limitation line; guarded by `TestConsoleBoundaryLines` (green).
- No tripwire, as authorized, with the stated rationale (counts false-negative; hashing risks secrets). The rationale is technically sound: a count cannot detect content swaps, and hashing dotfiles would persist secrets the manifest is forbidden to hold.

### F-06 — Git-derived secrets: VERIFIED CLOSED
- Refusal (not redaction) for identity strings: branch + entry paths + rename origins checked in `InspectWorktree` (covers capture **and** resume validation) and branch in `QuickStatus`. Rationale verified sound: redacting identity strings could collapse distinct paths.
- Typed-field scrub retained at form pass + op persistence boundary (idempotent by construction — redaction markers contain no matchable spans; verified by reading the span table).
- Commits need no scrubbing: hex validation admits nothing the span table matches (verified pattern set vs `[0-9a-f]` charset).
- **Demonstrated live**: secret-shaped path and AKIA-shaped branch both refuse; `TestCheckpointCanaryBranchRefused`, `TestResumeCanaryBranchRefused`, `TestRecordBranchScrubbed`, `TestConsoleCanaryBranchUnavailable` green.
- Residual note (pre-existing shared-matcher property, not an R1 defect): boundary characters glue (`feat-sk-test-…`) does not match the span table's boundary rule, so hyphen-joined shapes pass. R1 correctly applies the shared matcher everywhere required; span-table recall is owned outside B1.

### F-07 — waiting_user: VERIFIED CLOSED
- Shared `refuseNonLocal` helper names the actual state in both the checkpoint gate and the record gate (verified single definition, both call sites); runbook documents gateway-owned ineligibility. Message asserts the true state; existing allow/deny behavior otherwise unchanged (verified by diff-reading both gates).

### F-08 — repo_path: ACCEPTED LIMITATION (as authorized)
- Provenance-not-identity semantics documented in runbook + code comments + Console workbench line (all three verified present and mutually consistent).
- `TestResumePathIsProvenance` pins the semantics (moved tree resumes, original path preserved) — green.

### F-09 — Serialization: VERIFIED CLOSED
- mf2 is a genuine prefix code: magic + uvarint length-prefixed fields + uvarint entry count — verified injective by construction (self-delimiting lengths, fixed field order, explicit count).
- `TestFramingInjective` constructs a **real mf1-colliding pair** (asserts the collision first, so the test cannot pass vacuously) and asserts mf2 divergence — green.
- Separator/newline filename fixtures green; mf1 framer byte-identical to the original construction (verified by reading); per-row format dispatch defaults tagless rows to mf1; `TestResumeFromV1Row` inserts a genuine mf1 row via SQL and resumes from it — green.

### F-10 — Config independence: VERIFIED CLOSED
- `status.renames=true` pinned in shared `gitRun` for **all** status invocations (parent + submodule checks alike — verified at the single choke point, not per call site).
- Determinism audit comment verified accurate knob-by-knob (`-z`, explicit untracked flags, top-level execution, no diff, LC_ALL=C); `core.ignorecase` accepted variance is honestly scoped (byte-exact names preserved).
- Skew test (hostile config, identical entries) green.

### F-11 — Listing honesty: VERIFIED CLOSED
- Skip counting with deterministic injection test (injectable stat — no timing dependence), unreadable-subdirectory loud failure, depth/entry caps, UI surfacing of the skip count (template line verified). Incomplete listings cannot present as complete: skips render next to the listing.

## 4. Regression findings

**No regressions found.** Specifically verified:
- **B0 packet/attempt compatibility**: `build.packet` untouched (v0, no checkpoint references); `RecordAttempt` additive-only (`source_checkpoint` inheritance fires solely when a resumed `attempt_started` event exists with matching number; B0 flows without such events take identical paths); numbering/parentage/terminal semantics byte-identical outside the opkey-mismatch guard, which only newly-409s previously-ambiguous cross-op reuse.
- **Sender/dispatch**: R1 made zero changes under `server/sender.go` (working-tree sender diffs are Phase-D fetch_artifact work, identified by hunk inspection); held-assign non-burning path intact; no new dispatch triggers.
- **Migrations**: no schema change in R1 (verified: no migration files added/modified by R1); mf2 tag lives in manifest JSON.
- **RLS/tenant isolation**: no grant/policy changes; all new queries filter `tenant_id` and run inside `InTenantTx`; parity suite green.
- **UI honesty**: new surfaces render server-computed values with autoescape; SECRET refusal precedes persistence everywhere, so refused values cannot leak via render.
- The previously noted `TestAdapterStatus` flake reproduced once under `-race` load and passes in isolation; it is pre-existing, gateway-scoped, untouched by R1, and fails loudly (TempDir cleanup error, impossible to mistake for a pass). No data-race warnings in any run.

## 5. Database and concurrency verification results

Environment: PG 19.0 (Debian 19.0-1.pgdg13+1, verified via `SHOW server_version`), container `miniapp-p0-pg`, DSN composed in-shell from container env (never printed). Review used an isolated scratch role + run-unique test databases; all review-created DB resources reaped afterward (0 run DBs, role dropped, verified).

- Full suite `go test -p 1 -count=1 ./...`: **17/17 packages ok** (server 53s, tools 14s, store 7.9s).
- Targeted: checkpoint/resume/concurrency suites green; migration up/down + RLS/parity green; F-04/F-09/F-10/F-11 DB tests green.
- Race: `gateway` clean; server idempotency/concurrency subset clean; git-inspection subset clean. One `TestAdapterStatus` flake (see §4).
- `gofmt`/`go vet ./...`/`go build ./...`: clean.
- Fixture probes (no DB): TOCTOU hammer 0/1897 outside-hash hits with syscall-level mechanism proof; assume-unchanged + skip-worktree refusals demonstrated; AKIA branch/path refusals demonstrated; mf2 determinism demonstrated (identical digests across runs).
- Leftover check: 0 scratch run DBs remain; review role dropped; no repo files touched (only this report added).

## 6. Remaining risks and environmental limitations

1. **Privileged parent-dir swap** (runbook-accepted): TOCTOU below the per-component open — an attacker able to replace directories between top-level resolution and the first open can redirect inspection. Correctly documented as accepted; defeating it needs OS confinement outside B1's authority. No silent wrong-hash outcome is reachable through any observed path.
2. **Span-table recall** (pre-existing, shared): boundary-glued secrets evade matching everywhere the table is used, including the new refusal gates. R1 applies the table correctly; recall improvements belong to the shared table, not B1.
3. **DB verification scope**: full suite + targeted suites + race executed by the reviewer; multi-hour soak or fault-injection beyond the suite was not run.
4. **Live gateway/OpenCode behavior**: unchanged from B1 scope (local loop only); no live operations performed or needed for these findings.

## 7. Exact commands and test outcomes

All `go` invocations with `GOCACHE=/tmp/gocache-p5 GOPATH=/tmp/gopath-p5 GOTOOLCHAIN=local GOFLAGS=-mod=mod`; DB suites with `TEST_DATABASE_URL` composed in-shell from container env (redacted here and in logs).

- `gofmt -l .` → clean (no output)
- `go vet ./...` → exit 0
- `go build ./...` → exit 0
- `go test -count=1 -v -run 'TestStartAttempt|TestCheckpoint|TestInterrupt|TestResume|TestConcurrent|TestOpenAttempt|TestConsoleB1|TestProgressSurvives|TestLatestSkips' ./server/` → all PASS, 0 fail/skip
- `go test -count=1 -run 'TestMigrate|TestRLS|TestPrivilege|TestTenant|TestCrossTenant|TestPoolResets|TestDirectSession|TestRoleAttributes' ./store/` → ok (6.4s)
- `go test -count=1 -v -run 'TestPhaseARLSEnforced' ./store/` → PASS
- `go test -p 1 -count=1 ./...` → exit 0, 17/17 ok
- `go test -race ...` (idempotency/concurrency subset, filesystem subset, `./gateway/`) → ok except one `TestAdapterStatus` flake (passes in isolation; pre-existing)
- Fixture probes in `/tmp` (removed after): symlink-swap hammer, flag-refusal, secret-refusal, determinism — outcomes in §3/§4
- `SHOW server_version` → `19.0 (Debian 19.0-1.pgdg13+1)`; post-run: 0 run DBs, review role dropped

## 8. B0 compatibility assessment

Preserved. `build.packet` untouched at v0; `RecordAttempt` changes are strictly additive (cross-op 409 on previously-ambiguous reuse, op-boundary scrub, `source_checkpoint` inheritance gated on resumed-started events, accurate refusal message); numbering, parentage, terminal semantics, and opkey replay behavior for B0 flows are unchanged. Sender dispatch untouched by R1 (working-tree sender diffs are unrelated Phase-D fetch work, verified hunk-by-hunk). All B0 tests green unmodified in the full-suite run.

## 9. Final B1 acceptance recommendation

**ACCEPT B1.** Every finding is verified closed under its authorized disposition, no Critical or High findings remain or were introduced, and DB-backed verification is independently green. The two accepted limitations (F-05, F-08) are correctly documented with operator-visible surfacing, and the accepted TOCTOU residual is precisely bounded in the runbook. No further B1 review cycle is required; the remaining items (span-table recall, M5-SSE-class follow-ups owned elsewhere, live gateway behavior) belong to their owning tracks, not B1 acceptance.
