# Card 002 execution plan — B2 live proofs (planning only)

Planning artifact for `plans/cards/002-b2-live-proofs.md`. No code, no runbook/card edits, no live
operations, no credentials, no commits. Claim tags: VERIFIED (path + line), ASSUMPTION, PROPOSAL,
OWNER-DECISION.

## 1. Input baseline (Phase A, 2026-11-28)

| Input | Version |
|---|---|
| Tree HEAD | `a40c14d` (Card 002 prep; 1 commit ahead of `origin/main`, unpushed at audit time) |
| Runbook | `docs/gateway-phase-b-live-runbook.md` @ local HEAD (prep §0b/§3/§5b/§9 included) |
| Card 002 | `plans/cards/002-b2-live-proofs.md` @ local HEAD (Status: `draft`) |
| Card 001 | `done`; log `test-run-card001.log` present, 43,550 bytes, 7 PASS / 0 FAIL re-counted |
| Roadmap | `plans/miniapp-studio-ultimate-roadmap.md` (52 KB; B2/H/exit-evidence sections) |
| Gateway code | `gateway/`, `tools/build_provision.go`, `tools/provisiondata/setup.sh`, `cmd/studio/gateway.go`, `server/` @ HEAD |

Local-vs-`main` rule used throughout: conclusions cite the LOCAL tree above; anything that differs on `main`
(the prep commit) is flagged, never silently mixed. `plans/cards/TEMPLATE.md` was briefly unlisted
mid-session but is present — input anomaly resolved, no action.

## 2. Audit summary (Phase B)

Re-verified every runbook command, flag, pin, env var, expectation, and recovery row against HEAD and
installed CLIs (`--help` surface only; no authenticated/mutating invocations). The prep commit's own
changes were re-derived, not grandfathered.

VERIFIED (sample — full list in Card 002 §7): §2 pins (`tools/provisiondata/setup.sh:13-18`,
`go.mod` sprites-go v0.2.2/miniapp 73351a8, `tools/provision_test.go:352-354`); `sprite`
info/exec/console/destroy/list + `-s`; `opencode` mcp/models/auth incl. `login --provider`,
`models --verbose --refresh`; gateway provision/upgrade/command flags
(`cmd/studio/gateway.go:43-47,92-95,141-145`); §5 env incl. `_FILE` convention
(`server/config.go:63-106,142-156`); routes (`server/routes.go:19-23`); readyz shape
(`server/routes.go:72-90`); §12 strings (`setup.sh:30`, `build_provision.go:230`,
`gateway/server.go:217,496,533,539,547,564`); §10 payload keys (`gateway/server.go:460-482`);
fetch_artifact 400 (`gateway/server.go:705-709,853-861`); §11 continue schema
(`gateway/server.go:550-552`); setup.sh modes 0700/0600 (`setup.sh:51-78`); `runuser -u` syntax;
§15 backlog still open (`server/server.go:79-82`).

Findings requiring action (all else passed):

- F1 (PROPOSAL, minor): §12 row "§9 probe 2 ≠ 401 → 404 = unknown build (wrong BUILD id)" is wrong
  under the production `KDFProvider`, which derives for ANY non-empty build id
  (`gateway/keys.go:115-119`) — a wrong BUILD id yields 401, indistinguishable by design.
  404 means the envelope lacks `build_id` (botched probe JSON). Proposed correction: "404 =
  envelope lacks build_id (re-send exactly); NOTE: any non-empty build id derives, so verify
  BUILD from §6 output — the probe proves auth enforcement, not build resolution."
- F2 (VERIFIED, closes a scare): serve DOES wire callback keys — `server.New` falls back to
  `KDFProvider` from the org secret (`server/server.go:131-133,165-170`, fed by
  `STUDIO_GATEWAY_ORG_SECRET[_FILE]` via `server/config.go:106-110`). Configured serve → 401
  path live; missing secret → warn + 503. The §9 design (401 vs 503 discrimination) is coherent.
