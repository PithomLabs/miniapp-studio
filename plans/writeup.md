The central architectural decision is to make **Miniapp Studio the permanent control plane and source of truth, while Fly.io Sprites provide isolated, persistent execution environments for coding agents.** The coding agent is a worker inside this architecture, not the application itself and not the user's interface.

The important distinction in the writeup is that the Studio-side conversation infrastructure has been implemented and exercised, but Phase A is still awaiting its authorized security and recovery closure. The actual OpenCode-in-Sprite integration remains pending.

# Miniapp Studio and the Coding Agent in Fly.io Sprites

## Architecture, engineering challenges, and the road to an end-to-end autonomous development workflow

**Technical design note — November 25, 2026**

**Status:** Agent Gateway architecture approved. Phase A implementation exists but is not yet accepted. The bounded Phase A closure has been authorized. Phases B–E remain pending.

---

## 1. The fundamental design problem

Miniapp Studio is being designed around a fairly ambitious idea: a business owner should be able to describe a business process in ordinary language, have an AI coding agent implement the necessary software, inspect the work, make decisions, and eventually deploy the approved result — without having to operate a terminal, manage a coding-agent session, or understand the infrastructure underneath it.

That sounds like a coding-agent integration, but the engineering problem is considerably larger.

A coding agent is not a reliable application server. It can stop unexpectedly, ask questions, produce incomplete work, generate incorrect output, lose connectivity, or operate on untrusted repositories. Its process may disappear when its environment sleeps. Its output must not be trusted as an authorization decision. And its internal conversation history cannot become the permanent record of a customer's business process.

The architecture therefore needs to solve several distinct problems:

1. Maintain a permanent conversation between the business user and Miniapp Studio.
2. Delegate implementation work to a coding agent running in an isolated environment.
3. Translate the agent's native behavior into a stable Studio-owned protocol.
4. Preserve state through process termination, Sprite sleep, gateway restart, and Studio restart.
5. Allow genuine human clarification and decision-making without confusing a question with authorization.
6. Verify generated artifacts before presenting them for review.
7. Promote only explicitly approved artifacts into a separate runtime environment.

The guiding principle is:

> The coding agent performs the work. Miniapp Studio owns the work, its history, its permissions, and its outcome.

This separation is the foundation of the entire design.

## 2. The architecture: three distinct environments

The design separates the system into three principal environments.

### 2.1 Miniapp Studio — the control plane

Miniapp Studio is the permanent interface between the business owner and the automation platform.

It owns:

- Business-user conversations and messages.
- Build assignments and build state.
- Agent session references.
- Questions and user answers.
- Proposals and authorization requests.
- Commands sent to the gateway.
- Events returned by the gateway.
- Artifact metadata and verification results.
- Human approvals and deployment records.
- The user-facing conversation and review interface.

Studio is backed by tenant-scoped PostgreSQL and uses the Miniapp framework's `runtime.Service` for tool execution, policy enforcement, invocation, and audit boundaries.

The important point is that Studio does not depend on an agent process remaining alive. A conversation must still make sense if every Sprite is stopped and every agent session is temporarily unavailable.

Studio is also the component that decides whether an agent's answer is valid, whether a question is still current, whether an artifact can be reviewed, and whether an action has been authorized.

### 2.2 Builder Sprite — the execution environment

The Builder Sprite is a persistent Fly.io Linux environment assigned to an application or build workspace.

It owns:

- The checked-out source repository.
- Development dependencies.
- The coding agent's session files.
- Temporary build and test files.
- Generated artifacts.
- The Agent Gateway service.

The coding agent runs inside this environment. It can inspect and modify the assigned checkout, execute development commands, run tests, and produce artifacts.

The Builder Sprite is not the permanent business record. Its filesystem is an execution resource, not the authoritative database.

The design deliberately makes the Builder Sprite separate from the production application environment.

### 2.3 Runtime Sprite — the application environment

The Runtime Sprite is introduced in Phase E.

It runs the approved application, not the coding agent.

It receives only the approved deployment artifact and the production integration capabilities explicitly assigned to that application.

It must not contain the coding agent, the Agent Gateway command surface, or the Builder's unrestricted development environment.

The separation is important: an agent that can modify source code and execute arbitrary development commands should not automatically inherit the credentials or access privileges of a production application.

### 2.4 Architectural overview

```
flowchart TB
    U[Business User]
    B[Browser]

    S[Miniapp Studio]
    DB[(Studio PostgreSQL)]
    M[Miniapp runtime.Service]

    G[Agent Gateway]
    A[OpenCode Coding Agent]
    FS[(Builder Sprite filesystem)]

    R[Application Runtime Sprite]
    C[Fly Connectors]

    U --> B
    B <-->|HTTP + SSE| S
    S <--> DB
    S <--> M

    S -->|Signed HTTPS commands| G
    G -->|Signed HTTPS callbacks| S

    G <--> A
    A <--> FS

    S -->|Approved digest only| R
    C -.->|Approved production bindings| R
```

The architecture has two different conversations:

- **Business conversation:** User ↔ Miniapp Studio. This is permanent and authoritative.
- **Execution conversation:** Miniapp Studio ↔ coding agent. This is an internal protocol used to coordinate implementation.

The user never needs to connect to a Sprite terminal or manage an agent's native CLI session. The user sees one continuous conversation inside Studio.

---

## 3. Why the Agent Gateway exists

The Agent Gateway is the central integration component between Studio and the coding agent.

It runs as a service inside the Builder Sprite and acts as the agent's supervisor and protocol adapter.

Without a gateway, Studio would need to understand the native behavior of every coding agent it supports. It would have to manage process creation, sessions, questions, cancellation, output interpretation, and recovery itself.

That would tightly couple the product to a particular coding-agent implementation.

Instead, the gateway introduces a stable boundary.

Studio communicates using the Miniapp Gateway protocol. The adapter translates that protocol into the native interface of the coding agent.

For example:

| Studio operation | Gateway responsibility                                   |
| ---------------- | -------------------------------------------------------- |
| Assign build     | Start the agent in the assigned checkout                 |
| Continue build   | Resume the existing agent session with the user's answer |
| Cancel build     | Stop the supervised process and acknowledge cancellation |
| Check status     | Report gateway, session, and process state               |
| Fetch artifact   | Retrieve specified files and verify their digests        |
| Agent progress   | Translate native progress into a Studio event            |
| Agent question   | Convert a native yield into a structured question        |
| Agent completion | Report completion, subject to artifact verification      |

The gateway is designed to be coding-agent-independent, but the first implementation is specifically for OpenCode. A second adapter is not part of the current scope.

### 3.1 Why the gateway is a Sprite service

Fly.io Sprites have an important lifecycle characteristic:

- Their filesystem is persistent.
- Services survive Sprite sleep and wake.
- Exec and TTY processes stop when the Sprite sleeps.

This has a major architectural consequence.

The gateway must run as a **Sprite service**, not as an ordinary terminal or exec process.

The gateway service can remain available across sleep and wake. However, the coding-agent process cannot be assumed to survive.

After a wake, the gateway must be able to start a fresh supervised process and reopen the agent's persisted session files.

A design that relies on keeping an OpenCode process alive indefinitely would be incorrect.

### 3.2 Communication is asynchronous and recoverable

The intended communication paths are:

| Direction                | Channel                       |
| ------------------------ | ----------------------------- |
| Browser → Studio         | Authenticated HTTP            |
| Studio → Browser         | Server-Sent Events (SSE)      |
| Studio → Builder Gateway | Authenticated HTTPS commands  |
| Builder Gateway → Studio | Authenticated HTTPS callbacks |

There is no permanent Studio-to-Sprite socket.

Studio sends commands when work needs to happen. Those commands can wake the Builder Sprite. The gateway executes them and sends results back through callbacks.

This avoids making the availability of a long-lived network connection a requirement for the correctness of a build.

---

## 4. The coding-agent decision: OpenCode

The initial coding agent is OpenCode, using the official Fly Sprites OpenCode plugin.

The architectural choice is deliberate.

OpenCode is treated as an implementation behind the adapter contract, not as the owner of the business conversation or the application's control plane.

