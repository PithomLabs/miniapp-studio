# Miniapp Studio — The Ultimate Roadmap

| Field | Value |
|---|---|
| Document type | Unified product + engineering roadmap |
| Status | PROPOSAL for Project Owner ratification (consolidates all prior locked decisions; conflicts resolved in §4) |
| Date | 2026-11-27 |
| Author | Technical Director, per the Consolidated Engineering Operating Charter |
| Inputs | Engineering Constitution (`SKILLS.md`), locked product roadmap (`roadmap5.md`), prompts 27.1, 28, 28.1, 28.2, 28.3, 29, 30, App Category Guidelines (`app_category_guidelines.md`), Boundary B writeup 6 |
| Precedence | Project Owner instructions → this roadmap → task-specific plans. Conflicts resolve to the newer owner decision. |
| Prompt31.1 reconciliation | APPLIED 2026-11-27 (owner-approved amendments): B0–B5 numbering locked; dual Roadmap A/B + F0 structure adopted; runtime track relabeled RT to free R0–R8; environment rows reconciled; decisions D-3 closed, D-5 opened. See Part II. Original proposal text preserved; conflicts resolved inline and marked. |

---

## 1. The product vision, stated once

> *"When I set out to start Miniapp Studio I envisioned it to be a robust RPA (robotic process automation) multi-tenant framework in Go (e.g. a single company with multiple departments or a single agency with multiple clients is considered multi-tenant), the goal is that there is an organized way to code all those tasks regardless of its app category (class A to D) that can be visualized and organized via the Studio UI in a robust software engineering methodology (ready for per-PR preview environment, staging and prod environments)."*

Miniapp Studio is therefore **two products in one coherent system**:

1. **The RPA framework (the engine).** A multi-tenant Go framework for agent-invokable business-process automation: one organized way to write every task — from a single-record read to a multi-day durable workflow — classified by execution category (A/B/C/D), with the hard parts (durability, idempotency, effects, approvals, reconciliation, tenancy) owned by the platform, never reimplemented per task.
2. **Studio (the control plane).** The UI and governance layer that organizes those tasks: a living catalog of miniapps and tools, their declared categories and trust modes, their runs and evidence — plus the full software engineering lifecycle (local → CI → per-PR preview → staging → production) that Studio itself dogfoods on its own development.

The north star is unchanged: **the Self-Developing Software Engineer** — Studio developing Studio, with Muse as the initial resident agent, executed locally or in Fly Sprites, deployed through Fly Machines, persisted in Neon PostgreSQL, and governed by explicit human authorization.

---

## 2. Is the vision already covered by the previous prompts?

**Verdict: substantially covered (~80%), with four specific gaps.** Nothing in the vision contradicts a locked decision, but four elements exist only as fragments and must be promoted to first-class roadmap work.

### 2.1 Coverage matrix

| Vision element | Where it is covered today | Coverage |
|---|---|---|
| **Go as the exclusive implementation language** | `SKILLS.md` §2.1 (binding principle 5); category guidelines throughout | ✅ Covered — binding |
| **RPA: organized execution of business processes** | Category guidelines: miniapps defined as *agent-invokable business-process automations* against SaaS/custom REST APIs; four execution classes; per-tool classification; pure domain core + thin shell; ADR-001 gates; cross-cutting patterns (approval, keyed serialization, saga, reconciliation) | ✅ Covered — deeply (this is the single best-specified element) |
| **"App category class A to D"** | Category guidelines (claude/gemini/deepseek/chatgpt/Z sections all converge on the same taxonomy); prompt 28's three governing dimensions (category / placement / authorization) | ✅ Covered — taxonomy, decision procedure, obligations, artifacts, review checklist all exist |
| **Multi-tenant (company with departments / agency with clients)** | *Mechanics* covered: RLS + tenant-isolation tests (prompts 28, 28.1, 28.3), transaction-local tenant context (`SET LOCAL`), tenant-scoped idempotency (`tenant:entity:kind:op`), per-tenant blast-radius limits, tenant isolation in the security invariants table. *Product model* NOT covered: no tenant hierarchy ADR, no tenant provisioning/offboarding lifecycle, no per-tenant catalogs/quotas, no tenant-admin UI, and prompt 28.3 explicitly leaves "one Neon project per miniapp, per tenant, or per security boundary" as an open ADR | ⚠️ **Gap 1 — tenancy product layer** |
| **Organized way to code tasks *regardless of category*** | Guidelines §6 (domain core / ports / app shell / manifest structure), dependency direction, promotion path (A→B→C→D without rewriting rules) | ✅ Covered as *discipline*; ⚠️ the **build order of the framework itself** (contracts → A/B → effects → C → D → admission) is implied but never sequenced as milestones | 
| **Visualized and organized via the Studio UI** | `SKILLS.md` §11 (Focus Console, seven-stage spine, Deploy/Run scenes, Pulse, Line, Workbench) and roadmap5 P1 cover the *shell*; prompt 27.1 adds the GitHub Project execution board. But there is **no milestone for the miniapp catalog view** (browse/search by category, trust mode, tenant), **no run-handle dashboards** (Class B status, Class C progress, Class D status/signal/cancel), **no approval inbox**, **no quarantined-effects/reconciliation view** | ⚠️ **Gap 2 — catalog & run visualization** |
| **Robust SE methodology: per-PR preview** | Prompts 28.1 (three-environment dev contract), 28.2 (per-PR Fly preview + Fly-native CI + background agents), 28.3 (Neon-first: per-PR branches, CI branches, credential inheritance), Constitution §7 Stages 5–8 | ✅ Covered — the deepest-specified lifecycle element |
| **Staging environment** | Constitution §3.3/§8.3 (persistent Fly app + persistent Neon branch, release rehearsal, same-digest promotion), charter B6 (= B5) | ◐ Covered at *policy* level; no implementation-depth ADRs yet (acceptable — it is a later milestone, but its design work must be scheduled) |
| **Production environment** | Constitution §8.4, §18 (protected Machines, protected Neon branch, digest-bound human approval, rollback), charter B6 (= B5) | ◐ Same as staging — policy yes, implementation ADRs pending |

