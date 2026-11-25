Got it. This is a standing requirement for all coding-agent advice and implementation prompts:

**Solve the underlying problem, not merely the symptom.**

I will require the coding agent to investigate root causes and system-level failure modes, trace the complete execution path, preserve architectural invariants, add regression tests that demonstrate those invariants, and implement a durable correction. A workaround, superficial patch, or wording-only change is not an acceptable resolution when the underlying mechanism remains defective.

I will apply that standard to the current external-effect boundary review as well.

**Verdict: Not GO yet. One foundational assumption in the proposed approach needs correction before the coding agent starts.**

I verified the relevant source at the provisional Miniapp baseline, `73351a8`.

### **1.&#x20;**`effects.Service.Perform`**&#x20;does not fully solve the Class A transaction boundary**

The existing implementation does establish useful independent effect-ledger transactions:

- `claim()` commits `PENDING` with a framework-generated attempt ID.
- `authorizeAttempt()` performs authorization and commits the transition to `ATTEMPTING`.
- `Channel.Send()` executes between those transactions.
- `transition()` commits the outcome.

That sequence is real and reusable. See [effects.go](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/effects/effects.go?utm_source=chatgpt.com#L176-L229) and [authorize.go](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/effects/authorize.go?utm_source=chatgpt.com#L122-L150).

However, there is an important distinction.

`ClassA.Executor.Execute()` opens a PostgreSQL transaction and invokes the entire business handler inside it, passing that transaction as `ToolCall.Tx`. [Source](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/executor/classa/classa.go?utm_source=chatgpt.com#L19-L83)

Consequently, when the handler calls `effects.Perform()` synchronously, `Channel.Send()` executes outside the effect service's own transactions, **but the enclosing Class A transaction is still open**.

That has two consequences:

- The database transaction and its execution/entity locks remain held during the provider request.
- The provider can successfully perform an external action even though the enclosing Class A transaction subsequently rolls back or fails to commit.

Therefore, the proposed statement that `Channel.Send` executes outside *any* transaction is incorrect.

This is not necessarily a reason to discard the existing Effects service. It is a reason to define its actual relationship with the Class A executor and explicitly handle the resulting failure semantics.

### **2. The no-background recovery assumption also needs verification**

The current recovery implementation has an important limitation:

- A stale `ATTEMPTING` record can be quarantined through `adoptStale()`.
- `ListUnknown()` lists records already in `UNKNOWN`; it does not discover or quarantine stale records itself.
- `SettleEffect()` settles quarantined outcomes; it is not a general stale-operation sweeper.
- Stale `PENDING` takeover has an invocation-binding condition that must be examined when a subsequent attempt has a different invocation ID.

Sources: [adoption and recovery](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/effects/effects.go?utm_source=chatgpt.com#L390-L449), [ListUnknown](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/effects/effects.go?utm_source=chatgpt.com#L523-L549), [SettleEffect](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/runtime/lifecycle.go?utm_source=chatgpt.com).

The revised plan must establish exactly how a crashed Class A invocation can recover both stale `PENDING` and `ATTEMPTING` operations without relying on an unavailable background worker.

## **Required adjustment to the coding-agent instructions**

I recommend adding these requirements to `plan5.1_review.md` before proceeding:

1. Trace the complete nested transaction lifecycle: Class A executor → handler → `Perform` → authorization → provider send → outcome → Class A completion.
2. Establish whether the external provider call should be dispatched after the Class A transaction commits, or whether the existing synchronous nested-transaction arrangement can be safely retained under an explicitly bounded and tested contract.
3. Walk through provider success followed by Class A transaction failure. Demonstrate how the business state, effect ledger and subsequent invocation converge without losing or duplicating the operation.
4. Verify recovery for stale `PENDING` and `ATTEMPTING` records when the original Class A invocation has been interrupted and a subsequent invocation has a different identity.
5. Do not invent a sweeper, endpoint, recovery guarantee or public API that does not exist. Identify necessary Miniapp changes separately, with the correct ownership and regression tests.
6. Preserve tenant isolation across every recovery and operator path.

**Do not solve this by merely changing the wording from “outside any transaction” to “outside the effect transaction.”** The actual failure modes must be addressed.

The implementation plan should also retain the previously established multi-tenant-ready requirement and the joint evolution of Miniapp and Miniapp Studio.

Approval pending

I approve the direction of this documentation-only revision, but these two execution/recovery questions are foundational. Resolve them with source evidence and a crash-boundary analysis before resubmitting. No P1/P3 implementation, dependency changes, or commits are authorized by this review.

The standard for the coding agent remains: **find the underlying failure mechanism, correct it at the proper architectural boundary, and prove the correction with regression tests—not a workaround that merely makes the immediate scenario pass.**