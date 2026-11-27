**Decision: BLOCKED verdict upheld. One bounded remediation pass authorized.**

The review found genuine vulnerabilities, not theoretical concerns. H1/H2 are installation blockers because an agent can influence executable code despite the broker's operation allowlist. M1–M5 also need correction before trusting the broker's queue, recovery and acceptance evidence.&#x20;

adv_review_prompt3(1).md



### **Authorized remediation scope**

| Priority              | Findings | Required outcome                                                                         |
| --------------------- | -------- | ---------------------------------------------------------------------------------------- |
| Critical to integrity | H1, H2   | Complete build-context pinning and immutable owner-side staging                          |
| High importance       | M1, M2   | Race-safe queue claims and size-capped reads                                             |
| High importance       | M3, M4   | Owner-side authoritative results and safe crash reconciliation                           |
| High importance       | M5       | Enforce owner-side state and credential paths outside the workspace                      |
| Bounded hardening     | L1–L6    | Correct JSON parsing, queue behavior, status semantics, suite provenance and portability |

I also require image execution to be bound to the recorded image digest, not merely a mutable tag.

### **Instructions to the agent**

The adversarial review verdict BLOCKED is accepted.

You are authorized to perform ONE bounded repository-only remediation pass.

Fix H1, H2 and M1–M5. Address L1–L6 where corrections remain within the existing architecture. Bind container execution to the image digest produced by the verified build.

Priorities:

1. Build integrity: pin the complete testrunner build context, stage verified bytes in owner-controlled StateDir, re-hash the staged copy, and build only from that immutable snapshot. Include every build input.
2. Queue security: eliminate Lstat/ReadFile TOCTOU, reject symlinks and non-regular files, enforce size limits before allocation, and ensure the executed bytes are exactly the claimed bytes.
3. Evidence integrity: persist authoritative job results and hashes outside agent-writable workspace paths. Repository result files are untrusted projections.
4. Recovery integrity: startup reconciliation must never overwrite existing terminal results or fabricate successful/UNKNOWN records from unvalidated residue.
5. Configuration integrity: reject StateDir, EnvFile, pin files and lock files inside or symlinked into RepoRoot. Enforce ownership and permissions.
6. Suite provenance: record exact source revision, working-tree state, build digest, gate commands and package-level results in owner-controlled state. Do not equate an exit code alone with independent proof that the source or tests were trustworthy.
7. Correct low-severity parsing, quarantine, status and portability defects without unrelated refactoring.

Preserve all existing security boundaries and tests. Add regression tests reproducing every corrected exploit, particularly P3 (queue swap) and P7 (unpinned build input).

Do not install or start the broker. Do not touch Docker, PostgreSQL, systemd, networks, credentials or host configuration.

Do not ask the Project Owner to run diagnostic commands. Perform all permitted repository work and verification autonomously.

After this single remediation pass, deliver:

- Exact files changed.
- Finding-to-fix mapping.
- Reproduction tests demonstrating each exploit is closed.
- Full test results with exact pass/fail/skip counts.
- Remaining risks, explicitly classified.
- Installation-readiness verdict.

No further broad review cycle is anticipated. A focused verification of the corrected security boundaries will follow.

### **Residual-risk decision**

I accept the documented DSN exposure risk only within the dedicated scratch PostgreSQL environment, with production unreachable and the existing database role restrictions preserved.&#x20;

adv_review_prompt3(1).md



**No installation authorization yet.** Once this remediation is complete, we do one focused security verification and move to installation. No endless review loops.