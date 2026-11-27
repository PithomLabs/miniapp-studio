# ADR-018: Object storage and external-dependency isolation

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Previews and ephemeral runs get isolated external dependencies by default: per-PR storage namespaces
(preferring Neon's branch-aware object storage where it fits, uniquely named buckets/sandboxes otherwise); sandbox
credentials for email/payments/webhooks/integrations; production external side effects are impossible from preview
configurations, not merely discouraged.
**Basis:** prompt28.2 §5.5; prompt28.3 §12 note; verified branch-aware backend concept (N12).

## Context

Database isolation without storage/integration isolation still leaks across PRs (files) or into production (emails,
charges, customer records). Neon's branch-aware storage (N12) may collapse this category — but adoption needs its own
justification, not the PG mandate.

## Requirements

R219: per-PR storage isolation; sandbox providers; synthetic credentials; no prod communications/charges/mutations.

## Considered alternatives

- **Shared preview bucket with per-PR prefixes:** rejected as default — prefix discipline is convention, not
  isolation; one path bug contaminates across PRs. Allowed only where namespace-per-PR is technically unavailable,
  with compensating tests.
- **Neon branch-aware storage (preferred pending evaluation):** branch-scoped namespaces that fork and delete with
  the database branch — evaluated in T3 against fit (SDK, limits, cost); adopted only on evidence.
- **Production credentials in previews with "careful" code:** rejected absolutely — configuration must make prod
  side effects unreachable.

## Security / operational / cost implications

Preview config carries only sandbox/synthetic external credentials; a config audit (automated) asserts no production
endpoint, key, or bucket appears in any preview/app-test configuration. Storage lifecycle binds to branch lifecycle.

## Failure modes

Sandbox provider behaving unlike production (mitigation: contract tests against sandbox semantics; documented
``diffs); storage orphaned after branch delete (mitigation: reconciler covers storage namespaces); credential-type
confusion (prod key in sandbox field — mitigation: typed config + prefix/format validation per provider).

## Verification

LIVE: cross-PR storage contamination negative test; prod-endpoint unreachability from preview config (network +
config audits); sandbox side-effect capture test (email/payment attempts land in sandbox, never prod).

## Deferred

Neon storage adoption verdict (T3 evaluation); per-provider sandbox matrices.

## Developer-simplicity score

Positive: previews behave like production without endangering it. Developers test file flows and integrations against
real-shaped sandboxes with zero setup and zero risk of billing a customer or emailing a user.
