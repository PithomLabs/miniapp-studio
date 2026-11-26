# B1 Scoped Adversarial Security and Correctness Review

**Review:** B1 local development loop (checkpoints, progress, interruption recovery, Git awareness).
**Mode:** independent, read-only, adversarial. No code, tests, migrations, config, or docs modified; no commits/pushes; no cloud operations.
**Note on deliverable path:** prompt26 §14 names `docs/reviews/b1-adversarial-review.md`; the owner instruction for this turn explicitly ordered `plans/adv_review_prompt26.md`. This file is that report.

## 1. Executive verdict

**CONDITIONAL ACCEPTANCE.**

No Critical findings. After deliberate severity calibration (below), no High findings. Three Medium findings require bounded remediation or explicit owner acceptance before B2: non-atomic worktree inspection (symlink-swap TOCTOU + unbounded hash open), assume-unchanged/skip-worktree invisibility, and unbound worktree-side submodule state. Six Low findings and explicit coverage gaps are tracked in §11.

Severity calibration, stated plainly because the TOCTOU call is close: I confirmed the Lstat→Open race mechanism deterministically (strace: `newfstatat(AT_SYMLINK_NOFOLLOW)` followed by `openat(O_RDONLY)` with no `O_NOFOLLOW`), but could not trigger it in ~1,900 hammered trials; exploitation needs a malicious local process winning a nanosecond window, and only hashes (never content) persist, so there is no exfiltration primitive. Downgrading further would ignore a real defect in the review's central invariant; upgrading to High would overstate an untriggered, local-only, non-exfiltrating race whose post-validation-modification equivalent no inspection design can prevent. Medium with a concrete, small fix is the honest grade.

Environment limitation (material): database-backed verification could not be executed in this session — the sandbox network namespace cannot reach PG19 (:5434 refused), local `docker`/escalated execution is unavailable ("approval prompts are disabled"), and no local PostgreSQL binaries exist. All DB-backed suites (checkpoint CRUD/resume/concurrency, migration round-trip, RLS/parity, full server suite) were therefore reviewed by code inspection only, and every DB-dependent claim below is marked accordingly. All runnable verification passed: `gofmt`/`go vet`/`go build` clean; full suite green with DB tests skipping cleanly; all 12 git-inspection tests green incl. `-race`; adversarial fixtures in §3 executed.

## 2. Scope and files inspected

Primary (complete reads): `server/gitworktree.go` (486), `server/build_checkpoint.go` (615), `server/build_attempt.go` (327), `store/migrations/000024_checkpoints.up/down.sql`, `server/gitworktree_test.go` (404), `server/build_checkpoint_test.go` (683), `server/config.go` workspace handling, `server/routes.go` (B1 routes), `server/handlers_console.go` B1 assembly + templates (`scene_run.html`, `scene_evidence.html`, `layout.html` diffs), `server/gateway_callback.go` (unchanged — verified via diff), `tools/ingest.go` + `tools/question.go` + `tools/build.go` (B1-relevant paths), `tools/scrub.go` + `artifact/secrets.go`, `store/stream.go`, `store/tenant.go`, `store/pool.go`, `store/privilege_parity_test.go`, `store/migrate_down_test.go`, `server/config_test.go` (workspace part).

Governing/evidence docs: root `SKILLS.md` (full), `plans/prompt25.md` (B1 directive, full), `docs/architecture/unified-lifecycle-ui-implementation-plan.md` (B1 slice + B2 prerequisites), `docs/bootstrap-evidence.md` (B0+B1 records), `docs/local-agent-bridge.md` (B0 + B1 addendum).

Change attribution: B1's footprint is new untracked files (`server/gitworktree.go`, `server/build_checkpoint*.go`, `server/gitworktree_test.go`, migration `000024`) plus hunks in config/handlers/routes/templates/tests. Unrelated working-tree changes explicitly excluded: `server/sender.go` fetch_artifact work (Phase D; B1 touches no dispatch path — verified hunk-by-hunk), `tools/scrub.go` delegation refactor (verified byte-identical patterns), e2e/Phase-D/spec/gateway files.

## 3. Review method and tests executed

