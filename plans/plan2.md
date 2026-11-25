## Goal

Implement Miniapp Studio per [plan.md](/home/chaschel/Documents/go/miniapp-studio/plans/plan.md) (v3 direction, governing) and [prompt.md](/home/chaschel/Documents/go/miniapp-studio/plans/prompt.md) (v2 agent prompt, governing) in a new module `github.com/PithomLabs/miniapp-studio` under the studio folder: a first-party Miniapp on Fly that generates declarative MiniappSpecs via a Sprite-hosted builder agent, validates them deterministically, renders one golden-path app (Invoice Review Desk) with its own Fly-issued URL, through at least one real Sprites connector. I read all 24 plans files; v2/v1 and the per-model plans are superseded context, reconciled into v3 already.

## Success Criteria

- Gates G0–G12 from plan.md §11/§13.6 pass with named test evidence; one complete generated app owns a stable, slug-derived, persisted Fly URL (G10).
- Studio dogfoods the framework: its UI/API go through `runtime.Service` (`Invoke`/`Status`/`DecideApproval`/audit) and static `registry` registration like [crmread.go](/home/chaschel/Documents/go/miniapp/miniapps/crm-read/crmread.go); no bypass backend (G8).
- No raw provider or Fly-org secret in Sprite, spec, prompt, log, or artifact; connector catalog shows public/org/Sprite states separately (G1–G3).
- Demo claims match passing tests; DBOS stays roadmap (G7).

## Approach

New repo consumes the framework as a **pinned dependency** (`73351a8` or newer after review; miniapp builds clean, verified `BUILD_OK`), never a code copy. Verified framework truth shapes the plan: `fly/`, `tokens/`, `capability/clients/`, `executor/river/` are empty; `deploy/` is frozen interfaces only; `sprites-go` v0.2.2 is an unused go.mod entry (I inspected the SDK: Sprite CRUD, exec, filesystem, services, checkpoints, network policy, URL settings — **no connector/gateway APIs**, so Custom API needs a separate typed REST client); no MiniappSpec/DSL, no UI serving, REST is JSON `invoke|status|signal|cancel|decide`. So Studio owns: spec schema/validator, generic runner miniapp, renderer, Fly adapters, UI. Framework owns: invocation, auth, approvals, effects ledger, audit, Postgres/DBOS.

## Steps

1. **Scaffold studio repo**: `go.mod` (`github.com/PithomLabs/miniapp-studio`, go 1.26), `require miniapp vX.Y.Z` pin via replace-then-tag decision, CI mirroring [ci.yml](/home/chaschel/Documents/go/miniapp/.github/workflows/ci.yml) (gofmt, vet, `go test -p 1` on PG19), `docs/architecture/miniapp-studio-implementation-plan.md` as the prompt.md §11 deliverable.
2. **Fly verification spikes** (prompt.md §10, block topology): Sprite agent + real connector call; control-plane Custom API create/test; framework import on Sprite/Machine; MPG vs Neon compat with pgx pool + DBOS direct-conn requirement; autostop/autostart cold-start budget. Compare URL topologies (Sprite-per-app vs shared router vs app-per-app) and record the decision.
3. **MiniappSpec + deterministic validator**: one versioned schema (`app/data/ui/workflows/capabilities/policies/fixtures/presentation/metadata`), content hash, validator rejecting unknown components/ops/paths (G4).
4. **Generic runner miniapp + renderer**: one statically registered interpreter over `runtime.Service`; smallest coherent web stack (server-rendered; no SPA framework without evidence); fixture/dry-run with no silent writes (G5–G6).
5. **Fly adapters**: `sprites-go` behind Miniapp-owned interfaces; typed Connectors REST client with fakes/contract tests; slug normalize/reserve/collide lifecycle; URL persisted from authoritative Fly responses only (G10–G11).
6. **Studio-as-Miniapp + ecosystem UI**: Studio routes as first-party app definition; ecosystem explorer (connectors vs integrations vs SDKs); Custom API wizard (Define→Test→Scope); permissions card; run/evidence views (G2–G3, G8–G9, G12).
7. **Golden path + rehearsal**: Invoice Review Desk end-to-end, clean-state demo reset, README claims audit, organizer-eligibility email filed (G7).

## Validation Plan

- `go build ./...`, `go vet`, `gofmt -l`, `go test -p 1 ./...` per gate; contract tests for every Fly endpoint used; red-team abuse cases from prompt.md §12 (unapproved send rejected live, token-read-from-Sprite denied).
- Highest-risk validation: spike 2 (Sprite connector call + Custom API provisioning under real org privileges) and G10 (slug→real URL round-trip with collision/idempotency tests).

## Risks / Open Questions

- Scope vs Nov 29 deadline is extreme for dogfooding + per-app URLs + Custom API; cut line is §8.4 (one app, no canvas/marketplace/DBOS UI).
- Environment: `flyctl` exists but is unusable in my sandbox (read-only `~/.fly`); Fly spikes need unsandboxed runs or owner execution. Studio folder is not a git repo yet.
- Owner decisions needed before `go`: (1) framework pin revision + organizer-email sender; (2) MPG vs Neon default if spikes are inconclusive; (3) URL topology pick if Sprite-per-app limits/cost look bad; (4) which org-native connector anchors the demo if Box is unavailable; (5) confirm Studio repo may be `git init`'d here. Also confirm my reading that this turn's "plan.md/prompt.md" means *the governing spec to implement*, not files to rewrite — I left both untouched.