- F3 (VERIFIED gap): the runbook never mentions lifecycle/handoff receipts (0 matches) though
  B2-01B built them and the roadmap Phase 5 exit evidence names "handoff receipt local↔Sprite".
- F4 (VERIFIED collisions/scope): (a) roadmap Phase-0 gates (D1 verification, D2, D3, D4) share
  labels with runbook owner decisions D1–D6 but mean different things — evidence must prefix
  `runbook-Dx` vs `roadmap-Dx`. (b) Roadmap Phase 5 exit evidence is broader than the runbook:
  real-task dispatch (runbook: BUILDER-ALIVE no-op), checkpoint/restore demo (runbook: idle
  sleep/wake observation only), Workspace Preview label, Tasks-API holds. These are NON-GOALS
  for Card 002 unless the owner expands scope — recorded in §7, not silently adopted.

STALE: none found. UNVERIFIABLE-LOCALLY (must be observed live, with honest-unknown handling in
the runbook): exec-channel privilege (§7), service EUID (§7a), agent-uid channel (§7b fallback),
OAuth rung behavior (§8 P-E1→P-E3), model catalog + costs (§8b step 4), warm-vs-cold (§11),
first-wake latency. Each has a STOP/record path; no blind trust anywhere.

## 3. Reconciliations (Phase C)

**R1 — §3 vs Card 001 (full suite stays deferred).** The B2 gate = repo-local suite (covers the
fake-backed gateway tests that prove refusal/plumbing) + Card 001 citation-or-rerun (covers the
live DB/migration/privilege/store surface Studio + provisioning touch). Coverage argument per
stage: §6 provision needs store migrations + keys KDF (Card 001); §7–§8b need provisioned
service + CLI surface (repo-local + pins); §9–§11 need callback auth + session protocol
(repo-local unit + live proof itself). What the gate does NOT prove (stated, not waived):
full cross-package live suite, race/stress behavior, or any gateway live path — those ARE Card
002's object, so requiring them beforehand would be circular. Full-suite-green remains a
pre-demo gate owning its own card.

**R2 — D3 (no invented deployment).** VERIFIED no in-tree Studio-on-Fly procedure exists (no
`fly.toml`, only `ci.yml`; deploy mentions live in historical `plans/` docs, not executable
paths). Deployment stays owner-established outside the runbook; §5b is its acceptance gate
(record + Proxy listener + double-200), §9 proves the route. No tunnel substitution without a
new OWNER-DECISION.

**R3 — Boundaries carried into every stage below:** owner-held credentials (never in evidence);
restricted OAuth grants (builder prefix, creation limit 0); provider auth separated from Sprite
OAuth (distinct stores); N8 re-provision window checks before every re-provision with live
agents; N2 — live `gateway upgrade` PROHIBITED, no GO may authorize it, drift means STOP.

## 4. Sequenced execution plan

Party: OWNER = owner terminal (all live steps); AGENT = review output, verify vs expectation, record.
Log: one gitignored `test-run-card002.log` (`set -o pipefail`, `tee`), quoted verbatim in evidence.