Method: constitution + plan + directive + evidence first; full implementation reads (not diffs-only); call-path traces from HTTP/tool entry through auth/validation/FS/txn/persistence/render; adversarial fixtures for each Area A/B claim; attempted execution everywhere permitted.

Executed (all green unless noted):
- `gofmt -l`, `go vet`, `go build ./...`: clean.
- `go test -p 1 -count=1 ./...`: all packages ok; DB tests skip cleanly (verified SKIP, not silent pass, on checkpoint tests).
- All 12 `gitworktree` tests + `-race` subset: pass.
- Fixture P1 (content sensitivity): same-status/different-content → different digests. Pass.
- Fixture P2 (static symlink): recorded `non-regular file`, never followed. Pass.
- Fixture P3 (symlink-swap race, ~1,900 iterations): 0 outside-hash hits; mechanism proven separately via strace (see F-01).
- Fixture P4 (assume-unchanged): digest stable + `clean=true` while bytes differ. **Finding F-02 demonstrated.**
- Fixture P5 (ignored `.env` change): digest stable. Documented as designed-out-of-scope (Low F-05).
- Fixture P6 (submodule content): two different submodule byte-states → **identical digests**. **Finding F-03 demonstrated.**
- Fixture P8 (secret-shaped branch): persists verbatim, unredacted. **Finding F-06 demonstrated.**
- Skew probe (rapid content flips during inspection): 1,675/2,528 digests matched neither stable state (torn reads) — all fail-closed at resume (mismatch → refuse), never false accepts. Supports F-01's direction analysis.

Not executed (environment, not omission): every DB-backed test (checkpoint CRUD/resume/concurrency, migration up/down, RLS/parity, full server/tools suites with a database). The B1 evidence report claims these green with timings; I corroborated test *construction* by reading (genuine assertions, real concurrency, raw-SQL negative cases) but could not re-run them. Marked `NOT TESTED (execution)` wherever relied upon.

## 4. Findings by severity

### F-01 — Non-atomic worktree inspection: symlink-swap TOCTOU + unbounded hash open (Medium)

1. **Severity:** Medium (calibration in §1).
2. **Invariant:** a checkpoint manifest binds the bytes actually present; Studio never validates resume against unexamined state (prompt26 §4, §7 central invariant).
3. **Location:** `server/gitworktree.go:293-328` (`hashWorktreeFile`: `os.Lstat` at :295, `os.Open` at :309), `manifestEntries` at :180-262 (five sequential git subprocesses + N file reads, no snapshot).
4. **Root cause:** type check and content open are two syscalls with no `O_NOFOLLOW`, no fd-relative revalidation, no timeout/context on hashing; the whole inspection is a non-atomic sequence.
5. **Scenario:** a concurrent local process swaps `victim.txt` (regular, listed by `git status`) for a symlink to an outside file in the Lstat→Open window; the manifest records the outside file's digest as `victim.txt`'s. Swapping in a FIFO instead hangs `os.Open` forever — no context, no timeout — wedging the capture/resume request.
6. **Evidence:** strace proof of the window (`newfstatat(...AT_SYMLINK_NOFOLLOW)` then `openat(...O_RDONLY|O_CLOEXEC)`, no `O_NOFOLLOW`); 0/1897 live triggers (window is nanoseconds); skew probe shows non-atomicity effects are otherwise fail-closed (torn reads mismatch at resume). Static symlinks are safe (P2 fixture).
7. **Impact/boundary:** manifest-integrity + availability, local threat model only. No exfiltration (only hashes persist; error strings carry paths, never content). Post-validation modification by a fully-privileged local attacker dominates this primitive, which bounds its marginal severity.
8. **Remediation:** open with `O_NOFOLLOW`, `fstat` the fd and re-check regular/size from the fd (not the path), thread `ctx`/timeout through hashing, fail closed on any violation. Document residual point-in-time semantics.
9. **Regression test:** deterministic symlink-final-component test (must record note, never digest); FIFO-swap refusal or timeout test; keep the hammer as best-effort.

### F-02 — assume-unchanged / skip-worktree content changes are invisible (Medium)

