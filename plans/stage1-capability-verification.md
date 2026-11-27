# Stage 1 — Capability verification (S1.1 Fly, S1.2 Neon)

**Date:** 2026-11-27. **Method:** fetch + read official docs; in-repo prior evidence cited where used.
**Verdicts:** VERIFIED (official source inspected) · ATTESTED (reputable secondary or prior in-repo evidence) ·
ASSUMED (design may rely on it only with the named proof step) · DISPUTED (contradicted — design must avoid it) ·
UNVERIFIED (no usable source yet — proof step required).

## S1.1 — Fly / Sprites / GitHub

| # | Claim | Verdict | Source (checked 2026-11-27) |
|---|-------|---------|------------------------------|
| V1 | Per-PR isolated app + database + secrets + URL; redeploy per push; destroy on close; teardown/cost sections | ATTESTED | Fly Learn previews article — read in full by owner per review2 (2026-11-27); implementers re-read before relying on mechanics. Constraint: its GitHub Actions + Fly Managed Postgres examples must NOT be copied into our Neon-first, Fly-native implementation |
| V2 | `pull_request` default types are `opened`/`synchronize`/`reopened`; `closed` must be listed explicitly | VERIFIED | GitHub official events doc |
| V3 | Per-PR deployment serialization via Actions `concurrency.group` | ATTESTED | Article example; official re-verify at implementation |
| V4 | App installation token: JWT → `POST /app/installations/{id}/access_tokens`; `repositories`/`repository_ids` + `permissions` narrowing; 1-hour expiry | VERIFIED | GitHub official installation-token doc (read in full) |
| V5 | Webhook HMAC-SHA256, `X-Hub-Signature-256`, `sha256=` prefix, constant-time compare, published test vector | VERIFIED | GitHub official webhook-validation doc |
| V6 | Sprite CLI: create/list/destroy/exec/file/checkpoint/restore/proxy/url/config (+ exit codes, env vars) | VERIFIED | Official Sprites CLI reference (read in full) |
| V7 | `sprite exec` takes a timeout (30s default, 10min ceiling) | **DISPUTED** | No timeout flag in official CLI ref. Design uses sessions (list/attach/kill) + console detach + marker polling instead |
| V8 | Sprites pause when idle (no HTTP + no sessions); services keep alive + auto-restart on boot | VERIFIED | Official CLI reference (`sprite create` lifecycle notes) |
| V9 | Checkpoint create/list/info/delete + restore commands exist | VERIFIED | Official CLI reference. Checkpointable-disk requirement + warm-restore semantics: ASSUMED (proof: live test in Stage 2) |
| V10 | Network policy: DNS allowlist, outside-only setter, read-only inside, `REFUSED`, raw-IP + private-IP blocked, live reload, `{include:defaults}` | VERIFIED | Official Sprites networking doc (read in full) |
| V11 | Connectors: org-stored credential, gateway URL, deny-by-default, prefix/label/allow-all policy, endpoint allow+block lists (API-only), GitHub OAuth, Custom API, managed OpenRouter, callable only from inside a Sprite | VERIFIED | Official Connectors doc (read in full) |
| V12 | Sprite URL private-by-default; public = open internet with explicit warning | VERIFIED | Official networking doc. Shape `https://<name>-<org-id>.sprites.app` differs from ADR-001's recorded region-based shape → live-verify in Stage 2 |
| V13 | Sprite pricing $0.0385/CPU-hr + $0.021875/GB-hr | UNVERIFIED | Article-only; volatile. Proof: read pricing page at costing time. Spend caps required regardless |
| V14 | Sprites ship preinstalled agents/toolchains | ATTESTED | Article-only. Proof: inspect a fresh Sprite in Stage 2 |
| V15 | Sleep/wake with filesystem intact; wake on request | ATTESTED | Article-only (Services doc not inspected). Proof: live pause/wake test |
| V16 | Parallelism capped by org active-Sprite limit | ATTESTED | Article-only. Proof: org limits check before T2/T4 scale claims |
| V17 | `GITHUB_SHA` is the merge-branch SHA; head SHA via `pull_request.head.sha`; no runs on conflicted PRs; empty payload for merged/fork PRs | VERIFIED | GitHub official events doc. Impact: CI records head SHA from payload, never `GITHUB_SHA` |
| V18 | flyctl Machine/app ops surface | ATTESTED | In-repo prior evidence (flyctl 0.5.18, unified-lifecycle §7.5). Re-verify version at implementation |