| Stage | Prereq | Party + action (exact) | Expected | Evidence | Stop | Recovery |
|---|---|---|---|---|---|---|
| S0 preflight | Card 002 `authorized` | OWNER confirms §0b class of each step; AGENT confirms HEAD + inputs match §1 baseline AND the baseline commit is published and reachable (fetch + `git cat-file -t`) | Classes + versions + reachability recorded | §1-equivalent table | HEAD moved, inputs differ, or baseline commit unpublished/unreachable → STOP, re-baseline with owner present; never rely on unreviewable local-only evidence | Re-run audit on the published baseline |
| S1 pins/tree (§2) | S0 | OWNER runs §2 block verbatim | v1.19.6 only; branch/SHA/status owner-recognized | Full outputs + status | Any drift → STOP (D6: re-pin decision) | Re-pin decision, then restart S1 |
| S2 B2 gate (§3) | S1 | OWNER: fmt/vet/build, `env -u TEST_DATABASE_URL go test -p 1 ./...`, `command -v sprite`; Card 001 cite (HEAD `0c027ae`) or rerun 7-test cmd | All ok; 7 PASS if rerun | Suite tail + skips; gate log if rerun | Any red → STOP, root-cause triage | Fix + re-gate; no proofs on red |
| S3 build (§4) | S2 | OWNER: `CGO_ENABLED=0 GOOS=linux go build -o /tmp/studio-gateway ./cmd/gateway`, `file` | Static x86-64 ELF | `file` line; binary path | Non-static → STOP | Toolchain diagnosis |
| S4 env (§5) | S3 | OWNER assembles exports; generates org secret 0600 if new | All vars set, files 0600, CLI never echoes secrets | Var NAMES + perms only (never values) | Missing/unreadable → STOP | Fix env, re-check |
| S5 Studio ready (§5b, verifies C0) | GO-C0 executed + deployment record exists | OWNER records app/rev/URL/port; confirm `0.0.0.0` listen; `curl` healthz + readyz | Double-200 | App/rev/URL/listener/outputs | Non-200 → STOP, no provisioning. S5 authorizes nothing — it verifies C0's result only | Fix deployment under a new GO-C0 if it changes anything, re-probe |
| S6 provision (§6, C1+R) | S5 + C1 GO | OWNER: UI message (BUILD id), `gateway provision …`, `sprite info`, `sprite exec` verifies, staging hygiene, attach-path re-provision | JSON verbatim; oracle matches; staging clean 700/600 | All outputs verbatim | Any mismatch/stop-row → §12 | Attach converges; drift → STOP (N2, never upgrade) |
| S7 discovery (§7, R+C2) | S6 | OWNER: `id -u`, tool probe; C2 GO only if D5 install needed; re-provision retry check | Root-or-refuse honestly; same sprite on retry | Verbatim outputs | No-privilege → STOP (correct refusal) | D5 install → re-provision → re-verify |
| S8a identity, no-agent-run proofs (§7a/§7b part, R) | S7 | OWNER: service EUID from `/proc`, agent UID/GID discovery, direct `runuser` key-file/dir denial checks where supported. NO inbox matrix, NO supervised turns here | EACCES everywhere required; EUID/GID recorded | Exit codes + stderr verbatim | Any deviation → STOP, never weaken | Diagnose boundary |
| S9 OAuth (§8, C3) | S8a + C3 GO | OWNER: P-E1 → P-E2 → P-E3 in order, restricted grant | One rung works; prefix + 0-limit verified | Rung, grant params, redacted context | All fail → STOP, no 4th rung | Report; no workarounds |
| S10 provider (§8b, C4) | S9 + C4 GO | OWNER: console → `su studio-agent` → HOME/XDG gate → catalog → `auth login --provider openrouter` → verify 0600 | Byte-identical dirs; zero-cost primaries; key 0600 agent-owned | Catalog verbatim; limit + revocation plan (never key) | Any mismatch → STOP before key entry | Report envs; never wrong-home entry |
| S11 route (§9, R) | S10 | OWNER: probe 1 (expect 400), probe 2 (expect 401), `status`. SOLE AUTHORITATIVE interpretation (overrides runbook §12 wording for this execution): 404 = envelope lacks `build_id` (re-send exactly); any non-empty build id derives under KDF so a wrong BUILD id still yields 401 — verify BUILD from S6 output; the probe proves auth enforcement, never build resolution | 400 + 401 + running JSON | Codes + JSON + latency | 503 = keys unconfigured; timeout/refused = D3 broken → STOP | Fix cause, re-probe |
| S12 no-op (§10, C5) | S11 + C5 GO | OWNER: exact `assign` (BUILDER-ALIVE, D2 model); then `fetch_artifact` empty manifest | started→stream→idle→completed; fetch 400 | Timestamped callbacks; provider/model/version | Auth fail → STOP (not evidence) | §12; no fallback claims (no auto-failover exists) |
| S8b inbox matrix, post-assign (§7b part, R) | S12 first success | OWNER: inbox dir/mode check (2750/2730 vs EUID/GID), agent write + delete-own-drop probes. Must complete before S13's idle window (probes are commands) | Modes match; own-drop writes + deletes | Modes + probe results verbatim | Any deviation → STOP, never weaken | Diagnose boundary |
| S12b second-build continuity, post-no-op (§6 key refresh, C1+R) | S12 + C1 GO | OWNER, with N8 window checks first (staging 0700, no live untrusted siblings; drain + re-verify if needed): provision a second build on the same builder, then `status` the FIRST build | Keys refreshed WITHOUT service restart; first session undisrupted (status answers, no new `agent.failed`) | N8 pre-checks + drain record (if any), both provisions, first-build status before/after | First session disrupted or staging window violated → STOP | §12; report verbatim |
| S13 sleep/wake (§11, R+C5) | S8b + S12b | OWNER: record PIDs, 90s idle, status, warm/cold branch (orphan-PID absence check if cold + fenced `continue --turn N+1`), repeat @10min | Either outcome recorded; cold ⇒ failed-callback + zero orphans + resumed | Timing table both runs | Orphan PID survives → STOP | Report; never claim cleanup from callback alone |
| S8c answer.json protections (conditional, R) | A yield/continue cycle or second turn has completed (e.g., S13 cold resume) | OWNER, ONLY if `answer.json` exists: verify 0640 + agent readability + overwrite/remove denials. If no second turn ever occurs, record answer.json as NOT PROVEN with reason — carried, never waived or assumed | 0640 + readable + denials, or explicit NOT-PROVEN carry | Check results verbatim or carry record | Denial missing → STOP | Diagnose boundary |
| S8d agent-UID fallback (conditional, C5) | S8a direct channel UNAVAILABLE + S12 first success + C5 GO | OWNER, ONLY if no agent-uid exec channel exists: two supervised turns on a scratch build — (a) report `id -u/-g/-G` verbatim, (b) attempt keys-dir read, report exact error. Evidence grade: AGENT-REPORTED (weaker — supplements, never replaces kernel-observed denial). If S8a direct proofs passed, record NOT-APPLICABLE with pointer — never claimed either way | Agent uid/gids (not service) + permission-denied, or NOT-APPLICABLE | Transcripts verbatim + grade label | Wrong uid or keys readable → STOP | Diagnose boundary |
| S14 cleanup (§13, C6) | Any end-state + C6 GO | OWNER: `sprite destroy` (confirm prompt, no `--force`), empty-list proof | Destroy output + no match | Both verbatim (or D4 retention record + cost owner) | Anything remains → report, never force-clean | Manual reconcile under new GO |
| S15 evidence (§14) | S14 | AGENT assembles: pins, gates, §5b record, §9 probes, GO record, timings, transcripts, §12 hits, deviations | §14 list complete | Card 002 §7 report | Gaps → carry explicitly with blocker | — |

