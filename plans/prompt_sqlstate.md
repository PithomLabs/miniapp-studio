I found a much stronger explanation for the persistent `SQLSTATE 42501`, and I believe it identifies the underlying bug rather than another symptom.

**The likely root cause is PostgreSQL 16+&#x20;**`CREATEROLE`**&#x20;membership behavior interacting with an incorrect idempotency check. This is not evidence of a Neon-specific restriction.**

There is a particularly important detail in the PostgreSQL 18 documentation:

When a non-superuser with `CREATEROLE` creates a new role, PostgreSQL automatically grants that newly created role back to its creator with:

- `ADMIN TRUE`
- `SET FALSE`
- `INHERIT FALSE`

This is intentional. It allows the creator to administer the new role without automatically inheriting its privileges or being able to `SET ROLE` into it. PostgreSQL explicitly documents that the creator can separately grant itself the `SET` option.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.postgresql.org\&sz=32)

PostgreSQL+1



Now compare this with your migration:

1. `CREATE ROLE studio_lookup` succeeds.
2. PostgreSQL automatically creates a membership for the creator.
3. Your `pg_auth_members` existence check finds that membership.
4. The conditional `GRANT` is skipped.
5. The migration subsequently executes `ALTER FUNCTION ... OWNER TO studio_lookup`.
6. PostgreSQL rejects it because membership exists, but `SET ROLE` is not permitted.

This explains why F2 and F2b did not solve the problem: they checked membership existence, not the actual ability to change roles.

There is a second important detail: PostgreSQL role memberships are cluster-wide, not database-specific. Creating a fresh `studio_test_run_*` database on the same Neon branch does not reset `pg_auth_members`.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.postgresql.org\&sz=32)

PostgreSQL+1



That explains why fresh-database testing can repeatedly reproduce a failure caused by persistent branch-level role metadata.

## **1. The cheapest discriminating diagnostic**

Have the agent run this query through the existing authenticated Neon connection, using the same migration login:

```
SELECT
    pg_has_role(session_user, 'studio_lookup', 'MEMBER')
        AS is_member,
    pg_has_role(session_user, 'studio_lookup', 'SET')
        AS can_set_role,
    pg_has_role(session_user, 'studio_lookup', 'USAGE')
        AS inherits_privileges,
    COALESCE((
        SELECT bool_or(am.set_option)
        FROM pg_auth_members am
        JOIN pg_roles target ON target.oid = am.roleid
        JOIN pg_roles member ON member.oid = am.member
        WHERE target.rolname = 'studio_lookup'
          AND member.rolname = session_user
    ), false) AS direct_set_option,
    COALESCE((
        SELECT bool_or(am.admin_option)
        FROM pg_auth_members am
        JOIN pg_roles target ON target.oid = am.roleid
        JOIN pg_roles member ON member.oid = am.member
        WHERE target.rolname = 'studio_lookup'
          AND member.rolname = session_user
    ), false) AS direct_admin_option,
    current_user = session_user AS current_matches_session;
```

No password, DSN, hostname or secret is selected.

Expected diagnosis if my hypothesis is correct:

| Result                    | Expected |
| ------------------------- | -------- |
| `is_member`               | true     |
| `can_set_role`            | false    |
| `inherits_privileges`     | false    |
| `direct_set_option`       | false    |
| `direct_admin_option`     | true     |
| `current_matches_session` | true     |

That combination is the signature of the automatic `CREATEROLE` membership described above.

If `can_set_role` is already true, do not apply this diagnosis blindly. Capture the exact failing SQL statement first.

## **2. Minimal architectural correction**

The fix should be to grant the migration principal an explicit `SET TRUE` membership while preserving non-inheritance.

After ensuring `studio_lookup` exists:

```
GRANT studio_lookup TO SESSION_USER
    WITH INHERIT FALSE, SET TRUE;
```

This is supported PostgreSQL role-membership syntax. PostgreSQL permits modifying membership options through `GRANT`, and the `SET` option is specifically what controls `SET ROLE`.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.postgresql.org\&sz=32)

PostgreSQL+1



