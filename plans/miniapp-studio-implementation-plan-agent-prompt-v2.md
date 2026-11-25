# Coding-Agent Prompt v2: Produce the Fly-Native Miniapp Studio MVP Implementation Plan (Planning Only)

You are acting as a principal software architect, staff Go engineer, and product-minded technical planner for **Miniapp Studio**. Your task is to inspect the actual repository and produce a rigorous, evidence-based implementation plan for our review.

**Do not implement the plan. Do not write application code, migrations, UI code, tests, CI changes, or deployment configuration.** Your only deliverable is the implementation-plan document described below, plus any diagrams placed inline in that document. You may run read-only inspection commands and existing tests/builds to establish repository truth, but do not change source files, dependencies, generated files, lockfiles, Fly resources, databases, or git history. The only file you may create or modify is the requested plan document.

## 1. Product goal — treat this as the governing brief

We are building **Miniapp Studio**, an AI-native application factory for non-technical business users. Users describe a business need in natural language and generate a Miniapp-based application from reusable data, UI, integration, workflow, and governance primitives. We are not building twelve individual vertical apps.

The product should follow the GigaCatalyst-style model: give target-market end users generic primitives, an accessible UI, and composable workflows so they can generate their own applications without programming. The generated apps are definitions assembled from a common platform—not separately hand-coded products.

The existing `github.com/PithomLabs/miniapp` Go framework predates Hacksgiving and serves as the foundation. The visible Hacksgiving product is a polished, first-party Miniapp Studio application built using Miniapp itself, deployed as a serverless-style Fly.io app, plus the app generator/spec compiler, generic renderer/runner, connector ecosystem explorer, Custom API connector extensibility, and one complete generated Miniapp with its own stable Fly-issued URL derived from the user input.

The Hacksgiving brief requires a personal agent that runs on a Fly.io Sprite and uses at least one Sprites connector. The plan must make this compliance visible and testable. The agent—not merely a static web UI—must run on a real Sprite and make a real call through a Sprites connector. The generated Miniapp must also be deployed to a real Fly target and served at its own verified URL.

### Core product promise

> Describe the business application you need. Miniapp Studio composes it from governed, reusable primitives, connects it to the systems you authorize, and shows you what it will do before it runs.

### Long-term differentiator and claims boundary

Durable business-workflow execution is the central long-term Miniapp differentiator. It should eventually include DBOS-backed long-running workflows, durable waits/signals, approvals that survive interruption, retries, cancellation, stable idempotency, external-effect reconciliation, UNKNOWN-outcome handling, compensation where possible, workflow versioning, and audit/replay tooling.

**Durable workflow authoring and production-grade multi-day workflow guarantees are roadmap, not Hacksgiving demo scope.** Do not turn the hackathon into a DBOS showcase. Do not claim exactly-once external effects, zero duplicate effects, always-undo, durable multi-day waits, or workflow replay unless exact behavior exists and is proven by current tests. A Sprite filesystem checkpoint is not a DBOS workflow checkpoint and does not undo external API effects.

### Additional binding product and deployment requirements (revision 2)

These requirements are mandatory and supersede any earlier plan wording that treats them as optional or out of scope.

