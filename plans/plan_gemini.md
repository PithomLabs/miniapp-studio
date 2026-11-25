# Miniapp Studio: Consolidated Product & Hacksgiving Blueprint

**Miniapp Studio** turns the pre-existing `miniapp` Go framework into a **generic application & personal-agent generation platform**—delivering reusable primitives, a declarative spec/plan model, and a clean UI so users generate apps from intent rather than hand-building 12 separate vertical products.

To stay honest to the **Hacksgiving** rules (Nov 23–29, 2026: a personal agent running on a Fly.io Sprite using at least one Sprites connector) and the judging panel (**TypeSafe AI, Box, Fly.io, Pinecone, Ollama**), the design separates what is **built/demoed now** from what is **explicitly scoped as the post-hackathon durable workflow roadmap**.

---

## 1. Consolidated Best Features Across the Blueprints

* **Generate Declarative Specs/Plans, Not Arbitrary Code:** A Sprite-hosted agent generates a validated, schema-checked declarative app/plan specification (and/or data-driven step plan) executed by a fixed runtime/interpreter—avoiding arbitrary code execution in the trusted runtime and letting validation catch bad specs before deployment.


* **Three-Layer Separation of Concerns:**
1. **Miniapp Studio (UI & Builder Layer):** Describe/prompt input, live preview/spec inspector, permissions/connector policy view, approval inbox, run/checkpoint timeline, and governance controls.


2. **Spec Compiler / Plan Interpreter Layer:** Deterministic validation, capability/scope resolution, fixture checks, and execution over vetted primitives.


3. **Miniapp Foundation Layer (Go + DBOS + River + Neon / Local State):** Trusted execution, capability proxy, effect tracking, approvals, audit logs, and policy enforcement.




* **Six Core Primitive Families:**
* **Data & State:** Entities, filters, bindings, and scoped state.


* **UI:** Forms, tables/lists, detail/metric cards, approval cards, permissions cards, and run timelines.


* **Workflow & Control:** Triggers (manual, schedule, webhook, event), actions, branching/bounded loops, waits, and human approvals.


* **Integration & Connectors:** Sprites gateway connectors, custom REST/HTTP APIs, and capability discovery.


* **Governance & Safety:** Least-privilege scopes, out-of-band approvals (harness-agnostic even when a terminal harness lacks native confirmation hooks), rate/run limits, kill switches, untrusted-data tagging, and effect/reversibility declarations.


* **Presentation & Publishing:** Reusable layouts, saved/reopenable generated apps, and template gallery examples.




* **Sprites Hardware Isolation & Checkpoints:** One Sprite per personal agent for hardware-isolated execution, sub-second filesystem checkpoints for safe iteration/rollback, and off-Sprite tracking for external side effects that disk snapshots cannot rewind.


* **Honest Rule & Foundation Hygiene:** Keep the pre-existing `miniapp` framework as a pinned external dependency/foundation, build the Studio/generator/runner/connectors/UI as new Hacksgiving work, and only claim behaviors verified by actual tests.



---

## 2. Custom API Connectors & Supported Ecosystem Catalog

### Extensibility via Custom API Connectors (`docs.fly.io/sprites` & `docs.fly.io/sprites/api/connectors`)

Miniapp Studio treats **Custom API Connectors** as first-class extension points so users can connect any token-authenticated HTTP API without placing raw secrets inside the Sprite:

* **Credential Injection Modes:** Support configuring custom API connectors (`base_api_url`, token, and placement) across `header` (with configurable prefix, e.g., `Bearer`), `custom_header` (defaulting to `X-Api-Key`), `query_param`, and `url_path` templates—while keeping stored credentials off the Sprite and out of API responses.


* **Pre-Save & Stored Credential Testing:** Use `/v1/oauth/connections/custom_api/test` (with `connection_id` or explicit `access_token` + `test_url`) to validate connectivity before wiring a connector into a generated app, respecting origin-bound credential checks and internal-address rejection.


* **Default-Deny & Path-Level Policies:** Surface the connector's access policy (`allow_all`, Sprite name prefix/label matching, and endpoint allow/block path rules where explicit **block rules win** over allow rules) in Studio's **Permissions / Connectors** view so least privilege is visible before deployment.



