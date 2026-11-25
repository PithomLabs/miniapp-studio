## gemini

### Adversarial Review of `writeup2.md` (Multi-Tenant SaaS Lens)

Viewing Miniapp Studio as a **multi-tenant SaaS** changes the threat model and operational stakes: untrusted customer repos, arbitrary generated code, and hostile/compromised agent runs now happen across **multiple isolated tenants** sharing Studio infrastructure and Fly.io resources. While the core separation—keeping Studio as the authoritative control plane with PostgreSQL, isolating execution in Builder Sprites, and promoting only approved artifacts to Runtime Sprites—is sound, several multi-tenant failure modes stand out:

---

#### 1. Multi-Tenant Isolation & Security Risks

* **Inactive RLS on Phase A Tables Is a Direct Multi-Tenant Hazard:**
* Having `FORCE ROW LEVEL SECURITY` without `ENABLE ROW LEVEL SECURITY` on ten Phase A tables leaves database-enforced tenant isolation turned off at the engine level. Relying solely on application-level tenant predicates leaves zero safety net if a single query or new code path omits a tenant filter.




* **Intra-Sprite Privilege Separation (`studio-gw` vs. `studio-agent`) Has High Operational Fragility:**
* Running `studio-gw` and `studio-agent` as separate Unix users with a shared `studio-io` handoff folder inside a single Builder Sprite is a classic multi-tenant hardening trap. Any misconfigured service startup, shared writable path, symlink/TOCTOU race, or unexpected capability/privilege path on the Sprite weakens that boundary—and if provisioning fails closed when separation isn't supported, tenant onboarding/builds can stall unpredictably.




* **Secret Leakage Across Multi-Tenant Logs, Manifests & Read Paths:**
* Write-path regex scrubbing for specific token prefixes (OpenAI, AWS, Slack, GitHub) helps on `build.ingest_event`, `build.status`, `conversation.get`, and `conversation.stream`, but multi-tenant SaaS environments leak secrets through encoded/custom tokens, `artifacts.manifest` (deferred to Phase D), execution transcripts, and unredacted `app.invocations` records.




* **Cross-Tenant & Blast-Radius Risks on Fly MCP, Connectors & Pending Governance Holes (`F1`, `G-01`):**
* Scoped Fly MCP grants (`max_sprites_total = 0`, no network/checkpoint mutation) and keeping production connector bindings out of Builder Sprites are essential, but any over-broad token, shared resource pool, or unclosed `F1` / `G-01` governance gap creates cross-tenant or unauthorized-action blast radius.


* Additionally, a coding agent inside a tenant's Builder Sprite can write malicious or data-exfiltrating logic into generated code unless Phase D inspection/validation and Phase E Runtime Sprite isolation are strict.





---

#### 2. Distributed State, Protocol & SaaS Scale Hazards

* **Multi-Store Drift Across Tenants (Studio Postgres vs. Builder Filesystem + Outbox + OpenCode Session):**
* State is split across Studio PostgreSQL, the Builder Sprite's persistent filesystem, OpenCode's session files, and the gateway's callback outbox. Across many concurrent tenants, Sprite sleep/wake cycles, restarts, delayed callbacks, or duplicate commands can cause stale events to arrive after a turn is superseded or cancelled. Without strict **tenant + build + turn/epoch fencing**, "state guards" alone can drift under concurrency.




* **Unproven OpenCode Noninteractive Q&A Contract:**
* `opencode run --format json` and session continuation for structured Q&A is still unverified live, and falling back to a `.studio/inbox/` JSON file handoff adds a second state-synchronization path with crash/partial-write edge cases.




* **Unbounded SSE Connections in a Multi-Tenant Web App:**
* SSE streams without per-tenant/per-user concurrency caps and max connection lifetimes can exhaust connection pools or file descriptors during reconnect storms or high tenant concurrency.




* **Prematurely Treating Phase A as Complete:**
* Claiming Phase A readiness before RLS was actually enabled or server restarts were tested via real shutdown-and-boot shows a gap between test scripts and production-grade multi-tenant guarantees.