1. **Dogfood Miniapp:** Miniapp Studio itself must be built as a first-party Miniapp using the Miniapp framework's public APIs, runtime, UI/app primitives, capability path, and policy boundaries wherever supported. Do not build Studio as a bespoke sidecar that bypasses Miniapp. If the framework cannot yet render/host its own Studio UI or support required control-plane functions, inspect the gap and plan the smallest foundational extension. Avoid recursive self-generation: Studio has a deliberate, versioned bootstrap definition.
2. **Fly-hosted Studio:** Deploy Studio itself to Fly.io as a serverless-style web application on Fly Machines. Use Fly Proxy autostop/autostart where appropriate (`auto_start_machines = true`, `auto_stop_machines = "stop"` or `"suspend"`, `min_machines_running = 0` only if measured availability permits). Do not call this conventional FaaS. Fly Proxy starts/stops existing Machines; it does not create/destroy the Machine pool. Plan for cold starts, health checks, concurrency and background-worker implications. A DBOS worker or timer cannot be assumed to wake a scale-to-zero HTTP Machine.
3. **Every generated Miniapp has its own URL:** A user-supplied or user-confirmed app name/slug must produce a unique, stable, persisted URL for the generated app. Evaluate Fly Sprite independent HTTPS URLs as the preferred option: normalize the slug into a collision-safe Sprite/resource name, deploy the generic Miniapp runtime as an HTTP service (verify port/service/URL behavior), fetch the authoritative URL from the Fly API/CLI response, and store it with app identity/version. Keep URLs private/authenticated by default where supported; public publishing requires explicit consent, warnings, and app-level authorization. Do not invent hostnames or assume arbitrary custom subdomains.
4. **Fly-native platform:** Treat Fly.io as the primary compute, networking, deployment and lifecycle platform. Prefer Fly Apps/Machines, Sprites, Fly Proxy, Fly certificates/URLs, Fly Managed Postgres (MPG), and Fly ecosystem storage/services where technically appropriate. Reassess the old Neon default: evaluate MPG v2 compatibility with the framework, selected Postgres driver/pool, DBOS (if used), pricing, region and operational requirements. Do not silently replace Neon or assume MPG is compatible; record evidence and a decision. Do not use unsupported unmanaged Fly Postgres as a shortcut.
5. **Tight Fly CLI/API integration:** Use `flyctl` for reproducible source deployment, CI/release workflows and operator diagnostics; use supported Fly Apps/Machines/Sprites/Connectors APIs behind typed Miniapp-owned interfaces for product-driven provisioning and lifecycle. Evaluate official `sprites-go` for the operations it actually supports and add typed REST coverage for API gaps. Fly MCP may be used where appropriate with narrow OAuth scopes. Never place broad Fly org credentials in a Sprite, browser, MiniappSpec, generated artifact, prompt or log. Never build shell commands from untrusted user text.
6. **Hacksgiving is the first MVP:** It must be narrow but polished, usable and repeatable—not a throwaway mock. Plan responsive and coherent Studio UI, onboarding, generation/validation feedback, live app preview and deployed URL, connector permissions, meaningful loading/empty/error states, Fly provisioning/deploy statuses, diagnostics and a rehearsable demo reset. One complete generated Miniapp with its own URL is mandatory; multiple sample apps, marketplace, billing and full visual workflow editor are not.
7. **URL hosting options:** Compare (a) one Sprite per generated Miniapp with Fly-issued Sprite HTTPS URL, (b) shared Fly App/router with slug path/subdomain, and (c) one Fly App per generated Miniapp. Compare isolation, real URL semantics, cold starts, cost, resource limits, provisioning, upgrades, cleanup and security. Recommend an MVP architecture that guarantees a stable unique user-derived URL. Do not assume one Sprite per app is cost/scale free; do not assume a URL route is equivalent to resource isolation.
8. **Storage and state:** Evaluate Fly Managed Postgres (MPG) as the default platform database if compatible and available. Evaluate Fly-native Tigris or a supported S3-compatible provider for off-Machine blobs/ledgers. Persist app specs, hashes, Fly resource IDs, slugs, URLs and lifecycle state outside an autostopped Machine's ephemeral filesystem. Distinguish generated-app durable data from Sprite filesystem state.

## 2. Required source materials

Read and reconcile the following materials when they are accessible in the current coding-agent workspace:

- `miniapp-studio-consolidated-direction-v3.md` (the latest consolidated product/architecture direction; it supersedes v2).
- `miniapp-studio-consolidated-plan.md` (earlier detailed consolidated plan, if present).
- `plan_deepseek(4).md`, `plan_gemini(2).md`, and `plan_z(3).md` (the three attached plans, if present).
- All relevant Miniapp repository documentation, `AGENTS.md`, ADRs, prior implementation plans, review notes, and test plans.

Do not assume these documents are current code evidence. Use them as direction and inspect the actual repository before deciding what exists.

## 3. Mandatory repository reconnaissance before planning

Start by establishing the actual state of the workspace and repositories. At minimum inspect:

1. Current working directory, repository roots, git status, current branch, recent commits/tags, and uncommitted changes.
2. `AGENTS.md` at every relevant level, README files, `go.mod`, Go version, module replacements/workspaces, dependency pins, Makefiles/task runners, CI workflows, Fly configuration, and deployment entry points.
3. Existing `docs/architecture`, `docs/plans`, ADRs, risk registers, product specifications, and any previous Miniapp Studio/generator plans.
4. The Miniapp framework's actual public interfaces and implementations for contracts/manifests, registry, invocation, runtime, authorization, capability discovery/proxy, connector adapters, effects, approvals, audit, persistence, DBOS, River, Sprite placement/deploy, MCP/REST/CLI, and test kit.
5. Existing UI/frontend assets, if any. Identify the actual framework, templates, renderer, static assets, and server-rendering approach. If no UI exists, recommend the smallest maintainable UI approach compatible with the repository; do not introduce a large frontend framework without evidence and justification.
6. Existing schemas, migrations, database adapters, test fixtures, mocks/fakes, integration tests, and CI coverage.
7. Existing work related to the MiniappSpec, app DSL, plan interpreter, schema generation, JSON Schema, and capability registry. Do not assume features described in plans are implemented.
8. The pinned/current `sprites-go` dependency or checkout, if present, plus any Fly REST client or MCP integration.

For every major capability, classify it as one of:

- Implemented and verified by named tests/evidence.
- Implemented but not adequately verified.
- Partially implemented.
- Designed/documented only.
- Absent.
- Blocked by an environment or upstream limitation.

Cite actual repository file paths, symbol names, test names, commit SHAs and line numbers where practical. Never describe a design document as proof that the capability works.

Run existing non-destructive tests/build/type checks only if the environment permits. Record exact commands and results. If a command cannot run, state why; do not invent a green result. Preserve the repository's current working state.

## 4. Verify current Fly documentation and API contracts

Re-open current official documentation during planning. Record date checked, exact endpoint paths, authentication requirements, request/response schemas, scopes/permissions, version constraints, and relevant limitations.

Required sources:

- https://fly.io/hacksgiving/
- https://docs.fly.io/sprites
- https://fly.io/sprites/ecosystem/
- https://docs.fly.io/sprites/api/connectors
- https://docs.fly.io/sprites/api/connectors/list-connectors
- https://docs.fly.io/sprites/api/connectors/create-a-custom-api-connector
- https://docs.fly.io/sprites/api/connectors/test-a-custom-api-connector
- https://docs.fly.io/sprites/api/connectors/update-a-custom-api-connector
- https://github.com/superfly/sprites-go
- https://fly.io/docs/reference/fly-proxy-autostop-autostart/
- https://fly.io/docs/machines/api/apps-resource/
- https://fly.io/docs/machines/api/machines-resource/
- https://fly.io/docs/networking/custom-domain/
- https://fly.io/docs/mpg/
- https://fly.io/docs/flyctl/mpg/

The plan must distinguish these catalog surfaces precisely:

1. The public Fly Sprites ecosystem connector catalog (public metadata/listing).
2. Organization-configured connector connections. The documented `GET /v1/oauth/connections` lists connections configured for the authenticated organization; it is not the full global public catalog.
3. Per-Sprite connector discovery and authorization. Verify current behavior of the Sprite-side gateway discovery endpoint, commonly shown as `https://api.sprites.dev/v1/gateway/list`, and determine what is visible only from within a Sprite.

Also distinguish the ecosystem's **Connectors**, **Integrations/plugins/MCP/platform integrations**, and **SDKs**. Do not put plugins or SDKs into the API connector list. Studio should be able to display all supported public connectors and separately surface relevant integrations and SDKs, including the official Go SDK, without misrepresenting their type or availability.

Do not hardcode an old catalog count or assume all providers are enabled for the user's organization. Determine a defensible metadata refresh/versioning strategy from the current ecosystem page and API docs. Clearly mark community-maintained entries where Fly marks them as community.

### Custom API connector requirements

Custom API is a first-class Miniapp Studio extension point. Inspect the current API/OpenAPI schema for:

- Create, test, update, list, get, and provision operations as applicable.
- Supported authentication placement values (including header, custom header, query parameter, and URL path where still supported).
- Exact required/optional fields and naming behavior.
- Credential storage/response behavior.
- Whether create performs a test (do not assume it does).
- How the test endpoint handles stored versus supplied credentials and which origins it permits.
- Access-policy fields, defaults, supported path patterns, Sprite name-prefix/label restrictions, block/allow precedence, and connector-specific exceptions.
- Required Fly organization authorization scopes/privileges.
- Error semantics and status-body behavior.

