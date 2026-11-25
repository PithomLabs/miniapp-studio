APPROVED — C6 GO

The plan is implementation-ready. **No foundational or structural blocker prevents C6 from proceeding.** The sequencing is sound: correct F2–F6 before exposing state in the Console, build the server-rendered shell, wire persisted backend state, then complete the browser acceptance matrix. The proposed migrations and the decision to leave Phase D/E scenes honestly unfinished are consistent with the agreed boundaries.&#x20;

prompt20_plan(1).md

prompt20_plan(1).md



I approve implementation with four guardrails. These are bounded execution requirements, not reasons to reopen planning:

1\. **Keep the Run stage honest.** A successful or active Builder agent run is not proof that an approved Miniapp is deployed and running. Since Runtime deployment belongs to Phase E, do not mark the product's Run stage green from `build.status` alone. Keep it explicitly unproven until there is actual Runtime evidence; Builder activity belongs in Pulse.&#x20;

prompt20_plan(1).md



2\. **Keep&#x20;**`build.cancel_all`**&#x20;operator-only.** It must be a Studio control-plane action authenticated through the operator session, with the tenant derived server-side. Do not expose it to OpenCode, agents, MCP tools, or Gateway commands. Its UI must state clearly that it cancels every non-terminal build in that tenant.&#x20;

prompt20_plan(1).md



3\. **Make the F2 regression test reproduce the actual failure mode.** Simulate an ambiguous dispatch where the first request may have been accepted but its response is lost; insert a new user message before retry; then assert identical wire bytes and one logical execution. A failure before the request reaches the Gateway would not fully test the reported defect.&#x20;

prompt20_plan(1).md

prompt20_plan(1).md



4\. **Make dispatch visibility precise.** The plan appends Pulse dispatch rows only for `failed` and `exhausted`, while describing Pulse as showing terminal dispatch outcomes. Ensure successful acknowledgments are still visibly represented by a correlated persisted event, or include them in dispatch stream coverage. Do not imply the dispatch stream is exhaustive if it is failure-only.&#x20;

prompt20_plan(1).md



**Execution authorization:** Proceed with **U1 → U2 → U3 → U4 → U5**. Preserve the local-first boundary: no Sprite CLI, Fly operations, live OpenCode, Part 2/C5, Phase D/E implementation, commits, or pushes.

At Boundary B, stop and submit the implementation report with test evidence, browser evidence, and remaining residuals. No additional planning review is required before starting U1.&#x20;