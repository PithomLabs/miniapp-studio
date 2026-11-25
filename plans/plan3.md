# Consolidated plan to implement plan2_review.md + prompt2.md

## Goal

Execute [plan2_review.md](/home/chaschel/Documents/go/miniapp-studio/plans/plan2_review.md) (§4 revision request) and [prompt2.md](/home/chaschel/Documents/go/miniapp-studio/plans/prompt2.md) (§5 roadmap requirement) together, producing one revised, evidence-backed implementation plan at `docs/architecture/miniapp-studio-implementation-plan.md` in the studio repo. Doc-only: no code, deps, migrations, Fly resources, or edits to governing plans.

Note: the file lives at `plans/prompt2.md` (there is no `docs/prompt2.md`; `docs/` is currently empty).

## Source precedence

1. [plan.md](/home/chaschel/Documents/go/miniapp-studio/plans/plan.md) (v3) and [prompt.md](/home/chaschel/Documents/go/miniapp-studio/plans/prompt.md) (v2) remain the governing product/brief baseline, frozen.
2. `plan2_review.md` directs the revision: five required fixes (A–E), five decided owner questions (§3), §4 deliverable outline.
3. `prompt2.md` adds one roadmap requirement: a provider-neutral Agent Provider + Execution Environment abstraction (managed agents, self-hosted runtimes, coding-agent integrations as three distinct categories), seams-only in the MVP, full adapters post-hackathon.
4. Reconciliation: `prompt2.md` §5 asks to amend the coding-agent prompt, but `plan2_review.md` §4 forbids modifying governing plans. `prompt.md` stays frozen; the roadmap requirement flows directly into the revised implementation-plan doc per `prompt2.md` §5G instead.

## Success criteria

- All five `plan2_review.md` revisions resolved with repository evidence: (A) MVP runner execution contract (class, sync/async, approval persistence/decision, crash/duplicate/unknown-effect behavior, persistence-vs-DBOS boundary, no half durable engine); (B) Fly as first-class architecture (Studio serverless-style hosting, Sprite builder agent, per-app URL topology comparison on isolation/cost/cold-start/URL-stability/ops); (C) 10-step golden-path lifecycle acceptance test incl. redeploy URL stability and version history; (D) connector 3-state catalog + first-class Custom API with a negative out-of-scope-path test and gateway-held credentials; (E) UI acceptance criteria for Home, Create App, Connectors, Validate & Deploy, My Apps, Activity incl. loading/empty/error/success states.
- All five §3 owner decisions applied: frozen exact framework pin (no floating "or newer"); MPG-if-compatible else Neon without letting DBOS block the MVP; topology spikes before selection with real URL round-trip + collision tests; Box-preferred, Custom API fallback connector anchor; separate studio repo with root verification before init.
- Agent Provider roadmap phase included per `prompt2.md` §5G: provider-neutral Go interfaces, per-provider capability matrix, auth/secret architecture, Sprite/Machine lifecycle adapters, session/webhook contracts, threat model, contract tests, cost/ops trade-offs; MVP seams explicitly separated from post-hackathon integrations.
- Four-dimension framing adopted (app definition, agent provider, execution environment, business execution) with vendor-neutrality preserved: changing providers never rebuilds the generated app.
- Every gate G0–G12 mapped to concrete tasks + test evidence; blockers separated from optional improvements; unresolved decisions listed.

## Approach

Single revised document, evidence-first, reusing completed recon (all plans files, framework inventory, `sprites-go` surface). New evidence to gather: Class A vs DBOS execution semantics from `executor/classa`, `executor/dbos`, `approvals`, `runtime.Service` code/tests; framework pin freeze via test runs; Fly topology comparison; managed/self-hosted provider contracts from current official docs (never from memory). MVP establishes seams, not adapters: no managed-agent integration is claimed from a mere provider selector.

## Steps

1. Freeze framework pin: run miniapp's required suites against candidate commits, record exact commit + results (needs Postgres; none locally available — see risks).
2. Resolve runner execution semantics (A) from code/test evidence; write the persistence-vs-DBOS boundary contract.
3. Design Fly deployment architecture (B): Studio hosting, builder agent, generated-app targets; define the three topology spikes and comparison criteria.
4. Specify lifecycle acceptance tests (C) and connector/Custom API contracts with negative authorization tests (D).
5. Specify UI acceptance criteria per screen (E) plus the dogfooding component matrix (what serves UI vs invokes Miniapp ops).
6. Specify the Agent Provider roadmap phase: three integration categories, provider-neutral interfaces, per-provider matrix (Gemini/Claude managed, Gemini self-hosted, coding agents), secret isolation, threat model; mark MVP seams vs post-hackathon work.
7. Write `docs/architecture/miniapp-studio-implementation-plan.md` per the §4 outline extended with the roadmap phase; map G0–G12; self-audit every A–E/§3/§5G item to a section.

## Validation plan

- Readback audit: each A–E requirement, §3 decision, and §5G deliverable traceable to a doc section; every reused capability cites file/symbol/test; no invented paths, versions, or green results; gates table complete.
- Anti-claim checks: no durable waits/resumption/exactly-once without test evidence; no "managed agents implemented" from a selector; no fabricated Fly URLs; no mocked connector presented as live integration.
- Highest-risk validation: pin freeze (step 1) — without a test-backed pin, all reuse claims stay provisional.

## Risks / open questions

- No Postgres here (`TEST_DATABASE_URL` unset, no local server); pin-freeze tests need the compose DB or CI.
- `flyctl` still sandbox-blocked; Fly spikes need an authorized env — owner must say when/where they may run.
- Provider-doc verification (managed-agent integrations pages, `gemini-api-cli` self-hosted Fly provider repo) needs web access at implementation time; record check dates.
- Sequencing: §3 approves `git init` of the studio repo but §4 says stop after the doc. Assumption: doc-only until owner approves the revised plan; repo init/scaffolding waits for a later instruction.
- Pin base: freeze at current `73351a8` pending test evidence, or survey newer commits first?