## 5. Seven-field decision brief

1. Execution location: owner terminal for every live byte (S1–S14); agent prepares, verifies each proof
   against its expectation, records evidence. No agent-side live execution, ever.
2. Application target: Builder Sprite + `/tmp/studio-gateway` binary + Phase B integration surface
   (provision/service/OAuth/provider/callback/turns).
3. Database target: Neon dev branch; run-pattern DBs for any DB touch (S2 rerun only — the runbook proper
   touches Studio's DB through normal serving, no migrations, no manual DDL).
4. Infrastructure actions: exactly the runbook's C-classes — C0 deploy (outside runbook), C1 provision,
   C2 package install, C3 OAuth grants, C4 provider key, C5 agent runs, C6 destroy/retention. Nothing else.
5. Authorization class: R under Part 2 GO; EACH C-class step — C0 included — needs its own explicit
   owner GO at execution (one GO never covers another); S5 verifies C0's result and authorizes nothing;
   §6.5 upgrade PROHIBITED until N2 closes; no commits/pushes. For execution, this plan's register is
   authoritative where runbook §0b's C0 "prerequisite" wording is narrower (follow-up: §0b touch-up).
6. Verification/evidence: §14 list mandatory (pins, gates, §5b record, §9 probes, GO record, timings,
   redacted transcripts, §12 hits, deviations); agent marks nothing green it did not verify.
7. Recovery/rollback: S14 always runs; §12 interprets; partial state reported, never force-cleaned;
   second-build key refresh (§6) and attach-path convergence are the only in-runbook self-heals.

## 6. Dependencies + owner decisions / GOs still needed

- SATISFIED: Card 001 green (cited by S2). OPEN + BLOCKING: C0 Fly-hosted Studio deployed (§5b-passing).
- GO-C0 (standalone, FIRST-class): explicit owner authorization for the Fly-hosted Studio deployment
  itself — performer, app/revision/URL/port evidence recorded in S5. No procedure invented or performed
  by the plan/agent.
- Per-class GOs (each asked at execution, none pre-granted): C1 (S6, S12b), C2 (S7 iff D5 install),
  C3 (S9), C4 (S10), C5 (S12, S8d iff invoked, S13-resume), C6 (S14). Plus: D5 package route at execution;
  any D6 re-pin decision if §2 drifts.
- Card assignment / plan approval / `authorized` status / GO-A NEVER authorizes deployment or live ops —
  only the per-class GOs do. F1 runbook wording is a post-plan follow-up (S11 carries the sole instruction).

## 7. Blockers, non-goals, open proposals

- BLOCKER (only hard one): C0 — no Fly-hosted Studio exists in-tree or verified; S6+ cannot start without it.
- PROPOSAL F1: §12 probe-2 404 wording (see §2) — adopt before S11 or interpret live with this plan beside.
- NON-GOALS (roadmap deltas, need their own scope if wanted): real-task dispatch, checkpoint/restore demo,
  Workspace Preview label, lifecycle-receipt live proof (F3), Tasks-API holds. S12–S13 prove the supervised
  turn + wake/resume machinery those would build on — no more, no less.
- ROADMAP-EXIT CARRYOVER REGISTER (explicit non-implication): closing Card 002 does NOT declare the
  roadmap's B2/Phase-5 exit criteria complete. Each item below stays TRACKED until demonstrated live or
  explicitly dispositioned by the owner: (1) real engineering task dispatched Studio→Sprite→back;
  (2) checkpoint-failure-recovery demonstrated; (3) Workspace Preview shown and labeled;
  (4) handoff-receipt live proof; (5) Tasks-API holds observed. Disposition requires a named owner
  decision per item — silence keeps them open.
- Standing: full-suite-green stays pre-demo; N2 stays closed; roadmap-Dx vs runbook-Dx labels must be
  prefixed in all evidence (F4a).

## 8. GO/NO-GO recommendation

**CONDITIONAL GO — segmented; S0–S5 is NEVER one segment.** NO-GO on any C-class step without its
execution-time GO, and NO-GO on §6.5 always. **GO-A: S0–S4 only**, as soon as Card 002 is `authorized`
(local preflight, pins/tree, B2 gate, build, env prep — no live infra touched). **GO-C0: separate explicit
authorization** for the Fly-hosted Studio deployment (owner-performed, procedure outside the runbook).
**S5** runs after GO-C0 as verification-only. GO S6–S10 (C1–C4) once S5 passes, one class GO at a time.
GO S8b/S8d/S11–S15 (R + C5 + C6) on the §9 green + C5/C6 GOs; S8c rides the first second turn or
carries explicitly; S12b runs post-S12 under its own C1 GO. S12b and S8d are explicit verification
items — neither is claimed passed unless actually demonstrated (NOT-APPLICABLE / NOT-PROVEN with
reason where conditional). Single critical path: C0 deployment. If C0 is delayed, S0–S4 stay valid as staging evidence
(re-verify §2 freshness at execution per D6).