Design a user-facing wizard (Define → Test → Scope/Grant) only after verifying the sequence supported by the API. The UI must warn when creating a connection name would update/replace an existing connection. Default-deny must be the safe default. Never set allow-all by default.

The raw token must be submitted only to the trusted Studio control plane/Fly connector API. It must never enter the Sprite, LLM prompt/context, MiniappSpec, generated files, environment variables, logs, timeline, or returned UI payload. The public upstream API must satisfy Fly's current constraints; do not promise private/loopback support unless the docs prove it.

### Fly Go SDK boundary

Evaluate `github.com/superfly/sprites-go` for the Sprite operations it actually supports (lifecycle, exec, filesystem, services, checkpoints, network/resource/privilege policies as verified). Do not assume it includes connector-management APIs. Where it does not, propose a narrow typed Go REST client behind Miniapp-owned interfaces. Pin compatible SDK/toolchain versions and plan fakes/contract tests.

## 5. Target architecture the implementation plan must evaluate

Use this as the intended separation, but adjust only with repository evidence and explicit rationale:

### Layer 1 — Miniapp Studio (visible product)

- Natural-language describe/edit interface.
- Small template/gallery surface.
- App definition/spec inspector.
- Connector and Fly ecosystem explorer.
- Custom API setup/test wizard.
- Permission/capability summary.
- Live generated-app preview.
- Minimal run status/results.
- Basic stop/disable or explicit confirmation only if implemented and tested.

### Layer 2 — application composition/compiler

- Sprite-hosted builder/personal agent produces a declarative MiniappSpec.
- Deterministic schema and policy validator.
- Capability/connector/operation resolver.
- Optional fixture/dry-run validator.
- Generic, fixed spec interpreter/runner registered once.
- Generic UI renderer driven from declarative component definitions and data bindings.

### Layer 3 — Miniapp Go framework foundation

- Existing contracts/manifests/registry/runtime and public APIs.
- Authorization and capability enforcement.
- Provider adapters/proxy and external effect boundaries.
- Audit/effect/approval mechanisms only where actual code and tests support them.
- DBOS/River/durable workflow capabilities as verified foundations and roadmap integrations; do not imply they are part of the hackathon golden path.

### Fly placement and identity

For the Hacksgiving golden path, the Studio itself is a first-party Miniapp deployed on Fly Apps/Machines, using serverless-style autostop/autostart where appropriate. A personal builder agent must demonstrably run on a real authorized Sprite and use at least one real Sprites connector. The generated Miniapp must be deployed to a real Fly target and have its own stable URL derived from the user-provided slug.

The implementation plan must decide whether the best generated-app target is a dedicated Sprite (preferred to evaluate first because Sprites expose independent HTTPS URLs and can wake on request), a shared Fly App/router, or one Fly App per generated app. Compare these honestly. The single-app Hacksgiving MVP may share the builder-agent Sprite with its generated app only if the trust and process boundaries are explicit and it still gives the app a real unique URL. Do not fabricate URL hostnames; persist the URL returned by Fly's authoritative API/CLI. Default to private URL access; public exposure requires explicit publish consent and application-level authorization.

The trusted Studio control plane must own privileged organization-level actions, especially connector creation, Fly app/Sprite provisioning, Fly management credentials and persistent state. Do not hand broad organization tokens to the Sprite agent or browser. The plan must show precisely which process runs on the Studio Fly App, which process runs on a Sprite, where generated UI/data is served, how the URL is routed, and how the agent reaches Miniapp APIs. Draw a deployment diagram and trace the end-to-end trust, identity, credential and network boundaries.

Treat Fly.io as the primary infrastructure platform. Evaluate Fly Managed Postgres (MPG v2) in place of the older Neon default, including compatibility with the actual framework and DBOS code, driver/pool, networking, price and resilience. Evaluate Fly-native object storage such as Tigris where appropriate. Record any external service exception. Use `flyctl` for reproducible deployment/ops workflows and Fly/Sprites APIs behind typed adapters for product lifecycle operations. Do not shell out with user-supplied command strings. Use Sprites SDK only for verified coverage; connector-management REST API may require a separate typed client.