---

#### 3. Complexity vs. SaaS Value

* **Heavy Upfront Protocol & Interaction Surface Before End-to-End Proof:**
* Designing a broad gateway command/event vocabulary, three distinct user interaction modes (`Clarification`, `Proposal`, `Authorization Request`), and dual Q&A mechanisms before proving a single real OpenCode Q&A + resume + artifact round-trip inside a Sprite adds complexity before validating the core loop.





---

### What I Would Do Differently: Simpler, More Effective for Multi-Tenant SaaS

To make this **simpler to operate across tenants and more resilient against real failures**, I would adjust the plan as follows:

1. **Enforce Hard Multi-Tenant Boundaries at the Data & Infrastructure Plane, Not Inside the Agent Sandbox:**
* Immediately apply migration `000016` with both `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY` on all ten Phase A tables, and add automated tests running as the **restricted application DB role** to prove cross-tenant denial.


* Keep signing keys and tenant secrets out of the Builder Sprite's local filesystem as much as possible (sign/verify at Studio boundaries, use short-lived scoped credentials where needed), and use **one-way/immutable staging** for any file exchange rather than relying on fragile intra-Sprite Unix user splits plus shared mutable folders.


* Enforce strict **per-tenant and per-user SSE stream limits + max lifetimes** right away.




2. **Simplify the Gateway Protocol & Pick One Q&A Mechanism:**
* Keep the gateway contract lean (`assign`, `answer`, `cancel`, `status`, `collect_artifact`) with **monotonic `turn_id` / `epoch` fencing** on every command and callback so out-of-order or duplicate callbacks across sleeping/waking Sprites are rejected cleanly.
* Commit to **one** primary Q&A mechanism for the initial live integration (testing native JSON event + session resume first, and switching cleanly to a single atomic handoff if needed rather than carrying dual machinery from day one).




3. **Treat All Agent Output & Logs as Untrusted Across Tenants:**
* Don't rely on regex prefix lists as a complete secret defense; treat agent messages, manifests, and transcripts as untrusted/potentially sensitive at read/display boundaries, and close `F1` and `G-01` before enabling consequential external business effects.




4. **Prove One End-to-End Multi-Tenant Vertical Slice First:**
* After closing Phase A's RLS, write-path sanitization, and real kill-and-restart tests, prove **one complete vertical slice** for a single tenant: *prompt → Builder Sprite wakes → OpenCode runs & asks 1 question → user answers & session resumes → Studio verifies artifact digest + diff/tests → user approves → Runtime Sprite deploys it*. Then scale out multi-tenant ergonomics and polish.


## deepseek

# Adversarial Review of `writeup2.md` — Revised

**What changed from the prior review.** I previously critiqued the architecture as overbuilt for a single-user product. That critique rested on an assumption I made rather than one the writeup established. The writeup says "Miniapp Studio" and "business owner" in the singular, but it never says "one user per Studio." I read that as single-user; it is not. With the clarification that **Miniapp Studio is multi-tenant SaaS** — many business owners, each with their own Builder Sprites, one central control plane serving all of them — **two of my six findings are withdrawn and two new ones are added**. The remaining critiques are sharper because the architecture now has to justify itself against a real multi-tenant threat model, not a hypothetical one.

**What I can and cannot verify.** I have `writeup2.md` and the four source docs only as described in the writeup. Where I critique, I critique the writeup's articulation. If the source docs justify a decision the writeup does not, the finding is that the writeup fails as public articulation of the architecture.

---

## 1. Verdict

**The architecture is appropriately dimensioned for a multi-tenant SaaS control plane.** The control-plane / execution-plane split, tenant-scoped RLS, per-build isolation, and separation of Builder from Runtime are not over-engineering — they are the correct shape for a system that serves multiple tenants with different repositories, different connectors, different production applications, and different security postures.

