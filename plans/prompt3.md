Use this prompt with the Muse coding agent. It authorizes one adversarial review and one bounded remediation pass, without authorizing broker installation or infrastructure operations.

# Adversarial Security Review — Miniapp Studio Host Execution Broker

## Mission

Act as an independent principal security engineer conducting an adversarial code review of the newly implemented Miniapp Studio Host Execution Broker.

Your task is to determine whether an untrusted coding agent can exploit the broker to gain unintended host privileges, execute unauthorized Docker operations, access secrets, bypass authorization, or corrupt execution evidence.

The broker is a privileged host-side security boundary. A vulnerability here can compromise the developer workstation and undermine the entire agent-execution architecture.

Be skeptical. Do not accept implementation claims, comments, documentation, or passing unit tests as proof of security.

**Governing principles:** PRINCIPLES.md, pg_muse.md, local-first engineering, reality-first engineering, least privilege, explicit owner authorization, and fail-closed execution.

## 1. Scope and restrictions

Review the actual implementation, including:

- `hostexec/`
- `cmd/hostexec/`
- Systemd service unit and installation configuration
- Job and result queue implementation
- Operation schemas and validation
- Build-input pinning and verification
- Auto-disarm and owner-only re-arm
- `testrunner/` integration
- Documentation and security assumptions
- All relevant tests

Trace the complete execution path from agent-authored job to Docker operation and returned result.

**Do not:**

- Install or start the broker.
- Execute Docker infrastructure operations.
- Create, attach, disconnect or remove Docker networks.
- Launch containers.
- Access, print or modify credentials.
- Weaken sandbox restrictions.
- Broaden the operation allowlist.
- Change the authorization model.
- Delete or weaken existing tests to obtain a green result.

Repository inspection, source-level analysis, unit tests, race tests and fake-Docker testing are permitted.

## 2. Threat model

Assume:

1. The coding agent is untrusted input, even when acting with legitimate project authorization.
2. The agent can create, modify and submit job files within its permitted workspace.
3. The agent may deliberately construct malformed, adversarial or replayed requests.
4. The agent may race filesystem operations and submit concurrent jobs.
5. The agent may crash the broker, restart the host or interrupt an operation.
6. The agent may attempt to exploit Docker CLI arguments, mounts, environment variables, image selection or network configuration.
7. The agent may attempt to tamper with results, build pins, configuration or deployment state.
8. The Docker daemon and host are outside the agent's trust boundary.
9. Owner authorization applies only to the explicitly approved PostgreSQL test-runner workflow, not unrestricted future infrastructure operations.

Do not assume benign agent behavior.

## 3. Required adversarial investigations

### A. Privilege escalation and operation confinement

Determine whether any job can cause:

- Arbitrary command or shell execution.
- Execution of an unlisted Docker command.
- Arbitrary image selection or unapproved image substitution.
- Arbitrary host filesystem mounts.
- Access to the Docker socket from the runner.
- Host networking or privileged container execution.
- Access to unrelated containers, networks, volumes or host resources.
- Modification or deletion of unrelated host resources.

Inspect every operation, not merely the public API.

Verify that the broker constructs its own argument arrays and fixed resource parameters. Caller-controlled values must never become unrestricted command arguments.

Challenge the banned-token filter: it is supplementary validation, never the primary security boundary.

### B. Job queue and filesystem attacks

Test:

- Symlink and hardlink attacks.
- Path traversal and directory replacement.
- TOCTOU races between validation and execution.
- FIFO, device-file and unexpected file-type inputs.
- Malformed JSON, oversized requests and duplicate fields.
- Unknown fields, unknown operations and invalid types.
- Duplicate IDs and replay attempts.
- Concurrent submissions and simultaneous workers.
- Unauthorized modification of running jobs or completed results.
- Atomic state-transition failures and interrupted writes.

Check directory ownership, permissions, file creation modes, locking and atomic rename semantics.

