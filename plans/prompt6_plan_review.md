GO — prompt6_plan is approved for execution. I found no foundational or structural blocker. It correctly sequences the F2/F3/F7 fixes before Service-dependent tools, and preserves the P4 scope and external-effect restrictions. 
prompt6_plan.md

I have two implementation instructions to make explicit before execution.
1. Correct the idempotency-key default
The proposed deterministic content-hash idempotency key for writes needs refinement.

A content hash identifies the payload; it does not identify a distinct user intention. Two legitimate requests with identical payloads might be separate operations.

Use:
- A unique operation/idempotency key for each intentional write.
- Reuse that key when retrying the same operation.
- Bind the key to a payload fingerprint.
- Reject reuse of the same key with a different payload.
- Preserve content hash as spec identity, not as the sole operation identity.
Add tests proving that retries replay, while two intentional identical operations with different keys remain independent.
2. F1 owner decision
Deliver the HMAC-versus-DB-held-credential-verification comparison as a separate decision document, not merely inline in the final report.

My preference is docs/architecture/decisions/003-session-issuance-proof.md, following the existing P3 decision-record convention. Do not implement either option until the owner decision is made.
Approved execution sequence
Phase	Decision
A — F2, F3, F7	GO FIRST
B — Studio tools and registry	After Phase A tests pass
C — Six screens	After real backend handlers exist
D — F5/F6/F8–F13	Within P4 scope, no unnecessary expansion
F1/F4	Document and preserve release restrictions

The audit INSERT/WITH CHECK issue must be tested after F2; if it fails, record the actual framework or migration boundary and do not work around it.

Final conditions remain unchanged: no P5, no external writes, G-01 untouched, Miniapp unchanged absent §14 evidence, frozen plans untouched, and everything uncommitted.

Proceed. No further planning review is needed.