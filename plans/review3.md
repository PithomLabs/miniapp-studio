R2 is a substantial improvement: the PAT-in-Sprite design is gone, GitHub writes are separated from Sprite-side reads, lifecycle work is split into local and live stages, restore has an explicit approval gate, and the missing Discussions workflow is represented.

**My architectural decision is not yet GO.** Two important execution-boundary problems remain in the plan itself: B2-03 still places Connector reads in the Studio server rather than in the Builder Sprite, and B2-01B does not clearly prove that live Sprite lifecycle operations pass through Studio’s own authorized control plane. I’m checking those against the surrounding requirements, along with the remaining exposure and documentation gates, before deciding whether R2 can be accepted.&#x20;

issue-driven-engineering-workflow-plan(1).md

issue-driven-engineering-workflow-plan(1).md



The Fly documentation confirms the credential-handling improvement: GitHub OAuth credentials stay in Fly’s control plane, and the Connector is invoked from inside an authorized Sprite. It also describes Connector access restrictions in terms of request paths and Sprite identity—not HTTP methods. R2 therefore still needs to prove that a supposedly read-only GitHub Connector cannot perform writes; simply naming it “read-only” does not establish that security property.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



There is a related capability assumption to close: Fly’s published GitHub Connector guide explicitly lists repository, Issue and pull-request access, but does not document Discussions or Actions run/job/log access. B2-03 currently treats all of those as available without making their live verification a prerequisite.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



## **Architectural review: R2 — HOLD**



R3 revision required

Implementation not authorized



I confirm that the attached document reflects the major R2 corrections: OAuth instead of a Sprite-side PAT, a Studio-side write broker, local-first lifecycle work, separately authorized restoration, a Discussions write path, corrected dependencies, and a more careful `AGENTS.md` reconciliation. The plan retains its eight sections and has eleven proposed cards, including B2-01A/B and B2-09.

However, R2 fixes the previous design problems without fully closing several execution and security questions. I would not make this the implementation baseline yet.

### **Four blocking issues**

**1. B2-03 puts Connector calls in the wrong place.**

B2-03 currently says:

> Location: Studio server via Connector.

That conflicts with how Fly Sprites Connectors work. The Connector proxies requests made from inside an authorized Sprite; the control-plane credential is injected when the request is forwarded. It is not described as a Connector API that an ordinary Studio server can invoke.&#x20;

issue-driven-engineering-workflow-plan(1).md

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



The execution path should be explicit:

`Builder Sprite agent → Fly GitHub Connector → GitHub → sanitized evidence returned to Studio`

B2-03 should locate the actual read operations inside the Sprite. Studio should receive and persist the resulting evidence, not pretend to be the Connector caller.

**2. The read-only Connector boundary is asserted, not demonstrated.**

R2 says the Sprite Connector stays read-only. That is the correct security objective, but the proposed implementation does not yet prove it.

Fly documents Connector restrictions as allow/block patterns on request paths. Its own Sprite product material describes GitHub access that can include reading and pushing code. The public documentation does not establish that the path policy can distinguish HTTP methods.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



This matters because a GitHub API route may support both reading and changing the same resource. A path restriction alone is not evidence that every write method is blocked.

B2-02 and B2-08 need a concrete acceptance test:

- Identify every GitHub endpoint and operation the Sprite needs.
- Test the actual authorized Connector, not a fake or direct API substitute.
- Attempt unauthorized write operations through that Connector, including operations on routes also used for reads.
- Record which operations are allowed and refused.
- If a reliable read-only boundary cannot be demonstrated, do not describe the Connector as read-only. Restrict it to demonstrably safe operations and move the affected reads behind a Studio-side, read-only integration.

The required unapproved-write negative proof must test the real Connector boundary, not just prove that the Studio GitHub App refuses an unauthorized request.

**3. B2-03 assumes GitHub Connector capabilities that are not yet documented.**

The current Fly GitHub Connector guide advertises access to repositories, Issues and pull requests. It does not document the full set of Discussions, Actions runs, jobs and logs that B2-03 promises.&#x20;

issue-driven-engineering-workflow-plan(1).md

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



This does not prove those other operations are unavailable. It does mean the plan must not treat them as established capabilities.

Make B2-02's live verification an explicit gate for every operation B2-03 needs. B2-03 should proceed only with operations actually verified, and it needs a defined alternative for anything unsupported. The incident demo must not depend on an unverified Actions/logging capability.

