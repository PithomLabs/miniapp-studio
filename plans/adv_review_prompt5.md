# Adversarial Security Review — Miniapp Studio P4 Foundation

- Scope: `store/` + embedded migrations, `server/`, `cmd/studio` (P4 foundation only).
- Read-only review; no repository file created, modified, or deleted except this report.
- Reviewer posture: principal application-security architect / PostgreSQL specialist / senior Go reviewer.
- Date: 2026-11-24. Studio root: `/home/chaschel/Documents/go/miniapp-studio`.
- Databases used: disposable PostgreSQL 16.15 clusters in `/tmp` (fresh `initdb`
  per run, trust auth, destroyed after). No owner or production resources touched.

## A. Executive verdict

**APPROVE P4 FOUNDATION** — with four MAJOR findings (F1–F4) that must be tracked
with owners, corrections, and regression tests, plus nine MINOR findings and four
improvements.

The headline attack from the brief — **direct `session.create` invocation as the
application role** — *succeeds mechanically* (probed: `evil-id` minted and
resolvable as `tA/opA` without any credential check), and I classify it as the
closest call in this review. I nevertheless judge it MAJOR rather than BLOCKER,
for one evidence-backed reason: **it crosses no privilege boundary the credential
does not already cross.** Application-role SQL access already implies arbitrary
tenant context (`SET app.tenant_id` to anything), full DML on every tenant row,
direct `session_lookup`/`session_revoke` for any known hash, and (once F2 is
fixed) framework-table DML. A minted session grants no new *data* access; its
only incremental power would be server-mediated external actions (Fly/Sprites),
which do not exist yet in this foundation. The operator-token boundary — the one
that protects against network attackers without server secrets — is intact:
unmintable tokens, unguessable sessions, constant-time verification, no fixation.
Closing F1 properly needs an owner-level design decision (proof-of-verification
bound to issuance, e.g. an HMAC over tenant/binding/hash/expiry with a
lookup-role-held key, or DB-held credential verification), and that correction
can land at any time without breaking tools or screens. Blocking the foundation
on it would stall P4 on theater: *any* issuance path executable by the app role
is inherently re-callable, so no in-model tweak short of that design changes the
outcome.

Everything else structural holds under live fire: role separation is exact (zero
memberships, correct attributes, PUBLIC revoked everywhere), RLS + FORCE on all
six tables with correct fail-closed semantics (cross-tenant reads/updates/deletes
see/affect nothing; cross-tenant inserts 42501), the three session functions are
fixed-SQL, fully qualified, `search_path`-pinned and immune to temp-table and
`search_path` shadowing, secrets never appear in responses/logs/records, and
startup fails closed before the socket opens. The remaining MAJORs are a dead
framework-grant path (missing `USAGE ON SCHEMA app`, fail-closed), cross-tenant
foreign-key references with a cascade side effect, and deactivation not ending
live sessions — all fix-forward, none a bypass of authentication or tenant data
isolation.

## B. Security boundary diagram

```
Browser ──(1)──> HTTP server ──(2)──> session middleware ──(3)──> handlers/tools
  │                    │                     │                         │
  │ cookie:            │ Secure/HttpOnly/    │ cookie → SHA-256 hash → │
  │ studio_session     │ SameSite=Lax        │ session_lookup() →      │
  │ (opaque 256-bit)   │ POST-only logout    │ server-derived tenant   │
  │                    │ safeNext redirects  │ (never browser input)   │
  │                    │                     │                         │
  │              app pool (studio_app, max 8 conns)                   │
  │                    │                     │                    (4) tenant tx
  │        ┌───────────┴───────────┐         │                    SET LOCAL
  │        │ SECURITY DEFINER fns  │         │                    app.tenant_id
  │        │ owner: studio_lookup  │         │                         │
  │        │ (NOLOGIN,NOBYPASSRLS) │         │                         ▼
  │        │ lookup/create/revoke  │         │              ┌─────────────────────┐
  │        │ PUBLIC revoked; app   │         │              │ studio.apps/specs/  │
  │        │ executes 3, not the   │         │              │ deployments/runs    │
  │        │ 4th (operator_upsert  │         │              │ RLS+FORCE, tenant = │
  │        │ is owner-only)        │         │              │ current_setting(..) │
  │        └───────────┬───────────┘         │              └─────────────────────┘
  │                    │ narrow table privs  │
  │                    ▼                     │
  │        ┌─────────────────────┐           │        ┌──────────────────────────┐
  │        │ studio.sessions /   │           │        │ app.* framework tables   │
  │        │ studio.operators    │           │        │ DML granted BUT unusable:│
  │        │ app role: NO access │           │        │ missing USAGE ON SCHEMA  │
  │        │ (42501 verified)    │           │        │ app (F2, fail-closed)    │
  │        └─────────────────────┘           │        └──────────────────────────┘
  │
Owner plane (separate DSN, separate commands, never in request path):
  studio_owner ── migrate (DDL) ──> studio.* │ provision (EnsureAppRole, operator rows)
```

