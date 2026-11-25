# Scoped Adversarial Review — Prompt 12 Residual Closure

Read-only verification of the residual-closure pass against
`plans/prompt14.md`. No code, tests, docs, or configuration modified.
No live operations performed. Part 2 remains unauthorized.

Note: prompt14 §13 names this deliverable
`docs/adv-review-prompt12-residual-closure.md`; it was ordered as
`plans/adv_review_prompt14.md`. This file is that report.

## 1. Executive verdict

**ACCEPTED WITH BOUNDED RESIDUALS.**

All four closure prerequisites are independently verified: N1's
boot-time credential-drop self-test is on the true production path
with no bypass and no production escape hatch; N4's identifier
mapping is accurate against both original sources with v1 bannered
and v2 authoritative; R-DB reproduced green on PG19 (13/13 suite,
six-package race, all targeted F7/M15/migration sets, zero
inappropriate skips); R-NET hashes recomputed independently and
match exactly (tag → commit → tree → bytes). The N2–N10 register is
technically accurate with restrictions correctly placed for N2.

Bounded residuals requiring explicit tracking: two documentation
tightening items (N8's restriction is register-only, not yet in the
runbook at the point of use; N5's cold-path PID-absence check is
implied, not explicit), two trivial test/count precision items, and
the disclosed `TestAdapterStatus` flake, which I reproduced, root-caused,
and confirm is pre-existing, test-only, and non-blocking. None reopens
architecture; none blocks Part 1 acceptance.

## 2. Exact Git tree and commit state

- Branch `main`; HEAD `22ce9c476eb97fa07991e85d9667273af76c182c`.
- `2d020a3` = `2d020a357f98b1179b3753982a99906b8b74fede` (root):
  "Initial commit", Isagani Mendoza, 2026-11-25 08:39, LICENSE only.
- `22ce9c4` = `22ce9c47...` (parent `2d020a3`): message "prompt12.md",
  ibmendoza, 2026-11-25 12:05, bulk tree commit.
- Pre-review `git status --short`: 8 modified tracked files + 5
  untracked (`docs/prompt12-residual-closure-report.md`,
  `plans/prompt13.md`, `plans/prompt13_imp.md`,
  `plans/prompt13_plan.md`, `plans/prompt14.md`). Post-review state
  identical plus this report file. Zero Git writes performed.
- The claimed "seven modified + one new" is exact **for the
  residual pass**: the 8th modified file, `plans/prompt12.1.md`
  (committed empty, 522 lines uncommitted, mtime 12:18 — before all
  implementation files at 12:26+), is the task directive the pass
  executed against, consistent with the report's §2 disclosure.
  `plans/prompt13*.md`/`prompt14.md` are unrelated working-tree
  material from other sessions (planning + this review's own
  directive), correctly excluded. No deletions, no renames.

## 3. Changed-file inventory and scope assessment

| File | Why / residual | Matches report | Unrelated scope |
|---|---|---|---|
| `cmd/gateway/main.go` (+39) | N1: `checkAgentDistinct` + `checkAgentSpawn` + serve wiring | Yes, exact | None |
| `cmd/gateway/main_test.go` (+192, 7 new tests) | N1 regression tests | Yes (count verified: 10 total − 3 at HEAD) | None |
| `gateway/protocol.go` (2 comment lines) | M16/N4 hygiene (stale Phase-A wording) | Yes | None |
| `docs/gateway-phase-b-evidence.md` | N1 contract, N4 mapping + v1 banner, N2–N10 register | Yes, verified line by line | None |
| `docs/gateway-phase-b-live-runbook.md` | §7a, §7b rework, §6.5 N2 banner, M5-SSE | Yes | None |
| `docs/prompt11-correction-report.md` | N1 contract §7, M5-STDERR language | Yes | None |
| `plans/prompt11_plan.md` | N1 ownership wording, M5-STDERR note | Yes | None |
| `docs/prompt12-residual-closure-report.md` (new) | The pass's evidence report | N/A (deliverable) | None |

No production dependency added (`go.mod` untouched), Miniapp
untouched (separate repo, no changes), no CI/workflow edits, no test
edits outside `cmd/gateway/main_test.go`. Owner commits correctly
attributed (both predate the pass; bulk commit message "prompt12.md"
is opaque but outside review scope). Scope is clean.

## 4. N1 security and privilege self-test assessment

**Status: VERIFIED CLOSED** (with two trivial test-coverage notes).