**4. B2-01B does not prove Miniapp Studio controls the live Sprite lifecycle.**

B2-01B currently specifies:

> Location: owner terminal + Fly API.

That can prove Fly's provider-side lifecycle operations work. It does not prove that Miniapp Studio can provision, wake, sleep, connect or disconnect the Builder Sprite through its own authorization and operation boundaries.&#x20;

issue-driven-engineering-workflow-plan(1).md



The purpose of Miniapp Studio is to manage the engineering process. The live proof should therefore exercise the Fly adapter through Studio's lifecycle operations, with durable receipts and server-side authorization enforcement. A direct owner-terminal bootstrap may be a separately identified prerequisite, but it must not be presented as proof of Studio-managed lifecycle.

There is also a specific authorization gap: the recovery path includes destroying and reprovisioning the Sprite, while the card currently says only “per-operation owner GO.” Destruction needs its own explicit authorization bound to the exact Sprite and effect.

### **Additional corrections before the relevant cards execute**

**Workspace Preview needs an access policy.**

B2-07 authorizes a Sprite URL as a task-scoped development surface, but does not define its access mode or data restrictions. Fly supports private organization-authenticated URLs and public URLs; public mode allows anyone with the URL to access the served web application without a Fly login.&#x20;

issue-driven-engineering-workflow-plan(1).md

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly



Require private access by default, synthetic or sanitized data, and no production credentials or private production data. Making a URL public should require separate, explicit authorization for that exact exposure. Acceptance evidence should verify the actual URL access mode, not merely the presence of a URL.

**The GitHub App permission design needs a precise permission matrix.**

The plan correctly calls for least privilege, but the App permissions are not yet sufficiently specified for the proposed write operations. In particular, §3 includes branch pushes and PR creation, while B2-08 mentions Issues, Discussions and pull-request grants without explicitly addressing repository Contents permission.

GitHub documents that installation-token-based HTTP Git access requires the `Contents` repository permission. GitHub Apps also have distinct Issues and Discussions permission levels.&#x20;

issue-driven-engineering-workflow-plan(1).md

issue-driven-engineering-workflow-plan(1).md

[image](https://www.google.com/s2/favicons?domain=https://docs.github.com\&sz=32)

GitHub Docs+1



Before implementing B2-08, specify the minimum permissions for each operation, repository installation scope, and which permissions are deliberately excluded. Keep the App private key and installation credentials exclusively server-side, outside the Sprite, agent context, repository and Studio database.

**The proposed&#x20;**`AGENTS.md`**&#x20;reconciliation cannot govern implementation before approval.**

The plan says that, until the owner accepts the reconciliation, its new statements govern new work while the cited `AGENTS.md` passages govern their original scope. That is not a safe resolution of an unapproved conflict. The plan itself is still Proposed.&#x20;

issue-driven-engineering-workflow-plan(1).md



Until the owner approves the amendments, an implementation task that encounters one of these conflicts must stop and request a decision. The proposed plan cannot silently supersede the existing governing instructions.

Two smaller clarifications would also improve the plan:

- Explicitly state that Project #3 Draft Task Cards are manually created by the owner for the initial workflow; any Project V2 integration is a separate, deferred and authorized capability.
- Replace “non-mutating” for checkpoint capture with “non-destructive, task-scoped.” Capturing a Sprite checkpoint creates a persistent snapshot; Fly describes it as a backup of the environment, not an operation with no side effects.&#x20;

  issue-driven-engineering-workflow-plan(1).md

  [image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

  Sprites · Fly



## **Decision and next step**

The R2 architecture is materially better, and I do not recommend restarting the plan. I recommend a **focused Prompt27.1-R3 plan-only revision** that preserves the work already done and addresses the remaining issues above.

The critical acceptance tests are:

1. Genuine Connector reads originate inside the Builder Sprite.
2. The actual Connector's write-denial properties are demonstrated, or the design falls back to a provably safe read path.
3. Every required GitHub operation is live-verified before being promised by B2-03.
4. The live Fly lifecycle is exercised through Studio's own authorized adapter and operation boundary.
5. Workspace Preview is private by default, with a separate approval for public exposure.
6. GitHub App permissions and the `AGENTS.md` conflict rule are explicit.

I can verify those issues in the attached plan text. I cannot independently verify the agent's reported working-tree cleanliness or absence of side effects from this document alone.

**Final verdict: R2 is not rejected as an architecture direction; it is not yet accepted as an implementation baseline. No implementation authorization.**