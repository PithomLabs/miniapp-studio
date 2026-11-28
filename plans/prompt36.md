# Prompt 36 — Remediate RPA F0 Security Findings (Increments 1–3)

## Mission

Remediate the six substantiated HIGH/MEDIUM findings from the retrospective adversarial review:

`plans/adv_review_prompt35.md`

Scope:
- Increment 1: Secrets management.
- Increment 2: Credential broker.
- Increment 3: Tenant invitations and capability refresh.
- Cross-increment security and database boundaries.

The current review verdict is NOT APPROVED. Do not close F0 until the HIGH/MEDIUM findings are fixed, regression-tested, independently reviewed, and the required live Neon verification is complete.

Use the actual working tree as authoritative. Preserve all existing work, including the review report and prompt files. Do not reset, overwrite, or discard user changes.

## 1. First: investigate and reproduce

Read the complete adversarial report, relevant source, migrations, tests, and evidence documents.

Reproduce each finding against the current code before fixing it. Confirm whether the reported file locations and behavior still apply.

The review identifies:

| Finding | Severity | Required outcome |
|---|---|---|
| F35-01 | HIGH | Offboarded operators cannot be reactivated by stale invitations |
| F35-02 | MEDIUM | No nil-cancel panic on any F0 CLI error path |
| F35-03 | MEDIUM | DSNs and passwords cannot leak through CLI errors or logs |
| F35-04 | MEDIUM | OAuth refresh credentials cannot be exfiltrated through redirects |
| F35-05 | MEDIUM | Credential revocation reliably prevents minting and consuming leases |
| F35-06 | MEDIUM | Permission audit and file read operate on the same securely opened file |

Do not assume that fixing the reported line alone fixes the underlying problem. Trace affected call paths and shared helpers.

## 2. F35-01 — HIGH: invitation/offboarding authorization bypass

An outstanding invitation currently allows an offboarded operator to be reactivated because `invitation_accept` unconditionally sets `active=true`.

This must be corrected first.

Requirements:

- Acceptance must refuse an invitation when its intended operator/binding already exists in an inactive state.
- Do not reactivate inactive operators through invitation acceptance.
- Lock or otherwise serialize the operator-state check with acceptance so concurrent offboarding cannot be bypassed.
- Preserve atomicity across invitation consumption, operator provisioning, and session creation.
- An invalid or unauthorized acceptance must leave no consumed invitation, newly activated operator, or session.
- Preserve uniform external failure responses and token confidentiality.
- Investigate the existing offboarding path, not merely `invitation_accept`.

Important lifecycle requirement: prevent an invitation issued before offboarding from becoming usable again merely because the owner later reactivates the operator. Revoke outstanding invitations on offboarding or implement an equivalent generation/epoch safeguard. A newly authorized invitation must be explicitly issued after reactivation.

Add database regression and concurrency tests covering:

1. Active operator accepts a valid invitation.
2. Inactive operator cannot accept an outstanding invitation.
3. Offboarding wins a race against invitation acceptance.
4. Acceptance wins a race only when it is validly serialized before offboarding.
5. Reactivation does not revive an old invitation.
6. Owner reactivation followed by a fresh invitation succeeds.
7. Every failed acceptance rolls back all relevant state.

Because migration `000029` may already be applied, **do not rewrite an applied migration**. If SQL changes are needed, create a new corrective migration, normally `000031`, with a reversible down migration. Preserve migration history.

## 3. F35-02 — MEDIUM: nil-cancel panic

The review reproduced a panic in `brokerPool` when policy, production-guard, DSN loading, or DSN parsing fails.

Requirements:

- Fix the root cause in the shared pool/configuration lifecycle.
- Inspect all nine F0 database CLI verbs, not just the first reproduced command.
- Ensure cleanup functions are called only when initialized.
- Ensure guard refusals and malformed configuration return their intended errors, not nil-pointer panics.
- Ensure token-file reservations and other temporary resources are cleaned up after errors.
- Preserve fail-closed production guards.

