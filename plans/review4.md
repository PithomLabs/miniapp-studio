# **R3 architectural review**

HOLD — focused R4 required

**R3 is a meaningful improvement and resolves the four principal R2 blockers.** I would retain its architecture and most of its task breakdown. The Builder Sprite now originates Connector reads; lifecycle operations go through Studio; public Preview exposure is separately authorized; and the `AGENTS.md` conflict now correctly stops implementation pending an owner decision.

However, I found two remaining design-level blockers. There are also two security details that should become explicit acceptance tests.

## **1. The write-authorization proof tests the wrong boundary**

This is the most important remaining issue.

R3 establishes two distinct paths:

- Sprite reads go through Fly's GitHub Connector.
- GitHub writes go through the Studio-side GitHub App.

But B2-08 currently requires the unapproved write to be refused *through the actual Connector boundary*. That proves the Sprite cannot bypass Studio by writing directly through Fly's Connector. It does **not** prove that Studio itself refuses an unapproved write through its GitHub App broker.&#x20;

issue-driven-engineering-workflow-plan(2).md

issue-driven-engineering-workflow-plan(2).md



These are two separate security properties and need two separate negative proofs.

| Attack path                                     | Required proof                                                                                  |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Sprite → Fly Connector → GitHub mutation        | Actual Connector refuses the unauthorized write                                                 |
| Sprite/agent → Studio write broker → GitHub App | Studio refuses the request before the GitHub App executes it unless the exact approval is valid |

For the second proof, test missing, expired, revoked, mismatched-target and cross-operation approvals. Also verify that an accepted approval produces only its authorized effect.

**The broker must never let the Sprite obtain the GitHub App token or call its authenticated API directly.**

### **The merge permission needs special attention**

The R3 permission matrix grants `Pull requests: write` and `Contents: write`, while merge is supposed to require a separate owner authorization.

GitHub's permission documentation lists pull-request merging among the operations accessible with write permissions. The GitHub App permission model is resource-oriented; it does not give you a simple independent permission for “create PR but never merge.”&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.github.com\&sz=32)

GitHub Docs+2



That does not invalidate the App design. It means the Studio broker must enforce the finer-grained boundary.

Add explicit tests proving that:

- PR creation and updates require their own scoped approvals.
- A merge request is refused without a separate merge approval, even though the GitHub App installation token technically has the necessary permission.
- An approval for PR creation cannot be replayed to merge it.
- The agent cannot bypass the broker to obtain the same capability.

The same principle applies to Issues and Discussions. GitHub's Discussions API, for example, supports create, update and delete operations; `Discussions: write` is not a create-only permission.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.github.com\&sz=32)

GitHub Docs+1



## **2. The unsupported-read fallback is not fully planned**

R3 correctly recognizes that Fly's Connector might not provide every required operation. It says that an unverified operation must either use a Studio-side read-only integration or be deferred.&#x20;

issue-driven-engineering-workflow-plan(2).md



The problem is that the fallback is not represented clearly in the task dependency model.

B2-03 depends on B2-02. B2-08, which describes the GitHub App and its `Actions: read-only` permission, depends on B2-03. Yet B2-03 suggests using a Studio-side read-only integration when Connector capabilities are unavailable.

That creates an unresolved question: **what actually implements the fallback, and how can it exist before the card it might depend on?**

This matters especially for the incident demo. B2-04 is specifically about reading run `45666647515`, and its evidence requirements include that CI incident. But the plan also acknowledges that Actions runs, jobs and logs may not be available through the Fly Connector.&#x20;

issue-driven-engineering-workflow-plan(2).md



R4 should choose and document one of these approaches:

**Option A — Defer unsupported operations.** Keep B2-03 strictly Connector-based. If Actions evidence cannot be obtained through verified operations, explicitly defer that capability and make B2-04's CI-specific acceptance and the corresponding demo beat conditional.

**Option B — Build an independent Studio-side read-only integration.** Define it as part of the appropriate slice, including its authorization, credentials, permission scope, evidence, recovery and dependencies. Keep it separate from the write broker so the dependency graph remains acyclic.

Do not leave an unspecified alternative in the architecture and assume it will be available when the implementation reaches it.

## **3. Make Connector security probes safe by construction**

B2-02 appropriately requires attempts to perform unauthorized writes through the actual Connector.&#x20;

issue-driven-engineering-workflow-plan(2).md



But a probe that unexpectedly succeeds could modify a real repository.

Specify that the tests use a disposable test repository or explicitly authorized canary resources, with a defined cleanup path. The probe must stop on the first unexpected successful mutation. Cleanup must have its own authorization.

This is especially important because Fly documents Connector access controls in terms of Sprite identity and request paths, not a general per-HTTP-method permission system.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



## **4. Corrections that I consider successfully addressed**

I accept the following R3 improvements:

| Area                             | Review result                                                                     |
| -------------------------------- | --------------------------------------------------------------------------------- |
| B2-01A local lifecycle contracts | Correct local-first boundary                                                      |
| B2-01B live lifecycle            | Correctly routed through Studio and Fly adapter                                   |
| Destruction of Builder Sprite    | Separate explicit authorization                                                   |
| GitHub OAuth credentials         | No Sprite-side PAT or raw token                                                   |
| B2-03 Connector caller           | Correctly identifies the Builder Sprite                                           |
| Conditional read-only Connector  | Correct security objective and fallback principle                                 |
| Workspace Preview                | Private organization-authenticated default; public exposure separately approved   |
| Checkpoint capture               | Correctly distinguishes a persistent snapshot from a live-state mutation          |
| `AGENTS.md`                      | Stops on unresolved conflicts pending owner approval                              |
| Project #3                       | Manual owner-created Draft Cards are explicit; Project V2 integration is deferred |
| Appendix A                       | No longer proposes duplicating the published initiative                           |

The Preview access model is supported by Fly's documented Sprite URL controls: URLs can remain private to the organization or be made public, and public access removes the Fly login requirement. R3's default-private requirement is appropriate.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



The permission matrix also correctly identifies `Contents` as necessary for installation-token-based HTTP Git access.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.github.com\&sz=32)

GitHub Docs+1



## **Final decision**

**R3 is architecturally close, but I am not approving it as the implementation baseline yet.**

The remaining work is focused, not a redesign:

1. Separate the Connector's write-denial proof from the Studio App broker's approval-enforcement proof.
2. Explicitly enforce the separate merge-approval boundary despite the App's broader GitHub permissions.
3. Resolve the unsupported-read fallback and B2-03/B2-08 dependency ambiguity.
4. Make live Connector write probes safe by construction.

Once those are incorporated, I expect the plan to be reviewable for acceptance without another broad architectural revision.

**Implementation remains unauthorized.** B2-01A is the proposed first implementation task, not an approved task. No GitHub writes, cloud provisioning, App registration or other consequential operations are authorized by this review.