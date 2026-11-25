# Miniapp Studio — Implementation Plan

**Status:** APPROVED as the governing implementation plan, subject to the approval conditions below; framework baseline/pin remains provisional until P0 evidence passes.
**Scope:** Implementation is authorized within this plan's stated scope and gates. P0 repository/module scaffolding is explicitly authorized. No implementation may rely on the Miniapp dependency pin until its hard validation gate passes. No scope expansion or governing-plan changes are implied.
**Date:** 2026-11-24.
**Revision:** plan5_review six resolutions incorporated 2026-11-24 (approval resumption, external lifecycle, gateway path, URL ruling, provider rule, auth contract); resubmitted for final approval.
**Approval basis:** `plan4_z(2).md` records approval with no foundational or structural blockers, authorizes P0 scaffolding, and preserves the conditions in §1.2 and §13.

## 1. Executive summary

Build Miniapp Studio as a new Go module (`github.com/PithomLabs/miniapp-studio`) that consumes the
pre-existing Miniapp framework as a **pinned dependency** (provisional pin
`73351a801fc54e19ee7254e6fe3ea7fbed859378`, finalization gated on the test evidence in §2.1).
Studio is a first-party Miniapp: every consequential operation goes through the framework's
canonical `runtime.Service` path (`Invoke`/`Status`/`DecideApproval`/audit) and static
`registry` registration. A Sprite-hosted builder agent turns natural language into a versioned,
content-hashed declarative MiniappSpec; a deterministic validator accepts or rejects it; one
statically registered generic runner interprets it; a server-rendered UI displays it. One
golden-path app (Invoice Review Desk) deploys to Fly with its own stable, slug-derived,
persisted Fly-issued URL and performs a real connector-backed read. Managed-agent providers
and DBOS durable workflows are roadmap: the MVP establishes architectural seams only.

Studio development is also a disciplined proving ground for Miniapp: when implementation
evidence shows that a required, general-purpose capability is missing or defective in the
framework, improve Miniapp in its own repository rather than hiding the deficiency in Studio.
The framework change process in §14 governs when such work is warranted and how it is tested,
reviewed, and pinned.

### 1.1 Product scope and non-goals

Target user: non-technical business owners/operators who need internal apps without
engineering staff. Thesis: describe the software → Studio generates a governed Miniapp from
reusable primitives → Fly hosts it at a stable URL the user owns. Non-goals for the MVP:
twelve vertical apps, workflow canvas, marketplace, billing, multi-tenant admin,
white-label embed, custom domains, DBOS authoring UI, generalized undo.

### 1.2 Consolidated decisions (conflicts resolved)

- `prompt.md` stays frozen: `prompt2.md` §5 asks to amend the agent prompt, but
  `plan2_review.md` §4 forbids modifying governing plans — the Agent Provider requirement
  flows into this document (§12.1) instead.
- MVP runner is Class A (§4.2), not a DBOS subset: resolves the review's revision-A ambiguity.
- Topology and MPG/Neon are spike-decided with decision records, never assumed (§5).
- Managed-agent integrations are roadmap with MVP seams only (§12.1).
- The approval record `plan4_z(2).md` approves this plan's architecture and scope, subject to its
  hard P0 pin gate, spike-based decisions, claims discipline, and explicit cut line. It also
  authorizes P0 repository/module scaffolding. This does not authorize scope expansion or
  implementation of DBOS-grade durability, managed-agent adapters, or generalized undo.
- Miniapp and Studio may evolve together where evidence warrants a framework-level correction;
  that work is governed by §14 and does not turn the framework into a Studio-specific fork.
- Approval resumption uses the verified same-key re-`Invoke` → `pollApproval` path (§4.2); no
  redundant two-invocation mechanism; `Status` is observation-only for approvals.
- External operations use a durable intent/attempt/outcome/reconcile lifecycle outside the
  Class A transaction (§4.5); bounded reliability machinery, not DBOS.
- Agent gateway calls transit `runtime.Service` and the trusted adapter (§6.3); G1 and G8 test
  one end-to-end path.
- Shared-router URLs are non-compliant with the Fly-issued URL requirement as written (§5).
- Provider selection lives in generation/session config, never in MiniappSpec (§12.1).

## 2. Repository inspection and evidence inventory

### 2.1 Framework baseline and dependency pin (provisional; hard P0 gate)

- Inspected baseline: `73351a801fc54e19ee7254e6fe3ea7fbed859378` (`prompt31`), branch `main`.
  `origin/main` resolves to the same SHA; no newer commits existed to survey at the recorded
  inspection date (2026-11-24). Treat this as the **baseline candidate**, not an irrevocable pin.
- The approval record makes the pin gate hard: no Studio implementation may rely on this revision
  until the complete validation suite and the Studio import spike pass.
- P0 may create the Studio repository/module skeleton and its CI scaffolding under the explicit
  authorization recorded in `plan4_z(2).md`. Do not treat scaffolding or a `go.mod` reference as
  proof that the framework is compatible; avoid building Studio features on the provisional
  dependency before the gate passes.
- If P0 or a later Studio phase exposes a Miniapp defect or a necessary framework-level gap, apply
  the §14 classification and change-control process. A warranted Miniapp correction may produce a
  **new final framework SHA**. Studio must pin the exact resulting immutable commit, not float on
  `main` and not retain `73351a8` merely to avoid revisiting the pin.
