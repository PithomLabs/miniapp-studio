# Miniapp Studio — Consolidated Product Direction and Hacksgiving Scope

**Revision:** 2.0  
**Date:** 24 November 2026  
**Status:** Consolidated product/architecture direction for implementation planning; not an implementation specification  
**Purpose:** Reconcile `plan_deepseek(4).md`, `plan_gemini(2).md`, `plan_z(3).md`, the earlier `miniapp-studio-consolidated-plan.md`, and the product decisions made in this conversation.

---

## 1. Executive decision

Build **Miniapp Studio**, an AI-native application factory for non-technical business users. A user describes a business need in plain language. Studio converts that intent into a versioned declarative Miniapp specification assembled from reusable data, UI, integration, workflow, and governance primitives. Studio validates the specification, shows the required data sources and connector permissions, previews the generated application, and runs a bounded demo path through a fixed runtime.

**Do not build twelve separate business apps.** The twelve earlier ideas become examples, templates, test fixtures, and demonstrations of one generic platform.

The pre-existing Go Miniapp framework is the foundation, not the Hacksgiving product surface. The judges should see Miniapp Studio: its user experience, generated app specification, connector ecosystem browser, custom API connector path, deterministic validation, and one working application whose builder/personal agent runs on a Fly.io Sprite and uses at least one real Sprites connector.

### Product promise

> **Describe the business application you need. Miniapp Studio composes it from governed, reusable primitives, connects it to the systems you authorize, and shows you what it will do before it runs.**

### Longer-term differentiator

Miniapp's strategic differentiation is **durable business-workflow execution**. Generated applications should eventually support long-lived workflows, human approvals that can wait days, durable timers and events, interruption recovery, idempotent business effects, reconciliation of uncertain outcomes, compensation where possible, version-safe upgrades, and an auditable execution history.

This is a **roadmap differentiator, not a Hacksgiving demo claim**. Do not imply DBOS durable waits, workflow replay, exactly-once external effects, or production-grade compensation are already delivered unless repository evidence and exact acceptance tests prove each behavior. The Hacksgiving demo should focus on application generation and a real Sprite/connector path.

---

## 2. Reconciled decisions: what is adopted, refined, or deferred