Production path traced end to end (`cmd/gateway/main.go`):
`run()` → flag/config load → `serve()` → keys/sessions/outbox/dedup
open → `resolveAgent` (refuses unknown user/group) → supervisor with
production credential → `checkAgentDistinct` (UID≠EUID or boot
refuses) → `checkAgentSpawn(ctx, runner, "true")` (real `Start`
through the production supervisor, real `Wait`, exit-0 required) →
adapter → `RecoverSessions` → transport check → deliverer → listen.
Findings against prompt §5.1:

- Self-test runs before recovery, deliverer, and listen; single call
  sites for both checks, both inside `serve()`; `serveOn` is reached
  only from `serve()` in non-test code. No bypass path exists.
- The check uses the **same `Supervisor` object** later wired into
  the adapter — exact credential, UID, GID, and supplementary groups
  (verified in `serve()` body). This is a real spawn, not a
  `SysProcAttr` value assertion.
- Probe is the fixed string `"true"` at the production call site;
  parameterization exists only for test observability.
- `resolveAgent` failures, EPERM, and same-UID all refuse boot with
  non-secret diagnostics (UID numbers + OS error text; no keys).
- `AllowSameUID` is set `true` only in the two test helpers
  (`cmd/gateway/main_test.go`, `gateway/supervisor_identity_test.go`);
  `resolveAgent` never sets it (asserted by test), and no flag or
  env var can enable it. Escape-hatch containment is airtight.
- Non-Linux refuses via `proc_other.go` (compile-verified pattern;
  Darwin build gate covers compilation).

Tests (8 in file, 7 new — `TestResolveAgentRefusesUnknown`
predates at HEAD): distinctness refusal/pass, EPERM refusal with
diagnostic substrings, root-only real drop (correctly skips
unprivileged), identity plumbing via `id`, and two end-to-end
`run()` refusal tests proving no readiness bypass. Executed:
7 pass, 1 principled skip. Gaps (minor, noted not blocking):
no nonzero-exit probe test and no unwaitable-probe test for the
`code != 0` branch. The serve-success path past the checks is
untestable without privilege (inherent; live §7b covers it).

Permission-matrix/runbook consistency (§5.3): the corrected matrix
(service-EUID-owned gateway paths, setup-pinned keys/binary,
`studio-io` meeting point) matches `SecureDir`/`writeFileComplete`
(chgrp/chmod, never chown owner) and `setup.sh` exactly; §7a derives
EUID numerically from `/proc`; §7b compares numerically and keeps the
kernel-observed EACCES probes with the agent-report fallback
explicitly labeled weaker. The privilege contract (privileged
reference service; anything else must prove the full uid+gid+groups
transition live) is now stated identically in all three documents.
Live key-denied proof correctly remains Part 2.

## 5. N4 documentation and M5 mapping assessment

**Status: VERIFIED CLOSED.**

- Mapping verified against both originals: `plans/prompt10.md:480`
  defines M5 as SSE lifetime; `plans/adv_review_prompt10.md:87`
  reused M5 for the stderr ring. The note's collision account is
  accurate (including that the reuse originated in the prompt10
  review, not the correction).
- v1 carries an unmistakable SUPERSEDED banner placed immediately
  before its header, naming v2 + correction report as governing.
- v2 identified as current authoritative evidence; "prompt11 bundle
  all fixed" + M5-SSE exclusion applied everywhere completion is
  claimed (evidence, correction report, plan note).
- The one present-tense straggler is scoped in place; the remaining
  v1 present-tense rows sit visibly under the banner as audit
  history — acceptable, not misleading.
- SSE lifetime + concurrent-stream budget remain explicitly deferred
  (runbook §15 as `M5-SSE`, matrix row intact).
- Bare-`M5` audit: every remaining occurrence accounted for —
  mapping-note self-references, the §2 historical line (now
  annotated), the residual report's N4 row, prompt12.1 directive
  text, and the separate `docs/miniapp/*` namespace (Miniapp
  lifecycle M5 — different project, explicitly out of scope, valid
  to leave). Zero ambiguous references in current gateway docs.

## 6. R-DB independent PG19 verification matrix

Environment proven first: container `miniapp-p0-pg`,
`SHOW server_version` → `19.0 (Debian 19.0-1.pgdg13+1)`,
`SELECT version()` → PostgreSQL 19.0 x86-64. Databases present:
`miniapp_p0` (51 user tables, prior test residue) + `postgres`;
no user-data databases. All runs below used `miniapp_p0` via
in-shell-derived DSN (never printed); suites self-manage scratch
DBs + test-schema truncation per design.

