**Design approved. GO for repo-only broker implementation.**

This is the right direction for hands-off execution. The agent has established that the existing execution mechanisms cannot safely cross the sandbox boundary, so a narrowly scoped host-side broker is justified.

Four binding requirements:

1. **No generic command execution.** Typed operations only, fixed Docker resources, arguments, mounts and environment paths. No shell interpolation or arbitrary host paths.
2. **Treat the job queue as untrusted input.** Reject symlinks, path traversal, malformed or oversized jobs, duplicate IDs and unknown fields. Use atomic job transitions and honest `UNKNOWN` recovery.
3. **Preserve the privilege boundary.** The agent never receives Docker socket access. Explicitly document that the broker itself is a privileged security boundary, and harden its service.
4. **Bound the authorization.** The existing GO covers the approved PostgreSQL runner deployment sequence, not unrestricted future Docker operations. Bind execution to that workflow and automatically disarm deployment-only capabilities afterward.

Also pin the approved runner build inputs so the broker cannot unknowingly build an altered Containerfile.

**Important distinction:**

-

  GO

  &#x20;Broker code, tests and documentation.


-

  Not yet authorized

  &#x20;Broker installation, service startup and Docker operations.



After implementation, review the evidence once and authorize the one-time setup. No more manual Docker command sequences for the Project Owner.


## prompt

to clarify, so in theory, your requirements above apply not only to Muse, is that right? enlighten me


## answer

Yes. **These are general principles for safely giving AI agents the ability to perform consequential operations—not requirements unique to Muse.**

What we are designing for Muse is one concrete implementation of a much broader architectural pattern.

## **The general principle: Agents need capabilities, not unrestricted authority**

An AI agent should be able to accomplish useful work without receiving unrestricted access to the machine, infrastructure, credentials, or accounts on which it operates.

The architecture looks like this:

**AI Agent**

Reasons, plans and requests actions

Structured request + authorization

**Trusted Execution Broker**

Validates identity, permissions, scope and arguments

Approved operation only

**Protected Resources**

Docker, databases, GitHub, Fly.io, files, cloud infrastructure

Receipts, evidence and execution results

## **Principles that apply universally**

| Principle                  | Meaning                                                                     |
| -------------------------- | --------------------------------------------------------------------------- |
| Least privilege            | Give agents only the capabilities they need.                                |
| Capability-based access    | Expose specific operations, not unrestricted shell or system access.        |
| Explicit authorization     | Consequential actions require appropriately scoped approval.                |
| Trusted execution boundary | Enforce permissions outside the agent's control.                            |
| Input validation           | Treat every agent request as untrusted input.                               |
| Auditability               | Record what was requested, authorized and actually executed.                |
| Idempotency and recovery   | Handle retries, crashes and uncertain outcomes safely.                      |
| Secret isolation           | Never expose privileged credentials unnecessarily.                          |
| Fail closed                | Missing or invalid authorization means no operation.                        |
| Reality-first verification | Prove actual effects and security boundaries, not merely intended behavior. |

## **What is specific to Muse?**

The following are implementation choices, not universal requirements:

- JSON job queues.
- A host-side daemon.
- systemd.
- Docker-specific operation allowlists.
- Workspace files for communicating results.
- A particular broker protocol.

Another environment might use an MCP server, Kubernetes operator, cloud control plane, CI runner, RPC service or operating-system capability mechanism.

The architecture should remain the same even when the implementation changes.

## **The bigger architectural insight for Miniapp Studio**

This is particularly important to our north star:

**Miniapp Studio is not merely an application that talks to AI agents. It is a control plane that gives AI agents controlled capabilities to operate real systems.**

Muse, Claude Code, Codex, OpenCode or a future agent should not receive special trust merely because of its identity.

They should all interact through the same capability and authorization model.

The trusted execution infrastructure becomes reusable. Today it controls Docker. Tomorrow it might control Fly Machines, GitHub operations, database migrations, deployments or other infrastructure.

**One governing principle: Trust the authorization boundary and verified execution evidence, never the agent's unrestricted access or promises.**