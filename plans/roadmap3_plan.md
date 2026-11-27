# roadmap3 plan (revised 2026-11-24)

Revision note: the owner resolved both open questions — (1) Lead-to-Booking
is explicitly positioned as the candidate second demonstration and
generality test (implementation excluded, P5 and golden path unchanged);
(2) execution is one continuous drafting pass with no intermediate reviews.
Part 1 below is the as-executed gateway meta-plan (delivered:
`plans/agent-gateway-implementation-plan.md` + §14 report, 2026-11-24,
document-only). Part 2 is the new meta-plan for `plans/prompt_opencode.md`,
awaiting a GO.

---

# Part 1 — Agent Gateway plan (executed)

## Goal

Execute `plans/prompt_roadmap3.md`: produce `plans/agent-gateway-implementation-plan.md`,
a comprehensive implementation-ready plan for the Studio↔Builder-Sprite Agent
Gateway, plus the §14 final report — document-only, no implementation.

`plans/roadmap3.md` (GHL/Lead-to-Booking analysis) is context: it motivates why a
governed builder/agent story matters. It is not in scope to implement or to absorb;
the gateway plan references it only as a motivating scenario unless the owner
says otherwise (see Open Questions).

## Success Criteria

- All 23 sections from §9 present, each implementation-ready (not an essay).
- Mermaid diagrams for architecture, user→agent round trip, Q&A lifecycle,
  build/promotion lifecycle, and state machines; diagrams checked for valid
  Mermaid syntax.
- Every claimed path, API, SDK capability, and code structure verified in the
  current checkout or the pinned dependencies; proposed interfaces explicitly
  labeled as proposals (§10).
- Locked decisions (§2 A–F) and five-phase intent/sequence (§3) preserved;
  any genuine foundational conflict backed by repository evidence, not re-litigation.
- F1 and G-01 reconciled explicitly; neither marked resolved nor bypassed (§6).
- P5 builder roadmap untouched as implementation (§8); gateway reconciled with
  governing-plan Revision 2, not silently replacing it; no DBOS/new workflow
  engine without demonstrated requirement.
- Frozen docs (`plan4.md`, `plan5.1.md`, `plan6.md`) and historical evidence
  unmodified; no source edits, migrations, provisioning, commits, or pushes.
- §12 governance labels applied throughout (implemented / accepted-but-unbuilt /
  proposed / undecided / external / blocker); §11 root-cause discipline and
  §10 per-phase actionability checklists (all 17 bullets × 5 phases) satisfied.

## Approach

Inspection before drafting, in dependency order; draft sections in the order
their facts settle (state → architecture → protocol → phases → risks); then
self-audit the document against the §9/§10/§12 checklists before reporting.

Verification standard: no path or API from memory — `read_file`/`search` for
every citation. Behaviors distinguished as inspected vs executed. Unverifiable
claims (e.g., live Sprite sleep/wake, agent-CLI resume semantics that cannot
run here) labeled with the exact verification step deferred to an authorized
environment.

## Steps

1. Confirm conventions and boundaries: root `AGENTS.md` is absent (verified
   2026-11-24) — fall back to user-level `AGENTS.md` + project memory, and
   record that. Confirm no plans index exists (none found) → skip any index
   update per §13. Re-read frozen-doc list; never open them for edit.
2. Read governing plan Revision 2 (`plans/miniapp-studio-implementation-plan-consolidated.md`),
   ADRs 001–003, evidence `docs/p0-evidence.md`, `docs/p3-evidence.md`,
   `docs/p4-evidence.md`; extract the reconciliation surface (P5 scope, §14
   ownership rules, F1/G-01 approval requirements, cut line).
3. Survey current implementation: `server/` (routes, sessions, SSE/event
   capability — verify whether SSE exists), `store/` (migrations 000001–000010,
   session model), `tools/` + `runner/` (Service invocation, idempotency,
   audit), `apps/` + `fly/` (`SpriteManager`/`ConnectorManager`, fakes),
   `cmd/studio`, Miniapp runtime/audit/approval paths in the framework
   checkout (read-only).
4. Verify pins and tags: `go.mod` Miniapp pin vs any published `v0.1.0` tag;
   `sprites-go` v0.2.2 source for sessions/services/exec/exit-status/checkpoint
   semantics the gateway depends on (§5: HTTP 200 ≠ success).