| Check | Independent result |
|---|---|
| Full suite `go test -p 1 -count=1 ./...` | **13/13 ok, exit 0** (reproduced) |
| Six-package race | **all 6 ok, exit 0** (reproduced; race detected nothing) |
| Verbose skip audit | **355 pass, 1 skip** (`TestCheckAgentSpawnRealDrop`, root-only by design), **0 fail** |
| F7 set (16 tests: provision×7, F7×6, upgrade×3) | **16/16 PASS, 0 skip**, incl. `TestBuildProvisionSerializesAcrossSessions` (0.63s; genuine separate pools + delayed observer asserting max-in-flight 1 — inspected, not just named) |
| M15 (3 tests) | **3/3 PASS** — bodies read: full server boot with org secret → KDF-signed callback 200+ack; wrong-secret derived key → 401. Genuine wiring + negative coverage |
| Migration/RLS family | **20/20 PASS, 0 skip** across store+server (migrate idempotent/round-trip/framework-gate, RLS enable, Phase A RLS, tenant isolation/context/write/rollback, cross-tenant reference/cascade/status, pool reset, session lifecycle/expiry/provisioning, direct-read denial, role attributes) |
| gofmt / vet / build | clean / 0 / 0 (reproduced) |

Count notes (precision, not defects): the report's "14 F7 tests" is
the DB-backed subset of the 16 tests in those files (the 2 pure
ones — bundle-pinned, shortID — also pass); the "15
migration/RLS" label fits the strict-pattern subset of the 20 I
executed (all green either way — the substance claim holds under
every reading); "354 pass" undercounts my 355 by exactly the flake
failure in the report's own audited run (354+1 fail+1 skip = 356 ✓
reconciled, and consistent with the disclosed flake).

**R-DB: VERIFIED CLOSED.** Every number claimed is reproducible to
within documented counting-method differences, and I found no
skipped DB test anywhere.

## 7. R-NET provenance and hash verification

Independently recomputed (fresh fetch; bytes kept in /tmp only,
nothing copied into the repo):

- Tag `v1.19.6` → lightweight tag → commit
  `f4e363c8295fa90a03dddc451882fffe8fbce357` ✓ matches report.
- Tree entry `packages/opencode/src/cli/cmd/run.ts`: blob
  `944aec17abd75cc3272688e89a140b56cd2bb2d3`, 31940 bytes ✓.
- Downloaded 31940 bytes; SHA-256
  `e040e90d4c81cdd59b9568ee90a53c0209cd9ddf1b8b207d1a8685b2e26d57d0` ✓.
- Git blob SHA-1 computed over `blob <len>\0` + bytes:
  `944aec17abd75cc3272688e89a140b56cd2bb2d3` ✓ matches record
  and tree entry.

**R-NET: VERIFIED CLOSED.** The report's §8 record is exact in
every field including the lightweight-vs-annotated distinction.

## 8. N2–N10 residual/restriction matrix

Register accuracy verified against code + runbook placement:

| ID | Register accurate | Restriction effective | Notes |
|---|---|---|---|
| N2 | Yes (TOCTOU + empty-URL skip both real, verified in `build_upgrade.go`) | **Yes** — §6.5 banner sits directly above the only documented invocation; §12 rows reiterate; accidental use requires skipping the section containing the command | Inherent limit: procedural only; tool itself still runs. Adequate as tracked-with-restriction |
| N3 | Yes | N/A (wording fix; tracked) | Message texts verified in server.go |
| N4 | Closed (see §5) | N/A | — |
| N5 | Yes (pid-after-capture verified in code) | Partial — evidence register requires PID record + absence verification, but runbook §11's cold path never explicitly says "verify pre-restart agent PIDs are absent" (finding F-R2) | Needs one explicit step |
| N6 | Yes (unbounded head loop verified) | N/A (disclosure; tracked) | — |
| N7 | Yes (`_ =` transitions verified) | N/A (tracked) | — |
| N8 | Technically accurate | **No** — the "prohibition" exists only in evidence/register docs; nothing at the §6 re-provision point of use warns the operator (finding F-R1) | Re-provision is a routine documented op, unlike upgrade |
| N9 | Yes, and correctly framed (server sem is the relied-upon boundary; adapter has no claim) | N/A (tracked) | — |
| N10 | Yes | N/A (tracked) | — |

