# ADR-012: Webhook processing and durable execution

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Verify-then-persist-then-ack webhook pipeline: HMAC validation over raw body (V5) → delivery dedupe →
durable execution record → prompt ack → async worker execution. Per-PR serialization, idempotency keys, SHA+generation
fencing, and ownership-verified destroy. Controller restart never orphans execution state.
**Basis:** prompt28.2 §4.1, §7, §8; verified webhook/event mechanics (V2, V5, V17).

## Context

GitHub deliveries can duplicate, reorder, or go missing; controller processes restart; provisioning partially
succeeds. The pipeline must be correct under all of these, not just the happy path.

## Requirements

R205, R215–R216, R230–R231: state machines for deliveries, agent/CI executions, previews, resources; compensation for
partial provisioning; destroy only on verified ownership metadata.

## Considered alternatives

- **Synchronous execution in the webhook handler:** rejected — deliveries expect fast responses; agent/test runs take
  minutes; coupling them causes timeouts and lost work.
- **Webhook-only cleanup (close event):** rejected — missed deliveries orphan resources; reconciliation is mandatory
  (ADR-013).
- **Name-match destroy:** rejected — untrusted input must never select destruction targets; ownership metadata gates it.

## Security / operational / cost implications

Signature verification before any processing (no unauthenticated work). Dedupe bounds double-execution cost.
Persisted state enables restart recovery without re-provisioning spend.

## Failure modes

Poison-payload crash loops (mitigation: delivery quarantine + alert after bounded retries); worker starvation under
burst (mitigation: concurrency caps + queue depth alerts); fencing-clock skew (mitigation: monotonic generation, not
wall-clock ordering).

## Verification

LIVE: duplicate/out-of-order/missing delivery tests; restart-mid-execution recovery; partial-provisioning compensation;
ownership-gate negative tests (spoofed destroy refused).

## Deferred

Queue-depth autoscaling policy; multi-controller coordination (single controller assumed until scale demands otherwise).

## Developer-simplicity score

Neutral: invisible to developers when correct. Its payoff is developer trust — PRs behave deterministically and
resources never leak or vanish mysteriously.
