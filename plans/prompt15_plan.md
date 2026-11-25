# Plan to implement plans/prompt15.md — demo critical path

## Goal

Execute prompt15's bounded directive in one pass: close the three
documentation residuals (F-R1/N8, F-R2/N5, F-R3), then create
`docs/hackathon-demo-critical-path.md` as the execution plan for the
hackathon demo (narrative + Part 2 gate + Phase C/D/E plans + UI
contract + safety + priorities + dependencies), verify only the
touched documentation, report per prompt15 §9, and stop at the
live-authorization boundary.

## Success Criteria

- F-R1: N8 staging-window restriction sits beside the §6
  re-provision operation with the three operator checks; no
  fixed-in-code claim.
- F-R2: runbook §11 cold path records pre-restart agent PIDs and
  requires a post-recovery absence check; no
  agent.failed-only success claim.
- F-R3: closure-report counts enumerate exact sets (or label
  DB-backed subsets); tests untouched.
- Demo doc exists with prompt15 §5 A–F + §8 UI items 1–9, every
  judge-facing step labeled implemented / fake-backed /
  locally-tested / live-unverified / not-implemented, plus
  dependency graph, P0/P1/P2 breakdown, blocking owner decisions,
  exact Part 2 GO prerequisites, and an evidence-per-claim
  acceptance checklist.
- No Phase C/D/E code, no live operations, no commits without
  authorization, no broad review, Part 2 still unauthorized.

## Approach

Doc-only execution task (research + precise edits + one new
document). Two pre-decisions so the implementer never guesses:

1. **Missing roadmap refresh (owner-confirmed).**
   `miniapp-studio-roadmap-refresh-2026-11-25.md` does not exist
   and is not cited as a decision source; `plans/roadmap4.md`
   is the Nov 25 roadmap with *original* status. Confirmed
   fallback baseline: prompt15 §1–3 status table + roadmap4.md
   + accepted reports/evidence, newest accepted decision wins
   on conflict. Record the missing roadmap as a source
   limitation in the demo doc, not an implementation blocker.
   Gate: before treating this fallback as the authoritative
   roadmap for a GO decision, obtain explicit owner
   confirmation.
2. **Reuse-first mapping for the demo doc.** Ground every status
   label in existing surfaces, verified this turn: 25 Class A
   tools (`spec.*`, `deploy.*`, `conversation.*`, `build.*`,
   `question.answer`, …), 19 server templates (incl.
   `conversation.html`, `answer_conflict.html`, deploy + connector
   screens), SSE (`conversation.stream`), gateway signed
   outbox/inspect/requeue + Continue/turn fencing, spec
   validation + runner admission/dryrun, `deploy.provision`
   (P3-era app deploy — Phase E section must gap-analyze it
   against approved-digest/no-rebuild/health/rollback, not
   assume greenfield), migrations through 000016. Anything not
   found in-tree is not-implemented; anything exercised only via
   fakes/doubles is fake-backed even if unit-tested.

Non-goals: C/D/E/Part-2 implementation, live ops, commits,
broad adversarial review, P2 scope (fleet/billing/quotas/
marketplace/second adapter/new frameworks), Lead-to-Booking,
any Miniapp/F1/G-01/P5 change, any secret exposure.

## Steps

### P0 — Baseline confirmation (first; gates everything)

- U0: Confirm each prompt15 §3 bullet from repo evidence: `git
  log/status` (commits exist; never assume committed),
  migration list, tool/template inventories, N1/N4/R-DB/R-NET
  closure docs, P1–P4 probe state, N2–N10 register, flake
  classification. Record discrepancies; prompt15's table +
  accepted reports win on conflict (confirmed fallback). Record
  the missing roadmap-refresh file as a source limitation in
  the demo doc's baseline section.

### P1 — Residuals (independent of each other; U0 first)

- U1 (F-R1/N8): in runbook §6 beside the second-build refresh
  note, add the restriction: verify staging mode 0700 (exact
  check command), verify no live untrusted sibling agents
  (exact check), drain-first rule when unestablished. No
  fixed-in-code language. Verify by re-read + grep that no
  other §6 path contradicts it.
- U2 (F-R2/N5): in runbook §11 step 5 cold path, add: record
  agent PIDs pre-sleep (status + sprite ps), post-recovery
  absence verification + orphan/ambiguity reporting rule, and
  the ban on agent.failed-only success claims. Verify wording
  against N5's register entry.
- U3 (F-R3): in closure report §7, replace bare counts with
  exact-set wording derived from source (`grep ^func Test`
  over the provision/upgrade/store/server suites + the `-run`
  patterns that produced them): 16-file vs 14-DB-backed F7
  sets (name the 2 pure tests), 15-strict vs 20-broad
  migration/RLS sets (name the membership rule), 354/356
  verbose accounting. Do not touch any test.

