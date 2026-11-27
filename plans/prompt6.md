# Final Focused Adversarial Verification — Host Execution Broker (Prompt 6)

## Mission

Act as an independent principal security engineer performing the final adversarial verification of the Miniapp Studio Host Execution Broker following the Prompt 5 remediation.

The previous verification established that the original Prompt 3 vulnerabilities were closed, but discovered two new HIGH findings and one MEDIUM evidence-integrity gap.

The latest remediation claims to have fixed:

- P5-HIGH-1: Docker build command omitted the explicit Containerfile path.
- P5-HIGH-2: Owner-side result anchors could be overwritten after deleting a workspace copy.
- P5-MED-1: Test-result details were not frozen in owner-side provenance.
- P5-LOW-1: Startup sweep wording overstated what was known.
- P5-LOW-2: Git provenance was collected from the live repository instead of the executed snapshot.

Your task is to independently verify these corrections and determine whether the broker is ready for one-time installation.

**This is a focused verification, not another comprehensive architecture review.**

Governing references:
- `PRINCIPLES.md`
- `pg_muse.md`
- `docs/host-exec-broker.md`
- `docs/test-runner.md`
- Previous review: `plans/adv_review_prompt5.md`
- Current remediation implementation and regression tests.

## 1. Scope and restrictions

Inspect only the relevant implementation and tests:

- `hostexec/exec.go`
- `hostexec/queue.go`
- `hostexec/verify.go`
- `hostexec/serve.go`
- `hostexec/*_test.go`
- `cmd/hostexec/main.go`
- `testrunner/`
- `docs/host-exec-broker.md`
- `docs/test-runner.md`

Do not modify source, tests, documentation or configuration.

Do not:
- Install or start the broker.
- Execute Docker operations.
- Access or modify PostgreSQL.
- Run systemd operations.
- Create or modify networks or containers.
- Access credentials or secret files.
- Change authorization or security policies.
- Reopen previously settled architecture decisions without concrete contradictory evidence.

Repository inspection, unit tests, race tests, fake-Docker tests and temporary adversarial probes outside the repository are permitted.

## 2. Critical verification: P5-HIGH-1 — Docker build correctness

The previous defect was an `image.build` command missing the explicit Containerfile argument.

Verify that the actual broker-generated argv is:

`docker build -f <absolute-stage-path>/Containerfile -t <approved-tag> <absolute-stage-path>`

Confirm:

1. `-f` is present.
2. Its value points to the staged Containerfile, not the mutable repository.
3. The context path is the verified staged build context.
4. No caller-controlled field can substitute the Containerfile or context.
5. The argument is independent of the broker's working directory.
6. The pinned argv test asserts the complete expected command.
7. Documentation agrees with the implementation.

Determine whether the command conforms to Docker's documented CLI syntax.

Do not claim real-Docker execution was verified: Docker operations are prohibited in this review. The actual Docker build proof remains part of installation.

**Acceptance:** No remaining code or test defect causes the build to omit or misidentify the Containerfile.

## 3. Critical verification: P5-HIGH-2 — First-wins authoritative anchors

Reproduce the original attack:

1. Submit and finish a job with ID `x`.
2. Preserve the original owner-side anchor bytes and hash.
3. Delete the workspace `done/x.json` copy.
4. Resubmit job ID `x`.
5. Attempt to claim and execute it.
6. Run `verify`.

Verify all of the following:

- Claim rejects the reused ID as a duplicate by consulting owner-side anchors.
- The original anchor is not overwritten.
- The anchor's bytes and hash remain identical.
- Collision handling preserves evidence without replacement.
- The sweep does not resurrect or replace the ID.
- `verify` detects the missing workspace copy and reports `MISSING-COPY`.
- No path produces a clean verification result after the workspace copy has been deleted.
- Concurrent submissions and crashes cannot create two authoritative outcomes for one job ID.

Inspect the actual filesystem creation semantics.

`O_EXCL` must protect the authoritative record. Atomic publication must not silently replace an existing anchor.

**Acceptance:** Once an owner-side anchor exists, its history is immutable through all agent-accessible queue and workspace operations.

## 4. Critical verification: P5-MED-1 — Suite-evidence detail integrity

Inspect the implementation that captures test-result detail into `disarm.json`.

The required files are exactly:

- `gotest.json`
- `summary.txt`
- `exitcode`

Verify that each has:

- SHA-256 digest.
- Recorded size.
- Explicit status.

Test these conditions:

1. All three files exist and are recorded correctly.
2. Independent hash calculation matches the stored digest.
3. Modification after capture is detected.
4. Deletion after capture is detected.
5. Symlink substitution is recorded and cannot silently pass.
6. An oversized file is explicitly recorded.
7. Unreadable files are explicitly recorded.
8. Missing required files are explicitly recorded.
9. Malformed or incomplete manifests cannot qualify as complete evidence.
10. Repeated runs preserve the integrity of previous authoritative records.

Inspect whether hashing is streaming and size-bounded.

**Important distinctions:**

- A digest proves integrity relative to the captured bytes. It does not establish that the bytes were authentic or truthful before capture.
- A missing or incomplete required evidence manifest must block installation acceptance.
- Do not falsely claim that every subsequent run is frozen after deployment auto-disarms. The documented post-disarm mutable-details limitation remains an accepted residual unless the implementation explicitly changes it.

Verify that the owner's authoritative `StateDir` record is separate from agent-writable workspace copies.

## 5. P5-LOW-1 — Startup sweep wording

