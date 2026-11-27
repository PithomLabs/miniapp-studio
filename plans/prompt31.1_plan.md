## Goal

Produce a plan for executing prompt31.1: inspect the governing sources, reconcile (not duplicate) the existing
roadmap proposal, and author the authoritative Prompt31 product architecture + dual-roadmap document (Roadmap A
B0–B5, Roadmap B R0–R8, common F0 foundation) — document-only, with B2/demo work fully protected.

## Success Criteria

- Every prompt31.1 §8 input is inspected or explicitly recorded missing with a fallback rule; no claim rests on
  an unread source.
- The §10 twelve-question answer set is satisfiable from the deliverable; each answer traces to a section.
- All 15 §8 artifacts exist with per-capability IMPLEMENTED/ACCEPTED/DESIGNED/IN PROGRESS/PLANNED/DEFERRED/
  UNVERIFIED labels backed by file:line or evidence citations — never self-asserted.
- The B0–B5 UI coverage matrix is built from the actual handler inventory, each cell carrying an honest maturity
  state; the R0–R8 dependency matrix names gates without inventing implementation.
- Zero code, migration, or accepted-behavior changes; zero commits; B2 critical path untouched.

## Approach

Research-first, single-document, reconcile-over-create. The decisive scoping facts (verified this turn):
`plans/miniapp-studio-ultimate-roadmap.md` already exists as a ratification-pending proposal quoting the same
vision — so §10's "create or reconcile" starts with a full read of that document and reconciles unless it fails
explicit coverage criteria. `plans/roadmap4.md` and `plans/writeup2.md` were supplied mid-mission, read in full, and corroborate the
citations used (binding invariant, ownership table, invariants, fleet/metering gaps), with roadmap4's status
snapshot superseded and one external §6 misattribution corrected in PII.0. `app_category_guidelines.md` exists
but frames substrates (DBOS/River) illustratively; per governing
decision 14 it is read for failure-contract semantics only, never as an engine decision.

Key decisions:

- **D1 — Reconcile by default.** Inspect `miniapp-studio-ultimate-roadmap.md` against the §10 twelve questions
  and §8 fifteen artifacts first. Extend/amend it where it falls short; create a separate Prompt31 document only
  if it structurally cannot carry the dual-roadmap + F0 shape. Rejected: writing a parallel roadmap (guaranteed
  drift) and rewriting history (prior prompts stay untouched per §9).
- **D2 — Missing sources don't block (moot — sources supplied).** roadmap4/writeup2 arrived mid-mission and
  were read in full; the fallback was never needed. Kept as standing practice for future gaps.
- **D3 — Categories as contracts, not substrates.** Class A–D definitions come from the guidelines' failure
  taxonomy (what must survive what); every DBOS/River mention there is illustrative per its own header and
  prompt31.1 §5. Rejected: treating the guidelines as an engine mandate (contradicts §9).
- **D4 — Evidence labels need citations.** Each taxonomy label points at a repo path, evidence doc section, or
  named gap. "ACCEPTED" additionally requires the acceptance record. No label upgrades without new evidence.
- **D5 — UI matrix from code, not memory.** Enumerate `server/routes.go` handlers (+ templates/partials backing
  them) as the implemented column; map each B0–B5 required screen to handler-or-gap with a maturity state
  (`Available`, `Partially available`, `Planned`, `Not connected`, `Unavailable`).
- **D6 — Demo protection is structural.** The work is document-only by assignment; the plan adds no code steps,
  and the F0/R0 design explicitly must not reopen B0/B1/B2 without defect evidence.
- **D7 — One pass, section-gated.** Fifteen artifacts produced in dependency order (tenancy → automation →
  execution → UI → roadmap → ADRs/cards), each reviewed against its §8/§10 acceptance before the next begins;
  stops at the document, no implementation spill.

## Steps

1. **Source inspection.** Read fully: ADRs 001–018, `app_category_guidelines.md`, B0–B5 plans + evidence docs,
   `miniapp-studio-ultimate-roadmap.md`, prompts 27.1–30, Neon/Fly/connector/authorization contracts, UI handler
   inventory. Log each as read/missing with line-scale notes; record the roadmap4/writeup2 limitation.
