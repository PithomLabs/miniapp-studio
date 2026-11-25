

P3 ACCEPTED

P4 APPROVED TO START



Excellent progress. **P3 is accepted based on the reported implementation and test evidence. Proceed directly to P4.** No additional architecture-review cycle is needed.

The two major infrastructure decisions are now settled:

- **Deployment:** Sprite per generated application. The real provisioning, private-by-default access, binary deployment, service startup, URL access, and cleanup path have been demonstrated.
- **Database:** Fly Managed Postgres Developer on PostgreSQL 17, with Neon retained as the fallback. Production use remains conditional on the deployment-time smoke test.

The P3 adapters and lifecycle code also give P4 real interfaces to build against.

## **P4 — Studio server and user experience**

**Objective**

Turn the completed execution engine and infrastructure adapters into a usable application for business owners.

**1. Server foundation**

- Go HTTP server and route structure.
- Secure workspace sessions.
- Authentication and server-derived tenant context.
- PostgreSQL persistence and migrations.
- Application registry and ownership enforcement.
- Structured errors, audit events, and redacted logging.

**2. Six product screens**

**Home**

Apps, activity, status

**Create App**

Builder and live preview

**Connectors**

Catalog and Custom API

**Validate & Deploy**

Validation and deployment flow

**My Apps**

App and URL lifecycle

**Activity**

Runs, audit, diagnostics

**3. Acceptance**

- Every consequential operation goes through `runtime.Service`.
- Server-derived tenant identity on every request.
- Cross-tenant reads and writes denied and audited.
- Real connector configuration and permission views.
- No credentials exposed to browser or Sprite.
- All six screens have loading, empty, error, and success states.
- Actions are enabled only when their backend routes and authorization are implemented.

## **Five rules I want preserved during P4**

1. **Multi-tenancy is foundational.** Workspace-to-tenant mapping must be enforced in persistence, not just session middleware.
2. **MPG is the selected target, not yet a production-proven deployment.** The owner provisioning step and deploy-time smoke test remain explicit.
3. **The Sprite is not the trusted control plane.** All privileged operations remain in the Studio backend.
4. **External writes remain disabled.** G-01 is still unresolved; do not work around it.
5. **Do not start P5 prematurely.** P4 delivers the server and user experience. The builder agent and complete Invoice Review Desk journey belong to P5.

The P3/P4/P5 sequencing and one-app cut line remain unchanged.&#x20;

miniapp-studio-implementation-plan-consolidated(2).md



**My recommendation: begin with the P4 server foundation, authentication/session contract, and tenant-scoped persistence. Build the six screens on those verified interfaces rather than creating UI-first mock implementations.**

Keep everything uncommitted, as before.