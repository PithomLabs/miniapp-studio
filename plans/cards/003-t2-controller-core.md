# Card 003: T2 controller core, repo-local (fake-backed)

- Status: `draft`
- Roadmap: A (B3 preview track, first slice — blocked on Cards 001–002)
- Owner decision required: GO for repo-local implementation + tests only. Explicitly NOT authorized: controller
  deploy, GitHub App install, Neon branch provisioning beyond the dev workflow, any live run.

## Outcome

The Fly-native CI controller exists as reviewed Go code behind the established interfaces: webhook
verify → dedupe → persist → schedule, Sprite supervision via sessions+markers, per-run branch lifecycle
model, checks publishing model, and reconciler + caps — all exercised against fakes with contract tests,
ready for a later live-wiring card.

## Acceptance criteria

- [ ] Controller implements the T2 work units (1)–(5) against fake Sprite/Neon/GitHub surfaces.
- [ ] Contract tests prove: dedupe/idempotency, SHA+generation fencing, failure/cancel honesty, orphan
  reconciliation, restart recovery (mirroring the gateway Part-1 pattern).
- [ ] `ci.yml` replacement designed but NOT applied (filed as the follow-up card's first step).
- [ ] Adversarial self-review recorded; no live credentials, endpoints, or provisioning paths in code.
- [ ] Docs updated (controller design + capability matrix with executed-vs-unknown cells, as in Part 1).

## Dependencies

- Cards 001–002 green (stable base + proven Sprite operations to model fakes on).
- ADR-007/008/012/013/017 remain proposed: implementation follows their current text; any deviation escalates
  instead of silently redesigning.
- Live wiring is a separate future card with its own authorizations (controller deploy, App install, N8 proof).

## Decision brief

1. Execution location: agent sandbox only.
2. Application target: new controller package + fakes; no existing behavior changes except additive wiring.
3. Database target: none (fakes only; migration-adjacent work explicitly out of scope).
4. Infrastructure actions: none.
5. Authorization class: repo-local implementation + tests; no commits/pushes; no live operations of any kind.
6. Verification and evidence: `gofmt`, `go vet ./...`, `go build ./...`, new contract tests + full unit suite
   green; self-review notes; no live evidence claimed.
7. Recovery and rollback: uncommitted work reverts by checkout; no persistent state created anywhere.

## Report (§7)

(Pending execution.)
