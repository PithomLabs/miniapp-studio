**My verdict: APPROVED WITH TWO REQUIRED SAFETY AMENDMENTS.** The four-phase sequence is sound. I see no reason to reopen the architecture or delay Phase C implementation.

I particularly approve the reuse-first decisions: server-side C1 sender loop, real Gateway client against `httptest`, embedded templates with vanilla JavaScript, conditional migration, and no local OpenCode unless evidence demonstrates a need.&#x20;

prompt18_plan.md



However, two details must be resolved before Phase D implementation.

## **1. Artifact execution must not become host execution**

The proposed subprocess sandbox is explicitly local-development grade, with no user namespaces.&#x20;

prompt18_plan.md



I accept local subprocess development as a testing technique, but not as a security boundary for executing arbitrary agent-generated Go code.

`GOPROXY=off`, a temporary directory, and a timeout do not prevent a malicious test from accessing host files, environment variables, credentials, or network resources.

My decision:

- `artifact.verify` may run locally after strict input validation.
- `artifact.test` must execute untrusted artifacts inside an isolated execution environment with no host credentials, restricted filesystem access, network denied, and resource/time limits.
- If that isolation is unavailable, test only trusted repository fixtures. Do not execute arbitrary agent-produced artifacts on the developer's host.
- Label the limitation honestly. Local development acceptance cannot imply production-grade tenant isolation.

## **2. Human approval must remain outside agent authority**

The plan introduces `artifact.decide` to approve or request rework.&#x20;

prompt18_plan.md



This requires an explicit authorization boundary.

`artifact.decide` must be a human-only Studio operation, invoked through the authenticated Focus Console. It must never be exposed as an agent-callable tool or Gateway command.

The approval record must identify the tenant, build, exact artifact digest, human approver, decision, and timestamp. The agent may propose and request rework, but cannot approve its own output.

These are foundational invariants, not optional enhancements.

## **Execution instruction**

# Implementation Authorization — Prompt18 Local-First Completion

**OWNER GO: APPROVED**

Proceed with Prompt18 implementation.

The architecture and four-phase sequence are accepted, subject to the two binding safeguards below.

## Binding safeguards

### A. Artifact execution isolation

Do not treat an ordinary subprocess, temporary working directory, GOPROXY=off, or timeout as a security sandbox.

Implement artifact verification separately from artifact execution.

Untrusted agent-produced Go artifacts must only execute inside an isolated environment with:

- No access to host credentials or environment secrets.
- Restricted filesystem access.
- Network disabled.
- CPU, memory, process-count, and wall-clock limits.
- A disposable execution environment.
- Captured output with secret redaction.

If this cannot be established locally, restrict execution to trusted repository fixtures and report artifact execution as blocked/unverified. Never weaken the security boundary to make the demo green.

### B. Human-only artifact approval

`artifact.decide` must be a human-authorized Studio control-plane operation.

It must not be exposed through the Builder agent's callable tool surface, OpenCode permissions, or Gateway commands.

Approval must be bound to:

- Tenant.
- Application/build.
- Exact verified artifact digest.
- Authenticated human approver.
- Decision and timestamp.

Enforce authorization server-side. UI visibility or agent instructions are insufficient.

## Approved execution sequence

1. Phase C backend: C1–C4.
2. Focus Console shell: C6.
3. Phase D artifact pipeline and scenes, subject to safeguards A and B.
4. Full local Invoice Review Desk rehearsal.

Preserve the plan's existing implementation boundaries, migration discipline, three logical artifact operations, testing strategy, and evidence requirements.

## Non-negotiable boundaries

- Miniapp remains the exclusive app framework.
- Go remains the exclusive application implementation language.
- MiniappSpec remains canonical.
- No alternate production application runtime.
- No Sprite CLI invocations, including reads.
- No cloud operations.
- No live OpenCode execution.
- PostgreSQL 19 at 127.0.0.1:5434 only; isolated test databases.
- No external business effects; synthetic invoice data only.
- No commits or pushes without explicit authorization.
- No F1, G-01, P5, or accepted architecture changes.

## Implementation discipline

Follow the locked Implementation Agent Guidelines.

Investigate root causes, trace complete execution paths, enforce invariants at their owning boundaries, and provide red-before/green-after tests.

At each phase boundary, report exact test evidence, browser evidence, files changed, remaining limitations, and the next dependency.

Do not conduct another broad architecture review. Escalate only concrete invariant violations or foundational blockers.

Begin Phase C backend implementation now. Stop at each defined acceptance boundary and report evidence before proceeding to the next phase.

Phase B Part 2, C5 live proof, and Phase E remain gated off.

**I authorize the agent to begin Phase C implementation immediately.**

We should not spend more time planning what we have already decided. The first milestone is durable two-way interaction; after that, the Focus Console becomes the actual interface to the product. The cloud Sprite remains idle until the local product justifies its cost.