The Studio web service may use Fly Proxy autostop/autostart (`auto_start_machines=true`, `auto_stop_machines=stop|suspend`, `min_machines_running=0`) to provide serverless-style behavior, but do not call it conventional FaaS. Fly Proxy only starts/stops existing Machines; it does not provision/delete them. Evaluate cold starts, minimum warm capacity, background workers, DBOS timers, and queue wake topology. Scale-to-zero HTTP hosting and long-running workers may require separate Fly process groups/apps and explicit lifecycle policies.

For the longer-term product, keep execution semantics and compute placement as independent dimensions. Do not mandate one Sprite per generated template/app without evidence; nevertheless, every deployed generated Miniapp must have its own stable URL. Placement, URL ownership, isolation, lifecycle and cost must be decided together and tested.

### Fly-native deployment and per-Miniapp URL architecture (mandatory planning section)

The implementation plan must include a separate Fly deployment section with:

- **Studio self-hosting/dogfooding:** prove how the Studio app itself is authored as a first-party Miniapp using the framework public API, app definition, shared renderer/components and policy/invocation path. State what must be added to the framework if the required UI or composition support does not exist. No bespoke bypass backend.
- **Studio hosting:** Fly App/Machines, Docker/build path, `fly.toml`, process groups, region, health/readiness, secrets, deployment/rollback, autostop/autostart configuration and cold-start budget. Treat it as serverless-style Fly Machines, not FaaS.
- **Generated app hosting and URL:** every generated Miniapp gets a URL derived from its validated user slug. Evaluate Sprite URL, shared app-router, and per-app Fly App approaches and recommend one. If using Sprites, verify Sprite creation, name restrictions, HTTP service port, URL retrieval, URL auth mode, wake behavior, service lifetime and delete/restore behavior from current docs/API. Never invent URL format; store Fly-returned URL and resource identifiers.
- **Lifecycle:** create/provision idempotently; wait/poll for readiness; publish; pause/stop; wake; update spec/runtime; preserve URL; roll back; unpublish; delete/retire; reconcile orphan Fly resources if the DB commit fails or deployment outcome is unknown.
- **Naming:** normalize and validate user-entered slug; handle reserved names, collision and concurrent creation, rename semantics, stable URL mapping, retries and idempotency. Display requested slug separately from Fly resource name if rules require normalization.
- **Access control:** private by default where supported; explicit confirmation for public exposure; app-level user/session authorization for protected business data; revoke/pause and unpublish behavior; no reliance on URL secrecy.
- **Fly control-plane integration:** typed adapters for Apps/Machines, Sprites, connectors, network policy, URL configuration, certificates if applicable, and database/object storage resources. Clarify the responsibility of `flyctl`, Sprite CLI, Fly APIs and Fly MCP; use least-privilege identity and audit every privileged call.
- **Fly-native persistence:** evaluate MPG v2 versus existing Neon assumption based on code-level compatibility, DBOS requirements, pricing and region. Determine whether Tigris or another Fly-supported storage option is needed for artifacts/effect receipts. Do not leave authoritative app specs/URLs solely on a stoppable Sprite.
- **Operational topology:** distinguish Studio HTTP request serving from background workers, scheduled jobs and DBOS executors. Explain how each wakes, runs and is monitored on Fly. If durable workers are roadmap, keep them out of the MVP and explicitly document the later topology.

Required end-to-end acceptance: create an app from a user-provided name, produce a collision-safe slug, provision its Fly target, obtain and persist the actual Fly URL, open the URL and render the correct Miniapp, verify it is private by default, and verify an explicit public-publish path only if implemented. Destroy/recreate and retry paths must not create duplicate or orphaned resources without reconciliation.

## 6. Canonical declarative MiniappSpec

Recommend one canonical, versioned definition, not several competing formats called AppSpec, MiniappSpec, and WorkflowPlan. A workflow is a section of the application specification.

Evaluate a spec with sections such as:

- `app`: stable ID, name, description, schema/app version.
- `data`: logical entities, fields, relationships, sources, queries, filters, aggregates, bindings.
- `ui`: pages, navigation, reusable components, component properties, data bindings, user actions.
- `workflows`: trigger metadata and constrained typed steps.
- `capabilities`: logical Miniapp operation IDs and provider/connection bindings.
- `policies`: access rules, effect/read-write classification, explicit confirmation requirements, rate/action limits.
- `fixtures`: deterministic sample inputs and expected outputs/validation cases.
- `presentation`: simple layout/theme tokens; advanced theming can be roadmap.
- `metadata`: spec schema version, app version, immutable content hash, generator provenance (no secrets).