- Uncommitted state identified at baseline (left untouched by this planning document):
  `docs/architecture/` (Prompt32 proposed package), `docs/prompt32*.md`,
  `docs/review_architecture.md`. These are not implementation evidence. Prompt32.1 remains an
  architecture-documentation work order until its own reconciliation and F-0 approval process
  authorizes foundational implementation.
- Baseline build evidence: `go build ./...` succeeds (`BUILD_OK`); toolchain go1.27.3 against
  module `go 1.26.0`. This is not a substitute for the full gate.
- **Finalization gate (hard, before implementation relies on the pin):** run the CI suite from
  `.github/workflows/ci.yml` against the selected framework SHA on PG19 — `gofmt -l` clean,
  `go vet ./...`, `go test -p 1 ./...`, `TestSleepAcrossRestart` + `TestProbeRestart` must execute
  (not skip), `go test -race ./contract/ ./internal/... ./spikes/` — plus a Studio import spike
  (new module imports the selected framework revision and runs one Class A invoke + one approval
  park/decide round-trip including same-key re-`Invoke` resumption per §4.2). Record commands,
  SHAs, and results. If Miniapp changes after this gate,
  rerun the relevant Miniapp suite and Studio integration tests against the new exact SHA before
  merging or releasing Studio work that depends on the change.

### 2.2 Capability matrix (all paths repo-rooted at the miniapp framework)

| Capability | Status | Evidence |
|---|---|---|
| Manifests, policy, scope, tasks | Implemented | `contract/manifest.go` (`MiniappManifest`, `ToolManifest`, `Class`, `Placement`), `contract/policy.go`, `contract/run.go` |
| Static registration | Implemented | `registry/registry.go` (`Register`, `Resolve`, `Range`) |
| Canonical invocation + idempotent replay | Implemented | `runtime/service.go` (`Invoke`, `InvokeRequest/Result`), `runtime/store.go`, `runtime/validate.go` |
| Run lifecycle + approvals decide/settle | Implemented | `runtime/lifecycle.go` (`Status`, `Signal`, `Cancel`, `DecideApproval`, `SettleEffect`) |
| Class A sync executor (single tx) | Implemented | `executor/classa/classa.go`, `executor/executor.go` (`SyncExecutor`, claim hooks) |
| Class D DBOS executor | Implemented (not MVP path) | `executor/dbos/` (`Submit`/`Status`/`Signal`/`Cancel`, per-family apps) |
| Durable approvals w/ intent hash + expiry | Implemented | `approvals/approvals.go` (`Submit`, `Decide`, `CancelPending`, `Get`, `ByInvocation`) |
| Effect ledger PENDING→…→CONFIRMED/FAILED/UNKNOWN | Implemented | `effects/`, `contract/run.go` (`EffectState`) |
| REST verbs | Implemented | `transport/rest/server.go` (`invoke/status/signal/cancel/decide/healthz`) |
| MCP + CLI transports | Implemented | `transport/mcp/server.go`, `transport/cli/` |
| Postgres store + migrations 000001–000006 | Implemented | `internal/store/`, `migrations/`, PG19 compose |
| Reference miniapp pattern | Implemented | `miniapps/crm-read/` (manifest + JSON schemas + digest-pinned profile) |
| Determinism analyzer, testkit, preflight | Implemented | `analyzer/determinism/`, `testkit/`, `preflight/` |
| Deploy seam | Interfaces only | `deploy/targets.go` (`SpriteTasks`, `SpriteWorkspaces`, `MachineTarget`, `Promotion`) |
| Sprite/Machine/token/capability code | **Absent** | `fly/`, `tokens/`, `capability/clients/`, `executor/river/` are empty dirs |
| `sprites-go` usage | **Absent** | v0.2.2 in `go.mod` only; zero non-test references |
| MiniappSpec / generator / validator / UI | **Absent** | No matches for spec/DSL/HTML symbols in Go sources |

### 2.3 Dependency pins (recorded, not yet consumed)

- `github.com/PithomLabs/miniapp` @ provisional `73351a8` (see §2.1 gate).
- `github.com/superfly/sprites-go`: module-cache v0.2.2 inspected; local checkout
  `/home/chaschel/Documents/go/sprites-go` is source-identical plus newer `go.mod`
  (`client-signals/go` v0.5.0, `x/term` v0.47.0). SDK covers Sprite CRUD, URL settings,
  services, exec/sessions, filesystem, checkpoints, network policy, proxying; it has **no**
  connector/gateway APIs. Pin decision (released v0.2.2 vs checkout pseudo-version) is open (§13).
- Sprites API contracts verified 2026-11-24 at API version 0.1.12 (see §6.2).

## 3. Target architecture and component responsibility matrix

```text
Browser/phone
    |
    v
STUDIO WEB (new, server-rendered) ── Fly App/Machines, autostop/autostart
    |  screens: Home, Create App, Connectors, Validate & Deploy, My Apps, Activity
    |  calls runtime.Service only; owns no authz/business logic
    v
RUNTIME KERNEL (framework, pinned) ── Invoke/Status/DecideApproval/audit
    |  Class A tools: spec.validate, spec.render, connector.* (control plane),
    |  studio.* (app registry, slug reserve, deploy, url), runner.invoke
    v
GENERIC RUNNER (new, one static registration) ── interprets MiniappSpec as data
    |  connector reads via capability adapters -> Sprites gateway (no raw secrets)
    v
FLY (infrastructure) ── Sprites (builder agent, generated-app target),
    Machines (Studio), connector gateway, MPG-or-Neon Postgres
```

