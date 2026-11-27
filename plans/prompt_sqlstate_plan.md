## Goal

Prove the architect's CREATEROLE auto-membership diagnosis live, then fix the `studio_lookup` SET-capability invariant at its root — one canonical repair, fail-closed postconditions, live regression coverage — without weakening the ownership security boundary.

## Success Criteria

- The six-boolean diagnostic is recorded from the live dev branch and matches (or refutes with a new lead) the predicted `MEMBER true / SET false / ADMIN true` signature before any migration change.
- One canonical repair exists (in 000001): explicit `GRANT studio_lookup TO SESSION_USER WITH INHERIT FALSE, SET TRUE`, no existence-gated divergence, no touching the bootstrap grant.
- 000002 asserts the postcondition (`pg_has_role(..., 'SET')` true, CREATE held) and fails with a named invariant instead of repairing a second way.
- Migration failures capture version, SQLSTATE, server position/routine, and embedded-byte SHA-256 with zero credential exposure.
- The 10-step live regression scenario passes on Neon PG18; RLS, parity, and no-membership tests stay green; clean-database bootstrap repeats on the same branch.
- Baseline facts (this turn, read-only): no Go workspace, module is this repo, embed pattern `migrations/*.sql` confirmed; source digests recorded for 000001/000002 to compare against runtime bytes later.

## Approach

- Diagnose before operating: the diagnostic verb and owner-terminal run gate the migration edit. If the booleans differ from prediction, stop and chase the actual role chain — no blind fix.
- Repair once, assert everywhere else: 000001 owns the SET grant; 000002 only asserts the postcondition. This kills the F2/F2b divergence the architect flagged.
- Branch-persistent roles are the reason fresh databases didn't help — and also why the fix works: re-running fixed 000001 on a fresh database repairs the existing branch-level role in place.
- Honest test layering: repo-local tests pin the SQL text (read from the embed FS, i.e. what ships), the error capture, and the statement locator; the behavior scenario is an env-gated live test using a scratch probe role (created and dropped by the test) that skips plainly without Neon.

## Steps

1. **Diagnostic verb.** Add `studio dev diag-roles`: runs the architect's six-boolean query through a fake-testable `dbenv` collector, prints booleans plus already-known binding names only. Unit test with the fake querier. Owner runs `task dev:diag-roles` (new task, same file preconditions) and confirms completion; I verify from a result file.
2. **Decision gate.** Booleans match prediction → proceed. Anything else → stop, investigate the actual grantor/chain first.
3. **Migration repair.** 000001: replace the existence-gated grant with the unconditional explicit `WITH ADMIN FALSE, INHERIT FALSE, SET TRUE` grant (all three options explicit — omitted options retain existing values on modification; bootstrap grant untouched) plus a `pg_has_role(... 'SET')` postcondition that aborts loudly. 000002: replace my re-assert block with a postcondition assert naming the failed invariant; keep the CREATE grant. Pre-first-application reasoning already documented; extend the in-file note. Explicitly document: editing 000001 does not re-run it where already applied — disposable run DBs are recreated for verification; any persistent DB with 000001 recorded needs a separately controlled repair/forward-migration (none exists in this program: migrations run only on fresh run DBs).
4. **Failure instrumentation.** In the migrate path, unwrap errors with `errors.As` for `*pgconn.PgError`; record version, SQLSTATE, position, routine, and SHA-256 of the embedded bytes actually supplied. Add a dollar-quote-aware locator mapping position to statement, verified against a synthetic fixture. Fallback if the driver doesn't surface position cleanly: version + hash + SQLSTATE still pin the file.
5. **Regression tests.** New env-gated live test executing the architect's 10-step scenario against a scratch probe role (full cleanup); static tests asserting the repair SQL and postconditions exist in the embedded bytes; unit tests for the locator and error capture. No duplication of the existing parity/RLS/no-membership coverage.
6. **Freshness proof.** Runtime embedded-byte digest emitted at migrate time; owner compares against source `sha256sum`. Closes the stale-embed question with evidence, no cache clearing.
7. **Live acceptance.** Owner-terminal bootstrap repeats on the same branch; record migration SHA, version, PG version, capability booleans, counts. Update the evidence file; close or carry each pending item explicitly.

## Validation Plan

- Step 1: `go test ./cmd/studio/ ./dbenv/` (collector + shape tests); owner `task dev:diag-roles` output file reviewed for the boolean results.
- Step 3: `store` static SQL tests green; `gofmt`/`vet` clean; full unit suite re-run with counts.
- Step 4: locator unit test on synthetic multi-statement fixture; simulated PgError capture test.
- Step 5: live test green on Neon (owner terminal) plus honest-skip proof without binding; existing parity/RLS suites green.
- Step 7: repeated clean bootstrap on the same branch; evidence file updated with digests, versions, counts.

## Risks / Open Questions

- If migrate wraps driver errors opaquely, `errors.As` may not reach `PgError`; fallback (message + version + hash) still delivers file-level pinning; accepted explicitly.
- The bootstrap-superuser auto-membership is intentionally left untouched per the architect's nuance; the explicit grant is additive.
- Live steps depend on owner-terminal execution; each has exact commands and a file-based evidence return, no pasted values.
- Open questions: none blocking. All unknowns above are verification steps with fail-closed outcomes.
