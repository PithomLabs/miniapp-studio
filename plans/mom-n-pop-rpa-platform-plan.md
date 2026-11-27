# Mom-n-Pop Skills → Miniapp Studio RPA Platform Plan

Planning-only architecture mission (prompt34). No code, migrations, live
operations, provider auth, commits, or credential handling were performed.
This document is the sole artifact. Claim tags: VERIFIED (file citation),
ASSUMPTION (explicit), PROPOSAL, OWNER-DECISION.

## 0. Sources and method

- Reference: `https://github.com/hewi333/mom-n-pop-skills` per prompt34,
  studied via working snapshot `/home/chaschel/Documents/go/Mom-n-Pop-Skills-main/`
  (not a git repo; files dated Sep 11 — snapshot declared, not assumed
  identical to remote HEAD; re-verify on drift).
- Studio sources: `plans/miniapp-studio-ultimate-roadmap.md` (_locked
  decisions, F0/R0–R8, tracks), `Taskfile.dev.yml` + `Taskfile.yml`,
  `docs/dev-contract.md`, `docs/dev-serve.md`, `docs/taskfile.md`,
  `server/config.go` (`_FILE` convention), `dbenv`, `migratedb`,
  Miniapp framework `contract/` (A/B/C/D classes, trust modes, authority,
  ADR-001 template), `connector/`, `tools/`, `fly/` surfaces.
- Verified skill inventory: 16 skills, each studied exactly once —
  the 15 entries of prompt34 §1 plus `small-business-website-spec`
  (present on disk, omitted from §1; included per owner correction).
- Every recommendation traces to a source path below; anything needing
  live proof is recorded as a future implementation verification
  requirement, never assumed (§10 rule: no owner-terminal live runs in
  this assignment).

## 1. Verified skill inventory (prompt34 §1)

### 1.1 lead-to-payment (orchestrator)
`skills/lead-to-payment/SKILL.md` + `references/{godaddy-email-setup,
website-lead-intake}.md`. Owns the Intake→Qualify→Estimate→Approve→
Schedule→Invoice→Pay→Review pipeline across 9 stage handlers. Stage
pattern per skill: detect branch (new/existing/junk/repeat) → ask/infer →
write CRM lead → safety classification → log `stage_N_result`. Subagent
model: fresh subagent per stage inheriting only `MEMORY.md` + stage file,
no cross-stage memory. Freshness: hourly cron + `no_agent` guard +
`last_processed_uid` watermark + digest-before-detail + `is_duplicate`.
Safety: HIGH classification on anything customer-facing/financial;
plain-language preview + explicit authorization; zero autonomous sends.
First-run flood control: UID-seeded watermark + 24h window + hard cap 50.

### 1.2 crm-lite
`skills/crm-lite/SKILL.md`. Single-tenant SQLite (`~/.hermes/crm.db`)
contact ledger: customers/vendors/employees with lifetime value,
payment history, job tracking, tags, notes, follow-up dates, referral
sources. Query modes (natural-language/smart/exact); onboarding via
guided questions. No auth layer, no multi-user, no encryption at rest —
isolation is per-VPS filesystem.

### 1.3 estimator-engine
`skills/estimator-engine/SKILL.md`. Deterministic parameterized pricing:
base sqft rates × region × frequency × severity × extras, deposit math,
quote lifecycle, assumption surfacing, adjustment guardrails. Pure
decision tables, no external calls — the closest thing in the repo to a
portable domain kernel.

### 1.4 stripe-payments
`skills/stripe-payments/SKILL.md`. Payment Links + Invoices + Customers
via Stripe CLI/API; idempotency keys on creates; least-privilege keys;
never log PANs; test/live mode discipline; refund + dispute flows with
evidence capture.

### 1.5 stripe-link-cli
`skills/stripe-link-cli/SKILL.md` + `references/error-fixes.md`. OUTBOUND
spend agent: virtual one-time cards via Link CLI with push-approval,
~90s card TTL, combined poll+retrieve pattern, `/tmp` card hygiene
(write 0600, rm immediately). Error log is session-verified fixes —
evidence trail as documentation.

### 1.6 quickbooks-online
`skills/quickbooks-online/SKILL.md` + `references/{oauth2-flow,qbo-api,
qbo-mapping}.md`. Full OAuth2 (auth code, rotating refresh, re-auth on
`invalid_grant`), token encryption at rest, sandbox-vs-production base
URLs, `RequestId` idempotency header, customer/estimate/invoice/payment
CRUD + sparse update, service-business entity mappings. Most complete
connector discipline in the repo.

### 1.7 outlook-graph
`skills/outlook-graph/SKILL.md`. Microsoft Graph mail: OAuth2 + MSAL
token cache, delta-query polling, folder-scoped reads, attachment
handling, send-with-approval. Same credential shape as QBO, different
provider mechanics — the pair defines the adapter interface.

