**GO — P4 is approved to start.** The proposed bootstrap mechanism is sound as a design direction, and the rest preserves the approved architecture: fail-closed server startup, server-derived tenancy, embedded migrations, runtime-only tool invocation, and thin screens over real handlers.

I would make one correction to the acceptance tests: because the app role has no `SELECT` privilege on the sessions table, a direct SQL read should fail with a permission-denied error, not return zero rows. Test that denial explicitly; test zero-row results through the authorized lookup function for forged, expired, and revoked tokens.

**GO. No further architectural adjustments are needed.** The proposed implementation is appropriate, with two security details that should be explicit in the coding-agent instructions.

### **1. Make the SECURITY DEFINER boundary genuinely narrow**

Use three distinct database identities:

| Role            | Responsibility                                                                           |
| --------------- | ---------------------------------------------------------------------------------------- |
| Migration/owner | Owns tables and applies migrations; never used for application requests                  |
| Session lookup  | Dedicated `NOLOGIN`, `NOBYPASSRLS` function-owner role                                   |
| Application     | Normal runtime access; no direct session-table SELECT, no table ownership, no RLS bypass |

The lookup role should receive only the necessary column privileges and a dedicated RLS policy. It must not be assumable by the application role.

The function itself must:

- Accept only the SHA-256 token hash.
- Use fixed SQL, no dynamic SQL.
- Return only tenant ID, binding ID, and expiry.
- Filter revoked and expired sessions.
- Use fully qualified object names and a hardened `search_path`.
- Have `PUBLIC` execution revoked and execution granted only to the application role.

PostgreSQL explicitly warns that `SECURITY DEFINER` functions execute with their owner's privileges, that unsafe `search_path` settings can permit object substitution, and that function execution is granted to `PUBLIC` by default.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.postgresql.org\&sz=32)

postgresql.org+1



### **2. Strengthen the role and RLS tests**

Verify:

- App role cannot SELECT from sessions (expect permission denied, SQLSTATE `42501`, not zero rows).
- Lookup function returns no result for forged, expired, and revoked tokens.
- Neither app nor lookup role has `BYPASSRLS`.
- Neither can become the migration/table-owner role through role membership.
- Runtime roles do not own protected tables.
- RLS is enabled and forced where appropriate. PostgreSQL table owners normally bypass RLS unless `FORCE ROW LEVEL SECURITY` is applied.&#x20;

  [image](https://www.google.com/s2/favicons?domain=https://www.postgresql.org\&sz=32)

  postgresql.org+1



### **Approved execution order**

1. Server foundation and fail-closed configuration.
2. Sessions and the narrow lookup boundary.
3. Embedded migrations and tenant RLS.
4. Runtime-registered Studio tools and registry.
5. Six screens over real handlers.
6. PostgreSQL 19 security and acceptance testing, plus PG17 compatibility spot-check.

Everything else stays as specified: one operator per workspace, no signup, no P5, no external writes, no Miniapp changes without §14 justification, and no commits.

**GO — implement P4.** The security details above are implementation constraints, not another approval gate.