| Topic | Consolidated decision | Rationale / boundary |
|---|---|---|
| Platform, not twelve apps | **Adopt** one Studio, one declarative app model, one generic renderer/runner, many definitions/templates | This is the actual product thesis and gives judges an observable platform rather than a hand-built vertical agent. |
| Generate specs, not arbitrary programs | **Adopt** a typed declarative Miniapp specification interpreted by trusted, fixed code | Keeps generated artifacts inspectable, validatable, versionable, and constrained. Do not execute arbitrary LLM-generated Go, JavaScript, shell, or binaries in the trusted runtime. |
| Generic runner | **Adopt**, subject to repository inspection: one statically registered generic runner/interpreter handles runtime-generated app definitions as data | The existing framework statically registers compiled Miniapps. A generic interpreter is the bridge to apps created at runtime. Its implementation is new Studio/composition work unless proven otherwise. |
| Three-layer architecture | **Adopt** Studio → App Compiler/Spec Runner → Miniapp framework, with Fly Sprite/connector services as runtime infrastructure | Separates user experience, safe composition, and trusted execution. |
| End-user primitives | **Adopt** six families: data/state, UI, workflow/control, integrations/connectors, governance/safety, and evidence/presentation | These are reusable platform building blocks, not separate products. |
| Natural language + structured editor | **Adopt** conversational describe/edit plus a readable structured specification/plan view and live preview | No full drag-and-drop workflow canvas for Hacksgiving. The structured plan view is sufficient as the initial editor. |
| Connector catalog | **Adopt and elevate** to a first-class Studio area | User explicitly requires the Studio to display the supported Sprites connector ecosystem and allow extension through Custom API connectors. |
| Custom API | **Adopt as a first-class extension point** | It makes the platform extensible beyond Fly's built-in connector list. Use documented Fly APIs and security rules; do not create an unrestricted URL proxy. |
| Connector discovery | **Refine** into separate public catalog, organization connections, and Sprite-authorized runtime discovery | These are different data sources and must not be collapsed into one “connected” indicator. |
| Fly integrations and SDKs | **Adopt as distinct ecosystem sections**, not connector records | Plugins, MCP clients, platform integrations, and SDKs help developers connect to Sprites; they are not external business API connectors. |
| Sprite model | **Adopt one authorized personal Sprite for the Hacksgiving golden path**, unless an early spike proves a better minimal topology | Avoid the cost and complexity of one Sprite per generated sample app. Keep long-term placement policy independent from app identity. |
| Checkpoint / rewind | **Refine**: filesystem checkpoint/restore is a Sprite capability; it does not undo external effects or rewind DBOS/business workflow state | Show only if tested. Clearly label what a restore does and does not restore. “Always undo” is not an acceptable claim. |
| Approvals | **Refine**: a simple explicit confirmation may be included for one demo action if the pinned path is tested. Durable/out-of-band approval parking is roadmap unless verified | A UI confirmation is not equivalent to a durable approval workflow or separation-of-duties guarantee. |
| Receipt/effect ledger | **Retain as a framework/product principle**, but show it in Hacksgiving only if the exact path is implemented and tested | Do not claim every action is receipted or that duplicate external effects are impossible without evidence. |
| DBOS durable workflow | **Defer from Hacksgiving demo; retain as flagship roadmap** | The demo must not become a DBOS technical showcase at the expense of the Studio. Preserve future compatibility in the spec model and architecture. |
| Three generated agents in demo | **Defer**; one complete generated application is mandatory, a second short variation is optional | One working end-to-end example beats three incomplete applications. Gallery templates can demonstrate breadth without separate implementations. |
| All five hackathon partners in one flow | **Do not require** | Integrate partner tools where they materially help and are enabled in the user's organization. Do not bolt on TypeSafeAI, Box, Pinecone, Ollama, or other partner services merely to tick boxes. In particular, Fly's current ecosystem description presents TypeSafeAI Jev as a decision model (yes/no, choice, score with probability/confidence), not as the MiniappSpec schema validator; schema validation must remain deterministic application code. |
| Rule compliance | **Adopt** transparent provenance: pin and disclose the pre-existing framework; make Studio/generator/renderer/connector work clearly new Hacksgiving work; ask organizers about dependency eligibility | Do not imply pre-existing code was written during Hacksgiving. |

---

## 3. Target user and product boundaries

### Initial target user

Business owners, operators, and small teams who use multiple SaaS tools but cannot afford engineering teams or custom software for every operational need. They should be able to generate practical internal applications without writing Go, SQL, or frontend code.

The initial experience should be standalone and coherent. Embedded/white-label Studio for SaaS vendors is a future distribution channel, not a Hacksgiving requirement.

### Examples are outputs, not products

Examples include customer/sales workspaces, invoice review, onboarding checklists, task management, inventory views, issue triage, and operational dashboards. Each must be generated from the same primitives, schema, renderer, and runtime. These are gallery templates and demo fixtures—not twelve individually coded applications.

### Responsibility boundaries

| Miniapp Studio / Miniapp owns | External systems own |
|---|---|
| App definitions, versions, UI composition, capability declarations, policies, run metadata | Canonical business records and provider-native behavior |
| Logical business capabilities and provider bindings | Provider APIs, authentication grants, quotas, and service availability |
| Validation, preview, execution coordination, and Miniapp-side audit | External side effects and their actual outcome |
| Approval policy and user-facing evidence where implemented | Provider-specific authorization and final acceptance of requests |

Miniapp should remain CRM-agnostic and provider-agnostic at the business-logic layer. A logical operation such as `customer.search` should not be hardcoded to HubSpot, Salesforce, GoHighLevel, or any single provider. Provider adapters and connector metadata map logical capabilities to actual API operations.