5. Consult Fly Sprite docs/SDK references only for gaps the checkout cannot
   settle (sleep/wake, checkpoint restore, callback ingress); cite sources.
6. Draft §§1–6 (summary, UX, locked decisions, verified state, architecture,
   topology/sequences): architecture extends — never renames — existing
   `fly`/`apps`/`tools` boundaries; Builder vs Runtime Sprite separation
   mapped onto the existing deployment lifecycle.
7. Draft §§7–11 (data model, protocol, adapter contract, provisioning,
   persistence/recovery): refine (do not blindly adopt) the §4 event/identifier
   vocabulary with written rationale; define envelopes, versioning, ordering,
   ack, dedup, retry, replay; per-agent resume capabilities as contract
   entries with unknown-marked cells.
8. Draft §§12–17 (security incl. F1/G-01 reconciliation, Q&A/approvals,
   artifacts/review, promotion, failure modes with lifecycle state machines,
   observability/audit): every §5 failure and §6 trust item dispositioned.
9. Draft §§18–23 (testing/acceptance, five-phase roadmap with full §10
   checklists, dependencies, risks/owner-decisions, non-goals, definition
   of done): critical path + parallelizable work identified; test plan
   requires red-then-green regressions and replay/retry/race coverage.
10. Self-audit: 23-section checklist, Mermaid validity, citation spot-checks
    (every file/API claim re-opened), frozen/evidence untouched (`git status`),
    no source files modified, §12 labels complete; then emit the §14 report
    (path, sections, reuse, gaps, owner decisions, critical path, unverified
    assumptions, no-implementation confirmation).

## Validation Plan

- Structural: all 23 §9 headings present; five Mermaid diagrams present and
  syntactically valid; §10's 17 bullets answered per phase.
- Evidentiary: each "verified" claim traceable to a file path + symbol read
  this session; each proposal labeled; each deferred verification named with
  its environment.
- Boundary: `git status` shows only the new plan file (and this meta-plan);
  frozen docs byte-identical; F1/G-01 text contains no resolution claim;
  no P5-implementation or DBOS content beyond what §8 permits.
- Governance: §12 six-way labeling applied; blockers meet the six-part
  evidence standard or are downgraded to risks/questions.

## Risks / Open Questions

- Root `AGENTS.md` does not exist; will use user-level instructions + project
  memory and record the substitution. Risk: a convention the prompt assumes
  lives in a file that was never committed.
- No plans index found; defaulting to no index update. Risk: none if correct;
  owner can point at one if it lives elsewhere.
- Live Sprite/agent-CLI behaviors (wake-on-command, checkpoint restore, Codex/
  Claude resume) cannot execute in this sandbox; those sections will carry
  explicit unverified-assumption labels with precise deferred proofs.
- `v0.1.0` tag relationship from §8 is unverified until Step 4.
- Owner input needed (see message): (1) Lead-to-Booking positioning —
  context-only vs explicit roadmap positioning in the gateway plan;
  (2) execution shape after GO — single continuous drafting pass vs
  section-by-section checkpoints.
  → Resolved by owner: (1) explicit positioning (§2/§20/§22 of the
  delivered plan); (2) single pass. Both recorded in the revision note.

---

# Part 2 — OpenCode amendment plan (awaiting GO)

## Goal

Plan the execution of `plans/prompt_opencode.md`: produce the OpenCode §15
amendment (sections A–I) for the Agent Gateway plan as a reviewable,
implementation-ready document change — without editing
`plans/prompt_roadmap3.md` itself (prompts are inputs, not working docs)
and without implementing anything.

## Success Criteria

- Recommended deliverable: a separate amendment doc
  (`plans/agent-gateway-opencode-amendment.md` [PROPOSED NAME]) containing
  the §15 A–I content mapped onto the gateway plan's sections (locked
  decisions, §9 adapter contract, §10 provisioning, §12 security,
  §§18–19 acceptance), plus an exact merge list (which gateway-plan
  sections change, which lines of intent). The owner approves the merge;
  no silent rewrite of the approved gateway plan. (If the owner prefers
  direct revision on GO, the same content lands in the gateway plan
  with a changelog entry — confirm at GO.)