An important nuance: PostgreSQL's automatic membership is granted by the bootstrap superuser and cannot itself be altered by the CREATEROLE user. The creator's `ADMIN` option allows it to grant an additional membership to itself. That additional grant is the appropriate mechanism; do not attempt to revoke or manipulate the bootstrap grant.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.postgresql.org\&sz=32)

PostgreSQL



This preserves the security model:

- `studio_lookup` remains `NOLOGIN NOBYPASSRLS`.
- The migration principal can explicitly `SET ROLE`.
- Its normal session does not inherit lookup privileges.
- The application runtime role receives no membership.
- No `BYPASSRLS` permission is introduced.
- `FORCE ROW LEVEL SECURITY` remains intact.

Also independently verify that `studio_lookup` has `CREATE` on the `studio` schema. PostgreSQL requires both conditions for an ownership transfer: the executor must be able to `SET ROLE` to the new owner, and the new owner must have `CREATE` on the object's schema.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.postgresql.org\&sz=32)

PostgreSQL



## **3. Instructions for the coding agent**

# Root-Cause Investigation and Durable Fix: SQLSTATE 42501 / studio_lookup Ownership

## Mission

Resolve the recurring PostgreSQL error:

`must be able to SET ROLE "studio_lookup" (SQLSTATE 42501)`

Do not apply another speculative SQL patch.

Investigate the underlying role-membership lifecycle, prove the cause, implement the smallest correct fix, and add regression tests that reproduce the actual PostgreSQL behavior.

Preserve the security architecture:

- `studio_lookup` remains NOLOGIN and NOBYPASSRLS.
- SECURITY DEFINER functions remain owned by `studio_lookup`.
- The application runtime role receives no membership in `studio_lookup`.
- FORCE ROW LEVEL SECURITY remains enabled.
- No BYPASSRLS grants.
- No superuser dependency.
- No Neon-specific privilege workaround without evidence.

Never request, print, or expose credentials, connection strings, hostnames or secret values.

## 1. Investigate the PostgreSQL CREATEROLE behavior first

The highest-priority hypothesis is PostgreSQL 16+ automatic role membership.

Official PostgreSQL 18 behavior:

When a non-superuser CREATEROLE principal creates a role, PostgreSQL automatically creates a membership granting the newly created role back to its creator with:

- ADMIN TRUE
- SET FALSE
- INHERIT FALSE

This explains why a simple membership-existence check can return true while `SET ROLE` remains prohibited.

The existing migrations check whether a membership row exists but do not inspect `set_option`.

This is a likely root cause, not yet a live Neon finding. Verify it before changing the migration.

Also account for PostgreSQL catalog scope:

`pg_auth_members` is cluster-wide, not database-specific.

Creating a fresh run-pattern database does not reset roles or membership records on the same Neon branch.

Do not treat fresh database creation as proof of fresh role state.

## 2. Run the discriminating diagnostic

Use the existing owner-terminal database connection and run the following query without exposing the connection string:

```
SELECT
    pg_has_role(session_user, 'studio_lookup', 'MEMBER')
        AS is_member,
    pg_has_role(session_user, 'studio_lookup', 'SET')
        AS can_set_role,
    pg_has_role(session_user, 'studio_lookup', 'USAGE')
        AS inherits_privileges,
    COALESCE((
        SELECT bool_or(am.set_option)
        FROM pg_auth_members am
        JOIN pg_roles target ON target.oid = am.roleid
        JOIN pg_roles member ON member.oid = am.member
        WHERE target.rolname = 'studio_lookup'
          AND member.rolname = session_user
    ), false) AS direct_set_option,
    COALESCE((
        SELECT bool_or(am.admin_option)
        FROM pg_auth_members am
        JOIN pg_roles target ON target.oid = am.roleid
        JOIN pg_roles member ON member.oid = am.member
        WHERE target.rolname = 'studio_lookup'
          AND member.rolname = session_user
    ), false) AS direct_admin_option,
    current_user = session_user AS current_matches_session;
```

Expected result for the leading hypothesis:

- is_member = true
- can_set_role = false
- inherits_privileges = false
- direct_set_option = false
- direct_admin_option = true
- current_matches_session = true

Record only these boolean results.

If the result differs, investigate the actual role chain and grantor before proceeding.

## 3. Correct the membership invariant