### 2.2 The four gaps, stated precisely

1. **Tenancy product layer.** Tenant isolation is enforced; tenancy as a *product* is not designed. Needed: one ADR set defining the tenant model (organization → tenants; department/client mapping), tenant provisioning/offboarding, per-tenant catalogs and quotas, per-tenant connector credentials and trust modes, and the Neon topology for tenant data (project-per-tenant vs shared-project with RLS — prompt 28.3 already demands this decision).
2. **Catalog & run visualization.** The manifest already declares class, trust mode, idempotency identity, entity key, limits per tool. Studio must render it: catalog scene, run handles with progress/status, approval inbox, kill-switch and quarantine views, tenant switcher. Without this, the "organized way to code" is organized in the repository but invisible to the business owner.
3. **Runtime framework build track.** The B-series milestones govern Studio developing itself; the RPA framework (effect ledger, capability proxy, approval table, River/DBOS substrates, admission/validate step, determinism lint) needs its own dependency-ordered track so that category discipline is *enforced by machinery*, not just documented.
4. **Staging & production implementation depth.** Preview got a full design (prompt 28.2/28.3); staging and production have policy but no ADR set (topology, migration gating, release rehearsal script, rollback drill). These are scheduled design deliverables, not new scope.

### 2.3 One scoping decision the owner must make: what "RPA" means here

The category guidelines deliberately scope miniapps as **API-centric RPA** — business processes executed through SaaS/custom REST APIs, capability proxies, and durable workflows. They explicitly exclude **classic UI-RPA** (browser automation, screen scraping) as "outside the miniapp model; separate capability with its own security design." This was correct for narrowing the hackathon, but the founder vision says "robust RPA framework." The roadmap below assumes **API-centric RPA now, UI-RPA deferred** (see §13, Decision D-2). If UI-RPA is in the product, it becomes its own tracked capability with a security ADR — it must not be smuggled into miniapps.

---

## 3. Locked decisions inventory (what this roadmap must not reopen)

| # | Decision | Source |
|---|---|---|
| 1 | Miniapp is the exclusive application framework; Go is the exclusive language; MiniappSpec is canonical | `SKILLS.md` §2.1; roadmap5 |
| 2 | Studio is the control plane; agents execute separately; one Builder workspace per app; one active writer | `SKILLS.md` §2.1–2.2 |
| 3 | Sprites = development/execution infrastructure (untrusted/agent code, handoff, checkpoints); Machines = application runtime; never interchangeable; Workspace Preview ≠ Application Preview | `SKILLS.md` §3–4; charter §2 |
| 4 | Environments: Local (desktop + Neon dev branch), Preview (temp Fly app/Machine + temp Neon branch), Staging (persistent), Production (protected); digest-bound promotion; creation gates binding | `SKILLS.md` §3.3, §8; charter §2 |
| 5 | **Neon is the mandatory PostgreSQL provider everywhere** — local dev, Sprite dev, CI, per-PR previews, agents, staging, production, customer miniapps. No self-hosted PG, no Docker PG, no Testcontainers, no PG-in-Sprite/Machine | Prompt 28.3 (hard constraint, supersedes prompt 28 native-PG and 28.2 digest-pinned image); reaffirmed by prompts 29–30 |
| 6 | verify-full TLS enforced (N-TLS-1/2 refusal tests), R312 fingerprint, no credential exposure | Prompt 28.3 T1 evidence |
| 7 | Execution categories A/B/C/D per tool; lowest sufficient category; DBOS default-reject via ADR-001 Gates 1–4; category and placement are independent dimensions | Category guidelines; ADR-001 |
| 8 | Effect ledger (PENDING→ATTEMPTING→CONFIRMED/FAILED/UNKNOWN + quarantine + reconcile-by-read), idempotency identity, entity keys, approval-gated invocation, blast-radius limits, untrusted provider text — all framework-owned | Category guidelines §8 |
| 9 | Trust modes: dry-run / confirm / autonomous; agent never approves its own work; authorization explicit, scoped, durable | Constitution §1.3, §13; prompt 27.1 §6 |
| 10 | GitHub model: Discussions = initiatives, Issues = bugs only, Projects = Task Cards (7-field), PRs = code; Studio = durable execution/authorization/evidence record | Prompt 27.1 §2 |
| 11 | Three independent engineering lifecycles: agent execution, CI, preview — separate identities, credentials, resources | Prompt 28.2 §2 |
| 12 | One development contract, three verified environments (local / Sprite / clean CI) with clean-bootstrap gates and a parity report | Prompt 28.1 (database layer now Neon per 28.3) |
| 13 | Migration engine must be bounded end-to-end (context deadlines everywhere, dirty-state protocol preserved, no blind retries, no self-hosted PG for regression tests) | Prompt 30 |
| 14 | Hackathon demo direction: Studio developing itself with local Muse; Sprites live, not slides; six progressive steps ending at Production authorization boundary; nothing provisioned without readiness + authorization | Charter §D6/R-6; Constitution §20 |
| 15 | Governance: Technical Director directs, Project Owner authorizes consequential operations, agents execute named scope, reviewers verify evidence; seven-field decision brief mandatory | Charter §1, §8 |

---

## 4. Resolved conflicts & canonical vocabulary

These conflicts exist across the uploaded prompts. They are resolved here once; coding agents must use the canonical terms.

### 4.1 Milestone numbering (B0–B5 canonical per owner decision)

Prompt31.1 reconciliation: the owner REJECTED the charter's B1–B6 renumbering and locked **B0–B5** as canonical
(see D-3, now closed). The mapping below is preserved as a historical discrepancy record only — all current
text, phases, and evidence use B0–B5, and nothing is renamed or reordered.

| Canonical | Meaning | Charter name (superseded) |
|---|---|---|
| B0 | Studio manages one development task | B1 |
| B1 | Local loop + checkpoints + Git awareness + three-environment dev contract | B2 (was: B1 + parts of B2-01A) |
| B2 | Remote Builder Sprite: resident agent, handoff, checkpoints, recovery (incl. gateway "Part 2" live proof) | B3 |
| **B3** | **Application Preview: per-PR Fly + Neon** | B4 |
| B4 | Logs & troubleshooting loop | B5 |
| B5 | Staging & Production release controls | B6 |

