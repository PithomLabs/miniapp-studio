## Goal

Produce the Prompt32 architecture specification package — six coordinated design docs under `docs/architecture/` covering the Security Contract, Observability Contract, and Developer Experience/Enforcement Architecture as three perspectives on one runtime — plus an evidence-backed implementation roadmap. Architecture-only: no implementation, per the work order.

(Note: the work order lives at `docs/prompt32`, no `.md` extension — that is the file this plan implements.)

## Success Criteria

- All six specified documents exist with the required contents (§10 list): rationale, existing-capability inventory, boundaries, programming model, internal architecture, lifecycle diagrams, threat model, observability taxonomy, conformance matrix, migration strategy, risks/decisions, staged plan, acceptance criteria.
- Every requirement bullet in §§3–9 is addressed in exactly one owning document and cross-referenced, never duplicated or dropped.
- All §9 invariants and residuals are preserved and explicitly carried (C1–C7, lock ordering, separate auth tx, SettleEffect boundary, M5/P24-008 activation, revocation staleness, P31-001, P29-005, drill, CI limits).
- Existing code, proposed designs, assumptions, and owner-approval decisions are explicitly distinguished throughout.
- Worktree contains only the six new files; zero source, migration, dependency, or config changes.

## Approach

Research shows a clean starting state: HEAD is now `73351a8` ("prompt31", Prompt30 work committed), the tree is clean except the work order itself, and no `docs/architecture/` exists yet. The input corpus is bounded and known: `plan4.2.md` (1502 lines), lifecycle contract (1161), authorization design (480), closeouts, the Prompt21–31 review chain, four roadmap docs (~1500 lines total), and a small existing surface (`auth.Policy`/`Binding`/`Authenticate`/`Authorize`, pg-backed `audit.Service`, `registry.Register`/`Resolve`, rest/mcp/cli transports, no OTel dependency, empty `capability/clients/`).

Key decisions: (1) Inventory before design — map every proposed capability to existing source, existing contract, or genuinely new work first, enforcing the "do not redesign working mechanisms" rule. (2) Contract order S → O → DX, since DX is the mechanism that makes S and O real and must consume both; the unified architecture doc then synthesizes interactions rather than repeating domains. (3) Single glossary and single inventory appendix in `foundational-architecture.md` to prevent cross-doc drift; per-contract docs carry only their domain plus a "current state" pointer. (4) Threat model extends `docs/phase0/threat-model.md` (44 lines, owner-approved scope) rather than replacing it. (5) Diagrams as ASCII sequence/flow diagrams in Markdown (terminal-rendered, diffable). (6) No open questions — the work order is decision-complete; depth calibrated to "review-ready specification," stopping at the package plus roadmap handoff.

## Steps

1. Baseline (HEAD, clean-tree check) and read all inputs: governing plan/contracts, Prompt21–31 findings and dispositions, residual closeouts, roadmaps, threat model, and a source-surface survey of runtime/transport/registry/auth/audit.
2. Build the existing-capability inventory: per capability, existing / partial / documented-only / absent, mapped to source path or contract section.
3. Draft `security-contract.md` (identity + authn adapters, capability authorization/delegation, execution-time security context across Classes A–D, relationship to `authorizeAttempt`/approvals/Settle boundary, MCP non-backdoor rule).
4. Draft `observability-contract.md` (audit/logs/metrics/traces, OTel architecture + exporter failure behavior, correlation IDs with cardinality rules, redaction/sampling/retention, event/metric taxonomy, auto-instrumentation coverage list).
5. Draft `developer-experience-enforcement.md` (Business Capability abstraction, the five enforcement mechanisms, escape hatches without silent opt-outs, honest Go library-vs-isolation boundary).
6. Draft `foundational-architecture.md` (one-runtime synthesis, cross-contract interaction, per-class integration table, lifecycle diagrams, threat model, glossary, inventory appendix).
7. Draft `foundational-conformance-matrix.md` and `foundational-roadmap.md` (shared + class-specific conformance, fixture strategy, §11 sequencing with independently testable milestones, acceptance criteria, owner-decision register).
8. Audit pass: requirement-coverage map (§§3–9 → sections), §9 invariant checklist, cross-doc terminology consistency, prohibition compliance (`git status`), then stop for owner review.

## Validation Plan

Specs are validated by audit, not tests: a coverage table proving every §3–§9 bullet lands in an owning section; a §9 line-by-line preservation checklist; a grep-based terminology/glossary consistency pass; a relative-link resolution check over the six docs; a self-review read challenging each capability against the "enforced by construction" doctrine; and a final `git status` proving only six new files exist. Highest-risk step is the audit pass — with six interdependent docs, cross-reference drift or a silently dropped requirement is the most likely defect, so coverage mapping is mechanical (checklist-driven), not from memory.

## Risks / Open Questions

Risk: scope sprawl — three contracts plus unification can balloon into duplicated or contradictory text; contained by single-owner sections, one glossary, and the coverage audit. Risk: inventing cleaner replacements for proven mechanisms (authorization tx, attempt fencing); contained by the inventory-first rule and §9 checklist. Open questions: None — all material direction is in the work order and governing docs.