Replace membership-existence checks with an effective SET capability check.

Do not interpret membership as equivalent to the ability to SET ROLE.

After ensuring the role exists, establish the intended explicit membership:

```
GRANT studio_lookup TO SESSION_USER
    WITH INHERIT FALSE, SET TRUE;
```

Important:

- The automatic bootstrap-superuser membership created by CREATEROLE is not itself changed.
- The explicit grant provides the migration principal with the required SET capability.
- Do not introduce ADMIN TRUE on the additional membership.
- Do not allow application runtime roles to receive this membership.
- Preserve the creator's existing administrative authority only where PostgreSQL requires it for managing the role.
- If the authenticated migration principal lacks ADMIN OPTION and cannot perform the grant, stop with an explicit permission diagnostic. Do not escalate privileges or switch to a Neon superuser workaround.

Make the operation idempotent.

Postcondition:

```
SELECT pg_has_role(session_user, 'studio_lookup', 'SET');
```

Must return TRUE.

Use this effective capability check as the invariant, including in subsequent migration preconditions.

Inspect both `000001_roles.up.sql` and `000002_schema.up.sql`. Avoid maintaining two divergent implementations of membership repair.

## 4. Verify both ownership-transfer prerequisites

Immediately before the first ownership transfer in `000002`, verify:

1. The migration principal can SET ROLE to `studio_lookup`.
2. `studio_lookup` has CREATE on the `studio` schema.
3. The expected migration principal is executing the migration.
4. The functions currently belong to the expected migration owner.

Do not assume that granting CREATE on the schema solves the SET ROLE requirement.

If either required privilege is missing, report which invariant failed.

## 5. Capture the exact failing statement

The original diagnostic does not identify the exact migration statement.

Add safe error instrumentation around the migration execution path.

For PostgreSQL `PgError`, capture:

- Migration version and filename.
- SQLSTATE.
- Server error position.
- Error routine.
- Internal position, when available.
- SHA-256 of the exact migration bytes supplied to the migration driver.

Never emit a DSN, password or credential.

Use the PostgreSQL error position to identify the actual failing statement in the migration source.

Do not implement a naive semicolon splitter: the migration contains dollar-quoted procedural blocks.

If the failing statement is not one of the expected OWNER TO statements, investigate that statement independently.

## 6. Prove embedded migration freshness

Do not assume stale embedded SQL merely because the latest edits appear ineffective.

Normal Go build-cache behavior should account for embedded source changes.

Investigate:

- `go env GOMOD GOWORK`.
- The module selected by the actual Taskfile invocation.
- The directory and embedded files reported by `go list`.
- Whether Taskfile invokes `go run` from the intended repository.
- Whether a stale externally built binary is used anywhere.

Add a temporary diagnostic test or instrumentation that calculates SHA-256 from the actual embedded migration bytes immediately before they are supplied to the migration driver.

Compare the runtime digest against the intended source migration digest.

The report must identify the exact embedded migration version and SHA-256.

Do not indiscriminately clear the build cache or change module/workspace configuration without evidence.

## 7. Regression tests must reproduce the actual failure

Build tests around the PostgreSQL 16+ CREATEROLE semantics, not just a manually constructed membership with arbitrary privileges.

The regression scenario must:

1. Execute CREATE ROLE as a non-superuser with CREATEROLE.
2. Verify the automatically generated membership.
3. Verify membership exists but `SET ROLE` is false.
4. Demonstrate that the old existence-only condition incorrectly skips the grant.
5. Execute the corrected explicit SET grant.
6. Verify `pg_has_role(..., 'SET')` returns true.
7. Verify `INHERIT FALSE` remains in effect.
8. Verify the migration principal can complete the ownership transfer.
9. Verify the application role cannot SET ROLE to `studio_lookup`.
10. Verify NOBYPASSRLS and FORCE RLS invariants remain intact.

Test idempotent re-execution and an already-existing role.

If a live Neon test is necessary, use an approved non-production branch. Unit tests or mocks do not replace the live proof.

## 8. Do not redesign ownership prematurely

Retain `studio_lookup` as the non-login SECURITY DEFINER owner unless evidence proves this design cannot work under Neon.