---

## 4. Architecture

```text
Business user (browser / phone)
          |
          v
+---------------------------------------------------------------+
| LAYER 1 — MINIAPP STUDIO                                      |
| Describe / edit · template gallery · spec inspector · preview |
| connector ecosystem · custom API wizard · run view            |
+------------------------------+--------------------------------+
                               |
                               v
+---------------------------------------------------------------+
| LAYER 2 — APP COMPOSITION                                     |
| Sprite-hosted builder agent → declarative MiniappSpec         |
| schema validation → capability and permission resolution     |
| fixture/dry-run validation → generic renderer / spec runner   |
+------------------------------+--------------------------------+
                               |
                               v
+---------------------------------------------------------------+
| LAYER 3 — MINIAPP FRAMEWORK (pre-existing foundation)         |
| contracts · registry · invocation · policy · effects          |
| auth · audit · adapters · execution interfaces                 |
| durable DBOS/River capabilities: roadmap / gated by evidence   |
+------------------------------+--------------------------------+
                               |
                 +-------------+----------------+
                 |                              |
                 v                              v
       FLY SPRITE (Hacksgiving)          TRUSTED CONTROL PLANE
       personal builder agent            Studio backend / API
       generated-app preview/runner      authorized Fly management
       no raw provider secrets            connector provisioning
                 |                              |
                 +---------------+--------------+
                                 |
                                 v
                       SPRITES CONNECTOR GATEWAY
                       configured provider connections
                       per-Sprite access policies
                       Custom API connections
                                 |
                                 v
                     External SaaS / REST / data / models
```

### Placement decision

For the Hacksgiving build, start with one authorized personal Sprite that demonstrably runs the builder/personal agent and makes at least one real connector call. Generated app preview/runner may share that Sprite for the prototype if the repository and integration spike show that this is the smallest reliable path.

Keep privileged control-plane operations—especially organization-level connector creation and Fly management credentials—on the trusted Studio backend. Do not put a broad Fly organization token into the Sprite-hosted agent. Do not require one Sprite per gallery template or one Sprite per generated demo app.

For the longer-term product, compute placement should follow trust, isolation, workload, lifecycle, and cost requirements. Trusted production execution may remain on Fly Machines; agent-authored or bounded untrusted tasks may use Sprites; durable workflows may orchestrate isolated Sprite tasks. Execution semantics and compute placement are independent dimensions.

---

## 5. Declarative Miniapp specification: one canonical model

Do not create competing “AppSpec”, “MiniappSpec”, and “WorkflowPlan” formats that drift. The implementation plan should recommend one canonical, versioned **MiniappSpec**. A workflow definition is one section of that app specification, not a second unrelated language.

Suggested top-level sections (not a frozen schema):

| Section | Responsibility |
|---|---|
| `app` | Stable ID, name, description, schema version, app version, owner/workspace reference |
| `data` | Logical entities, fields, relationships, query definitions, sources, filters, aggregates, bindings |
| `ui` | Pages, navigation, components, properties, data bindings, user actions |
| `workflows` | Trigger metadata and a constrained sequence/graph of typed steps |
| `capabilities` | Logical operations required by the app and provider/connection bindings |
| `policies` | Roles, allowed operations, approval classification, rate/action limits, effect classes |
| `fixtures` | Test inputs, sample records, expected validation and preview outcomes |
| `presentation` | Basic layout/theme tokens and labels; advanced branding is roadmap |
| `metadata` | Immutable content hash, provenance, creation/update timestamps, generator/model metadata where safe |

### Generation and validation rules