### 4.2 Database doctrine supersessions

Prompt 28 (native PG19 on host + PG in Sprite) and prompt 28.2 (digest-pinned PG container for CI) are **superseded** by prompt 28.3 (Neon-first everywhere) and confirmed by prompt 30. Any document or Taskfile still referencing local PG provisioning, PGDG installs, or container-image parity claims is stale and must be corrected when touched. Taskfiles remain the declarative orchestration layer; `studio dev` remains the canonical interface; Neon CLI/API does the provisioning.

### 4.3 Dimensions vocabulary

Three independent selectors, always stated together for any task or tool: **execution category (A/B/C/D)** × **compute placement (Machine / Sprite / hybrid / local)** × **trust mode (dry-run / confirm / autonomous)**. Category is per tool, never per miniapp; placement follows the trust profile of the code, never the identity of the caller.

---

## 5. Target architecture on a page

```
┌──────────────────────────────────────────────────────────────────────┐
│  STUDIO CONTROL PLANE (Go, Fly Machines + Local Desktop)             │
│  Focus Console UI: Catalog · Runs · Approvals · Deploy · Run · Pulse │
│  Durable state: conversations, tasks, approvals, artifacts, receipts │
│  Authorization boundary · Infra adapter (Fly/Neon, allowlisted ops)  │
└───────────────┬──────────────────────────────────────────────────────┘
                │ governed execution boundaries (Gateway, Builder lease)
┌───────────────▼──────────────────────┐  ┌───────────────────────────┐
│  EXECUTION LOCATIONS (Axis A)        │  │  APPLICATION RUNTIMES     │
│  Local Desktop (default)             │  │  (Axis B)                 │
│  Fly Sprites: resident agent,        │  │  Local · Preview ·        │
│  untrusted/generated code, CI exec,  │  │  Staging · Production     │
│  handoff target                      │  │  Fly Machines, immutable  │
│  Hardened sandbox: artifact tests    │  │  digest-bound releases    │
└───────────────┬──────────────────────┘  └────────────┬──────────────┘
                │            ┌─────────────────────────┘
┌───────────────▼────────────▼─────────────────────────────────────────┐
│  NEON POSTGRESQL (mandatory everywhere)                              │
│  Production project(s) (protected) │ Non-production project          │
│  developer branches · agent branches · CI branches · PR branches ·   │
│  staging branch · run-pattern databases                              │
└──────────────────────────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────────────────────────┐
│  MINIAPP RPA FRAMEWORK (the engine, in Go)                           │
│  Manifest (class/trust/limits per tool) · Admission & validate step  │
│  Effect ledger + reconciliation · Capability proxy + token broker    │
│  Approval table · Idempotency & entity keys · Blast-radius limits    │
│  Substrates: Go handler+tx (A) · River (B/C) · DBOS (D, gated)       │
└──────────────────────────────────────────────────────────────────────┘
│
└── GitHub: Discussions (initiatives) · Issues (bugs) · Project Task Cards · PRs + CI checks
```

---

## 6. The organized way to code tasks (execution model, canonical summary)

Every task (tool, webhook trigger, scheduled job) is coded through the same four-step discipline, regardless of eventual category:

1. **State the business promise** in one customer-observable sentence.
2. **Classify** with the ordered questions: completes in one request/transaction → **A**; one independently retryable unit → **B**; bounded batch with per-record identity → **C** (River vs DBOS pilot recorded when material); must outlive its initiator (durable waits, human decisions, sagas) → **D** candidate, pass ADR-001 Gates 1–4, write the record. Torn between two classes: take the simpler, record why.
3. **Place** by trust profile: trusted code → Machine; agent-authored/generated code → Sprite; one bounded isolated task inside a D workflow → hybrid. Never choose placement from who is calling.
4. **Structure**: pure `domain/` core (no I/O, clock, rand, framework imports) + thin category shell in `app/` + manifest declarations (class, idempotency identity `tenant:entity:kind:op`, entity key or `no_entity` justification, trust modes, limits, kill switch) + class-specific tests (A: transaction/idempotency replay; B: enqueue atomicity; C: kill-and-resume; D: crash-at-every-transition, determinism lint, replay, version strategy).

Reclassification is a design change in one PR (manifest + decision record + tests + runbook). The framework, not the tool, owns: effects, idempotency enforcement, approvals, reconciliation, limits, audit, tenant context.

---

## 7. The multi-tenancy model (Gap 1 — proposed shape, requires ADR set)

**Proposed model for ratification (Decision D-1):**