1. **Severity:** Medium.
2. **Invariant:** manifest digest binds tracked worktree content; `clean=true` means content matches.
3. **Location:** `server/gitworktree.go:98` (status parsing trusts `git status` completeness) + `manifestEntries`.
4. **Root cause:** content flagged assume-unchanged/skip-worktree never appears in `git status`, so it is never hashed; nothing detects the flags.
5. **Scenario:** `git update-index --assume-unchanged f` (or a sparse checkout), modify `f`, capture → digest stable, `clean=true`, bytes differ (fixture P4). Resume validates against unexamined bytes.
6. **Evidence:** demonstrated live (digest stable + clean with different bytes on disk).
7. **Impact/boundary:** silent resume validation over wrong bytes; realistic via sparse-checkout workflows or a malicious agent command.
8. **Remediation:** run `git ls-files -v`, refuse (or explicitly record + require reconcile for) lowercase-flagged paths.
9. **Regression test:** fixture asserting capture refuses or flags assume-unchanged content.

### F-03 — Worktree-side submodule content and pointer unbound (Medium)

1. **Severity:** Medium.
2. **Invariant:** same as F-02, for git-tracked submodule paths.
3. **Location:** `server/gitworktree.go:299-301` (non-regular → note, no bytes) + digest input.
4. **Root cause:** submodule paths are directories → always `note="non-regular file"` with empty digests on the worktree side; the staged gitlink SHA is recorded only when the index side is dirty, so two different submodule byte-states (demonstrated: s1 checkout vs s1+local modification, both ` M vendor/sub`) serialize identically.
5. **Scenario:** agent updates vendored submodule content; checkpoint digest identical before/after; resume approves against unexamined dependency source.
6. **Evidence:** demonstrated live (identical `sha256:46b600e2…` across different bytes; entry dump shows empty digests).
7. **Impact/boundary:** silent wrong-content validation for repos with submodules.
8. **Remediation:** record submodule HEAD via `git rev-parse HEAD:<path>` (or refuse dirty submodules with explicit reconcile). Regression: two-content digest-inequality fixture.

### F-04 — Idempotency replay does not verify operation identity (Low)

1. **Severity:** Low.
2. **Invariant:** same opkey replays the same logical operation.
3. **Location:** `server/build_checkpoint.go:366-377` (checkpoint conflict path returns prior row without checking build/attempt/payload match); `server/build_checkpoint.go:553-560` (resume shares the `"start:"+opkey` namespace with `StartAttempt`).
4. **Root cause:** replay keys on opkey string alone.
5. **Scenario:** deliberate opkey reuse across builds (or across start vs resume forms) returns a misleading "already recorded/resumed" naming another build's checkpoint/attempt. No state is written on replay paths (verified), and random 128-bit opkeys make accidental collision impossible — impact is operator confusion only.
6. **Evidence:** code trace; no execution needed.
7. **Impact/boundary:** console UX honesty, operator-confusable only.
8. **Remediation:** verify build (and op intent, e.g. separate `resume:` namespace) on the conflict path; 409 on mismatch. Regression: cross-build same-opkey test.
9. **Blocks:** no.

### F-05 — Ignored files invisible to manifest; no tripwire or documented boundary (Low)

1. **Severity:** Low.
2. **Invariant:** resume validation accounts for behavior-affecting workspace state.
3. **Location:** `server/gitworktree.go:98` (`--untracked-files=all` still respects `.gitignore`).
4. **Root cause:** ignored paths never enter the manifest by design, but no tripwire or documented boundary exists.
5. **Scenario:** operator edits `.env` (database URL) between capture and resume → identical digest → resume "validated" against a different effective config (fixture P5 demonstrated digest stability).
6. **Evidence:** demonstrated (stable digest across `.env` rewrite).
7. **Impact/boundary:** stale-config resume; bounded to ignored paths, which are definitionally outside git state — hence Low, not Medium.
8. **Remediation:** document the boundary in the runbook + optionally record an ignored-file count tripwire (`status --ignored --porcelain` count) that forces reconcile on change.
9. **Blocks:** no.

### F-06 — Untrusted git strings persist and render unscrubbed (Low)