### P2 — Demo critical-path document (U0 first; narrative first)

- U4 (narrative A): 13-step judge story per prompt15 §5A with a
  per-step status label from the fixed taxonomy; each label
  cites its evidence (file/test/doc) or states what is missing.
  Never promote planned behavior to capability.
- U5 (Part 2 gate B): reuse the runbook as-is; extract
  genuinely-blocking owner decisions (org, model/auth, Studio
  reachability D3, retention, pins) and the 10-item evidence
  checklist with observable pass criteria. State the separate-GO
  requirement; execute nothing.
- U6 (Phase C): concrete tasks around the known missing piece
  (Studio→Gateway continue-dispatch sender loop) + turn/question
  binding, payload validation, atomic handoff, fencing,
  stale-answer rejection, same-session proof via live P-G3,
  SSE replay, restart/reconnect. Constraints: file-drop only,
  no invented flags, no prose-as-metadata, no second mechanism.
- U7 (Phase D): narrow Invoice-Desk-only pipeline per the
  required foundation + 9 negative-test classes; contract
  enforcement at validation/admission/build boundaries (prompt
  text insufficient); no marketplace, no deployment inside D;
  record the build-locus decision (recommended: Studio-side
  fetch-verify-build reusing `runner/`+`spec/`, sandboxed)
  with rationale and the rejected alternative.
- U8 (Phase E): minimal Runtime path per the 8 bullets; gap
  analysis of existing `deploy.provision` vs approved-digest
  pinning, no-rebuild transfer, health-before-live, rollback,
  Runtime recording. Deployment never automatic.
- U9 (rest): safety/governance (8 rules + synthetic data +
  G-01), P0/P1/P2 classes with the exclusion list, dependency
  graph (prompt15 §7 sequence as spine), blocking decisions,
  GO prerequisites, per-claim acceptance checklist; UI §8
  items 1–9 (design contract, zone/scene maps from the Appendix
  inventory, backend source-of-truth + event/API deps,
  OPS/DEV rules, clarification/approval/kill-switch contracts,
  responsive/a11y, C/D/E sequence, prototype-vs-real table).

### P3 — Verify + report (last)

- U10: doc-only verification: re-read all touched sections;
  grep audits for dangling refs, stale counts, and forbidden
  claims (simulated receipts/URLs, agent-done ⇒ green,
  live-implying language); `git status` proves only 3 paths
  changed (runbook, closure report, new demo doc); confirm no
  live commands ran. Final report per prompt15 §9. Stop.

## Validation Plan

- U1–U3: each edit re-read in full context + `grep` for the
  finding's key terms (N8/staging-0700, N5/PID-absence, exact
  counts) showing exactly one authoritative home; cross-check
  N8/N5 register entries still agree.
- U4–U9: outline-completeness check against prompt15 §5A–F +
  §8(1–9) + §8 deliverables 3–6 (every required element
  present, every status label evidenced, every judge claim
  mapped to observable evidence or marked unverified).
- U10: `git status --short` (expect only the 3 doc paths);
  `grep -rin` sweeps for capability-claim verbs on
  live-unverified items; confirm zero `flyctl`/`sprite`/
  provision/exec invocations in the turn. Highest-risk check:
  the per-step status labels — a single inflated label breaks
  the document's whole purpose, so each is spot-audited back
  to source.

## Risks / Open Questions

- **Confirmed (was the open item): the roadmap-refresh file is
  missing.** Fallback baseline stands per owner confirmation;
  recorded as a source limitation, not a blocker. Standing
  gate: obtain explicit owner confirmation before treating
  the fallback as the authoritative roadmap for any GO
  decision.
- Phase D build locus and kill-switch scope are demo-doc
  decisions with recommendations, not user questions — no
  reply needed.
- None. No other user-owned choice is outstanding; all other
  unknowns resolve from repo evidence during U0.

## Appendix: plan_ui inventory (researched, for U9)

Zones: rail (Gallery, Connectors, Evidence, Settings, Pulse +
OPS/DEV toggle + hold-700ms kill); spine (7 pills
idle/active/proven/refused, clickable); middle scenes (gallery,
spec, validate, connect, deploy, run, evidence, refused,
settings); Pulse stream (event/receipt/alert/approval cards +
filters + pending badges + expiry countdowns + inline
approve/reject); Line (suggestion chips + single input; submits
become Pulse events). Scripted-only behaviors to replace: timed
`pipeline()` sleeps, RUNDEMO fixture data, fabricated Fly URL,
seeded events/receipts, Custom API wizard, kill switch,
hardcoded gates/metrics. Missing vs prompt15: clarification
event type + reply-to-Line binding, kill-switch scope,
empty/loading/error/offline states (only a bare empty note
exists). Baseline: dark tokens, 64px rail, 900px Pulse
slide-over, reduced-motion support.
