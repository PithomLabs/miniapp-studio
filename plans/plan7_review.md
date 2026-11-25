**Start implementation now. I approve Revision 2 as the governing plan. There is no reason for another general architecture-review cycle.**

The two transaction/recovery blockers have been converted into explicit execution rules, a four-point crash table, and a framework correction with a clear ownership boundary. Most importantly, the plan no longer pretends that a provider operation and a Class A transaction commit atomically. G-01 is identified rather than hidden, and writes are prohibited until it is fixed.&#x20;

miniapp-studio-implementation-plan-consolidated(2).md

miniapp-studio-implementation-plan-consolidated(2).md



### **Start in this order**

1\. **P1 — implement MiniappSpec and the deterministic validator now.** Build the schema, version/hash policy, error taxonomy, fixtures, and deliberate-rejection test. Keep this work independent of the provisional Miniapp dependency.

2\. **Finish the P0 framework gate in parallel.** Run the required PG19 test suite, restart tests, race subset, and Studio import/approval-resumption spike. Until these pass, do not build framework-dependent features on `73351a8`. This is an existing approval condition, not a new review requirement.&#x20;

miniapp-studio-implementation-plan-consolidated(2).md



3\. **P2 — build the runner and renderer once P0 passes.** Then proceed through the Fly adapters, UI, and golden path in the existing dependency order.&#x20;

miniapp-studio-implementation-plan-consolidated(2).md



4\. **Keep external writes disabled until G-01 is implemented and verified.** The acceptance test must prove that a new invocation can safely take over an abandoned `PENDING` operation, that concurrent takeovers have one winner, and that authorization, tenant, payload, and attempt fencing remain intact. G6 already makes this a release condition.&#x20;

miniapp-studio-implementation-plan-consolidated(2).md

miniapp-studio-implementation-plan-consolidated(2).md



### **Two implementation safeguards**

There is one important architecture invariant I will not let the team lose: **multi-tenant readiness must be real from the first persistent data model, not just a workspace-to-tenant mapping.** §8.1 establishes the basic mapping and cross-tenant denial, but implementation must also scope specs, versions, executions, approvals, effects, audit records, connector bindings, and deployment resources to their owning tenant. Tenant identity must be derived server-side, and cross-tenant access must be tested across reads, writes, approvals, connectors, and URLs. This does not stop P1 schema validation; it must be enforced before committing tenant-bearing persistence and resource ownership designs.&#x20;

miniapp-studio-implementation-plan-consolidated(2).md



Also, make two small accuracy corrections as implementation documentation is touched: call §4.5 an independently committed effect-ledger lifecycle *inside a still-open Class A handler*, not one outside the Class A transaction; and change the provider-neutrality test to verify that changing provider selection does not mutate an existing spec or its hash, rather than requiring different LLMs to generate byte-identical specs. Neither requires reopening the architecture.

**My decision:** no more plan revisions or review rounds before coding. Begin P1 and finish the P0 gate. Treat G-01 and the four crash-point tests as implementation gates for external writes, not reasons to delay the read-only golden path.