1. **Severity:** Low.
2. **Invariant:** secrets never persist in checkpoint/UI/agent-context surfaces (prompt25 §4.1/§6).
3. **Location:** branch/HEAD/paths flow unscrubbed through `InspectWorktree` → checkpoint row + event payload → templates + `build.packet` branch hint. (Note/agentContext *are* scrubbed pre-persist — verified.)
4. **Root cause:** scrubbing applied to operator prose only, not to git-sourced strings.
5. **Scenario:** branch or filename containing a secret-shaped span (malicious repo content or careless naming) persists verbatim into `checkpoints`, renders in Console, and can enter agent context via packet/branch references. Demonstrated: canary branch preserved verbatim in `WorktreeState`.
6. **Evidence:** fixture P8b (verbatim preservation); template `{{.branch}}` rendering (autoescaped — no XSS, secrecy-only issue).
7. **Impact/boundary:** secret persistence + display + agent-context exposure for attacker-influenced names.
8. **Remediation:** `RedactSecretSpans` on branch/paths at persist time (comparison-safe: deterministic both sides, capture and resume scrub identically). Regression: canary-branch/filename test asserting stored redaction.
9. **Blocks:** no.

### F-07 — `waiting_user` builds refused with wrong message; exclusion undocumented (Low)

1. **Severity:** Low.
2. **Invariant:** honest, documented state-machine boundaries.
3. **Location:** `server/build_checkpoint.go:65-67` (`lockOpenBuild`), mirrored in `server/build_attempt.go:190-192` (B0-consistent).
4. **Root cause:** `waiting_user` (non-terminal) is refused with "Build is terminal".
5. **Scenario:** hybrid flow (gateway asked a question) → local checkpoint/interrupt/start/resume/record all 400 with a message that misstates the state. No unsafe transition is possible (fail-closed direction), and cancel remains available as the escape hatch — but an operator cannot distinguish "terminal" from "gateway-owned, cancel to proceed".
6. **Evidence:** code trace; consistent across B0/B1 paths (compatibility held, message wrong in both).
7. **Impact/boundary:** operator confusion; no unsafe state reachable.
8. **Remediation:** accurate message + one runbook line documenting that gateway-driven (`waiting_user`) builds are checkpoint-ineligible locally. Regression: message assertion test.
9. **Blocks:** no.

### F-08 — `repo_path` recorded but never compared on resume (Low)

1. **Severity:** Low.
2. **Invariant:** mismatched workspace references must be detected (prompt26 §6 attack list).
3. **Location:** stored at `server/build_checkpoint.go:363`, never read in `ResumeAttempt` (`:532-615`).
4. **Root cause:** provenance-only field with no validation use.
5. **Scenario:** workspace moved/reconfigured between capture and resume to a byte-identical checkout → resume validates (digest matches) against a different path than recorded, silently. Content-identical so harmless in practice; the gap is a missing check the prompt explicitly requires, plus operator confusion about which checkout was validated.
6. **Evidence:** code trace (zero reads outside display paths).
7. **Impact/boundary:** provenance honesty; no corruption path (digest still gates content).
8. **Remediation:** compare normalized `repo_path` with explicit-reconcile escape, or document provenance-only status. Regression test either way.
9. **Blocks:** no.

### F-09 — Manifest digest serialization is ambiguous (Low, theoretical)

1. **Severity:** Low.
2. **Invariant:** distinct states → distinct digests.
3. **Location:** `server/gitworktree.go:475-486` (`|`-joined fields, `\n`-joined entries; paths/notes unescaped).
4. **Root cause:** no field framing; a path containing `|`/`\n` (legal on Linux) can shift serialization.
5. **Scenario:** contrived filenames required on both sides of a comparison; no practical exploit constructed (states are compared across time, not attacker-chosen pairs, and git paths with newlines are vanishingly rare).
6. **Evidence:** mechanism by inspection; NOT demonstrated (stated as theoretical per review discipline).
7. **Impact/boundary:** digest collision requires adversarial filenames at both capture and resume.
8. **Remediation:** length-prefix fields or per-entry JSON array hashing.
9. **Regression test:** newline/pipe filename fixture asserting distinct digests.
10. **Blocks:** no.