Do not move function ownership to the Neon provisioning owner, `neon_superuser`, or the application login.

Do not remove the ownership boundary merely to bypass migration errors.

Do not restructure all ownership transfers out of the migration chain unless a specific verified platform restriction requires it.

## 9. Live acceptance criteria

The fix is accepted only after:

- The diagnostic demonstrates the actual membership state.
- The migration principal has effective SET capability.
- The schema CREATE prerequisite is independently verified.
- The exact previously failing migration statement succeeds on Neon PG18.
- All four SECURITY DEFINER functions have the intended owner.
- The application runtime principal cannot SET ROLE to `studio_lookup`.
- RLS and tenant-isolation tests remain green.
- Repeated clean-database bootstrap succeeds on the same persistent Neon branch.
- No unexplained skips or false-green results remain.

Record the migration SHA, exact migration version, PostgreSQL version, database role capability results and test counts.

Do not claim live proof based solely on unit tests.

## 10. Guiding principle

Solve the role-membership lifecycle defect, not merely the immediate migration error.

The correct design must distinguish:

- Membership existence.
- Privilege inheritance.
- Ability to SET ROLE.
- ADMIN OPTION.
- Schema ownership-transfer prerequisites.

These are distinct PostgreSQL capabilities.

Correct the actual invariant, test the failure mechanism, and preserve the security boundary.

## **4. How to establish exactly which SQL failed**

The error message alone identifies a failed `SET ROLE` permission check, but not necessarily which ownership transfer triggered it.

Require the agent to instrument the migration runner to capture the wrapped `*pgconn.PgError` using `errors.As`.

Capture:

- Migration version (`000002`).
- SQLSTATE.
- PostgreSQL error position.
- PostgreSQL error routine.
- SHA-256 of the embedded migration bytes actually supplied to the driver.

Use the server error position to identify the offending statement. Do not split SQL naively on semicolons because procedural blocks contain semicolons.

## **5. Is Go embedding stale?**

I consider stale embedding a lower-probability explanation.

Go's `embed` package incorporates files at compile time, and the Go build cache accounts for build inputs. Ordinary source edits should not require clearing the build cache.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://pkg.go.dev\&sz=32)

Go Packages+1



More plausible issues are:

- Taskfile runs from an unexpected module.
- A workspace replacement selects another checkout.
- A separately built executable is being invoked.
- The migration runner reads another embedded filesystem.

Have the agent verify:

```
go env GOMOD GOWORK
go list -f '{{.Dir}} {{.EmbedFiles}}' ./store/migrations
sha256sum store/migrations/000002_schema.up.sql
```

Then compare that source digest against the SHA-256 calculated from the actual embedded bytes immediately before migration execution.

That is stronger evidence than deleting caches and hoping the problem disappears.

## **6. My preferred architectural outcome**

I would retain the existing ownership design.

**Migration principal**

Controlled migration authority

Can explicitly SET ROLE to `studio_lookup`.

**studio_lookup**

Non-login security owner

Owns SECURITY DEFINER functions; NOLOGIN, NOBYPASSRLS.

**Application runtime role**

Restricted execution

No membership in `studio_lookup`; no SET ROLE access; FORCE RLS remains enforced.

I would reject moving function ownership to a broadly privileged Neon administrative role just to make migrations pass.

The present design has a coherent security rationale. The migration principal needs explicit ownership-transfer authority, not the application.

## **My conclusion**

The key insight is that **PostgreSQL membership is not a Boolean relationship**.

It has distinct capabilities:

| Capability | Meaning                                |
| ---------- | -------------------------------------- |
| MEMBER     | Membership exists                      |
| INHERIT    | Privileges are automatically inherited |
| SET        | Can execute `SET ROLE`                 |
| ADMIN      | Can grant or revoke membership         |

Your previous implementation treated `MEMBER` as proof of `SET`. PostgreSQL 18 explicitly makes those independent.

I would investigate and fix that invariant before considering Neon restrictions, changing ownership architecture, or restructuring migrations.

**The coding agent's priority should be: reproduce the automatic CREATEROLE membership, prove the skipped-GRANT defect, repair the SET capability, and verify the runtime security boundary remains unchanged.**

That is a root-cause fix rather than a fourth workaround.