1. The LLM emits data conforming to the schema; it does not emit executable programs.
2. The deterministic validator checks schema/version, component types, bindings, logical operations, connector references, method/path allowlists, permission consistency, expression syntax, bounds, and required fixtures.
3. Unknown components, unknown operations, undeclared connector paths, unbounded loops, arbitrary URLs, and policy-bypassing instructions are rejected.
4. Model output and connector/provider content are untrusted data—not executable instructions and not authority.
5. A valid spec gets an immutable version/content hash. Editing creates a new version; an active run remains bound to the version with which it started.
6. Preview and dry-run must not silently execute external writes.
7. The generic runner is registered once in the framework or composition host. User-created apps are declarative data interpreted by that runner, not newly compiled Go binaries.
8. Keep the initial expression/step language deliberately small and typed. Avoid embedding a general-purpose programming language in the Hacksgiving prototype.

### UI rendering

The Studio should render the generated application from declarative UI component definitions and schemas. Forms should be schema-driven; tables, detail views, metric cards, navigation, status/error states, and simple charts should be reusable primitives. The implementation plan must inspect existing UI/backend assets and choose the smallest coherent web stack; do not introduce a large frontend framework without a repository-based reason.

---

## 6. Primitive families exposed to non-technical users

### 6.1 Data and state

Entities/collections, fields and types, relationships, filters, sorting, pagination, search, aggregations, metrics, scoped state, source bindings.

### 6.2 UI and presentation

App shell/navigation, dashboard, metric card, table/list, search/filter, detail view/card, form, action button, chart/summary panel, status/loading/empty/error states, approval/confirmation card, permissions card, run status/timeline. Themes, full branding and advanced layout are later product work.

### 6.3 Workflow and control

Manual trigger, connector read/action, model step, transform, condition/branch, bounded loop, result/finish, and (only where tested) a simple confirmation gate. Future primitives include schedule, webhook/provider-event triggers, durable wait/signal, long-lived approval, retry/backoff, cancellation, deadlines, entity serialization, compensation/saga.

### 6.4 Integration and connectors

Public connector catalog, organization-connection discovery, per-Sprite access state, connection selection, Custom API connector wizard, operation/capability discovery, provider adapter bindings, and webhooks as a later capability.

### 6.5 Governance and safety

Declared capability set, least-privilege policy, provider/Sprite/path boundaries, read/write classification, effect/reversibility classification, confirmation/approval requirements, run/action/recipient limits where implemented, kill/disable controls where implemented, secret redaction, untrusted-content handling, and audit/evidence surfaces.

### 6.6 Evidence and lifecycle

Spec inspector, validation report, fixture/dry-run report, app version history, run result/status, connector call evidence, and errors. Full durable timeline, UNKNOWN-effect inbox, replay debugger, checkpoint history, compensation/rewind interface, app publishing/version promotion, and multi-user administration are roadmap unless independently verified.

---

## 7. Fly Sprites ecosystem and connector architecture

This section is a mandatory product requirement. Miniapp Studio must display the supported Fly ecosystem and let users extend it through Custom API connectors. Keep the following catalogs separate.

### 7.1 Three connector states—never collapse them

1. **Public ecosystem catalog:** connector/provider types Fly publicly lists on the Sprites ecosystem page. This is a browsable catalog and metadata source; it does not mean the current organization has enabled the connector.
2. **Organization-configured connections:** the connections available to the authenticated Fly organization. Fly documents `GET https://api.sprites.dev/v1/oauth/connections` as listing connectors configured for the authenticated organization. It is not, by itself, the full public provider catalog.
3. **Per-Sprite gateway visibility/authorization:** what a particular Sprite can actually call through the gateway. The Sprite-side gateway discovery surface (`https://api.sprites.dev/v1/gateway/list`) and the connector's access policy must be checked in the Sprite context. Verify the current response semantics before implementing the UI.

The Studio must present distinct states such as `Listed in Fly ecosystem`, `Not configured for organization`, `Configured for organization`, `Allowed for this Sprite`, `Denied for this Sprite`, `Needs scope/authorization`, `Test passed`, `Test failed`, and `Unknown`. Do not infer these states from provider name alone.

### 7.2 Ecosystem Explorer: connectors vs integrations vs SDKs

The Studio should provide a coherent **Fly Ecosystem** area with separate tabs/sections:

- **Connectors:** external services and APIs reachable by Sprites through the connector gateway. This is the end-user integration catalog.
- **Integrations:** agent clients, plugins, Remote MCP, platform integrations, and SDK integrations that help agents/developers use Sprites. They are not business API connectors.
- **SDKs:** language libraries, including the official Go SDK. These are developer tooling, not end-user connector instances.

The public ecosystem page is the source for catalog metadata and links. The implementation plan must decide how metadata is refreshed/versioned and must include every connector Fly lists at the time of implementation, not a stale count or a handpicked subset. Do not claim a live API exists for the whole marketing catalog unless verified; use the public ecosystem directory as the catalog source and the authenticated APIs for live configuration/authorization status. Show integrations and SDKs in separate sections. The current Fly description of TypeSafeAI Jev identifies it as a decision model for yes/no, choice, and score decisions with probability/confidence; do not mislabel it as the JSON Schema validator or type system.

### 7.3 Custom API connectors are first-class

Studio must support adding a token-authenticated public HTTP API that does not have a dedicated Fly connector. Custom API is not an optional post-hackathon idea; it is part of the platform's extensibility story and should be part of the Hacksgiving implementation scope. The implementation plan should scope a narrow but real create/test/use path (one public token-authenticated API and one declared operation is sufficient). If current organization privileges or upstream constraints prevent a live path, record the blocker and provide a clearly labeled fallback—not a mocked integration presented as live.

The official documentation currently lists these endpoints:

- `POST /v1/oauth/connections/custom_api` — create a Custom API connection.
- `POST /v1/oauth/connections/custom_api/test` — test a Custom API connection or test supplied credentials before saving.
- `PUT /v1/oauth/connections/custom_api/{id}` — update an existing Custom API connection.
- `GET /v1/oauth/connections` — list configured organization connections.

The create documentation explicitly states that creating does not test the credential. Treat testing and creation as separate steps/states. The docs also state that creating another Custom API connection with the same name updates the existing connection in place; the UI must warn/confirm before replacing an existing connection.

Documented authentication placement modes include `header`, `custom_header`, `query_param`, and `url_path`, with provider-specific fields. The exact request/response schema, required fields, default policy values, endpoint-rule syntax, and error semantics must be taken from the current OpenAPI/API docs by the coding agent—not copied from stale examples.

The gateway is designed to keep connector-managed provider credentials in Fly's control plane and attach them to requests from authorized Sprites. Do not send raw provider secrets to the Sprite, agent context, MiniappSpec, filesystem, environment, logs, run timeline, or API responses. Custom API upstreams must obey the current Fly restrictions; do not promise private/loopback API support where Fly blocks it.

### 7.4 Access policies and least privilege

- Default deny; never select “allow all” as the default.
- Restrict each connector to the intended Sprite using name prefix and/or required labels where supported.
- Restrict request paths with allow/block rules where supported; verify exact semantics and whether a connector type supports path rules.
- If both allow and block rules exist, follow the documented precedence (verify from current source; do not assume).
- Show the effective policy in the Studio before deployment and show the Sprite-level authorization result after deployment.
- Privileged connector-management calls must execute in the trusted Studio control plane using an authorized Fly identity, not through a broad token handed to the Sprite-hosted builder agent.
- Redact credentials and token-bearing URLs from all logs and UI surfaces. Prefer header-based secret transport where supported; if query/path auth is supported, explain its exposure risks and ensure it is never logged.

### 7.5 Connector vs capability discovery

Fly connector discovery answers: **which external services can this Sprite reach?**

Miniapp capability discovery answers: **which business operations may this generated app invoke, under which schemas, policies, and effect classifications?**

They are complementary but not interchangeable. A connection to an API does not grant the generated app unrestricted authority over every endpoint. Miniapp must register operation descriptors with stable logical IDs, method/path constraints, request/response schemas, read/write/effect classification, approval policy, limits, and timeouts. Generated specs reference only those operations. The runtime rejects undeclared methods, paths, and operation IDs.

