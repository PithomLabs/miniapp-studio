 P4 foundation is implemented and green. Status:

  Migrations + three-role boundary (store/). Four embedded migrations (roles, schema + RLS + session functions, schema USAGE fix-forward, RLS
  enablement fix-forward): six tenant tables with forced RLS and tenant-scoped uniques, sessions/operators denied to the app role, three
  SECURITY DEFINER functions (lookup/create/revoke) owned by the non-login lookup role with PUBLIC revoked, operator provisioning constrained to
  pre-provisioned rows with bounded expiries. 13 store tests prove the boundary: 42501 on direct reads, empty results for forged/expired/revoked
  tokens, role attributes from catalogs (no bypass, no ownership, no membership path, forced RLS), tenant isolation with and without context,
  rollback, idempotent provisioning.

  Server skeleton (server/, cmd/studio). Fail-closed config (all-problems reporting, *_FILE secrets, both-sources refusal, Fly pair rule), wiring
  mirroring the framework's production shape (app pool with MPG-budgeted settings, policy, registry + runner, executors, service), session
  middleware (cookie → hash → lookup → tenant context; validated login redirects; 401 for non-GET), login/logout with constant-time credential
  scan and secure cookie flags, health/readiness, structured errors that never leak internals. 13 server tests cover config rejection, the full
  login flow, revocation, forgery, and redirect validation.

  Honest findings fixed at the root. Single-role RLS bypass (tests assert through a disposable restricted role); FORCE without ENABLE
  (fix-forward migration); NULL-vs-no-row function semantics; parallel-package DB races (serial -p 1 enforced, matching Miniapp CI).

  Next: Studio tools + registry via Invoke, then the six screens and acceptance suite. Uncommitted; miniapp untouched.

◆ Worked for 19m 55s · 10:48 PM · 24/Nov

◆ Recap: Building the Miniapp Studio control plane; foundation is green with migrations, session boundary, and server skeleton tested. Next:
         register Studio tools through runtime.Invoke.