Trust assumptions made explicit: (i) the app DSN is a highly privileged
server-side secret — its compromise already equals full tenant-data compromise
(RLS binds buggy code, not DSN thieves); (ii) operator login tokens and the
operators file live outside the DB; (iii) `session_create` enforces issuance
*preconditions*, not issuance *authorization* (F1).

## C. Role and function scorecard

Catalog values below are live `psql`-equivalent output from the disposable
cluster (probe P1), not comment paraphrase.

| Object/role/function | Actual privileges/owner | Verified invariant | Attack result | Verdict |
| -------------------- | ----------------------- | ------------------ | ------------- | ------- |
| `studio_owner` (migration/provision) | superuser in tests; owns schema + 6 tables + `studio_schema_migrations` | Only identity running DDL/provision; never in request path | N/A (trusted) | PASS |
| `studio_app` (runtime) | LOGIN, NOBYPASSRLS, no super/createdb/createrole, 0 memberships; USAGE on `studio`, DML on 4 tenant tables only | Cannot read/write sessions/operators; cannot SET ROLE; cannot DDL/policy/function-owner change | 20/20 hostile SQL attempts denied (42501/owner errors) | PASS |
| `studio_lookup` (function owner) | NOLOGIN, NOBYPASSRLS, 0 memberships; owns exactly 4 functions, 0 tables | Minimal privilege: `ar`+column-UPDATE on sessions, `arwd` on operators; USAGE only on schema | Cannot log in; cannot be assumed (A9) | PASS |
| `session_lookup(hash)` | owner lookup, DEFINER, STABLE, `search_path=studio`, exec: app+owner only | Fixed SQL, qualified, minimal output (tenant/binding/expiry), revoked/expired never returned | Shadowing, injection hash, forged/empty/oversized tokens all denied | PASS |
| `session_create(...)` | owner lookup, DEFINER, VOLATILE, `search_path=studio`, exec: app+owner only | Guards enforced: provisioned+active operator, `now < exp ≤ now+30d`; NULL on refusal | **Direct mint as app role succeeds (A1/A2)** — preconditions only, no issuance authorization | **MAJOR (F1)** |
| `session_revoke(hash)` | owner lookup, DEFINER, VOLATILE, `search_path=studio`, exec: app+owner only | Idempotent NULL; effective on next request (no caching) | Unscoped by tenant — revokes any known hash (needs 256-bit hash) | PASS with note |
| `operator_upsert(...)` | owner lookup, DEFINER, exec: migrate-time `session_user` only | App role denied (42501); idempotent upsert incl. deactivate/reactivate | A7 denied; P4 lifecycle verified | PASS |
| Tenant tables (`apps`, `app_specs`, `deployments`, `runs`) | owner studio_owner; app DML; RLS+FORCE; single USING+WITH CHECK tenant policy each | No-ctx/garbage/empty ctx sees nothing; cross read/upd/del affect 0 rows; cross insert 42501; no pool leak via helper | Full 10-case RLS matrix green; FK reference hole found (F3) | PASS with **MAJOR (F3)** |
| Framework `app.*` grants | DML granted, RLS+FORCE present (except global-by-design `kill_switches`) | Intended runtime path | **42501: missing USAGE ON SCHEMA app (F2)** — grants ineffective, fail-closed | **MAJOR (F2)** |

