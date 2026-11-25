Approval withheld — structural revisions required

I reviewed the complete implementation plan. **The overall architecture is sound, and I agree with its product direction.** The separation between Studio, the Miniapp framework, the declarative specification, the generic runner, and Fly infrastructure is coherent. The plan also correctly distinguishes implemented framework capabilities from missing functionality, keeps DBOS out of the MVP, and treats the framework dependency pin as provisional rather than pretending its compatibility has been fully verified.&#x20;

miniapp-studio-implementation-plan.md

miniapp-studio-implementation-plan.md



However, I see **three structural issues that should be resolved before implementation approval.**

## **1. Approval does not have a defined execution-resumption contract**

Section 4.2 specifies:

- Class A synchronous execution.
- An invocation that can return `waiting_approval`.
- A separate `DecideApproval` call.
- A subsequent `Status` query.
- No DBOS wake, background worker, or other asynchronous execution mechanism.

What is missing is the actual transition from *approved* to *executed*.

Suppose a user asks the Invoice Review Desk to perform an operation requiring confirmation. The operation parks for approval. The user approves it. What executes the original operation after that decision?

The plan currently describes recording and querying approval, but not resuming the business operation.&#x20;

miniapp-studio-implementation-plan.md



**Required revision:** Choose and document one explicit contract:

- **Recommended for the MVP:** Approval authorizes a subsequent, separately submitted invocation. The new invocation references the approval decision and has defined idempotency semantics.
- Alternatively, implement a verified continuation mechanism that resumes the original invocation after approval.

Do not imply that `DecideApproval` followed by `Status` automatically resumes execution. Add tests proving that no protected action executes before approval, that approval leads to exactly the intended execution path, and that duplicate decisions or submissions cannot cause duplicate actions.

## **2. The Class A transaction boundary is not reconciled with external side effects**

This is the most important architectural issue.

Section 4.2 describes a single-transaction Class A execution model. Section 3 assigns deployment, URL management and connector operations to Studio tools. Section 5 requires actual Fly resource provisioning and reconciliation.&#x20;

miniapp-studio-implementation-plan.md

miniapp-studio-implementation-plan.md

miniapp-studio-implementation-plan.md



These operations cross two independent transactional systems:

- PostgreSQL, which stores application state and execution records.
- External systems, including Fly and connector providers, which cannot participate in the PostgreSQL transaction.

Consider this failure:

1. Studio requests creation of a generated app's Sprite.
2. Fly successfully creates the Sprite.
3. Studio crashes before PostgreSQL records its URL and resource identity.
4. The database transaction rolls back.

The external resource still exists. PostgreSQL cannot roll back that creation.

The plan mentions orphan reconciliation, `UNKNOWN` effects and avoiding blind retries, which is good. But it does not define the durable mechanism that makes those guarantees possible.&#x20;

miniapp-studio-implementation-plan.md

miniapp-studio-implementation-plan.md



**Required revision:** Define a durable external-operation lifecycle independently of the Class A transaction. At minimum, specify:

- Persisted operation intent before external execution.
- Stable idempotency/resource identity.
- Explicit provisioning and failure states.
- Separately committed attempt and outcome records.
- Recovery after process crashes and ambiguous API responses.
- Reconciliation of externally created resources that were not committed locally.

This does not require introducing DBOS into the MVP. It does require acknowledging that a PostgreSQL transaction cannot make a Fly API operation atomic.

Also, keep the MVP's actual connector operations read-only unless the external-effect lifecycle is proven. Explicit approval alone does not solve the transaction problem.

## **3. The Sprite gateway call conflicts with the dogfooding requirement**

There is a tension between two requirements:

- G1 requires the Sprite-hosted agent to make a real gateway connector call.
- G8 requires consequential operations to go through `runtime.Service` and its audit path.

The security section also says the framework owns authorization and that the agent is untrusted.&#x20;

miniapp-studio-implementation-plan.md

miniapp-studio-implementation-plan.md

miniapp-studio-implementation-plan.md



If the Sprite agent directly calls the gateway, that request appears to bypass the framework's canonical invocation and audit path. Per-Sprite gateway restrictions are useful, but they are not automatically equivalent to Miniapp authorization, policy evaluation and audit.

**Required revision:** Draw and specify the actual call path.

My recommended design:

1. The Sprite agent requests a connector operation through a narrowly scoped capability tool.
2. That request enters the Miniapp runtime through `runtime.Service`.
3. The framework validates the declared operation and applicable policy.
4. The trusted adapter executes the gateway request using Fly's credential-handling mechanism.
5. The result is recorded against the invocation and returned to the agent.

The agent can still demonstrate real connector usage from its Sprite environment without gaining authority to bypass the framework. G1 and G8 should test this same end-to-end path.

## **Additional decisions to make explicit**

These are not independently fatal, but should be settled in the plan.

**Per-app URL semantics.** The shared-router option uses a slug path on one host, whereas the other options provision separate resources. That distinction matters because the acceptance criteria require a stable, persisted, Fly-issued URL for each app. A shared route should not be treated as equivalent to an independently provisioned Fly URL without explicitly changing that requirement.&#x20;

miniapp-studio-implementation-plan.md



**Agent provider independence.** The roadmap correctly establishes four independent dimensions, but putting provider references in MiniappSpec metadata or policies risks coupling agent-provider configuration to the application definition. Prefer storing generation-provider selection in a separate generation/session configuration. The generated application's specification should remain unchanged when the agent provider changes.&#x20;

miniapp-studio-implementation-plan.md



**Authentication and ownership.** The plan references browser session authentication but does not fully define identity, app ownership, session security or tenant isolation. This needs a minimum viable contract before the UI is exposed to real users.&#x20;

miniapp-studio-implementation-plan.md



## **What I approve already**

The following architectural decisions are correct and should be preserved:

- Separate Studio from the pre-existing Miniapp framework.
- Use Miniapp as a pinned dependency, not copied source.
- Make Studio itself a first-party Miniapp using `runtime.Service`.
- Generate declarative, immutable, content-hashed specifications.
- Use one statically registered generic interpreter rather than executing arbitrary generated code.
- Keep the Hacksgiving MVP Class A and avoid pretending it has DBOS-grade durability.
- Keep managed Gemini, Claude and other agent providers on the roadmap behind provider-neutral interfaces.
- Build one complete, genuinely working Invoice Review Desk rather than twelve superficial demo applications.
- Treat actual Fly and connector behavior as acceptance-test requirements, not assumptions.&#x20;

  miniapp-studio-implementation-plan.md

  miniapp-studio-implementation-plan.md



## **Verdict**

**I approve the architectural direction, but not the implementation plan in its current form.**

The three issues above are not requests for more features. They concern fundamental execution contracts: what happens after approval, how external effects survive crashes, and who actually controls connector execution.

Resolve those contracts in the document, update the relevant acceptance tests and gate definitions, and retain the existing scope. There is no reason to redesign the entire platform or introduce DBOS prematurely.

One final existing condition remains: the framework pin must stay provisional until the PostgreSQL-backed CI suite and Studio import/approval spike in §2.1 actually pass.&#x20;

miniapp-studio-implementation-plan.md



**My recommendation: revise these specific contracts, then submit the plan for final approval.**