### F-10 — `status.renames` config sensitivity can cause false refusals (Low)

1. **Severity:** Low.
2. **Invariant:** deterministic manifest for a given tree state.
3. **Location:** `server/gitworktree.go:98` (no `-c status.renames=true` pin; contrast `LC_ALL=C` which is pinned).
4. **Root cause:** rename detection follows user/system git config; same tree yields R-entry vs D+A-entry serializations (different digests) under different settings.
5. **Scenario:** operator toggles git config between capture and resume → spurious mismatch → refuse + reconcile flow. Fail-closed direction; availability/cosmetic only (false accept impossible: different serializations never collide).
6. **Evidence:** code trace + git semantics.
7. **Impact/boundary:** spurious resume refusals.
8. **Remediation:** pin `-c status.renames=true` (and audit `core.quotepath`-style knobs; `-z` already neutralizes quoting).
9. **Blocks:** no.

### F-11 — Workbench listing silently skips unreadable entries (Low)

1. **Severity:** Low.
2. **Invariant:** display honesty (operator sees what exists).
3. **Location:** `server/gitworktree.go:454-457` (`item.Info()` error → `continue`).
4. **Root cause:** read errors drop entries without marker (inconsistent with the entry-cap path, which fails closed).
5. **Scenario:** unreadable file vanishes from the workbench view silently. Display-only (resume validation uses the manifest, never the listing).
6. **Evidence:** code trace.
7. **Impact/boundary:** incomplete operator view; no safety impact.
8. **Remediation:** surface a `skipped` count/marker. Trivial test.
9. **Blocks:** no.

## 5. Detailed Git/filesystem security findings

Covered by F-01 (TOCTOU + skew + hang), F-02 (assume-unchanged — see §6), F-03 (submodules — see §6), F-05 (ignored files), F-09 (serialization), F-10 (config), F-11 (listing). Additional verified-safe points, stated so the absence of findings is evidence, not omission:

- **Static traversal:** `checkRepoPath` (absolute/`..` rejection) + git-generated paths only; verified against `../`, absolute, backslash-literal (Linux), and NUL-impossible (`-z` framing) cases. Intermediate symlinked parents cannot arrive except via the F-01 race (git never emits paths through symlinked dirs — verified by design + P2 fixture).
- **Static symlinks:** recorded as `non-regular file`, never followed (P2 fixture: `link.txt` note-only while `real.txt` hashed normally).
- **Bounds:** 4 MiB/file, 64 MiB total, 5,000 entries, 8 MiB git output, 30s git timeout — all verified in code; total-budget accounting survives racy size changes (bounded ≈68 MiB worst case by counter arithmetic); caps refuse, never truncate (rename-missing-source, over-cap output, and entry-cap paths all error).
- **No shell, allowlisted subcommands** (`gitRun` switch), `LC_ALL=C`, `GIT_OPTIONAL_LOCKS=0`, no `reset`/`clean`/checkout/mutation primitives anywhere in the file (verified by search).
- **Workspace root:** absolute + dir-validated at config load; per-call `EvalSymlinks` (no stale cached resolution); inside-work-tree + top-level enforcement. Residual parent-dir-swap noted as Info (narrow, read-only impact, confused-which-repo at worst).
- **Unborn/detached/mode/unmerged/conflict states:** covered by passing tests (executed) plus code trace; unmerged paths hash worktree bytes with markers (deterministic, honest).
- **show-ref fallback:** correct direction; timeout-misclassified-as-unborn is Info-level (contrived, both sides consistent).
- **No-mutation:** `TestInspectNoMutation` passes; all commands + hashing paths read-only by construction (verified: no `O_CREATE`/write calls outside test fixtures).

## 6. Checkpoint and resume findings

Covered by F-04 (replay identity — wait, renumbered: replay finding is §4 item on idempotency; see mapping below), F-07 (waiting_user), F-08 (repo_path), F-02, F-03. Verified sound:

- Tenant/build/attempt binding enforced in SQL (`WHERE tenant_id`, composite FKs, unique `(tenant,build,seq)`), RLS ENABLE+FORCE + least-privilege grants in migration 000024 (round-trip covered by the dynamic-version down test).
- Latest-complete selection deterministic (`complete` filter + `seq DESC` + unique constraint); incomplete rows can never be written by the app (always `complete=true`) and are excluded anyway (test inserts the adversarial row raw).
- Seq allocation under the build-row lock (concurrent uniqueness covered by `TestConcurrentCheckpointSeq` — reviewed, not executed: DB unavailable).
- Replay paths write nothing; opkeys are 128-bit random hex validated to 32–128 chars; event IDs namespaced per operation family except the filed start/resume namespace-sharing note (folded into the replay finding).
- Resume validates latest-match (superseded 409), full digest diff (refuse-or-reconcile with mandatory note), open-attempt exclusion, and links the new attempt to its checkpoint; nothing is ever reset/cleaned/overwritten (verified: no such primitives in any B1 path).
- Crash/restart behavior: sweep-free design (rows are explicit; orphans impossible by construction — claims settle in-txn); `TestProgressSurvivesRestart` + post-restart resume reviewed (not executed — DB).
- B0 compatibility preserved: `build.packet` untouched, `RecordAttempt` opkey idempotency intact (additive `source_checkpoint` only), numbering/parentage/terminal semantics unchanged, held-assign non-burning sender path untouched by B1 (sender diff is Phase-D fetch work only), no dispatch triggers added by B1.

## 7. Concurrency and persistence findings

No concurrency defect found by inspection; execution-dependent claims explicitly not re-run:

- Lock order verified globally acyclic: build `FOR UPDATE` → turn → question → conversation `KeyShare` → advisory xact lock → fresh inserts, in every path (`Start/Interrupt/Create/Resume/RecordAnswer/ingest/messageSend` + both `AppendStream` implementations); the only other advisory user (`build_provision`) uses a disjoint key domain. The U3 deadlock fix is correct and complete for all traced callers.
- Single-flight semantics: build-row serialization makes same-opkey concurrent submits safe (replay, no dupes); `UNIQUE(tenant,build,seq)` + `ON CONFLICT DO NOTHING` + PK make double-persist impossible at the DB layer, not just the app layer.
- Crash windows: claim-then-settle is single-txn per op; sweep/replay paths write nothing on conflict; shutdown mid-dispatch is a sender concern (unchanged, out of B1 scope).
- **Not executed:** `TestConcurrentCheckpointSeq`, `TestConcurrentStartOneWins`, race detector runs on DB paths, migration round-trip, RLS/parity suites — code-reviewed (tests assert genuine invariants: unique seqs under 8 writers, single-winner starts, catalog-derived parity) but not runnable here. No concurrent-*resume* test exists (coverage gap, §10).

## 8. Secret-scrubbing and tenant-isolation findings

- Scrub-before-persist verified on every B1 free-text path (checkpoint note/context, interrupt note; B0 attempt fields pre-existing). Except F-06 (git-sourced strings), no unscrubbed persistence found.
- Scrubber itself (`artifact/secrets.go`, shared table — verified byte-identical patterns to the pre-existing `tools` table it consolidates): documented non-exhaustive by design; base64/encoded-span evasion passes through as a *stated* limitation, not a defect. Encoding-evasion robustness is correctly out of scope for B1's acceptance (would fail the "do not invent scope" test to demand it here).
- Tenant isolation: session+Origin on all B1 POST routes (verified in `routes.go`), server-side tenant from session (never form input), tenant predicate on every checkpoint/event/stream query, RLS FORCE as backstop, grants least-privilege, test-role parity list extended (code-verified; suite not run). No agent-tool registration for checkpoint ops (registry diff contains only B0/Phase-D tools) — operator-only boundary holds.
- UI: `html/template` autoescape throughout; `template.HTML` uses are static strings + `HTMLEscapeString` parts only; no JS changes in B1 (SSE reuse); DEV surfaces show no checkpoint secrets (context/notes never rendered; branch/HEAD/digests only — covered by F-06 for the secret-shaped-name edge).

## 9. B0 compatibility assessment