## D. Findings register

### F1 — `session_create` is directly invocable by the app role: issuance authorization assumed, not enforced (MAJOR, non-blocking)

- ID/severity: F1, MAJOR. Closest call in the review; judged non-blocking for
  the reason in §A (no boundary crossed beyond the credential's existing power;
  durable fix needs an owner design decision and can land without breaking screens).
- Location: `store/migrations/000002_schema.up.sql:157-175` (function body and
  `GRANT EXECUTE ... TO studio_app`); caller `store/sessions.go:89-111`
  (`Create`, whose doc says "server-authenticated pair" — an assertion the DB
  cannot check); login path `server/login.go:74`.
- Reproduction (live, disposable PG 16.15): as `studio_app` with no credential:
  `SELECT studio.session_create('evil-id','ev11…','tA','opA', now()+1h)` →
  `evil-id`; `SELECT * FROM studio.session_lookup('ev11…')` → `tA/opA` with live
  expiry. That hash, set as `studio_session` cookie, authenticates at HTTP as
  tenant `tA`. Prerequisites: app-role SQL access + knowledge of one
  provisioned `(tenant, binding)` pair (low-entropy, operator-chosen strings).
  Guards that DO hold: unprovisioned → NULL, deactivated → NULL, past/>30d
  expiry → NULL (A3–A5).
- Root cause: the credential lives outside the database (operators file), so the
  function cannot verify it; EXECUTE-by-app-role is simultaneously the
  legitimate server path and the attack path, indistinguishable at the DB boundary.
- Impact: any holder of the app DSN (or a future SQL-injection flaw executing
  as `studio_app`) can mint HTTP-usable sessions for any provisioned operator
  without the operator token. No *new data* access is granted (the same DSN
  already reads all tenants via arbitrary `SET app.tenant_id` + direct lookup);
  incremental power is limited to server-mediated side effects, which do not
  exist yet in this foundation — the reason to fix before external-action tools.
- Durable correction (owner decision required — pick one): (a) proof-of-verification:
  login handler passes `proof = HMAC(server_key, tenant‖binding‖token_hash‖expiry)`;
  function recomputes with a `server_key` stored in a lookup-owned table the app
  role cannot read; NULL unless valid. Key rotation via owner-only function.
  (b) Move verification into the DB: store login-token hashes in `operators`
  (owner-written), `session_create` takes the presented token and compares
  inside the DEFINER boundary. (a) keeps hashes out of the DB; (b) is simpler
  but widens DB-held secrets. Either preserves the three-role model. Do NOT
  "fix" by revoking app EXECUTE with no replacement path — the server needs one.
- Regression test: `TestDirectSessionCreateRequiresProof` — raw
  `session_create` as app role with a self-chosen hash and no/invalid proof
  returns NULL (fails today: returns an id), while `Store.Create` post-login
  still succeeds. Plus a doc test locking the trust assumption until fixed.
- Blocks P4 continuation: no, but must be fixed (or formally risk-accepted)
  before tools that trigger external/irreversible effects ship.

### F2 — Framework table grants are ineffective: no `USAGE ON SCHEMA app` (MAJOR, non-blocking)

- ID/severity: F2, MAJOR. Fail-closed (denial, not leak), but the wired
  runtime path is dead and the framework-RLS interplay is unverifiable until fixed.
- Location: `store/migrations/000002_schema.up.sql:209-212` (table GRANTs with
  no companion schema grant — the same bug class 000003 fixed for `studio`);
  wiring `server/server.go:96-113` (DBOS executor, classa, effects, runtime
  Service all constructed over the app pool); `store/migrate.go` (no USAGE fix).
- Reproduction (live): as `studio_app`, `SELECT count(*) FROM app.invocations`
  (and approvals/audit_events/effect_records/kill_switches) → `42501 permission
  denied for schema app`. Consequently `runtime.Service` invocations, DBOS
  workflows, and effects/audit writes through the app pool all fail today;
  `server.New` succeeds only because construction issues no queries, and no
  committed test invokes the Service.
- Root cause: table privileges without schema USAGE; migration review caught it
  for `studio` (000003) but the `app` grants in the same file were missed.