Add regression tests for each distinct failure stage, including malformed DSN, policy load failure, and production-guard refusal. Assert that no panic occurs and the correct error is preserved.

## 4. F35-03 — MEDIUM: DSN/password leakage

Requirements:

- Never expose a raw DSN, password, URL userinfo, or encoded credential through parse errors, CLI output, logs, or wrapped errors.
- Fix the URL-encoded password bypass in `redactDSN`.
- Prefer structural credential-span redaction or avoid returning raw parser errors altogether.
- Audit both broker CLI and migration/provisioning error paths.
- Preserve useful, non-sensitive diagnostic context.

Add canary tests for malformed DSNs, percent-encoded usernames/passwords, reserved characters, and nested wrapped errors. Assert that neither decoded nor encoded credentials appear in captured output.

## 5. F35-04 — MEDIUM: OAuth redirect credential exfiltration

Requirements:

- The OAuth refresh exchange must not forward refresh tokens or client secrets to redirected endpoints.
- Reject redirects explicitly; do not rely on Go's default redirect behavior.
- Treat unexpected 3xx responses as a controlled exchange failure.
- Preserve TLS verification, request timeouts, response-size limits, and safe error reporting.
- Test 301, 302, 303, 307, and 308 responses.
- Verify that the redirect destination never receives the credential-bearing request body.

This fix is mandatory before any production connector wires the broker's HTTPExchanger.

## 6. F35-05 — MEDIUM: credential revocation races

Define and implement a clear, linearizable revocation contract.

Requirements:

- If revocation commits before Mint finalizes, Mint must not issue a usable lease.
- If revocation commits before Consume, Consume must reject the lease.
- Existing unconsumed leases must not remain usable after credential revocation.
- A Mint already in flight must re-check credential state after the network exchange and before persisting a lease.
- Ensure the check is bound to the exact credential version used for the exchange; a concurrent re-enrollment must not cause stale operations to affect the new credential.
- Keep database locks and transactions appropriately bounded; do not hold database locks across provider network calls.
- Revoke, Mint, and Consume must behave correctly under concurrency.

Add deterministic failure-injection and concurrency tests for both revocation orderings. Do not accept TTL-bounded revocation as an undocumented substitute for immediate revocation.

Also inspect F35-07, where a stale `invalid_grant` response may revoke a newly re-enrolled credential. Correct it if it shares the same versioning/concurrency defect.

## 7. F35-06 — MEDIUM: secret-file permission/read TOCTOU

The current Stat-then-ReadFile design can audit one file and read another, and follows symlinks.

Requirements:

- Create a shared secure file-opening helper.
- Open without following the final-component symlink (`O_NOFOLLOW` or the appropriate platform equivalent).
- Validate permissions and file type using `fstat` on the opened descriptor.
- Read bytes from that same descriptor; never re-open the path after checking it.
- Preserve the existing owner-only permission requirements.
- Fail closed on symlinks, non-regular files, unexpected ownership/modes where applicable, and I/O errors.
- Apply the helper consistently to all secret-bearing file paths: `_FILE` sources, KEK, refresh token, invitation token, DSN, operator/key/grant files, and other relevant credential paths.
- Consider parent-directory/path assumptions explicitly. Do not claim protection against races the implementation does not actually prevent.

Add deterministic symlink rejection and file-replacement race tests. The permission audit must prove that the bytes read are the bytes audited.

## 8. Other LOW findings

Review F35-07 through F35-17. Triage each by actual risk and relationship to the six primary fixes.

- Fix low-severity issues that are directly coupled to these remediations or are cheap correctness/security defects.
- Record the remaining items with clear rationale and a follow-up recommendation.
- Do not expand this assignment into unrelated refactoring.
- Reconcile the report's count: its summary says twelve LOW findings, but the listed IDs F35-07 through F35-17 enumerate eleven. Correct the count or identify the missing finding; do not invent one.

