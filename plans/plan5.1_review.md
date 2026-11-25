Understood. **Approval means approve when there are no foundational or structural blockers.** It does not mean demanding perfection or holding up the project for optional improvements. I will apply that standard.

Having reviewed the revised 605-line plan, my verdict is:

Not approved yet — one foundational execution contract remains unresolved

The six revisions are incorporated coherently, and the overall architecture is substantially stronger. I agree with the verified approval-resumption mechanism, the URL ruling, provider separation, and the minimum authentication model.

However, I found a material inconsistency between three sections.

## **1. External execution versus the Class A transaction**

Section 4.2 defines Class A as executing its handler inside a single PostgreSQL transaction, with synchronous request/response execution and no background jobs.&#x20;

miniapp-studio-implementation-plan-consolidated(1).md



Section 4.5 correctly requires external operations to happen outside that transaction.&#x20;

miniapp-studio-implementation-plan-consolidated(1).md



But §6.3 specifies:

`Agent → capability tool → runtime.Service.Invoke → trusted adapter → gateway`

It does not explain how the trusted adapter executes the external request outside the Class A transaction while retaining the framework's authorization, invocation and audit guarantees.&#x20;

miniapp-studio-implementation-plan-consolidated(1).md



These requirements cannot simply be assumed to coexist. This needs an explicit execution contract.

**Required resolution:** Define the framework-controlled external-effect boundary. For example, `Invoke` could authorize and durably commit an operation intent, after which a trusted dispatcher executes the provider call outside the transaction and records its outcome. Specify how this remains synchronous for the MVP, or how a subsequent invocation resumes it without introducing an unauthorized background workflow engine.

## **2. The external-operation attempt sequence needs one correction**

Section 4.5 currently says to execute the external call and then commit the attempt record.&#x20;

miniapp-studio-implementation-plan-consolidated(1).md



That leaves an important crash window.

The durable sequence should explicitly be:

1. Commit operation intent.
2. Commit attempt-start record, including stable operation identity.
3. Execute the external request.
4. Commit the outcome.
5. On an ambiguous result, record `UNKNOWN` and reconcile before any retry.

Otherwise, a crash immediately after the provider accepts a request but before the attempt record is committed leaves recovery without reliable evidence of what was attempted.

Also specify how the reconciliation sweeper is triggered, since §4.2 prohibits background jobs in the MVP. The lifecycle and its recovery mechanism must agree.

## **3. Two non-blocking corrections**

These do not require architectural redesign, but should be corrected in the next revision.

**Provider-independence test:** §12.1 requires byte-identical specs from the same generation inputs across different providers.&#x20;

miniapp-studio-implementation-plan-consolidated(1).md



That tests cross-provider generation determinism, not provider independence. Test instead that changing the selected provider does not mutate an already-persisted MiniappSpec or its content hash.

**Authentication acceptance coverage:** §8.1 defines useful ownership and session guarantees, but the G0–G12 table does not explicitly require tests for session expiry/revocation, unauthenticated access, workspace ownership and cross-tenant denial. Add those to an appropriate gate before exposing the Studio UI.&#x20;

miniapp-studio-implementation-plan-consolidated(1).md

miniapp-studio-implementation-plan-consolidated(1).md



## **What is already acceptable**

I see no reason to reopen the broader architecture:

- The same-key approval resumption path is clearly specified.
- The Miniapp/Studio joint-evolution principle is appropriate.
- The provider-neutral application definition is correct.
- The separation of Miniapp's framework responsibilities from Studio's product responsibilities is sound.
- The Hacksgiving cut line remains controlled.
- The framework pin remains provisional, and the outstanding P0 tests correctly prevent dependent implementation from relying on it.&#x20;

  miniapp-studio-implementation-plan-consolidated(1).md



**Decision:** Resolve the external-effect execution boundary and attempt-start durability. Fix the two acceptance criteria while doing so. No broader redesign is warranted.

Once those are corrected, I would expect to approve the plan without requiring another architectural overhaul.


## take note

also take note that the default setup is that Miniapp Studio should be multi-tenant ready

Absolutely. **Multi-tenant readiness is a default architectural requirement for Miniapp Studio, not a future enhancement.**

There is an important distinction I want preserved:

- **Multi-tenant-ready from day one:** mandatory.
- **Full multi-tenant administration and organization-management UI:** can remain outside the Hacksgiving MVP.

The current plan already establishes a one-to-one mapping between a Studio workspace and a Miniapp tenant, but limits the MVP to one operator identity per workspace. These are compatible with multi-tenant readiness, provided the deployment can serve multiple independently isolated workspaces.&#x20;

miniapp-studio-implementation-plan-consolidated(1).md



### **Requirements to carry into the next revision**

1\. **Tenant isolation:** Every app, specification, version, execution, approval, effect, audit record, connector binding and deployment resource must be tenant-scoped.

2\. **Server-enforced identity:** Tenant context must derive from authenticated server-side identity, never from an arbitrary tenant ID supplied by the browser or agent.

3\. **Data isolation:** Database queries, relationships, uniqueness constraints and resource lookups must prevent cross-tenant access by construction.

4\. **Infrastructure isolation:** Fly resources, Sprite identities, connector permissions, credentials and app URLs must be mapped to their owning tenant. Shared infrastructure must never imply shared authority.

5\. **Execution isolation:** Miniapp authorization, capability evaluation, approvals and audit must preserve tenant context across every execution boundary.

6\. **Independent lifecycle:** Tenant app creation, deployment, updates, failures and retirement must not corrupt or expose another tenant's resources.

7\. **Regression tests:** Explicitly test cross-tenant reads, writes, invocation, approval, connector access, URL resolution and resource ownership.

### **MVP interpretation**

The MVP can still have one operator per workspace. It does not need invitations, teams, complex RBAC, organization administration or a tenant-management dashboard.

However, the underlying architecture must support multiple workspaces and tenants simultaneously, with enforced isolation.

**I am treating this as a governing requirement alongside the joint evolution of Miniapp and Miniapp Studio.** It should influence the architecture, data model, authentication contract, framework improvements and acceptance gates—not be added as a superficial checkbox later.