### 7.6 Fly Go SDK and control-plane integration

The Fly ecosystem page lists an official Go SDK at `github.com/superfly/sprites-go`. The plan should evaluate and pin it behind Miniapp-owned interfaces for supported Sprite lifecycle, exec, filesystem, service, checkpoint, and policy operations. Do not assume the SDK covers every connector-management API. Implement a separate typed Go REST client for documented Connectors API endpoints where necessary, with contract tests and fake implementations.

Check Go toolchain and SDK version compatibility against the actual repository. Do not upgrade the whole project toolchain without a documented compatibility reason and test evidence.

---

## 8. Hacksgiving deliverable: one golden path

### 8.1 Mandatory evidence

A judge should be able to observe, rather than merely hear claims, that:

1. A personal agent runs on a real Fly.io Sprite.
2. That Sprite-hosted agent makes at least one real call through a Sprites connector.
3. The Studio displays the public connector ecosystem and distinguishes it from organization-configured and Sprite-authorized connections.
4. Studio exposes Custom API as an extension path and demonstrates a real tested Custom API connection if feasible.
5. A non-technical user describes an app in natural language.
6. The Sprite-hosted agent produces a declarative MiniappSpec, not arbitrary executable code.
7. Deterministic validation gives visible pass/fail results, including at least one rejected invalid/unsafe spec or operation.
8. Studio renders a working generated app from reusable UI/data primitives.
9. The generated app performs one bounded, real connector-backed operation and shows its result.
10. The README/demo explicitly distinguishes delivered capabilities from the durable-workflow roadmap.

### 8.2 Recommended demo app

Use **Invoice Review Desk** as the primary demonstrator if the required connectors are available: a user describes an invoice review workspace; Studio generates a dashboard/table/detail view, links invoice documents through a native connector such as Box, and obtains a purchase-order or reference record through a Custom API test service. The rendered app shows the invoice, extracted fields or sample fixture, matched reference, and review status.

Do not hardcode Box or any provider as mandatory. On the first implementation day, inspect the user's actual organization catalog and choose a native connector that is enabled and reliable. If Box is unavailable, choose another actual connector that best supports the demo. The Custom API connector should provide the extensibility proof.

A read-only connector-backed demo is preferable to a risky external write. If a write is included, require an explicit user confirmation and show only the behavior verified by the pinned implementation.

### 8.3 Three-minute demonstration sequence

| Time | Scene | What it proves |
|---|---|---|
| 0:00–0:20 | Introduce Studio and enter a plain-language business request | The product is an application factory for non-programmers |
| 0:20–0:45 | Show connector ecosystem browser and live org/Sprite statuses | Connector extensibility is real; catalog and connected state are not confused |
| 0:45–1:15 | Sprite-hosted agent creates the MiniappSpec | The personal agent runs on a Sprite and creates a real declarative artifact |
| 1:15–1:40 | Inspect app definition, components, data bindings, connector operations, and permissions | This is more than a prompt wrapper |
| 1:40–2:00 | Run deterministic validation; show one deliberate rejection | Safety and correctness are enforced outside the LLM |
| 2:00–2:40 | Render and use the generated application against a real connector-backed read | Working generated UI and actual connector use |
| 2:40–3:00 | Show the app as one template generated from general primitives; explain durable workflow roadmap accurately | Product generality and credible long-term vision |

One complete app is mandatory. A second generated variation is optional and must not jeopardize the primary golden path. A full workflow canvas is not part of the Hacksgiving scope.

### 8.4 Hacksgiving cut list

Do not require the following for the submission: full drag-and-drop workflow designer, marketplace, billing, multi-tenant accounts, white-label embed, custom domains, advanced theming, dozens of provider-specific Miniapp business adapters, arbitrary code generation/execution, multi-Sprite orchestration, multi-day DBOS waits, durable approval parking, saga/compensation UI, UNKNOWN reconciliation console, or a claim of generalized rewind/undo.

