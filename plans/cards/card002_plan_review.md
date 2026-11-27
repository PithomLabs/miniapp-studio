# Review: Card 002 B2 live-proofs execution plan

**Verdict: REVISE — not yet approved for implementation.**

The plan is thorough, evidence-oriented, and generally aligned with the Card 002 scope and the governing roadmap. The stage structure, owner-terminal execution boundary, per-class authorization model, Neon/test-gate reconciliation, credential separation, N2 prohibition, N8 precautions, and explicit non-goals are sound.

Two sequencing defects must be corrected before implementation. One authorization ambiguity and two scope/evidence clarifications should also be incorporated.

## Required corrections

### 1. Split the initial GO from C0 readiness (structural)

The recommendation says GO for **S0–S5** as soon as Card 002 is authorized (plan §8, lines 150–153), but S5 explicitly requires **C0 deployed** (stage table S5, line 101). C0 is also described as the single hard blocker (§6–§8). These statements cannot all be true at once.

Amend the sequence to:

- **GO A: S0–S4 only** after Card 002 is authorized. This covers local preflight, pins/tree, the B2 test gate, binary build, and owner-held environment preparation.
- **GO C0: separate explicit owner authorization** for the Fly-hosted Studio deployment, which remains outside the runbook and must be owner-established. The plan/agent must not invent or perform a deployment procedure.
- **S5: readiness verification only after C0 deployment**. S5 proves the already-deployed Studio's listener and readiness; it does not authorize or perform deployment.
- **S6 onward:** remain gated by S5 passing and the corresponding individual C1–C6 GO at execution time.

Do not describe S0–S5 as one unconditional segment.

### 2. Split the identity and inbox proof stage (structural)

S8 currently places both service/agent identity checks and the inbox permission matrix before OAuth/provider authentication and before S12's first supervised assignment (plan §4, line 104).

The runbook's §7b says the inbox/workdir is created at first assign and directs the operator to run §10 first if needed. But §8b explicitly prohibits any agent run before provider authentication; §10 is the first model call. The plan's fallback identity proof also relies on supervised turns, so it cannot be performed before §8b.

Amend as follows:

- **S8a before OAuth:** perform only proofs available without an agent run: service EUID, agent UID/GID discovery, and direct `runuser` key-file/directory denial checks where supported.
- **After S12's first successful no-op assignment:** perform the inbox ownership/mode/write/delete/answer-file matrix, once the workdir exists.
- Any agent self-report fallback requiring a supervised turn must also occur only after §8b and the first successful model call. Label it agent-reported evidence, not kernel-observed proof.

Keep the inbox matrix as a mandatory proof; move it to the point where its prerequisites exist.

## Additional amendments

### 3. Make C0's authorization explicit

C0 is outside C1–C6, yet it is a consequential Fly deployment. Add it explicitly to the owner-decision/authorization register as its own GO. Card assignment, plan approval, or authorization of S0–S4 must not imply C0 deployment permission. Record who performs the deployment and the app/revision/URL/port evidence, without inventing the deployment procedure.

### 4. Resolve F1 before an operator relies on §12

The plan identifies a wrong §12 interpretation for the 404 probe result, but leaves the correction as non-blocking because S11 can interpret it with the plan beside the runbook. That creates two competing operator instructions.

Before S11, either correct the runbook wording or explicitly amend the execution packet so the corrected interpretation is the sole authoritative instruction. Preserve the distinction: a missing `build_id` envelope is different from an unknown non-empty build ID under the described KDF behavior. Do not claim the probe establishes build resolution.

### 5. Preserve roadmap exit honesty

The plan correctly records that real-task dispatch, checkpoint/restore demonstration, Workspace Preview, lifecycle-receipt live proof, and Tasks-API holds exceed the current runbook scope. State explicitly that **closing Card 002 does not, by itself, declare the roadmap's B2/Phase-5 exit criteria complete**. Those outstanding roadmap evidence items must remain tracked separately until actually demonstrated or explicitly dispositioned by the owner.

## Items accepted as sound

- The local-vs-`main` input baseline is explicit and avoids silently mixing versions.
- Re-auditing prep-authored changes rather than grandfathering them is the right standard.
- The focused B2 gate plus Card 001 evidence is a defensible reconciliation, provided the plan continues to state what it does not prove and keeps full-suite-green as a pre-demo gate.
- No invented Fly deployment path; C0 remains an external owner-established prerequisite.
- Owner-terminal execution, no agent-side live operations, secret non-disclosure, restricted OAuth, distinct provider/Sprite credentials, N2 prohibition, N8 precautions, and per-class GOs are appropriate.
- The runbook's limited live proof is not inflated into real-task dispatch, checkpoint/restore, or full roadmap-exit evidence.

## Review handoff

Revise `plans/cards/card002_plan.md` to incorporate Required Corrections 1–2 and Additional Amendments 3–5. Keep the plan planning-only until the revised plan is reviewed and approved. Do not execute live steps, provision resources, enter credentials, or treat this review as an authorization to perform C0 or any C-class operation.

**Approval status: withheld pending the revised plan.**

*Review scope note: this review evaluates the uploaded plan's structure, sequencing, authorization model, and consistency with the known Card 002/runbook/roadmap constraints. The plan cites a local-only prep baseline (`a40c14d`) that is ahead of `main`; the corresponding local-only source diffs were not included with this upload, so their implementation details and cited line-level claims have not been independently re-verified here.*
