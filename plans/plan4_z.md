# Approval Record — Miniapp Studio Implementation Plan

**Document reviewed:** `upload/miniapp-studio-implementation-plan.md` (2026-11-24, planning-only revision per `plan2_review.md` §4 + `prompt2.md` §5G)
**Verdict:** ✅ **APPROVED — no foundational or structural blockers found.**
**Cross-checked against:** framework code read earlier in this session (commit-level: registry, runtime/service.go, approvals, effects, executor/classa, deploy/targets.go, empty `fly/`/`tokens/`/`executor/river/` dirs, `sprites-go` v0.2.2 in go.mod only), the live Sprites Connectors OpenAPI spec and ecosystem page (fetched 2026-11-24), and `upload/hacksgiving.md` rules.

## 1. Cross-check results (plan claims vs. independently verified facts)

| Plan claim | Verified in session | Result |
|---|---|---|
| Static registration only → ONE generic runner is the correct move | `registry/registry.go` static; no dynamic registration | ✓ consistent |
| Class A sync executor implemented; River/DBOS runner path absent | `executor/classa/classa.go` present; `executor/river/` empty dir | ✓ consistent |
| Approvals park → `DecideApproval` → `Status` re-query, intent hash + expiry | `approvals/` + `runtime/service.go:parkApproval` + REST verb `decide` (`cmd/miniapp/main.go:68`) | ✓ consistent |
| Effect ledger PENDING→ATTEMPTING→CONFIRMED/FAILED/UNKNOWN, no blind retry | `effects/`, `contract/run.go` | ✓ consistent |
| Deploy seam is interfaces only (`SpriteTasks`); no Sprite/Machine/connector code yet | `deploy/targets.go`; `fly/`, `tokens/` empty | ✓ honestly disclosed |
| `auth_method` enum header/custom_header/query_param/url_path | OpenAPI `CustomApiConnectionRequest.auth_method` | ✓ exact match |
| `AccessPolicy`: allow_all (never default-on), endpoint wildcards, block-before-allow, name_prefix, sprite_labels; DB connectors reject endpoint rules | OpenAPI `AccessPolicy` schema | ✓ exact match |
| Custom API create/test endpoints; create does not test; test stores nothing | `POST /v1/oauth/connections/custom_api` + `/test` (separate, with `connection_id`) | ✓ consistent |
| Connector catalog display: 3-state model, integrations/plugins/MCP/SDKs never shown as connectors | Ecosystem page scrape 2026-11-24 | ✓ consistent |
| Durable workflows (DBOS authoring, waits/wake, sagas, replay) = roadmap §12.2, MVP §4.2 boundary forbids claiming them | Matches framework reality (DBOS exists but runner is Class A) and the owner's no-overclaiming requirement | ✓ consistent |
| Hacksgiving rule compliance: builder agent on a real Sprite + real connector call (G1); pinned dep + disclosure + organizer filing (§11, P6); deadline Nov 29 | `hacksgiving.md` | ✓ satisfied by design |

## 2. Conditions attached to this approval (all are the plan's own gates)

1. **P0 pin gate is hard:** no implementation may rely on pin `73351a8` until the §2.1 CI suite (gofmt/vet/test/-race, `TestSleepAcrossRestart`/`TestProbeRestart` executed-not-skipped, on PG19) and the Studio import spike pass. If the gate fails, the plan's own mitigation applies (fix or re-pin before P1/P3).
2. **Spike-decided items remain spikes:** URL topology (a/b/c) and MPG/Neon are decided by P3 decision records, not by this approval; §13 items 2–5 stay owner decisions.
3. **§13.6 respected:** this approval authorizes implementation planning→execution; repo init/module scaffolding still requires the owner's explicit go signal (given here as: **yes, proceed with P0 scaffolding**, per owner request to approve).
4. **Claims discipline is binding:** §4.2 boundary language ("crash-rollback + replay convergence is not durable workflow recovery") and the P6 claims audit are release gates, not suggestions — the demo and README may only claim what G0–G12 evidence shows.

## 3. Watch items (not blockers)

1. **Schedule compression is the dominant real-world risk.** Five-plus days remain before the Nov 29 midnight PST deadline, and the plan's process appetite (12 gates, decision records, full CI re-run, six screens) exceeds what a solo hackathon sprint usually absorbs. The §9 cut line (one complete app) and P1∥P3 parallelism are the correct mitigations — enforce the cut line ruthlessly; prefer G7 (demo readiness) over G12 polish if forced to choose.
2. **Connector-demo anchor depends on org catalog state** (Box availability unconfirmed) — the plan already checks live catalog day one with Custom API fallback (§11). Keep the fallback rehearsed, since it doubles as the extensibility demo.
3. **Full-access token requirement** for custom-API control-plane operations (403 otherwise) must be tested in the authorized Fly environment early (P3), since it gates the wizard demo.
4. **Checkpoint/restore is not in the golden path** (§7.2); it remains an optional demo flourish only if G7 is already green. Do not let it displace the ten golden-path steps.
5. **Internal cross-references** (`plan2_review.md` §3/§4, `prompt2.md` §5G) were taken at face value; they are the owner's governing documents and were not independently audited in this review.

## 4. Scope of this approval

Approves: the plan's architecture (§3), Class A MVP execution contract (§4), Fly topology approach with spike-gated selection (§5), connector three-state model + Custom API wizard (§6), UI/golden-path scope (§7), security analysis (§8), sequencing (§9), gates (§10), and roadmap delineation (§12) — as the governing plan for implementation, subject to §2 above.

Does not approve: any extension of the MVP contract beyond §4.2 (durable claims, undo-of-world, managed-agent adapters), scope additions after Nov 25, or modification of frozen governing documents (`prompt.md` stays frozen per §1.2).
