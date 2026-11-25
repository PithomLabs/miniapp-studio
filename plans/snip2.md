P4 APPROVED — GO

The approach is sound and follows the approved roadmap. **Start P4 with the server foundation, authentication, tenant-scoped persistence, and static tool registration.** No additional architecture review is required.

I have three implementation conditions to emphasize.

### **1. Resolve the session-to-RLS bootstrap boundary**

This is the most important detail in the design.

A request must identify its tenant from a validated server-side session, but the session itself is stored in a tenant-scoped PostgreSQL table. You cannot apply the tenant's RLS context before discovering which tenant owns the session.

Design an explicit, narrowly privileged session-lookup mechanism:

1. Receive the opaque session token.
2. Hash and validate it against the session store through a restricted lookup path.
3. Obtain the server-controlled tenant and credential-binding identity.
4. Establish transaction-local tenant context.
5. Perform all normal application operations under RLS and explicit tenant predicates.

Do not solve this with a general-purpose `BYPASSRLS` application connection or a tenant ID supplied by the browser.

Add regression tests for this bootstrap boundary.

### **2. Keep external mutations disabled**

P4 may test deployment and connector operations using injected fakes, but must not perform live external mutations.

This includes Sprite provisioning, deployment, connector creation/update, and URL changes. The Custom API wizard can exercise its real client interface through test transports, while unconfigured production paths must say so honestly.

G-01 remains unresolved. No workaround.

### **3. Make the runtime the actual enforcement boundary**

HTTP handlers should authenticate, derive the request context, and invoke tools through `runtime.Service`. They must not directly call tool handlers or bypass framework authorization and audit.

The persistence layer should independently enforce tenant ownership through RLS.

## **Approved P4 execution order**

| Step | Deliverable                                | Acceptance                                                       |
| ---- | ------------------------------------------ | ---------------------------------------------------------------- |
| 1    | `cmd/studio`, configuration, server wiring | Fail-closed startup, health/readiness                            |
| 2    | Sessions and authentication                | Opaque sessions, expiry, revocation                              |
| 3    | Embedded PostgreSQL migrations             | Tenant-scoped tables, RLS, indexes                               |
| 4    | Studio tools and registry                  | `spec.validate`, registry, deploy, connectors through `Invoke`   |
| 5    | Six screens                                | Real handlers, honest states, no fabricated data                 |
| 6    | Security and acceptance                    | PostgreSQL 19, tenant matrix, session attacks, secret-leak tests |

This preserves the approved six-screen scope and keeps P5 out of the implementation.&#x20;

miniapp-studio-implementation-plan-consolidated(2).md



**Final instruction: GO.** Implement the foundation first, then build the six screens over verified backend interfaces. Keep Miniapp untouched unless §14 evidence genuinely requires a framework correction. Keep all changes uncommitted.