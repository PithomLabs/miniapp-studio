**Do it now, before Phase B.** Adversarial review belongs at phase boundaries, particularly when the next phase depends on the correctness of the previous phase's data model, security boundaries, and protocols.

I have reviewed `docs/gateway-phase-a-evidence.md` against the Phase A acceptance criteria and governing architecture.

## **My adversarial review criteria**

I evaluate six things:

1. **Architectural invariants:** Did implementation preserve ownership, tenant isolation, execution boundaries, and approved scope?
2. **Security:** Can an untrusted user, callback, or agent cross a trust boundary, expose secrets, or authorize its own actions?
3. **State integrity:** Can retries, concurrency, duplicate events, or out-of-order delivery corrupt business state?
4. **Recovery:** Does the application genuinely recover from process restarts, not merely reconstruct an object from the same database?
5. **Evidence:** Were the tests actually executed? Do they demonstrate the claimed invariant, including negative cases?
6. **Scope discipline:** Were F1, G-01, P5, and other locked decisions preserved?

## **Phase A adversarial findings**



One major finding

One evidence gap



**1. MAJOR — Conversation-body secret redaction**

The evidence explicitly says:

- Event payloads and question options are scrubbed on read.
- Flat conversation bodies remain verbatim.
- The reason given is transcript preservation.&#x20;

  gateway-phase-a-evidence.md



However, our Phase A security requirement says secrets must not appear in persisted agent messages.

An agent could inadvertently include a credential in its response. Autoescaping protects against HTML injection; it does not protect secrets.

This needs a focused correction or convincing implementation-level proof that secret-bearing content cannot reach those message bodies.

**2. MINOR — Restart recovery proof**

The recovery test starts a fresh server, but the old process remains running to avoid double-shutdown hazards.&#x20;

gateway-phase-a-evidence.md



That demonstrates fresh-instance reconstruction, but does not conclusively demonstrate a complete Studio process restart.

I want one clean restart test with the old instance properly shut down and its resources closed before the new instance starts.

**What I accept without further review:**

The unified conversation stream, transactional message submission, callback deduplication, HMAC verification, PostgreSQL coverage, and transition testing are all supported by the reported evidence. Gateway-side sequence ordering and live OpenCode proofs correctly remain Phase B/C work.&#x20;

gateway-phase-a-evidence.md

gateway-phase-a-evidence.md



## **One bounded closure task**

I authorize this focused correction. No general re-review.

# PHASE A — ADVERSARIAL CLOSURE

Act as the security and reliability engineer reporting to the Miniapp Studio Senior Architect.

Address only the two findings identified in the Phase A adversarial review.

### Finding 1: Conversation-body secret handling

Investigate the complete ingestion, persistence, retrieval, audit, and SSE paths for user and agent messages.

Determine exactly where secret-shaped content can enter persistent conversation bodies.

The Phase A requirement is that secrets must not appear in persisted agent messages, logs, SSE responses, or rendered conversation history.

Implement a durable correction at the appropriate ownership boundary if the requirement is not satisfied.

Preserve useful conversation content and record redaction provenance where appropriate.

Add regression tests covering secret-shaped canaries in user messages, agent messages, question text, and event-derived conversation content. Verify both database persistence and every read/delivery path.

Do not silently weaken the requirement or retain an unsanitized copy in an ordinary accessible table.

### Finding 2: Genuine Studio restart recovery

Strengthen `TestPhaseARestartRecovery`.

Shut down the original server instance correctly, close its resources exactly once, and initialize a fresh instance against the same persistent database.

Demonstrate that:

- An outstanding question survives.
- Its answer remains applicable.
- Conversation history is reconstructed.
- SSE replay works from the previous cursor.
- Duplicate callback delivery does not reapply effects.
- No process-local memory is required.

Add regression coverage for the actual lifecycle.

### Boundaries

Do not:

- Change the Gateway v1 protocol.
- Redesign the conversation schema unless the root cause requires it.
- Provision Sprites or install OpenCode.
- Resolve F1 or G-01.
- Expand P5 or implement Lead-to-Booking.
- Introduce new infrastructure or a workflow engine.
- Commit or push.

Run the relevant security and recovery tests, followed by the full Phase A gates.

Update `docs/gateway-phase-a-evidence.md` with the findings, corrections, and actual executed results.

If either issue cannot be resolved without a foundational architectural decision, stop and present concrete evidence rather than introducing a workaround.

## **What happens next**

Once those two findings are closed and the relevant gates pass, **I will consider Phase A accepted and move directly to Phase B — Sprite Agent Gateway and OpenCode provisioning. No additional general adversarial review cycle or another GO request will be necessary.**

We are progressing, not reopening architecture decisions.