- Impact: any P4 tool calling the Service fails closed with 42501. No data
  crosses tenants (denial). Masks the real RLS-interplay question (§9): once
  USAGE exists, framework RLS (same `app.tenant_id` setting, `USING`+`WITH CHECK`
  on invocations/approvals/effect_records, USING-only on audit_events) must be
  proven through executor-owned connections.
- Durable correction: fix-forward migration `000005_app_usage`:
  `GRANT USAGE ON SCHEMA app TO studio_app;` (lookup role untouched — its
  functions never reference `app.*`). Then prove: tool-shaped `InTenantTx`
  performing a Service invocation succeeds with tenant armed and fails without.
  Do not work around with `SET ROLE`, additional grants on `app.*` internals,
  or a second privileged pool.
- Regression test: `TestFrameworkTablesReachableAsApp` — bare `SELECT` on each
  granted table returns 0 rows (not 42501); `TestServiceInvocationTenantScoped`
  — invocation inside `InTenantTx(tA)` writes tenant-`tA` rows, bare invocation
  fails closed. Both fail today (42501), pass after.
- Blocks P4 continuation: no (security-fail-closed), but blocks *functional*
  tools work: fix before the first Service-invoking tool.

### F3 — Cross-tenant foreign-key references accepted; cascade deletes cross tenants (MAJOR, non-blocking)

- ID/severity: F3, MAJOR. Genuine isolation-weakness shape, contained (no reads
  or writes of another tenant's rows), cheap fix-forward.
- Location: `store/migrations/000002_schema.up.sql:54,65,83`
  (`REFERENCES studio.apps(id)` on the global PK, no tenant match);
  contrast tenant-scoped uniques on `:47,59,76`.
- Reproduction (live): in `tA` context,
  `INSERT INTO studio.app_specs(id,tenant_id,app_id,version,…)
  VALUES('spec-x','tA','app-b',…)` where `app-b` belongs to `tB` → success.
  Then deleting `app-b` as owner cascades and deletes `tA`'s `spec-x` (verified
  count 0 after). Same shape exists for `deployments` and `runs`.
- Root cause: referential integrity is global while authorization is per-tenant;
  nothing requires `child.tenant_id = parent.tenant_id`.
- Impact: requires knowing/guessing another tenant's app id; yields no data
  access, but pollutes integrity (dangling cross-tenant links invisible to the
  victim) and gives one tenant's legitimate delete destructive power over
  another tenant's rows.
- Durable correction: fix-forward migration adding
  `UNIQUE (tenant_id, id)` on `studio.apps` and replacing the three FKs with
  composite `FOREIGN KEY (tenant_id, app_id) REFERENCES studio.apps(tenant_id, id)
  ON DELETE CASCADE`, backfilled after a data check (fail the migration if a
  cross-tenant reference already exists — do not silently reassign). Keep RLS
  as the access layer; the FK becomes the integrity layer.
- Regression test: `TestCrossTenantReferenceRejected` — the R8a insert fails
  (FK violation), same-tenant insert succeeds; `TestCascadeStaysInTenant` —
  deleting `tB`'s app leaves `tA` rows intact. Fail today, pass after.
- Blocks P4 continuation: no.

### F4 — Deactivating an operator does not end live sessions; no revoke-by-operator path (MAJOR, non-blocking)

- ID/severity: F4, MAJOR. Offboarding gap bounded by the 12h TTL.
- Location: `store/migrations/000002_schema.up.sql:142-152` (`session_lookup`
  never consults `operators.active`); `store/provision.go:57-69`
  (`UpsertOperator` flips `active` only); no revoke-by-(tenant,binding) function
  exists; `store/sessions.go:137-150` (revoke is per-hash only, and hashes are
  not enumerable by design).
- Reproduction (live): create session for `tA/opA` → lookup 1 row → set
  `active=false` → lookup still 1 row (usable cookie until expiry or per-hash
  revoke). Reactivation semantics untested by the suite beyond new-session
  refusal (`store/tenant_test.go:95-112` covers create-refusal only).
- Root cause: `active` gates issuance, not validity; offboarding has no session
  handle to revoke (hashes known only to cookie holders).
