# ADR-003: Session issuance proof — HMAC vs DB-held verification

**Status:** Proposed 2026-11-24. **Decision:** DEFERRED to the owner — this
record compares only; neither option is implemented and no selection is made.
**Basis:** P4 prompt6 finding F1; current P4 behavior is opaque tokens +
`studio.sessions` lookup (`store/sessions.go`, migrations 000001–000002).

## Context

Every authenticated request must prove its session cookie was issued by this
server for a provisioned operator — not forged, and (for the DB option) not
revoked or expired. Two proof mechanisms are on the table:

- **(a) HMAC-signed stateless tokens.** The cookie carries
  `session_id || expiry || HMAC(server_secret, session_id || expiry)`.
  Verification recomputes the MAC: no database read on the auth path.
- **(b) DB-held credential verification.** The cookie carries an opaque
  random token; verification hashes it and resolves it through the
  `SECURITY DEFINER studio.session_lookup` function, which enforces
  expiry and revocation inside the database. This is what P4 implements.

The question is whether (a) should replace or complement (b).

## Option (a) — HMAC stateless proof

How it would work: at login, after the existing operator checks, the server
mints `id.expiry.mac` with a server-held secret (rotated out of band, old
secrets retained during overlap). Middleware verifies the MAC in constant
time and rejects expired ids without touching Postgres. Revocation would
need a separate denylist (DB table or short expiries), because a valid MAC
is otherwise unforgeable-but-irrevocable.

Strengths:

- Auth path needs no database round-trip: lower p99, immune to DB
  slowness/outages for already-issued sessions.
- Verification is a pure function of the token + secret: trivially testable,
  no RLS or role plumbing on the read path.
- Scales horizontally with nothing shared except the secret.

Weaknesses and costs:

- Revocation is no longer free. Instant operator offboarding (F4) needs a
  denylist lookup — reintroducing the DB read the option was meant to
  avoid — or short expiries plus refresh machinery, which is a second
  session protocol to design, test, and operate.
- The HMAC secret is a new crown jewel: compromise forges arbitrary
  sessions for any tenant until rotation completes, with no per-session
  record to invalidate. The DB option's blast radius is one row per token.
- Secret rotation needs overlap handling (verify against N secrets, mint
  with the newest) and a documented emergency-rotation runbook.
- Statelessness complicates the audit story: "which sessions exist for this
  operator" has no table to query; logout becomes client-side deletion
  unless a server-side record exists anyway.

## Option (b) — DB-held verification (current P4)

How it works today: `session_create` (SECURITY DEFINER) refuses
unprovisioned/deactivated operators at issuance; `session_lookup` resolves a
token hash to (id, tenant, binding, expiry) only for live, unrevoked rows;
`session_revoke` ends a session immediately. The app role can execute exactly
these three functions and read nothing else in `studio`.

Strengths:

- Revocation and deactivation are immediate and total: one row update ends
  every session for an operator, and the issuance guard stops new ones
  (proven by `TestLoginUnprovisionedOperator401`).
- The database is the single source of truth: sessions are enumerable,
  auditable, and expire by server clock (`now()`), not client claims.
- No long-lived signing secret: compromise of any one token costs one
  session; there is no key whose theft forges sessions.
- Already implemented, migrated (000001–000002, 000009), and covered by
  the P4 suites, including the F5 401 mapping and the F9 stable-id lookup.

Weaknesses and costs:

- Every authenticated request pays one indexed DB lookup (hash equality on
  `studio.sessions.token_hash`). Cost is small but nonzero, and auth
  inherits DB availability.
- Horizontal scale shares the sessions table (uncontroversial at MVP
  scale; connection pooling already bounds it).

## Comparison

| Concern | (a) HMAC stateless | (b) DB-held (current) |
|---|---|---|
| Verify cost | CPU MAC only | one indexed lookup |
| Instant revocation | needs denylist/refresh | free (row update) |
| Offboarding (F4) | denylist or short expiry | `active=false` + delete rows |
| Secret blast radius | all tenants until rotated | one session per token |
| Session inventory/audit | no table (unless added) | `studio.sessions` |
| Operates during DB outage | yes (until denylist needed) | no |
| Implementation state | unbuilt | built, migrated, tested |

## What the owner decision needs

1. Whether instant, total revocation (F4) outranks DB-independent auth for
   the MVP threat model. If yes, (b) already satisfies it and (a) must add
   a denylist, erasing its main advantage.
2. Whether a second session protocol (refresh/rotation runbook) fits the
   P5 budget. If no, (a) should stay unbuilt.
3. If (a) is ever adopted: secret storage/rotation design, overlap policy,
   denylist-or-expiry revocation semantics, and audit-inventory
   requirements — all decided before implementation.

Until the owner decides, (b) remains the only session mechanism, and no
HMAC code, secret plumbing, or dual-path verification may land.