## S1.2 — Neon

| # | Claim | Verdict | Source (checked 2026-11-27) |
|---|-------|---------|------------------------------|
| N1 | PG19 same-day 2026-11-19; supported 14–19; default 18; select-at-create; minors auto-apply on restart; no skip/downgrade; major upgrade = new project + migrate | VERIFIED | Official Neon version-support doc (read in full) |
| N2 | PG19 caveats: no Data API / Realtime / anonymization; some extensions unavailable incl. `anon` | VERIFIED | Same. Impact: extension inventory ("none") must be re-proven on PG19 |
| N3 | Branch = CoW clone, delta writes, zero parent impact, isolated, dedicated compute per branch, root `main`, sequence-copy ID caveat | VERIFIED | Official Neon branching doc (read in full) |
| N4 | Branch TTL/expiration for CI/ephemeral use | VERIFIED concept | Official branching doc links expiration guide; CLI `--expires-at` attested in-repo (neon 9.9.3). Full flag check at implementation |
| N5 | Schema-only branching exists | VERIFIED | Official branching doc |
| N6 | History window: 6h Free / 1d paid default; up to 7d Launch / 30d Scale | VERIFIED | Official branching doc |
| N7 | Roles are branch-scoped and duplicated to child branches; 500/branch; Console/API/CLI roles get `neon_superuser` unless `--access`; SQL-created roles minimal | VERIFIED | Official Neon roles doc |
| N8 | Child branches inherit parent role **passwords** | ASSUMED-CONSERVATIVE | Role duplication verified; password carry-over not explicitly stated. Fail-closed: assume carry-over + mandatory reset before untrusted access; executable proof required in Stage 2 |
| N9 | Branch `protected` flag; plan-gated protection; password regeneration on protect | PARTIAL | `protected` attribute exists in API shapes; plan gating + semantics NOT verified → conditional; proof: manage/branches doc + live check |
| N10 | Project-scoped API keys (org-admin-created, single project, shown once, immediate revoke); personal + org variants | VERIFIED | Official Neon API-keys doc |
| N11 | Role password reset/reveal via Console/CLI/API | VERIFIED | Official Neon roles doc |
| N12 | Object Storage / Functions / Auth branch with data (branch-aware backend) | VERIFIED concept | Official branching doc. Adoption needs separate justification (not covered by PG mandate) |
| N13 | neon CLI 9.9.3 surface; `branches create` defaults `--secrets=true` (prints credentials) → always `--no-secrets` | ATTESTED | In-repo prior first-hand evidence (unified-lifecycle §7.5). Re-verify version at implementation |
| N14 | Quotas, branch/project limits, pricing, concurrency capacity | UNVERIFIED | Proof: pricing page + account check before T2/T3 scale commitments |
| N15 | Scale-to-zero / resume behavior | REFERENCED | Official page exists, not inspected. Proof: read + live resume-latency measurement |

## Design consequences (binding on S1.4/S1.5)

1. No `sprite exec` timeout reliance anywhere (V7 disputed) — supervision uses sessions + markers + heartbeats.
2. CI records `pull_request.head.sha` from the webhook payload, never `GITHUB_SHA` (V17).
3. Connectors are Sprite-side only (gateway 401s off-Sprite calls) — the controller needs its own GitHub credential path (V11).
4. Least-privilege Neon roles must be SQL-created or `--access`-scoped; default Console/API/CLI roles are `neon_superuser` members (N7).
5. Branch credentials are treated as inherited until the Stage-2 proof says otherwise; rotation-before-untrusted-access is unconditional (N8).
6. Spend-cap and quota architecture must not hardcode V13/N14 figures — parameterized limits + live verification gates.
7. Fly article examples (GitHub Actions workflows, Managed Postgres per PR) must not be copied into the implementation (review2); the disputed `sprite exec` timeout stays excluded — sessions, markers, heartbeats.