Verify that sweep-generated UNKNOWN records assert only what the broker actually knows.

The reason should communicate:

`residue found at startup; execution state unknown`

Confirm that the record does not falsely assert a particular crash or mid-execution event.

Verify that:
- The operation remains `UNKNOWN`.
- Existing terminal records remain untouched.
- The operation and size are included where validated.
- Unparseable residue is handled honestly.
- The wording is consistent in code, tests and documentation.

This is a correctness and evidence-truthfulness check, not a reason to reopen the sweep architecture.

## 6. P5-LOW-2 — Git provenance from the executed snapshot

Verify that `gitProvenance` reads the actual per-job execution snapshot, not the live mutable repository.

Inspect and test:

1. The snapshot's `.git` metadata is used.
2. The `git -C` argument points to the staged snapshot path.
3. A fake-git test detects accidental use of the repository path.
4. Git HEAD and dirty state correspond to the snapshot used for the run.
5. A repository mutation after snapshot creation does not change the recorded provenance of that run.
6. Git-unavailable or malformed metadata is represented explicitly, never fabricated.

The provenance must identify what was actually executed.

## 7. Cross-boundary consistency

Verify the interaction among the five corrections.

In particular:

- Image-build evidence binds to the verified staged build context.
- Run evidence binds to the per-job repository snapshot.
- Git provenance refers to the same snapshot.
- Test-result detail hashes refer to the files captured for that run.
- Owner-side job anchors bind to the exact execution result.
- Job IDs cannot be reused to replace prior history.
- Missing or incomplete evidence cannot accidentally satisfy installation acceptance.

Do not accept individually passing tests if the cross-component behavior allows the original exploit chain to return.

## 8. Required focused regression tests

Run the existing regression tests and inspect whether they actually prove the required invariants.

| Finding | Required outcome |
|---|---|
| P5-HIGH-1 | Explicit absolute staged Containerfile argument |
| P5-HIGH-2 | Delete/resubmit rejected; immutable anchor; MISSING-COPY |
| P5-MED-1 | Three detail files hashed; tampering and absence detected |
| P5-LOW-1 | Truthful startup-residue wording |
| P5-LOW-2 | Git provenance comes from executed snapshot |

Re-run the highest-risk anchor chain end to end.

Also run:
- `gofmt -l .`
- `go vet ./...`
- `go build ./...`
- `go test -p 1 -count=1 ./...`
- `go test -race -count=1 ./hostexec/ ./testrunner/`
- `sh -n testrunner/entrypoint.sh`
- Both swap tests with `-count=10`.

Report exact commands and pass/fail/skip counts.

Distinguish:
- Tests genuinely executed.
- Tests skipped because of environment restrictions.
- Fake-Docker verification.
- Source-level verification.
- Live-Docker verification (not performed in this review).

Do not call an exit code of zero a complete pass when tests have skipped.

## 9. Existing accepted residuals — do not reopen without new evidence

The following were explicitly accepted or bounded in Prompt 5:

- One historical P3 swap-test anomaly, independently classified as non-blocking after extensive repeated and independent verification.
- In-place snapshot churn, where the manifest truthfully describes the bytes actually executed.
- Trusted Docker-daemon TOCTOU.
- Agent visibility of the scratch-cluster DSN under the accepted database isolation model.
- Test-gutting versus legitimate development, handled through frozen provenance and mandatory Project Owner diff review.
- Workspace test-result details after auto-disarm, subject to the documented capture-time integrity limitation.

Only reopen a residual if new evidence demonstrates a materially different or more severe failure.

## 10. Installation prerequisite: P5-MED-2

The owner-side state and credential boundary still requires an installation-time operational attestation.

Verify that the documentation requires:

- Exact `StateDir` and `EnvFile` paths.
- Evidence that the agent sandbox cannot write to or read protected owner-side state and secrets as applicable.
- Correct `ExtraAgentRoots` configuration.
- Protected ownership and permissions.
- Fail-closed behavior when the attestation is missing or incorrect.

Do not attempt to perform this host-side attestation yourself. It is a separate installation prerequisite.

A correct path outside `RepoRoot` is not sufficient evidence that the path is outside agent access.

## 11. Final verdict

Choose exactly one:

1. **READY FOR INSTALLATION** — all focused corrections verified and no unresolved material implementation blocker remains; operational attestation is ready to be performed as installation step zero.
2. **READY WITH OWNER ACCEPTANCE** — only a clearly documented, bounded residual requires explicit owner acceptance.
3. **BLOCKED** — any of the five corrections is ineffective, an original exploit remains reproducible, or an authoritative evidence/authorization boundary is broken.

Be precise about the distinction between installation readiness and installation authorization.

This review does not authorize installation.

## 12. Final report

Deliver a concise report containing:

1. Executive verdict.
2. Five-finding status and evidence.
3. Anchor deletion/resubmission reproduction results.
4. Suite-evidence hash integrity results.
5. Git provenance snapshot verification.
6. Complete regression test counts.
7. Remaining accepted residuals.
8. P5-MED-2 installation prerequisite.
9. Whether the broker is ready for the one-time installation decision.

Do not modify anything and do not perform any infrastructure operation.

## Governing principle

This is the final focused verification of the broker's corrected security boundaries.

Do not demand theoretical perfection. Require demonstrable enforcement of the actual invariants.

If the five corrections hold and the operational installation prerequisite is understood, finish the review and let Miniapp Studio move forward.

No further broad review cycle.