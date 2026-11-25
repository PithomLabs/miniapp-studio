GO — Phase A approved

I approve `plans/prompt8_plan.md`. Proceed with one uninterrupted implementation pass. No intermediate review is needed.

**I am resolving D1 now: choose option (a), the unified&#x20;**`conversation_stream`**&#x20;table.**

Design requirements:

- One durable stream row per renderable message or agent event.
- A monotonically increasing sequence per conversation.
- Allocate the sequence transactionally with the underlying message/event insertion.
- Enforce uniqueness at the database level.
- Keep message payloads and agent-event payloads in their respective source tables; the stream stores references, not duplicate content.
- Make SSE replay use this single sequence as its `Last-Event-ID`.
- Preserve per-build event sequencing separately for Gateway protocol ordering.

This gives us reliable mixed-traffic replay across concurrent messages, multiple builds, and Studio restarts. It is preferable to relying on application code to coordinate uniqueness across two independent tables.

The evidence report path `docs/gateway-phase-a-evidence.md` is approved.

**Execution boundaries remain firm:** Phase A only; no live Sprites, OpenCode installation, OAuth, external integrations, business writes, F1/G-01 changes, or commits.

I am formally driving the development sequence as Senior Architect:

1. Phase A — Conversation Infrastructure: authorized now.
2. Phase B — Sprite Agent Gateway and OpenCode provisioning.
3. Phase C — Genuine two-way conversation proof.
4. Phase D — Build and artifact review.
5. Phase E — Approved application deployment.

Complete Phase A and report its actual evidence. I will make the Phase B authorization decision against the exit criteria—not start another general planning cycle.