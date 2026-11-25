## Phase 0 complete — root cause proven live on PG19

**Environment:** PG19 Docker restored (`19.0 (Debian 19.0-1.pgdg13+1)`, same image digest as CI). All forensics ran against an isolated scratch DB + scratch role; `miniapp_p0` untouched (reads only); scratch DB, role, and credential file all removed afterward, verified gone. Container left running.

### Root cause: version-table shadowing via `search_path` — proven for both runners independently

The golang-migrate pgx driver resolves its version-table schema with `SELECT CURRENT_SCHEMA()` and qualifies all tracking SQL with it. CI role `studio` + default path `("$user", public)` + Studio `000002`'s `CREATE SCHEMA studio` flips resolution mid-suite. Demonstrated live with the exact helper sequence, staged:

| Stage | Observation |
|---|---|
| Baseline (fresh DB, role `studio`) | `current_schema()`=`public`; all `to_regclass()` NULL |
| After 1st `MigrateUp` + `store.Migrate` | v6 + v16 recorded in `public.*`; `current_schema()` now **`studio`** |
| 2nd `MigrateUp` (framework) | Replays v1 → `policy "tenant_isolation" for table "invocations" already exists (42710)` — **byte-identical to CI**; creates `studio.schema_migrations` at **`1/dirty=true`** — the CI dirty error |
| 2nd `store.Migrate` (Studio) | Replays `000001` (idempotent DO block, succeeds silently) then `000002` → `policy "tenant_isolation" for table "apps" already exists (42710)` — **the second CI collision**; `studio.studio_schema_migrations` at `2/dirty=true` |
| OID proof | unqualified `to_regclass()` → OIDs `566335`/`566344` ≠ `public` OIDs `565623`/`565799`; `current_schemas` = `{pg_catalog, studio, public}` |

19 `tenant_isolation` policies intact afterward — failed replays rolled back; only the shadow tables + dirty flags remain. No concurrency needed (consistent with zero `t.Parallel`); sequential helpers suffice.

### Secondary lines

- `public.schema_migrations does not exist`: consistent with the passing negative test `TestMigrateRequiresFramework` (deliberately queries it on a framework-less scratch DB). Needs CI timestamps for final confirmation, but no other repo path produces it.
- `role "root"`: no repo code path connects as root — container healthcheck (`pg_isready`, no `PGUSER`) remains the only candidate; needs CI timestamps. Fixable regardless.
- CI/tree skew stands: reconcile line numbers against the CI commit.

### Decision for you

Forensics are closed; per instruction nothing was implemented. The fix belongs in studio-owned migration configuration (schema-qualified tracking or pinned `search_path`), never in conditional DDL. Say the word to proceed to implementation (Phase 1 decision → fix → regression matrix → CI correction → evidence report), or redirect as preferred.