At minimum, explicitly disposition:
- Stale `invalid_grant` auto-revoke after re-enrollment.
- Owner-DSN handling outside the secrets contract.
- Unused/narrow redaction helper.
- Unused `STUDIO_BROKER_KEK_FILE`.
- NaN/Inf invitation TTL behavior.
- Refresh-token whitespace handling.
- Provider-controlled OAuth error strings.
- Unnecessary DELETE grants.
- Documentation drift.
- Concurrent invitation loser error assertions.
- Capability scan coverage gaps.

## 9. Reconcile existing live Neon evidence

The review's environment skipped all F0 database tests because it lacked Neon access. However, earlier owner-side verification was supplied in this conversation:

- Migration v28 verified; targeted `TestBroker|TestPrivilegeParity` run passed in `store` (197.469s).
- Migration v30 verified; targeted `TestInvite|TestCapability|TestPrivilegeParity` run passed in `store` (201.027s).

Do not misrepresent these results as tests skipped everywhere. Reconcile the earlier owner-side evidence against the exact test names and current test inventory. Distinguish:
- Tests that passed in the owner's live Neon environment.
- Tests skipped in the review/agent environment.
- Tests newly added by this remediation and not yet run against Neon.

Do not claim that the old live results verify new code.

## 10. Verification and adversarial re-review

After remediation:

- `gofmt -l .`
- `go build ./...`
- `go vet ./...`
- `go test ./... -count=1`
- Relevant `go test -race` suites for secrets, broker, invitations, capability, store, and dbenv.
- All new failure-injection, token-canary, redirect, race, and offboarding regression tests.
- Migration up/down source review, RLS, grant mirrors, privilege parity, and migration ordering.

Run a fresh, independent adversarial pass against the remediated code. Do not merely mark the original findings fixed based on implementation claims. Re-run the attack probes and regression tests.

The review must explicitly verify:
- No remaining invitation/offboarding bypass.
- No secret leakage through F0 CLI errors.
- No redirects forwarding OAuth credentials.
- No usable leases after revocation wins.
- No Mint result after revocation wins during exchange.
- No symlink or check/read race returning unaudited secret bytes.
- No panic on guard or policy refusal.

## 11. Live-operation boundaries

You are authorized to modify local source, migrations, tests, and documentation in the assigned scope.

You are **not** authorized to:
- Apply migrations to live Neon.
- Modify Fly resources.
- Enroll real provider credentials.
- Perform OAuth enrollment against real accounts.
- Mutate staging or production.
- Commit or push changes.

Prepare exact owner-side commands for the corrected live database gate. Inspect the actual Taskfile and test harness before writing commands; do not guess.

The expected sequence will include:
1. `task dev:init` to apply the corrective migration to `studio_dev_ui`, once the owner runs it.
2. A focused live Neon test covering broker, invitations, capability, privilege parity, and the new offboarding regression.
3. Any required live race/concurrency verification supported by the existing test harness.

Do not execute these live operations yourself.

## 12. Deliverables and verdict

Preserve `plans/adv_review_prompt35.md` as the original review record.

Create:

`docs/rpa-f0-remediation-evidence.md`

Include:
- Worktree/commit identity.
- Each finding's original severity and final disposition.
- Root cause and exact correction.
- New regression tests and reproduction results.
- Migration details, including any new version and down migration.
- Test counts, skipped counts, race results, and environmental limitations.
- Reconciliation of prior owner-side Neon v28/v30 evidence.
- Fresh adversarial review findings.
- Exact owner-side live verification commands.
- Remaining LOW findings and their disposition.

Final verdict must be one of:
- **NOT APPROVED** — any HIGH/MEDIUM defect remains or material verification is incomplete.
- **APPROVED WITH LIMITATIONS** — all HIGH/MEDIUM defects are fixed and reviewed, but explicitly stated non-blocking verification remains.
- **APPROVED** — all HIGH/MEDIUM defects are fixed, regression-tested, independently re-reviewed, and the required live Neon gates pass.

Do not close F0 merely because local tests are green. Do not begin Increment 4 implementation until F35-01 is fixed and live Neon verification is complete. Increment 4 design work may continue independently.

Do not commit or push.