---

## 9. Durable workflow roadmap — the strategic differentiator

The product roadmap should make durable workflow central, but keep it distinct from the Hacksgiving scope. The implementation plan must inspect the current Miniapp framework and record which DBOS/River/effect/approval capabilities are genuinely implemented, tested, and production-hardened. Do not rely on older plans or assertions as proof.

### Intended future experience for end users

A non-technical user should eventually be able to define a business process such as:

- When an invoice arrives, extract its details.
- Match it with a purchase order.
- If it matches, request approval from the responsible manager.
- Wait for approval for days if necessary.
- On approval, initiate the permitted next action.
- If an API times out with an unknown result, reconcile by reading the provider before deciding whether to retry.
- Preserve step evidence, approvals, external receipts, and business history.
- Survive worker/process restarts and continue from the correct durable state.

### Roadmap capabilities

1. DBOS-backed durable workflow definitions and runtime for Class D-style business processes, subject to the existing Miniapp architecture's workload-specific adoption policy.
2. Durable timers, waits, signals, webhook/event wakeups, and human approvals that survive app/browser/worker interruption.
3. Effect state machine and immutable effect ledger with explicit `PENDING`, `ATTEMPTING`, `CONFIRMED`, `FAILED`, and `UNKNOWN` handling where the framework adopts these semantics.
4. Stable idempotency identities, provider-aware reconcile-by-read, and a human UNKNOWN/exception inbox.
5. Cancellation, deadlines, retries/backoff, entity serialization, rate/concurrency limits, and blast-radius controls.
6. Saga/compensation design for operations that can be compensated; irreversible actions are explicitly labeled as irreversible.
7. Version-safe workflow upgrades, old-version drain, replay/debugging, audit evidence, and operational recovery.
8. A visual workflow authoring and run-inspection experience that compiles to the constrained workflow definition.

### Correctness language

Never claim exactly-once execution of external providers. The target is honest at-least-once step execution with idempotency and reconciliation sufficient to prevent duplicate *confirmed* business effects where provider semantics permit it, no silently lost operations, and no blind retry of unknown outcomes. The exact guarantee must be stated per operation/provider and supported by crash/failure tests.

Keep these state domains distinct:

- **MiniappSpec state:** app definition/version/content hash.
- **Business workflow state:** DBOS checkpoints, timers, signals, approvals, step results, effect ledger.
- **Sprite filesystem state:** Fly checkpoint snapshots of files/environment.

Restoring a Sprite checkpoint does not rewind DBOS state or reverse an external API call. Any later Rewind feature must coordinate effect reconciliation/compensation and flag irreversible effects; filesystem restore alone must never be advertised as business undo.

---

## 10. Repository provenance and Hacksgiving rule compliance

The Hacksgiving brief requires the submitted personal agent to run on a Sprite, use at least one Sprites connector, and be built during Hacksgiving week. Because the Miniapp framework predates the event:

- Inspect and pin the exact pre-existing framework revision used.
- Keep the new Studio, spec generator/compiler, UI renderer, connector catalog/management integration, and Hacksgiving demo clearly attributable to work done during Hacksgiving.
- Prefer the framework as a pinned dependency rather than copying its old source into the new submission repository, subject to organizer guidance.
- Disclose the pre-existing foundation clearly in README/submission materials.
- Ask Fly organizers whether the pre-existing framework dependency is acceptable and preserve their answer. If eligibility is uncertain, prepare a minimal fallback only after a deliberate decision; do not duplicate large framework subsystems without need.
- Do not claim the pre-existing framework or old commits were created during Hacksgiving.

---

## 11. Main risks and acceptance gates