- Every external fact cited to source + version + access date: official
  plugin README, `package.json`, release history, source/tests as needed;
  Fly ecosystem page; MCP endpoint docs. No invented CLI flags, APIs, or
  install procedures; OpenCode 1 vs 2 differences recorded from the repo,
  never assumed identical.
- Pinned recommendations with rationale: OpenCode version (stable 1.x
  preferred per the prompt unless evidence supports 2), plugin version,
  Bun/Node runtime, config format, MCP endpoint, auth expectations. No
  floating branches.
- Topology question answered from sources (§D 1–7): where the OpenCode
  process runs, where the plugin installs, where commands execute, where
  the repo lives, how the gateway drives OpenCode, how events are
  captured, how sessions resume. If the official topology differs from
  "OpenCode inside the Builder Sprite", the implications and the
  supported recommendation are written out — no silent bypass.
- OAuth/non-interactive tension surfaced honestly: browser-OAuth initial
  authorization, identity/org, persistence across sleep/wake, scoping
  (name prefixes, creation limits, MCP permissions, destructive-op
  confirmations). No invented non-interactive workaround, no CLI-install
  bypass (per §E).
- Adapter mapping complete: each §G capability (start, assign, messages,
  progress, input-needed detection, answer forwarding, resume, completion
  detection, artifact retrieval) mapped to a stable OpenCode interface or
  explicitly marked unsupported with a tested-boundary design.
- §H 15-step acceptance proof merged into the Phase C/D acceptance
  criteria as the OpenCode-specific proof track; the question→answer→
  same-session-resume round trip is the named critical condition (a
  `/sprites-status`-style check alone is explicitly insufficient).
- F1/G-01 positions unchanged; P5/Lead-to-Booking scoping unchanged;
  frozen docs untouched; document-only (no code, provisioning, commits).

## Approach

Sources before drafting: read the official plugin repository and Fly docs
first, fill the topology/auth/version facts, then draft the amendment
against the gateway plan's section map, then audit for consistency (no
contradiction with locked decisions, no invented interfaces, unknowns
labeled with deferred proofs).

## Steps

1. Fetch and read: plugin README + `package.json` + releases + relevant
   source/tests; `https://fly.io/sprites/ecosystem/`; MCP auth docs.
   Record access dates; treat README + package metadata as source of truth.
2. Answer §D topology (1–7) strictly from those sources; document where
   sources are silent and name the live verification for each gap.
3. Resolve §F pins: OpenCode 1.x vs 2, plugin version, runtime, config
   format; verify npm availability path + local-checkout fallback as
   documented (do not execute installs — planning only).
4. Specify §E auth: initial OAuth ceremony, org/identity, persistence,
   scoping, destructive-op confirmations; reconcile with gateway key
   distribution (§12 of the gateway plan) without conflating the two
   mechanisms (Fly MCP OAuth ≠ gateway HMAC build keys).
5. Map §G capabilities onto the §9 `AgentAdapter` interface method by
   method; mark unsupported cells with boundary designs.
6. Draft the amendment doc (§15 A–I) + merge list + changelog entry text.
7. Audit: citation check (every external claim re-opened), consistency
   check vs gateway plan §§3/9/10/12/19/22, boundary check (no source
   touched, frozen docs intact), then report.

## Validation Plan

- Structural: all nine §15 sections (A–I) present; merge list names exact
  target sections; §H proof track has executable steps.
- Evidentiary: each external claim has source+version+date; each pin is
  exact; each gap names its deferred live proof; no CLI flag or API
  appears without a source line.
- Boundary: only the amendment doc (+ this meta-plan) written; no
  implementation, provisioning, or commits; F1/G-01/P5 scoping intact.

## Risks / Open Questions

- External sources drift: plugin README/releases may move between planning
  and implementation — the amendment must version-stamp everything and
  state the re-verification rule.
- Deliverable shape: separate amendment doc (recommended) vs direct
  gateway-plan revision — confirm at GO.
- If official sources show OpenCode is designed to run *outside* the
  Sprite with remote execution, the "agent inside Builder Sprite" model
  needs an explicit supported-topology decision — the meta-plan requires
  the amendment to surface this rather than assume it away.
- Auth bootstrap (browser OAuth for a headless builder lifecycle) is the
  highest-risk unknown; the amendment must not invent a workaround.
