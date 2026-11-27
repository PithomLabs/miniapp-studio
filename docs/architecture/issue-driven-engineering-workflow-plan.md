# Issue-Driven Engineering Workflow Plan — Miniapp Studio

- Status: PROPOSED (planning-only; no implementation authorized)
- Rev R2 (review_prompt27.1 response): OAuth credentials, write-path
  split, B2-01A/B local-first, restore authorization, B2-09
  Discussions card, corrected dependencies, AGENTS.md reconciliation
  proposal, Appendix A retitle. All other accepted work preserved.
- Rev R3 (review3 response): Sprite-side Connector reads, demonstrated
  read-only boundary with fallback, verified-ops-only + conditional
  demo beats, Studio-controlled B2-01B with separately authorized
  destruction, Preview access policy, App permission matrix,
  AGENTS.md stop rule, owner-created cards, non-destructive
  capture wording. All other accepted work preserved.
- Rev R4 (review4 response): split Connector-denial vs broker-
  enforcement proofs; merge boundary with no-replay/no-bypass
  tests; B2-10 Studio-side read-only fallback (Option B) with
  acyclic dependencies; safe-probe procedure; B2-03/B2-08/B3-01
  dependency updates. All other accepted work preserved.
- Source directives: `plans/prompt27.1.md` (governing) executing
  Prompt27-R1 (`plans/prompt27.md`, Discussion #2); newest accepted
  decision wins on any conflict.
- Execution record: `plans/prompt27.1_plan.md` (P0–P6 preserved).
- Research: Prompt27.1 sweep 2026-11-26 (b0b1-impl, roadmap-docs,
  connector-auth, console-ci, github-side + synthesis); author
  first-hand corroboration where noted.
- Working-tree baseline: clean at `eb900d6` + 5 plan entries; this
  document is the single permitted new file.

Evidence tiers used throughout (never conflated):

| Label | Meaning |
| ----- | ------- |
| Built | Inspected implementation + test bodies in this tree |
| API-verified | Observed via live GitHub API GET in this session |
| Web-fetch-verified | Observed via persisted public page fetch, not live API |
| Screenshot-verified | Owner screenshot baseline (prompt27.1 §3), never API-checked here |
| Discussion-stated | Claimed in Discussion #2 text; observed evidence only |
| Proposed | Designed here, not built |

---

## 1. Product vision, architecture and end-to-end workflow

**North star: Miniapp Studio — The Self-Developing Software Engineer.**
A personal AI software engineer living inside its own persistent,
recoverable computer. Miniapp Studio directs its work, Fly Sprites
give it a secure place to execute, and Connectors give it controlled
access to the outside world. (Discussion #2, web-fetch-verified.)

The product is NOT an IDE chatbot, a generic agent platform, a
GitHub Issues clone, a Fly dashboard, or a conventional dev/deploy
platform. It is an AI engineering environment where a human owner
directs a resident agent, provides authorization, and receives
verifiable outcomes.

### Component responsibilities

| Component | Role | Status |
| --------- | ---- | ------ |
| Miniapp Studio (local) | Command center: tasks, conversations, approvals, decisions, evidence, history; supervises agents; durable authorization record | Built (B0/B1 + Console) |
| Builder Sprite (Fly) | Resident agent's computer: repo, deps, Linux toolchain, persistent workspace, checkpoint/restore, code/test/repro; mutable Workspace Preview URL | Planned (B2) |
| Sprites Connectors | Controlled external access (GitHub first): repo context, Issues, Discussions, PRs, Actions evidence where supported; credentials stay in the connector control plane | Partial (4-op Custom API surface built; GitHub via OAuth Connector planned, B2-02 — never PATs) |
| Fly Machines | Application runtime: formal Preview, Staging, Production; immutable digest-bound artifacts | Planned (B3/B5) |
| Neon | Per-environment databases: local PG for dev, isolated Preview branch, protected Staging/Production branches | Planned (B3/B5) |

The agent workspace and the application runtime remain distinct. A
Sprite is not Staging or Production.

### Two previews (binding distinction)

| Preview | Infrastructure | Meaning |
| ------- | -------------- | ------- |
| Workspace | Sprite URL | Mutable app inside the dev computer; quick inspection, never release proof |
| Application | Fly Machine + Neon branch | Immutable candidate under deployment verification |

### End-to-end workflow

Owner briefs in Studio → agent works in the Builder Sprite →
checkpoints persist → Workspace Preview shows live state →
Builder Sprite agent reads incident/CI evidence through the
genuine Connector and returns sanitized evidence to Studio →
bugs become authorized linked Issues → fixes land via authorized
PRs → B3+ promotes immutable artifacts through Preview with
per-environment approval. Every consequential step carries an
attributable, durable, scoped human authorization (§3); the board
and the chat never authorize anything.

---

## 2. GitHub Issues and Discussions intake and classification

Binding division (prompt27.1 §2, newest owner decision — supersedes
prompt27's broader Issues scope and kanban.md's early Parent-Issue
sketch wherever they differ):

| Record | Holds | Examples |
| ------ | ----- | -------- |
| GitHub Discussion | Vision, architecture, decisions, questions, roadmap, milestones, retros, unscoped ideas | North Star, B2 plan review, incident retrospective |
| GitHub Issue | Possible/confirmed bugs ONLY | CI failure defect, reproducible crash |
| Project Task Card (Draft, not a repo Issue) | Implementation work packages with status/owner/evidence | B2-01A Sprite lifecycle contracts |
| Pull Request | Code changes | Repair for the linked defect |
| Miniapp Studio PG | Execution state, approvals, evidence | Attempts, checkpoints, decisions |

No feature Issues, no Parent-Initiative Issues. Simple localized
problems stay inside the existing task and never become records.

### Intake flows

**Discussion intake:** agent (or owner) identifies durable
architectural/strategic/milestone value → agent drafts content,
selects an existing category → presents purpose/content/category →
owner authorizes → agent creates through the approved integration →
agent reports URL + identifier. No silent creation, no invented
categories: category IDs come from live inspection (GraphQL), and
the plan records the one observed category, `Ideas` (Discussion #2
meta, web-fetch-verified), with the full list explicitly unverified
from here (REST has no categories endpoint — verified 404; GraphQL
needs working auth).

**Issue intake:** agent determines a problem is complex, scoped, and
needs independent tracking → proposes a linked Issue carrying
observed behavior + evidence, repo/commit/context, reproduction or
its limits, suspected-vs-confirmed cause, impact, acceptance
criteria → owner authorizes → agent files → agent reports URL + id.
Discussion-to-Issue transition preserves the Discussion as the
reasoning record, links the Issue, never double-creates without a
stated reason.

**Task Cards:** owner-created Draft items on Project #3 carrying the
§6 work packages; agents update status/evidence only through
authorized operations, never by editing board policy. For the
initial workflow the owner creates all cards manually; any Project
V2 API integration is a separate, deferred, explicitly authorized
capability — never assumed present.

**Milestone Discussion (per milestone):** what was intended, what
shipped, supporting evidence, deviations, unresolved risks, next
milestone. Proposed but never published by this task.

---

## 3. Authorization boundaries and refusal behavior

Standing principle: agents recommend, investigate, prepare, and
report. They never perform consequential actions without Project
Owner authorization. A recommendation is never authorization; a
direct instruction authorizes the exact action requested and
nothing else. The agent stops when the next consequential action
is outside its approval scope.

### Lifecycle (every consequential action)

Propose → draft (purpose, content, category/target) → owner
authorizes → execute through the approved integration → report
URL/identifier/receipt → subsequent actions re-authorize in their
own scope. Repair ≠ PR ≠ merge ≠ deploy; Preview ≠ Production.

### Coverage matrix (existing mechanism → gap)

| Action class | Existing enforcement (Built) | Still needed (Proposed) |
| ------------ | ---------------------------- | ----------------------- |
| Read GitHub (issues, discussions, runs, logs) | None (no read surface) | Read tools via Connector; task-level GO |
| Create/edit/comment/close Issues & Discussions | None | Studio-side GitHub App writes only, per-action owner GO; Sprite stays read-only |
| Write code / change workspace | Session+Origin+opkey on Console ops; attempt/checkpoint records | Sprite-side writer identity + WIP-1 enforcement |
| Checkpoint capture | B1 capture op (task-scoped) | Unchanged: capture stays task-scoped |
| Checkpoint restore | None (B1 has no remote restore) | Explicit per-restore owner GO bound to Sprite + checkpoint + target workspace + expected effect; pre-restore state preserved |
| Push branches, create PRs | None | Per-push/per-PR owner GO via the GitHub App (permission matrix in B2-08, incl. Contents); branch naming + scope check |
| Merge | None | Separate owner GO after green review; App executes, never the agent directly |
| Fly/Neon reads | CLI discovery (versions pinned) | Supervised agent invocation rules |
| Fly/Neon mutations, deploys, migrations | Op-class doctrine only | Infra adapter + per-op approval binding digest/target/env |
| Destructive (branch/DB/app/delete) | Deny-by-default doctrine | Exceptional auth + safeguards, or continued denial |

### Credential boundaries

Fly's GitHub OAuth Connector holds credentials in the Fly control
plane and injects them when forwarding requests: the Builder Sprite
never receives raw tokens. Sprite access is denied by default and
granted only to the designated Builder Sprite; actual GitHub scopes
and accessible operations are verified live during B2-02. No PAT
may live in Sprite files, environment variables, shell history, or
agent configuration; the deleted classic PAT stays deleted. There
is no silent fallback to ordinary authenticated `gh` CLI access —
any non-Connector path requires its own explicit authorization.

### Write-path split (GitHub mutations)

The Sprite agent must not be able to bypass Miniapp Studio: the
Sprite Connector grants READ-ONLY GitHub access (repository, CI,
Issues, Discussions, and other evidence). All GitHub writes are
performed by a trusted Studio-side integration using a narrowly
permissioned GitHub App (separate Issues, Discussions, and
pull-request grants, least privilege). Every write binds the exact
approval, target record, and operation at the server boundary.
Two separate negative proofs are required and must not be
conflated: (1) the Connector-denial proof (B2-02) — the Sprite
cannot mutate GitHub through Fly's Connector; (2) the
broker-enforcement proof (B2-08) — Studio refuses the unapproved
write before the GitHub App executes it, even with Connector read
access in hand. Approving a proposal never implies approving
its execution.

### Approval records

Approvals persist in Studio PG (`studio.approvals` pattern:
attributable decider, timestamp, scoped target, decision, note),
extended per slice to bind the authorized target (record URL/ID,
workspace, environment, artifact digest, migration set where
applicable). Same-outcome replay returns the receipt; changed
scope requires a new approval; expiry/revocation deny by default.
Refusal renders the reason and the exact unblocking action.

### Refusal behavior

Missing/expired/mismatched authorization fails closed with a named
reason at the server boundary — never a prompt, a hidden button, or
a board field. The Project Authorization column is a view over
Studio records, never the permission system: Ready ≠ authorized.
Agent self-authorization (answering its own approval, reusing one
approval for another action) is a violation to refuse and report,
not a state to represent.

---

## 4. Project memory, milestone governance and B2–B5 integration

### Memory hierarchy (no duplication, no second tracker)

| Layer | Holds | Authority |
| ----- | ----- | --------- |
| GitHub Discussions | Deliberation, provenance, milestone narratives, links | Reasoning record (never auto-decision) |
| Repository docs | Accepted architecture, specs, roadmap, plans, evidence | Canonical on acceptance |
| GitHub Issues (+ PRs) | Defect backlog, code changes | Work tracking |
| Studio PG | Execution state, approvals, evidence, history | Runtime truth |

A Discussion becomes an accepted decision only when distilled into
a canonical repo document after owner approval. Milestones close
with a retrospective Discussion; full contents are never copied
into the repo.

### Context recovery (minimal process)

1. Read the charter (SKILLS.md) and this plan's §4 reconciliation notes.
2. Read the accepted roadmap and ADRs.
3. Inspect milestone/decision Discussions and linked Issues/PRs.
4. Inspect Studio evidence for the claimed state.
5. Verify repository state live; report conflicts before proposing.

### B2–B5 integration (reconciled, newest-wins)

- **B0/B1 accepted as built** (packet v0, attempts, checkpoints,
  resume, Console, runbook, evidence). No redesign.
- **B2** extends SKILLS handoff-only scope with the owner-directed
  resident agent + genuine Connector + issue/discussion intake
  (prompt27 §6 governs). Workspace Preview (Sprite URL) belongs
  here; it is inspection, never release proof.
- **B3** owns formal Application Preview (Machine + Neon branch,
  build-once OCI chain) and records the ADR-001 mechanism
  amendment (Sprites stay dev/Workspace-only).
- **B4** owns troubleshooting/repair loops on real telemetry.
- **B5** owns Staging/Production governed release; full release
  management is explicitly not a Hacksgiving blocker.
- Resolved conflicts: B0–B5 labels (Rev2/SKILLS win); bugs-only
  Issues + Draft Task Cards (prompt27.1 wins over prompt27's
  broader table and kanban's early sketch); Machines for
  Preview/Prod with ADR-001 amendment pending (B3 records it);
  demo = Sprite-resident incident repair on run 45666647515 with
  Studio-itself dogfood as the enduring frame. Open: ADR-002
  (MPG PG17) vs Neon-per-env needs an explicit owner amend-or-
  retire decision.
- AGENTS.md reconciliation (proposed, owner to approve — the file
  itself is untouched by this plan): (a) §2.1 line 58, "An
  approved application runs in a separate Runtime Sprite" →
  amend to Machines for Preview/Production with Sprites retained
  for development and Workspace Preview, coordinated with the B3
  ADR-001 amendment; (b) §2.4 line 90 P5 golden-path scope and
  §10 line 396 Phase B scheme → mark as completed-program
  history, with B0–B5 naming all forward milestones; (c) §8
  lines 346–359 PG19 @ 127.0.0.1:5434 → NOT in conflict, retained
  verbatim as the mandatory local baseline (Neon applies to
  Preview/Staging/Production only). Until the owner approves the
  amendments, this Proposed plan governs nothing: any
  implementation task encountering one of these conflicts must
  stop and request an owner decision. Nothing here silently
  supersedes the existing governing instructions.

---

## 5. Minimal data, Connector/API and Focus Console changes

Reuse-first: extend, never duplicate. All changes below are
proposed; nothing here is built.

### Data (minimal deltas, migration discipline as built)

- External-record linkage: the smallest durable binding between
  Studio rows (builds/attempts/approvals) and GitHub records
  (issue/PR/discussion/run URLs + IDs + SHAs) — proposed shape
  decided at implementation (linkage table vs columns), same RLS/
  grant/parity discipline as migrations 000023/000024.
- Approval scope extension: bind target record workspace,
  environment, artifact digest, migration set where applicable
  (pattern: `studio.approvals` + digest columns).
- No credential storage anywhere: Connector tokens stay in the
  Sprites control plane / server-side secret state, exactly as
  today.

### Connector/API (genuine operations only)

- GitHub access through Fly's built-in OAuth Connector (B2-02),
  reusing the existing list/test/attest patterns where they fit;
  document precisely which operations the Sprites Connector
  performs vs ordinary API calls — no silent substitution, ever.
  Custom-API PAT connections are not an acceptable substitute.
- Gaps to close in slice order: Sprite-side read surface
  (issues, runs, logs — reads execute in the Sprite, Studio
  persists evidence) before any write; OAuth token lifecycle
  design; Get/Delete connection API+UI; live scope/operation
  verification; surfacing usage guidance to agents;
  approval→effect binding so an approval gates the Connector
  call.
- Category handling: IDs from live GraphQL inspection only;
  never assume, create, or rename categories.

### Focus Console (extend, never redesign)

- New cards/routes follow the proven pattern: scene entry +
  template, fact-shaping branch, per-intention opkeys, redirect;
  new Pulse kinds get a branch + CSS class + filter entry.
- Proposed surfacing: issue proposal drafts with authorize/refuse
  actions, approval receipts with scope, agent progress + evidence
  links, incident/runner views fed by Connector reads. Spine
  derivation rules and guardrails (Run never green) unchanged.

---

## 6. Dependency-ordered implementation slices

Seven fields per slice: execution location · application target ·
database target · infrastructure actions · authorization class ·
verification evidence · recovery/rollback. All slices are proposed;
priorities below are initial proposals for owner confirmation.

### B2-01A — Sprite lifecycle contracts, local (FIRST TASK)

- Source: review R2-3 (Local-first split). Scope: implement and
  test the complete Sprite lifecycle contract locally:
  deterministic adapters/fakes for provision/wake/sleep/connect/
  disconnect, receipt shapes, authorization and refusal behavior
  for each transition. No live Fly contact of any kind.
  Exclusions: live API calls (B2-01B), agent runtime (B2-05),
  Preview URLs (B2-07), fleets. Deps: none (first).
  Workstream B2 · Priority P0 · Executor: Builder.
- Location: local workstation + local PG only. App: none. DB:
  local PG (contract-test fixtures only). Infra: none. Auth:
  task-level GO (no cloud surface exists). Evidence: contract
  tests green incl. refusal paths, adapter/fake parity proof.
  Recovery: N/A (nothing provisioned).

### B2-01B — Live Fly Sprite lifecycle proof

- Source: review R2-3 (Local-first split) + review3 R3-4. Scope:
  conduct the explicitly authorized live proof THROUGH Miniapp
  Studio's own lifecycle operations: provision one persistent
  Builder Sprite, prove wake, sleep, connect, disconnect via the
  Fly adapter with durable receipts and server-side authorization
  enforcement; record the Sprite as the agent's computer. An
  owner-terminal bootstrap is a separately identified prerequisite
  only — it never counts as proof of Studio-managed lifecycle.
  Exclusions: everything in B2-01A (already proven), agent runtime
  (B2-05), Preview URLs (B2-07), fleets. Deps: B2-01A genuinely
  complete. Workstream B2 · Priority P0 · Executor: Owner hands
  to Builder.
- Location: Studio lifecycle ops → Fly adapter → Fly API (owner
  terminal only for the separately authorized bootstrap). App:
  none (infra only). DB: local PG (lifecycle receipts). Infra:
  Sprite create/delete, power operations. Auth: per-operation
  owner GO; destruction carries its own explicit authorization
  bound to the exact Sprite and effect; reads task-level.
  Evidence: Sprite IDs, adapter-driven state-transition receipts
  with approval refs, wake-latency notes. Recovery: destroy (own
  authorization) + reprovision; no app state at risk.

### B2-02 — GitHub Connector via Sprites (genuine operation)

- Source: Discussion #2 §Connector + §5 gaps + reviews R2-1/R3-2.
  Scope: GitHub access through Fly's built-in GitHub OAuth
  Connector (preferred; credentials stay in the Fly control plane
  and are injected at request-forwarding time — the Sprite never
  receives raw tokens). Deny all Sprites by default; authorize
  only the designated Builder Sprite. Then PROVE the boundary:
  enumerate every GitHub endpoint and operation the Sprite needs;
  test the actual authorized Connector (never a fake or direct
  API substitute); attempt unauthorized writes through it,
  including on routes also used for reads; record allowed vs
  refused per operation. Naming it "read-only" without this
  matrix proves nothing — Fly documents path patterns, not HTTP
  methods, and one route may serve both reads and writes. Never
  store a PAT in Sprite files, environment, shell history, or
  agent configuration; the deleted classic PAT stays deleted. No
  silent fallback to ordinary authenticated `gh` CLI access.
  Fallback rule: if read-only cannot be demonstrated, restrict
  the Connector to demonstrably safe operations and move affected
  reads behind a Studio-side read-only integration. Exclusions:
  token rotation design (follow-up card), provider catalog UX.
  Deps: B2-01B. Workstream B2 · Priority P0 · Executor: Owner +
  Builder.
- Location: Studio Console + Sprites API for the grant; Builder
  Sprite for the boundary tests. App: none. DB: none (no
  credential storage anywhere in Studio) beyond the verification
  ledger rows. Infra: OAuth Connector grant + Builder-only Sprite
  authorization; no app resources. Auth: per-action owner GO
  (grant/verify/probe-writes are scoped writes). Evidence:
  attested connection record, verified scope list, per-operation
  allowed/refused matrix from live probes against the ACTUAL
  authorized Connector (this card owns the Connector-denial
  proof: the Sprite cannot mutate GitHub through Fly's Connector;
  broker enforcement is proven separately in B2-08), sanitized
  test transcript, Connector-vs-direct call ledger. Probe safety:
  all unauthorized-write attempts run against a disposable test
  repository or explicitly authorized canary resources with a
  defined cleanup path; the probe stops on the first unexpected
  successful mutation; cleanup carries its own authorization.
  Recovery: revoke grant / remove Sprite authorization; nothing
  deployed.

### B2-03 — Read-only GitHub operations surface (Sprite-side)

- Source: connector-auth gap (b) + review R3-1/R3-3. Scope: the
  Builder Sprite agent performs list/read operations through the
  Fly GitHub Connector; Studio receives and persists sanitized
  evidence — Studio is never the Connector caller. Execution
  path: `Builder Sprite agent → Fly GitHub Connector → GitHub →
  sanitized evidence returned to Studio`. Only operations live-
  verified under B2-02 may be promised; anything unverified
  (Discussions/Actions runs/jobs/logs until proven — the
  published guide documents repositories, Issues, and pull
  requests only) proceeds solely via B2-10, the defined Studio-
  side read-only fallback. Exclusions: all writes (separate
  cards + approvals). Deps: B2-02 (primary path), B2-10
  (fallback branch only). Workstream B2 · Priority P0 ·
  Executor: Builder.
- Location: Builder Sprite via Connector; Studio records. App:
  Studio itself (dogfood reads). DB: local PG (evidence rows
  only). Infra: none. Auth: task-level GO for reads; no mutation
  surface exists. Evidence: Sprite-side read transcripts against
  known records (run 45666647515 metadata), per-operation
  verification status, refusal tests for writes. Recovery: N/A
  (read-only); revoke connection on suspicion.

### B2-04 — Incident intake from run 45666647515

- Source: Discussion #2 demo mission. Scope: agent reads the real
  CI failure via the Connector, investigates as far as evidence
  allows, and proposes ONE linked bug Issue (observed behavior,
  repo/commit, repro limits, suspected-vs-confirmed, impact,
  acceptance criteria); owner authorizes filing. Exclusions:
  diagnosis beyond evidence, repair (separate approval), CI
  config changes. Deps: B2-03, B2-08 (filing requires the
  approval→effect binding). Workstream B2 · Priority P0 ·
  Executor: Sprite agent proposes, Owner authorizes.
- Location: Sprite reads, Studio records. App: Studio repo.
  DB: local PG (proposal + approval rows). Infra: none. Auth:
  per-action GO for the Issue filing; reads task-level.
  Evidence: proposed issue text, approval receipt, filed URL +
  id, no-double-creation check. Recovery: close with reason on
  misfire (authorized separately).

### B2-05 — Checkpoint capture/restore on Sprite

- Source: B1 core + Discussion #2 priorities + review R2-4. Scope:
  extend the accepted checkpoint/resume/interrupt semantics to the
  remote workspace: capture on Sprite, deliberate failure,
  restore, reconstructed-context labeling; journal of recovery
  points. Capture is task-scoped (non-destructive: a snapshot is a
  persistent side effect, but it changes no live state); every restore
  requires explicit approval bound to Sprite identity, checkpoint
  identity, target workspace, and expected recovery effect, with
  a recoverable pre-restore state preserved first. Exclusions:
  cross-workspace moves (B2-06), auto-recovery. Deps: B2-01B.
  Workstream B2 · Priority P0 · Executor: Builder.
- Location: Sprite execution, Studio recording. App: Studio
  repo checkout on Sprite. DB: local PG (checkpoint rows,
  restore approvals). Infra: Sprite workspace only. Auth:
  task-level GO for capture; explicit per-restore owner GO with
  the four-part binding. Evidence: checkpoint rows, restore
  receipts with approval refs, pre-restore states,
  deliberate-failure drill log. Recovery: restore is the
  recovery; rows are append-only.

### B2-06 — Local↔Sprite handoff (single writer)

- Source: SKILLS §14 + Discussion #2 priorities. Scope: move the
  active task between Local Desktop and the Builder Sprite with
  receipts; enforce one active writer; stale-state banners on the
  non-owning side; symmetric return path. Exclusions: fleets,
  live process migration. Deps: B2-01B, B2-05. Workstream B2 ·
  Priority P0 · Executor: Builder.
- Location: both ends, Studio-mediated. App: Studio repo. DB:
  local PG (handoff receipts, owner flips). Infra: Sprite
  workspace only. Auth: per-handoff owner GO (moves live work).
  Evidence: preflight, transfer manifest, owner-flip receipt,
  conflict cards on interruption. Recovery: re-handoff back;
  never silent discard.

### B2-07 — Workspace Preview on Sprite URL

- Source: Discussion #2 priorities + review R3-A. Scope: serve the
  mutable in-Sprite app build on its Sprite URL for inspection;
  labeled everywhere as Workspace Preview, never release proof.
  Access policy: private organization-authenticated URLs only;
  synthetic or sanitized data; no production credentials or
  private production data, ever. Making a URL public requires
  separate, explicit authorization for that exact exposure.
  Exclusions: immutable artifacts, Machines, Neon (B3). Deps:
  B2-01B. Workstream B2 · Priority P1 · Executor: Builder.
- Location: Sprite serves, Studio links. App: Studio repo build.
  DB: Sprite-local/dev only. Infra: Sprite service row. Auth:
  task-level GO for the private surface; public exposure needs
  its own explicit GO. Evidence: live URL + VERIFIED actual
  access mode (not URL presence alone) + inspection notes +
  non-proof labeling check. Recovery: stop the service; nothing
  promoted.

### B2-08 — Approval→effect binding + Console integration

- Source: connector-auth gaps (c)+(e) + §3 + review R2-2. Scope:
  implement the write-path split: Sprite Connector stays
  read-only; a Studio-side GitHub App with narrow per-area grants
  performs approved writes; bind approval rows to App calls so
  approval gates execution (close the empty-Channels gap);
  surface usage guidance to agents; Console cards for proposals,
  scoped approvals/receipts, progress, and incident/runner views.
  Exclusions: new authorization doctrine (§3 is the doctrine).
  Deps: B2-02, B2-03, B2-10 (App registration must exist before
  binding). Workstream B2 · Priority P0 · Executor: Builder.
- Location: Studio server + Console. App: Studio itself. DB:
  local PG (approvals, linkages). Infra: GitHub App registration
  (owner-held credentials, least privilege; minimum per-operation
  permissions, repository installation scope, and deliberately
  excluded permissions specified below — including Contents for
  installation-token git access; key and credentials server-side
  only, outside Sprite, agent context, repository, and Studio
  database). Auth: the binding itself is enforced server-side;
  UI never authorizes. Evidence: broker-enforcement gate tests
  per the proof paragraph above (missing/expired/revoked/
  mismatched/cross-operation refusals plus exact-effect
  acceptance), merge-boundary tests (separate approval,
  no create→merge replay, no bypass), Issues and Discussions
  granularity tests, UI refusal-path tests. Positive-effect
  tests use the same canary discipline as B2-02 probes
  (disposable resources, stop-on-surprise, authorized cleanup).
  Recovery: revoke approval rows and App grant; deny-by-default
  restored.

  Broker-enforcement proof (distinct from B2-02's Connector-
  denial proof): the suite proves the STUDIO BROKER refuses the
  unapproved write before the GitHub App executes it — covering
  missing, expired, revoked, mismatched-target, and cross-
  operation approvals — and that an accepted approval produces
  exactly its authorized effect and nothing else. It further
  proves the broker never lets the Sprite obtain the App token
  or call its authenticated API directly (no bypass path exists
  in code, config, or granted scopes).

  Merge boundary (finer than GitHub permissions): the App token
  technically permits merging, so Studio enforces the split —
  PR creation and updates require their own scoped approvals; a
  merge request is refused without a separate merge approval; a
  creation approval cannot replay into a merge; the agent cannot
  bypass the broker to obtain merge capability. Same principle
  for Issues and Discussions (create/update/delete granularity —
  write is never treated as create-only).

  GitHub App permission matrix (minimums; least privilege):
  Issues read+write (file/comment/close defect Issues), Discussions
  read+write (create authorized Discussions), pull requests
  read+write (open/update PRs), Contents read+write (clone/fetch
  and push branches over installation-token HTTPS), Actions
  read-only (runs/jobs/logs evidence), metadata read (implied).
  Deliberately excluded: administration, secrets, environments,
  deployments, packages, members, workflows write. Installation
  scoped to the single Studio repository. Any operation needing
  more returns for a new explicit authorization.

### B2-09 — GitHub Discussions write path

- Source: review R2-A (feature-initiative workflow). Scope:
  propose a Discussion, discover available categories live
  (GraphQL IDs only — never assume, create, or rename), prepare
  its contents, obtain owner authorization, create it through
  the Studio-side GitHub App, report URL + identifier; refusal
  paths for unauthenticated/unauthorized attempts. Exclusions:
  milestone retrospectives content (per-milestone work), category
  administration. Deps: B2-03, B2-08. Workstream B2 · Priority
  P1 · Executor: Sprite agent proposes, Owner authorizes.
- Location: Studio server via GitHub App. App: none (records
  only). DB: local PG (proposal + approval rows). Infra: none
  new. Auth: per-action owner GO for the create; reads and
  category discovery task-level. Evidence: draft text, category
  evidence, approval receipt, created URL + id. Recovery: close
  or delete with reason on misfire (authorized separately).

### B2-10 — Studio-side read-only integration + App registration

- Source: review R4-3 (Option B fallback). Scope: register the
  GitHub App (owner-held credentials, read permissions) and
  implement Studio-side read-only operations for EXACTLY the
  operations B2-02 proves unavailable through the Connector
  (possibly the empty set, recorded with evidence — the
  registration itself is still required downstream). Separate
  slice from the B2-08 write broker by construction, so the
  graph stays acyclic. Exclusions: all writes (B2-08),
  Connector operations (B2-02/B2-03). Deps: B2-02 (needs its
  verification matrix). Workstream B2 · Priority P1 · Executor:
  Builder.
- Location: Studio server via GitHub App (read-only). App: none
  (records + evidence only). DB: local PG (read models, App
  registration record). Infra: App registration. Auth: per-action
  owner GO for registration; reads task-level. Evidence: App
  record, per-operation fallback coverage (or verified-empty set
  with the B2-02 matrix attached), read transcripts. Recovery:
  revoke App grant; Connector path unaffected.

### B3-01 — Application Preview (stretch, ordered)

- Source: Discussion #2 stretch + unified §7.6. Scope: build-once
  OCI image from the verified bundle, temp Machine + ephemeral
  Neon branch, digest-pinned deploy, per-Machine running-digest
  assertion, teardown. Exclusions: Staging/Production (B5).
  Deps: B2 core (B2-01A/B, B2-02..B2-10) genuinely complete,
  including the B2-08 approval binding before any cloud
  deployment. Workstream B3 · Priority P2 (stretch) · Executor:
  Builder.
- Location: agent-invoked CLIs under supervision. App: Studio
  repo image. DB: Preview Neon branch (seeded, sanitized). Infra:
  Machine + branch lifecycle. Auth: per-environment owner GO.
  Evidence: provenance row (bundle→image→Machine), health +
  digest receipts, teardown verification. Recovery: destroy
  Machine + branch; no promotion path touched.

### Independence map

B2-01A stands alone and gates B2-01B (local contracts before live
proof). Every other card's acceptance requires live evidence, so
each depends on B2-01B directly or transitively — never on B2-01A
alone: B2-02 needs B2-01B; B2-03 needs B2-02 (fallback branch
additionally needs B2-10); B2-04 needs B2-03 + B2-08; B2-05 needs
B2-01B; B2-06 needs B2-01B + B2-05; B2-07 needs B2-01B; B2-08 needs
B2-02 + B2-03 + B2-10; B2-09 needs B2-03 + B2-08; B2-10 needs B2-02;
B3-01 needs the full B2 core including B2-08 and B2-10. The graph
is acyclic: B2-10 consumes B2-02's verification matrix but nothing
points back into it except as a completed prerequisite.
Parallelizable pairs: (B2-02, B2-05), (B2-04, B2-06), (B2-07,
B2-08), (B2-04, B2-09), (B2-07, B2-10). No card touches another's
exclusions; shared touchpoints (approval rows, Console patterns)
are additive.

---

## 7. Acceptance evidence and Hacksgiving demo script

### Per-class acceptance (each independently verifiable)

- **Local loop:** B0/B1 suites green unmodified; new slices add
  tests without weakening old ones.
- **Connector genuine:** every GitHub operation names the
  Connector call or the ordinary API call that performed it; a
  test asserts no silent substitution path exists.
- **Sprite lifecycle:** wake/sleep/connect/disconnect receipts;
  checkpoint/restore drill with deliberate failure.
- **Refusals:** unauthenticated, cross-op replay, expired
  approval, and unsafe-resume cases each fail closed with named
  reasons (mirroring the B1-R1 refusal matrix style).
- **Recovery:** interrupted work resumes from durable rows on a
  fresh server; teardown of every provisioned resource verified
  by list commands.
- **CI signal:** the incident intake card reproduces the *signal*
  (failed `go test` step on run 45666647515) from Connector
  reads; any root-cause claim requires its own authorized repair
  slice with a green re-run as proof.
- **Preview honesty:** Workspace Preview labeled non-proof;
  Application Preview asserts per-Machine running digests (B3).

### Demo script (adapted 13 beats; honest evidence each step)

1. Cockpit: Console shows the task, spine, and empty inbox.
2. Wake the Builder Sprite; connect; receipt shown.
3. Verify the GitHub Connector live (attested record, no secrets).
4. Pull run 45666647515's failure through the Connector — only
   via operations live-verified under B2-02; display the observed
   signal with its limits stated (degraded-but-honest evidence
   where a capability is unavailable, never a claimed read).
5. Agent proposes ONE linked bug Issue with evidence (draft only).
6. Owner authorizes; agent files; URL + id reported, no dupes.
7. Agent investigates inside the Sprite; checkpoints persist.
8. Deliberate failure → checkpoint restore → reconstructed label.
9. Authorized repair; tests green inside the Sprite.
10. Authorized PR with evidence links; review outside the demo.
11. Workspace Preview on the Sprite URL (labeled non-proof).
12. Report: rows, receipts, URLs — every claim clickable.
13. Stretch (only if B3-01 is genuinely done): Application Preview
    with digest + teardown proof; never at the core demo's expense.

---

## 8. Risks, prerequisites and explicit exclusions

| # | Risk / prerequisite | Handling |
| - | ------------------- | -------- |
| 1 | `gh` auth invalid in this environment | Public reads used; Project API unverified; owner re-auth unblocks live checks (no agent login attempts) |
| 2 | Project #3 fields/views/workflows | Screenshot-verified baseline stands; live drift check at first authenticated contact |
| 3 | Discussion categories | Only `Ideas` observed; full list via GraphQL before any category selection |
| 4 | Toolchain drift (go.mod 1.27.3 vs CI 1.26) | Observed, not root-caused; repro slice pins and proves |
| 5 | Port mismatch (CI 5432 vs testdb 5434 gate) | Observed signal only; repro slice proves or eliminates |
| 6 | OAuth token lifecycle (refresh/rotation) | Designed in B2-02 follow-up before any long-lived use; PATs never reintroduced |
| 7 | ADR-001/ADR-002 amendments | B3 records ADR-001 mechanism change; ADR-002 needs explicit owner amend-or-retire |
| 8 | Sprite networking/auth setup | Owner-gated; separate-auth design, no copied credential stores |

**Explicit exclusions:** Staging/Production management; fleet/quota/
billing; provider marketplace; workflow canvas; embedded terminal;
auto-promotion; multi-region; live process migration; production
data in agent context (forbidden); any B0/B1 contract change.

## Appendix A — North Star Discussion draft (unpublished)

Working title (deliberately distinct — must never duplicate the
published initiative): `Miniapp Studio — Engineering Direction
Working Draft (Planning Artifact, Do Not Publish As-Is)`.
Published Discussion #2 keeps the canonical title `Miniapp Studio
— Product Vision, Engineering Principles and Hacksgiving
Direction`. Proposed category: `Ideas` (only observed category;
confirm full list before selecting). Contents: the §1 vision
table, the §2 record division, the §3 lifecycle in owner language,
the B0–B5 roadmap, the demo mission. This draft lives here only;
publishing needs its own proposal → authorize → create cycle, and
any publication must link — never clone — Discussion #2.

## Appendix B — Decision log

- Bugs-only Issues + Task Cards win (prompt27.1 newest).
- Rev2 B0–B5 labels win (SKILLS §14).
- Machines for Preview/Prod; ADR-001 amendment recorded as B3 work.
- Demo centers the Sprite-resident incident repair; dogfood frames it.
- `gh` failure recorded as environment limitation with public
  fallbacks used and screenshot tier preserved.

*End of plan. Awaiting architectural review and implementation
authorization. No implementation was performed to produce this
document.*
