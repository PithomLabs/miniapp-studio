# Plan: execute revised `plans/prompt34.md` → `plans/mom-n-pop-rpa-platform-plan.md`

Planning-only meta-plan for the RPA platform architecture mission. No implementation,
no live ops, no commits. Tracks the REVISED prompt34 (secrets-first preamble, remote
reference URL, §§1–10 incl. §4.1–4.8 and §8 adversarial review).

## 0. Starting position (verified 2026-11-28)

- Reference read in full: README, all 16 `SKILL.md` files named in §1, plus
  `small-business-website-spec` (17th dir on disk — see below), `.env.example`,
  all references, script structures, and the cheat-sheet example (~7.3K lines).
  Architecture understood: single-tenant-per-VPS, conversational, Markdown
  skills + Python CLIs + cron + SQLite + Telegram; secrets in `~/.hermes/.env`
  (0600); recurring patterns (draft→approve→execute, dry-runs, `no_agent` cron
  + UID tracking, per-skill verification checklists, wired-vs-documented
  honesty, rate-card single-sourcing, dual-approval paths, adapter-per-provider).
- Reference source rule: prompt34 cites the GitHub URL but the working copy at
  `/home/chaschel/Documents/go/Mom-n-Pop-Skills-main/` is **not a git repo**
  (no revision pin; files dated Sep 11). The mission verifies against the
  remote if network allows, else declares the local tree (path + content
  hashes) as the snapshot source. Traceability (§10) depends on this being
  explicit, never assumed.
- Skill inventory lock (per owner correction): the verified inventory is 16
  skills **including** `small-business-website-spec` exactly once — i.e. the
  15 entries in prompt34 §1's bullets counted with stripe×2/mailchimp×2
  confusion resolved as: the §1 list **plus** website-spec, deduplicated to
  16, each studied exactly once. No double-counting stripe/mailchimp pairs.
- Studio inputs confirmed present: ultimate roadmap (vision, locked decisions,
  tracks, F0/R0–R8), `Taskfile.dev.yml` + `Taskfile.yml`, `docs/dev-contract.md`,
  `docs/dev-serve.md`, `docs/taskfile.md`, Go config/secret-loading code
  (`server/config.go` `_FILE` convention, `dbenv`, `migratedb`), `connector/`,
  `tools/`, `fly/` surfaces, Miniapp framework + ADR-001 execution categories.
- Prior accepted work (000027 ACL repair, Card 001/002 gates, prompt33 UI,
  tenancy/approvals/effects/Neon-only doctrine) is context: the mission cites
  accepted behavior where it reuses it and proposes nothing contradicting
  locked decisions.

## 1. Execution phases

### Phase A — Source lock + skill inventory (prompt34 §1)
1. Pin the reference snapshot (remote diff if reachable, else local content
   hashes) and record it as the deliverable's sources baseline.
2. Per-skill capsules for the locked 16, each: purpose, flows, data/credentials,
   integrations, scheduling, safety posture, verification story, Hermes-specific
   couplings — every claim cited to a file path.
3. Extract the safety-critical subset explicitly: live-money paths, credential
   handling shapes, autonomous execution tiers, dual-approval coverage and gaps.

### Phase B — Outcomes + platform abstractions (prompt34 §2–§3)
1. Restate the §2 workflow catalog as the acceptance suite the primitives must
   satisfy (canonical lead-to-payment + the seven others). Reference use cases
   only — no workflow implementations.
2. Draft capability areas A–I as contracts-first sketches: domain placement
   rule (framework vs F0 vs adapter vs Miniapp, anti-giant-CRM guard), typed
   capability model, orchestration within A/B/C/D + ADR-001 (no engine without
   evidence), provider-neutral connectors with explicit capability deltas,
   framework-owned approvals (PENDING/APPROVED/REJECTED/EXPIRED, material-bound,
   principal-separated), effect protocol with UNKNOWN quarantine, event/cron
   intake (deterministic-first), tenancy/Neon/RLS preservation, provider-
   decoupled agent/owner UX.

### Phase C — Secrets + environment architecture (prompt34 §4.1–4.8)
1. Audit current state first: Taskfile vars, `dev serve-init`/`serve` file
   contracts, `_FILE` loader + precedence, production guard, existing tests.
2. Design: canonical variable inventory schema; external `.env` layout +
   permissions + parse/validate rules; Fly sync workflow reusing ONLY verified
   CLI/API mechanisms (unverified = open decision with a verification
   procedure, never invented commands); unified Taskfile verbs; single Go
   loader with explicit precedence and fail-closed matrix; deployment-vs-
   tenant credential split with broker abstraction; seven-environment
   isolation table; executable acceptance tests per §4.8.
3. Hard invariants carried through: same contract, never same values or
   storage; tenant credentials never ambient; dev never inherits prod.

### Phase D — Mapping + deliverable (prompt34 §5–§6)
1. Place every primitive: framework / F0 / adapter / Miniapp / orchestration /
   Engineering UI / Automation Operations Studio / secrets-contract owners
   (incl. which components own local loading, Fly sync, runtime config, and
   brokerage). Name dependency phases and reused accepted capabilities by path.
2. Draft all 17 deliverable sections incl. Go contract sketches (illustrative,
   not dumps), three mapped workflows (lead-to-payment + two cross-domain),
   dependency-ordered roadmap, test strategy (crash/retry/duplicate/authz/
   isolation/leakage), risks/tradeoffs/non-goals, and the traceability matrix.
3. Per-primitive record: problem, owner, contract, failure/recovery, boundary,
   tests, exists/extend/new.

### Phase E — Adversarial review + self-verification (prompt34 §8, §10)
1. Self-review, then a distinct adversarial pass over all 14 §8 challenges
   (over-abstraction, state duplication, approval safety, failure contracts,
   isolation, lock-in, engine adoption, idempotency, premature building,
   leakage/precedence, crossover, unsafe sync, secret confusion, roadmap
   conflicts); resolve substantive findings in the document; classify the
   rest as blocking/material/non-blocking with dispositions recorded.
2. Verify against §7 (no port, no Python/Hermes dependency, no giant CRM, no
   hardcoded providers, no canvas, no billing, no reopened locks, no invented
   schema, no unapproved engine, no Roadmap A bleed, no credential or
   crossover violations) and §9 (single file created, nothing else touched,
   no credentials handled, no live ops, no commits).
3. Report: file path + minimum primitive set + first vertical slice +
   sequence + secrets workflow + explicit owner decisions (with defaults,
   non-blocking) + review findings. No implementation GO implied.

## 2. Boundaries (binding, from prompt34 §9)

Planning only; no code, migrations, provisioning, provider auth, Fly sync,
credential handling, commits, or pushes. Roadmap and ADRs are read-only
sources — deltas surface as PROPOSAL/OWNER-DECISION entries only. Create
ONLY `plans/mom-n-pop-rpa-platform-plan.md`.

## 3. Risks / open points

- Single-execution size (study + 17-section deliverable + dual review) is
  large; mitigated by the completed scoping read above.
- Remote-vs-local reference drift: handled by Phase A snapshotting, never
  assumed identical.
- Fly CLI/API verification depends on sandbox network + installed CLIs;
  anything unverifiable becomes an open decision with a procedure.
- Per owner correction: **no owner-terminal live database runs** in this
  assignment. Anything requiring live verification (Neon behavior, Fly sync
  mechanics, provider API shapes) is recorded in the deliverable as future
  implementation verification requirements with exact procedures — the
  mission proves what is provable from code/docs and labels the rest.
- No clarification needed from the owner before starting: the prompt is
  complete, boundaries explicit, and open items have decided fallbacks.
  Genuine architecture forks discovered mid-mission become OWNER-DECISION
  entries per §8, not interactive asks.
