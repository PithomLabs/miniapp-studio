The plan is well structured, and I would preserve its overall sequence. However, **I would not authorize implementation unchanged: F4 still has a crash-recovery correctness flaw.**

The plan assumes that dropping a stale claim after recovery is safe because the pump died. That does not follow. An agent could already have emitted an event, the Gateway could have persisted it to the outbox (or delivered it), and the Gateway could then crash before marking the command complete. Dropping the claim and launching the command again can execute the same logical work twice. A durable claim plus a mutex does not resolve that ambiguity.&#x20;

prompt11_plan(2).md



### **Required amendments before execution**

**1. F4 — make ambiguous claims non-retryable**

Change the recovery rule. A stale claim must not automatically become `Drop`.

Use these semantics:

- Persist the claim before dispatch, including the message ID, payload digest, build/session/turn identity, and invocation ID.
- Drop a claim only when the supervisor can prove that process creation failed and no child was started.
- If a process may have started, or events may already have reached the outbox, recover the claim as **indeterminate**. Reap any identifiable orphan, reconcile persisted events, and do not relaunch that same logical command.
- Use the authorized F6 continue-from-failed path, as a new fenced turn, for deliberate recovery. Do not disguise that as a retry of the original command.
- Reject reuse of the same MessageID with a different payload digest.

Add the missing adversarial test: agent emits an event → event is persisted to the outbox → Gateway crashes before `Complete` → same command is delivered again. Prove that the agent is not launched a second time and the event is not duplicated. Also test crashes between claim persistence, process start, and recording the process identity.

The right guarantee here is **at-most-once dispatch with explicit recovery for ambiguous outcomes**, not an unqualified exactly-once claim.

**2. F7 — define the non-destructive failure cases**

The plan specifies what happens when the service is healthy, but not when it is unhealthy or `GetService` itself fails.&#x20;

prompt11_plan(2).md



Make the rule explicit:

- Absent service: create it.
- Existing healthy service: preserve it.
- Existing unhealthy service or service-inspection error: fail closed, report the condition, and do not delete or recreate it.
- Replacement remains an explicit upgrade operation.

Also prove that the per-Sprite lock coordinates genuinely concurrent provisioning invocations, not merely goroutines within one process.

**3. B1 — close the filesystem permission specification**

`SysProcAttr.Credential` is the appropriate fail-closed direction, but the plan must explicitly define the Gateway's effective identity and required privilege to launch a different UID. If the Sprite cannot provide that privilege, agent execution must refuse; no same-UID fallback. The live identity and key-read-denied proof remains a Part 2 gate.&#x20;

prompt11_plan(2).md



Also specify the question-file ownership and mode, the Gateway's ability to read that agent-owned file, and the required group membership or equivalent privilege. The directory's `2730` mode and the answer file's `0640` mode alone do not establish that the Gateway can read questions.

### **What I approve**

The remaining plan is directionally sound. In particular, F2/F3's `os.OpenRoot` approach, F5's process-outcome terminality, F6's explicit continuation and turn fencing, F8/F9's inspect/requeue and per-attempt signing, and the M1–M16 disposition can remain as planned.

The OpenCode CLI documentation does support `opencode run --session <id>` as the way to continue a specific session. That supports the F6 choice, but does not replace the required live proof that the pinned agent version actually resumes the intended session. [OpenCode CLI commands documentation](https://opencode.ai/docs/cli/commands?utm_source=chatgpt.com)【turn702582view0】

Keep the PostgreSQL 19 Docker-only testing requirement unchanged. Keep Part 2, all live Sprite/Fly/OAuth operations, Miniapp changes, and commits unauthorized.&#x20;

prompt11_plan(2).md

prompt11_plan(2).md



**Instruction:** Amend `plans/prompt11_plan.md` with these three clarifications, then proceed with the bounded correction pass. Do not reopen accepted architecture or start another broad review cycle. Stop after the correction report and evidence are complete, with Part 2 still unexecuted.