The spec is data, not code. It must not carry arbitrary scripts, shell fragments, function bodies, unrestricted URLs, or free-form instructions that bypass registered operations.

The plan should describe:

- Canonical schema and migration/versioning policy.
- Validation stages and error taxonomy.
- Typed output/input references between steps.
- A deliberately small and bounded expression language (or no expression language for MVP).
- Immutable version/content hash and run-to-spec binding.
- How user edits create a new version without mutating an active run.
- Deterministic rendering and fixture testing.
- The one generic runner's registration/invocation path.

## 7. Reusable primitives

The plan must define a realistic Hacksgiving subset and a roadmap expansion for these families:

1. **Data/state:** entities, fields, relationships, sources, filters, search, sort, pagination, aggregates, metric bindings.
2. **UI:** app shell/navigation, dashboard, metric card, list/table, detail view, form, action button, basic chart, status/loading/empty/error state, connection/permission panel, simple run result.
3. **Workflow/control:** manual trigger, connector operation, model operation (optional), pure transform, condition/branch, bounded loop, result/finish. Scheduled/webhook/provider-event triggers, durable wait/signal, approval parking, retries, cancellation, deadlines, entity serialization and compensation belong to roadmap unless explicitly tested.
4. **Integrations:** public connector catalog, organization connection state, per-Sprite authorization state, connector selection, Custom API wizard, logical capability mapping.
5. **Governance/safety:** least privilege, declared operations, method/path constraints, secret redaction, untrusted-content handling, read/write/effect class, explicit confirmation, operation/run limits, stop control, audit evidence where implemented.
6. **Evidence/lifecycle:** spec inspector, validation report, fixture/dry-run report, version, run result, connector call evidence, basic error diagnostics. Full durable timeline, UNKNOWN queue, reconciliation console, replay debugger, compensation designer and publish/upgrade UI are roadmap.

Do not build a full drag-and-drop workflow canvas for Hacksgiving. Prefer conversational input plus a structured plan/spec inspector that a non-technical user can understand and edit through constrained controls.

## 8. Hacksgiving implementation scope and demonstration

### Must demonstrate

- A personal agent process running on a real Fly Sprite.
- At least one real Sprites connector call from that Sprite.
- Studio itself dogfooded as a first-party Miniapp, deployed to Fly.io.
- Studio UI that feels like a polished MVP app factory, not just a CLI or prompt textbox.
- Studio hosted as a Fly Machine service with documented serverless-style autostop/autostart behavior or a deliberate warm-capacity decision.
- A public Fly connector ecosystem browser and honest organization/Sprite status indicators.
- Custom API as a first-class extension path; plan for one actual Custom API creation/test/use flow if feasible and supported by privileges/time.
- A plain-language request generates a declarative MiniappSpec.
- Deterministic validation shows accepted and rejected cases.
- A generic renderer shows a real working generated UI.
- The generated Miniapp is deployed to a Fly target and receives its own stable URL derived from the user's name/slug.
- A bounded, repeatable connector-backed data operation in that generated app.
- A clear README/demo separation between Hacksgiving functionality and post-Hacksgiving durable-workflow roadmap.

### Demo application

Plan one application end-to-end, preferably an Invoice Review Desk if the available connectors make that credible. It may use a native connector such as Box and a Custom API connected to a small public test service. Do not assume Box or any named connector is enabled; inspect the actual organization first and choose the lowest-risk real provider available. A second prompt variation is optional, not a delivery dependency.

Prefer a read-only connector-backed workflow. If adding a write, require an explicit confirmation and only claim the behavior tested end-to-end.

### Do not put into the critical path

- DBOS multi-day waits or a full durable workflow UI.
- Visual workflow canvas.
- Marketplace, billing, multi-user enterprise SaaS, white-label embedding, customer-owned custom domains (Fly-issued per-app URLs are mandatory and not cut).
- Dozens of provider-specific business adapters.
- Arbitrary generated-code execution.
- Multi-Sprite fleet orchestration beyond what the single generated-app URL requires.
- General undo of external side effects.

## 9. Durable workflow roadmap