2. **Gap + status matrix.** Build the capability inventory with seven-state labels and citations; reconcile pass:
   verdict on extend-vs-create for the roadmap document with reasons tied to §10 coverage.
3. **F0 domain core.** Draft tenancy/organization/delegation model (company + agency patterns), Automation
   entity + versions/workflows/runs/targets, ownership/inheritance/delegation rules. Each rule states its RLS,
   Sprite-policy, and audit consequences or marks them open.
4. **Execution contracts.** Translate Class A–D guidelines into per-class capability/placement/authorization/
   timeout/retry/observability/verification profiles; runtime alternatives (PG-backed, DBOS, River) framed as
   ADR evaluations with decision criteria, not selections.
5. **UI architectures.** Roadmap A coverage matrix (B0–B5 screens × five maturity states, handler-mapped);
   Roadmap B information architecture (14 areas) as navigation/structure spec with honesty rules; shared
   design-system/API/terminology contracts; deployment-topology left to a named ADR.
6. **Roadmaps + promotion.** R0–R8 dependency matrix with gates; B-roadmap UI-completeness overlay preserving
   accepted statuses; six-environment contract table (local → production) with artifact-immutability and
   no-shared-mutable-state rules; per-automation vs per-app release distinction.
7. **Close-out package.** Security/threat model, ADR backlog (unresolved only, no invented decisions),
   implementation Task Cards, twelve-question traceability index, final consistency pass (every §10 answer
   resolves to a section; every label has a citation).

## Validation Plan

- Inventory audit: each §8 input has a read/missing log entry; spot-check three citations resolve to real lines.
- Reconcile decision reviewed against explicit criteria before any drafting (no silent fork, no silent rewrite).
- Per-section acceptance: F0 rules each carry consequence-or-open marking; UI matrix cells each have a handler
  path or a named gap; roadmap edges each name a gate; ADR backlog contains decisions only, no pre-made selections.
- Honesty audit: search the draft for fixture/telemetry-like claims — each must carry its illustrative/planned
  label; B3–B5 screens must show non-Available states.
- Final: twelve-question traceability walked end to end; zero repo modifications verified via status check.
- Highest-risk validation: the F0 tenancy/automation core (step 3) — every downstream artifact inherits its
  mistakes, so it gets a dedicated re-read before steps 4–7 build on it.

## Approval amendments (owner, binding on execution)

1. **Locked numbering:** retain B0–B5; document the charter's B1–B6 discrepancy as history, never rename/reorder.
2. **Two explicit UIs:** Focus Console serves Roadmap A; a separate Operations/QA/Management experience serves
   Roadmap B; shared design system/services/identity/authorization, separate information architectures.
3. **Two maturity dimensions** (supersedes the plan's single-state framing): UI maturity (Designed, Implemented,
   Partially implemented) × Functional maturity (Functional, Partial, Planned, Not connected, Blocked). A complete
   B5 interface may be UI-implemented while promotion stays functionally Planned.
4. **F0 vs R0:** F0 = shared platform contracts; R0 = RPA product architecture on F0; R1–R8 = implementation.
   Reconcile T/S/E/H/O tracks into the two roadmaps + F0; no competing taxonomies.
5. **Unresolved decisions stay unresolved:** API-centric vs UI-RPA, Neon project-per-tenant topology, two-UI
   deployment topology — surfaced explicitly, never silently promoted. Six environment concepts stay distinct.

## Risks / Open Questions

- Scope sprawl across fifteen artifacts: contained by section gates and the single-document rule; if a section
  needs depth beyond the pass, it becomes a named ADR-backlog item rather than expanding inline.
- `miniapp-studio-ultimate-roadmap.md` may partially contradict prompt31.1 corrections (e.g., E-series naming,
  single-tenant R0): step 2 resolves each conflict explicitly with newest-owner-decision precedence, recorded.
- ~~Missing roadmap4/writeup2~~ supplied and read mid-mission: no provenance gap remains; taxonomy cites the
  guidelines directly, ops gaps cite roadmap4 §5/writeup2 §6, and the one external misattribution is corrected
  in PII.0.
- Open questions: none blocking. The only genuine unknown — whether the existing roadmap proposal can carry
  the reconciled shape — is step 2's explicit verdict, not a pre-plan question.