### Displaying Supported Connectors & Ecosystem (`fly.io/sprites/ecosystem/`)

Rather than hardcoding a static list that can go stale or vary by organization rollout, Studio's **Connectors & Ecosystem Explorer** should dynamically query the Sprites Connections API (e.g., `GET /v1/oauth/connections` for configured/available connections in the org) and organize the ecosystem across the exact sections from `[https://fly.io/sprites/ecosystem/](https://fly.io/sprites/ecosystem/)`:

* **`#connectors` (Sprites API Gateway Connectors):** Group connectors into *Managed by Sprites* vs. *Your Connections* (OAuth, BYOK API keys, and Custom API connectors), highlighting key hackathon-relevant providers (such as Box, GitHub, Slack, OpenRouter, Ollama, Meta/Muse Spark, Pushover, and ZeroGPU, noting that availability varies by organization).


* **`#integrations` (Agent Plugins, Remote MCP & Platform Integrations):** Display how agents and external tools connect to Sprites—including Remote MCP (`[https://sprites.dev/mcp](https://sprites.dev/mcp)` / `mcp.fly.io`), coding-agent plugins/skills (Claude Code, Codex, Gemini CLI, Cursor, Pi, Hermes, etc.), and platform integrations.


* **`#sdks/golang` (Go SDK & Runtime Alignment):** Surface Go SDK / Go runtime integration status alongside the platform's Go control plane so builders can see how generated specs and connector calls map cleanly into Go tooling.



---

## 3. Vision vs. Reality: Delineating Durable Workflows on the Roadmap

**Durable workflow** is the flagship long-term differentiator of the `miniapp` architecture—giving generated apps multi-day waits, crash-safe step checkpoints, idempotent external-effect ledgers (`PENDING -> ATTEMPTING -> CONFIRMED | FAILED | UNKNOWN`), reconciliation sweepers, and saga compensation. However, full production-grade durable workflows belong on the post-hackathon roadmap—not as an unverified claim in the hackathon demo.

| Dimension | Hackathon Demo Scope (Nov 23–29, 2026)

 | Post-Hackathon Roadmap (Durable Workflow Differentiator) |
| --- | --- | --- |
| **Core Demo Focus** | Studio UI + spec/plan generator + connector discovery (including Custom API connectors) + Sprite deployment + basic approval/checkpoint/rewind flow on 1–2 concrete apps.

 | Full Class D durable workflow engine across multi-day sleeps/events, partitioned entity queues, saga rollbacks, and automated reconciliation sweepers.

 |
| **State & Recovery** | Working spec execution, Sprite filesystem checkpoints/restore, and tested baseline flows on the pinned foundation.

 | Hardened multi-version workflow upgrades/patches, DBOS/River operational SLAs, PITR recovery playbooks, and quarantined `UNKNOWN`-effect resolution pipelines.

 |
| **Governance & Scale** | Single-user personal Sprite model, visible connector/path permissions, approval gate, and kill switch.

 | Enterprise multi-tenancy, fine-grained RBAC/org policies, per-principal blast-radius quotas, and embedded/white-label SaaS mode.

 |
| **How to Pitch It** | Show what actually runs live on the Sprite + connector + UI today, and label durable multi-step/multi-day orchestration as the **Roadmap Differentiator** powered by the underlying Go/DBOS/River foundation.

 | Never claim "zero duplicate effects" or "unbreakable multi-day durability" unless a live test in the demo/repo proves that exact case.

 |

---

## 4. Practical Build Sequence for Hacksgiving Week

1. **Wire Connectors First:** Verify your organization's active connectors and test a **Custom API Connector** (`POST /v1/oauth/connections/custom_api/test` + create) alongside at least one standard Sprites connector with credentials kept off the Sprite.


2. **Build the Studio-to-Spec Loop:** Create a focused UI (Describe/Spec view, Connectors & Ecosystem browser, Approval/Run view, and Checkpoint/Kill-Switch controls) where a Sprite-hosted personal agent generates a validated declarative spec/plan and runs it.


3. **Demo One Complete Flow + Explicit Roadmap:** Show end-to-end generation, connector usage, approval/checkpoint behavior, and connector/policy visibility—then clearly frame **durable workflows** as the architectural moat being expanded next on the roadmap.
