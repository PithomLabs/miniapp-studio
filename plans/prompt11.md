The corrective task should address the nine acceptance items in `plans/adv_review_prompt10.md`, with particular attention to the root cause of the security failures.

I've also made **PostgreSQL 19 on your existing Docker instance the sole database testing standard**. No PG16/17 substitution or mixed-version results.

# Implementation Directive — Agent Gateway Phase B Part 1 Corrections

**Governing review:** `plans/adv_review_prompt10.md`\
**Scope:** Bounded correction and verification of Phase B Part 1.\
**Status:** NOT ACCEPTED pending correction.\
**Execution:** Local/fake-backed only. Part 2 live installation remains unauthorized.

## 1. Objective

Correct all acceptance findings identified in the adversarial review and establish a secure, recoverable Agent Gateway suitable for subsequent owner-authorized live installation inside a Fly.io Builder Sprite.

This is not an invitation to patch individual symptoms.

**Investigate and correct the underlying architectural and execution-path failures.**

For every finding, trace the complete execution path, identify the violated invariant, determine why the existing design allowed the failure, and correct the problem at the appropriate ownership boundary.

A fix that merely makes the existing test pass while preserving the underlying failure mode is unacceptable.

Preserve the existing architecture, protocol, and scope unless concrete evidence establishes that a change is necessary.

## 2. Mandatory engineering discipline

For every correction:

1. Reproduce the original failure before modifying the code.
2. Trace the root cause through all relevant components.
3. Identify affected invariants and secondary failure modes.
4. Design a durable correction at the correct ownership boundary.
5. Add a regression test that fails against the original implementation.
6. Implement the correction.
7. Prove the regression test passes.
8. Exercise concurrency, crash, restart, and malformed-input paths wherever relevant.
9. Run the complete required verification gates.
10. Update evidence and operational documentation.

Do not:

- Add a mutex to conceal a missing durable command-claim mechanism.
- Add permission checks while retaining symlink-following filesystem operations.
- Retry indefinitely to conceal missing callback recovery.
- Convert errors into successful responses.
- Suppress errors or weaken assertions.
- Hardcode successful test outcomes.
- Introduce a second persistence mechanism without a demonstrated requirement.
- Claim that fake-backed tests establish live Sprite behavior.

For fixes spanning multiple components, verify the system-level invariant rather than testing only the modified function.

## 3. PostgreSQL testing standard — mandatory

**Use PostgreSQL 19 exclusively for all database-backed development, integration, migration, and regression testing.**

The owner has an existing PostgreSQL 19 instance running through Docker.

Use the established connection configuration, with the existing PG19 endpoint at:

`127.0.0.1:5434`

First inspect the repository's test configuration and environment. Confirm the actual server version using:

```
SELECT version();
SHOW server_version;
```

The result must identify PostgreSQL 19.x.

Set the project's database test configuration to use that instance consistently, without printing credentials or DSNs containing secrets.

### Testing rules

- All database-backed tests use PostgreSQL 19.
- All migration tests use PostgreSQL 19.
- All RLS and tenant-isolation tests use PostgreSQL 19.
- All integration and end-to-end tests use PostgreSQL 19.
- Race tests that require a database use PostgreSQL 19.
- Run serially using `-p 1` for the database suite, consistent with the established project convention.
- Use isolated disposable test databases or schemas. Never destroy or reset an existing database containing user data.
- Preserve the existing Docker PostgreSQL 19 instance.
- Do not substitute PostgreSQL 16 or 17.
- Do not silently fall back to a developer-local or newly provisioned database.
- If PostgreSQL 19 is unavailable, stop database-backed verification and report the exact connection failure. Do not claim completion.

The standard full-suite command is:

```
go test -p 1 ./...
```

The required race suite covers all six touched packages:

```
go test -race -p 1 ./gateway ./store ./tools ./server ./cmd/studio ./cmd/gateway
```

Also run:

```
gofmt -l .
go vet ./...
go build ./...
```

Record the PostgreSQL version, Docker-backed connection identity, exact commands, exit codes, package results, skips, and failures in the evidence report.

**PostgreSQL 19 is the project's uniform database verification baseline.**

## 4. Mandatory correction bundle

Follow the acceptance conditions in the adversarial review.

### B1 — Enforce the Gateway/agent operating-system identity boundary

