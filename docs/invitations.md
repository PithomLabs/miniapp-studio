# Tenant invitations (F0)

Invitations enroll new operator bindings into a tenant workspace on
the existing flat identity model: operators are `(tenant, binding)`
pairs with no role hierarchy, so any session-authenticated operator
of a tenant may invite, revoke, and resend. An accepted invitation
grants only its predetermined binding within its designated tenant.

## Lifecycle

1. **Create** (`dev invite-create`): records `(tenant, binding,
   token_hash, expiry)`; the raw 256-bit token goes to a new 0600
   `--token-file` (never printed) for out-of-band delivery.
2. **Accept** (`dev invite-accept`): reads the token from its 0600
   file and runs ONE transaction — `invitation_accept` (guard →
   operator upsert → consume) then `session_create` — so a failed
   accept rolls back completely and stays retryable. The issued
   session token goes to a new 0600 `--session-token-file`.
3. **Revoke** (`dev invite-revoke`): kills the token immediately.
4. **Resend** (`dev invite-resend`): rotates — revokes the old hash
   and issues a fresh token for the same binding.
5. **List** (`dev invite-list`): metadata with live/consumed/
   revoked state.

## Security properties

- Hash-only persistence (`store.MintToken`/`HashToken`, same rule
  as sessions); canary tests pin it.
- Uniform accept failure (`ErrInviteInvalid`) for bad, expired,
  revoked, consumed, and foreign tokens — no enumeration oracle.
- Exactly-once acceptance under concurrency (atomic consume in
  the accept function; race test pins one winner, one session).
- Tenant stamped from the authorized parameter, never client
  input; RLS confines all member operations.
- Expiry enforced in the function guard (default 7d, max 30d).

## Residual

HTTP accept + session-bound invite endpoints are a later
increment: current services take the authorized tenant as a
parameter (documented contract), and dev verbs satisfy it via
owner-terminal trust. No server handlers were added here.