- Impact: a deactivated operator's stolen or retained cookies stay valid up to
  `SessionTTL` (12h). No amplification beyond the operator's own tenant.
- Durable correction: add `studio.session_revoke_operator(p_tenant_id,
  p_binding_id)` (DEFINER, lookup-owned, EXECUTE to owner only — or to app role
  if a future admin tool needs it, with tenant fixed server-side, never from
  the caller) setting `revoked_at` for all live rows of the pair; call it from
  `provision --deactivate` (new flag) in the same owner flow as
  `UpsertOperator(..., false)`. Optionally also join `operators.active` in
  `session_lookup` as belt-and-braces — but that alone resurrects sessions on
  reactivation, so the explicit revoke is the durable part.
- Regression test: `TestDeactivateEndsSessions` — create → deactivate via new
  path → lookup is `ErrNoSession`; new create refused; reactivate → new create
  works, old token still dead. Fails today at the first lookup, passes after.
- Blocks P4 continuation: no, but must close before multi-operator production use.

### F5 — Valid-token/deactivated-operator login returns 500, bad token 401: credential oracle (MINOR)

- Location: `server/login.go:74-78` (any `Create` error → 500) vs `:67-72`
  (no match → 401 + identical page).
- Reproduction (live): `POST /login` with the *correct* token while the
  operator row is deactivated → 500, no cookie; wrong token → 401. The status
  delta tells an online attacker a guessed token was valid. Server-side log
  detail is a static string (no secret) — verified in probe output.
- Root cause: issuance-refusal (an authentication outcome) mapped to a server-error path.
- Durable correction: in `login`, map `Create` refusal ("session refused") to
  the same 401 + "Invalid credentials." page as a bad token; reserve 500 for
  transport/DB errors. Keep the server-side log line (it aids ops; contains no secret).
- Regression test: `TestLoginDeactivatedIs401` — deactivate, valid token → 401
  with no cookie and the same body as a bad token. Fails today (500), passes after.
- Blocks: no.

### F6 — `safeNext` permits control characters in the redirect target (MINOR)

- Location: `server/login.go:42-47`, used at `:84-88`; deny path
  `server/middleware.go:56-61` shares the prefix-only style.
- Reproduction (live): `POST /login` with `next=/%0d%0aSet-Cookie:%20x=1` →
  `302 Location: "/\r\nSet-Cookie: x=1"` (raw, via ResponseRecorder). Open
  redirects themselves are correctly neutralized (`https:`, `//`, `javascript:`,
  double-encoded `//` all → `/`, probed).
- Root cause: validation checks prefix only, never the character set.
- Impact: contained — verified over a real socket (`httptest.NewServer`) that
  Go emits no split response (client observed a single mangled `Location`,
  zero cookies). Still, the validator should not rely on transport sanitizing.
- Durable correction: reject (→ `/`) any `next` containing ASCII controls,
  DEL, or — after one decode — anything outside a conservative set; share one
  helper between login and deny. Add table test incl. CRLF, tab, `%00`.
- Regression test: `TestLoginNextRejectsControls` — CRLF/tab-bearing `next`
  yields `Location: /`. Fails today, passes after.
- Blocks: no.

### F7 — Pool preserves session-level settings across checkouts; no reset guard (MINOR)

- Location: `server/server.go:67-75` (pool config: no `BeforeAcquire` /
  `AfterRelease` reset); helper `store/tenant.go:15-34` (correctly uses
  `SET LOCAL` — the live paths are safe).
- Reproduction (live): `SELECT set_config('app.tenant_id','tA',false)` on one
  acquired app-pool connection, release without reset → six subsequent bare
  `SELECT count(*) FROM studio.apps` all returned `tA`'s rows (2,2,2,2,2,2).
  Control: `InTenantTx` (LOCAL) leaves nothing (bare count 0 after, R7a).
- Root cause: pgxpool does not reset session state by default; safety rests
  entirely on every future caller remembering LOCAL-only.
- Impact: no live misuse — all foundation code uses the helper. Pure footgun
  for tools-step code (or the framework executor, if it ever session-SETs).