This is the primary blocker.

The current implementation creates separate Unix users but never uses them. The agent inherits the Gateway identity and can access HMAC signing keys.

Investigate the actual execution and privilege model before choosing the implementation.

Establish a kernel-enforced separation in which:

- Gateway signing keys are accessible only to the Gateway's trusted execution identity.
- OpenCode runs as `studio-agent`.
- Agent attempts to read Gateway signing keys fail with `EACCES`.
- The supervisor performs a real privilege drop before launching the agent.
- No shared-user or permission-only fallback is permitted.
- Service environment `PATH` is explicitly pinned to trusted, root-owned executable locations.
- Staging key copies are restricted to `0600` and removed once installation succeeds.
- The shared inbox has an explicitly designed ownership and permission model.

Investigate whether the environment supports a privileged Gateway process with credential dropping, `SysProcAttr.Credential`, a constrained privilege helper, or another demonstrably supported mechanism.

Do not invent a Sprite service `user` field or assume the Fly service API supports user assignment when its current interface does not.

If the selected mechanism requires root privileges, verify that the actual environment can provide them during the separately authorized live runbook. If that cannot be established, provisioning must fail closed.

**Do not weaken the boundary to make file exchange convenient.**

### F2/F3 — Correct file-drop filesystem security

The current implementation has two demonstrated vulnerabilities:

- Reading `question.json` follows symlinks and can expose Gateway key material.
- Writing `answer.json` follows pre-existing symlinks and can overwrite arbitrary Gateway files.

Correct the filesystem operations themselves.

Requirements:

- Use directory-relative operations where appropriate.
- Reject symlinks and unexpected file types.
- Validate ownership, permissions, and file sizes.
- Use `O_NOFOLLOW` and safe creation semantics.
- Make answer-file replacement atomic and resistant to path substitution.
- Use a deliberate directory ownership and sticky-bit/group model.
- Ensure agent-written questions and Gateway-written answers have distinct ownership and integrity properties.
- Do not allow the agent to overwrite or impersonate Gateway-authored answers.

Add regression tests covering:

- Question symlink to an HMAC key.
- Answer symlink to a Gateway-owned file.
- Pre-created regular files.
- Oversized question files.
- Malformed JSON.
- Concurrent file replacement.
- Restart during file creation.

The agent must be able to submit a question and read its authorized answer, but it must not be able to forge the answer or manipulate Gateway-controlled files.

### F4 — Correct concurrent command execution

The current check-then-execute-then-record implementation allows concurrent duplicate Continue commands to launch multiple agent processes.

Correct the underlying command-claim and execution lifecycle.

Requirements:

- Persist an execution claim before dispatching the command.
- Serialize execution for a build/session.
- Preserve stable command identity.
- Make duplicate requests observe the existing execution state or replay its stored result.
- Handle crashes between claim, dispatch, process launch, session update, and result persistence.
- Reconcile interrupted commands without blindly launching duplicate work.
- Preserve ordering for distinct Continue operations.

A process-local mutex alone is insufficient because it does not protect against Gateway restart.

Regression tests must prove:

- N concurrent identical Continue commands cause exactly one logical agent execution.
- N distinct valid commands execute in a correctly serialized order.
- A Gateway crash during command execution does not produce duplicate logical execution after recovery.
- A partially completed command cannot silently strand its session.

### F5 — Correct terminal event semantics

A recoverable mid-run OpenCode error event must not prematurely terminal-fail a build that subsequently exits successfully.

Correct terminal-state determination at the adapter's ownership boundary.

- Intermediate error events should be reported as progress with diagnostic detail where appropriate.
- Terminal success/failure must be determined from the completed process outcome and the recognized question-yield condition.
- Question yield must never be misreported as successful build completion.
- Nonzero exit codes must remain honest failures.
- Do not hide actual execution failures.

Add regression tests for error-then-success, error-then-failure, question-yield, malformed output, and process termination.

### F6 — Restore a usable continuation path after recovery

The current recovery routine marks builds failed, but the Gateway subsequently rejects both assignment and continuation, leaving recovered work unusable.

Correct the recovery state machine.

An explicit authorized Continue after recovery must be capable of launching a fresh supervised process with the preserved workdir and session lineage where the agent session is genuinely resumable.

Requirements:

- No automatic silent restart.
- No fabricated `agent.resumed`.
- Verify the actual recovered session before reporting resumed.
- Preserve build and turn identity.
- Return honest failure when the session cannot be resumed.
- Prevent stale or superseded turns from continuing.
- Update runbook §11.5 to describe the implemented behavior precisely.

Add an end-to-end regression:

`recover → authorized continue → supervised resume → agent.resumed → completed`

### F7 — Make provisioning non-disruptive for shared Builders

The current provisioning routine unconditionally deletes and recreates the Gateway service, potentially killing other builds using the same application-scoped Builder Sprite.

Correct the provisioning lifecycle.

Prefer:

- Create service only when absent.
- Detect an existing healthy service and preserve it.
- Refresh per-build key material without restarting the service.
- Protect concurrent provisioning operations.
- Make version upgrades an explicit controlled lifecycle operation, not a side effect of creating another build.

If the architecture requires a different occupancy policy, demonstrate why and reject conflicting build assignments cleanly.

Regression test: provisioning a second build must not terminate or disrupt the first build's active agent session.

### F8/F9 — Correct callback durability and freshness

The current outbox can become permanently failed with no recovery mechanism, and callback envelopes are signed at emission time rather than delivery time.

Correct both underlying problems.

**Callback freshness:**

- Re-sign each delivery attempt using a fresh `SentAt`.
- Preserve the original stable `MessageID` and logical event identity.
- Preserve payload integrity and build identity.
- Keep Studio's five-minute freshness verification unchanged.
- Never weaken replay protection to accommodate delayed delivery.

Test an envelope older than five minutes successfully redelivering after fresh signing. Independently prove that stale forged bytes remain rejected.

**Terminal outbox recovery:**

- Provide an operator-visible inspection path.
- Provide a controlled requeue mechanism for failed items.
- Surface failed outbox depth/status clearly.
- Preserve original event identity during requeue.
- Correct the misleading comment referring to a nonexistent automatic polling mechanism.
- Document the Phase C Studio sender/reconciliation requirement.

Also correct the event-pump behavior that currently logs and drops an event when outbox persistence fails.

An event must not disappear silently because of disk-full, corruption, or another persistence error. Establish an explicit durable failure/degraded-state/reconciliation path.

Regression tests:

- Old callback re-signed and accepted.
- Stable identity prevents duplicate application.
- Terminal failure can be inspected and requeued.
- Requeued callback receives acknowledgment and drains.
- Outbox enqueue failure is surfaced and cannot masquerade as successful event delivery.

## 5. Minor findings M1–M16

Correct the following where appropriate, or document an explicit carry decision with technical justification. Nothing may silently disappear.

| Finding | Required disposition                                                                            |
| ------- | ----------------------------------------------------------------------------------------------- |
| M1      | Parent-directory fsync after atomic rename in dedup, outbox, and session stores                 |
| M2      | Explicit HTTP client timeout for callback delivery                                              |
| M3      | Handle SIGTERM; graceful shutdown; ensure agent lifetime is not tied to an HTTP request context |
| M4      | Process-group spawn and termination, including child processes                                  |
| M5      | Drain stderr safely with bounded handling                                                       |
| M6      | Recovery continues past individual corrupt records; eliminate Update-then-emit loss window      |
| M7      | Process reaper uses precise executable identity and start-time validation                       |
| M8      | Collector uses file-descriptor-based confinement to eliminate TOCTOU                            |
| M9      | Handle NDJSON scanner errors and report truncation                                              |
| M10     | Enforce file-drop size limits                                                                   |
| M11     | Enforce Gateway-side turn fencing                                                               |
| M12     | Enforce HTTPS scheme for Studio/Gateway transport                                               |
| M13     | Validate operator CLI commands and safely handle large artifact outputs                         |
| M14     | Record binary SHA-256 identity during provisioning and verify it in the runbook                 |
| M15     | Add production KDF provider/DeriveKey round-trip tests                                          |
| M16     | Correct protocol documentation, turn ID semantics, and safe build-ID filenames                  |

Prefer correcting small root-cause defects while already working in the affected component. For deferred items, give the exact reason, risk, and owning phase.

## 6. Evidence and documentation corrections

Update:

- `docs/gateway-phase-b-evidence.md`
- `docs/gateway-phase-b-live-runbook.md`

Correct every evidence overstatement identified in review sections G and J.