The initial setup uses:

- OpenCode 1.x, with v1.19.6 as the planned pinned version.
- The official `@flydotio/sprites-opencode-plugin` checkout.
- The plugin pinned to the reviewed commit `f63a85a7da9fa2f42b21877ab58b35d18ccaa682`.
- The documented JSON event output from `opencode run --format json`.
- Native session references for continuation.
- Exact permission rules rather than blanket automatic approval.

The plugin's package availability was not dependable at the time of the plan, so the reviewed local Git checkout is used instead of assuming that a published npm package can be installed.

### 4.1 The important topology distinction

There is a subtle but important difference between how the official plugin is commonly described and how it is intended to operate here.

In this architecture:

1. OpenCode itself runs inside the Builder Sprite.
2. OpenCode accesses the assigned source repository locally.
3. The Sprites plugin provides authenticated Fly MCP capabilities.
4. Any remote Sprite operations through MCP remain subject to the assigned grant and permission scope.

The plugin must not mislead OpenCode into believing that it is running outside the Builder Sprite and should operate on that same Builder as a remote target.

The amendment therefore disables the plugin's injected guidance with `guidance: false` and provides Builder-specific instructions.

The MCP integration remains enabled. Disabling incorrect guidance must not disable the actual Fly tools.

This topology has not yet received its live acceptance proof.

### 4.2 How the adapter is intended to operate

The first adapter uses the documented OpenCode CLI surface rather than relying on an experimental HTTP API.

The main operations are:

```
opencode run --dir <checkout> --format json "<assignment>"
```

For continuation:

```
opencode run --session <sessionRef> --format json "<answer>"
```

For session inspection:

```
opencode session list --format json
```

The JSON output is intended to provide structured events that the adapter can translate into the gateway vocabulary.

The adapter must not scrape terminal screens, infer state from conversational prose, or fabricate event meanings.

The initial approach deliberately avoids `opencode serve` plus `--attach`. That configuration may become a performance optimization if cold startup proves expensive, but it is not required for correctness.

### 4.3 Why OpenCode is a difficult integration

The CLI is not the entire problem.

The critical question is whether OpenCode's noninteractive `run` command can support the Studio question-and-answer lifecycle.

OpenCode has a question tool, but its behavior when invoked inside a noninteractive run needs actual proof.

There are three possible outcomes:

- The run emits a structured question event and exits or can be terminated safely.
- The process blocks in a way the supervisor can manage.
- The process errors or behaves in a manner that cannot provide a reliable continuation contract.

The plan defines a primary and fallback approach.

**Primary approach:** capture the native question event, emit `agent.question`, park the Studio turn, and resume the same OpenCode session with the user's answer.

The proof must demonstrate that OpenCode actually consumes the answer as the resolution of the pending question. Merely continuing the conversation is not sufficient.

**Fallback approach:** use a deterministic JSON file handoff in the shared `.studio/inbox/` directory. The agent writes a question file and exits; the gateway writes the answer file; the resumed agent reads that structured answer before continuing.

This is an adapter-specific, testable protocol. It is not an invitation to parse arbitrary prose or invent CLI behavior.

If neither path works, the correct response is to stop and return the unresolved issue for an owner decision.

---

## 5. The security model: treat the Builder as untrusted

The Builder Sprite runs repositories and commands that may be controlled or generated by the coding agent.

Consequently, the gateway must be designed under the assumption that the agent or repository could be hostile.

The security model cannot depend on instructions asking an agent to behave safely.

### 5.1 Separate process identities

An important correction to the original gateway security design is that merely storing an HMAC key in a `0600` file is insufficient when the gateway and agent run as the same Unix user.

A process running as that user may still read the file.

The revised design separates the identities:

| Process       | Identity       |
| ------------- | -------------- |
| Agent Gateway | `studio-gw`    |
| Coding agent  | `studio-agent` |

The gateway owns its HMAC signing keys. The agent must receive an operating-system `EACCES` error when attempting to read them.

The shared artifact/question handoff uses a separate, deliberately scoped `studio-io` group and directory.

The key boundary is enforced by Unix permissions, not by hiding a filename from the agent.

If the Sprite cannot support this separation, provisioning must fail closed rather than silently launching an agent with access to gateway credentials.

### 5.2 Constrain the Fly MCP grant

The OpenCode plugin must not receive unrestricted access to every Sprite in the organization.

The grant is scoped to the full deterministic name of the assigned Builder Sprite, with:

- `max_sprites_total = 0`, because Studio provisions the Builder.
- No ability to create or destroy Sprites.
- No ability to change network policy.
- No ability to restore checkpoints outside Studio's recorded operation.
- Positive and negative tests proving that other Builder and Runtime Sprites are inaccessible.

The grant itself is part of the security boundary. Prompt instructions alone cannot provide this guarantee.

### 5.3 Keep production credentials out of the Builder

Fly Connectors provide a way to route provider API calls without exposing the underlying provider credentials directly to Sprite processes.

However, a connector is not a universal sandbox. An authorized process with access to a connector may still be able to use the capability it provides.

Therefore:

- Builder Sprites do not receive production connector bindings.
- Development-labeled connections may be provided only where explicitly approved.
- Runtime Sprites receive only the production bindings required by the approved application.
- Provider tokens must not appear in prompts, repositories, logs, or artifacts.

### 5.4 Authentication between Studio and Gateway

Studio and the Gateway use a purpose-built HMAC-SHA256 protocol with per-build keys.

The intent is to provide:

- Authenticated commands from Studio.
- Authenticated events from the Builder.
- Timestamp freshness checks.
- Message deduplication.
- Tenant/build binding.
- Replay resistance.
- Strict rejection of forged or tampered messages.

The Gateway never gets to authorize itself. It can request clarification, make a proposal, or request authorization, but the authoritative decision remains in Studio.

---

## 6. The protocol and durable state model

A central design requirement is that the Studio and the gateway agree on a stable vocabulary independent of OpenCode's internal event schema.

The protocol envelope contains:

```
{
  "protocol": "miniapp-gateway/1",
  "message_id": "uuid",
  "build_id": "build-reference",
  "turn_id": "turn-reference",
  "sent_at": "RFC3339 timestamp",
  "signature": "HMAC signature",
  "kind": "command or event",
  "name": "event or command name",
  "payload": {}
}
```

The commands are:

- `assign`
- `continue`
- `cancel`
- `status`
- `fetch_artifact`

The event vocabulary includes:

- `agent.started`
- `agent.progress`
- `agent.message`
- `agent.question`
- `agent.proposal`
- `agent.authorization_request`
- `agent.artifact_ready`
- `agent.completed`
- `agent.failed`
- `agent.resumed`
- `agent.cancelled_ack`

The vocabulary is designed to distinguish events that matter to the business process. For example, an agent asking for authorization is not equivalent to authorization being granted.

### 6.1 At-least-once delivery, idempotent effects

The system assumes that messages can be delivered more than once.

It therefore does not rely on exactly-once network delivery.

Instead, it uses durable identifiers and idempotent processing.

Studio deduplicates gateway events using the tenant/build/event identity. The gateway maintains command processing state and a durable callback outbox.

The expected behavior is:

- A duplicate command does not start a second logical build.
- A duplicate callback does not apply the same state transition twice.
- A lost callback is retried.
- Delayed events are retained and interpreted through state guards.
- A restart does not erase the authoritative conversation history.

The Studio database is the source of truth. Gateway state is execution state that can be reconstructed from persisted session files and Studio records.

### 6.2 Question, proposal, and authorization are different things

There are three different kinds of interaction:

| Type                  | Meaning                                                         | What the user's response does             |
| --------------------- | --------------------------------------------------------------- | ----------------------------------------- |
| Clarification         | The agent needs missing business information                    | Supplies information                      |
| Proposal              | The agent recommends a choice                                   | Records the selected decision             |
| Authorization request | The agent needs permission for a described consequential action | Creates a separate authorization decision |

An agent cannot grant itself permission by emitting an event or writing something into its own files.

Studio must validate the action and record the decision independently.

This distinction matters because the system is intended to automate business processes, not merely generate code.

---

## 7. What has actually been implemented: Phase A

Phase A was designed to establish the Studio-side conversation and protocol foundation without requiring a live Sprite.

The implementation created the first version of the durable conversation model, new build and question tools, the callback receiver, SSE replay, and the browser-facing conversation interface.

### 7.1 Delivered components

The original implementation report records:

- Migrations `000011` through `000015`.
- Conversations and messages.
- Builds, turns, and questions.
- Commands and events.
- The unified conversation stream.
- Artifact and approval metadata.
- New build, conversation, ingestion, and question tools.
- The v1 Gateway envelope with HMAC signing and verification.
- The authenticated callback endpoint.
- The SSE event endpoint.
- Conversation pages, question presentation, and answer conflict handling.

The new tools increased the registered Studio tool count from thirteen to twenty-three.

The implementation also established several important behaviors:

- `message.send` persists the message and ensures an open build atomically.
- Callback event deduplication uses a stable event identity rather than relying on HTTP retry identity.
- A superseded question cannot silently accept a stale answer.
- Authorization answers must be explicit `approved` or `rejected` decisions.
- Conversation history is append-only.
- Unknown event types are retained rather than silently discarded.
- The browser can reconnect and replay events using a cursor.

### 7.2 A significant sequencing decision

The original design considered separate sequence numbers for different parts of the conversation.

The implementation settled on a unified `conversation_stream` sequence for messages and events.

That sequence is allocated inside a transaction under a per-conversation advisory lock.

The honest guarantee is **monotonic ordering, not gaplessness**. An aborted transaction can create a gap, so replay uses greater-than cursor semantics, not an assumption that every integer must exist.

Per-build gateway event ordering remains a distinct Phase B responsibility.

### 7.3 What Phase A proved — and what it did not

The initial evidence report recorded successful formatting, vetting, builds, tests, race tests, migration round trips, and a scripted fifteen-step functional path.

The test path covered a simulated conversation from message submission through a question, answer, continuation, completion, and artifact metadata.

However, the implementation did not use a live Builder Sprite or real OpenCode session.

That distinction is fundamental:

- Phase A demonstrated that Studio could receive, persist, validate, render, and replay the conversation protocol.
- It did not demonstrate that OpenCode could be installed in a real Sprite, yield a question, consume an answer, resume after sleep, or deliver real build artifacts.

The subsequent adversarial review also found three acceptance issues. Consequently, the original statement that Phase A was fully satisfied is superseded by the later review.

---

## 8. Current challenges discovered so far

The most useful way to understand the project's current state is to separate defects found in the completed Studio-side work from the unresolved problems in the live agent integration.

### 8.1 Phase A: secret handling was not safe enough

The original Phase A implementation persisted agent-generated event payloads and question bodies verbatim.

Some read paths scrubbed secret-shaped values, but the scrubbing was incomplete.

The adversarial review demonstrated that an embedded `sk-live-` canary in a question body could be persisted and returned through:

- `build.status`
- `conversation.get`
- `conversation.stream`

The original tests did not adequately exercise question-body content or embedded secrets.

The closure plan moves sanitization to the write boundary: `build.ingest_event` must sanitize the payload before event persistence and question derivation.

Read-path sanitization remains as defense-in-depth.

The planned pattern set covers common OpenAI-style keys, AWS access-key identifiers, Slack token forms, and GitHub token prefixes. The implementation must preserve surrounding text and avoid needlessly redacting ordinary prose.

There is also an important residual: the Miniapp framework persists invocation inputs and results in `app.invocations` by design. Studio does not modify that framework behavior as part of this closure. The evidence must disclose that residual rather than claim that every unsanitized copy has disappeared from the entire system.

### 8.2 Phase A: the RLS configuration did not enforce what the report claimed

The review found that the ten new Phase A tables had:

- `FORCE ROW LEVEL SECURITY`
- But not `ENABLE ROW LEVEL SECURITY`

FORCE without ENABLE leaves the policy mechanism inactive.