| Gate | Required evidence |
|---|---|
| G0 — repository and dependency truth | Exact repo/module state, framework revision, Go toolchain, tests, Fly SDK/API pins, build/CI status, and known blockers recorded |
| G1 — Sprite and connector compliance | Real personal agent process runs on a Sprite and successfully makes a real gateway call through at least one Sprites connector; no raw provider secret appears in Sprite filesystem/env/spec/logs |
| G2 — connector ecosystem truth | Public catalog, organization connection state, and per-Sprite authorization shown as distinct data sources/states; integrations and SDKs are not mislabeled as connectors |
| G3 — Custom API | Secure wizard or tested provisioning flow creates/tests a token-authenticated public API connection; no secret is returned, persisted in generated spec, or passed into the Sprite; default-deny policy verified |
| G4 — spec generation | Prompt produces a schema-valid MiniappSpec; invalid component/operation/path/policy is rejected deterministically; content version/hash behavior is tested or scoped explicitly |
| G5 — UI generation | Generated UI renders from the spec using reusable components; bindings resolve against fixture/live source data; errors and empty states are visible |
| G6 — app execution | One bounded, repeatable generated app path executes on the Sprite and shows connector-backed output; no unapproved external write occurs |
| G7 — demo readiness | Clean-state rehearsal, documented known limitations, no hidden mock result presented as live, and exact claims match passing tests |

### Risk register

| Risk | Mitigation |
|---|---|
| Looks like a prompt wrapper | Make MiniappSpec visible; show deterministic validation, permissions, operation contracts, and a working rendered app |
| Public catalog confused with connected state | Separate metadata catalog, organization connections, and Sprite-level discovery/policy |
| Org lacks a featured connector | Check actual org state early; select an available native connector; Custom API is the extensibility fallback |
| Custom API becomes arbitrary proxy | Restrict token-auth API setup to Fly's supported connector model; declare operation paths/methods/schemas; reject undeclared calls |
| Secret leaks to agent/spec/log | Keep privileged operations in trusted control plane; redact secrets; test absence from Sprite and generated artifacts |
| Generated spec can bypass policy | Static validator plus runtime authorization; no LLM-only security decisions; deny unknown operations by default |
| Framework portability to Sprite fails | Run an early compatibility spike; separate Studio control plane from Sprite agent; use a small tested fallback only if justified |
| Scope overwhelms hackathon | One golden path; no full workflow canvas; no marketplace/multi-tenant/billing/DBOS durable workflow UI |
| Misleading “undo” or durability claim | Separate filesystem checkpoints, external effects, and DBOS workflow state; claim only tested behavior |
| Pre-existing foundation eligibility challenged | Organizer question, pinned dependency, transparent attribution and clear Hacksgiving-week commits |

---

## 12. External references to re-check during implementation planning

The coding agent must re-open current official sources and record exact versions/dates; this document is a direction, not a substitute for current API/schema inspection.

- [Fly.io Hacksgiving rules](https://fly.io/hacksgiving/)
- [Fly.io Sprites overview](https://docs.fly.io/sprites)
- [Fly.io Sprites ecosystem — connectors, integrations and SDKs](https://fly.io/sprites/ecosystem/)
- [Sprites connector API overview](https://docs.fly.io/sprites/api/connectors)
- [List organization-configured connectors](https://docs.fly.io/sprites/api/connectors/list-connectors)
- [Create a Custom API connector](https://docs.fly.io/sprites/api/connectors/create-a-custom-api-connector)
- [Test a Custom API connector](https://docs.fly.io/sprites/api/connectors/test-a-custom-api-connector)
- [Official Go SDK](https://github.com/superfly/sprites-go)

---

## 13. Final consolidated thesis

**Miniapp Studio gives non-technical business users a way to generate Miniapp-based applications from reusable, governed primitives rather than commissioning every application separately.** The Studio is the product users see; the Sprite-hosted personal agent generates a constrained declarative specification; the compiler validates it; the renderer produces the UI; connector discovery and Custom API make the platform extensible; and the Miniapp framework supplies trusted execution and policy boundaries. The Hacksgiving proof is one complete generated application running through a real Sprite and real Sprites connector. Durable DBOS workflows remain the central long-term differentiator and must be presented as roadmap until their end-user authoring and runtime guarantees are verified.