**The engineering sequence is still wrong** and still costs time and certainty. Phase A was built against fakes; three closure findings exist because the fakes did not exercise the paths that the real Gateway would have. The correct sequence is *vertical slice first, then close what the slice reveals*. That observation survives the multi-tenant clarification.

**Two specific simplifications still apply** — the file-drop handoff priority and the event vocabulary — and they are not multi-tenant consequences.

**Two new findings emerge that the writeup should address** because they are multi-tenant-specific: tenant lifecycle management of the Builder fleet, and cost/metering. Neither is a blocker, but both are the kind of gap that becomes expensive after the first month of operation.

**The HMAC protocol is defensible in multi-tenant context**, but the writeup does not say why. That is the fixable part.

---

## 2. What the writeup gets right that I did not credit

**The control plane / execution plane split is correct for multi-tenant SaaS.** Studio is the only component that can see the tenant database, the tenant-to-Sprite assignment, the billing, and the approval history across all builds and all Sprites. If the Builder held the authoritative conversation record, a Builder compromise would destroy a tenant's business history. The writeup's central principle — *"Studio owns the work, its history, its permissions, and its outcome"* — is exactly the right invariant, and it depends on the physical separation.

**Tenant-scoped RLS on ten new Phase A tables is not overhead; it is the primary isolation boundary.** Phase A's most serious finding — `FORCE ROW LEVEL SECURITY` without `ENABLE ROW LEVEL SECURITY` — is catastrophic specifically because these tables hold tenant data across multiple business owners. In a single-user product, this would have been a curious config mistake. In multi-tenant Studio, it is the exact class of bug that leaks one customer's data to another. The writeup treats it as a major security finding and a mandatory closure item; that is the correct response.

**Separate Unix users for `studio-gw` and `studio-agent`** is correct for multi-tenant because each Builder Sprite is a tenant-scoped execution environment that must not be able to compromise other tenants' Builders through shared credentials or cross-Sprite lateral movement. The writeup's insistence that the boundary be OS-enforced rather than prompt-enforced is exactly right.

**The Runtime Sprite as a distinct environment from the Builder** is correct for multi-tenant, because the Runtime runs a tenant's customer-facing production application. Agent code, dev tools, and the Builder's checkout have no business being on the same Sprite as the tenant's customer data and production connector bindings. My prior review downgraded this; I was wrong. The three-environment model is right.

**The Phase A closure plan is well-sequenced** and the three mandatory corrections are the right three. That they were needed is the finding (F4 below); that they are being closed properly is credit.

The following critiques apply to what the writeup **adds beyond** these correctly-justified components, not to the components themselves.

---

## 3. Withdrawn findings

**F1 (was: collapse to single-Sprite architecture) — WITHDRAWN.** This critique assumed a single-user product. For multi-tenant SaaS, collapsing Studio and the Builder would destroy the tenant boundary, eliminate central audit, prevent cross-tenant billing, and make a Builder compromise destroy the tenant's business history. The split is correct. I withdraw the finding entirely.

**F6 (was: three-environment model over-dimensioned) — WITHDRAWN.** I argued that for a personal agent, the Runtime Sprite could be folded into the Builder. That argument does not hold for multi-tenant: the Runtime serves a tenant's production application, which must be isolated from the Builder's agent code and dev tools. The three-environment model is the correct dimension. I withdraw the finding.

---

## 4. Remaining findings

### F2 — File-drop handoff should be primary, not fallback

*Unchanged from prior review. This finding is about the OpenCode adapter, not the architecture.*

The writeup specifies native question-event capture as primary and file-drop as fallback, while itself conceding that *"this behavior cannot be assumed from the existence of the question tool."* If the behavior cannot be assumed, it should not be primary. File-drop is strictly more robust: it requires no reliable signal from a non-interactive CLI about when a question is pending, does not require the agent process to survive across turns, and is testable in isolation.

The writeup contains the correct decision and labels it "fallback." Flip the priority. Make file-drop primary; native event capture is an optimization attempted only after the robust path works.

### F3 — HMAC protocol needs justification in the writeup

