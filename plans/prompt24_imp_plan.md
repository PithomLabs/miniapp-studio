Review revision complete. All work is in the plan document; no implementation or cloud operations performed.
  
  1. Blocking item — deployable artifact contract (resolved in doc). Added binding §7.6 defining the full provenance chain
  the review required: verified bundle (digest B) → one image build (pinned Go toolchain + pinned base, hermetic inputs from
  fetched bundle bytes) → OCI image (immutable digest I + provenance record) → Preview → Staging → Production Machines all
  running I. The revision establishes: build-once (never per-environment); store-by-digest with registry recommendation (Fly
  registry default, GHCR alternative, choice recorded in B4); verify at rest and at runtime (running digest queried via
  fly machine status --json / fly status --json, mismatch fails the release); deploy-by-digest-only adapter allowlist
  (source-path deploy forms excluded, so promotion cannot rebuild); B→I→toolchain/base provenance rows surfaced in Evidence.
  It records that ADR-001's Sprite-per-app deployment mechanism is superseded for Preview/Staging/Production because no image
  pipeline existed then — Sprites remain dev environments — with supersession recorded in B4. B4 scope/acceptance extended (6
  items incl. provenance row, running-digest check, source-deploy negative test); B6 promotion guards and acceptance
  rewritten around the image digest; §8.2 flow 3 and a §9.2 invariant row updated. No new phase added, per the review.
  
  2. B2/B3 prerequisite (clarified). B3 now states the binding minimum: checkpoints write/read APIs + git-manifest convention
  + one demonstrated resume-after-kill must be demonstrated-with-evidence before handoff work starts. Only B2's workbench
  slice may fold or slip. Demo-path wording in §10/§11 updated to match.
  
  3. Repository integrity (investigated, gated). .agents/skills/ and root .mcp.json are confirmed absent (never tracked, not
  in Trash, not in dangling commits, no lookalikes in home). I audited this session completely: no write/edit/remove call
  targeted either path (all removes were /tmp scratch, __pycache__, chrome profiles), which excludes my session and my
  research agents as the cause. Correlated owner-side activity in the same window: ~/.agents/skills/neon/ created 10:50, repo
  .agents/ modified 10:53, and new SKILLS.md (engineering constitution), plans/skills_review.md,
  plans/prompt24_plan_review.md at 11:20–11:32 — consistent with an owner skills reorganization, but the contents of the
  removed paths were never recorded anywhere I can access, so nothing is recoverable repo-side. .agents/memory/ is verified
  intact. Per the review, this is now P0 acceptance item 5 (resolve status/recovery with you before B1 starts) plus §13 row
  13 — I did not assume it was intentional. Note: SKILLS.md declares itself governing guidance; I have not ingested it yet
  and will treat it as binding input from the next turn.
  
  Verification: 15 sections intact, 12 phases × 16 fields intact, secret scan clean, tree otherwise untouched (71 entries =
  68 prior + review file + 2 owner constitution files + this doc's earlier creation; zero tracked-file modifications by me).
  
  The plan is ready for your final approval and the Bootstrap 1 GO.

◆ Worked for 4m 32s · 11:33 AM · 26/Nov

◆ Recap: Revised the lifecycle plan for the review: image chain, checkpoint prerequisite, integrity gate. Next action: owner
         approves plan and issues Bootstrap 1 GO.
