**Card 001 is a focused database regression verification task**, not a feature-development task.

Its purpose is to prove that the recent database fixes actually work against the real Neon PostgreSQL development branch.

It covers six tests:

1. Two privilege-parity tests.
2. Studio lookup role set-repair cleanup.
3. Migration down/up replay.
4. Tenant-setting cleanup when connections return to the pool.
5. Destruction and replacement of connections that cannot be reset.

The agent runs the tests, captures results and timings, and fixes any failures at their root cause.

**Explicitly out of scope:** a full-suite rerun, infrastructure changes, commits, and pushes. Full-suite verification is deferred to the pre-demo gate.&#x20;

001-suite-green-neon.md

In essence: **prove the recent database and connection-management fixes are safe before proceeding to B2 and B3.**

Changelog:

Cleanup root-caused (two layers). DROP OWNED fails with 42501 — the ADMIN-FALSE membership theory
     confirmed live. And SET ROLE worked, but the drop then failed on schema USAGE: the probe holds
     CREATE, which does not imply USAGE. Fixed by granting USAGE alongside CREATE.
  2. The PUBLIC failure is solved. The dump shows probe_transfer_fn surviving with default PUBLIC EXECUTE
     — SetRepair's leftover polluting the shared DB. Once cleanup goes green, line-260 goes with it.
  3. The privilege paradox narrowed. I read pgx v5.13.0's source: every batch error overwrites the result
     error, so mid-file statement failures would surface — the silent-42501 theory is dead, every
     GRANT/REVOKE succeeded, and my own dump is now the prime suspect (its INNER JOINs silently drop
     PUBLIC entries). The dump now prints raw proacl ground truth plus OID and by-text/by-OID
     cross-checks, so the next run ends this either way.

  Committed locally (push still yours). Falsifiable predictions for the confirm run: SetRepair green,
  line-260 green with 264/276 still red, parity red with the raw ACL naming the truth, and the other four
  green. 000027 stays out deliberately — this run must observe the hole, not the repair.



 Root cause closed and repair committed. The story in full:

  Mechanism (proven, not theorized). The migrator is a mere INHERIT FALSE member of studio_lookup: it
  passes ownership checks — hence clean v26 with zero errors — but its ACL writes on studio_lookup-owned
  functions take no effect. Tables were unaffected because they stay migrator-owned; OWNER TO targets only
  the function statements, so the ownership boundary is exactly the divergence line. Only the 000001 SET
  capability was ever real, and no migration used it. The raw ACL {=X/studio_lookup,...} on all four
  functions is the receipt.

  Repair (committed, 4 files). Forward 000027_function_acl_repair: SET ROLE → revoke PUBLIC on all four →
  restore the intended grants → self-asserting postconditions → RESET ROLE, with a down that never reopens
  the hole. The test-role mirror's three function grants moved onto a dedicated SET ROLE connection with
  double RESET. No new tests needed: the gate itself discriminates — post-repair, session_* test-side TRUE
  requires the mirror fix to actually stick, and 000027 fails its own migration loudly on any regression.
  Gates green, 426/306/0.

  Prediction for the green run: all seven PASS, including the cycle test proving 000027's down/up
  round-trip. One structural follow-up belongs on a separate card later: per-file adapter atomicity for
  the partial-application class.



 Card 001 is closed — verified from the log, not the headline. All seven PASS in 468s: cycle 198s
  (including the v27→v15→v27 round-trip through the amended 000026), parity 84s, roleattrs 44s, setrepair
  26s, pools 94s/21s, hook 0.2s. Zero failures, zero diagnostic dumps emitted, v27 reached throughout.

  What this gate actually proved: a real privilege hole (explicit PUBLIC EXECUTE on all four SECURITY
  DEFINER functions) is closed by 000027's owner-role repair with self-asserting postconditions, the
  test-role mirror writes function ACLs as the owner, and the probe cleanup is order-independent. Six
  commits sit locally spanning triage → mechanism → repair → closure:
