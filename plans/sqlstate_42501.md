# Architect diagnostic request: persistent SQLSTATE 42501 on Neon PG18

## Symptom (unchanged across three fix attempts)

`task dev:bootstrap` → `dev init` → Studio migrations fail on a fresh database with:

```
ERROR: must be able to SET ROLE "studio_lookup" (SQLSTATE 42501)
```

Full error line (exact migration version + statement) not yet captured; error text passes through DSN
redaction, so it is safe to share. No credentials may appear in any reply — ever.

## Environment (verified)

- Neon PG18 non-prod project, persistent `dev` branch; migrator connects as the branch owner role over
  verify-full TLS (bound endpoint policy, live handshake proven).
- Owner is NOT a superuser; it holds CREATEROLE (proven: `CREATE ROLE studio_lookup` in 000001 succeeds).
- Every bootstrap attempt provisions a FRESH run-pattern database (`studio_test_run_*`, new random name),
  applies the framework baseline (miniapp chain, succeeds — Studio 000002 begins executing), ensures roles,
  then runs the Studio chain via golang-migrate (pgx5 driver, one transaction per migration).
- `go run` rebuilds on every invocation; migration SQL is `go:embed`ded, so edits take effect on re-run
  (assumed reliable — never directly verified; see Q3).

## Failing-area code (exact)

`store/migrations/000001_roles.up.sql` (after fix F2):

```sql
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'studio_lookup') THEN
    CREATE ROLE studio_lookup WITH NOLOGIN NOBYPASSRLS;
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_auth_members m
    JOIN pg_roles r ON r.oid = m.roleid
    JOIN pg_roles u ON u.oid = m.member
    WHERE r.rolname = 'studio_lookup' AND u.rolname = session_user
  ) THEN
    EXECUTE format('GRANT studio_lookup TO %I', session_user);
  END IF;
END $$;
```

`store/migrations/000002_schema.up.sql` (after fix F2b), grant section preceding four
`ALTER FUNCTION ... OWNER TO studio_lookup` statements:

```sql
GRANT CREATE ON SCHEMA studio TO studio_lookup;
DO $$ BEGIN
  IF NOT EXISTS ( /* same membership check as 000001 */ ) THEN
    EXECUTE format('GRANT studio_lookup TO %I', session_user);
  END IF;
END $$;
```

Both fixes are idempotent; both predate the latest failing run. The `studio_lookup` role is NOLOGIN,
NOBYPASSRLS, branch-scoped (Neon). No test pins against owner-side membership or lookup CREATE.

## Fix history

- **F1:** migrate driver scheme (`postgresql://` → `pgx5://` rewrite). Fixed, verified, unrelated to 42501.
- **F2:** added the 000001 membership grant (Postgres OWNER TO rule: executor must hold membership in the new
  owner). Same error after.
- **F2b:** added `GRANT CREATE ON SCHEMA studio TO studio_lookup` + membership re-assert inside 000002 (second
  half of the OWNER TO rule; self-sufficiency regardless of prior state). Same error after.

## Ruled out (with reason)

- Stale dirty state: every attempt uses a fresh database; orphans are abandoned, never reused.
- Framework chain: applies cleanly (Studio 000002 starts executing).
- 000009 as the failure point: it cannot execute before 000002 passes.
- Privilege-parity impact: the membership test constrains runtime roles as members; the owner is the member.
  No test pins lookup's schema privileges.
- DSN/TLS/endpoint: live fingerprint proves bound verify-full connectivity on the same branch.

## Questions for the architect

1. **Q1 (root cause):** Given membership + CREATE are both asserted immediately before the OWNER TO statements
   in the same migration, what mechanism can still produce `must be able to SET ROLE "studio_lookup"`? Is there
   a Neon-specific restriction on `GRANT <role> TO <branch-owner>`, on `ALTER ... OWNER TO`, or on
   branch-scoped `pg_auth_members` visibility that would silently void the grant?
2. **Q2 (discriminating evidence):** What is the single cheapest check that distinguishes "membership missing at
   OWNER TO time" from "a different statement is failing"? Propose owner-terminal diagnostics that print no
   secrets (e.g., version/dirty state via the existing redacted fingerprint path, or catalog queries whose
   output contains only booleans/counts).
3. **Q3 (build staleness):** Is there any known `go run` + `go:embed` staleness mode (build cache, Taskfile
   working directory, module replacement) that could run pre-fix migration SQL despite saved edits — and what
   one command proves which SQL actually executed?
4. **Q4 (minimal fix):** Assuming Q1 confirms a Neon restriction rather than a logic bug, what is the smallest
   ownership design that preserves the threat model (SECURITY DEFINER functions owned by a non-login role,
   app role with zero session/operator access, FORCE RLS)? Alternatives on the table: owner-held membership
   (current, failing), functions owned by a Neon-provisioned role, or restructuring the OWNER TO out of the
   chain.

## Constraints on the answer

- Do not request credentials, DSNs, IDs, hostnames, or pasted values of any kind.
- Prefer a diagnosis that can be confirmed by one owner-terminal command with secret-free output.
- If the answer is "collect X first", specify the exact command and the exact expected-vs-actual decision rule.
