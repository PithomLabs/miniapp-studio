# Coding-Agent Prompt: Produce the Miniapp Studio Implementation Plan (Planning Only)

You are acting as a principal software architect, staff Go engineer, and product-minded technical planner for **Miniapp Studio**. Your task is to inspect the actual repository and produce a rigorous, evidence-based implementation plan for our review.

**Do not implement the plan. Do not write application code, migrations, UI code, tests, CI changes, or deployment configuration.** Your only deliverable is the implementation-plan document described below, plus any diagrams placed inline in that document. You may run read-only inspection commands and existing tests/builds to establish repository truth, but do not change source files, dependencies, generated files, lockfiles, Fly resources, databases, or git history. The only file you may create or modify is the requested plan document.

## 1. Product goal — treat this as the governing brief

We are building **Miniapp Studio**, an AI-native application factory for non-technical business users. Users describe a business need in natural language and generate a Miniapp-based application from reusable data, UI, integration, workflow, and governance primitives. We are not building twelve individual vertical apps.

The product should follow the GigaCatalyst-style model: give target-market end users generic primitives, an accessible UI, and composable workflows so they can generate their own applications without programming. The generated apps are definitions assembled from a common platform—not separately hand-coded products.

The existing `github.com/PithomLabs/miniapp` Go framework predates Hacksgiving and serves as the foundation. The visible Hacksgiving product is the Studio UI, the app generator/spec compiler, the generic renderer/runner, the connector ecosystem explorer, Custom API connector extensibility, and one complete generated app.

The Hacksgiving brief requires a personal agent that runs on a Fly.io Sprite and uses at least one Sprites connector. The plan must make this compliance visible and testable. The agent—not merely a static web UI—must run on a real Sprite and make a real call through a Sprites connector.

### Core product promise

> Describe the business application you need. Miniapp Studio composes it from governed, reusable primitives, connects it to the systems you authorize, and shows you what it will do before it runs.

### Long-term differentiator and claims boundary

Durable business-workflow execution is the central long-term Miniapp differentiator. It should eventually include DBOS-backed long-running workflows, durable waits/signals, approvals that survive interruption, retries, cancellation, stable idempotency, external-effect reconciliation, UNKNOWN-outcome handling, compensation where possible, workflow versioning, and audit/replay tooling.

**Durable workflow authoring and production-grade multi-day workflow guarantees are roadmap, not Hacksgiving demo scope.** Do not turn the hackathon into a DBOS showcase. Do not claim exactly-once external effects, zero duplicate effects, always-undo, durable multi-day waits, or workflow replay unless exact behavior exists and is proven by current tests. A Sprite filesystem checkpoint is not a DBOS workflow checkpoint and does not undo external API effects.

## 2. Required source materials

Read and reconcile the following materials when they are accessible in the current coding-agent workspace:

- `miniapp-studio-consolidated-direction-v2.md` (the consolidated product/architecture direction from this conversation; if not present, follow this prompt as the source of truth).
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

For the Hacksgiving golden path, prefer one real authorized personal Sprite to minimize complexity. The Sprite-hosted personal agent must demonstrably run and use at least one Sprites connector. The trusted Studio control plane must own privileged organization-level actions, especially connector creation and Fly management credentials. Do not hand a broad organization token to the Sprite agent.

The plan must decide exactly which process runs in the Sprite, where the web Studio is hosted, where the generated app is rendered/executed, and how the agent reaches the Miniapp runtime. Draw a deployment diagram and trace the end-to-end trust boundaries. Do not hand-wave these boundaries.

For the longer-term product, keep execution semantics and compute placement as independent dimensions. Do not mandate one Sprite per generated template/app without evidence; placement should depend on trust, isolation, workload, lifecycle and cost.

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
- Studio UI that feels like an app factory, not just a CLI or prompt textbox.
- A public Fly connector ecosystem browser and honest organization/Sprite status indicators.
- Custom API as a first-class extension path; plan for one actual Custom API creation/test/use flow if feasible and supported by privileges/time.
- A plain-language request generates a declarative MiniappSpec.
- Deterministic validation shows accepted and rejected cases.
- A generic renderer shows a real working generated UI.
- A bounded, repeatable connector-backed data operation in that generated app.
- A clear README/demo separation between Hacksgiving functionality and post-Hacksgiving durable-workflow roadmap.

### Demo application

Plan one application end-to-end, preferably an Invoice Review Desk if the available connectors make that credible. It may use a native connector such as Box and a Custom API connected to a small public test service. Do not assume Box or any named connector is enabled; inspect the actual organization first and choose the lowest-risk real provider available. A second prompt variation is optional, not a delivery dependency.

Prefer a read-only connector-backed workflow. If adding a write, require an explicit confirmation and only claim the behavior tested end-to-end.

### Do not put into the critical path

- DBOS multi-day waits or a full durable workflow UI.
- Visual workflow canvas.
- Marketplace, billing, multi-user SaaS, white-label embedding, custom domains.
- Dozens of provider-specific business adapters.
- Arbitrary generated-code execution.
- Multi-Sprite orchestration.
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
17. Durable workflow roadmap centered on DBOS, without claiming it in the demo.
18. Sequenced implementation phases, work breakdown, dependencies, owners/roles if relevant, estimates, milestones and cut lines.
19. Test strategy: unit, schema, fixture, contract, integration, security, failure injection, end-to-end and demo rehearsal.
20. Acceptance gates and objective pass/fail criteria.
21. Risk register with likelihood/impact, detection, mitigation and fallback.
22. Open decisions and assumptions that require owner approval; do not silently decide material product questions.
23. Source links, date checked, dependency/toolchain pins and evidence appendix.

Each task must identify: purpose, impacted packages/files (only after inspection), dependencies, implementation notes, test evidence, acceptance criteria, and estimated effort. Make dependencies explicit and avoid vague tasks such as “implement security” or “finish connector integration.”

## 12. Architecture and planning quality requirements

- Solve underlying problems, not just immediate symptoms. Require root-cause investigation, system-level failure-mode analysis, regression tests for invariants, and durable in-scope fixes; avoid superficial workarounds.
- Prefer the simplest mechanism that satisfies a stated requirement. For every new subsystem, explain why existing framework code or a simpler approach is insufficient.
- Preserve dependency direction: generated apps depend on Miniapp public APIs; the framework must not import individual generated apps; transports and UI do not own separate authorization/business logic.
- Never let the LLM be the security or correctness authority. The compiler, schema validator, capability registry, authorization layer, and runtime must enforce policy deterministically.
- Never let raw connector credentials enter prompts, specs, logs, Sprite filesystems, agent environment variables or generated app artifacts.
- Separate public connector discovery from organization access and Sprite-level authorization.
- Separate Sprite filesystem checkpoints from business workflow checkpoints and external effects.
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
