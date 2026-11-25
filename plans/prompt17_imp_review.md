**My verdict: the agent correctly stopped. I approve the diagnosis and authorize the correction.**

This is a genuine runbook defect, not pin drift or an infrastructure failure. The failure is caused by a whitespace-sensitive `grep` expression that cannot match the pretty-printed GitHub API response. The independent check found the pinned OpenCode v1.19.6 release intact.&#x20;

prompt17_imp.md



There is no structural or foundational blocker. The §3 diagnostic checks also passed, but they do not override the §2 STOP condition.&#x20;

prompt17_imp.md



### **My decision**

1. Authorize the runbook correction.
2. Require a whitespace-tolerant match, rather than merely inserting one literal space.
3. Require an explicit failure when no v1 release tag is found.
4. Preserve the v1.19.6 pin and all existing safety restrictions.
5. **The original explicit Part 2 GO remains in force.** No new authorization round is necessary for this bounded correction.

### **Instruction to the agent**

# Authorization — Correct §2 Release-Tag Verification and Resume Part 2

**OWNER AUTHORIZATION: APPROVED**

The owner authorizes correction of the defective OpenCode release-tag verification command in `docs/gateway-phase-b-live-runbook.md`.

The previous explicit Phase B Part 2 GO remains active. This is not a new planning exercise and does not require another GO request.

## 1. Correct the root cause

The existing grep assumes no whitespace between `tag_name` and its colon. GitHub returns pretty-printed JSON containing whitespace.

Replace the fragile extraction with whitespace-tolerant matching.

Recommended implementation:

```
release_json="$(curl -fsSL 'https://api.github.com/repos/anomalyco/opencode/releases?per_page=10')" || {
  echo "STOP: failed to retrieve OpenCode releases"
  exit 1
}

v1_tags="$(printf '%s\n' "$release_json" \
  | grep -oE '"tag_name"[[:space:]]*:[[:space:]]*"v1[^"]*"' \
  | sed -E 's/.*"(v1[^"]*)".*/\1/' \
  | sort -u)"

if [ -z "$v1_tags" ]; then
  echo "STOP: no v1 release tags found"
  exit 1
fi

printf '%s\n' "$v1_tags"

if [ "$v1_tags" != "v1.19.6" ]; then
  echo "STOP: unexpected v1 release set; explicit pin review required"
  exit 1
fi
```

Do not change the pinned OpenCode version or its SHA-256.

## 2. Verify the correction

Execute the corrected public release query and verify:

- The command successfully finds the v1 release.
- The output is exactly `v1.19.6`.
- Empty results and unexpected versions fail explicitly.
- The command does not silently pass on malformed or absent matching data.

Record the actual output and exit status.

## 3. Resume the previously authorized Part 2 execution

After correcting the runbook:

1. Re-run §2 completely, including repository-state and all pin checks.
2. Re-run §3 pre-live gates.
3. If both sections pass, continue with §4 and the remaining approved live runbook sections in order.
4. Do not repeat already completed checks unnecessarily, but retain fresh evidence for the current execution.
5. Do not ask for another GO confirmation.

## 4. Restrictions remain binding

- N2: no live Gateway upgrade, including `--force`.
- N8: staging permission-window checks before re-provisioning.
- N5: record pre-sleep PIDs and verify absence after cold recovery.
- Preserve all credential and identity-separation rules.
- Stop on any new failed gate or unexpected condition.
- Do not silently substitute pins, models, organization, or deployment topology.
- No Phase C/D/E implementation.
- No unapproved code changes.
- No commits or pushes.

## 5. Completion report

Report the §2 correction, actual verification output, fresh §3 results, and subsequent runbook execution evidence.

If another STOP condition is encountered, stop and report its root cause. Do not bypass it.

**FINAL INSTRUCTION: The original Part 2 GO remains active. Correct §2, revalidate §2 and §3, and proceed with the authorized live verification.**

**Architectural status:** approved to resume Phase B Part 2. The previous STOP was appropriate; this correction closes that procedural blocker. The agent should now execute the approved live proof—not produce another plan.