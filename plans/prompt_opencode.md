Agreed. **OpenCode is now the designated initial coding agent for Miniapp Studio's Builder Sprites.** This is an important architectural decision, not merely an implementation preference.

I verified Fly.io's ecosystem page and the official [`superfly/sprites-opencode-plugin`](https://github.com/superfly/sprites-opencode-plugin?utm_source=chatgpt.com) README.

The integration provides several useful capabilities:

- OpenCode connects to Fly MCP at `https://mcp.fly.io`.
- The plugin provides Sprite management, remote execution, filesystem operations, checkpoints, and services.
- It supports browser-based OAuth.
- It supplies workflow guidance and safeguards for destructive Sprite operations.
- OpenCode 1 requires version 1.18.23 or later; OpenCode 2 has a separate beta configuration.
- OpenCode remains the agent, while its remote development and execution work happens inside Sprites.

One especially important distinction: **the Fly plugin connects OpenCode to Sprites; it does not itself implement our Miniapp Studio conversation bridge.** We still need the Agent Gateway and the Studio-owned conversation protocol.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly



## **Updated instruction for the coding-agent planning prompt**

Add the following section to the previous prompt, preferably under the locked architectural decisions and coding-agent adapter architecture.

## 15. Mandatory Coding Agent: OpenCode + Official Fly.io Sprites Plugin

**OpenCode is the designated initial and primary coding agent for Miniapp Studio Builder Sprites.**

This is an approved project decision, not an optional technology comparison.

Official references:

- Fly.io Agent Ecosystem: [https://fly.io/sprites/ecosystem/](https://fly.io/sprites/ecosystem/)
- Official plugin: [https://github.com/superfly/sprites-opencode-plugin](https://github.com/superfly/sprites-opencode-plugin)
- Plugin README: [https://github.com/superfly/sprites-opencode-plugin/blob/main/README.md](https://github.com/superfly/sprites-opencode-plugin/blob/main/README.md)

### A. Adopt the official integration

Use the official `@flydotio/sprites-opencode-plugin` package.

For OpenCode 1, the documented configuration is:

```
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["@flydotio/sprites-opencode-plugin"]
}
```

The plugin requires OpenCode 1.18.23 or later. Its documented installation uses OpenCode's automatic Bun-based plugin installation.

The official repository also supports OpenCode 2 through a separate plugin entry point and configuration format. Do not assume the two versions have identical plugin APIs.

The implementation plan must identify the recommended OpenCode version to pin for the first production implementation. Prefer the stable OpenCode 1 integration unless repository evidence or an explicit owner decision supports OpenCode 2.

If the package is unavailable from npm at implementation time, follow the official repository's instructions for a local checkout and absolute `file://` configuration. Do not invent an alternative installation procedure.

### B. Understand exactly what this plugin provides

The official plugin connects OpenCode to Fly MCP at:

`https://mcp.fly.io`

It provides:

- Hosted MCP access to Fly.io Sprites.
- Browser-based OAuth authentication.
- Sprite listing and creation.
- Remote command execution.
- Remote filesystem operations.
- Service management.
- Filesystem checkpoints and restoration.
- Outbound network-policy operations.
- Sprite workflow guidance.
- Safeguards and approval prompts for destructive operations.

Do not recreate these capabilities unnecessarily in Miniapp Studio.

The official plugin must be used as the supported integration between OpenCode and Fly.io Sprites.

### C. Preserve the distinction between OpenCode, the plugin, and Agent Gateway

These are three different things:

**OpenCode:** The actual coding agent. It understands implementation instructions, examines repositories, writes code, runs tests, and produces artifacts.

**Official Sprites OpenCode Plugin:** The Fly-provided integration that gives OpenCode access to Sprite infrastructure through Fly MCP.

**Miniapp Studio Agent Gateway:** Our own integration component responsible for the business-facing two-way conversation protocol, build assignments, user questions, agent messages, event delivery, session association, and artifact reporting.

The Fly plugin does NOT automatically provide the Miniapp Studio conversation bridge.

Do not incorrectly treat the plugin's MCP connection as the mechanism for forwarding business-user chat messages to Studio.

### D. Define the OpenCode execution topology

The implementation plan must investigate and explicitly distinguish:

1. Where the OpenCode process runs.
2. Where the Sprite-management plugin is installed and configured.
3. Where the coding agent's actual development commands execute.
4. Where the repository and persistent working files reside.
5. How the Agent Gateway interacts with the OpenCode process.
6. How agent messages, questions, progress, and completion events are captured.
7. How a session is resumed after Sprite sleep or process termination.

The desired execution model is that the coding agent performs its development work in the persistent Builder Sprite, rather than accidentally running builds and file changes on Studio's host machine.

The official README explains that OpenCode can run outside the Sprite while using the plugin's remote execution tools to perform work in the Sprite. Therefore, do not assume that installing the plugin alone moves the OpenCode process into the Sprite.

Investigate the correct supported execution arrangement for our architecture. In particular, determine how to run OpenCode inside the Builder Sprite, how the official plugin is loaded there, and how its authentication and session state persist across sleep/wake cycles.

Where the official plugin is designed for a different topology, document the implications and recommend a supported approach rather than silently bypassing it.

### E. Authentication and credential boundaries

The official plugin uses browser-based OAuth for Fly MCP.

The plan must specify:

- How the initial OAuth authorization is performed.
- Which identity and Fly organization the Builder Sprite operates under.
- How authentication state survives normal Sprite lifecycle events.
- How access is restricted to the intended Sprites and organization.
- Whether restricted Sprite-name prefixes or creation limits are appropriate.
- How OpenCode's Fly MCP permissions are configured.
- Which destructive operations require human confirmation.

Do not place Fly API tokens or unrelated third-party API credentials in build prompts, source files, repositories, logs, or application artifacts.

Do not invent a non-interactive authentication workaround or install the Sprite CLI merely to bypass the supported plugin authentication flow.

### F. Versioning and compatibility

Pin and document:

- OpenCode version.
- `@flydotio/sprites-opencode-plugin` version.
- Node.js/Bun runtime requirements.
- Plugin configuration format.
- MCP endpoint.
- Authentication expectations.

Inspect the official plugin's `package.json`, README, release history, source, and tests where necessary.

Record verified compatibility facts and distinguish them from assumptions.

Do not depend on an unpinned floating Git branch in a reproducible production setup.

### G. OpenCode adapter contract

The Miniapp Studio Agent Gateway must have an OpenCode adapter as its first fully implemented coding-agent integration.

The adapter must translate between Studio's stable event protocol and OpenCode's actual supported session and execution interfaces.

Investigate OpenCode's supported programmatic interfaces, SDKs, CLI modes, event streams, session management, and non-interactive execution behavior.

Do not invent CLI flags or assume that terminal output is a reliable structured event protocol.

The implementation plan must establish how Studio can:

- Start an OpenCode build.
- Send an initial assignment.
- Receive assistant messages.
- Receive progress and tool activity.
- Detect when OpenCode requires user input.
- Forward a user's answer.
- Resume the same conversation and session.
- Detect completion or failure.
- Retrieve generated files, diffs, and test reports.

If OpenCode does not expose a particular capability through a stable supported interface, explicitly identify that limitation and design a tested adapter boundary around it.

### H. Acceptance proof

The first OpenCode integration must demonstrate this complete round trip:

1. Studio creates a build assignment.
2. Studio wakes or provisions the assigned Builder Sprite.
3. The Sprite starts the pinned OpenCode environment.
4. The official Fly Sprites plugin is loaded.
5. OpenCode verifies its authenticated Sprite connectivity.
6. OpenCode inspects the assigned repository inside the Sprite.
7. OpenCode sends a progress event to Studio.
8. OpenCode asks the business user a real clarification question.
9. Studio displays that question in the original conversation.
10. The user answers.
11. Studio forwards the answer to the same OpenCode session.
12. OpenCode resumes work.
13. OpenCode generates a small artifact and runs a test inside the Sprite.
14. OpenCode reports its result and evidence to Studio.
15. Studio displays the result in the original conversation.

Test process interruption, Sprite sleep/wake, duplicate events, reconnects, and session recovery.

Do not consider the integration complete merely because OpenCode can list Sprites or execute a remote shell command.

### I. Future agent portability

OpenCode is the first and primary implementation.

Keep the Agent Gateway and Studio conversation protocol sufficiently independent that other coding agents can be added later through adapters.

However, do not prematurely implement multiple coding-agent adapters or a generalized multi-agent orchestration framework.

The first milestone is a reliable, production-quality OpenCode integration.

**My architectural recommendation:** Run OpenCode as the coding agent inside the Builder Sprite, with the official Fly plugin providing Sprite-management and Fly MCP integration. Build our own Agent Gateway around OpenCode's supported session interfaces. This preserves the official Fly integration while giving Miniapp Studio ownership of the user-facing conversation.

The plugin's actual README and package metadata should be treated as the technical source of truth during planning, since its OpenCode 1 and 2 entry points differ.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly



One critical acceptance condition: prove the complete question → user answer → same OpenCode session resumes round trip. A successful `/sprites-status` check alone does not prove our architecture works.