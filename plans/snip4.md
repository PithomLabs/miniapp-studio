P4 APPROVED — GO

This revision resolves the bootstrap boundary without introducing a broad database privilege bypass. The three-role model, forced RLS, embedded migrations, runtime-only tool invocation, and six-screen scope are appropriate.

I have **one important implementation clarification**, not a reason to delay starting.

### **Clarify the three function contracts**

The requirement says all three SECURITY DEFINER functions use hash-only input.

That is straightforward for lookup and revoke. For `create`, however, the database must also know which tenant and credential binding own the new session, and when it expires.

Define one of these mechanisms explicitly:

- The function accepts the token hash plus server-authenticated tenant/binding/expiry values and validates them against the provisioned single-operator configuration.
- Or it accepts only the hash and derives the remaining fields from a trusted, pre-provisioned database record.

In either case, **never accept a plaintext token or trust browser-supplied tenant identity**. The application role must not be able to manufacture a session for another tenant.

Also verify the lookup role owns the functions, not the sessions table; has `NOLOGIN`, `NOBYPASSRLS`; has no membership path from the application role; and receives only the privileges required by those functions.

### **Approved execution order**

1. Server skeleton and fail-closed configuration.
2. Three-role session bootstrap and security regression tests.
3. Embedded migrations, forced RLS, and tenant-scoped constraints.
4. Studio tools registered and invoked exclusively through `runtime.Service`.
5. Six screens using real handlers and honest unconfigured states.
6. PostgreSQL 19/17 validation, tenant/session attack matrices, and secret-leak audits.

**Preserve:** no P5, no external writes, no Fly live mutations, no Miniapp changes without §14 justification, frozen documents untouched, and no commits.

**GO.** Implement P4. The function-signature clarification belongs in the implementation design; it does not warrant another approval cycle.