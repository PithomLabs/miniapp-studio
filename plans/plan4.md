# Miniapp Studio — Implementation Plan

**Status:** Revised per `plans/plan2_review.md` §4 + `plans/prompt2.md` §5G; owner-approved direction with provisional framework pin.
**Scope:** Planning only. No code, dependency, migration, deployment, or governing-plan changes.
**Date:** 2026-11-24.

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

## 2. Repository inspection and evidence inventory

### 2.1 Framework pin (provisional, per approval condition)

- Candidate: `73351a801fc54e19ee7254e6fe3ea7fbed859378` (`prompt31`), branch `main`.
  `origin/main` resolves to the same SHA; no newer commits exist to survey (checked 2026-11-24).
- Uncommitted state (left untouched): `docs/architecture/` (Prompt32 proposed package),
  `docs/prompt32*.md`, `docs/review_architecture.md`. None of it is implementation evidence.
- Baseline verified this run: `go build ./...` succeeds (`BUILD_OK`); toolchain go1.27.3
  against module `go 1.26.0`.
- **Finalization gate (hard, before implementation relies on the pin):** run the CI suite from
  `.github/workflows/ci.yml` against the pinned SHA on PG19 — `gofmt -l` clean, `go vet ./...`,
  `go test -p 1 ./...`, `TestSleepAcrossRestart` + `TestProbeRestart` must execute (not skip),
  `go test -race ./contract/ ./internal/... ./spikes/` — plus a Studio import spike (new module
  imports the pinned framework and runs one Class A invoke + one approval park/decide round-trip).
  Record commands, SHAs, and results. Until then the pin is provisional and every reuse claim below
  is conditional on it.

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
| Framework | Pinned dep | Invocation, auth, approvals, effects, audit, Postgres/DBOS | Import studio code (dependency direction) |
| Fly adapters | New, behind owned interfaces | Sprite/Machine/connector lifecycle, typed REST client | Fabricate URLs, log secrets |

## 4. MiniappSpec and interpreter contracts (revision A: execution semantics)

### 4.1 Canonical spec

One versioned `MiniappSpec` (JSON Schema, versioned, content-hashed with SHA-256 hex digest).
Sections: `app` (id, slug, name, versions), `data` (entities, queries, bindings), `ui`
(pages, components, bindings, actions), `workflows` (trigger + typed steps), `capabilities`
(logical op IDs + connection bindings), `policies` (roles, effect classes, confirmation rules,
limits), `fixtures` (inputs + expected outputs), `presentation` (tokens), `metadata`
(schema version, content hash, provenance — no secrets). Workflow is a section, not a language.
Editing creates a new immutable version; a run binds the version it started with.

### 4.2 MVP execution contract (no half durable engine)

- **Class:** the MVP runner is **Class A only** (`executor.SyncExecutor.Execute`): synchronous,
  in-request, single tenant transaction (claim → entity lock → liveness recheck → handler in
  savepoint → terminal completion commit, per `executor/classa/classa.go`).
- **Sync vs async:** all MVP runner operations are synchronous request/response. No background
  jobs, no River (package absent), no DBOS workflows in the MVP path.
- **Approvals:** two-call explicit flow over existing APIs — `Invoke` returns
  `waiting_approval` + `approvalID` (parked by `runtime/service.go:parkApproval`, persisted by
  `approvals.Service.Submit` with intent hash + expiry); user decides via `DecideApproval`
  (`approvals.Service.Decide`); client re-queries via `Status`. No long-poll (incompatible with
  scale-to-zero) and no DBOS wait/signal wake.
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
| (b) Shared Fly App/router | Weak (shared process) | Slug path/route on one host | Fastest | Cheapest | A route is not isolation; app-data separation must be proven, not assumed |
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

### 6.3 Control-plane boundary

Full-access connector operations run only in the trusted Studio backend after an authorized
user action. Raw tokens and Fly-org credentials never enter the Sprite, prompts, specs,
logs, timelines, or browser responses. Wizard order: Define → Test (green) → Scope/Grant
(default-deny, Sprite name-prefix/labels + endpoint rules). Miniapp registers one operation
descriptor per allowed call (logical ID, method, path, schemas, effect class, approval,
limits, timeout); the runtime rejects anything undeclared. A dedicated negative test proves
an out-of-scope path is denied and receipted.

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

## 8. Security and failure analysis

- **Trust boundaries:** browser ↔ Studio backend (session auth) ↔ framework kernel (credential
  bindings) ↔ Fly control plane (org token, backend-only) ↔ Sprite agent (least capability,
  no secrets) ↔ gateway (attaches credentials) ↔ providers.
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

## 9. Sequenced implementation tasks and dependency graph

```text
P0 pin gate (§2.1) ──┬──> P1 spec+validator ──> P2 runner+renderer ──> P5 golden path
                     ├──> P3 Fly adapters ──> P4 Studio UI ─────────┘
                     └──> (all) ──> P6 rehearsal + claims audit
```

