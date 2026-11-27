# Final Focused Verification — Host Execution Broker (prompt6)

## Executive verdict: READY FOR INSTALLATION

All five corrections verified effective with reproduced evidence.
No unresolved material implementation blocker remains. The P5-HIGH-1
build argv now names the staged Containerfile explicitly; P5-HIGH-2
anchors are first-wins with delete/resubmit rejected and
verify-blindness eliminated; P5-MED-1 detail hashes freeze the three
evidence files with explicit gap statuses; sweep wording asserts only
what is known; git provenance reads the executed snapshot.

Remaining notes are three LOWs (acceptance-checklist manifest line,
absolute-path hygiene, partial-anchor sweep wart — details below),
none material to the installation decision.

Readiness is not authorization: this review does not authorize
installation. Real-Docker build proof stays part of installation
(Docker operations were prohibited here), and the P5-MED-2 sandbox
attestation must be performed as install step zero. No further
review cycle is needed.

Verification-only. Nothing modified; no install, Docker,
PostgreSQL, or systemd operations. The `docker` CLI was never invoked.

## Five-finding status

| Finding | Status | Evidence |
|---|---|---|
| P5-HIGH-1 build `-f` | CORRECTED, VERIFIED | Exact argv reproduced; pinned test; docs agree |
| P5-HIGH-2 anchor first-wins | CORRECTED, VERIFIED | §3 attack reproduced step-by-step: rejected, immutable, MISSING-COPY |
| P5-MED-1 detail hashes | CORRECTED, VERIFIED | 10-item matrix: tests + audit + probes (one branch code-only) |
| P5-LOW-1 sweep wording | CORRECTED, VERIFIED | Exact string in code + test; docs consistent |
| P5-LOW-2 snapshot provenance | CORRECTED, VERIFIED | Fake-git wiring test + `.git`-in-stage mechanism probe |

## P5-HIGH-1 — Build argv (§2): PASS all 7 items

Remediated argv (`hostexec/exec.go:254`):

`docker build -f <stage>/Containerfile -t studio-test-runner:1 <stage>`

1. `-f` present. 2. Value is the staged Containerfile
   (`filepath.Join(stage, "Containerfile")`), never the live tree.
3. Context is the verified stage (`StateDir/stage-image`, the same
   bytes `StageContext` hashed). 4. No caller control: the sole
   image-build caller (`serve.go:387`) passes a fixed join; R1 probe
   confirms hostile job params leave the argv byte-identical.
5. Absolute iff StateDir is absolute (deterministic under systemd;
   see LOW note). 6. `TestArgvFixed` pins the complete command and
   `TestBuildArgvFileTarget` pins `-f`/target/context separately.
7. `docs/test-runner.md:57` now shows `-f …/Containerfile`.
R1 probe output: `["docker" "build" "-f"
"/state/stage-image/Containerfile" "-t" "studio-test-runner:1"
"/state/stage-image"]` — exact match. CLI syntax conforms (`-f`
inside context is the standard form). Real-Docker execution was
not and could not be verified here — it is installation's first proof.

## P5-HIGH-2 — First-wins anchors (§3): PASS, attack chain closed

R2 probe reproduced the §3 attack exactly against the new code:

1. Finished job `x` (anchor + hash written).
2. Preserved anchor bytes.
3. Deleted `done/x.json`.
4. Resubmitted id `x` (different op).
5. Claim result: `ok=false err=job x.json duplicate (anchored)` —
   rejected by consulting owner-side anchors, resubmission quarantined.
6. Anchor bytes byte-identical after the attempt.
7. `verify`: `checked=0 problems=[MISSING-COPY x.json …]` — exactly
   one problem, never clean.
8. Planted `running/x.json` + sweep: `swept=[]`, no repo copy
   recreated, anchor untouched — no resurrection.
9. Double-authoritative-outcome audit: `running/` O_EXCL serializes
   claims, anchor O_EXCL serializes records, collision preserves
   under `conflict-` names loudly (`TestAnchorCollisionPreserved`).