## 9. `TestAdapterStatus` flake investigation

**Reproduced**: 0/15 clean runs, then failure within 5 race runs:
`TempDir RemoveAll cleanup: ... /sessions: directory not empty`.
Race widens the window (slower goroutine scheduling), as expected
for a cleanup race.

**Root cause (traced, matches the report exactly):** `Adapter.Cancel`'s
kill-escalation path returns right after `Kill()` without awaiting
pump drain; the adapter pump then performs its terminal
`Sessions.Update` (session-file write) concurrently with the test's
`TempDir` removal. The failing syscall is on the `sessions`
directory — precisely the pump's write, not a stray artifact.

**Classification: pre-existing, test-only, non-blocking residual.**
Justification: (1) the test, adapter `Cancel`, and pump paths are
prompt11-era — `git diff` proves none were touched by this pass;
(2) TempDirs are unique per test — no cross-test contamination;
(3) the double is scripted — no leaked process (nothing to reap);
(4) it fails **loudly** (cleanup error fails the test), so it can
neither false-green the suite nor conceal a broken invariant —
every green run I report genuinely passed.
Production-lifecycle impact: none beyond a negligible ordering
observation (pump's terminal write vs server's cancelled mark can
interleave; both terminal, Studio never continues cancelled builds).
It does not undermine the verification result; re-runs are valid.

## 10. Confirmed defects and documentation discrepancies

Severity scale per prompt §8. No Critical/High/Medium items. Three
Low items, all bounded:

**F-R1 — N8 restriction not operator-visible (Low/doc).** The
evidence register claims re-provisioning beside untrusted siblings
is prohibited, but the runbook's §6 re-provision path carries no
banner or check, unlike N2's §6.5 treatment — yet re-provision is
routine while upgrade is exceptional. Correction: add the N8
constraint beside the §6 second-build refresh note (verify staging
`0700` + no live untrusted siblings, or drain first). Regression:
doc-review checklist item; no code test applicable.

**F-R2 — N5 cold-path absence check implicit (Low/doc).** Runbook
§11 records PIDs and warm/cold outcome but never explicitly
requires verifying pre-restart agent PIDs are absent after cold
recovery — the exact proof N5's orphan residual needs.
Correction: one explicit §11 step ("list pre-restart agent PIDs;
confirm absent; report orphans"). No code test applicable.

**F-R3 — Report count imprecision (Low/evidence).** "14 F7 tests"
omits the 2 pure tests in the same files without saying so, and the
"15 migration/RLS" membership is undefined (my strict-pattern
reconciliation reaches 15; the broad family is 20 — all green
either way). Correction: enumerate exact sets or label "DB-backed
subset". No re-run needed; substance verified.

Not filed (deliberately): the two N1 micro test gaps (nonzero-exit
probe, unwaitable probe) — 3-line fail-closed branches whose refusal
behavior is fully covered at the EPERM/distinctness/e2e level;
adding them is optional hygiene, not a defect. The pre-existing
flake is classified in §9, not as a new defect.

## 11. Tests and checks not run, with exact reasons

- Live proofs (R1–R10, EACCES, setup.sh, OpenCode, OAuth, wake):
  unauthorized Part 2 — not run by design; runbook homes verified
  to exist with observations + stop conditions.
- `TestCheckAgentSpawnRealDrop` positive path: requires root for a
  genuine uid transition; correctly skipped unprivileged here (its
  refusal halves all executed).
- Per-test DB re-verification beyond the sets above: covered by the
  full suite + race runs instead (superset).
- Nothing else was skipped: every runnable gate, count, hash, and
  probe in the prompt was executed.

## 12. Final acceptance recommendation

**ACCEPTED WITH BOUNDED RESIDUALS.**

N1, N4, R-DB, and R-NET are independently verified closed; N2–N10
are correctly tracked with N2's restriction verified effective;
the flake is demonstrated pre-existing, test-only, and loud. The
bounded residuals: F-R1 + F-R2 doc lines before Part 2, F-R3 count
wording, the carried N2/N5/N8 fixes at their point of use, and the
unchanged Part 2 live gates (which now additionally require no new
general review to begin — verification of the two doc lines
suffices).

- No live Sprite/Fly/OAuth operations were performed.
- Part 2 remains unauthorized.
- No source, tests, plans, or documentation were modified by the
  reviewer, other than the review report itself.
- No commits or pushes were made.