Preserved on every prompt26 §11 axis, verified by diff + behavior trace (B0 DB suites not re-runnable here; noted): `build.packet` v0 untouched; `RecordAttempt` opkey idempotency + numbering + parent linkage intact (B1's `source_checkpoint` inheritance is additive and conditional); accept/reject/rework + terminal semantics unchanged; scrubbing intact (shared-table refactor verified pattern-identical); tenant isolation intact; manual bridge flow untouched; held Gateway `assign` non-burning sender path untouched (B1 adds no dispatch triggers; the only sender diff is unrelated Phase-D fetch work); no second engine/store/pipeline/shell introduced (single `store.AppendStream` + tools twin with identical lock order; git via direct exec, allowlisted read-only subcommands; no terminal). Studio PG remains authoritative; worktree stays execution state.

## 10. Coverage gaps and untested scenarios

1. **DB-backed suites not executed by this review** (environment: no PG reachable, escalation unavailable): all of `build_checkpoint_test.go`, DB migration/RLS/parity tests, race runs on DB paths, full `go test ./...` with a database. Code-reviewed instead; the B1 evidence report's green claims are *consistent* with (not independently reproduced by) this review. Highest-value re-run before B2: `TestConcurrentCheckpointSeq`, resume tests, parity test.
2. **No symlink/traversal fixture tests** in `gitworktree_test.go` (verified by search): static-symlink, symlink-dir, `..`-rejection, and confused-deputy paths are untested. Required with F-01's fix.
3. **No concurrent-resume test** (two simultaneous resumes / resume-vs-checkpoint race). Design serializes correctly; prove it.
4. **No per-route auth sweep test** for the four B1 POST routes (middleware verified by reading only). Recommend a route-table test asserting session+Origin on every mutating console route.
5. **Live TOCTOU trigger unobserved** (0/1897 + strace mechanism proof). The regression test for F-01 should assert the *mechanism* (O_NOFOLLOW open, fd-based fstat) deterministically, not win the race.
6. **Ignored-file tripwire, submodule HEAD binding, assume-unchanged detection**: no tests exist because the behavior doesn't; each ships with its finding's fix.

## 11. Prioritized remediation recommendations

1. **F-01 (TOCTOU):** open with `O_NOFOLLOW`, `fstat` the fd, re-check regular + size from the fd; add ctx/timeout to hashing; keep fail-closed errors. Add deterministic symlink tests.
2. **F-02 (assume-unchanged):** `git ls-files -v` tripwire; refuse-or-record flagged paths. Add fixture test.
3. **F-03 (submodules):** record worktree submodule HEAD (`git rev-parse HEAD:<path>`) or refuse dirty submodules with reconcile path. Add two-content fixture test.
4. **F-04 + F-07 + F-08:** one-line-to-small fixes (replay identity check with separate resume namespace; accurate waiting-state message + doc line; repo_path compare-or-document).
5. **F-05/F-06/F-09/F-10/F-11:** scrub-before-persist for git strings; ignored-file boundary doc (+ optional count tripwire); length-prefix digest framing; pin `status.renames`; surface listing skips.
6. **Verification:** re-run DB suites + race with PG access (covers §10.1); add §10.2–10.4 tests.
7. **Before B2:** all of the above resolved or explicitly risk-accepted by the owner; B2 must re-audit resume validation anyway since remote workspaces change every filesystem assumption in §5.

## 12. Final B1 acceptance recommendation

**CONDITIONAL ACCEPTANCE.**

No Critical findings. No High findings. The implementation is careful, well-tested where runnable, honest in its evidence, and architecturally compliant — but three Medium findings (F-01, F-02, F-03) defeat silent resume validation in realistic configurations (racing local process, sparse checkout, submodules), and the DB-backed suites could not be independently re-executed in this session. B1 is accepted for local use conditional on: (a) remediating F-01–F-03 (or explicit owner risk-acceptance with runbook warnings), (b) fixing or formally carrying F-04–F-11, (c) an independent green DB-suite + race run with PG access, and (d) the coverage tests in §10.

B2 is not authorized by this review. Remote execution invalidates the local-filesystem assumptions underlying §5; B2 needs its own adversarial review regardless.