| Component | Owner | Responsibility | Must never |
|---|---|---|---|
| Studio web/UI | New (studio repo) | Render screens, collect input, call `runtime.Service` | Decide authz, touch provider secrets, shell out user text |
| Studio tools | New, registered miniapps | App registry, spec versions, slug/URL lifecycle, deploy orchestration | Bypass `Invoke`/audit path |
| Generic runner | New, one registration | Validate + interpret MiniappSpec, fixture/dry-run, bounded connector reads | Execute arbitrary code, call undeclared ops |
| Builder agent | New, Sprite-hosted process | NL → MiniappSpec drafts, validation-feedback loop | Hold Fly-org token or provider secrets |
| Framework | Miniapp repo, consumed as exact pinned dep | General-purpose invocation, auth, approvals, effects, audit, persistence, execution and observability capabilities; may be improved when §14 warrants | Import Studio code, absorb Studio-only product logic, or expose unsupported internals as a substitute for public contracts |
| Fly adapters | New, behind owned interfaces | Sprite/Machine/connector lifecycle, typed REST client | Fabricate URLs, log secrets |

## 4. MiniappSpec and interpreter contracts (revision A: execution semantics)

### 4.1 Canonical spec

One versioned `MiniappSpec` (JSON Schema, versioned, content-hashed with SHA-256 hex digest).
Sections: `app` (id, slug, name, versions), `data` (entities, queries, bindings), `ui`
(pages, components, bindings, actions), `workflows` (trigger + typed steps), `capabilities`
(logical op IDs + connection bindings), `policies` (roles, effect classes, confirmation rules,
limits), `fixtures` (inputs + expected outputs), `presentation` (tokens), `metadata`
(schema version, content hash, provenance — no secrets, no provider references per §12.1). Workflow is a section, not a language.
Editing creates a new immutable version; a run binds the version it started with.

### 4.2 MVP execution contract (no half durable engine)

- **Class:** the MVP runner is **Class A only** (`executor.SyncExecutor.Execute`): synchronous,
  in-request, single tenant transaction (claim → entity lock → liveness recheck → handler in
  savepoint → terminal completion commit, per `executor/classa/classa.go`).
- **Sync vs async:** all MVP runner operations are synchronous request/response. No background
  jobs, no River (package absent), no DBOS workflows in the MVP path.
- **Approvals (verified resumption contract):** `Invoke` on an approval-gated tool parks via
  `runtime/service.go:requestApproval`/`parkApproval` (persisted by `approvals.Service.Submit`
  with intent hash + expiry) and returns `waiting_approval` + `approvalID`. The decision is
  recorded out-of-band via `DecideApproval` (`approvals.Service.Decide`, intent-hash bound,
  expiry-checked). Execution resumes when the client re-submits `Invoke` with the **same
  idempotency key**: `claim` finds the existing record → `replay` (`RunWaitingApproval`) →
  `pollApproval` (`service.go:370`) → decided+approve executes the original operation
  (`executeSync` for Class A, `grantedRun` for Class D); denied/expired/cancelled settles
  terminally via `settleDenied` with no execution; still-pending returns `waiting_approval`
  again. `Status` (`lifecycle.go:27`) is observation-only — it never advances an approval.
  Same-key resubmission never creates a second invocation; terminal outcomes replay verbatim.
  No long-poll (incompatible with scale-to-zero) and no DBOS wait/signal wake. No redundant
  two-invocation mechanism: the framework path above is the contract, proven by the G4 tests.
- **Crash/restart:** a crash mid-attempt rolls the transaction back; nothing partial commits.
  Same-key resubmission converges via the runtime replay path (`service.go:replay`, C6 rule);
  duplicate submissions attach, never double-execute.
- **Uncertain external effects:** ledger states `PENDING/ATTEMPTING/CONFIRMED/FAILED/UNKNOWN`
  (`contract/run.go`); reconcile-by-read where the adapter supports it, otherwise quarantine +
  operator `SettleEffect`. Blind retry of UNKNOWN is forbidden.
- **Boundary:** persisted application state (specs, slugs, URLs, approvals, ledger rows in
  Postgres) is **not** durable workflow recovery. Claiming auto-resumption, multi-day waits,
  replay debugging, or exactly-once external effects in the MVP is a plan violation.

### 4.3 Validator and dry-run

Deterministic, LLM-independent: schema/version check, known component/op/endpoint references,
method/path allowlists, permission consistency, expression bounds, required fixtures.
Unknown → reject with actionable errors. Dry-run executes fixtures only, never external writes.
At least one deliberate rejection is demoed and tested.

### 4.4 Primitive catalog (MVP subset vs roadmap)

MVP: data (entities, fields, filters, sort, pagination, aggregates, bindings); UI (shell,
dashboard, metric card, table, detail, form, action button, basic chart, status/empty/error,
permissions card, run result); workflow (manual trigger, connector read, transform, branch,
bounded loop, notify, finish, explicit confirmation); integrations (§6); governance (declared
ops, method/path constraints, effect classes, confirmation, limits, stop, redaction, audit
where implemented); evidence (spec inspector, validation/fixture reports, versions, run
results). Roadmap: schedules, webhooks, durable waits/signals, approval parking/wake,
retries, cancellation, sagas, UNKNOWN inbox, replay debugger, theming, marketplace.

### 4.5 Durable external-operation lifecycle (outside the Class A transaction)

A PostgreSQL transaction cannot span Fly or provider APIs, so every Studio operation with an
external side effect (Sprite/Machine provisioning, connector writes where proven, URL
registration) follows this bounded lifecycle, independent of the §4.2 request transaction.
This is reliability machinery, not a workflow engine: no DBOS, no waits, no sagas.