The review verified that the current application queries have explicit tenant predicates, and it did not find a cross-tenant path through those audited queries. That is useful containment, but it does not replace the database-level isolation layer.

The correction is migration `000016`, enabling RLS on all ten Phase A tables, retaining FORCE, and extending the catalog and behavioral tests.

This is a major security finding and a mandatory Phase A closure item.

### 8.3 Phase A: the restart proof was not a real restart

The original recovery test started a second server instance while the first was still running.

That demonstrated that a fresh instance could reconstruct state from the database, but it did not prove the actual shutdown-and-restart lifecycle.

The closure plan changes `Server.Close` to be idempotent and error-returning and rewrites the test to shut down the first server before starting the second.

The corrected test must demonstrate that:

- A parked question survives.
- The answer still applies.
- Conversation history reconstructs.
- SSE replay works from the previous cursor.
- A duplicate callback does not reapply state.

This is important because the whole system is designed around recovery from process loss.

### 8.4 Phase A: smaller correctness findings

The review also identified four contained issues:

- M1: inconsistent lock ordering between question answers and event ingestion can cause a database deadlock.
- M2: `questionIDsFor` is unused code.
- M3: `gateway.Parse` should reject trailing JSON data.
- M4: the old migration header inaccurately describes the stream sequence as gapless.

The approved closure plan includes these small corrections where they remain isolated and appropriate.

Other findings are carried forward rather than expanding the closure unnecessarily:

- SSE streams currently have no maximum lifetime or concurrent-stream budget.
- The framework invocation-data persistence behavior must remain disclosed.
- `artifacts.manifest` is agent-supplied JSON and will need secret-safe handling in the artifact review phase.
- Evidence-report count and wording inconsistencies must be corrected.

### 8.5 Phase B: the real Sprite execution path does not yet exist as an accepted integration

This is the next major engineering milestone.

The Studio callback endpoint and protocol exist, but the actual Builder-side gateway must still be built, installed, supervised, and exercised.

The first implementation must prove that the gateway can:

- Start inside the assigned Builder Sprite.
- Run as a persistent Sprite service.
- Accept authenticated Studio commands.
- Launch and supervise OpenCode.
- Translate OpenCode's structured events.
- Persist callbacks in a durable outbox.
- Retry callbacks until Studio acknowledges them.
- Resume from persisted state after gateway restart or Sprite sleep.

Until that works, Studio is coordinating a protocol against test-generated events, not a real coding agent.

### 8.6 OpenCode's native question behavior is still a critical unknown

The central integration risk is not simply whether OpenCode can run a command or edit a file.

It is whether it can participate in the complete interaction lifecycle.

The required proof is:

1. OpenCode receives an assignment.
2. It asks a real clarification question.
3. The question appears in the original Studio conversation.
4. The user answers.
5. The gateway resumes the same OpenCode session.
6. OpenCode demonstrably consumes the answer as the resolution of the question.
7. It continues the requested work.

A transcript that merely contains the user's answer is not sufficient proof.

The file-drop fallback exists precisely because this behavior cannot be assumed from the existence of the question tool.

### 8.7 OAuth and Fly MCP scope are unproven live

The plugin requires an authenticated Fly connection.

The plan identifies three possible headless bootstrap paths:

- Complete the OAuth flow using the URL and a browser.
- Port-forward the callback.
- Use an owner-handled transfer of the authentication file.

These are defined experiments, not established working methods.

The access grant must also be scoped to the assigned Builder and tested from the agent's actual execution context. A successful login alone does not prove least-privilege isolation.

### 8.8 Agent and gateway secrets need an operating-system boundary

The gateway and agent cannot share an identity if the gateway needs to hold signing keys.

Separate Unix users and a restricted artifact handoff are mandatory.

This introduces a real provisioning question: can the Builder Sprite support the required user separation, service identity, privilege drop, and file permissions?

The design deliberately refuses to treat this as optional. If the platform cannot enforce the boundary, the gateway must not quietly fall back to a shared-user process.

### 8.9 Business-effect authorization remains restricted

Two existing governance issues continue to matter:

- **F1:** The operator session-issuance security decision remains deferred.
- **G-01:** Stale `PENDING` invocation ownership remains unresolved.

