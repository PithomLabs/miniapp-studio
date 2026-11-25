Phase A accepted

**The bounded closure satisfies the outstanding Phase A acceptance criteria. Phase A is formally closed. Phase B is now authorized to begin under the existing Agent Gateway implementation plan.**

I reviewed the closure evidence against the three mandatory findings and M1–M4.

### **Acceptance decision**

All three mandatory items have demonstrated corrections:

| Requirement                   | Evidence                                                                                   | Decision |
| ----------------------------- | ------------------------------------------------------------------------------------------ | -------- |
| RLS enforcement               | Migration 000016; red at v15, green at v16; catalog and application-role behavioral tests  | Accepted |
| Agent-content secret handling | Write-path sanitization; database, tool, page and SSE canary tests; red-before/green-after | Accepted |
| Genuine restart recovery      | Ordered shutdown, observable completion, fresh boot, answer application and replay         | Accepted |

M1–M4 are also completed. The deadlock finding is particularly important: the pre-fix reproduction exposed not only inconsistent lock ordering, but also an error-classification defect that concealed database deadlocks as not-found responses. Correcting both addresses the underlying problem rather than merely changing lock order.

The full gates are reported green on PostgreSQL 19.0 and 17.12, including race tests and migration round-trip through v16. The evidence also correctly records the remaining `app.invocations` residual and carries M5–M8 forward.&#x20;

gateway-phase-a-evidence(1).md



I am accepting the reported execution evidence; I am not claiming to have independently rerun the repository tests.

### **Phase B — authorized starting direction**

Phase B has a different objective from Phase A:

**Turn the tested Studio-side protocol into a functioning, supervised coding-agent execution environment inside a real Builder Sprite.**

Follow the existing implementation sequence:

1. **Gateway transport:** implement the Studio gateway client and Builder Gateway command handler against fakes.
2. **Durability:** implement command deduplication, persistent callback outbox, acknowledgments and retry behavior.
3. **Agent supervision:** implement the process supervisor, session manager and OpenCode adapter.
4. **Security and provisioning:** implement separate gateway/agent Unix identities, HMAC-key isolation, pinned OpenCode/plugin configuration, and restricted Fly MCP grants.
5. **Live installation:** install the gateway as a persistent Sprite service and prove authenticated status communication.
6. **Recovery:** prove gateway restart and Sprite sleep/wake behavior, including honest `agent.resumed` or `agent.failed` reporting.

The Phase B acceptance target remains:

- Real Builder Sprite successfully provisioned.
- Gateway service installed and verified.
- Signed `status` round trip succeeds.
- A supervised OpenCode no-op turn executes.
- Sprite sleeps and subsequently wakes.
- The agent session is resumed or failure is honestly reported.
- Studio receives the appropriate authenticated event.
- Execution timings and resource cleanup are recorded.

These are the existing Phase B requirements, not a relaxed substitute for the later Phase C interaction proof.&#x20;

agent-gateway-implementation-plan.md



### **Boundaries remain unchanged**

Three important points carry forward:

- **M5–M8:** stream lifetime, invocation-data sensitivity, artifact-manifest scrubbing and evidence discipline remain obligations of their assigned phases.
- **F1, G-01 and P5:** no changes or workarounds. Consequential business writes remain disabled while G-01 is unresolved.
- **Live environment:** begin with fake-backed implementation and local verification. Obtain explicit authorization before live Sprite provisioning, OAuth actions or other external infrastructure changes. No commits or pushes without authorization.

### **Next architectural milestone**

The next meaningful milestone is not simply getting OpenCode to launch. It is proving that a Gateway can supervise it without sharing secrets or depending on a permanently running process.

**Phase B must finish with a real, recoverable Builder Gateway. Phase C must then prove the complete human question → answer → same-session OpenCode resume lifecycle.**

No additional general Phase A review is needed. We move forward.