In particular:

- Correct the gateway test count.
- State that file-drop happy-path doubles are tested but directionality was previously disproven.
- State explicitly that real OpenCode has not been exercised.
- Do not call scripted-double compliance a live P-G3 proof.
- Identify unauditable OpenCode source claims and their required verification.
- Record F4 concurrency failure and correction.
- Record F6 recovery behavior honestly.
- Record F8/F9 delivery/requeue requirements.
- Correct the runbook's recovery section after implementing the real resume path.
- Add live UID/key-read-denied and inbox access checks.
- Add staging-key cleanup and restrictive-mode checks.
- Correct the stderr-hang interpretation.
- Distinguish all fake-backed evidence from future live Sprite proofs.

The capability matrix must clearly separate:

1. Implemented and tested.
2. Implemented but insufficiently tested.
3. Specified but not implemented.
4. Explicitly not built.
5. Live behavior not yet verified.

No cell may claim actual OpenCode or Sprite execution unless it happened.

## 7. Miniapp application contract

Preserve the binding application invariant:

- Miniapp is the exclusive application framework.
- Go is the exclusive implementation language for Miniapp applications.
- MiniappSpec is the canonical application specification.
- Miniapp framework/runtime is the exclusive execution model.

Ensure the Builder instructions and assignment-construction path explicitly reinforce this before artifact-bearing turns.

Do not implement the Phase D artifact-admission pipeline as part of this bounded correction.

## 8. Strict scope boundaries

Permitted:

- Phase B Part 1 source corrections.
- Regression tests.
- Fake-backed integration tests.
- Provisioning code and scripts.
- Evidence and runbook corrections.
- Local verification against the existing PostgreSQL 19 Docker instance.

Forbidden:

- Live Sprite provisioning or deletion.
- Fly API mutations.
- OAuth authentication.
- Live Gateway installation.
- External infrastructure operations.
- Production connector configuration.
- Miniapp framework modifications.
- F1/G-01 changes.
- P5 scope changes.
- New coding-agent adapters.
- New workflow engines.
- Commits or pushes.

**Part 2 remains on hold.** This task does not authorize live execution.

## 9. Completion and acceptance criteria

The task is complete only when:

1. B1 OS-level identity separation is implemented and enforced fail-closed.
2. F2/F3 symlink exploits are refused.
3. F4 concurrent identical Continue executes once.
4. F5 terminality reflects actual process outcomes.
5. F6 recovery supports an explicit, honest continuation path.
6. F7 provisioning does not disrupt an active co-located build.
7. F8/F9 callback persistence, requeue, and freshness work correctly.
8. Event-pump persistence failures are handled without silent loss.
9. Evidence and runbook corrections are complete.
10. M1–M16 are fixed or explicitly dispositioned.
11. All named regression tests pass.
12. The original adversarial probes P1–P4 are rerun and demonstrate:
    - P1: single execution under concurrent duplicate Continue.
    - P2: question symlink exfiltration refused.
    - P3: answer symlink clobber refused.
    - P4: recoverable mid-run errors do not cause premature terminal failure.
13. PostgreSQL-backed gates are green on PostgreSQL 19 Docker only.
14. Formatting, vet, build, full tests, and race tests are green.
15. No forbidden scope changes occurred.

## 10. Final handoff

Produce a correction report describing:

- Root cause for every finding.
- Durable correction and source locations.
- Red-before/green-after evidence.
- Regression tests and actual results.
- PostgreSQL 19 Docker verification details.
- M1–M16 dispositions.
- Updated evidence and runbook.
- Remaining live-only unknowns.
- Explicit statement that Part 2 was not executed.

Stop after presenting the correction evidence.

**No further general adversarial review cycle is required.** Once the bounded correction conditions are demonstrated, the Senior Architect will verify the corrections and determine whether to authorize Part 2.

The goal is not to make the current tests green again. The goal is to remove the underlying failure modes so that the same vulnerabilities cannot reappear through another execution path.

**My emphasis:** B1 cannot be solved by merely creating Unix users. The agent process must actually run under a different OS identity, and the file-drop protocol must preserve that separation in both directions. Also, a process-local mutex is not enough for command deduplication across restarts.

Use this prompt as the governing correction task. PostgreSQL 19 Docker is the uniform test baseline; live Sprite work remains on hold.