### 1.8 mailchimp-integration
`skills/mailchimp-integration/SKILL.md`. API-key auth, datacenter-prefixed
base URLs, audience/tag/segment management, double-opt-in compliance,
unsubscribe honoring, campaign reporting. Compliance-first connector.

### 1.9 marketing-campaign-builder
`skills/marketing-campaign-builder/SKILL.md`. Template-driven campaign
composition over mailchimp primitives: audience selection, content
assembly, schedule windows, approval preview, send + report-back.
Demonstrates skill→skill composition via a shared primitive layer.

### 1.10 construction-agent
`skills/construction-agent/SKILL.md` + `references/{google-workspace-setup,
jobtread-api-setup}.md` + `scripts/{daily_briefing,drive_to_jobtread,
google_mail,jobtread_pave}.py`. Vertical adapter bundle: Gmail/Drive/
Calendar ingestion → JobTread Pave API project sync → daily briefing.
Only skill family with executable code (~1.3K lines Python CLI scripts,
env-key auth, argparse surfaces). Proves the adapter-bundle pattern.

### 1.11 grasshopper-voicemail-monitor
`skills/grasshopper-voicemail-monitor/SKILL.md` + `references/{cron-script-
arguments,notification-email-monitor}.md`. No-API integration via email
notification parsing: UID-tracked IMAP polling, transcription extraction,
urgency triage, digest + alert routing. The canonical polling-adapter
reference (deterministic intake without provider webhooks).

### 1.12 financial-analysis
`skills/financial-analysis/SKILL.md`. Read-only analytics over exports/
APIs: P&L ratios, cash-flow runway, anomaly flags, benchmark comparisons.
Pure functions over fetched data; no writes — the cleanest Class-A-shaped
domain logic in the repo.

### 1.13 small-business-ai-transformation
`skills/small-business-ai-transformation/SKILL.md` + `references/
quickbooks-financial-analysis.md`. Methodology skill, not runtime code:
discovery → ROI-ranked roadmap → phased rollout with rollback criteria.
Informs rollout/adoption staging, not platform contracts.

### 1.14 agent-cheatsheet-builder
`skills/agent-cheatsheet-builder/SKILL.md` (+ `examples/cheat-sheet-
example.md`). Generates owner-facing capability documents with a hard
honesty rule: only document wired, tested capabilities; forbidden phrases
for aspirational claims. Direct ancestor of the platform's capability
attestation requirement.

### 1.15 hermes-production-ops
`skills/hermes-production-ops/SKILL.md` + `references/{gateway-restart-
notifications,gateway-streaming-config}.md`. Production discipline for the
agent runtime: config-change protocol, restart-notification policy,
streaming-flood diagnosis, session hygiene, backup/restore of agent state.
The ops-runbook pattern the Automation Operations Studio inherits.

### 1.16 small-business-website-spec
`skills/small-business-website-spec/SKILL.md` + `references/base44-
pricing.md`. No-code website delivery spec via Base44 with Link-CLI
purchase flow. Included once per owner correction; platform-relevant
only as a second Link-CLI consumer (proves spend-primitive reuse) and a
content-delivery vertical stub. Otherwise out of RPA platform scope.

## 2. Interpretation notes (Honesty Directives applied)

- **Tenancy inversion.** The reference is single-tenant-per-VPS with zero
  in-code tenant isolation (VERIFIED: crm-lite, `.env` layout). The
  platform inverts this: tenant scoping enforced in contracts, persistence
  (RLS), and authorization — never assumed from deployment topology.
- **No Python runtime in the platform.** Scripts are studied for logic and
  provider mechanics, then re-expressed as Go adapters behind provider
  contracts. The platform never shells to the reference scripts.
- **Dialogue is a channel, not the product.** Telegram conversational UX
  maps to the agent interaction contract (proposal/preview/confirm shapes),
  rendered by both the visual workspace and the agent channel.
- **Checklists become gates.** Every per-skill verification checklist maps
  to an automated gate (test, admission check, or runtime assertion) or is
  recorded as a documented manual procedure — never dropped silently.
- **Money paths stay human-gated.** Every live-money or customer-visible
  send in the reference requires owner authorization; the platform encodes
  this as non-bypassable approval requirements, not agent instructions.

## 3. Platform abstraction (prompt34 §3)

### A. Platform domains
Five domains, each independently versioned and replaceable behind its
contract. Anti-giant-CRM guard: no domain stores another domain's
operational records; cross-domain joins happen in read models keyed by
stable IDs, never by shared mutable tables.
PROPOSAL: `Party` (contacts, orgs, consent state), `Commercial`
(estimates, invoices, payments, rate cards), `Engagement` (campaigns,
messages, reviews, callbacks), `Workforce` (jobs, schedules, field
evidence), `Ledger` (audit-grade event + receipt log; append-only).
Each domain: record schemas, lifecycle states, retention rule. A Miniapp
implements one domain slice or one vertical composition — never two
domains' writes in one tool (enforced by capability-ref review).

