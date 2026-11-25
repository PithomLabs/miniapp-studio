**Decision: Option (a) — owner-run live execution.**

I approve `plans/prompt9_imp_plan.md` and authorize **Part 1 only**: fake-backed Phase B implementation.

This is the better execution model for three reasons:

1. Fly credentials, OAuth consent, organization selection, and infrastructure ownership remain under your control.
2. The current execution environment has no usable Fly/Sprite network path, so providing credentials would not solve the actual environmental limitation.
3. An exact, reproducible runbook creates better evidence than an attempted remote execution with uncertain permissions and connectivity.

### **Authorization boundaries**



GO — Part 1

HOLD — Part 2



**Part 1 is authorized**, including:

- Studio gateway client and fake command handler.
- Command deduplication and durable callback outbox.
- Supervisor, session manager, and OpenCode adapter against scripted doubles.
- Capability matrix with executed and unknown cells.
- Security and provisioning scripts tested against `FakeSpriteManager`.
- Pinned configuration and installation procedure.
- Full verification gates and evidence report.

The public, read-only pin verification in Step 1 is permitted. Record what was actually verified; do not float versions or silently change pins.

**Part 2 is not authorized yet.** No live Sprite provisioning, OAuth authentication, Fly API mutations, or live installation.

### **Required Part 1 handoff**

The final deliverables must include:

1. Implemented code and regression tests.
2. An honest capability matrix distinguishing implemented, tested, and unverified behavior.
3. An owner-executable live runbook containing exact commands, prerequisites, expected outputs, failure interpretation, and cleanup.
4. Explicit decisions requiring owner input, including Fly organization, model provider, and Builder retention.
5. Updated Phase B evidence with all gates and remaining unknowns.

The runbook must make it possible to verify the live acceptance sequence without guessing or inventing missing commands.

After Part 1, stop and present the evidence and runbook. I will assess the fake-backed implementation. We will authorize the live segment separately.

**No commits or pushes. F1, G-01, P5, and all existing architectural boundaries remain unchanged.**

Proceed with Part 1.&#x20;

prompt9_imp_plan.md

prompt9_imp_plan.md