Create a separate roadmap section, not a Hacksgiving milestone. Evaluate current code to establish what is already present and what is only designed. The future product should expose DBOS-backed long-running processes with durable waits/events, approvals that survive restarts, safe retry and cancellation, idempotency identities, effect ledger, UNKNOWN quarantine, read-based reconciliation, compensation/sagas, entity serialization, workflow versioning, replay/debugging and audit evidence.

The implementation plan must describe the migration from the Hacksgiving bounded runner to a durable execution adapter without destabilizing the Miniapp framework or making DBOS a mandatory technology selector for every workflow. Existing architecture principles still apply: execution class and compute placement are independent; choose River/DBOS per workload and evidence; provider credentials do not enter workflow payloads; unknown external outcomes are not blindly retried.

## 10. Scope and schedule planning

The plan must prioritize a small number of end-to-end deliverables against the Hacksgiving deadline (November 29, 2026, midnight PST). Include a critical path, parallelizable work, explicit cut lines, and daily exit criteria only if they remain useful given the actual current date and repository state.

Do not present a fantasy schedule. Estimate work in engineer-hours or S/M/L with assumptions; identify the riskiest integration spikes first. Recommended first spikes to evaluate:

1. Can the chosen Sprite run the agent and call a real connector from inside the Sprite?
2. Can the trusted Studio backend provision/manage the Sprite and create/test a Custom API connector with the available Fly permissions?
3. Can the pinned Miniapp framework be consumed from the planned Studio repo and run in the chosen Sprite/Machine topology?
4. Can a minimal MiniappSpec be generated, validated deterministically, rendered into UI, and used for one connector-backed read?
5. Can Studio itself run as a first-party Miniapp on a Fly App/Machine, using the Miniapp renderer and public framework path?
6. Can one generated Miniapp be deployed to a Fly Sprite (or chosen Fly target) and receive a stable, slug-derived URL that opens the correct app?
7. Can Fly Proxy autostop/autostart meet Studio's latency/availability needs without breaking state, sessions or background execution?
8. Is Fly Managed Postgres (MPG v2) compatible with the actual Miniapp persistence/DBOS driver and connection pool, and is it viable for the MVP cost envelope?

## 11. Required implementation-plan document structure

Write a single, cohesive implementation plan at the repository's appropriate documentation path. Prefer `docs/architecture/miniapp-studio-implementation-plan.md` unless existing conventions indicate a better path; explain any deviation.

The document must contain:

1. Executive summary and recommendation.
2. Product scope, target user, product thesis, and explicit non-goals.
3. Consolidated decisions and an ADR/change log for conflicts resolved.
4. Repository/workspace inventory and source-of-truth findings.
5. Current-state capability matrix: implemented/tested/partial/design-only/absent/blocked with file/test evidence.
6. Functional user journeys and acceptance outcomes.
7. Logical architecture and deployment topology diagrams.
8. Trust boundaries, secrets flow, agent identity, user authorization, and approval separation.
9. Canonical MiniappSpec design, schemas, versioning/hash policy, compiler/validator, generic runner and renderer responsibilities.
10. Primitive catalog with Hacksgiving subset vs future roadmap.
11. Connector catalog model: public metadata vs organization connection vs per-Sprite authorization; connectors vs integrations vs SDKs.
12. Custom API connector API contract, wizard UX, privilege model, security controls, testing and failure states.
13. Fly Go SDK/API integration approach and exact dependency pins.
14. Data/persistence model and storage placement, with trade-offs.
15. UI information architecture, screens, states and implementation stack recommendation.
16. Hacksgiving golden path and demo storyboard.
17. First-party Studio-as-Miniapp dogfooding architecture, bootstrap strategy and evidence.
18. Fly-native deployment topology: Studio Fly App/Machines, serverless-style autostop/autostart, `flyctl` and API responsibilities, Fly MCP boundary, Fly Managed Postgres/storage decision, CI/release and operational runbooks.
19. Per-generated-Miniapp URL and lifecycle design: user slug validation/reservation, Fly target strategy, URL retrieval/persistence, privacy/publication, collision/idempotency, upgrades, rollback and cleanup.
20. Durable workflow roadmap centered on DBOS, without overclaiming it in the demo.
21. Sequenced implementation phases, work breakdown, dependencies, owners/roles if relevant, estimates, milestones and cut lines.
22. Test strategy: unit, schema, fixture, contract, integration, security, failure injection, Fly provisioning/URL lifecycle, end-to-end and demo rehearsal.
23. Acceptance gates and objective pass/fail criteria, including self-dogfooding, Fly deployment and per-app URL.
24. Risk register with likelihood/impact, detection, mitigation and fallback.
25. Open decisions and assumptions that require owner approval; do not silently decide material product questions.
26. Source links, date checked, dependency/toolchain pins and evidence appendix.