- **Hierarchy:** Organization (the Studio customer: a company or agency) → **Tenants** (a department, or a client) → Projects/miniapps. The example in the vision maps directly: an agency = organization; each client = tenant; each department = tenant.
- **Isolation spine (already enforced, extend only):** PostgreSQL RLS keyed on tenant; transaction-local tenant context; tenant-scoped idempotency identities and entity keys; per-tenant blast-radius limits and kill switches; per-tenant capability credentials (the token broker never lets tenant A's connector reach tenant B); authorization principals scoped per tenant (agent principal ≠ authority principal ≠ tenant admin).
- **Neon topology (prompt 28.3 ADR, now with tenancy):** separate production and non-production Neon **projects**; tenant data placement decided per security boundary — default **shared project + RLS-protected schema/branch per environment** for small tenants, **dedicated project** for tenants with compliance/isolation demands. The decision is per-tenant-tier, recorded, and never convenience-driven.
- **Lifecycle:** tenant provisioning (create tenant, seed synthetic fixtures, provision roles with rotated credentials, register quotas), suspension, offboarding (export, verified data deletion, credential revocation, audit trail).
- **Studio UI:** tenant switcher in the Context bar; per-tenant catalog and quota views; tenant-scoped Pulse.
- **Non-goal until tenants exist in anger:** billing/metering productization (quota *enforcement* is in scope; invoicing is not).

---

## 8. Studio UI: visualizing and organizing the tasks (Gap 2 — proposed shape)

Extends the existing Focus Console contract (left rail, seven-stage spine, contextual middle scene, Pulse, Line) — no new shell:

| Scene | Content |
|---|---|
| **Catalog (new, left rail)** | All miniapps and tools for the selected tenant: declared class (A/B/C/D badge), trust mode, entity key, limits, last verified digest, version, owner, health. Filter by class/trust/tenant/status. Manifest-as-rendered-artifact — the UI reads the same manifest the admission step enforces, so it can never drift from what runs |
| **Runs (new)** | Per-tool run history; Class B job status; Class C progress (processed/failed/remaining); Class D workflow status with signal/cancel (operator actions, authorization-checked); quarantined UNKNOWN effects with reconcile actions; approval inbox for `confirm`-mode invocations with expiry and material-change hashing |
| **Deploy (existing, extended)** | Environment manager: Local / Preview / Staging / Production identities, URLs, health, artifact digest, Neon project/branch IDs, migration status, release history, rollback evidence, per-PR preview list with expiry |
| **Run / Pulse (existing)** | Structured logs first (B4); environment-scoped filtering by deployment/commit/digest; Pulse records runs, approvals, refusals, handoffs, reconciliations |
| **Workbench (existing, deferred detail)** | Source, Git, terminal, agent tasks — the developer view; the catalog/runs views are the business-organization view. Both read the same durable state |

**Design rule:** the UI organizes by *what the business promised* (tool/promise/tenant), decorated by *how it executes* (class/trust/placement) — never the reverse. Status colors stay evidence-based (green requires inspectable evidence; an agent's claim alone never turns a stage green).

Prompt31.1 reconciliation: this Focus Console is the **Roadmap A (Engineering Studio)** experience, covering B0–B5
developer workflows; it must not subsume Roadmap B. The separate **Automation Operations Studio** experience
(operations/QA/management IA) is specified in Part II. Both share the design system, domain services, identity,
authorization, and audit/event contracts — never the information architecture, and never a duplicated backend.

---

## 9. The ultimate roadmap

### 9.1 Tracks

| Track | Scope | Feeds |
|---|---|---|
| **RT — RPA runtime framework** | Contracts, substrates, effects, approvals, admission | The engine behind every miniapp (relabeled from R per Prompt31.1: R0–R8 now names the RPA roadmap phases) |
| **T — Tenancy** | Tenant model, provisioning, quotas, Neon topology | Multi-tenant productization |
| **S — Studio control plane & UI** | Catalog, runs, deploy, ops scenes; GitHub integration | Visualization & organization |
| **E — Engineering lifecycle** | Dev contract, CI, previews, staging, production | The methodology (dogfooded on Studio itself) |
| **H — Handoff & continuity** | Resident agent, Sprites, checkpoints, single-writer | B2; roadmap5 P2 |
| **O — Observability** | Structured logs → ingestion → metrics/traces | B4; roadmap5 P5 |

### 9.2 Phased plan

**Phase 0 — Foundations (NOW — largely complete, closing this month)**
- ✅ Constitution, charter, governance, GitHub project model, category guidelines, Neon-first dev contract (T1), migration-bounded-execution directive issued.
- In flight: D1 gateway §J corrections (Local-only), D2 Boundary B matrix restructure (Local-only, two clean-slate passes), D3 role-separation reconciliation vs accepted U5.12/U5.12.1 evidence, D4 initial-commit plan awaiting owner GO, R-5 read-only cloud inventory.
- **Exit evidence:** full suite + race green; two consecutive clean-slate matrix passes; commit plan approved; implementation-plan GO granted after immutable-image provenance + B2/B3 dependency corrections land.

**Phase 1 — Contracts & tenancy design (RT0 + T0)**
- Freeze the platform contracts as code + ADRs: sync-first envelope (5 s budget illustrative), effect ledger protocol, approval table (PENDING→APPROVED/REJECTED/EXPIRED, material hash, out-of-band approver), idempotency identity + entity key rules, manifest schema v1, trust modes, blast-radius limits.
- Tenancy ADR set (§7): tenant model, Neon topology per tenant tier, credential inheritance/rotation rules, provisioning/offboarding lifecycle, quotas.
- Staging/production ADR set scheduled (Gap 4): topology, migration gating, release rehearsal, rollback drill.
- **Exit evidence:** ADRs accepted by owner; contract test suite (fault-injection, no self-hosted PG) green; admission validator rejects manifests violating contracts with typed errors.

**Phase 2 — Class A/B substrate + Catalog v0 (RT1 + S1)**
- Class A handler+tx path with authorization-before-dispatch, audit, ledgered external calls; Class B River transactional enqueue, unique keys, retry/terminal policy, dead-letter inspection.
- Catalog scene v0 renders the manifest registry (class/trust/limits/digest per tool, per tenant).
- First real miniapps: Studio's own internal automations (dogfood) — each classified lowest-sufficient, with probes.
- **Exit evidence:** category review checklist passes in PR for every tool; drift test proves manifest ↔ behavior; catalog v0 shown against live manifests.

**Phase 3 — Effects & capability boundary (RT2)**
- Capability proxy + token broker (credentials never in payloads, logs, checkpoints); effect ledger with UNKNOWN quarantine; reconciler sweeper (River, periodic); kill switch effective without redeploy; per-principal/per-tenant caps.
- **Exit evidence:** chaos tests: crash-after-send yields UNKNOWN + quarantine + reconcile, never silent duplication; refusal probes for unapproved writes; tenant-A cannot touch tenant-B effects.

**Phase 4 — CI as the third proof (E2)**
- Go CI controller on a Fly Machine (webhook verify, dedupe, durable execution requests, concurrency/spend caps) + isolated CI Sprites + **Neon CI branch per validation run** (fresh child of the synthetic-data parent, migrations applied, credentials rotated, branch deleted after, reconciliation for abandoned branches).
- Results published as GitHub checks; required checks + branch protection configured; failed/skipped never reported green.
- **Exit evidence:** one commit SHA validated in a clean Sprite against a clean Neon branch; parity report rows filled for CI; duplicate/out-of-order webhooks handled; controller restart loses nothing.

**Phase 5 — Resident agent & handoff (H / B2)**
- The existing Part 2 live runbook on the real Builder Sprite (after D1 verification + per-operation owner authorization): provision/attach, Tasks API holds, pause/wake, checkpoint/restore, single-writer lease enforcement, handoff receipt local↔Sprite, evidence returned to Studio.
- **Exit evidence:** one real engineering task dispatched from Studio to a resident agent in the Sprite and back, with checkpoint-failure-recovery demonstrated; Workspace Preview shown and labeled as workspace preview.

**Phase 6 — Per-PR Application Preview (E3 / B3)**
- Preview controller: per-PR Fly app + Machines + **Neon PR branch** (synthetic fixtures, migrations, credential rotation vs branch inheritance), serialized deploys per PR with SHA fencing, update-on-push, destroy-on-close plus scheduled reconciliation GC, cost caps (max active previews, idle expiry, DB storage limits), PR comment with URL + health, agent read-only diagnostics (no org credentials).
- **Exit evidence:** two simultaneous PRs never share mutable DB state; stale deploy cannot overwrite newer commit; PR close destroys everything (verified by reconciliation sweep, not just the webhook); unmerged code cannot reach staging/prod.

**Phase 7 — Class C/D + Runs UI (RT3 + RT4 + S2)**
- Class C: framework progress records, per-record idempotency, bounded concurrency, partial-failure policy; River-vs-DBOS PILOT on the forcing workload (contact-import) recorded per ADR-001.
- Class D: DBOS selective adoption (default-reject), determinism lint in CI, replay tests, version strategy, partitioned queues per entity, compensation discipline, recovery runbooks.
- Runs scene v1: progress, workflow status/signal/cancel, approval inbox, quarantine/reconcile views.
- **Exit evidence:** kill-and-resume green for C; crash-at-every-transition + worker-kill replay green for D; one real durable workflow (e.g., invoice-to-ledger with approval) demonstrated end-to-end in Studio UI with an approval decision by the owner.

**Phase 8 — Staging (E4)**
- Persistent staging Fly app + persistent Neon staging branch; same-digest promotion from a Preview-passed candidate; migration rehearsal (expand-and-contract verified); restart/recovery drills; representative non-production data policy.
- **Exit evidence:** a release candidate promoted Preview→Staging with identical digest; release + rollback rehearsed; evidence recorded in Studio.

**Phase 9 — Production & the operating loop (E5 / B5, then steady state)**
- Protected production Machines + protected Neon production branch; explicit owner approval naming digest + migration set; deploy, health verification, release history, rollback preserved; incident → sanitized evidence bundle → agent investigation → fix → same lifecycle (the B4 troubleshooting loop closes here).
- Steady state then expands per roadmap5 P1–P5 productization (local DX, continuity, preview productization, production ops, observability platform) with R/T/S tracks maturing into the multi-tenant RPA product.
- **Exit evidence:** one real production release of Studio itself with full provenance chain (commit → spec → build → source digest → image digest → checks → approvals → deployment → health); the six-step demo direction from the charter is fully demonstrable.

### 9.3 Environment enablement schedule (when each environment comes online)

| Environment | Compute | Database | First enabled | Standing authorization |
|---|---|---|---|---|
| Local Development | Desktop | Neon developer branch | Now (T1 verified) | Standing within assigned scope |
| Workspace Preview | App running inside the Builder Sprite | Neon branch (agent-scoped) | Phase 5 | Per-operation owner GO |
| Engineering PR Preview | Temp Fly app/Machine for Studio's own PRs | Neon PR branch per PR | Phase 6 (design; prompt28.2) | Owner GO per policy (or delegated policy once proven) |
| Application Preview | Temp Fly app/Machine for the developed miniapp/automation | Neon PR branch per PR | Phase 6 | Owner GO per policy (or delegated policy once proven) |
| Staging | Persistent Fly app | Persistent Neon staging branch | Phase 8 | Separate owner authorization per release |
| Production | Protected Fly Machines | Protected Neon production branch | Phase 9 | Explicit operation-specific owner authorization, always |

Prompt31.1 reconciliation: the prior table conflated the preview concepts into one row. The six environment
concepts above are distinct and non-interchangeable; CI (Fly Machine controller + CI Sprites + per-run Neon
branches, Phase 4) is engineering lifecycle infrastructure, not an environment, and is tracked separately.
No preview environment may mutate production data or reuse production secrets without an explicitly defined
and approved operational requirement.

Read-only discovery of existing Fly/Neon resources remains allowed within scope at any phase; **no cloud mutation without the owner authorization class named above**.

### 9.4 Milestone → demo mapping (charter six-step direction)

| Demo step (charter R-6) | Roadmap phase |
|---|---|
| 1. Studio manages its own development locally | Phase 0–2 (B0–B1) |
| 2. Work transfers to a real Fly Sprite with preserved context | Phase 5 (B2) |
| 3. Studio deploys a real Application Preview (Fly + Neon branch) | Phase 6 (B3) |
| 4. Real logs + troubleshooting cycle | Phase 7 + observability (B4) |
| 5. Staging + Production authorization boundary demonstrated | Phase 8–9 (B5) |
| 6. Real Production release when all gates genuinely pass | Phase 9 |

---

## 10. Governance constants (apply to every phase)

- Seven-field decision brief on every substantive directive: execution location, application target, database target, infrastructure actions, authorization class, verification/evidence, recovery/rollback.
- Authorization classes (charter §8): local source/tests standing within scope; read-only Fly/Neon inspection within scope; Sprite provisioning, Preview creation, Staging mutations, Production deploy/migrate/rollback, destructive operations, commits, pushes — each a **separate, explicit owner authorization**; an approval for one environment never implies another; destructive defaults to deny.
- Agents investigate, draft, recommend, report; they never self-authorize, never expand scope by interpretation, and their completion claims are hypotheses until evidence verifies them (status vocabulary per Constitution §17.2).
- Migrations live in Git, applied per environment under its gate; branches isolate, they do not promote; tests own what they mutate (run-unique names).
- Cost discipline: temporary resources carry expiry + ownership metadata + reconciliation cleanup; cost never silently weakens isolation or release safety.

---

## 11. Risks and watch items

| Risk | Mitigation |
|---|---|
| Neon credential/role inheritance across branches | Resolve with executable proof before untrusted workloads touch branches; rotate inherited passwords; controlled synthetic parent (never developer branch as default parent) |
| Migration hangs / ambiguous failures | Prompt 30 directive: bounded contexts everywhere, dirty-state protocol, fault-injection regressions; never blind-retry |
| Scope creep into "framework first" before any user value | Every phase ships a usable vertical slice dogfooded on Studio itself; useful-before-complete stays binding |
| DBOS-everywhere / shadow workflow engines | ADR-001 default-reject with written records; simpler class wins; RETIRE DBOS path kept open |
| UI-RPA ambiguity | Explicit scope decision (D-2) before any browser-automation capability is built |
| Demo pressure vs invariants | "We will not provision merely to look impressive" is standing doctrine; readiness + usefulness + authorization gates all provisioning |
| Single-writer violations during parallel work | Isolated worktrees + isolated database state mandatory for parallel local streams (charter R-2) |
| Two UIs drifting into two backends | Shared domain services, contracts, identity/authorization by rule (Part II); any duplicated business logic is a defect |

---

## 12. Non-goals (unless separately approved)

Classic UI/screen-scraping RPA (pending D-2); billing/metering productization; multi-cloud or multi-database abstractions; a generic workflow canvas, plugin marketplace, or agent registry; metrics/traces before real instrumentation exists; a speculative final PostgreSQL schema; any second application framework or language.

---

## 13. Decisions required from the Project Owner

| # | Decision | Status / Recommendation |
|---|---|---|
| D-1 | Ratify the tenancy model (§7 + Part II F0): org/tenant-scope/workspace/delegation; Neon topology per tier | OPEN — final topology per ADR with cost/isolation tradeoffs shown; never silently promoted |
| D-2 | Confirm RPA scope: API-centric now; UI-RPA as a separate future capability with its own security ADR | OPEN — confirm deferral explicitly; UI-RPA must not be smuggled into miniapps |
| D-3 | Milestone numbering | CLOSED — owner rejected B1–B6; B0–B5 locked (§4.1 records the history) |
| D-4 | Approve this roadmap as the umbrella under which existing streams and B-milestones continue unchanged | OPEN — approve with amendments |
| D-5 | Deployment topology of the two UI experiences (one frontend with separate workspaces vs distinct frontends) | OPEN — evidence-backed ADR required; shared backend regardless |

## 14. Immediate next actions (next ~30 days)

1. Close Phase 0: D1 verification, D2 double clean-slate pass, D3 reconciliation, D4 commit plan → owner GO; land the immutable-image provenance correction and B2/B3 dependency clarification → implementation GO.
2. R-5 read-only Fly/Neon inventory delivered; record it as the pre-Phase-4/6 baseline.
3. Draft the three ADR sets of Phase 1 (contracts freeze, tenancy incl. Neon-per-tenant topology, staging/production) as owner review packets.
4. Begin RT1 (Class A/B substrate) locally per the standing environment rules; first two dogfood miniapps classified and cataloged.
5. Keep the B2 Sprite runbook staged (not executed) behind D1 verification + owner authorization.

---

*Bottom line: the vision is already ~80% encoded in the locked decisions — the categories discipline, the lifecycle methodology, and the Neon/Fly/GitHub doctrine are the strongest-specified parts of the project. What this roadmap adds is the missing 20% made explicit — the tenancy product layer, the Studio catalog/runs visualization, the runtime framework's build order, and staging/production design depth — sequenced into one phased plan that keeps Studio useful before it is complete, and keeps every environment, database, and release behind an explicit, evidenced authorization gate.*

---

# Part II — Prompt31.1 reconciliation (2026-11-27, owner-approved)

## PII.0 Reconcile verdict

Extend, don't duplicate: this document carries the authoritative product architecture after the §4.1, §8, §9,
§11, §13, §14 amendments above plus the sections below. No competing roadmap was created. Prior prompts
(27.1–30) stand untouched as implementation workstreams. `roadmap4.md` (Nov-25 gateway roadmap, Go/Miniapp
binding invariant, control/execution-plane ownership, security invariants, ops-readiness gaps) and `writeup2.md`
(consolidated adversarial review: retain HMAC/outbox/v1 events/separate UIDs, add fleet-lifecycle + metering
as longer-term gaps) were read in full and corroborate the citations used here, with three caveats recorded:
(a) roadmap4's Phase A–E status snapshot is superseded by later accepted evidence — current status comes from
§PII.1, never from that snapshot; (b) the external claim attributing the Class A–D taxonomy to "roadmap4 §6"
is a misattribution (§6 is a retained-decisions table) — taxonomy provenance is `app_category_guidelines.md`
§3, as cited in PII.3; (c) roadmap4's tenant+workspace model is extended, not contradicted, by PII.2's
organization/delegation layer, as prompt31.1 requires.

## PII.1 Capability status matrix

Labels: IMPLEMENTED (in repo) · ACCEPTED (evidence + approval) · DESIGNED (documented) · IN PROGRESS ·
PLANNED · DEFERRED · UNVERIFIED. One row per capability; citations, not assertions.

| Capability | Status | Evidence / note |
|---|---|---|
| Class A–D taxonomy + discipline | DESIGNED | `plans/app_category_guidelines.md` §2–§6; enforcement machinery is RT-track work |
| Class A runtime (invoke + approve + resume) | ACCEPTED | Miniapp framework; P0/P2 gates passed per evidence docs |
| Effect ledger, idempotency, blast radius | DESIGNED | Guidelines §8; Studio-side implementation planned (RT2) |
| Neon-first data layer (dev/branch/TLS/R312) | ACCEPTED | `docs/t1-evidence.md`; live Neon proofs recorded |
| Bounded migrations + dirty protocol | ACCEPTED | Prompt30 adapter; live bootstrap proofs in `docs/prompt30-evidence.md` |
| CI controller / per-PR previews / agents | DESIGNED | ADR-007/008/009/012/013/014; implementation T2–T4 gated, not started |
| Tenant isolation mechanics (RLS/tests) | ACCEPTED | `testdb` + tenant suites; `docs/*-evidence.md` |
| Tenancy product (org/delegate/provision/quotas) | PLANNED | Designed in PII.2 below; D-1 topology open; fleet-lifecycle + metering gaps corroborated by roadmap4 §5/writeup2 §6 |
| Automation catalog (product) | PLANNED | R2; schema proposed in PII.3, no code |
| Scheduling / orchestration / SLA runtime | PLANNED | Engine decision open per governing decision 14 |
| Operations dashboard | PLANNED | IA proposed in PII.5; no code |
| Per-task promotion pipeline | PLANNED | R5; app-level promotion policy exists (§9.3) |
| Tenant onboarding flows | PLANNED | R6; lifecycle outlined in PII.3 |
| Metering/billing | DEFERRED | Quota enforcement in scope; invoicing explicitly out (non-goals) |
| B0 task management | ACCEPTED | Accepted scope per prompt31.1; conversation/question/approval handlers live |
| B1 local dev loop | ACCEPTED | Accepted scope; apps/spec/deploy pages live |
| B2 remote agent + connectors | IN PROGRESS | B2-01A accepted; gateway Part 1 done; live proofs pending |
| B3 Application Preview UI | PLANNED (UI partial) | Deploy/provision/retire handlers exist; no preview backend |
| B4 troubleshooting UI | PLANNED | No incident/RCA routes; activity feed only |
| B5 release UI | PLANNED | No release/staging/prod routes |
| Staging/production implementation | PLANNED | Policy in §9.3; ADRs pending per Gap 4 |

## PII.2 F0 domain core

### Tenancy and delegation

- **Organization:** a company, agency, or account owner. Owns billing relationship and top-level admins.
- **Tenant scope:** a security and data-ownership boundary. A department (company model) or a client business
  (agency model) is a tenant scope — not a nested afterthought.
- **Workspace:** a collaborative area inside a tenant scope. Never a tenant by itself.
- **Membership / Role:** a user or agent's relationship to an organization or tenant, with permissions.
  A user may belong to multiple tenants; a client's admin is not the agency's admin.
- **Delegation:** explicit, scoped, revocable authority granted across organizations (agency operates client
  scopes). No implicit cross-tenant access, ever.
- **Company pattern:** one organization, Finance/HR/Operations tenant scopes; corporate oversight is explicit
  delegated read, not ownership.
- **Agency pattern:** agency organization plus separately owned client organizations; operators act under
  per-client delegation, never ambient authority.
- Consequences (binding on R1): tenant identity propagates into every automation, run, record, connector,
  credential, log, approval, and audit event; RLS + fail-closed isolation extend the existing mechanics;
  shared templates never imply shared data/credentials; Sprite/app placement stays an infrastructure decision
  (ADR-001, ADR-008), never the tenancy definition. Neon project topology stays OPEN (D-1).

### Automation entity (first-class)

**Tenant scope → Automation** (owner, business purpose, category A–D, lifecycle: draft → preview → staging →
production → deprecated → retired) **→ Automation versions** (source, contract, tests, dependencies;
immutable once promoted) **→ Workflows** (triggers, activities, dependencies, rules) **→ Execution runs**
(status, evidence, results, recovery) **→ Deployment targets** (dev/preview/staging/production bindings).
One Automation ≠ one Miniapp ≠ one Fly app ≠ one Sprite: an automation may compose several of each. Ownership,
identity, versioning, classification, and dependencies live on Automation; R2 catalogs it, R3 executes it,
R5 promotes it per version.

## PII.3 Class A–D execution contracts

Source: `plans/app_category_guidelines.md` §3 (failure contracts + decision procedure). Substrate column lists
the guidelines' candidates as unevaluated input only — engine selection needs explicit ADRs per governing
decision 14. Category × placement × trust stay independent dimensions (§4.3).

| Class | Shape | Failure contract | Agent contract | Candidate substrate (unevaluated) | Tests required |
|---|---|---|---|---|---|
| A — Ordinary operation | One request, no resume | Retry whole request; idempotency key on writes; ledgered single effects (timeout = UNKNOWN) | Inline result, ~5s sync budget | Go handler + Postgres tx; no queue/engine | Unit, contract, authorization probes, idempotency replay |
| B — Simple background job | One retryable unit | Retry from scratch; no preserved progress | Run handle; poll result | River, transactional enqueue + unique keys | Unit, enqueue atomicity, retry/terminal, duplicate collapse |
| C — Bounded multi-step | Many records, bounded period | Resume without replaying completed records; per-record idempotency | Run handle + progress (processed/failed/remaining) | River + progress record; DBOS pilot only if machinery resembles a workflow engine | Kill-and-resume, per-record idempotency, partial-failure policy |
| D — Durable workflow | Survives process replacement; waits, checkpoints, human/external events | Resume from last completed step; never repeat confirmed effects | Run handle + status/signal/cancel | DBOS Transact Go only via ADR-001 gates + written record | Crash-at-every-transition, replay determinism, worker-kill recovery, version compatibility |

Cross-cutting (combine with any class): approval-gated invocation (single approval table, PENDING→
APPROVED/REJECTED/EXPIRED + material hash); keyed serial execution per entity; saga/compensation (D, sometimes
C; never auto-compensate UNKNOWN); reconcile-by-read + sweeper; strict policy guardrails in CI/production.

## PII.4 UI architectures and maturity matrices

### Roadmap A — Engineering Studio (Focus Console)

UI maturity: Designed / Implemented / Partially implemented. Functional maturity: Functional / Partial /
Planned / Not connected / Blocked. Handler citations verified in `server/routes.go` + `server/templates/`.

| Area | UI maturity | Functional maturity | Basis |
|---|---|---|---|
| B0 conversations/questions/approvals | Implemented | Functional (accepted scope) | Handlers + templates live; Phase A accepted |
| B1 apps/spec/deploy pages | Implemented | Functional (accepted scope) | CRUD/validate/preview/deploy/retire handlers + templates live |
| B2 agent console (attempts/checkpoints/kill/resume) | Implemented | Partial | Controls live; B2-01A accepted, live proofs pending |
| B2 connectors + custom wizard | Implemented | Partial | List/wizard/test/create handlers live; genuine-connector proof pending |
| B2 Workspace Preview | Partially implemented | Planned | Interstitial/result templates only; no backend |
| B3 Application Preview scenes | Partially implemented | Planned | Deploy pages exist; no preview controller (T3 gated) |
| B4 incident/RCA/recovery | Planned | Planned | No routes; activity feed is not incident management |
| B5 staging/prod/release/rollback | Planned | Planned | No routes; policy only (§9.3) |

Honesty rule: every non-Functional cell renders its state explicitly (badged Planned/Partial/Blocked);
fixtures are labeled examples; no control implies an unsupported operation; no fabricated history.

### Roadmap B — Automation Operations Studio (separate experience)

Own information architecture for operators/QA/managers (14 areas): org/tenant overview, automation catalog +
discovery, process/workflow visualization, automation details/versions/ownership, schedules/triggers, execution
monitoring + run history, QA/test results, approval + exception queues, SLA/operational health, connectors,
environment/release management, audit/governance, cost/usage. Role-scoped views (e.g., department approvals
without source/credential access). No code exists (all PLANNED); design now, implement incrementally behind
domain/runtime gates. Shares with A: design system, identity/tenant context, API + authorization contracts,
terminology, audit/evidence services. Deployment topology (one frontend vs distinct apps) is open (D-5).

## PII.5 R0–R8 roadmap, dependencies, and track reconciliation

| Phase | Scope | Gate: needs |
|---|---|---|
| R0 | RPA domain architecture (tenancy, Automation contract, category rules, operating model) | F0 design (PII.2–PII.3); no single-tenant shortcut catalog |
| R1 | Hierarchical multi-tenancy + tenant lifecycle | R0; Neon topology ADR (D-1) |
| R2 | Automation catalog, definitions, versioning | R1 |
| R3 | Scheduling, orchestration, execution runtime | R2; engine ADR (governing decision 14) |
| R4 | Operations/QA/management workspace | R3; F0 observability contracts |
| R5 | Per-automation preview/staging/promotion/rollback | R2; E-track staging/prod gates (Phase 8–9) |
| R6 | Tenant onboarding + administration | R1, R5 |
| R7 | Metering, quotas, billing | R6 (billing itself stays DEFERRED; model must not block it) |
| R8 | Enterprise scale (capacity, resilience, governance) | R7 + production release management |

Track reconciliation (no competing taxonomies): RT (runtime framework) feeds F0 contracts + R3 runtime;
T (tenancy) feeds F0 tenancy + R1/R6; S (Studio/UI) splits by workspace: Focus Console work → Roadmap A,
catalog/runs/ops scenes → Roadmap B R2/R4; E (engineering lifecycle) is Roadmap A delivery + R5's staging/prod
gates; H (handoff) → B2/B3; O (observability) → F0 contracts + R4. Phase 0–9 (§9.2) continue as the sequenced
delivery plan; R-phases attach at their gates, they do not renumber it.

Cross-roadmap rules: R-series consumes F0 identity/tenancy/authorization/execution/persistence contracts and
E-track environments; B-series never blocks on R implementation (B2 demo proceeds); shared engines, connector
frameworks, release managers, and authorization systems are implemented once, in F0.

## PII.6 Threat model (concise)

Trust boundaries: tenant scopes (company departments, agency clients); delegation edges (explicit, revocable);
environment tiers (dev → preview → staging → production, no shared mutable state/secrets); agent vs authority
principals (agents never self-approve, never hold org credentials). Key vectors and controls: cross-tenant
read/write (RLS + fail-closed context + tenant test suites); delegation abuse (scoped, expiring, audited
grants; agency operators never ambient); confused deputy via shared templates (templates carry no data or
credentials); credential exfiltration (tenant-scoped broker, verify-full TLS, redaction-first evidence);
promotion of unreviewed automation (immutable versioned artifacts, digest-bound human approval, rollback that
preserves history); forged agent activity (evidence requires inspectable artifacts, never agent claims). Full
STRIDE-per-surface treatment is ADR-backlog item T1.

## PII.7 ADR backlog and proposed Task Cards

Unresolved decisions only (no selections made here): F0-1 Neon project topology per tenant tier (D-1);
F0-2 two-UI deployment topology (D-5); F0-3 runtime substrate evaluation against R3 requirements (governing
decision 14); F0-4 delegation token format + lifecycle; F0-5 quota/metering event schema (billing-compatible);
R0-1 catalog schema + lifecycle states; R0-2 scheduling/dependency/SLA model; R0-3 operations IA detailed spec;
T1 full threat model per surface. Proposed Task Cards (titles; file in GitHub Project only on owner GO):
"Draft F0 tenancy ADR", "Draft Automation entity + migration plan", "Class contracts enforcement matrix",
"Roadmap A UI coverage pass (B0–B5 matrix→screens)", "Roadmap B IA wireframes", "R0 domain package (Go types)".

## PII.8 Twelve-question traceability

1. What is Miniapp Studio? §1 + Part II header. 2. Who uses it? §8 + PII.4 (two workspaces, personas).
3. Why two UIs? §8 note + PII.4 (distinct daily workflows, shared backend rule). 4. What is shared? §5 + PII.2
(F0 contracts). 5. Secure org/department/client coexistence? PII.2 tenancy + PII.6. 6. What is an Automation?
PII.2 entity. 7. Class A–D operational differences? PII.3. 8. Author/execute/test/recover? §6 + PII.3 + RT track.
9. Preview/stage/promote/rollback? §9.3 + PII.5 R5. 10. Roadmap dependencies? PII.5. 11. Real vs planned? PII.1
+ PII.4 matrices. 12. What next? §14 + PII.7 (F0 ADRs first, B2 demo unblocked).