Filesystem semantics confirmed: single `O_EXCL` create of the
self-verifying `<hash>\n<payload>` anchor file; no rename-over path
remains. `TestAnchorFirstWins` encodes this attack as regression.
Acceptance met: anchor history is immutable through all
agent-accessible operations.

## P5-MED-1 — Detail hashes (§4): PASS (item 7 code-only, item 9 procedural)

`hashResultsDir` (`serve.go:261-297`) emits exactly one entry per
required file (`gotest.json`, `summary.txt`, `exitcode`), each with
sha256 + size + explicit status (`ok|missing|unreadable|oversized|
irregular`). Reads stream through a 64 MiB cap; `Lstat` never follows
symlinks; `disarm.json` carries the manifest owner-side, first-wins.

1. All three recorded — `TestDisarmProvenance` asserts ok/64-hex/size.
2. Independent hash match — textbook `sha256.New` + `io.Copy` over
   exact read bytes; test asserts stability + tamper-sensitivity.
3. Modification flips the digest — tested (rewrite → new hash).
4. Deletion → `missing` — tested.
5. Symlink → `irregular`, never followed — tested; Lstat/Open
   swap-window analyzed sound (forgery needs a hash preimage;
   only digests enter the owner-side record, so no exfil).
6. Oversized (Lstat size and mid-read growth) → `oversized` —
   tested + branch-audited.
7. Unreadable → `unreadable` — code-verified only (3-line branch,
   no test; same-uid `chmod 000` case).
8. Missing → `missing` with error note — tested.
9. Incomplete manifests block acceptance — procedurally enforced:
   design docs state gaps "must block install acceptance"; the
   acceptance checklist does not yet operationalize it (LOW doc note).
10. Repeat runs preserve the record — first-wins tested; live-dir
    detail overwrite after disarm stays the accepted residual (§9).

Capture-vs-authenticity is honestly represented in code comments and
docs. StateDir record is separate from workspace copies by
construction + MED-2 attestation.

## P5-LOW-1 — Sweep wording (§5): PASS

Reason is now: `residue found at startup for <op> (<n> bytes);
execution state unknown; reconcile via inspect.readonly before any
retry` — asserts only residue presence, validated op/size, and
unknown state. Status stays UNKNOWN; terminal verdicts untouched
(`TestSweepKeepsTerminalVerdict`); unparseable residue quarantined;
wording identical in code and test, accurately paraphrased in docs.

## P5-LOW-2 — Snapshot provenance (§6): PASS all 6 items

`recordDisarm` passes `StateDir/stage-run` (not the repo) to
`gitProvenance`; single-threading guarantees it is this run's bytes.

1. `.git` travels in the snapshot — R5: `git -C stage` works.
2. `-C` targets the stage — R5 mechanism + `TestDisarmProvenance`
   fake-git log asserting `stage-run` present, repo path absent.
3. Fake-git tripwire present and passing.
4. HEAD/status describe the snapshot — R5: stage HEAD == live HEAD
   at snapshot time.
5. Live mutation (new commit) leaves stage HEAD/status unchanged —
   R5 demonstrated.
6. Git failure → `unavailable: …`, empty status → `(clean)` —
   explicit, never fabricated.

## Cross-boundary consistency (§7): PASS

Build evidence ↔ staged context (pins + `-f` + digest record);
run evidence ↔ per-job snapshot (ro mount + manifest on result);
git provenance ↔ same snapshot (stage-run); detail hashes ↔
captured files (disarm manifest); anchors ↔ exact results (O_EXCL
first-wins); id reuse impossible (claim + sweep consult anchors);
incomplete evidence fails acceptance by documented rule (checklist
line pending, LOW). No individually-passing test masks a composed
exploit: the R2 chain exercises claim→finish→delete→resubmit→
sweep→verify end to end.

## New notes (all LOW, non-blocking)

- **P6-LOW-1:** acceptance checklist step 5 should name the
  `results_manifest` all-ok + hash-recomparison check (design
  section already requires it). One doc line.
