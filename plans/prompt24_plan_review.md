CONDITIONAL APPROVAL — ONE STRUCTURAL BLOCKER

I reviewed the implementation plan. Overall, this is a strong architectural plan. **The bootstrap-first strategy, phased schema evolution, Sprite integration, CLI-driven infrastructure operations, and separation of Preview, Staging, and Production are sound.** I would not ask for a rewrite.

However, I am withholding final approval until one important architecture gap is resolved.

## **1. Blocking issue: the deployable artifact contract**

The plan correctly requires the same immutable artifact to travel from Preview to Staging to Production. But it does not adequately define the transition from the existing verified application artifact to the actual deployable Fly Machine image.

The current Phase D artifact pipeline verifies an application bundle and tests it in a sandbox. B4 then assumes that this verified artifact can be deployed directly to Fly. Those are not necessarily the same thing.&#x20;

unified-lifecycle-ui-implementation-plan.md

unified-lifecycle-ui-implementation-plan.md



I require the plan to explicitly define this provenance chain:

**Verified Miniapp application bundle**

Source artifact digest

**Build immutable runtime image**

Pinned toolchain and base image

**OCI/container image**

Immutable image digest and source provenance

**Preview**

Fly Machine

**Staging**

Fly Machine

**Production**

Fly Machines

All three environments deploy the exact same image digest. Never rebuild separately for each environment.

The plan must establish where this image is built, stored, verified, and referenced. It must also define how the Fly CLI deploys the immutable image rather than accidentally rebuilding source during promotion.

This does not necessarily require another phase. It can be incorporated into B4 and B6, with explicit acceptance tests verifying the image digest actually running on Fly.

## **2. Two non-blocking clarifications**

**B2 dependency:** B3 requires demonstrated checkpoints, but the demo-critical path omits B2 and says it can be folded into B1/B3. Define the minimum checkpoint capability required before Sprite handoff, so the dependency is unambiguous.&#x20;

unified-lifecycle-ui-implementation-plan.md

unified-lifecycle-ui-implementation-plan.md



**Repository integrity:** The report says `.agents/skills/` and `.mcp.json` disappeared after the initial baseline. Do not assume this was an owner-side change. Investigate their status and recovery before implementation begins. Preserve any recoverable owner work.

## **3. Decision**

I approve the plan's **overall direction, phase structure, and architecture**. I require the narrow artifact-provenance correction before final approval.

There is no reason to restart the planning exercise or change the bootstrap sequence.

**Instruction to the coding agent:** Revise the existing document in place, address the immutable runtime-image provenance chain, clarify the B2/B3 prerequisite, and report the repository-integrity discrepancy. Do not implement anything or perform cloud operations.

Once those points are resolved, I expect to issue implementation GO beginning with Bootstrap 1—not jump into building the entire platform.

That is the direction I want us to maintain: **Miniapp Studio first becomes capable of managing its own development; then it progressively earns the ability to build, test, deploy, and operate itself.**