1. **Intent (committed first):** persist an operation record with a stable idempotency/
   resource identity, target, parameters hash, and state `PROVISIONING` before any external call.
2. **Attempt (committed separately):** execute the external call; commit an attempt record with
   request/response refs (redacted), provider-side IDs, and outcome.
3. **Outcome:** on success commit `READY` with authoritative IDs/URL; on definitive failure
   commit `FAILED` with reason; on timeout/ambiguous response commit `UNKNOWN` — never blind-retry.
4. **Recovery:** on restart or `UNKNOWN`, reconcile by reading the provider (existence + identity
   match) before deciding; adopt confirmed-remote resources, fail or retry only when the
   provider proves the prior attempt did not land.
5. **Reconciliation sweep:** a bounded, idempotent sweeper adopts or retires externally created
   resources missing local commits (e.g., Sprite created, crash before URL persisted) and
   reports orphans it cannot attribute; a sweep never deletes provider state without an
   explicit, approved, audited action.
6. **Ledger binding:** each external effect is recorded in the Miniapp effect ledger
   (`PENDING/ATTEMPTING/CONFIRMED/FAILED/UNKNOWN`) with the same stable identity; operator
   `SettleEffect` remains the human path for unresolvable outcomes.

MVP connector operations stay read-only unless a write completes this lifecycle with crash
and ambiguous-response tests. Explicit approval alone does not satisfy this section.

## 5. Fly deployment and URL topology (revision B)

Three deployment concerns, decided independently: (1) Studio web/API on Fly App/Machines with
`fly.toml`, health checks, secrets via `flyctl`, autostop/autostart evaluated for
serverless-style scale-to-zero (Fly Proxy starts/stops existing Machines; it never
provisions/destroys the pool — cold start, min warm capacity, and session/state consequences
must be measured, and no background worker may be assumed to wake a stopped HTTP Machine);
(2) the personal builder agent as a real process on its designated Sprite using a real
connector; (3) each generated app with a stable, user-derived, persisted Fly-issued URL
(private/authenticated by default; public publish needs explicit consent + app-level auth).

### URL topology comparison (spike before selection; cheapest ≠ correct)

| Option | Isolation | URL ownership | Provision/latency | Cost/limits | Notes |
|---|---|---|---|---|---|
| (a) Sprite per generated app | Strong (own microVM) | Independent Sprite HTTPS URL | Sprite create + service start | Per-Sprite cost/limits | Preferred to evaluate first; wake-on-request to verify behaviorally (no SDK wake call exists) |
| (b) Shared Fly App/router | Weak (shared process) | Slug path/route on one host | Fastest | Cheapest | **NON-COMPLIANT** with the stable persisted Fly-issued URL requirement as written — a route is not an independently provisioned Fly URL. Admissible only if the requirement is explicitly amended. Spikes compare (a)/(c) as compliant candidates |
| (c) Fly App per generated app | Strong (own app) | Per-app Fly hostname | App create + deploy | Highest ops/cost | Heaviest; justified only if (a) fails limits |

SDK-verified operations available behind owned interfaces: `CreateSprite[WithOrg]`,
`GetSprite` (returns authoritative `URL` + `URLSettings`), `DeleteSprite`,
`UpdateSprite`/`UpdateURLSettings` (auth mode, labels), services create/start/stop/delete
with `HTTPPort`, exec/sessions, filesystem, checkpoints, network policy. Invariants for the
winner: slug normalize → reserve → collision-safe resource name; URL persisted only from
authoritative API/CLI responses (never concatenated); updates preserve the URL; failed
deploys leave last-known-good or report downtime; orphan Fly resources reconciled on
DB-commit failure or unknown outcomes.

### Persistence

MPG (Fly Managed Postgres) is the default candidate; Neon is the fallback. Decide by
compatibility spike: pgx pool modes, DBOS direct-connection requirement, migrations
000001–000006, region/price. DB constraints must not block the MVP. No authoritative state on
ephemeral Machine/Sprite filesystems; blobs via Tigris/S3-compatible if needed.

## 6. Connector integration architecture (revision D)

### 6.1 Three states, never collapsed

