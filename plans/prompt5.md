Use this prompt for the final focused adversarial verification. It is deliberately narrower than the original review: verify that the reported vulnerabilities are actually closed, investigate the remaining anomalies, and determine installation readiness.

# Focused Adversarial Verification — Miniapp Studio Host Execution Broker

## Mission

Act as an independent principal security engineer conducting the final focused adversarial verification of the remediated Miniapp Studio Host Execution Broker.

The previous review produced a BLOCKED verdict. One repository-only remediation pass has now been completed.

Your responsibility is to verify that the original security findings are genuinely closed through implementation and reproducible evidence, not merely that corrective code and tests exist.

**The objective is an installation decision, not another architecture review.**

Governing references:

- `PRINCIPLES.md`
- `pg_muse.md`
- `docs/host-exec-broker.md`
- `docs/test-runner.md`
- Previous adversarial report: `plans/adv_review_prompt3.md`
- Remediation implementation and current tests.

## 1. Scope and restrictions

Review the actual remediation diff and the relevant implementation:

- `hostexec/policy.go`
- `hostexec/queue.go`
- `hostexec/exec.go`
- `hostexec/serve.go`
- `hostexec/stage.go`
- `hostexec/verify.go`
- `hostexec/*_test.go`
- `cmd/hostexec/main.go`
- `testrunner/`
- `docs/host-exec-broker.md`
- `docs/test-runner.md`

Review only the security boundaries affected by H1/H2, M1–M5, L1–L6 and image-digest binding.

Do not restart the original broad review of every informational finding.

**Permitted:**

- Source inspection and adversarial reasoning.
- Existing regression tests.
- Unit tests and race tests.
- Temporary external probes under `/tmp`.
- Focused concurrency and filesystem experiments that do not modify infrastructure.

**Forbidden:**

- Modifying repository source, tests, documentation or configuration.
- Installing or starting the broker.
- Running systemd operations.
- Docker commands or infrastructure operations.
- PostgreSQL access, database migrations or credential access.
- Weakening security boundaries or changing owner authorization.

This is verification-only. Any discovered defect must be reported, not silently repaired.

## 2. Priority verification areas

### V1 — H1/H2: Build integrity and immutable staging

Independently establish:

1. The complete `testrunner/` build context is covered by the installation-time pinning policy.
2. Missing, altered, unexpected and symlinked files are refused.
3. The broker stages verified bytes in owner-controlled storage.
4. Verification is performed against the staged bytes, not merely the original workspace.
5. Docker builds only from the staged context.
6. An agent cannot modify the staged bytes through its permitted workspace access.
7. The P7 marker-file attack is now refused.
8. The 20-stage swap-consistency test genuinely proves the intended invariant.

Attempt the original attack patterns:

- Add an unpinned `package main` source file.
- Modify an existing build input.
- Replace a file during staging.
- Delete or introduce unexpected files.
- Replace source files with symlinks.
- Attempt to manipulate the staging path.

**Acceptance:** No agent-controlled workspace modification can introduce executable code into a verified image build.

### V2 — H1b/L5: Per-job repository snapshots and provenance

This is a distinct trust boundary from image building.

Verify:

- Every run uses a per-job snapshot rather than the mutable live repository.
- The snapshot is created and verified safely.
- Tests, build and analysis operate against the same frozen snapshot.
- The mounted snapshot is read-only.
- Tree hash, Git HEAD, working-tree status and available package-level results are frozen in provenance.
- The provenance record identifies precisely what was tested.
- Subsequent repository modifications cannot alter the evidence for an earlier run.
- Snapshot contents exclude runtime-only directories and reject unexpected symlinks.

**Important accepted limitation:**

The broker cannot technically distinguish malicious test-gutting from legitimate development when both produce modified source trees.

Do not claim this problem is solved by hashing.

The required safeguard is snapshot consistency, frozen provenance and mandatory Project Owner diff review before accepting the result.

Verify that this limitation is accurately represented in the documentation and acceptance workflow.

### V3 — Image digest binding

Verify that:

1. A successful verified build records the actual image ID.
2. A run resolves the current image tag to its image ID.
3. The broker refuses missing or mismatched images.
4. A stale image or a tag pointing to a different image cannot silently execute.
5. The digest being verified is the digest actually selected for execution.

