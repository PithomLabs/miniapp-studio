# ADR-001: Generated-app hosting and URL topology

**Status:** Decided 2026-11-24. **Decision:** Sprite per deployed Miniapp (option a) for the MVP.
**Basis:** live spike evidence in `docs/p3-evidence.md` (this repo) + governing plan §5.

## Context

G10 requires every published Miniapp to own a unique, stable, persisted
Fly-issued URL derived from the user slug. The shared-router option (b) was
already ruled non-compliant: a route is not an independently provisioned Fly
URL. This record decides between (a) Sprite per app and (c) Fly App per app.

## Evidence for (a) — proven live

- Provisioning: create in ~1.5s with labels; delete verified.
- URL: Fly-issued `https://<slug-derived>-<region>.sprites.app`, persisted
  from the API response; private (`sprite`/org) by default; public mode is an
  explicit, separate setting.
- Serving: service with `httpPort` running in ~6s; binary deploy
  (write + exec + service) proven; URL returned HTTP 200 from the service in
  ~1s from cold with bearer auth; anonymous callers get the login wall.
- Isolation/cost: hardware-isolated microVM per app; per-second billing, free
  when idle; warm/cold wake documented.
- Labels support per-Sprite connector access policies.

## Evaluation of (c) — static, rejected for MVP

Per-app Fly Apps would each need an image build/push/release pipeline and
per-app operational handling (deploys, rollbacks, certs, scaling config) that
does not exist in this codebase. No spike was run because the machinery to
spike does not exist; building it solely for comparison would exceed the MVP
cut line. Revisit (c) only if (a) hits documented org Sprite
count/running limits or cost envelopes — read `OrgInfo` limits via the API
before any fleet use (P4 checklist item).

## Conditions

- Slug→resource naming flows through the `apps` package (normalize, reserve,
  collide deterministically); URLs are stored verbatim from API responses.
- Private URLs by default; public publish requires explicit user action plus
  application-level authorization (URL secrecy is not authentication).
- P5 rehearsal measures natural sleep→wake latency and records runbook notes.