Public ecosystem catalog (metadata browsed from Fly's directory) vs organization-configured
connections (live `GET /v1/oauth/connections`) vs per-Sprite authorization (gateway
visibility + access policy in Sprite context). UI vocabulary: Listed / Not configured /
Configured / Allowed / Denied / Needs scope / Test passed / Test failed / Unknown.
Integrations, plugins, MCP, and SDKs render in separate sections — never as connectors.

### 6.2 Verified Connectors API contract (checked 2026-11-24, Sprites API 0.1.12)

- `GET /v1/oauth/connections[?provider=]` — org-configured connections only (bearer auth).
- `POST /v1/oauth/connections/custom_api` — create; **requires full access** (403 otherwise);
  required `name`, `base_api_url`, `access_token`; `name` is the org identity (same-name
  create **updates in place** — UI must warn/confirm); create does **not** test; token stored
  encrypted, never returned; empty/missing `access_policy` denies Sprite use.
- `POST /v1/oauth/connections/custom_api/test` — one request, stores nothing; `connection_id`
  tests a stored connector; stored credential is sent only to stored origins; outcome is in the
  200 body (`status: error` on credential failure, not an HTTP error); requires full access.
- `auth_method`: `header` (default, `auth_header_prefix`, default `Bearer`) | `custom_header`
  (default `X-Api-Key`, hop-by-hop/identity headers rejected) | `query_param` (default
  `api_key`) | `url_path` (`auth_url_path_template`, `{token}` substitution; `test_url` is
  relative for this method, absolute otherwise).
- `AccessPolicy`: `allow_all` (never default-on) | `allowed_endpoints` (exact + trailing-`*`
  wildcards) | `blocked_endpoints` (case-insensitive, **checked before allow rules**) |
  `name_prefix` | `sprite_labels` (all required). Database connectors reject endpoint rules.
- Gateway call form: `https://api.sprites.dev/v1/gateway/{provider}/{conn_id}/{path}`.
- Remaining to verify at spike time against a real Sprite: gateway discovery endpoint
  semantics, update endpoint, error bodies — docs alone are insufficient for Sprite-context
  behavior.

### 6.3 Control-plane boundary and governed gateway path

Agent connector calls transit the framework — a direct Sprite→gateway call would bypass
`Invoke`/audit and violate G8. The specified end-to-end path (G1 and G8 test this same path):

1. The Sprite-hosted agent requests a connector operation through a narrowly scoped
   capability tool exposed by the Studio backend (agent identity + declared operation only).
2. The request enters `runtime.Service.Invoke`; the framework authenticates the caller,
   validates the declared operation against the registered descriptor and policy, and
   authorizes (approval-gated where declared).
3. The trusted adapter executes the gateway request using Fly's credential handling; the
   credential is attached by the gateway, never by the agent.
4. The result is recorded against the invocation (ledger + audit) and returned to the agent.

Full-access connector operations run only in the trusted Studio backend after an authorized
user action. Raw tokens and Fly-org credentials never enter the Sprite, prompts, specs,
logs, timelines, or browser responses. Wizard order: Define → Test (green) → Scope/Grant
(default-deny, Sprite name-prefix/labels + endpoint rules). Miniapp registers one operation
descriptor per allowed call (logical ID, method, path, schemas, effect class, approval,
limits, timeout); the runtime rejects anything undeclared. A dedicated negative test proves
an out-of-scope path is denied and receipted, and a trace test proves every gateway call in
the golden path carries its invocation/audit record.

## 7. UI, end-user workflows, golden path (revisions C + E)

### 7.1 Screen acceptance criteria (MVP must look like commercial software)

| Screen | Required outcome |
|---|---|
| Home | Professional dashboard: app list w/ status, recent activity, entry points |
| Create App | Conversational builder + live preview + spec inspector side by side |
| Connectors | Ecosystem explorer (3-state statuses) + Custom API wizard (Define→Test→Scope) |
| Validate & Deploy | Validation report w/ actionable errors, deploy progress, failure recovery |
| My Apps | Apps, URLs, lifecycle state, update/retire; URL opens correct app version |
| Activity | Deployment + execution evidence, redacted logs, error diagnostics |

Every screen: responsive, accessible controls, and complete loading/empty/error/success
states. No SPA framework without repository evidence — server-rendered UI calling
`runtime.Service` is the default.

### 7.2 Golden-path acceptance test (revision C)

Automated end-to-end, repeatable from a clean state: (1) NL description → (2) valid
MiniappSpec generated → (3) invalid spec deliberately rejected → (4) preview rendered →
(5) real Sprites connector attached → (6) deployed to Fly → (7) Fly-issued URL received and
persisted → (8) URL opened independently and app used → (9) spec edited and redeployed →
(10) URL stable, version history recorded, apps independent, failed deploys non-corrupting.
Demo app: Invoice Review Desk (Box-preferred native connector per plan2_review.md §3, Custom API
for the reference/PO record; read-only core; any write needs explicit confirmation).
The G4 resumption contract (approve → same-key re-`Invoke` → execute; deny/expire → terminal)
is exercised in e2e against the real kernel, and every golden-path gateway call is asserted
to carry its invocation/audit record per §6.3.

## 8. Security and failure analysis

- **Trust boundaries:** browser ↔ Studio backend (session auth) ↔ framework kernel
  (`Invoke`/auth/audit) ↔ Fly control plane (org token, backend-only) ↔ Sprite agent, which
  reaches the gateway only via §6.3 (capability tool → `Service.Invoke` → trusted adapter;
  agent holds no secrets, gateway attaches credentials) ↔ providers.
- **Abuse cases (each needs a test or a documented spike):** prompt-injected instruction in a
  document/connector response; undeclared method/path/operation; Custom API credential leak
  (assert absence from Sprite FS/env, spec, logs, artifacts); unauthorized Sprite calling a
  scoped connector; stale connection/token expiry; spec tampering (hash mismatch → refuse);
  unbounded loops/oversize responses (validator bounds + runtime caps); rate-limit exhaustion;
  duplicate submit (same-key attach); crash between provider success and local record
  (UNKNOWN + reconcile, never blind retry); timeout-unknown outcomes; restore-after-external-write
  (checkpoint ≠ business undo — label exactly what restore does/doesn't).
- **Effect classes:** every external op declares reversible/compensatable/irreversible;
  irreversible ops need explicit confirmation and can never be "undone" by checkpoint restore.
- **Dogfooding audit rule:** any Studio code path performing a consequential operation outside
  `Invoke`/audit is a defect, caught by code review + an allowlist test over the service surface.

### 8.1 Minimum auth and ownership contract (before UI exposure)

The framework's credential bindings/grants files are demo scaffolding (`cmd/miniapp/main.go`
says production auth arrives later); the MVP therefore defines this minimum contract, all
enforced server-side, with external IdP deferred to roadmap:

- **Identity:** one operator identity per workspace for the MVP (single-user; multi-user
  accounts are cut). The operator authenticates to Studio; Studio maps the session to exactly
  one framework credential binding. No anonymous or shared-default credentials in any
  deployed environment.
- **Session security:** opaque session handles with idle + absolute expiry, secure transport
  only, no credential values in URLs/logs/storage; logout revokes server-side.
- **App ownership:** every app record carries owner workspace + creating identity; all
  Studio tools scope reads/writes by that ownership; one app cannot read or mutate another
  app's definition, data, or Fly resource bindings.
- **Tenant isolation:** Studio workspace maps 1:1 to a Miniapp tenant; all kernel calls carry
  that tenant context; cross-tenant access attempts are denied and audited.
- **Agent identity:** the Sprite builder agent authenticates as a distinct, least-privilege
  principal that may only invoke its declared capability tools; it cannot approve its own
  actions, mint credentials, or reach the Fly control plane.

## 9. Sequenced implementation tasks and dependency graph

```text
P0 pin gate (§2.1) ──┬──> P1 spec+validator ──> P2 runner+renderer ──> P5 golden path
                     ├──> P3 Fly adapters ──> P4 Studio UI ─────────┘
                     └──> (all) ──> P6 rehearsal + claims audit
```

| Phase | Tasks (each: purpose, files, tests, exit) | Depends |
|---|---|---|
| P0 | Authorized repo/module scaffolding + CI (gofmt/vet/test/golangci); verify baseline; run the hard framework pin gate; classify any necessary Miniapp corrections under §14; record and pin the exact passing framework SHA | Approval |
| P1 | MiniappSpec JSON Schema + version/hash policy; deterministic validator + error taxonomy; fixtures | P0 |
| P2 | Generic runner registration + interpreter (Class A); server renderer; dry-run; deliberate-rejection test | P1 |
| P3 | Owned Fly interfaces; `sprites-go` adapter; typed Connectors REST client + fakes/contract tests; topology spikes → decision record; MPG/Neon spike → decision record; slug/URL lifecycle | P0 |
| P4 | Six screens per §7.1; ecosystem explorer; Custom API wizard; permissions card; run/evidence views; all via `runtime.Service` | P1, P3 |
| P5 | Builder-agent loop (draft → validate → fix); Invoice Review Desk; §7.2 e2e; negative authz test; secret-absence tests | P2, P3, P4 |
| P6 | Clean-state rehearsal, README claims audit vs passing tests, organizer-eligibility filing, submit | P5 |

Cut line (Hacksgiving): one complete app; no workflow canvas, marketplace, billing,
multi-tenant admin, white-label, custom domains, DBOS UI, undo-of-world. Second app optional.

### 9.1 Cross-cutting Miniapp framework improvement lane

Every implementation phase must record framework defects or capability gaps discovered while
meeting its acceptance criteria. A finding is not automatically a framework change: classify it
using §14, identify the smallest correct owner, and implement only work necessary for the approved
Studio scope or an explicitly approved foundational milestone. Framework changes are made in the
Miniapp repository with their own tests and evidence; Studio changes remain in the Studio
repository. The Studio dependency is updated only to an exact immutable Miniapp commit that has
passed the relevant framework and Studio integration checks. This lane does not relax P0, G0,
C1–C7, Prompt32.1, or any other existing activation/approval gate.

## 10. Acceptance tests and exit gates (G0–G12 mapped)

| Gate | Pass/fail evidence |
|---|---|
| G0 repo/dependency truth | Final Miniapp SHA (including any warranted framework corrections) + CI results recorded; Studio import spike green; SDK/API pins recorded; framework-change records and blockers listed |
| G1 Sprite + connector | Agent process on real Sprite makes a real gateway call **via the §6.3 governed path** (same end-to-end path G8 tests); invocation/audit trace present; secret-absence assertions pass |
| G2 ecosystem truth | 3-state statuses render from distinct sources; SDKs/integrations not labeled connectors |
| G3 Custom API | Wizard create→test→scope passes; no secret in spec/Sprite/logs; default-deny verified |
| G4 spec generation | Valid spec accepted; invalid component/op/path rejected deterministically; hash/version tested; approval resumption proven (approve → same-key re-`Invoke` → executes; deny/expire → terminal with no execution; duplicates attach safely) |
| G5 UI generation | UI renders from spec; bindings resolve; empty/error states visible |
| G6 app execution | Bounded connector-backed path runs repeatably; no unapproved writes; external ops follow §4.5 (intent→attempt→outcome, crash/ambiguous cases reconciled, never blind-retried) |
| G7 demo readiness | Clean-state rehearsal passes; claims == passing tests; limitations documented |
| G8 dogfooding | All consequential ops via `Invoke`/audit; gateway calls transit the §6.3 path (shared with G1); bypass-allowlist test green; gaps explicit |
| G9 Fly deployment | Studio reachable on Fly; reproducible `flyctl` deploy; autostop behavior tested or warm-capacity justified; no ephemeral-FS state; provisioning reconciled per §4.5 (crash-before-commit leaves no orphan or a reported one) |
| G10 per-app URL | Slug → real persisted unique URL; opens correct app; collision/privacy/lifecycle tested; shared-route designs excluded unless the requirement is amended (§5 ruling) |
| G11 control-plane integration | Least-privilege adapters; full-access ops backend-only; secret ownership; §4.5 lifecycle ops exercised in golden path |
| G12 MVP polish | First-run → deploy → URL → recovery rehearsal passes repeatably |

### 10.1 Test strategy

Unit (validator, slug/URL, policy checks) → schema/fixture (spec corpus incl. adversarial
rejections) → contract (Fly/Sprites endpoints, fakes in CI, live in authorized env) →
integration (runner + kernel + Postgres) → security (§8 abuse cases, secret-absence) →
failure injection (crash mid-attempt, duplicate submit, UNKNOWN outcomes, restore-after-write)
→ e2e golden path (§7.2) → demo rehearsal (clean-state, claims audit).

## 11. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Pin invalidation or framework gaps | P0 gate blocks dependent feature work; treat `73351a8` as a baseline candidate, make only warranted Miniapp corrections under §14, then pin and test the exact resulting SHA |
| Scope vs Nov 29 deadline | §9 cut line; one golden path; roadmap honestly labeled |
| Prompt-wrapper perception | Visible spec, validator, permissions, rendered app (§7.2 steps 2–4) |
| Org lacks demo connector | Box-preferred, Custom API fallback; check live catalog first |
| Custom API becomes arbitrary proxy | Operation descriptors + runtime deny-unknown + negative test (§6.3) |
| Secret leakage | Backend-only privileged calls; redaction + absence tests (§8) |
| Framework-on-Sprite portability | P3 import spike; Studio stays on Machines, Sprite holds agent only |
| Misleading durability/undo claims | §4.2 boundary; §8 abuse cases; claims audit in P6 |
| Pre-existing-foundation eligibility | Pinned dep + disclosure + organizer filing (P6); no copied code |
| Fly topology/cost surprise | Spikes before selection; decision records required (§5) |
| Framework scope creep | §14 ownership classification; only evidence-backed changes necessary for approved gates; preserve P0–P6 cut line |

The approval record identifies schedule compression as the dominant delivery risk. Keep the
one-app cut line; if time forces a trade-off, prioritize a truthful, repeatable G7 demo rehearsal
over optional G12 polish. Do not add scope after November 25 without an explicit owner exception.

## 12. Roadmap and deferred capabilities

### 12.1 Agent Provider + Execution Environment abstraction (prompt2.md)

Adopt the four-dimension framing: app definition / agent provider / execution environment /
business execution. Design (post-hackathon implementation, MVP seams only):

- Provider-neutral Go interfaces: discovery/registration, account+auth references,
  agent/model selection, session create/terminate, execution-target selection, tool-call and
  worker lifecycle, status/events, callbacks/webhooks, logs/usage, error taxonomy, secret
  isolation, capability discovery + compatibility validation.
- Three distinct categories (never conflated): managed providers (Gemini/Claude Managed
  Agents on Sprites) | self-hosted runtimes (Gemini API CLI on Sprite/Machine) |
  coding-agent integrations (Claude Code, Codex, Cursor, MCP clients).
- MVP seams: provider selection lives in generation/session configuration only — the
  MiniappSpec carries zero provider references (not in `metadata`, `policies`, or anywhere
  else). Execution-target enum (sprite/machine) and adapter interface declarations complete
  the seam. No managed-agent adapter implemented in the MVP; a provider selector alone
  never counts as an integration. Test: the same generation inputs under two different
  provider selections produce byte-identical specs.
- Vendor-neutrality invariant: changing providers never rebuilds the generated app.
- Evidence required at roadmap time: current managed-agent integration docs, `gemini-api-cli`
  self-hosted Fly provider repo, per-provider threat model, contract tests, cost/ops trade-offs.

### 12.2 Durable workflows (flagship, post-hackathon)

DBOS-backed waits/signals, approval wake, retries, idempotency, UNKNOWN inbox, sagas,
versioning, replay — only behind crash/failure tests. The MVP §4.2 contract must not be
extended ad hoc; migration to a durable adapter is an explicit roadmap phase.

## 13. Unresolved owner decisions

1. Final Miniapp pin: `73351a8` is the inspected baseline candidate; freeze the exact immutable SHA that passes §2.1, including any separately justified and tested Miniapp corrections under §14. Do not assume the original baseline must remain the final pin.
2. `sprites-go` pin: released v0.2.2 vs pseudo-version of the newer local checkout.
3. Authorized Fly environment: when/where Fly spikes (`flyctl`, org token, Sprite runs) may execute.
4. URL topology + MPG/Neon: decided by P3 spikes; owner ratifies decision records.
5. Demo connector anchor if Box is unavailable in the org (re-check live catalog day one).
6. Repo init + module scaffolding: explicitly authorized for P0 by the approval record `plan4_z(2).md`; authorization does not waive the hard pin gate or authorize later scope expansion.

## 14. Joint evolution of Miniapp and Miniapp Studio

### 14.1 Governing principle

Miniapp Studio is built on Miniapp and is also a real-world proving ground for it. Development
therefore has two related outcomes: a useful Studio product and a stronger general-purpose Miniapp
framework. When implementation evidence demonstrates that a required capability belongs in Miniapp
and that the framework is defective or incomplete, improve Miniapp in its own repository rather
than compensating with a Studio-only workaround.

The governing rule is: **improve the framework where the underlying capability belongs there, but
do not turn Studio development into an unrestricted framework redesign.** Miniapp remains
independent of Studio; the dependency direction is Miniapp Studio → Miniapp, never the reverse.

### 14.2 Ownership test and classification

For every issue discovered during implementation, answer: would another Miniapp application
reasonably need this capability, and is the deficiency in the framework's general execution,
security, authorization, persistence, effects, observability, deployment abstraction, or developer
experience contract?

| Classification | Correct owner and treatment |
|---|---|
| Defect in an existing Miniapp contract or implementation | Fix in the Miniapp repository when necessary for Studio correctness; add a focused regression test proving the invariant; preserve established behavior and contracts |
| Missing general-purpose framework capability required by an approved Studio gate | Design the smallest reusable Miniapp API/mechanism; check it against the foundational contracts and invariants; obtain any required architectural approval before implementation; test it in Miniapp and through Studio |
| Studio product behavior, workflow, UI, app-generation policy, or Studio-specific orchestration | Implement in the Studio repository using supported Miniapp interfaces; do not generalize prematurely |
| Useful but nonessential framework improvement | Record in the Miniapp roadmap/backlog with evidence and defer; do not expand the Hacksgiving cut line |
| Change that alters a foundational guarantee, execution-class semantics, authorization model, lock ordering, recovery semantics, or a frozen governing contract | Stop and raise an explicit architecture/owner decision; do not silently implement under this plan |

### 14.3 Required framework change protocol

Every Miniapp change motivated by Studio must have a change record containing:

1. **Observed need:** the Studio acceptance criterion, reproducible failure, or concrete integration evidence that exposed the gap.
2. **Ownership rationale:** why the capability belongs in the general-purpose framework rather than Studio.
3. **Baseline and invariant impact:** affected packages/contracts and an explicit check against C1–C7, authorization transaction boundaries, lock ordering, attempt fencing, `SettleEffect`, M5/P24-008 activation requirements, UNKNOWN quarantine, and class-specific recovery semantics where relevant.
4. **Smallest durable correction:** solve the underlying framework problem; do not add a local bypass, duplicate subsystem, or symptom-only workaround.
5. **Regression evidence:** tests that fail before the change and pass after it, including failure/race/replay cases appropriate to the invariant. A build or interface-presence test alone is insufficient.
6. **Framework validation:** run the relevant Miniapp CI and conformance suites; record skipped tests and environmental limitations honestly.
7. **Studio integration evidence:** update Studio to consume the public supported API and run the relevant import, contract, and end-to-end tests.
8. **Version discipline:** keep Miniapp and Studio changes separately reviewable; after the Miniapp change is committed, update Studio to the exact immutable Miniapp commit SHA and rerun the applicable pin/integration gate. Never depend on a mutable branch or uncommitted local checkout.
9. **Documentation/status:** update the owning Miniapp contract, capability-status classification, and roadmap when the public contract or guarantee changes. Do not describe a proposal as implemented or a source inspection as execution proof.

### 14.4 Relationship to Prompt32 / Prompt32.1 foundations

The Prompt32 foundational architecture work is not an excuse to duplicate Security, Observability,
or Developer Experience mechanisms in Studio. Miniapp remains responsible for its own shared
framework guarantees; Studio declares business needs and uses the framework's supported boundary.

However, the uploaded Prompt32.1 work order explicitly limits that reconciliation to documentation
and prohibits production implementation until the reconciled package is returned for owner review
and F-0 / subsequent milestones are separately authorized. Therefore:

- A clearly evidenced bug fix that preserves an already-approved and tested Miniapp contract may be handled as a targeted framework correction under the applicable implementation approval.
- A new foundational capability, changed guarantee, or change to existing execution/security semantics must first be reflected in the governing architecture and receive the relevant owner approval. Do not treat the Prompt32.1 documentation task as implementation authorization.
- If a Studio gate depends on such an unapproved foundational change, mark that gate blocked or narrow the Studio behavior to the already-approved contract; do not silently bypass the framework.
- Existing residuals remain open until their required execution evidence exists. Architectural documentation, Studio tests, or source inspection do not close operational drills or framework-specific residuals.

### 14.5 Scope and release boundary

The framework improvement lane is warranted-work-only. It does not authorize broad refactoring,
Class B/C implementation, DBOS feature expansion, new provider integrations, changes to frozen
governing prompts, or unrelated framework roadmap items. Any framework correction must directly
serve an approved Studio requirement or fix a demonstrated defect, preserve the applicable
architectural invariants, and pass its evidence gate. The Hacksgiving cut line and P0–P6 sequence
remain controlling.

## 15. Sources appendix (all checked 2026-11-24 unless noted)

- `plan4_z(2).md`: approval record; approves the plan with no foundational/structural blockers,
  sets the P0 pin and claims-discipline conditions, confirms spike-based owner decisions, and
  explicitly authorizes P0 repository/module scaffolding.
- `prompt32.1.md`, `prompt32.1_plan.md`, `prompt32.md`, and `prompt32_plan(1).md`: foundational
  architecture reconciliation inputs. Prompt32.1 is documentation-only and does not itself
  authorize production changes to Miniapp's foundational contracts.

- Framework repo @ `73351a8` (= `origin/main`): `go build ./...` → `BUILD_OK`; file/symbol
  evidence cited inline (§§2–4, 8).
- `sprites-go` v0.2.2 module + local checkout: SDK surface verified by source inspection
  (no connector APIs; no wake call; URL via `GetSprite`).
- Sprites API 0.1.12: list-connectors, create-custom-api-connector,
  test-custom-api-connector (full OpenAPI bodies captured this run).
- To verify at spike time (URLs recorded, content not yet captured): gateway discovery
  semantics, custom-api update endpoint, autostop/autostart reference, Machines app/machine
  APIs, MPG docs + `flyctl mpg`, ecosystem directory snapshot, managed-agent integration
  docs, `gemini-api-cli` self-hosted Fly provider.
