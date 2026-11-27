**Architectural decision: APPROVED, with two mandatory implementation safeguards.**

The eight-unit plan correctly addresses the root causes. I agree with the two snapshot domains and the decision to bind the image by resolving and comparing its digest rather than introducing another Docker dependency.&#x20;

prompt4_plan.md



I found two details that must be corrected during implementation:

### **1. Do not hardlink authoritative evidence into the workspace**

U4 proposes writing the owner-side result first, then publishing the repository copy using a no-clobber hardlink.

**This is unsafe.** Hardlinks share the same inode. An agent able to modify the workspace copy could consequently modify the supposedly protected owner-side anchor.

Use two independent files:

- Authoritative result in owner-protected `StateDir`.
- Separate no-clobber copy in the repository.
- Verify the repository copy against the owner's stored hash.

prompt4_plan.md

### **2. Protect owner-side state from the entire agent sandbox**

Being outside `RepoRoot` is insufficient. The agent previously demonstrated access to files under `/tmp`.

`StateDir`, `EnvFile`, build pins and authorization state must be outside **all agent-accessible and agent-writable paths**, not merely outside the repository.

Verify this at installation and fail closed if the boundary cannot be established.

prompt4_plan.md

### **Execution authorization**

GO — one remediation pass

Proceed with all eight units, including these safeguards. Preserve the existing authorization architecture, add regression tests, and do not touch Docker, PostgreSQL or systemd.

H1b/L5's owner diff-review limitation is accepted as honestly documented. We cannot technically distinguish malicious test-gutting from legitimate development simply by examining a dirty worktree.

**No additional planning or broad review cycle.** After implementation, perform one focused verification of the corrected security boundaries, then decide installation readiness.

Reality first. Build the fix, prove it, move forward.