- **P6-LOW-2:** crash-partial anchor (disk-full/power-loss window)
  makes `SweepStartup` return an error → `Serve` exits → systemd
  restart loop until the owner deletes the partial file (R2
  demonstrated the error; `serve.go:85` confirms abort). No agent
  exploit path (StateDir owner-side). Recommend skip-with-warning
  instead of abort + a recovery doc line. The "first record
  stands" message is misleading when the first record is malformed.
- **P6-LOW-3:** config paths are not required to be absolute, so
  `-f`/context absoluteness inherits StateDir hygiene (R1 shows the
  relative form). Deterministic under systemd; recommend enforcing
  absolute paths in `LoadConfig` + one doc line.

## Regression table (§8)

| Finding | Required outcome | Result |
|---|---|---|
| P5-HIGH-1 | Absolute staged `-f` Containerfile | EXACT (R1 + 2 tests) |
| P5-HIGH-2 | Resubmit rejected; immutable anchor; MISSING-COPY | EXACT (R2 + 2 tests) |
| P5-MED-1 | 3 files hashed; tamper/absence detected | YES (tests + audit; item 7 code-only) |
| P5-LOW-1 | Truthful wording | EXACT STRING |
| P5-LOW-2 | Provenance from snapshot | YES (fake-git test + R5) |

## Test evidence (§8)

| Check | Command | Result |
|---|---|---|
| gofmt | `gofmt -l .` | clean, exit 0 |
| vet | `go vet ./...` | clean, exit 0 |
| build | `go build ./...` | clean, exit 0 |
| full serial | `go test -p 1 -count=1 ./...` | exit 0, 19 ok, 0 FAIL/panic, 3 no-test-files pkgs |
| unit broker | `go test -count=1 ./hostexec/` | ok, 56/56 PASS, 0 fail/skip |
| unit runner | `go test -count=1 ./testrunner/` | ok, 9/9 PASS, 0 fail/skip |
| race | `go test -race ./hostexec/ ./testrunner/` | ok, both |
| swap ×10 | `-run 'TestClaimSwapLoop\|TestStageSwapConsistency' -count=10` | 10/10 pass |
| shell | `sh -n testrunner/entrypoint.sh` | clean |
| R1 argv | `/tmp/p6probe/r1` | exact match + evil-params immune |
| R2 anchors | `/tmp/p6probe/r2` | chain closed; partial-anchor wart noted |
| R5 provenance | `/tmp/p6probe/r5` | stage HEAD stable across live commit |

Executed: unit, race, fake-Docker (`TestDisarm*`, fixture script),
source-level, probes. Skipped: none (0 skips everywhere).
Live-Docker: not performed (prohibited; installation's first proof).
`hostexec verify` CLI end-to-end needs a non-`/tmp` non-repo
StateDir, unavailable in this sandbox (environmental; covered at
unit level). Probes re-runnable under `/tmp/p6probe/`.

## Accepted residuals (§9 + prior)

Historical P3 anomaly (non-blocking, undisturbed); manifest
truthfulness under churn; trusted-daemon TOCTOU; scratch-DSN
visibility; hollow-green → diff review; post-disarm mutable detail.
No residual reopened — no new contradictory evidence found.

## P5-MED-2 prerequisite (§10): DOCUMENTED, awaiting performance

Docs install step 0 requires: exact StateDir/EnvFile, sandbox-deny
demonstration, `extra_agent_roots` coverage, ownership/permissions
(step 3 rules), fail-closed no-attestation→no-install. All five §10
bullets addressed. The attestation itself is the owner's install-time
act and was correctly not attempted here.

## Installation readiness

READY FOR INSTALLATION. All five corrections hold under adversarial
re-verification; the §8 table is fully green; residuals are bounded
and documented. This review does not authorize installation — that
remains the owner's one-time decision, with step zero (attestation)
and first live proof (real `docker build` via a broker job) as
gates. No further review cycle.
