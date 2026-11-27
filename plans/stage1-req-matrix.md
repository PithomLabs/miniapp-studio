# Stage 1 — Requirement traceability matrix (S1.3)

**Sources:** `plans/prompt28.2.md` (§3–§13), `plans/prompt28.3.md` (§1–§10).
**Legend — Existing:** file path or NONE. **Stage:** S1 (design artifact) or T1–T5 (implementation track).
Proofs marked LIVE require live evidence in Stage 2; none may be satisfied by documentation alone.

## prompt28.2 — Fly-native lifecycle

| REQ | Source | Requirement | Existing | Missing / ADR | Stage | Acceptance |
|-----|--------|-------------|----------|---------------|-------|------------|
| R201 | 28.2 §2 | Go/Taskfile/Machine/Sprite/GitHub/PG/control-plane responsibility split | Partial (`fly/`, `gateway/`, Taskfiles) | Controllers, webhook path | S1+T2–T4 | ADR + implemented split |
| R202 | 28.2 §2 | Three independent lifecycles (agent / CI / preview); no shared mutable workspace, credentials, or DB | NONE | ADR-007/009/014 | S1 | Contract doc + enforced separation (LIVE) |
| R203 | 28.3 §2 | 28.2 §3 native-PG `task dev:setup` SUPERSEDED by Neon provisioning | `Taskfile.dev.yml`, `studio dev` | ADR-004/005/006 | T1 | Neon setup flow (LIVE); §3 retired, not implemented |
| R204 | 28.2 §4.1 | CI controller on Fly Machine: webhook rx, sig verify, dedupe, persist-before-ack, limits, provision, execute, report, cancel/retry/timeout, reconcile | NONE (gateway server patterns reusable) | ADR-007/012 | T2 | Full §13 CI matrix (LIVE) |
| R205 | 28.2 §4.1 | Prompt webhook ack; execution on separate worker, never in HTTP handler | NONE | ADR-012 | T2 | Load + restart test (LIVE) |
| R206 | 28.2 §4.1 | Evaluate PG-backed state+leases+outbox vs DBOS; no automatic DBOS | `gateway/outbox.go`, store patterns | ADR-007 | S1 | ADR decision recorded |
| R207 | 28.2 §4.2 | CI runs exact immutable commit SHA; head SHA from payload (see V17) | NONE | ADR-007 | T2 | SHA recorded per run (LIVE) |
| R208 | 28.2 §4.2 | CI executes canonical Taskfile targets (no second command set) | `Taskfile.dev.yml` (dev) | CI Taskfile surface | T2 | Same targets local+CI (LIVE) |
| R209 | 28.2 §4.2 | CI validation contract: Go 1.27, port/URL, PG19, suites, zero allowlist changes, no silent substitution | Partial (`testdb`, ci.yml) | C4 redesign (ADR-006) | S1+T2 | Contract doc + enforced (LIVE) |
| R210 | 28.2 §4.2 | 28.2 digest-pinned CI image SUPERSEDED by Neon CI branch + Neon verification | ci.yml (Actions+image) | ADR-004/007 | T2 | Neon CI run (LIVE) |
| R211 | 28.2 §4.3 | GitHub commit checks via least-privilege GitHub App; CI vs preview vs agent checks separated | NONE | ADR-011/017 | T2 | Checks visible on PR (LIVE) |
| R212 | 28.2 §4.3 | Required checks + branch protection; agent push alone never merges | NONE | ADR-017 | T2 | Merge-blocked proof (LIVE) |
| R213 | 28.2 §5.1 | Per-PR isolated Fly app + machines + DB + config + secrets + URL + logs (+ storage as applicable) | NONE | ADR-009 (+005,018) | T3 | Two-PR isolation proof (LIVE) |
| R214 | 28.2 §5.2 | Per-PR DB isolation invariant; explicit ADR choice, not convenience | NONE | ADR-009 (Neon branch-per-PR) | S1 | ADR recorded |
| R215 | 28.2 §5.3 | Lifecycle events: opened/reopened/synchronize/closed/failure/missed-webhook behaviors | NONE | ADR-009/012/013 | T3 | Event matrix (LIVE) incl. missed-webhook recovery |
| R216 | 28.2 §5.3 | Per-PR deploy serialization; SHA+generation fencing vs stale overwrites | NONE | ADR-009 | T3 | Rapid-push + stale-job tests (LIVE) |
| R217 | 28.2 §5.3 | Deterministic globally-unique resource names (repo+PR, never bare PR number) | NONE | ADR-009 | T3 | Naming audit on live resources (LIVE) |
| R218 | 28.2 §5.4 | Preview DB: provision → migrate → fixtures → ready → deploy → health+smoke; never prod data/creds | NONE | ADR-009/015 | T3 | Ordered sequence proof (LIVE) |
| R219 | 28.2 §5.5 | Per-PR object-storage + external-service isolation; sandbox providers; no prod side effects | NONE | ADR-018 | T3 | Isolation tests (LIVE) |
| R220 | 28.2 §5.6 | Preview exposure policy (public vs auth); no security-by-obscurity; no prod/control-plane reachability | NONE | ADR-009 | T3 | Policy doc + negative tests (LIVE) |
| R221 | 28.2 §5.7 | Cost controls: autostop, max previews, per-PR limits, lifetime, idle expiry, storage caps, closure cleanup, reconciliation/GC, ownership metadata, attribution/alerts | NONE | ADR-013 | T3 | Limits enforced + orphan-GC proof (LIVE) |
| R222 | 28.2 §5.8 | Preview URL+state published on PR; read-only agent diagnostics (health/status/logs/migration/smoke) without RW org creds | NONE | ADR-009/011 | T3 | Agent-visible diagnostics, privilege audit (LIVE) |
| R223 | 28.2 §6.1 | Authorized triggers (approved label); verify repo/state/user/label/eligibility; untrusted issue text; no privilege from public text | NONE | ADR-014 | T4 | Trigger + negative tests (LIVE) |
| R224 | 28.2 §6.2 | One Sprite per run from clean state; fresh vs checkpoint-restored evaluated; no shared mutable dir; sleep/wake for same-PR continuation | NONE | ADR-008 | T4 | Isolation + checkpoint proofs (LIVE) |
| R225 | 28.2 §6.3 | GitHub App (never PAT); short-lived per-run repo-scoped tokens; controller/Fly/model creds outside Sprite; connector evaluated; token treated readable-by-agent | NONE | ADR-011 | T4 | Token audit + connector decision (LIVE) |
| R226 | 28.2 §6.4 | Agent workflow: clean Sprite → clone → base → deps → headless agent → validation → commit → branch → push → PR-gated-on-validation → record → retain/destroy | NONE | ADR-014 | T4 | E2E run (LIVE) |
| R227 | 28.2 §6.4 | Never push to protected main; failing tests → stop or marked-draft-with-failures, never green | NONE | ADR-014/017 | T4 | Negative + draft-policy tests (LIVE) |
| R228 | 28.2 §6.5 | No long-held exec (see V7: sessions+markers); durable run metadata, heartbeats, timeouts, cancel, retry limits, failure classes, abandonment cleanup | NONE | ADR-008/014 | T4 | Kill/timeout/cancel proofs (LIVE) |
| R229 | 28.2 §6.6 | Pluggable agent/model adapters; spend limits + per-run budgets; restricted egress; hostile-code assumption | NONE | ADR-014 | T4 | Budget + egress enforcement (LIVE) |
| R230 | 28.2 §7 | Integrated flow agent→PR→CI→preview→diagnostics→review→merge→destroy without privilege merger; no self-approval; no CI↔preview circular gate | NONE | ADR-007/009/014/017 | S1+T3/T4 | Flow doc + distinct check names + gating proof (LIVE) |
| R231 | 28.2 §8 | State machines: webhook deliveries, agent/CI executions, previews, resources; restart recovery; dup/out-of-order/missing handling; idempotency; ownership-verified destroy | Partial (gateway outbox/dedup patterns) | ADR-012/013 | T2–T4 | Machine docs + chaos proofs (LIVE) |
| R232 | 28.2 §9 | Threat model over the 12 listed vectors; trust boundaries + credential ownership per environment | NONE | Threat-model doc (S1.6) | S1 | Published model; tests trace to it in Stage 2 |
| R233 | 28.2 §10 | Taskfiles declare prereqs/setup/DB/init/validation/CI/build/ops; Go stays in Go; same local↔CI contract | `Taskfile.dev.yml` partial | ADR-006; prod surface iff needed | T1/T2 | Task inventory + parity proof (LIVE) |
| R234 | 28.2 §11 | 12 ADRs with full field set; Fly uncertainties verified by proof, not assumed | `decisions/001–003` | ADR-004–018 (S1.5) | S1 | Numbered records published |
| R235 | 28.2 §12 | Arch docs: lifecycle, diagrams, controller, preview, agent, state machines, threat model, cost; amend (never duplicate) existing docs | Partial | S1.6 docs | S1 | Docs published + stale refs removed |
| R236 | 28.2 §13 | Local/CI/preview/agent/resilience proof matrices | NONE | Per-track plans (S1.8) | T1–T4 | All matrices green (LIVE) |
| R237 | 28.2 §14 | End-to-end acceptance properties 1–11 | NONE | Per-track plans (S1.8) | T1–T4 | E2E demonstration (LIVE) |

## prompt28.3 — Neon-first PostgreSQL

| REQ | Source | Requirement | Existing | Missing / ADR | Stage | Acceptance |
|-----|--------|-------------|----------|---------------|-------|------------|
| R301 | 28.3 §1 | Neon mandatory for ALL PG workloads (control plane, dev, agents, CI, previews, prod, miniapps, internal) | NONE | ADR-004 | T1–T5 | Inventory: zero non-Neon PG (LIVE audit) |
| R302 | 28.3 §1 | No self-hosted PG: no apt/PGDG server installs, no Docker PG, no Testcontainers PG, no PG in Machines/Sprites. psql/client tools allowed | `testdb` (violates), ci.yml (violates) | ADR-004/006; C4 redesign | S1+T1/T2 | Prohibition enforced; grep+runtime audit (LIVE) |
| R303 | 28.3 §2 | Reconcile superseded decisions 1–6 (native install, PGDG, owner install proc, maintainer roles, CI image, Testcontainers) | Superseded texts in master plan, `pg_muse.md`, `testdb` | S1.4 records | S1 | Supersession records published; no dual-satisfying text |
| R304 | 28.3 §3 | ADR: Neon project + branch hierarchy (prod / non-prod / dev / agent / CI / PR / miniapp) | NONE | ADR-005 | S1 | ADR published |
| R305 | 28.3 §3 | Separate prod vs non-prod projects (strongly considered); never derive untrusted branches from sensitive prod; synthetic/schema-only/anonymized | NONE | ADR-005/015 | S1+T1–T4 | Topology proof + data-lineage audit (LIVE) |
| R306 | 28.3 §3 | Miniapp project-per-x decision (miniapp/tenant/boundary) with cost/isolation/quota/lifecycle/observability/recovery evaluation | NONE | ADR-016 | S1 | ADR published; no shared-project default |
| R307 | 28.3 §4 | `task dev:setup` via Neon: CLI/API preflight → auth → project validate → dev-branch reconcile → credentials → env config → `studio dev` verify → connectivity+version proof | `task dev:setup` (native-PG shaped) | ADR-005/006 | T1 | Clean-host run (LIVE) |
| R308 | 28.3 §4 | Management creds outside agent processes + app runtimes; no local-PG-server dependence; Taskfile orchestrates, Go implements | Partial | ADR-006/010 | T1 | Credential-placement audit (LIVE) |
| R309 | 28.3 §5 | Per-execution disposable CI branch: sanitized parent → create → migrate → creds → test → collect → delete → reconcile orphans | NONE | ADR-007 | T2 | Branch-per-run proof + orphan-GC (LIVE) |
| R310 | 28.3 §5 | Sprite connects to Neon over its endpoint; runs no PG server; branch↔SHA association | NONE | ADR-007 | T2 | Association record per run (LIVE) |
| R311 | 28.3 §5 | Preserved: Go 1.27, port/URL, validation targets, zero allowlist changes (as redesigned by C4), extension inventory (re-proven) | Partial | ADR-006 | T2 | Parity proof (LIVE) |
| R312 | 28.3 §5 | Neon verification replaces digest verification: project/branch/parent IDs, configured + actual (`SHOW server_version[_num]`) version, extensions, compute config | Fingerprint (image-shaped) | ADR-006 | T2 | Fingerprint fields on live runs (LIVE) |
| R313 | 28.3 §6 | Per-PR Neon branch with repo/PR/SHA/app/project/branch/config/state/expiry association; Fly app connects exclusively to it | NONE | ADR-009 | T3 | Association + exclusivity proof (LIVE) |
| R314 | 28.3 §6 | PR-DB strategy: persistent-per-PR vs per-commit-generation; migrations across commits; data reset/preserve; stale-deploy fencing; atomic config switch | NONE | ADR-009 | S1+T3 | ADR + strategy proof (LIVE) |
| R315 | 28.3 §6 | Preview controller reconciles Neon↔Fly state; close/merge deletes branch + independent expiry/GC | NONE | ADR-009/013 | T3 | Reconcile + delete proofs (LIVE) |
| R316 | 28.3 §7 | Management creds vs DB creds separated; project-scoped API keys in approved secret store; never org-wide keys to agents; never Mgmt keys in app envs; distinct creds per domain; least-privilege app+migration roles; never owner/`neon_superuser` as app cred | NONE | ADR-010 | T1–T5 | Credential-domain audit (LIVE) |
| R317 | 28.3 §7 | Verify inheritance; investigate protection+regeneration; rotate/replace inherited passwords before untrusted access; CI/agent cannot reach prod; no secrets in logs/commits/artifacts/checkpoints | NONE | ADR-010; N8/N9 proofs | S1+T2/T4 | Inheritance proof + rotation proof (LIVE) |
| R318 | 28.3 §8 | Document + design for plan capabilities: concurrency, project/branch limits, charges, compute, storage/history, scale-to-zero/resume/autoscaling, expiry, deletion, recovery/retention | NONE | ADR-013; N14/N15 proofs | S1+T2/T3 | Cost model + live quota behavior (LIVE) |
| R319 | 28.3 §8 | Explicit expiry on API/CLI branches (never assumed); reconcile expired/abandoned/partial resources | NONE | ADR-013 | T2/T3 | Expiry + GC proofs (LIVE) |
| R320 | 28.3 §9 | Taskfile commands: auth verify, project preflight, dev-branch provision, env config, migrate, integration test, CI-branch lifecycle, preview-DB lifecycle, diagnostics, reconcile. Taskfiles invoke `studio dev`; Neon logic in Go/controller, not shell | `Taskfile.dev.yml` partial | ADR-006 | T1–T3 | Task inventory exercised (LIVE) |
| R321 | 28.3 §10 | Update master plan, parity report, `pg_muse.md`, setup docs, CI/preview docs, ADRs; remove stale local-PG/Docker refs | Partial | S1.6 | S1 | Diffs reviewed; stale refs gone |
| R322 | 28.3 §10 | 10 ADRs (project boundaries → failure recovery) | `decisions/001–003` | ADR-004–018 (S1.5) | S1 | Records published |
| R323 | 28.3 §10 | Executable acceptance proofs (9 listed: local-without-server … same contract) | NONE | Per-track plans (S1.8) | T1–T5 | All green (LIVE); design alone never counts |
| R324 | 28.3 §11 | Governing principle: Neon = PG layer; Fly = compute/execution; Go = behavior; Taskfiles = workflows; GitHub = review. Exceptions need owner-approved amendment | NONE | ADR-004 | S1 | Principle ratified in ADR-004 |

## Double-coverage notes (28.2 vs 28.3 overlap, resolved)

- 28.2 §3 local-PG setup vs 28.3 §4 Neon setup → 28.3 wins (R203); 28.2 §3 retired.
- 28.2 §4.2 digest CI image vs 28.3 §5 Neon CI branch → 28.3 wins (R210); digest language retired for DB layer.
- 28.2 §5.2 cluster-per-PR vs DB-per-PR vs 28.3 §6 branch-per-PR → 28.3 wins in mechanism; the isolation-vs-cost
  evaluation duty transfers to ADR-009/ADR-005 (branch-per-PR vs project-per-PR).
- 28.2 §5.5 object storage vs 28.3 §12 note (branch-aware storage) → both feed ADR-018; adoption separately justified.
- 28.2 §11 ADRs vs 28.3 §10 ADRs → merged into ADR-004–018 (S1.5); no topic dropped (coverage map in S1.5 section).