### B. Capability model
Capabilities are the ONLY unit the agent is offered: typed, versioned,
idempotency-declared operations grouped by domain (`commercial.estimate.
create@v1`, `engagement.message.send@v1`). Shape follows the framework
`CapabilityRef{provider, operation, version}` + `ApprovalRequirement` +
`IdempotencyRequirement` (VERIFIED: framework `contract/manifest.go`).
Inputs/outputs are JSON Schemas validated at admission; every mutating
capability declares its key scope. Semantic versioning: additive fields
minor, behavior change major. Deprecation requires a dated migration
note, never silent removal.

### C. Orchestration and the engine question
Default: A/B/C-class Miniapp tools + framework admission (trust modes
dry-run/confirm/autonomous, per-tenant policy) + deterministic
schedulers. A durable workflow engine (Class D / DBOS substrate) is
adopted ONLY through the framework ADR-001 gate (VERIFIED: framework
`docs/adr/miniapps/adr001-template.md`: scored 0–14 rubric, three gates,
human reviewer, REJECT|DEFER|PILOT|ADOPT). Until such an ADR is approved,
multi-step flows compose as staged capabilities with explicit checkpoints
(the lead-to-payment stage pattern), crash-safe via persisted stage
receipts and idempotent replays. OWNER-DECISION: whether the first
vertical slice needs Class D at all (default PROPOSAL: no).

### D. Miniapp structure contract
Every automation Miniapp ships: manifest (id, version, class, declared
capabilities, approval + idempotency + retry declarations per the
framework contract), tool handlers (pure domain logic; no ambient I/O),
capability attestations (machine-readable wired-vs-planned statement,
generated from admission tests — the cheatsheet honesty rule made
executable), per-stage receipts, and a verification appendix mapping
each reference checklist item to a gate or a documented manual step.

### E. Connector architecture
Provider-neutral core, one adapter per provider. Required adapters for
the workflow catalog: email intake/send (Graph/Gmail shapes), SMS/voice
notifications, payments (links/invoices/refunds/disputes), accounting
(OAuth2 + rotating refresh + sparse update + RequestId idempotency),
marketing (audiences, compliance gates, send/report), project-management
(query APIs + grant-key auth), calendar/drive file sync. Each adapter
contract declares: auth mechanism, capability deltas vs the neutral
core (explicit: what this provider cannot do), rate limits + backoff,
error taxonomy mapped to the effect protocol, sandbox-vs-production
endpoints, credential shape it consumes from the broker. Generic HTTP/
webhook/CSV adapters cover the long tail — with schema validation and
size/time bounds, never raw passthrough.

### F. Approval model
Framework-owned, non-bypassable. Lifecycle PENDING → APPROVED | REJECTED
| EXPIRED (terminal states immutable). Every approval binds: tenant,
material fingerprint (exact payload hash — approve-what-you-saw),
principal (owner/operator role, never the requesting agent: agents
propose, principals decide), scope + expiry, and decision receipt.
Draft→preview→confirm is the only path for customer-visible sends,
money movement, and external publishes. Expiry is enforced by the
runtime (stale approvals cannot execute late). Dual-approval routing
(service-credentials path vs summoned-owner path, per the reference
dual-path model) is preserved as two explicit authority kinds, never
collapsed. OWNER-DECISION: default expiry windows per capability class
(PROPOSAL: 24h standard, 1h money).

### G. Effect protocol
Every external call executes as: intent record (idempotency key bound to
tenant + capability + payload fingerprint) → attempt with bounded retries
(per manifest `RetryPolicy`, provider backoff honored) → recorded outcome
(success | failed-retryable | failed-terminal | UNKNOWN). UNKNOWN
(provider may have executed, outcome uncommitted) quarantines the
operation: no blind retry; reconcile via provider idempotency keys,
authoritative lookup, or operator evidence; resolve to a terminal state
before any dependent step proceeds. Provider outages degrade to queued
retry with operator-visible backlog, never silent drops. Duplicates are
impossible by construction (key collision returns the original receipt).

### H. Events, triggering, scheduling
Deterministic-first intake: provider webhooks where trustworthy (signed,
replayed-safe), UID/delta-tracked polling where not (the grasshopper
pattern), plus cron schedules with overlap guards (`no_agent`-equivalent:
skip when a prior run is live, never parallel-mutate). Schedulers emit
events into the orchestration layer; every trigger carries tenant,
idempotency key, and provenance. First-run flood control is mandatory
for every poller (seeded watermark + window + hard cap). OWNER-DECISION:
polling intervals per intake (defaults proposed per provider cost).