Determine whether a malicious agent can replace or forge a result after the broker has executed an operation.

### C. Authorization and deployment auto-disarm

This is a critical area.

The initial owner authorization is limited to the approved PostgreSQL test-runner deployment sequence.

Verify:

- Every deployment operation is restricted to the approved resources and exact workflow.
- Deployment operations cannot be repurposed with caller-supplied parameters.
- Deployment capabilities automatically disarm after the first genuinely successful full suite run.
- Disarm state survives broker restarts.
- The agent cannot forge a successful test result to trigger disarm.
- The agent cannot restore or re-arm deployment permissions.
- Owner-only re-arm is enforced at the actual trusted execution boundary.
- Replayed, stale or interrupted jobs cannot bypass the authorization state.
- A successful partial run is never treated as a successful full suite.
- Repeatable read-only or non-deployment operations retain only their explicitly intended scope.

Examine the distinction between an operation being technically available and an operation being authorized.

### D. Docker security boundary

Inspect every constructed Docker invocation.

Verify:

- No `--privileged`.
- No host networking.
- No Docker socket mount.
- No unexpected published ports.
- Only exact allowlisted network and container names.
- Only explicitly approved mounts and read-only modes.
- No unrestricted Docker environment or option injection.
- No ability to inspect or manipulate arbitrary host containers.
- No accidental exposure of the database outside its approved network.

The broker's Docker access is privileged. Its operation surface must therefore be smaller and more constrained than the Docker CLI itself.

### E. Credential and information leakage

Inspect:

- DSN handling.
- Environment-file paths and permissions.
- Docker command logging.
- Standard output and standard error capture.
- Result serialization.
- Error messages and crash reports.
- Temporary files and cleanup.
- Systemd configuration.
- Repository, Go toolchain and module-cache mounts.

Verify that neither credentials nor sensitive environment values appear in logs, job records, result files, process arguments or image layers.

Do not introduce a new credential design when the existing testdb harness already provides the required least-privilege roles and isolated databases.

### F. Crash recovery, concurrency and uncertain effects

Attack:

- Broker termination during operation execution.
- Host restart with jobs in `running/`.
- Lock contention.
- Duplicate requests.
- Docker command timeout.
- Partial Docker success followed by broker failure.
- Result-write failure.
- Disarm-state persistence failure.

Determine whether every ambiguous outcome becomes `UNKNOWN` rather than falsely reporting success.

Ensure recovery reconciles actual Docker state before retrying uncertain operations. Never assume an interrupted request had no effect.

### G. Systemd and host privilege boundary

Inspect the actual service unit and installation assumptions.

Evaluate:

- Dedicated service identity and required privileges.
- `NoNewPrivileges`.
- Filesystem restrictions.
- Writable-path confinement.
- Network address-family restrictions.
- Docker socket access.
- Service startup and restart behavior.
- Single-instance enforcement.
- Configuration and pin-file ownership.
- Protection against unauthorized unit or configuration replacement.

Identify the minimum privileges required for the broker to perform its authorized operations.

Do not blindly recommend security settings that prevent required functionality. Demonstrate the actual boundary and any residual privilege.

### H. Build-input pinning and supply-chain integrity

Verify:

- Every executable build input is covered by the intended hash pin.
- The pin cannot be regenerated or modified by the untrusted agent to bless tampered files.
- Build and runtime verify the same inputs.
- Containerfile and runner source drift causes refusal.
- The pin helper cannot silently approve arbitrary changes.
- Image identity and digest are recorded.
- The broker cannot be tricked into executing an unexpected image.

Identify the trusted source of the initial pins and who is permitted to update them.

## 4. Mandatory adversarial tests

Where feasible, implement or run tests demonstrating:

1. Arbitrary command injection is rejected.
2. Unknown operations and fields are rejected.
3. Arbitrary mounts and network modes are rejected.
4. Symlink and path traversal attacks fail.
5. Duplicate and replayed jobs cannot create unintended effects.
6. Concurrent operations cannot violate single-flight guarantees.
7. Forged suite results cannot trigger auto-disarm.
8. Agent-controlled inputs cannot re-arm deployment operations.
9. Tampered build pins cause refusal.
10. Interrupted operations produce honest `UNKNOWN` states.
11. The broker does not leak secrets into results.
12. The service cannot be manipulated into executing outside its allowlist.

Use fake Docker or deterministic doubles for adversarial operation tests. Never attack the live host Docker daemon during this review.

Tests must demonstrate the security invariant, not simply assert that a helper returned an expected error.

## 5. Finding classification

Classify every confirmed finding:

- **Critical:** arbitrary host execution, privilege escalation, unrestricted Docker access, credential compromise, or authorization bypass with severe consequences.
- **High:** ability to escape operation confinement, bypass deployment approval, forge successful execution, or manipulate privileged state.
- **Medium:** meaningful isolation, recovery, auditability or integrity weakness.
- **Low:** limited security weakness or operational defect with contained consequences.
- **Informational:** hardening opportunity without a demonstrated vulnerability.

For each finding provide:

1. Identifier and severity.
2. Exact affected files and code locations.
3. Attack preconditions.
4. Concrete attack sequence.
5. Demonstrated impact.
6. Evidence or reproducing test.
7. Root cause.
8. Smallest durable correction.

Separate demonstrated vulnerabilities from theoretical concerns.

Do not manufacture findings simply to make the review appear comprehensive.

## 6. Bounded remediation pass

After completing the adversarial review:

- Fix confirmed Critical, High and genuine structural defects within the existing broker authorization and architecture.
- Fix confirmed Medium defects when the correction is local and does not expand infrastructure or authority.
- Add regression tests demonstrating each corrected invariant.
- Preserve existing tests and contract behavior.
- Avoid cosmetic refactoring, speculative abstractions and unrelated repairs.
- Do not silently broaden permissions or operation capabilities.

If a finding requires changing the approved security boundary, obtaining additional host privileges, broadening owner authorization, or changing the fundamental architecture, stop and report it for an owner decision.

Perform only one remediation pass. Do not create an endless review/remediation loop.

## 7. Final verification

Run the applicable checks:

- gofmt
- go vet
- go build
- full serial test suite
- broker unit tests
- race tests
- adversarial tests
- fake-Docker end-to-end tests

Report exact commands, exit codes, pass/fail/skip counts and environmental limitations.

A test that skipped is not a passing test. Clearly distinguish source-level verification, fake-Docker verification and actual host-side verification.

Do not install the broker or execute any Docker infrastructure operation.

## 8. Final deliverable

Produce a concise but evidence-rich adversarial review report containing:

1. Executive verdict: ACCEPTABLE, ACCEPTABLE WITH RESIDUAL RISKS, or BLOCKED.
2. Confirmed findings ordered by severity.
3. Exploitability and impact.
4. Remediation performed and regression tests added.
5. Unresolved risks and owner decisions required.
6. Final test evidence.
7. Whether the broker is ready for one-time installation.

The final verdict must answer this question:

**Can an adversarial coding agent, using only the permissions and workspace access it actually possesses, cause the privileged host broker to perform an operation outside the explicitly authorized PostgreSQL test-runner workflow?**

Do not answer based on intention or documentation. Answer from inspected code, adversarial tests and demonstrated enforcement.

## Governing rule

The objective is not to prove that the broker works under friendly conditions.

The objective is to determine whether it remains safe when the agent behaves maliciously, the filesystem behaves unexpectedly, processes crash, and execution results are ambiguous.

Reality-first engineering: demonstrate the security boundary before trusting it with host privileges.

No broker installation, service startup, Docker network changes, container attachment or runner execution is authorized by this review.

**Expected outcome:** one serious adversarial review, one bounded remediation pass, then an installation decision. No endless review cycle.