Assess the stated trusted-Docker-daemon assumption, but do not expand the review into hypothetical malicious-daemon attacks.

### V4 — M1/M2: Queue claim integrity

Re-examine the original P3 exploit.

Verify that the current claim implementation:

- Uses directory-descriptor-relative filesystem operations.
- Refuses symlink following.
- Checks regular-file identity through file descriptors.
- Enforces size limits before allocating unbounded memory.
- Protects against FIFO blocking.
- Verifies device/inode identity where required.
- Prevents claimed bytes from diverging from executed bytes.
- Does not allow poison jobs to block legitimate pending jobs.

**Critical anomaly investigation:**

The implementation report disclosed one unexplained swap-test failure among more than 25 passes.

Do not dismiss this as flakiness.

Determine:

- Which invariant failed.
- Whether the failure was in production code, test setup or test orchestration.
- Whether it can be reproduced deterministically.
- Whether zero outside-byte claims can be established under repeated adversarial swapping.
- Whether the test itself is asserting the correct security property.

If the anomaly remains unexplained, report it as unresolved. Do not declare the queue boundary verified merely because subsequent runs pass.

### V5 — Snapshot tearing during in-place writes

The remediation report identifies a residual risk: an in-place writer may tear a snapshot even though rename-based swaps are closed.

Inspect the implementation and documented quiesce protocol.

Determine:

- How a changing file is detected.
- Whether the snapshot can contain partially written content.
- Whether the before/after hashes reliably detect concurrent modification.
- Whether failed verification prevents execution.
- Whether the protocol requires the writer to cooperate.
- What happens if an agent ignores the quiesce procedure.

Do not claim immutability if the underlying consistency property depends on cooperative behavior.

Classify this residual accurately:

- Acceptable with documented and enforced owner workflow.
- Conditional on an operational quiescence gate.
- Installation blocker if inconsistent snapshots can execute undetected.

Recommend the smallest necessary control. Do not invent a new snapshot architecture without evidence.

### V6 — M3/M4: Evidence integrity and crash reconciliation

Verify that:

- Authoritative result anchors reside in owner-protected `StateDir`.
- Repository result copies are independent files, never hardlinks to authoritative anchors.
- Tampering with repository copies is detected by `verify`.
- The owner-side anchor cannot be modified through agent workspace access.
- `Finish` cannot silently overwrite an existing authoritative result.
- Concurrent or repeated job IDs cannot corrupt result history.
- Startup sweep never overwrites terminal results.
- Invalid or fabricated running residue is rejected or quarantined.
- Uncertain operations remain `UNKNOWN`.
- A recovered `UNKNOWN` job is reconciled against actual state before retry.

Reproduce the original P4 attack:

- Plant a running record for an already-finished ID.
- Plant an unexecuted ghost record.
- Restart/sweep logic through its unit-testable entrypoint.

Verify that prior terminal evidence remains intact and fabricated jobs do not become trusted execution records.

### V7 — M5: Owner-side state and credential placement

This is a critical trust boundary.

Verify that the configuration rejects:

- `StateDir` inside the repository.
- `EnvFile` inside the repository.
- Paths resolving inside `/tmp`, `/dev/shm` or other agent-accessible locations.
- Symlinked paths that resolve into agent-writable locations.
- Unsafe ownership or world-writable state directories.
- Version-controlled or repository-anchored credential files.
- Unsafe lock and pin-file locations.

Most importantly, inspect the actual threat model for the installation:

**Outside `RepoRoot` does not automatically mean outside agent access.**

Establish the exact owner-side directories that must be protected and whether the runtime sandbox actually prevents the agent from modifying them.

Do not assume path validation alone creates a security boundary.

If protection depends on an installation-time ownership or permission condition, make it an explicit fail-closed prerequisite.

### V8 — Auto-disarm, provenance and L1–L6

Verify:

- Disarm occurs only after a complete, genuinely executed suite run meets its acceptance conditions.
- `disarm.json` is first-wins and cannot be overwritten by an agent.
- Git HEAD and dirty state are recorded when available.
- Package counts and build identity are preserved.
- Git-unavailable conditions are represented honestly.
- The owner-side provenance is authoritative.
- A repeat run cannot silently replace earlier acceptance evidence.
- Re-arm remains owner-only.
- Trailing JSON data and duplicate keys are rejected.
- Pretty-printed jobs and ordinary trailing newlines work.
- Quarantine names cannot overwrite prior records.
- Indeterminate operations use `UNKNOWN`.
- Toolchain paths are passed from broker configuration, not hardcoded to one workstation.
- Existing approved operations remain bounded by the same fixed argv policy.

## 3. Mandatory regression verification

Run the existing regression tests and inspect whether they actually reproduce the former attacks.

Minimum required evidence:

| Attack                                | Required outcome                      |
| ------------------------------------- | ------------------------------------- |
| P7 unpinned source injection          | Refused                               |
| Build-context modification            | Refused                               |
| Build-stage race                      | No attacker-controlled bytes execute  |
| P3 queue symlink-swap                 | Zero outside-byte claims              |
| FIFO swap                             | No blocking or broker wedge           |
| Oversized queue file                  | Rejected without unbounded allocation |
| Existing terminal result during sweep | Preserved                             |
| Fabricated running residue            | Not trusted                           |
| Workspace result tampering            | Detected against owner anchor         |
| In-repo/agent-accessible StateDir     | Configuration refused                 |
| In-repo/agent-accessible EnvFile      | Configuration refused                 |
| Image tag mismatch                    | Execution refused                     |
| Ambiguous execution result            | UNKNOWN                               |

Run applicable:

- `gofmt`
- `go vet ./...`
- `go build ./...`
- Full serial test suite.
- Race tests on `hostexec` and `testrunner`.
- Focused reproduction probes.

Report exact pass, fail and skip counts.

A passing unit test is not sufficient where a direct filesystem or concurrency attack can independently test the claimed invariant.

## 4. Finding classification

Use these categories:

- **BLOCKER:** Demonstrated ability to execute untrusted code through an authorized operation, bypass authorization, modify authoritative state, access protected secrets or forge trusted acceptance evidence.
- **HIGH:** Significant security-boundary weakness requiring correction before installation.
- **MEDIUM:** Contained risk with explicit operational controls or a bounded correction.
- **LOW:** Robustness or maintainability defect without meaningful privilege or integrity impact.
- **ACCEPTED RESIDUAL:** Risk explicitly understood and accepted within the documented threat model.

Do not elevate every theoretical concern into a blocker.

Do not downgrade a demonstrated exploit merely because a test normally passes.

## 5. Final decision

Choose exactly one verdict:

1. **READY FOR INSTALLATION** — all material security invariants have reproducible evidence; residual risks are understood and bounded.
2. **READY WITH EXPLICIT OWNER ACCEPTANCE** — no demonstrated privilege-boundary failure remains, but one clearly scoped operational residual requires owner acceptance.
3. **BLOCKED** — an actual exploit, unresolved critical anomaly or untrusted execution/evidence path remains.

The unexplained P3 failure must be resolved or explicitly classified with evidence.

The in-place snapshot-writing risk must have a defensible consistency guarantee or a concrete operational gate.

The owner-side evidence and state directories must be demonstrably outside agent write access.

## 6. Final report

Produce a concise, evidence-rich report containing:

1. Executive verdict.
2. Original H1/H2/M1–M5/L1–L6 status.
3. Verification results for V1–V8.
4. P3 anomaly investigation and reproducibility findings.
5. Snapshot consistency analysis.
6. Owner-side state and credential boundary evidence.
7. Exact test results and environmental skips.
8. Remaining accepted residuals.
9. Whether one-time broker installation is authorized for consideration.

Do not perform installation, start the broker, or execute Docker, PostgreSQL or systemd operations.

Do not commission another broad review after this one. Any remaining finding must be tied to concrete evidence and a specific effect on the installation decision.

## Governing principle

We are not seeking mathematical proof that software can never fail.

We are establishing sufficient, reproducible evidence that the broker enforces its intended security boundary under the threat model we actually have.

The goal is to finish this verification, make an informed installation decision and return to building Miniapp Studio.

**My expectation:** one focused verification pass. If the material invariants hold and the two disclosed residuals are safely bounded, we install. No reopening already-settled architecture decisions.