The Gateway's own build orchestration can proceed, but consequential external business writes through the Miniapp runtime remain disabled until the relevant G-01 correction and acceptance tests are complete.

This prevents the coding agent from becoming an alternate route around Miniapp's execution and authorization boundaries.

---

## 9. The approved Phase A closure

The latest approved plan, `plans/prompt9_plan.md`, orders the closure as follows.

| Order | Work                        | Required proof                                                                                   |
| ----- | --------------------------- | ------------------------------------------------------------------------------------------------ |
| 1     | RLS migration               | v15 test fails the RLS assertion; v16 enables it; application-role isolation works               |
| 2     | Write-path secret scrubbing | Canary content is sanitized before persistence and remains safe through every read/delivery path |
| 3     | Genuine server restart      | First server shuts down before the second starts; state and replay reconstruct                   |
| 4     | M1–M4                       | Lock-order correction, dead-code removal, strict parsing, accurate sequence comment              |
| 5     | Full gates and evidence     | PG19/PG17, formatting, vet, build, tests, race tests, migration round trip, corrected evidence   |

This is a bounded task, not a redesign of the Gateway.

The three mandatory corrections are:

1. Secrets must not persist unsanitized in Studio-owned agent-message surfaces.
2. Recovery must be proved through an actual shutdown and fresh boot.
3. RLS must be enabled on every Phase A table.

The closure is authorized, but the current record establishes approval of the plan, not completion of its implementation.

**Phase A remains unaccepted until the corrected evidence is produced.**

---

## 10. What is expected to work after Phase A is accepted

Once the closure passes, Phase B begins.

### Phase B — Build and install the Agent Gateway

The expected outcome is a functioning gateway service in the Builder Sprite.

It must accept signed commands, operate its local OpenCode adapter, manage the agent's lifecycle, and deliver authenticated callbacks to Studio.

Its durable outbox and session bookkeeping must survive gateway restarts.

The live acceptance target is deliberately small:

1. Install the gateway in a real Builder Sprite.
2. Verify an authenticated `status` round trip.
3. Start a supervised no-op agent turn.
4. Put the Sprite to sleep.
5. Wake it through a new command.
6. Resume or report the agent session honestly.
7. Deliver the resulting `agent.resumed` event.

The test records timings and confirms that no unmanaged processes or resources remain.

This phase also has to establish the actual OpenCode installation, plugin configuration, OAuth scope, process separation, and adapter capability matrix.

### Phase C — Prove genuine two-way interaction

Phase C is where the system must stop being a protocol simulation and become a real conversation with a real coding agent.

The expected proof is a complete user-to-agent round trip:

- User sends a business requirement.
- Studio records the message.
- Gateway wakes the Builder Sprite.
- OpenCode starts in the assigned checkout.
- Progress reaches Studio.
- OpenCode asks a real question.
- Studio presents the question.
- The user answers.
- The gateway resumes the same session.
- OpenCode consumes the answer.
- The agent completes the work.
- Studio receives the result.

The same test must handle disconnects, reconnects, delayed callbacks, duplicates, cancellation, agent termination, gateway restart, Sprite sleep/wake, and Studio restart.

A passing status endpoint is not sufficient. The real acceptance condition is the full question→answer→same-session-resume behavior.

### Phase D — Make the agent's work reviewable

The agent's output cannot simply be trusted because it reports completion.

Phase D establishes the artifact pipeline.

The gateway collects files, produces a manifest, and calculates SHA-256 digests. Studio requests the specific artifacts, verifies the digests, and stores the verified review material.

The review interface is expected to show:

- A unified source-code diff.
- A generated MiniappSpec.
- A machine-readable test report.
- The agent's relevant execution transcript.
- The artifact's verification and provenance information.

Studio must validate generated specifications through the existing MiniappSpec admission and validation path.

A claimed successful build with failing tests must not be presented as a successful result.

The user can approve, reject, or request rework, but there is no automatic deployment in this phase.

### Phase E — Promote approved work into a Runtime Sprite

Only after Phase D is accepted does deployment promotion begin.

