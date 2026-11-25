GO — implementation authorized

I reviewed `plans/prompt9_plan.md`. **The execution order and all three core design judgments are approved.** Proceed with the bounded Phase A closure. No additional approval is needed before implementation.

### **Three design judgments**

**1. Write-path secret scrubbing — approved.**

Scrubbing at `build.ingest_event` is the right ownership boundary: sanitize the authenticated event payload before inserting it or deriving question records. Keep read-path scrubbing as defense-in-depth for existing rows.&#x20;

prompt9_plan(1).md



One refinement: make the credential patterns token-aware. Do not let a generic `sk-` expression indiscriminately redact ordinary prose. Define the accepted token shapes, boundaries, and minimum credential-like suffix clearly, and test both detection and false positives. Include common underscore variants such as `sk_live_` and `sk_test_` where intended. Do not describe the pattern collection as an exhaustive secret detector.

Also, verify the callback signature against the original envelope before sanitizing its payload. The `app.invocations` raw-input persistence remains the explicitly disclosed Miniapp residual; do not claim that every copy across the entire system is sanitized.

**2. Idempotent, error-returning&#x20;**`Server.Close`**&#x20;— approved.**

`sync.Once` with a retained close error is appropriate. Update every call site to handle the returned error rather than silently discarding it. The restart test must prove that the first server's shutdown completed before the second server starts—not merely that `Close()` was called.&#x20;

prompt9_plan(1).md



**3. RLS migration and tests — approved.**

Migration `000016` is the correct durable correction. Preserve FORCE, enable RLS on all ten Phase A tables, and verify both catalog flags. The staged v15-to-v16 test and behavioral test using the application role are appropriate. Ensure the behavioral test proves that unarmed tenant context exposes zero rows and correctly armed context exposes only that tenant's rows.&#x20;

prompt9_plan(1).md



### **One qualification on M4**

I approve correcting the inaccurate "gapless" comment, but the proposed round-trip and version gates establish schema behavior, not necessarily migration-file integrity. Before editing `000014`, inspect whether the actual migration runner verifies migration checksums or otherwise treats applied migration files as immutable.

- If it does not, the disclosed comment-only edit is acceptable.
- If it does, leave the historical migration untouched and correct the statement in current documentation instead.

This is a guardrail, not a reason to hold the other work.

### **Execution authorization**

Proceed in the planned order:

1. RLS migration and red-before/green-after tests.
2. Write-path secret scrubbing and end-to-end canary tests.
3. Genuine shutdown/restart lifecycle.
4. M1–M4, subject to the M4 qualification.
5. Full gates and corrected evidence report.

Keep M5–M8 recorded and carried. Preserve every existing boundary: no Phase B work, no Miniapp changes, no F1/G-01 changes, no P5 expansion, no infrastructure changes, and no commits or pushes.

**Acceptance decision:** Once the three mandatory corrections are demonstrated by the corrected evidence and all required gates pass, I will accept Phase A and move directly to Phase B. No further general adversarial review cycle.