- Durable correction: set `AfterRelease: RESET app.tenant_id` (narrow, cheap)
  on both the server pool and any test/store pools; document LOCAL-only as an
  inviolable rule in `store/tenant.go`. Do not `DISCARD ALL` (needlessly heavy).
- Regression test: `TestPoolResetsTenantSetting` — session-SET then release,
  then bare reads see 0. Fails today, passes after.
- Blocks: no.

### F8 — Sprites token file read without trimming; minted files end with `\n` (MINOR)

- Location: `server/server.go:127-140` (`token := string(raw)`, no trim) vs
  `cmd/studio/main.go:190` (`token+"\n"` written 0600). Adapters pass the token
  verbatim into `Bearer ` headers (`fly/connectors.go:69`,
  `fly/sprites.go:24-28` — no trim there either).
- Reproduction: code-evidence (composition of the two verified lines): a
  token minted by this repo's own command reaches the adapter as `"tok\n"`,
  and Go HTTP clients reject newline-bearing header values, so Fly/Sprites
  calls fail when the optional Fly identity is configured. (Untriggered in
  tests — no test configures `SpritesTokenFile`.)
- Root cause: inconsistent file-secret convention (`config.go envOrFile` and
  `readSecretFileE` trim; this read does not).
- Durable correction: `strings.TrimSpace` at `server.go:133` (one line);
  add `TestNewTrimsSpritesToken` with a fake `fly.SpriteManager`/`ConnectorManager`
  capture or a token-file fixture asserting the adapter receives the trimmed
  value. Alternatively trim inside the two constructors — server-side trim is
  the smaller, single-point fix.
- Blocks: no (optional path, no handlers use it yet).

### F9 — `whoami.session` is always empty; `Lookup` never returns the session id (MINOR)

- Location: `store/sessions.go:71-83` (scans tenant/binding/expiry only;
  `Session.ID` set solely by `Create`), `server/routes.go:39-44`.
- Reproduction (live): authenticated `GET /api/whoami` →
  `{"binding":"wop","session":"","tenant":"tW"}`.
- Root cause: `session_lookup` does not select `s.id`, so the middleware's
  `ctxSession` is always `""`.
- Impact: cosmetic today; a future audit trail keyed on `session` would log
  empties. (Tenant/binding — the authorization inputs — are correct.)
- Durable correction: either select `s.id` in `session_lookup` and scan it
  (one-line SQL + scan change; id is a random 128-bit value, safe to return),
  or drop `session` from whoami/context. Keep it consistent, not both.
- Regression test: extend `TestLoginFlow` — whoami `session` non-empty (or
  field absent, per the chosen fix). Fails today under the populate option.
- Blocks: no.

### F10 — Duplicate login tokens across tenants accepted; last match wins silently (MINOR)

- Location: `server/config.go:146-177` (validates per-entry shape + tenant
  uniqueness, not token-hash uniqueness), `server/login.go:60-66` (loop
  overwrites `matched`).
- Reproduction: code-evidence (six-line loop, unambiguous): two operators
  sharing one `login_token_sha256` both validate, and login binds the later
  entry's tenant with no warning.
- Root cause: uniqueness enforced on the wrong column for authentication.
- Impact: low — requires operator misconfiguration with a shared high-entropy
  token; effect deterministic (last wins), no cross-tenant data access beyond
  the operator's own legitimate tenants.
- Durable correction: reject duplicate `login_token_sha256` in `loadOperators`
  (same style as the tenant check). One-line-adjacent fix.
- Regression test: `TestLoadRejectsDuplicateTokenHash`. Fails today, passes after.
- Blocks: no.

### F11 — Dead code and dangling doc pointers (MINOR)

- `server/errors.go:33-36` `notFound` is never called (verified by search).
  Delete it or wire it as the mux 404 — today unknown routes get Go's default
  body (probed: `404` both authed and anonymous, no leak either way).
- `store/migrations/000001_roles.up.sql:4` points to a "README deploy section"
  that does not exist (README has no role-creation docs; the real path is
  `studio provision`). Move the pointer to the provision command's usage.
- `store/sessions.go:113-115` `CreateWithToken` is test-only by convention
  (verified: only `store_test.go` callers). Keep, but the tools contract (§F)
  must forbid production use — caller-chosen tokens weaken the issuance story.