### I. Multi-tenancy and deployment
Tenant scoping enforced at contract admission (tenant in every
capability call), persistence (RLS + fail-closed context, per accepted
Studio doctrine), authorization (tenant-derived identity, cross-tenant
denial tested), and credential brokerage (tenant credentials never
ambient, never shared). Neon topology follows the locked doctrine
(Neon-only, verify-full; project-per-tier shape per the accepted ADR
track — the mission proposes no topology change). Environment ladder
dev → preview → staging → production with no shared mutable state or
secrets between rungs; promotion moves code + config contracts, never
data or credentials.

## 4. Secrets and environment contract (prompt34 §4)

### 4.1 Current-state audit (VERIFIED from code)
- Task layer: `Taskfile.dev.yml` vars carry paths + names only, never
  values; `dev serve-init` generates random 0600 secret files on first
  run and refuses to overwrite or implicitly rotate; `dev serve` assembles
  child-only env and execs.
- Go loader: `server/config.go` `_FILE` convention — each secret reads
  from `VAR` or `VAR_FILE` (exactly one), fail-closed on missing,
  misconfigured, or unreadable inputs.
- Production guard: every dev verb refuses platform-protected branches
  and `production`/`prod` names; bindings predating identity fail closed.
- Coverage: deployment credentials only. Tenant automation credentials
  have no contract yet — that is the gap this section closes.

