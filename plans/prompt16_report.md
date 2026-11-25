# Phase B Part 2 — GO Readiness Report

## 1. Readiness verdict: BLOCKED

Two concrete doc findings (both one-line fixes, both need owner authorization since this task is read-only). Everything else assessed is green. This is not authorization — readiness ≠ GO.

## 2. Evidence table

| Prerequisite | Status | Source |
|---|---|---|
| Tree/branch understandable | ✅ Verified-current | `main`, 3 commits; delta = prompt15 docs + pre-existing `plan_ui_z.md` M |
| Pins documented + consistent | ✅ Verified-current | Runbook §2 = setup.sh (v1.19.6, sha256, plugin commit) = go.mod (sprites-go v0.2.2, miniapp 73351a8) |
| §3 pre-live gates fresh | ✅ Verified-current | gofmt empty, vet 0, build 0, 13/13 ok on PG19 19.0, this turn |
| F-R1/F-R2/F-R3 present | ✅ Verified-current | Read in place (runbook §6/§11, closure §7) |
| N2/N5/N8 at point of use | ⚠️ Blocked | N2 §6.5 banner + N5 §11 + N8 §6 all present — but §12 upgrade row contradicts the N2 ban (finding B2) |
| §2 tree-state expectation | ⚠️ Blocked | Says "no commits, expect all-untracked"; repo has 3 commits (finding B1) |
| Cleanup/evidence §§13–14 | ✅ Verified-current | Exact commands + empty-list proof + evidence list |
| Race suite, P1–P4, targeted DB sets | Previously-tested, not re-executed | prompt12/14 records; not claimed fresh |
| All live proofs (R1–R10, EACCES, OAuth…) | Live-unverified | By definition pre-Part 2 |
| D1–D6 | Owner-decision-required | See packet |

## 3. D1–D6 owner decision packet

- **D1 Fly org** — needed: org name for Builder Sprites. Blocks: start (§6 provisions into it). Consequence: cost + trust boundary owner.
- **D2 Model provider + auth** — needed: provider + owner-provisioned auth method. Blocks: agent runs (§10; provisioning works without it). Consequence: spend + data egress to provider.
- **D3 Studio reachability** — needed: (a) owner machine + tunnel URL, or (b) Fly-hosted Studio URL. Blocks: §9 onward. **Largest dependency:** sprite must POST callbacks to Studio. Consequence: tunnel = owner-operated ingress; Fly-hosted = deployed surface.
- **D4 Retention** — destroy-to-zero (default) or retain named builder + cost owner. Needed by §13.
- **D5 Packages** — preinstall vs live apt-install; discoverable in §7. Needed at §7 on mismatch.
- **D6 Pins** — hold listed pins; any drift needs explicit re-pin approval. Needed at §2.

No credentials requested; org secret/token stay owner-held 0600 files.

## 4. Remaining blockers (only these)

- **B1 — Runbook §2 stale tree-state expectation.** Line 40 says "repo has no commits yet: expect all-untracked tree"; repo has 3 commits, so the owner's first command already contradicts the book. Fix (owner-authorized one-liner): expect `main` at `de85523` + the known uncommitted prompt15 docs, record verbatim.
- **B2 — §12 upgrade row contradicts N2 ban.** "upgrade: live agent run → repeat with --force deliberately" invites the prohibited live upgrade. Fix: "STOP and report — §6.5 upgrade prohibited live until N2 closes."
- Observation (not blocking): §9 "start it per the standard deploy" never spells `go run ./cmd/studio serve` (+ §5 env); recommend one clarifying line with B1/B2.

## 5. GO checklist

F-R1/F-R2/F-R3 present ✓ · B1+B2 doc fixes applied · D1/D2/D3 resolved · D6 confirmed + pins re-verified at execution · §3 green at execution · N2/N8 prohibitions accepted · D4/D5 resolved at their steps · explicit owner GO uttered.

## 6. Recommended next action

Reply with: (a) authorization for the two one-line runbook fixes (or apply them yourself), (b) D1/D2/D3 values, (c) the explicit sentence authorizing the Part 2 live run. Until then Part 2 stays unauthorized; no live commands ran in this assessment and the tree is byte-identical to snapshot (verified via before/after `git status`).