Each task must identify: purpose, impacted packages/files (only after inspection), dependencies, implementation notes, test evidence, acceptance criteria, and estimated effort. Make dependencies explicit and avoid vague tasks such as “implement security” or “finish connector integration.”

## 12. Architecture and planning quality requirements

- Solve underlying problems, not just immediate symptoms. Require root-cause investigation, system-level failure-mode analysis, regression tests for invariants, and durable in-scope fixes; avoid superficial workarounds.
- Prefer the simplest mechanism that satisfies a stated requirement. For every new subsystem, explain why existing framework code or a simpler approach is insufficient.
- Preserve dependency direction: generated apps depend on Miniapp public APIs; the framework must not import individual generated apps; transports and UI do not own separate authorization/business logic.
- Never let the LLM be the security or correctness authority. The compiler, schema validator, capability registry, authorization layer, and runtime must enforce policy deterministically.
- Never let raw connector credentials enter prompts, specs, logs, Sprite filesystems, agent environment variables or generated app artifacts.
- Separate public connector discovery from organization access and Sprite-level authorization.
- Separate Sprite filesystem checkpoints from business workflow checkpoints and external effects.
- Treat Fly as the primary hosting/control-plane environment. Use Fly APIs and CLIs deliberately, not merely as deployment decoration.
- The Studio itself must use Miniapp; the first-party dogfooding path must share the same renderer, invocation and policy architecture as generated Miniapps.
- Every generated Miniapp must receive a unique stable Fly-issued URL derived from its user-provided slug; never fabricate a URL or treat URL secrecy as authentication.
- Do not call Fly Machines conventional function-as-a-service. Describe autostop/autostart as serverless-style request-driven Machine lifecycle and account for the existing-Machine-pool limitation and non-HTTP background work.
- Prefer Fly Managed Postgres (MPG v2) and Fly-native storage where code compatibility and cost permit; explicitly justify any continued Neon or external-state dependency.
- Do not promise reversibility for irreversible operations. Distinguish reversible, compensatable and irreversible effects.
- Do not claim “exactly once” for external effects. Define the real failure windows and what the provider/API adapter can prove.
- Treat provider responses, documents, emails, and LLM outputs as untrusted content; they cannot grant authority or override system policy.
- Include abuse cases: malicious instruction in a document, connector returns unexpected content, path/method escape, Custom API credential leakage, unauthorized Sprite, stale connection, spec tampering, malicious loops, oversize response, rate-limit exhaustion, duplicate submit, crash between provider success and local record, timeout with unknown external outcome, and restore after external write.
- Do not force all Hacksgiving partner products into the product. Use them only where available and useful. Be accurate about partner products: the current Fly ecosystem description presents TypeSafeAI Jev as a decision model (yes/no, choice, score with probability/confidence), not as the MiniappSpec schema validator. Deterministic schema validation belongs in our own code.
- Distinguish required, optional, and roadmap work. Do not hide risky features in “nice-to-have” lists while depending on them for acceptance.

## 13. Output discipline

- **Planning only.** Do not implement code or modify any source files.
- Do not invent repository paths, package names, existing functions, test results, dependency versions, or product capabilities.
- Clearly label every assumption and every unverified claim.
- If evidence conflicts across old plans and current code, current code/tests win; record the conflict and recommended resolution.
- If a Fly endpoint/schema is undocumented or ambiguous, report that uncertainty and propose a verification spike; do not reverse-engineer it by guesswork.
- Use current official Fly documentation and record when it was checked.
- The plan must be reviewable by a senior architect and actionable by a coding agent after approval.
- Finish with a concise list of decisions the product owner must make before implementation begins.

**Final deliverable:** one implementation-plan Markdown document at the agreed documentation path. No code changes, no dependency changes, no database changes, no deployments, no commits.