### 4.2 Canonical variable inventory schema
PROPOSAL. One registry (checked-in schema, values never) with entries:
`{name, scope: deployment|tenant, sensitivity: secret|config,
rotation: static|rotating|session, consumers: [...], environments:
[dev, preview, staging, production]}`. Every variable the platform reads
must exist in the registry; unknown variables in a loaded file are
rejected (typo'd secret names fail closed, mirroring the binding
file's unknown-field rejection).

### 4.3 External `.env` layout
PROPOSAL. One file per environment, outside the repo, 0600:
`studio.<env>.env`. Strict `KEY=VALUE` subset (no interpolation, no
multiline, no `export`): parse with a validating reader (reject blank
keys, duplicates, unknown keys per §4.2, values over size bound).
Comment header records owner + rotation date. Values never echoed by
any tooling; a redaction helper scrubs `KEY=VALUE` shapes from
diagnostics before display.

### 4.4 Fly synchronization workflow
PROPOSAL, gated on verification: sync is a push-only, audited,
human-confirmed operation (`fly secrets set` per app, from the parsed
local file, diff-previewed before apply). Only mechanisms verified
against the installed Fly CLI/API surface may be scripted — each sync
step records its verification evidence (command + observed behavior).
Anything unverifiable is an OPEN DECISION with a verification procedure,
never an invented command. Rollback = re-push the previous recorded
revision (every sync records a receipt: app, keys-set, timestamp,
operator).

### 4.5 Taskfile integration
PROPOSAL. Extend the existing vars pattern: `ENV_FILE` var defaulting
per task family; new verbs `dev:secrets:validate` (parse + schema +
permission audit, zero values shown), `dev:secrets:diff <env>`
(against Fly, planned-changes only), `dev:secrets:sync <env>`
(confirmed push + receipt). Tasks pass file paths; Go owns parsing,
validation, and redaction. Conforms to the established doctrine:
Taskfiles orchestrate, Go implements. Every RPA CLI verb enforces the
same production guard as the dev verbs (refuse protected/prod bindings
before any provider call).

### 4.6 Unified configuration loader
PROPOSAL. One Go loader for the whole Studio (server, gateway, workers,
CLIs): precedence CLI flag > process env > `_FILE` > `.env` file >
compiled default (non-secrets only) > fail. Secrets may come ONLY from
process env, `_FILE`, or `.env` — never flags, never defaults, never
code. Startup fails closed listing missing/invalid keys BY NAME (never
values). Precedence conflicts (same key from two sources) fail closed
rather than silently preferring. Loader behavior is unit-tested per
source, including adversarial inputs (giant files, duplicate keys,
unknown keys, bad permissions — world-readable secret files refuse).

### 4.7 Deployment vs tenant credentials
Hard split, enforced by type and by storage:
- **Deployment credentials** (DB URLs, platform API keys, signing keys):
  loaded by the unified loader, ambient to the process, rotated by
  operators, never visible to agents or Minispecs, never in logs.
- **Tenant automation credentials** (one tenant's Stripe/QBO/Gmail keys):
  NEVER ambient. Held by a credential broker behind per-tenant,
  per-capability, expiring grants: an automation receives a
  single-use-or-short-lived lease bound to (tenant, capability, intent),
  audited at issue and at use; leases never cross tenants, never persist
  in automation state, and expire by default. OWNER-DECISION: broker
  storage backend (PROPOSAL: Neon with envelope encryption via a
  deployment-held KEK, reusing the existing Postgres-only doctrine;
  alternative: external vault — heavier, deferred absent a compliance
  driver).

### 4.8 Acceptance criteria (executable)
For each: a named test or procedure that fails before, passes after.
(1) Loader matrix: every source × every failure mode incl. adversarial
inputs. (2) Precedence conflicts fail closed with the key named.
(3) Missing secret fails boot naming the key, leaking no value (assert
on captured output). (4) Tenant lease cannot be used cross-tenant,
post-expiry, or twice (negative tests). (5) World-readable secret file
refuses. (6) Fly sync dry-run shows exact planned changes; confirmed
sync writes a receipt; rollback restores. (7) No secret bytes in any
log/audit/fingerprint artifact (redaction test with canary values).
(8) Unknown `.env` keys rejected. Items 6–7 against live Fly are FUTURE
implementation verification requirements (no live runs in this mission).

## 5. Roadmap and codebase mapping (prompt34 §5)

- **Framework**: capability contract shape, trust modes, authority
  principals, approval/idempotency declarations, A/B/C/D admission,
  ADR-001 durable-substrate gate — REUSED as specified; no framework
  change proposed. (VERIFIED: `contract/manifest.go`,
  `contract/policy.go`, `docs/adr/miniapps/adr001-template.md`.)
- **F0 (shared platform)**: tenancy/identity/authorization contracts,
  unified loader (§4.6), broker abstraction, provider-neutral connector
  core, effect protocol, approval runtime — PROPOSAL: these land in F0
  per the roadmap's shared-contracts rule, sized by the primitive
  catalog in §6.2.
- **Roadmap B R0–R8**: R0 consumes this document as its domain-architecture
  input (PROPOSAL, pending owner acceptance — the roadmap is read-only
  here); R1–R8 phasing unchanged, with the §6.16 roadmap attaching
  primitive delivery to existing gates. No renumbering, no new phases.
- **Roadmap A**: untouched. B-series never blocks on R implementation;
  shared engines stay in F0. No B-milestone text is modified.
- **Existing Studio code reused**: `_FILE` loader pattern, production
  guard, Neon/RLS persistence doctrine, Focus Console + B-workspace UI
  patterns (new scenes follow them; no new shell), connector registry
  shape (extended, not replaced), Taskfile/Go-split doctrine.

## 6. Deliverable record (prompt34 §6)

### 6.1 Platform domains
See §3.A. Five domains with the anti-giant-CRM guard (no cross-domain
operational writes; joins in read models on stable IDs).

### 6.2 Platform primitives (extracted catalog)
| # | Primitive | Contract essence | Failure / recovery | Owner |
|---|---|---|---|---|
| P1 | `party.contact.upsert` | Typed contact record, dedupe keys, consent state | Validation-terminal; duplicates merge by key, never fork | F0 |
| P2 | `commercial.estimate.create` | Pure pricing kernel: rate card × inputs → line items + assumptions | Invalid inputs terminal with reasons; deterministic replay | F0 |
| P3 | `commercial.invoice.issue` | Idempotent issue + send; links estimate | Provider UNKNOWN → quarantine + reconcile (§3.G) | F0 |
| P4 | `commercial.payment.collect` | Link/invoice/refund/dispute ops, idempotency keys, test/live separation | Failed-terminal on decline; disputes open evidence tasks | F0 |
| P5 | `commercial.spend.authorize` | Outbound virtual-card flow: push approval, short-TTL retrieve, immediate hygiene | Expiry → recreate; TTL-breach → quarantine, never reuse | F0 |
| P6 | `engagement.message.send` | Templated send with compliance gates (opt-in, unsubscribe) + report-back | Compliance violation blocks pre-send; provider errors per §3.G | F0 |
| P7 | `engagement.intake.poll` | UID/delta-tracked polling + watermark + first-run flood control | Overlap skip; poison message quarantined after N | F0 |
| P8 | `engagement.review.request` | Post-job review ask with timing rules + response capture | No-response expires silently (non-blocking) | F0 |
| P9 | `workforce.job.sync` | External PM bidirectional sync with conflict rule (platform-wins-on-explicit-fields) | Conflict → operator task with both versions | F0 |
| P10 | `workforce.briefing.compile` | Deterministic digest assembly from jobs/messages/calendar | Missing source degrades section with reason, never fabricates | F0 |
| P11 | `ledger.receipt.append` | Append-only event + decision + effect receipts | Write failure fails the enclosing operation (no silent loss) | F0 |
| P12 | `agent.propose` / `agent.confirm` | Proposal/preview/confirm shapes for the interaction contract | Malformed proposal rejected at admission | F0 |
| P13 | `tenant.credential.lease` | Brokered short-lived grants (tenant, capability, intent, expiry, audit) | Expired/foreign lease refused; use logged | F0 |
| P14 | `schedule.cron.manage` | Cron definitions with overlap guard + provenance + idempotency keys | Overlap skips with receipt; missed runs alert, never backfill silently | F0 |

Each primitive: approval class (human-gated for P3–P6 sends/money),
idempotency key scope, retry bounds, and tests per §6.9. All PROPOSAL.

### 6.3 Execution flow control
Orchestration = staged capabilities + persisted stage receipts +
framework admission. No central engine without an approved ADR-001.
Crash anywhere replays from the last receipted checkpoint; retries
honor manifest bounds + provider backoff; duplicates resolve to the
original receipt. UNKNOWN outcomes quarantine per §3.G. Human gates
interrupt deterministically (PENDING blocks dependents; EXPIRED kills
the branch).

### 6.4 State models
- Records: per-domain schemas with lifecycle states (e.g. estimate
  draft→sent→accepted|expired; invoice issued→paid|void; campaign
  draft→scheduled→sent→reported).
- Approvals: PENDING → APPROVED|REJECTED|EXPIRED, terminal immutable,
  material-bound (payload fingerprint), principal-separated.
- Effects: intent → attempt(s) → success|failed-retryable|
  failed-terminal|UNKNOWN→(reconciled terminal).
- Pollers: watermark + window + cap + poison-quarantine.
- Leases: tenant + capability + intent + expiry + single-use flag.
- Receipts: append-only, content-addressed where verifiable.
- State-authority rule (no triplication ambiguity): the provider is
  source of truth for remote state; domain records are caches carrying
  provenance + TTL; receipts are audit-only and MUST never be read for
  current state. Any read path violating this fails review.

### 6.5 Credential & configuration architecture
Per §4: canonical inventory schema (§4.2), external `.env` layout
(§4.3), verified-only Fly sync with receipts and rollback (§4.4),
Taskfile verbs (§4.5), unified Go loader with explicit precedence and
fail-closed matrix (§4.6), deployment-vs-tenant split with brokered
leases (§4.7), and the eight executable acceptance criteria (§4.8).

### 6.6 Multi-tenancy strategy
Per §3.I: admission-level tenant binding, RLS + fail-closed context,
tenant-derived identity with tested cross-tenant denial, brokered
tenant credentials (never ambient), Neon-only persistence under the
locked doctrine, environment ladder with no shared mutable state or
secrets. No single-tenant shortcuts anywhere in shared code.

### 6.7 Agent/owner experience contract
Two renderings of one contract. The agent speaks proposal → preview →
confirm: every consequential action is first a typed proposal the owner
can inspect, then an explicit confirmation bound to the exact material.
The visual workspace (Automation Operations Studio) renders the same
states: queues, approvals, runs, receipts, drill-downs. Provider
mechanics (Telegram vs web UI) never leak into capability semantics.
Honesty rule inherited from the cheatsheet builder: the workspace shows
wired capabilities with live status; planned items are labeled Planned
with reasons, never shown as working.

### 6.8 Change strategy
- Reuse without change: framework contracts (capability/trust/authority/
  approval/idempotency shapes), A/B/C/D admission, ADR-001 gate,
  `_FILE` loader pattern, production guard, Neon/RLS doctrine.
- Extend: connector registry (provider adapters + capability deltas),
  Taskfile verbs (secrets lifecycle), UI scenes (new workspaces, same
  shell), broker abstraction (new, F0-owned).
- Propose new (owner-gated): F0 primitive implementations (§6.2),
  unified loader consolidation, lease/audit persistence, scheduler
  runtime. Class D adoption requires its own ADR-001 — explicitly not
  assumed.

### 6.9 Test strategy
Per primitive: contract tests (schema/admission/versioning), failure
injection (crash mid-stage, duplicate delivery, provider 429/500,
UNKNOWN reconciliation), authz matrix (tenant isolation, principal
separation, expired-lease refusal), idempotency replay proofs, approval
lifecycle (incl. expiry enforcement), scheduler overlap + flood-control
proofs, secret-leakage canaries (logs, errors, fingerprints, receipts).
Connectors: provider sandbox suites where offered, else scripted
doubles asserting the mapped error taxonomy. All red-before/green-after;
no mock-green (doubles assert wire behavior, never mirror the code).

### 6.10 Risks, tradeoffs, non-goals
- Orchestration without a durable engine risks hand-rolled state
  machines; mitigated by staged receipts + the ADR-001 escape hatch
  kept genuinely open (not a pretextual no).
- Broker is new security-critical surface: envelope encryption + audit
  + negative tests are release blockers, not follow-ups.
- Provider drift (API changes, TTLs, rate limits): adapter contracts
  version independently; deltas re-verified on a schedule.
- Scope creep into vertical features: the platform ships primitives +
  one vertical slice; further verticals are Minispecs, not platform.
- NON-GOALS: giant CRM, canvas builder, billing engine, UI-RPA/screen
  scraping, Python/Hermes runtime dependency, hardcoded providers,
  reopening locked roadmap decisions.
- Portability (anti-lock-in): receipts and domain records export as
  plain JSON/CSV on demand; no proprietary encoding is load-bearing;
  provider adapters wrap public APIs only, so any adapter is replaceable
  without touching domains or workflows.

### 6.11 Placement table
| Primitive/system | Framework | F0 | Adapter | Miniapp | Studio UI | Secrets contract |
|---|---|---|---|---|---|---|
| Capability/trust/authority shapes | owns | uses | — | declares | — | — |
| P1–P14 implementations | — | owns | — | — | — | — |
| Provider adapters (QBO/Graph/Stripe/Mailchimp/PM/voice) | — | core iface | owns each | — | — | consumes leases |
| Approval runtime | owns model | operates | — | requests | renders queue | — |
| Effect/retry/UNKNOWN handling | — | owns | maps errors | — | renders backlog | — |
| Schedulers + intake | — | owns | poll impls | — | renders runs | — |
| Credential broker + leases | — | owns | — | — | — | contract owner |
| Unified loader + Taskfile verbs | — | owns | — | — | — | contract owner |
| Vertical slices (lead-to-payment etc.) | — | — | — | owns composition | renders scenes | consumes |
| Attestations + receipts | schema | operates | — | emits | renders | — |

### 6.12 Dependencies
F0 contracts → adapters → vertical slices → workspaces; broker before
any live-money primitive; scheduler runtime before intake-dependent
slices; unified loader before Fly sync; ADR-001 (if pursued) before any
Class D declaration. Cross-cutting: test strategy lands with each
primitive, not after. Nothing in R blocks Roadmap A work.

### 6.13 Roadmap deltas (PROPOSALS — roadmap is read-only here)
- R0 consumes this document as domain-architecture input (no R0 text
  change proposed beyond attachment).
- F0 work-breakdown gains: broker abstraction, unified loader
  consolidation, lease/audit persistence, scheduler runtime — sized at
  implementation planning, not here.
- No new phases, no renumbering, no B-series changes, no locked-decision
  changes. Any conflict discovered during implementation returns as an
  OWNER-DECISION, never a silent edit.

### 6.14 Go contract sketches (illustrative, not implementations)
```go
// Capability invocation: tenant-bound, idempotent, admission-checked.
type InvokeRequest struct {
    TenantID  string          // server-derived, never caller-supplied
    Capability CapabilityRef  // provider/operation/version pin
    Authority policy.Authority // user | service | trigger principal
    Payload   json.RawMessage // validated against the capability schema
    IdempotencyKey string     // bound to tenant+capability+payload hash
}

// Approval lifecycle: terminal states immutable, material-bound.
type ApprovalState string // PENDING | APPROVED | REJECTED | EXPIRED
type Approval struct {
    ID string; TenantID string; State ApprovalState
    MaterialFingerprint string // hash of the exact approvable payload
    RequestedBy string; DecidedBy string // never equal: agents propose
    Scope string; ExpiresAt time.Time; Receipt string
}

// Effect outcome: UNKNOWN quarantines, never blind-retries.
type EffectOutcome string // success | failed_retryable |
                          // failed_terminal | unknown
type EffectRecord struct {
    IntentID string; Attempt int; Outcome EffectOutcome
    ProviderRef string; ReconciledBy string // for ex-UNKNOWN records
}

// Credential lease: short-lived, single-purpose, audited.
type Lease struct {
    ID string; TenantID string; Capability string; IntentID string
    ExpiresAt time.Time; UsesLeft int; IssuedAt time.Time; AuditRef string
}
```

### 6.15 End-to-end mapping (three workflows)
**W1 lead-to-payment (canonical).** Intake P7 (email/voice pollers) →
P1 contact upsert (dedupe) → P2 estimate (rate card) → F approval
(owner previews exact estimate) → P6 send → P3 invoice on acceptance
→ P4 collect (link/invoice) → P8 review ask → P11 receipts at every
stage. Human gates: estimate send, invoice send, payment-link send.
Crash-safe via stage receipts; UNKNOWN provider outcomes quarantine
the affected stage only.

**W2 missed-call → booked job (cross-domain).** P7 voice intake
(transcription + urgency triage) → P1 contact → P12 proposal drafted
→ F approval (callback script preview) → P6 callback send →
calendaring handoff → P8 review. Demonstrates polling-adapter +
approval + engagement composition without payments.

**W3 campaign-to-cash (cross-domain).** Audience select (P1 segments)
→ P6 templated compose → F approval (exact content + recipient count)
→ P6 scheduled send → delivery/report-back → P4 collect on
responders → P11 full-funnel receipts. Demonstrates compliance gates
(opt-in/unsubscribe enforced pre-send) + marketing/finance join.

Coverage of all eight §2 workflows (no orphan primitives, no unmapped
workflow): lead-to-payment → §6.15 W1; missed-call→job → W2; campaign
→ W3; estimate→invoice → P2+P3+F; payment-link → P4+F; review-request
→ P8; voicemail monitor → P7; daily ops briefing → P10+P7+P11.

### 6.16 Dependency-ordered roadmap
1. Contracts + loader + guard rails (capability schema, unified
   loader, broker abstraction design, approval runtime).
2. Ledger + receipts + attestations (evidence before effects).
3. Intake + scheduling (pollers, cron, flood control, overlap guards).
4. Domain primitives P1/P2/P6 (read/compute + one gated send).
5. Money primitives P3/P4/P5 behind live approvals + broker leases.
6. Connectors per vertical need (QBO/Graph/Stripe/Mailchimp/PM/voice).
7. First vertical slice: W1 lead-to-payment end to end.
8. Operations Studio scenes + agent channel on the same contract.
9. Harden: failure injection suites, drift re-verification, scale review.

### 6.17 Traceability
Every primitive traces: P1←crm-lite; P2←estimator-engine; P3←
quickbooks-online+estimator; P4←stripe-payments; P5←stripe-link-cli
(+website-spec as second consumer); P6←mailchimp-integration+
marketing-campaign-builder; P7←grasshopper-monitor+outlook-graph;
P8←lead-to-payment stage 8; P9←construction-agent; P10←construction-
agent+financial-analysis; P11←lead-to-payment receipts+hermes-ops;
P12←lead-to-payment stages+cheatsheet honesty; P13←oauth2-flow,
token-cache, Link TTL handling; P14←cron-script-arguments. Domains
← skill clusters; approvals ← draft→preview→confirm + Hermes dual
paths; UNKNOWN handling ← Link error-fixes + QBO retry taxonomy;
tenancy inversion ← roadmap doctrine (not the reference); rollout
staging ← transformation methodology. Unresolved: NOTHING is
claimed from files not read — full inventory in §1.

## 7. Adversarial review dispositions (prompt34 §8)

Fourteen challenges exercised; five findings fixed in this document,
nine reviewed clean. No blocking findings remain open.

FIXED IN DOCUMENT:
- F1 over-abstraction (material): P10 had no mapped workflow → added
  §6.15 eight-workflow coverage list proving no orphans.
- F2 state duplication (material): added the state-authority rule to
  §6.4 (provider = truth, domain = cache with provenance+TTL, receipts
  audit-only, never read for state).
- F3 approval safety (material): dual-approval routing was implicit →
  explicit two-authority-kinds rule in §3.F.
- F4 lock-in (non-blocking): added portability bullet to non-goals.
- F5 crossover (non-blocking): RPA CLIs explicitly bound to the same
  production guard (§4.5).
REVIEWED CLEAN (no change required): failure contracts (crash/retry/
duplicate/UNKNOWN all specified with tests); isolation (tenant/RLS/
principal rules + negative tests); engine adoption (ADR-001 gate kept
genuinely open); idempotency (scoped keys, collision semantics,
cross-tenant binding); premature building (MVP discipline per §8
below); leakage/precedence (fail-closed matrix + canary tests);
unsafe sync (verified-only + receipts + rollback); secret confusion
(deployment/tenant hard split + lease semantics); roadmap conflicts
(no renumbering, no B changes, no reopened locks verified).

## 8. Handoff (prompt34 §10)

- **Minimum primitive set:** P11 receipts + P1 contacts + P7 intake +
  P13 leases + F approvals + G effects. Nothing executes externally
  before these six exist.
- **Slice target:** W1 lead-to-payment end to end on real adapters,
  then W2 (no-payments composition proof), then W3 (compliance proof).
- **Initial capability sequence:** contracts/loader/guard → ledger →
  intake/scheduling → P1/P2/P6 → money primitives behind live approvals
  + leases → connectors per vertical → W1 slice → workspaces → harden.
- **Working config workflow:** canonical inventory → per-env 0600 files
  → validating reader → Taskfile verbs (validate/diff/confirmed-sync
  with receipts) → unified Go loader → brokered tenant leases; Fly sync
  verified-only, rollback by recorded revision.
- **Decisions for the owner (defaults proposed, non-blocking):** Class D
  need for slice one (default no); approval expiry windows (24h/1h);
  poller intervals per intake; broker storage backend (Neon envelope
  encryption); R0 attachment of this document.
- **Review dispositions:** five fixed above; zero blocking open.
- **Future implementation verification requirements (no live runs were
  performed in this mission):** Neon branch-protection marker behavior
  on the live project; Fly CLI/API sync mechanics per §4.4; provider
  sandbox suites per adapter; red-before/green-after per §6.9.