- Blocks: no. No regression tests needed beyond continued `go vet` cleanliness.

### F12 — Readiness does not verify migrations; startup does not check schema version (MINOR)

- Location: `server/routes.go:28-37` (ping-only `/readyz`),
  `server/server.go:62-143` (no version gate), `cmd/studio/main.go:39-56`
  (serve path never calls `store.Migrate` — correct separation, but nothing
  checks the outcome of the separate step).
- Reproduction: by construction + live behavior — a server started against a
  migrated-DB-minus-Studio-tables answers `{"status":"ready"}` while `/login`
  500s. Fail-closed (no auth bypass: 500, no cookie), but orchestrators get a
  false ready.
- Durable correction: at startup (after pool ping) and in `/readyz`, compare
  `studio_schema_migrations.version` to the embedded latest (fail/503 on
  mismatch or dirty); keep actual migration in the owner-only `migrate`
  command. Assert the 503 path in `TestReadyzUnmigrated`.
- Blocks: no.

### F13 — Test-breadth gaps on the security boundary (MINOR)

- `cmd/studio` has zero test files: `provision` (role creation, operator row,
  0600 operators-file upsert incl. rotation-overwrite) and `mint-sprite-token`
  are unpinned. The first-time ordering trap (provision → migrate → provision,
  since `Migrate` requires the role and `UpsertOperator` requires the schema)
  is documented only in an error string (`store/provision.go` /
  `cmd/studio/main.go:143`).
- No committed test pins: the A-series denials beyond the two current 42501
  assertions (SET ROLE, DDL, policy/function-owner changes, `operator_upsert`
  is covered); temp/`search_path` immunity; down/up cycle (P7 verified
  manually); multi-error determinism of secret-bearing inputs; P6-style
  no-secret-in-error assertions for migrate/connect paths.
- The suite's "disposable restricted roles" claim is really "the real app role
  with a per-run random password" — adequate, but no *second* low-privilege
  persona probes least-privilege drift.
- Durable correction: add the named tests from F1–F12 plus
  `TestProvisionFlow` (role+row+file round-trip on a scratch DB) and
  `TestMigrateErrorsRedactSecrets`. Do not gate tools on this list.
- Blocks: no.

## Improvements (optional, non-blocking)

- I1 — Rate-limit `POST /login` (and count failed attempts). Tokens are
  high-entropy bearer secrets, so online guessing is infeasible; this is
  hygiene + noisy-attacker signal, not a vulnerability fix.
- I2 — `__Host-` cookie prefix for `studio_session` (requires Secure + Path=/
  + no Domain — all already true) to pin the cookie to the host.
- I3 — Atomic operators-file write (`write tmp + rename`) in
  `upsertOperatorsFile`; a crash mid-`WriteFile` today truncates the registry.
- I4 — Deployment guidance: require `sslmode` (TLS) on both DSNs at the
  platform layer; passwords cross the wire otherwise. `EnsureAppRole`'s
  `ALTER ROLE ... PASSWORD` likewise needs a TLS-or-local connection.
- I5 — Document the F1 trust assumption (`store/doc.go` + runbook): app DSN =
  tenant-data-equivalent secret; list exactly what it can and cannot do.

## E. Test evidence

Toolchain: `go1.27.3 linux/amd64`, `GOTOOLCHAIN=local`. PostgreSQL: disposable
16.15 (Ubuntu build, user-space extract, fresh `initdb`, trust auth,
`127.0.0.1`, destroyed after each run). Prior reported evidence (13+13 tests,
PG-backed, `-p 1`) was re-executed independently below — nothing is taken on trust.

| Command / probe | Result |
| --------------- | ------ |
| `gofmt -l .` | clean, exit 0 |
| `go vet ./...` | pass, exit 0 |
| `go build ./...` | pass, exit 0 |
| `go test -p 1 ./...` (no DB) | pass; DB tests skip, config/unit green |
| `go test -p 1 -count=1 -v ./store/ ./server/` on disposable PG 16.15 | **26/26 pass, 0 skip, 0 fail** (13 store: 8 session/migrate/role + 5 tenant/provision; 13 server: 8 HTTP + 5 config) |
| `go test -race -p 1 -count=1 ./store/ ./server/` on disposable PG 16.15 | pass, exit 0, both packages `ok` |
| `go list` pin check | `miniapp v0.0.0-20261123181255-73351a801fc5`; repo HEAD `73351a8`; zero tracked modifications (`git status`: only `??` entries) |
| Probe P1 catalog (roles/members/owners/RLS/policies/funcs/ACLs/default-privs) | all match design; PUBLIC exec false ×4; 0 memberships; lookup owns 4 funcs/0 tables |
| Probe P2 direct-create attack (A1–A14) | mint+lookup succeed (F1); all guards refuse correctly; 20/20 privilege-escalation attempts denied; shadowing defeated |
| Probe P3 RLS matrix (R1–R10 + R6c) | textbook isolation; FK hole (F3) + pool-reset footgun (F7) found; framework-schema denial found (F2) |
| Probe P4 deactivation | live session survives deactivate (F4) |
| Probe P5 HTTP (H1–H13) | fixation/double-login/logout/forge/oversize/expiry/unknown-route all correct; 500-vs-401 oracle (F5); CRLF contained (F6) |
| Probe P6 secrecy | migrate/connect errors redact password (`user=u database=db`); static refusal strings |
| Probe P7 down/up cycle | clean: schema removed, re-applied v4 `dirty=false`, RLS restored |
| CRLF over real socket | single mangled `Location`, zero injected headers — transport fail-safe, validator still at fault (F6) |
| PG 19 primary + PG 17 MPG spot-check | **Blocked by environment**: docker socket denied from sandbox; no PG17/19 binaries on disk (only client tools + user-installed 16.15 server). Mitigation: migration SQL uses nothing version-sensitive (RLS/FORCE predate 9.5; `current_setting(missing_ok)`, `ON CONFLICT`, DEFINER/`SET search_path` all long-stable) — compat argued by inspection, not executed. Not counted as green. |

## F. Remaining P4 implementation contract

Tools, registry entries, and the six screens must preserve:

1. **Validate-or-deny**: every session path treats `Lookup` error (including
   transport errors) as unauthenticated; handlers check `TenantOf`'s `ok`.
2. **Server-derived tenancy only**: tenant comes from the session row, never
   from cookies/params/headers/bodies; `InTenantTx` (LOCAL) is the only way to
   arm `app.tenant_id` — never session-level SET (F7).
3. **Fix F2 first**: no Service-invoking tool works until `USAGE ON SCHEMA app`
   lands; then prove framework-RLS interplay through executor connections
   (executor-owned connections must carry tenant context or fail closed).
4. **Read-only boundary + G-01**: no tool bypasses the runner's admitted-spec
   path; no external writes; `unconfiguredBackend` stays until the P5 gateway.
5. **CSRF strategy for new mutating endpoints**: SameSite=Lax + POST-only
   sufficed for login/logout; new POST/PUT/DELETE routes need origin checks or
   anti-forgery tokens — do not assume cookies alone suffice.
6. **No browser-controlled identifiers into privileged calls**: fixed tenant
   from context; `CreateWithToken` stays test-only; future admin functions
   (e.g. F4's revoke-by-operator) take tenant from the session.
7. **Run-bound identity**: bind tool runs to the content hash + session tenant
   at admission; re-validate at execution; audit security-relevant actions.
8. **F1/F4 closure before external effects**: proof-of-issuance (F1) and
   deactivate-revokes (F4) must land before tools that deploy, spend, or
   irreversibly act.

## G. Final decision

**Proceed: the P4 foundation is approved and Studio tools + registry work should
continue immediately.** The three-role model, RLS posture, function hardening,
session lifecycle, cookie handling, configuration fail-closure, and startup
wiring withstood direct adversarial execution; the four MAJORs are tracked,
fix-forward, and precisely scoped with regression tests, and none permits
unauthenticated access, cross-tenant data access, RLS bypass, or secret
disclosure. Per the brief, no further review of this foundation is requested —
the next adversarial review is the integrated P4 review after tools and all six
screens exist.
