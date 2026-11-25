**Approval withheld for a few bounded structural corrections.** The residual-closure work is substantially sound, and I see no reason to reopen the accepted architecture. However, I would not approve the complete deliverable yet because the demo critical-path document has inconsistencies that could lead implementation in the wrong order.

### **Required corrections**

**1. Phase B exit requirements are missing from the critical path.**

The live runbook §15 explicitly says two items must close before Phase B exit: M5-SSE stream-lifetime/concurrency limits and `STUDIO_GATEWAY_BINARY` serve-time wiring. Neither appears in the demo document's priorities or dependency graph, which moves directly from Part 2 to Phase C.&#x20;

gateway-phase-b-live-runbook(1).md

hackathon-demo-critical-path.md



Add a Phase B exit gate between Part 2 and Phase C, or explicitly document their deferral and ensure Phase B is not declared complete.

**2. Phase C correctness requirements are under-prioritized.**

C2 (question/build/turn binding and stale-answer rejection) and C3 (payload validation and atomic handoff) are necessary for a trustworthy answer-to-same-session continuation. Yet P0 calls out C1 and C5 while turn fencing and stale-answer rejection appear under P1; C3 is not clearly prioritized at all.&#x20;

hackathon-demo-critical-path.md

hackathon-demo-critical-path.md



Make C2 and C3 P0 prerequisites for accepting C5's live same-session proof. A demo should not demonstrate a successful continuation while deferring the safeguards that establish which question was answered and which turn receives that answer.

**3. Phase D negative-test priority contradicts itself.**

D4 is part of the P0 pipeline, but negative artifact tests also appear in P1. The acceptance checklist requires every negative class to be demonstrably refused. Choose one priority and make the dependency graph and acceptance criteria consistent.&#x20;

hackathon-demo-critical-path.md

hackathon-demo-critical-path.md

hackathon-demo-critical-path.md



**4. Phase E's demo scope needs consistent treatment.**

Phase E is described as optional depending on scope and time, but the judge narrative and acceptance checklist unconditionally include independent deployment, health verification, and rollback. Make those acceptance items conditional on the owner choosing Phase E for the demo, or commit to Phase E as part of the demo's P0 scope.&#x20;

hackathon-demo-critical-path.md

hackathon-demo-critical-path.md



**5. One small residual in F-R3's exact-set reporting.**

The closure report enumerates the 15 strict migration/RLS tests, but also claims the broader 20-test family passes without enumerating its membership or defining the membership rule. Either document that broader set or remove the unsupported broader-count claim. The 354 + 1 + 1 accounting and the 14-of-16 F7 breakdown are clearly explained.&#x20;

prompt12-residual-closure-report.md



### **What I do approve**

- The N8 re-provisioning restriction is now at the relevant §6 operation, with the staging-mode, sibling-agent, and drain-first checks.&#x20;

  gateway-phase-b-live-runbook(1).md

- The N5 cold-recovery procedure records pre-sleep PIDs and requires explicit post-recovery absence checks rather than trusting `agent.failed` alone.&#x20;

  gateway-phase-b-live-runbook(1).md

- The baseline accurately distinguishes implemented, fake-backed, live-unverified, and missing capabilities. It does not pretend the sender loop or artifact pipeline exists.&#x20;

  hackathon-demo-critical-path.md

- The locked Focus Console design is explicitly treated as binding, with real backend state replacing prototype simulations.&#x20;

  hackathon-demo-critical-path.md

- The separate Part 2 GO boundary and G-01 restrictions remain intact.&#x20;

  hackathon-demo-critical-path.md


**My verdict:** the residual corrections are acceptable in substance, but the demo critical path needs these bounded alignment fixes before overall approval. No broad adversarial review or architecture redesign is warranted. The next pass should be documentation-only, preserve the existing three-file scope, and stop without live operations or commits.