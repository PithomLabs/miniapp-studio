# ADR-011: GitHub App permissions and credential brokerage

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** One GitHub App with minimal granted permissions; the controller mints per-run installation tokens narrowed
to exact repositories + minimal permissions (V4); Sprites GitHub connector evaluated per-use for in-Sprite API calls
with path allow-lists; git push inside Sprites uses the short-lived token treated as agent-readable.
**Basis:** prompt28.2 §6.3; verified token + connector mechanics (V4, V11).

## Context

Clone/push/PR operations need a token inside the Sprite (agent-readable by construction); API operations can avoid
that via connectors. The design minimizes what each token can do and how long it lives.

## Requirements

R225 (clone-and-PR minimum: `contents:write` + `pull_requests:write` on one repo), R222 (read-only diagnostics without
RW org creds), R211 (least-privilege reporting).

## Considered alternatives

- **Personal access tokens:** rejected — broad, long-lived, owner-bound; violates every least-privilege requirement.
- **Connectors for everything:** rejected as exclusive — git-over-HTTPS needs a token in the Sprite regardless;
  connectors cover API calls, tokens cover git.
- **Single long-lived installation token reused across runs:** rejected — 1h narrowing-per-run is the designed control.

## Security / operational / cost implications

App private key lives in the controller's secret store (owner-provisioned). Token minting is logged per run with
scope. Connector path allow-lists (API-managed) restrict in-Sprite GitHub API reach.

## Failure modes

Over-broad App installation grants (mitigation: installation audit at setup + narrowed minting regardless); token
exfiltration from Sprite (mitigation: 1h expiry + single-repo scope bounds damage; revocation via App, not token,
on incident); connector policy drift (mitigation: policy-as-code review + playground verification).

## Verification

LIVE: token-scope audit per run (repositories + permissions + expiry asserted); connector negative tests (blocked
paths refused); installation-permission inventory.

## Deferred

Fine-grained App permission tuning after first live runs; connector-vs-token split per operation (recorded in T4 plan).

## Developer-simplicity score

Neutral-positive: developers authorize once (App install); per-run mechanics are invisible. No PAT creation, rotation,
or scoping rituals for developers.
