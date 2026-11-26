PHASE D PLAN APPROVED — D0 GO

The plan is architecturally sound. I found no reason to reject Phase D or reopen accepted Boundary B decisions.

However, I want three matters resolved before implementation advances beyond D0. They are important implementation invariants, not reasons to discard the proposed architecture.

## **1. Three mandatory implementation gates**

**A. Artifact identity and bundle structure — D0**

There is one inconsistency to resolve.

Section 4 says ingestion creates an artifact row for every well-formed manifest entry. Section 5 says there is one `bundle` row as the decision unit, with member files stored inside the bundle.

Define precisely how these relate:

- What constitutes a bundle claim?
- How are individual files associated with the bundle?
- Which metadata is included in the root digest?
- How are the Studio envelope, toolchain declaration, MiniappSpec and member-file digests bound without circular hashing?

The digest must bind all security-relevant metadata as well as the exact source bytes. A changed toolchain declaration must not leave the artifact identity unchanged.



phase-d-implementation-plan.md

phase-d-implementation-plan.md



**B. Artifact transport — D1**

The proposed 50 MB artifact limit can become approximately 67 MB when base64 encoded, before JSON overhead.

Demonstrate that the existing Gateway callback and command transport can safely handle that maximum. Account for HTTP body limits, memory buffering, database payloads, retries and partial transfers.

Do not silently assume the existing fetch handler supports this workload. Introduce bounded chunking or another durable transfer mechanism if necessary.

**C. Docker execution security — D3**

The proposed isolation direction is acceptable, but `--network=none`, resource limits and read-only root filesystem are not sufficient as a complete security specification.

Require, at minimum:

- Non-root execution.
- `--cap-drop=ALL`.
- `no-new-privileges`.
- Default seccomp enforcement and appropriate AppArmor/SELinux confinement where available.
- No privileged mode, host namespaces, host devices or Docker socket.
- Read-only artifact source mount.
- Separate disposable writable scratch/cache mounts.
- Bounded process tree and reliable cleanup.

The artifact tested must remain byte-identical to the artifact verified. Re-hash before execution and prevent the test process from modifying its source files.

If the actual environment cannot demonstrate these protections, ship the honest `blocked` result. Do not weaken the boundary to achieve a green demo.

phase-d-implementation-plan.md

## **2. Additional decision to settle**

Resolve the new MiniappSpec version lifecycle before D1:

Does a proposed specification become a persisted `studio.app_specs` version before verification, or is it committed atomically with human approval?

Either model can work, but the decision must preserve a single canonical specification and prevent approval of one version while persisting another.

phase-d-implementation-plan.md

## **3. Approved execution sequence**

1.

   GO

   **D0 — Contract and boundaries.** Freeze bundle identity, transport constraints, Spec lifecycle and security requirements.

2. **D1 — Retrieval and admission.** Begin only after D0's artifact contract is frozen.

3. **D2 — Artifact verification.** Deterministic checks and durable evidence.

4. **D3 — Isolated testing.** Prove the real execution boundary or ship blocked.

5. **D4 — Human authorization.** Digest-bound, human-only decision.

6. **D5 — Focus Console integration.** Only after trustworthy backend state exists.

7. **D6 — Invoice Review Desk rehearsal.** Complete end-to-end acceptance and report.

The rest of the proposal is accepted, including the three-tool architecture, migration strategy subject to schema verification, RLS requirements, negative test matrix, and Phase E separation.&#x20;

phase-d-implementation-plan.md



## **4. Explicit authorization**

**D0 implementation is authorized.** D1–D6 are approved as the intended roadmap, subject to their dependency gates.

Preserve the local-first restrictions, the accepted uncommitted work, PG19 endpoint guard, and the human approval boundary.

No Sprite, Fly, live agent sessions, deployment, Phase E, commits or pushes.

**My architectural decision: Proceed with D0.** Once the contract and security decisions are frozen, continue in sequence without another broad planning review. Stop at the established Phase D acceptance boundary and report the evidence.