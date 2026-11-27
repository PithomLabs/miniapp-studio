# prompt2_plan — Restore Local Database Test Connectivity

## Goal

Establish a durable, secure, repeatable execution path through which the
coding agent runs PostgreSQL-backed integration tests from its normal
authorized environment against the existing host Docker PostgreSQL 19
(`127.0.0.1:5434`) — without the owner manually executing every test
and without granting the agent unrestricted host or Docker access.
Implements `plans/prompt2.md`; governed by PRINCIPLES.md.

## Success Criteria

- The agent shell runs `pg_isready` plus a real SQL query against host
  PG19 and records the exact version and endpoint.
- Migration `000025` applies; receipt-store, RLS/grant, and
  privilege-parity tests execute (not skip) and pass.
- The complete serial suite runs with exact pass/fail/skip counts and
  zero DB tests skipped for connectivity.
- Isolation is proven preserved: the Docker socket stays denied from
  the agent shell and non-PG egress stays blocked (demonstrated by
  negative checks, not assumed).
- Configuration, startup/shutdown, health checks, and recovery are
  documented as a standard environment artifact every future agent
  inherits.
- No reinstall/recreate/replace of PostgreSQL or Docker; no
  application database-architecture change; no production
  credentials or data in tests.

## Approach

Pursue the directive's three options in ranked order — (1) sandbox
network configuration, (2) dedicated integration-test container,
(3) host-side execution — and stop at the first viable one. A
Phase-1 investigation tests each option against actual harness and
host capabilities before anything is implemented, and an owner
decision gate selects the path. The established failure mechanism
(caller-side isolation: separate network namespace, socket policy —
not a broken database) means the fix is a least-privilege path, never
an environment rebuild. The existing `testdb` harness (least-privilege
`studio_app_test`, run-unique databases, `owner.dsn` convention)
already satisfies "dedicated credentials + isolated database", so no
new credential design is introduced.

## Key Decisions

- **D1 — Ranked pursuit with a gate.** Investigate options 1→2→3 in
  order; implement only the owner-authorized winner. Rationale: least
  new machinery first; a container second; manual execution third,
  recorded as temporary per PRINCIPLES §5 if chosen.
- **D2 — Per-command unsandboxed execution is workaround-class.**
  Tool-level one-shot escalation puts the owner back in the loop on
  every run with broad scope, so it cannot satisfy the acceptance
  criterion; usable only as an explicitly authorized bounded bridge.
- **D3 — The durable artifact is documentation plus bound config.**
  The standard-environment record (config, startup/shutdown, health,
  recovery) is the deliverable; runner assets exist only if option 2
  wins, and only the minimum (launch command, result handoff).
- **D4 — Verification proves isolation, not just connectivity.**
  Negative checks (socket still denied, non-PG egress still blocked)
  are acceptance gates alongside the green suite.

## Steps

### Phase 0 — Host baseline (owner shell; the agent cannot do this)

Record what only the host context can answer: container name/state
(`docker ps`), port actually served
(`pg_isready -h 127.0.0.1 -p 5434`), compose network name
(`docker inspect`, for option 2 feasibility), server version. The
agent records the verbatim results. Gates everything below.

### Phase 1 — Capability investigation (agent, read-only + owner answers)

- Confirm the session-network enablement procedure and its exact
  scope (allowlisted destination:port vs broader host access).
- Assess runner-container feasibility: compose-network attach,
  repo mount, result-handoff convention, lifecycle ownership.
- Assess escalation semantics only to document why D2 holds.
- Deliverable: ranked recommendation with evidence → **owner
  decision gate**. No implementation before authorization.

### Phase 2 — Implement the authorized path only

Bounded by option: config + docs for option 1; minimal runner
assets + launch procedure for option 2; exact-command runbook for
option 3. Stop and request a decision before any
security-boundary change exceeding the directive.

### Phase 3 — Verify and report

Run the Validation Plan, record exact commands with pass/fail/skip
counts and fake-vs-real distinctions, update the stored
environment rule with the established path, and report remaining
limitations.

## Validation Plan

From the agent shell once the path exists (never printing the DSN):

- `pg_isready -h <endpoint> -p 5434` → accepting connections.
- `SELECT version(); SHOW server_version;` → PG 19.x recorded.
- `TEST_DATABASE_URL=$(cat /tmp/boundary-b/owner.dsn) go test -p 1
  -count=1 ./store/ ./testdb/` → 000025 applied, receipt,
  RLS/grant, and parity tests executed and green.
- `go test -p 1 ./...` → full serial suite, exact counts, no
  connectivity skips.
- Negative gates: `docker ps` still permission-denied; a
  non-allowlisted egress attempt still blocked.
- A zero exit code with skipped DB tests is reported as incomplete,
  never as full verification.

## Risks / Open Questions

- **R1 — Enablement scope.** If the network path cannot be narrowed
  to the database endpoint:port, least privilege fails and the
  owner must accept the broader scope or fall back to option 2/3.
- **R2 — Lifecycle ownership.** Who starts/stops/refreshes a bridge
  or runner, and stale-resource hygiene, must be explicit in the
  environment doc.
- **R3 — Credential handling.** The DSN is read from the 0600 file
  at use time and never printed, logged, or persisted elsewhere.
- **Q1 (owner/harness):** exact network-enablement procedure and
  scope — unanswerable from inside the sandbox.
- **Q2 (owner shell):** compose network name and container state.
- **Q3 (owner):** which option is authorized after Phase 1.

## Non-goals

Reinstalling PostgreSQL or Docker; changing application database
architecture for a sandbox limitation; agent Docker-socket access;
blanket host networking; exposing PostgreSQL beyond loopback;
PG version changes; production credentials or data.
