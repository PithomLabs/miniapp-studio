Goal
  
  Implement plans/prompt4.md: finish the P0 framework gate (Docker diagnosis → PG19 suite → import/approval-resumption spike → exit decision), then implement P2 (F1–F4 corrections, generic runner/interpreter, server renderer, dry-run)
  with the §5 test coverage. Hard stop if P0 cannot pass. Frozen files, scope, and no-commit rules respected throughout.
  
  Success Criteria
  
  • P0: Docker diagnosis recorded with exact cause; PG19 suite + restart-executed + race subset green at the recorded SHA, or a precise host-level blocker report with zero skipped-as-green tests; import spike proves Class A invoke plus
    park/decide/same-key resumption through pollApproval with Status observation-only.
  • P2: F1–F4 corrected with named regression tests; statically registered runner executing validated v1 specs through runtime.Service (GET-only, declared ops, normalized paths, server-resolved connections); renderer output honest
    (validated-only, escaped, no write controls); dry-run provably provider-free; tenant derived from server auth context with cross-tenant denial tested.
  • Deliverables per §8: updated docs/p0-evidence.md, P2 code + tests, final implementation report. Everything uncommitted.
  
  Approach
  
  Two facts shape the sequencing. First, pre-gathered diagnosis: the socket is nobody:nogroup rw and the user is in nogroup, yet my sandboxed shell still gets permission-denied — so this smells like sandbox socket restriction rather than
  host misconfiguration, meaning P0 PG steps may need the owner's shell; Phase A verifies that without touching permissions (no chmod, no bypass, per §A1). Second, F1–F4 are genuinely P2-adjacent (validation-boundary API shape, field
  matrix, path predicate, connection registry), so B1 lands first inside P2 and every finding ships with its named regression test failing-before/passing-after.
  
  Steps
  
  1. Phase A1: complete Docker diagnosis (context, daemon reachability, socket, groups), record verdict; stop with precise required action if user-shell/admin step needed.
  2. Phase A2–A4: PG19 suite, restart-executed checks, race subset, import/approval-resumption spike, exit decision with exact SHA (or root-caused blocker, §14-classified; no P2 on failure).
  3. Phase B1: F1 validation boundary (nil-on-error + struct-level entry + comment accuracy), F2 field matrix, F3 shared path predicate + normalization policy, F4 tripwire + server connection registry.
  4. Phase B2–B6: runner registration over runtime.Service, strict interpreter, read-only runtime enforcement, tenant-from-auth, renderer, dry-run.
  5. Phase §5 tests: admission/interpreter/renderer/integration/dry-run/failure-security matrices; final report; readback audit.
  
  Validation Plan
  
  • Gates re-run per phase (gofmt, vet, build, test, -race for touched packages); framework suites only in the authorized PG environment with exit codes recorded.
  • Each F1–F4 correction demonstrated by its named regression test failing on the pre-fix tree and passing after (verified via git stash-free before/after runs, repo left uncommitted).
  • Traceability: every §5 bullet maps to a test; every prompt4 prohibition re-checked at the end (no P3/P5, no writes, no frozen-file edits, no commits).
  
  Risks / Open Questions
  
  • Likeliest outcome of step 1 is "owner-shell action required" — that is a valid P0 result, not a failure; P2 framework-dependent work then waits per the hard gate.
  • P1-non-blocking MINORs (F5–F9) are document-and-pin only unless a P2 invariant needs them; will not expand into them unprompted.
  
Reply go to start, or tell me what to adjust.
