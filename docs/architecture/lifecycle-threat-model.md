# Lifecycle threat model (Stage 1, S1.6)

**Scope:** the three lifecycles (agent / CI / preview) + controller + Neon + GitHub integration.
**Method:** each prompt28.2 §9 vector mapped to boundaries, controls (with ADR), and residual proof owed in Stage 2.
**Doctrine:** issue/PR/comment/repo content is untrusted input; agent-readable secrets are assumed compromised;
deny-by-default everywhere; fail closed.

| # | Vector (§9) | Primary boundary | Controls (ADR) | Residual proof (Stage 2) |
|---|-------------|------------------|----------------|--------------------------|
| T1 | Malicious issue/PR instructions | Trigger authorization | Approved-label + user + repo + eligibility checks (014); untrusted-content doctrine | Negative trigger tests |
| T2 | Prompt injection via repo content | Run isolation | Per-run Sprite + egress policy (008); least-privilege creds (010) | Containment tests |
| T3 | Malicious build scripts / migrations | Preview/CI isolation | Per-PR branch + serialized fenced deploys (009); synthetic fixtures (015) | Cross-PR contamination negatives |
| T4 | Cross-PR database access | Neon topology | Branch-per-PR + exclusivity binding (009); project separation (005) | Exclusivity audit |
| T5 | Credential extraction from agent processes | Credential domains | Short-lived narrowed tokens (011); rotation-before-use (010); no secrets in checkpoints (008) | Placement audit + hygiene scans |
| T6 | Repository / webhook spoofing | Webhook pipeline | HMAC-verify-then-process (012); ownership-verified destroy | Spoof + replay negatives |
| T7 | Replay / duplicate events | Idempotency | Delivery dedupe + idempotency keys + generation fencing (012) | Dup/out-of-order tests |
| T8 | Unauthorized preview access | Exposure policy | Authenticated-by-default previews (009); no security-by-obscurity | Access negatives |
| T9 | Production-network reachability | Project separation | Distinct prod projects + keys (005); preview egress cannot reach prod (009) | Network + config audits |
| T10 | Supply-chain compromise (deps) | Execution + data | Pinned toolchains (007); fixture determinism (015); egress policy bounds exfil (008) | Provenance + determinism checks |
| T11 | Malicious dependency install | Egress + isolation | Per-run policy with `defaults`-plus-explicit (008); fresh-or-verified-clean start | Policy enforcement tests |
| T12 | Resource exhaustion / runaway spend | Caps + reconcile | Pre-provision budget checks (013); per-run budgets (014); orphan GC (013) | Over-budget refusal proofs |
| T13 | Accidental production deploy | Promotion gates | Human digest-bound approval for prod paths (retained from prior phases); separate prod project (005) | Gate negative tests |

## Trust-boundary summary

Controller (trusted, owner-operated) ↔ Sprites/previews (untrusted code, least-privilege) ↔ Neon (project-separated)
↔ GitHub (review authority). Every arrow crossing a boundary carries exactly one credential domain (ADR-010), and
every consequential crossing is logged with identity, scope, and expiry.