| Phase | Tasks (each: purpose, files, tests, exit) | Depends |
|---|---|---|
| P0 | Finalize pin per §2.1 gate; scaffold module + CI (gofmt/vet/test/golangci) | Approval |
| P1 | MiniappSpec JSON Schema + version/hash policy; deterministic validator + error taxonomy; fixtures | P0 |
| P2 | Generic runner registration + interpreter (Class A); server renderer; dry-run; deliberate-rejection test | P1 |
| P3 | Owned Fly interfaces; `sprites-go` adapter; typed Connectors REST client + fakes/contract tests; topology spikes → decision record; MPG/Neon spike → decision record; slug/URL lifecycle | P0 |
| P4 | Six screens per §7.1; ecosystem explorer; Custom API wizard; permissions card; run/evidence views; all via `runtime.Service` | P1, P3 |
| P5 | Builder-agent loop (draft → validate → fix); Invoice Review Desk; §7.2 e2e; negative authz test; secret-absence tests | P2, P3, P4 |
| P6 | Clean-state rehearsal, README claims audit vs passing tests, organizer-eligibility filing, submit | P5 |

Cut line (Hacksgiving): one complete app; no workflow canvas, marketplace, billing,
multi-tenant admin, white-label, custom domains, DBOS UI, undo-of-world. Second app optional.

## 10. Acceptance tests and exit gates (G0–G12 mapped)

| Gate | Pass/fail evidence |
|---|---|
| G0 repo/dependency truth | Pin SHA + CI results recorded; SDK/API pins recorded; blockers listed |
| G1 Sprite + connector | Agent process on real Sprite makes a real gateway call; secret-absence assertions pass |
| G2 ecosystem truth | 3-state statuses render from distinct sources; SDKs/integrations not labeled connectors |
| G3 Custom API | Wizard create→test→scope passes; no secret in spec/Sprite/logs; default-deny verified |
| G4 spec generation | Valid spec accepted; invalid component/op/path rejected deterministically; hash/version tested |
| G5 UI generation | UI renders from spec; bindings resolve; empty/error states visible |
| G6 app execution | Bounded connector-backed path runs repeatably; no unapproved writes |
| G7 demo readiness | Clean-state rehearsal passes; claims == passing tests; limitations documented |
| G8 dogfooding | All consequential ops via `Invoke`/audit; bypass-allowlist test green; gaps explicit |
| G9 Fly deployment | Studio reachable on Fly; reproducible `flyctl` deploy; autostop behavior tested or warm-capacity justified; no ephemeral-FS state |
| G10 per-app URL | Slug → real persisted unique URL; opens correct app; collision/privacy/lifecycle tested |
| G11 control-plane integration | Least-privilege adapters; secret ownership; lifecycle ops exercised in golden path |
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
| Pin invalidation (tests fail on `73351a8`) | §2.1 gate blocks P1+; survey/fix before any reliance |
| Scope vs Nov 29 deadline | §9 cut line; one golden path; roadmap honestly labeled |
| Prompt-wrapper perception | Visible spec, validator, permissions, rendered app (§7.2 steps 2–4) |
| Org lacks demo connector | Box-preferred, Custom API fallback; check live catalog first |
| Custom API becomes arbitrary proxy | Operation descriptors + runtime deny-unknown + negative test (§6.3) |
| Secret leakage | Backend-only privileged calls; redaction + absence tests (§8) |
| Framework-on-Sprite portability | P3 import spike; Studio stays on Machines, Sprite holds agent only |
| Misleading durability/undo claims | §4.2 boundary; §8 abuse cases; claims audit in P6 |
| Pre-existing-foundation eligibility | Pinned dep + disclosure + organizer filing (P6); no copied code |
| Fly topology/cost surprise | Spikes before selection; decision records required (§5) |

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
- MVP seams: provider reference fields in spec `metadata`/policies (no vendor coupling of
  capabilities/UI/governance), execution-target enum (sprite/machine), adapter interface
  declarations. No managed-agent adapter implemented in the MVP; a provider selector alone
  never counts as an integration.
- Vendor-neutrality invariant: changing providers never rebuilds the generated app.
- Evidence required at roadmap time: current managed-agent integration docs, `gemini-api-cli`
  self-hosted Fly provider repo, per-provider threat model, contract tests, cost/ops trade-offs.

### 12.2 Durable workflows (flagship, post-hackathon)

DBOS-backed waits/signals, approval wake, retries, idempotency, UNKNOWN inbox, sagas,
versioning, replay — only behind crash/failure tests. The MVP §4.2 contract must not be
extended ad hoc; migration to a durable adapter is an explicit roadmap phase.

## 13. Unresolved owner decisions

1. Pin base confirmation: freeze at `73351a8` once the §2.1 gate passes (no newer commits exist).
2. `sprites-go` pin: released v0.2.2 vs pseudo-version of the newer local checkout.
3. Authorized Fly environment: when/where Fly spikes (`flyctl`, org token, Sprite runs) may execute.
4. URL topology + MPG/Neon: decided by P3 spikes; owner ratifies decision records.
5. Demo connector anchor if Box is unavailable in the org (re-check live catalog day one).
6. Repo init + module scaffolding: awaiting explicit authorization (not granted by this approval).

## 14. Sources appendix (all checked 2026-11-24 unless noted)

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