The intended sequence is:

1. The user approves an identified, verified artifact digest.
2. Studio provisions a separate Runtime Sprite.
3. The approved artifact is transferred to that environment.
4. The application is installed as a service.
5. Health checks verify the application.
6. Studio records the runtime Sprite, URL, artifact digest, and deployment state.
7. Rollback redeploys a previously approved digest.

The Builder Sprite is not copied wholesale into production.

The Runtime Sprite contains the approved application, not the agent, the Agent Gateway, or its development privileges.

The deployment record and conversation remain in Studio.

---

## 11. The expected end state

The intended experience is deceptively simple for the business owner:

They describe what they want. Studio acknowledges the request and shows progress. The coding agent works inside an isolated Builder Sprite. When it needs information, Studio asks the user. When it has a proposal, Studio presents the choice. When it has completed the work, Studio displays the diff, validation results, and other evidence.

The user can ask for corrections and continue the same conversation.

When the result is satisfactory, the user explicitly approves it for deployment. Studio promotes the approved artifact into a separate Runtime Sprite.

Behind that experience is a system with clear boundaries:

- Studio owns the business process and durable record.
- Miniapp owns framework-level execution, policy, and audit boundaries.
- The Builder Sprite owns development execution.
- The Gateway owns the agent's supervision and protocol translation.
- OpenCode owns its native coding session.
- The Runtime Sprite owns the approved running application.
- Connectors own provider credentials and controlled integration access.
- The human remains the authority for business decisions and consequential permissions.

The coding agent is replaceable. The business conversation, execution history, authorization record, and deployment provenance are not.

That is what makes Miniapp Studio more than a chat interface attached to a terminal: it is intended to become a governed operating platform for AI-assisted business work.

---

## 12. Engineering priorities and what must not be compromised

The immediate priority is not adding more agent features. It is establishing the correctness of the existing foundations.

The sequence should remain strict:

**First:** close Phase A's security and recovery findings.

**Second:** prove the Gateway and OpenCode work together inside a real Builder Sprite.

**Third:** prove a real question-and-answer interaction, including session resumption.

**Fourth:** prove that generated artifacts can be collected, verified, reviewed, and corrected.

**Fifth:** prove that only an approved artifact can reach the Runtime Sprite.

Several shortcuts would undermine the entire architecture:

- Treating a passing API status check as proof of a working agent.
- Treating a new process as proof of a resumed session.
- Treating a user's answer as authorization.
- Giving the Builder production connector credentials.
- Giving the agent access to Gateway signing keys.
- Trusting an agent's claim that tests passed.
- Deploying directly from the Builder filesystem.
- Treating Sprite-local state as the permanent business record.
- Bypassing unresolved F1 or G-01 decisions.

The design is intentionally conservative at these boundaries. A coding agent that can build software but cannot demonstrate the safety and provenance of its work is not yet a reliable business automation platform.

---

## 13. Controlling engineering records

The following repository documents define the current architecture and its acceptance criteria:

1. `plans/agent-gateway-implementation-plan.md` — governing five-phase architecture and implementation roadmap.
2. `plans/agent-gateway-opencode-amendment.md` — OpenCode designation, plugin topology, OAuth, permissions, adapter behavior, and live proof requirements.
3. `plans/adv_review_prompt8_phase_a.md` — adversarial Phase A review and verified findings.
4. `plans/prompt9_plan.md` — approved bounded Phase A closure execution plan.
5. `docs/gateway-phase-a-evidence.md` — original Phase A implementation evidence; it must be updated after closure and must not be treated as the final acceptance record while the findings remain open.

**Current decision:** Phase A closure is authorized; Phase A acceptance is pending. Phase B is the next phase, but it must wait for the corrected Phase A evidence.

The writeup follows the governing gateway plan, the OpenCode amendment, the adversarial findings, and the approved closure plan. In particular, the original Phase A evidence is not treated as final acceptance because the later live review demonstrated otherwise.&#x20;

agent-gateway-implementation-plan.md

agent-gateway-opencode-amendment(1).md

adv_review_prompt8_phase_a.md

prompt9_plan(1).md
