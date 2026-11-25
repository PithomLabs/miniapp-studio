Not approved yet — two foundational blockers remain

I reviewed `plan6(8).md` and cross-checked its central assumptions against the Miniapp source at the provisional `73351a8` baseline.

The proposed direction is good, but **the plan still assumes that the existing Effects implementation solves more than it actually proves.** I would not send the coding agent forward with that assumption.

## **1. The Class A transaction boundary is still misunderstood**

The plan says `Channel.Send` executes outside any transaction and proposes:

`Class A handler → Perform → claim/authorize transactions → Send → outcome transaction`

plan6(8).md

The source shows a more complicated situation:

- `ClassA.Executor.Execute` opens a PostgreSQL transaction and executes the entire handler inside it.
- The handler receives that transaction through `ToolCall.Tx`.
- `effects.Service.Perform` opens its own independent transactions for claiming and authorizing the attempt.
- `Channel.Send` executes after the authorization transaction commits.
- However, the original Class A transaction remains open until the handler returns and Class A commits.

Sources: [Class A executor](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/executor/classa/classa.go?utm_source=chatgpt.com), [Effects service](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/effects/effects.go?utm_source=chatgpt.com).

This is consequential.

Imagine that a provider successfully creates an invoice payment, and Miniapp records the effect as `CONFIRMED`. Immediately afterward, the enclosing Class A transaction fails to commit.

The external action remains successful, but the application's corresponding database changes roll back.

The independent effect transaction does not make those two outcomes atomic.

**Required correction:** Have the coding agent investigate and specify the actual transaction choreography, including this failure scenario. It must determine how external effects and local business state converge after Class A rollback, without blind retries or lost business state.

Do not simply change the wording to say that the provider call happens outside the *Effects transaction*. That would leave the actual problem unresolved.

## **2. The no-background recovery path is not yet demonstrated**

The plan proposes using the existing lease, adoption, `UNKNOWN` quarantine, `ListUnknown` and `SettleEffect` mechanisms.&#x20;

plan6(8).md



There is a gap here.

The current code establishes:

- `adoptStale` converts an old `ATTEMPTING` record to `UNKNOWN`.
- `ListUnknown` lists records that are already `UNKNOWN`.
- `SettleEffect` permits an authorized operator to settle those records.
- Stale `PENDING` records follow a different takeover path.

There is a particularly important constraint: `AuthorizeTakeover` checks the persisted effect's `invocation_id` against the current invocation ID. A subsequent Class A invocation has a different invocation identity.

Meanwhile, the ordinary same-key replay of an interrupted Class A invocation does not rerun its handler; it returns `interrupted`.

Sources: [Effects recovery](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/effects/effects.go?utm_source=chatgpt.com), [Effect authorization](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/effects/authorize.go?utm_source=chatgpt.com), [Invocation replay](https://github.com/PithomLabs/miniapp/blob/73351a801fc54e19ee7254e6fe3ea7fbed859378/runtime/service.go?utm_source=chatgpt.com).

Therefore, the plan must not assume that the existing lease automatically recovers every interrupted operation.

**Required correction:** Demonstrate recovery for both cases:

| Crash point                                          | Required recovery proof                                                         |
| ---------------------------------------------------- | ------------------------------------------------------------------------------- |
| After `PENDING` commits, before `ATTEMPTING`         | How a later invocation safely adopts or resolves the operation                  |
| After `ATTEMPTING` commits, before provider response | How the operation becomes `UNKNOWN`, is reconciled, and is never blindly resent |
| After provider success, before outcome commit        | How provider evidence determines the outcome                                    |
| After outcome commit, before Class A commit          | How local business state and the confirmed external effect converge             |

Specify the actual user/request/operator action that triggers recovery. Do not describe lazy adoption as an autonomous sweeper.

## **What I would change in plan6**

The work order should require the agent to:

1. Trace the complete nested transaction lifecycle, including the still-open Class A transaction.
2. Investigate the external-success/Class-A-rollback failure at the architectural level.
3. Determine whether the correct solution requires a split-phase Class A execution contract, a transactional effect-intent mechanism, a different recovery contract, or another demonstrably sound design.
4. Establish a real recovery mechanism for both stale `PENDING` and `ATTEMPTING` records, including cross-invocation ownership.
5. Verify lock ordering, tenant isolation, attempt fencing, idempotency and cancellation semantics.
6. Implement nothing in this revision. Identify any necessary Miniapp framework correction separately, with regression tests and conformance requirements.

Importantly, the agent should compare architectural alternatives rather than being instructed to preserve `Perform` at all costs.

## **What is already correct**

The remaining direction is sound:

- Multi-tenant readiness is mandatory from the start.
- Studio remains a first-party consumer of Miniapp.
- Miniapp and Studio evolve together when evidence warrants framework improvements.
- No background workflow engine or DBOS expansion is introduced into the Hacksgiving MVP.
- The provider-independence and authentication acceptance tests are appropriate.
- The provisional framework pin and existing approval conditions remain intact.



plan6(8).md

plan6(8).md



## **Final verdict**

**I approve the objective and documentation-only scope of plan6, but not its current technical approach.**

The two blockers are not optional robustness improvements. They determine whether the platform can truthfully claim that externally executed business operations remain safe and recoverable when the application crashes.

Resolve those two issues first. Keep the rest of the scope unchanged.

And give the coding agent this governing instruction:

> Investigate the underlying failure mechanisms, not merely the observed symptoms. Do not assume an existing abstraction solves a problem simply because its API appears suitable. Trace the actual transaction, lock, authorization, execution and recovery paths. Where the architecture is insufficient, correct the underlying design at the proper framework boundary. Require regression tests proving the invariant across crashes, retries, concurrency and tenant boundaries. Do not introduce superficial workarounds that preserve the same failure mode.

That is the standard I will apply when reviewing the next revision.