*Softened from prior review. In multi-tenant context, HMAC is defensible. The writeup just does not say why.*

My prior critique was that HMAC is overkill for a channel between trusted components. That is not quite right in multi-tenant context. There is a real defense-in-depth argument: if a Builder Sprite is compromised, per-build HMAC keys limit the blast radius of forged callbacks — the compromised Builder can only forge messages for its own build, not for other tenants' builds.

However, this argument is not what the writeup says. The writeup says HMAC provides "authenticated commands," "timestamp freshness checks," "message deduplication," "tenant/build binding," "replay resistance," and "strict rejection of forged or tampered messages." Four of those six (authentication, freshness, dedup, replay) are provided by HTTPS + dedup by `message_id` alone, without HMAC. Only "tenant/build binding" and "forged/tampered rejection" specifically need HMAC, and the writeup does not explain that these are what HMAC is *for*.

There are two acceptable fixes:

1. **Justify HMAC specifically.** Add a sentence: "HMAC per-build keys exist so that a compromised Builder cannot forge callbacks for other tenants' builds, even if the attacker obtains a valid Studio session token." That makes the choice defensible.

2. **Simplify to a per-build bearer token.** If the threat model does not require per-build forgery resistance (which it might not, given that Studio can verify the callback's build_id against the calling Sprite's identity anyway), a per-build bearer token in the `studio-gw` environment achieves the same authentication without the signing ceremony.

I am not recommending option 2 over option 1 without knowing the threat model. I am recommending that the writeup **state which threat the HMAC defends against**, and simplify if that threat does not require HMAC specifically.

### F4 — Phase B should come before Phase A closure

*Unchanged from prior review.*

The writeup's priorities section says: *"First: close Phase A's security and recovery findings. Second: prove the Gateway and OpenCode work together inside a real Builder Sprite."* That order is backwards.

Phase A's three closure findings exist because Phase A was built against fakes:
- Secret scrubbing incomplete on question bodies — could not have been found without real question content.
- RLS `FORCE` without `ENABLE` — could not have been found without real multi-tenant traffic through the affected tables.
- Restart test used two servers instead of one — could not have been found without the discipline of testing a real restart.

**The correct sequence is: prove the smallest end-to-end works against reality, then close the findings reality reveals.** The current sequence closes findings against the fakes, then connects to reality, then discovers the findings that mattered are different.

The multi-tenant context makes this more urgent, not less: Phase A closures involve security boundaries that touch tenant isolation. You want to discover what those boundaries actually are before you harden the wrong ones. Phase A closures remain necessary; they just come after Phase B reveals which closures are real.

### F5 — The event vocabulary can be simplified without losing information

*Unchanged from prior review.*

Eleven event types (`agent.started`, `agent.progress`, `agent.message`, `agent.question`, `agent.proposal`, `agent.authorization_request`, `agent.artifact_ready`, `agent.completed`, `agent.failed`, `agent.resumed`, `agent.cancelled_ack`) cover three underlying states:

- **The agent is still working and here is some output** — covers `started`, `progress`, `message`, `resumed`, `artifact_ready`.
- **The agent needs input from the user** — covers `question`, `proposal`, `authorization_request`.
- **The agent has terminated** — covers `completed`, `failed`, `cancelled_ack`.

The distinction between clarification, proposal, and authorization is real and important. But it belongs as a **field** (`kind: clarification | proposal | authorization`) on the input-needed event, not as three separate event types. The distinction is preserved; the vocabulary is smaller; adding a fourth kind of input is a field change, not a schema change.

Three event types instead of eleven. One handler in Studio with three branches instead of eleven handlers. Simpler to test, simpler to version, simpler to reason about under concurrency.

The writeup's vocabulary is thorough. Thorough is not the same as correct.

---

## 5. New findings from multi-tenant context

### F7 — Tenant lifecycle management of the Builder fleet is not addressed

The writeup covers how Studio talks to an assigned Builder Sprite, how the Fly MCP grant is scoped to that Sprite, and how credentials never leave the gateway. It does not cover:

- **Builder provisioning.** When does a tenant's first Builder Sprite get created? When the tenant signs up? When they describe their first application? What does the tenant see during provisioning?
- **Builder assignment.** If a tenant has multiple applications, each with its own build history, do they share a Builder or each get their own? How does Studio route a build to the correct Builder?
- **Builder decommissioning.** When a tenant deletes an application, is the Builder Sprite destroyed? What happens to the checkout, the session files, the artifacts that have not yet been pulled to Studio?
- **Builder compromise handling.** If a Builder is compromised (via the agent, via a malicious repository, via the Sprite's own surface), how does Studio detect it, isolate it, and prevent lateral movement to the tenant's Runtime Sprites? The writeup covers the OS-level `studio-gw` vs. `studio-agent` split but not the fleet-level incident response.
- **Tenant deprovisioning.** When a tenant leaves Studio, what cascades to their Builders and Runtimes? The writeup's tenant-scoped RLS is good; a tenant deprovisioning that leaves orphaned Builders is a GDPR and billing problem.
- **Cross-tenant audit.** Studio can see all Builders for all tenants. What does the audit log show for "which tenant's Builder was active at time T?" and "which of the two tenants who share... wait, they don't share Builders, right?" Actually if they don't share, the tenancy model is per-application or per-workspace. That needs to be stated.

None of these are blockers. All of them affect the tenant data model, the audit log schema, and the Fleet observability requirements. The writeup should at least state the tenancy model: **per-tenant, per-application, or per-workspace Builders?**

### F8 — Cost and metering for the fleet is not addressed

Multi-tenant SaaS has a cost model. Studio serves N tenants, each with M Builders, each Builder consuming Sprite-hours, storage, connector calls, and (eventually) Runtime Sprite hours. The writeup does not mention:

- **Sprite-hour accounting.** Who stops idle Builders? Studio, or the tenant? If Studio, on what schedule?
- **Per-tenant quota.** How many Builders can a tenant have? How many concurrent builds? What happens when a tenant's Sprite-hour budget is exhausted?
- **Cross-tenant fairness.** If one tenant's Builder is in a tight loop generating callbacks, does that tenant's load affect the fleet's ability to serve other tenants?
- **Billing surface.** Studio can see all tenant activity; the billing dashboard should be a Studio surface, not a Fly dashboard.

The writeup's "what success looks like" section describes a smooth user experience. That experience has a cost, and the cost has to be metered and billed. The writeup does not say how.

---

## 6. What I would do differently

The prior review offered three options; two are now withdrawn. The remaining two:

### Option B — Keep the multi-tenant split, drop the ceremony

This remains valid. If Studio must remain multi-tenant (which it must), keep the control-plane/execution-plane split but remove ceremony that exists because the split was not designed minimally:

- **HMAC → justify or simplify.** Per F3, either state the threat HMAC defends against or replace with a per-build bearer token.
- **Callback outbox → SSE or poll-for-events.** If the Builder-to-Studio channel can hold a long-lived SSE stream, that is the entire state sync. If not, a "poll for events since cursor" endpoint is simpler than durable outbox semantics.
- **11 events → 3 with subtype fields.** Per F5.
- **Session resume → file-drop primary.** Per F2. Each turn is a fresh process that reads the answer file and continues. Accept 10–30s cold-start per turn in exchange for eliminating the entire session-resume failure class.

Estimated implementation reduction: **20–30%.** Estimated time to end-to-end working: **~3 weeks instead of 5 phases.**

### Option C — Keep the architecture and the ceremony, simplify the sequencing

If the ceremony is justified by the multi-tenant threat model (which the writeup has not yet argued, but which is arguable), apply the following sequencing changes:

- **F2** — Flip primary/fallback: file-drop becomes primary; question-event capture becomes optimization.
- **F4** — Reorder: Phase B (Gateway-in-Sprite with a no-op agent) **before** Phase A closure. You will find different findings, and they will be the findings that matter for the multi-tenant threat model.
- **F5** — Reduce event types to 3 with subtype fields, even if you keep HMAC and the outbox.
- **F7** — State the tenancy model explicitly; add tenant lifecycle policy.
- **F8** — Add a fleet cost section.

Estimated implementation reduction: **10–15%.** Estimated time to end-to-end working: **unchanged, but with better targeting of the closures.**

---

## 7. Concrete recommendations

If I could change three things about the writeup before any of the above:

1. **State the tenancy model explicitly in §1.** "Each tenant has N Builder Sprites, one per application workspace" or "one per tenant" — pick and state. Every subsequent decision about Builder provisioning, RLS scope, the Fly MCP grant, and the audit log depends on this answer. The writeup currently assumes it without saying it.

2. **Reorder the phases so Phase B (Gateway-in-Sprite with a no-op agent) comes before Phase A closure.** Build a no-op Gateway inside a real Sprite, prove a `status` round trip, put it to sleep, wake it, prove `agent.resumed` fires. Do this *today*, before closing Phase A's security findings. The closures are necessary, but they should be closed against the finding set that reality reveals, not the finding set that fakes produced.

3. **Justify the HMAC, or replace it.** One sentence — "HMAC per-build keys limit forged-callback blast radius if a Builder is compromised" — makes the choice defensible. If the threat model does not require that specific protection, a per-build bearer token is simpler and achieves the same authentication.

**The writeup's own §12 priority list should become, in this order:**

1. State the tenancy model.
2. **Prove a no-op agent runs in a real Sprite**, asks a deterministic question, consumes a deterministic answer from a file, and reports success.
3. **Prove the same path survives Sprite sleep/wake and a full Studio restart.**
4. **Close only the Phase A findings that the real Gateway exposes.**
5. **Then harden the protocol** — justify HMAC or simplify; state the outbox's role or replace it with SSE.
6. **Then Phase C, D, E** in the current sequence.

The architecture is right. The sequencing is not. The remaining simplification opportunities are real but second-order.

---

## 8. What the multi-tenant clarification changes about the finding set

| Prior review | Revised |
|---|---|
| F1: split not justified | **Withdrawn.** Split is correct for multi-tenant; the writeup's central principle depends on it. |
| F2: file-drop should be primary | **Unchanged.** About the adapter, not the architecture. |
| F3: HMAC overengineered | **Softened.** Defensible in multi-tenant context; the writeup must justify it or simplify. |
| F4: Phase A sequencing | **Unchanged.** Strengthened by multi-tenant context: you want to discover the real security findings before closing the wrong ones. |
| F5: event vocabulary | **Unchanged.** About schema design, not architecture. |
| F6: three-environment over-built | **Withdrawn.** Runtime Sprite is correct for multi-tenant production isolation. |
| — | **New F7:** tenant lifecycle management of the Builder fleet. |
| — | **New F8:** cost and metering across the fleet. |

The writeup is now judged against a product (multi-tenant SaaS) where most of its architectural choices are correct. The remaining critiques are about **sequencing** (F4), **schema design** (F5), **articulation** (F3), **adapter discipline** (F2), and **gaps the writeup does not cover** (F7, F8). None of them requires changing the architecture.

**The single sentence that would improve the writeup most:** *"Miniapp Studio serves multiple business owners from a single control plane; each tenant has one or more Builder Sprites assigned per application workspace, isolated at the OS layer and at the database layer, and the control plane holds the only authoritative record of every tenant's conversation, build history, and approval chain."*

That sentence makes the rest of the writeup read as the correct architecture for the correct product. Without it, the writeup reads as an ambitious architecture whose product justification is left to the reader.

The single sentence that would improve the writeup *second-most*: *"Phase B (Gateway in a real Sprite) precedes Phase A closure because we want to close the findings the real system reveals, not the findings the fakes produce."*

Together, those two sentences would reframe the entire document as disciplined multi-tenant engineering rather than ambitious multi-tenant engineering. The distance between those two is where projects succeed or fail.