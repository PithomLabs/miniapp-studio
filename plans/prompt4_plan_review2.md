# Implications for Sprites, and Whether Studio Is "The Security Layer"

## Answering the question directly

**No — Miniapp Studio is not the security layer for Sprites. Calling it that would be a category error, and one with real consequences for how the system fails.**

What Studio *is*: the **control plane** that orchestrates isolated execution environments and governs what agents are authorized to do inside them. It sits above Sprites, uses Sprites' security properties, and adds a capability layer on top. It does not provide the isolation itself, and it must not — because if it did, a compromise of Studio would be a compromise of the whole fleet.

The correct decomposition is three layers, each of which provides a different kind of safety:

| Layer | What it provides | What it cannot provide | Who is responsible |
|---|---|---|---|
| **Fly Sprites** | Hardware isolation (Firecracker), network policy enforcement, persistent filesystem with checkpoint/restore, connector gateway (credentials outside the guest) | Typed operation semantics, business authorization, durable capability audit, approval workflow, effect reconciliation | Fly |
| **Broker pattern** | Typed operations, fixed arguments, scoped capabilities, bounded authorization (auto-disarm), honest UNKNOWN recovery, audit at capability level | Isolation, credential custody, network policy, disk persistence | Studio's architecture (the broker is one implementation) |
| **Miniapp Studio** | Tenant-scoped capability authorization, workflow orchestration, approval gates, deployment provenance, conversation as durable record | Low-level isolation, real-time capability enforcement, credential custody | Studio |

Each layer depends on the layer below it. Sprites could exist without the broker (many agent deployments use Sprites without one). The broker could exist without Studio (it did — the Muse broker). Studio could exist without Sprites (in principle, it could orchestrate any VM substrate). The combination is the product, but no single layer is "the security layer" for the layers above or below it.

---

## Why the "security layer for Sprites" framing is dangerous

Three specific risks:

**1. It implies Studio holds the isolation primitive.** If Studio were the security layer for Sprites, then breaking out of Studio would mean breaking out of Sprites. That is exactly the failure mode the architecture should prevent. The correct posture is: **Sprites are secure regardless of what Studio does.** Studio's job is to *use* Sprites' guarantees, not to provide them.

**2. It conflates two failure modes.** A compromise of Studio and a compromise of a Sprite are different incidents with different blast radii and different recovery procedures. Conflating them hides the real question — *what can an attacker who controls one of these things actually do?* The answer must be "much less than if they controlled both," and that answer depends on the layers being independent.

**3. It makes Studio a single point of trust.** A control plane that is also the security layer is a control plane whose compromise breaks all tenants. The correct posture is: **Studio should be replaceable without compromising the isolation of any tenant's Sprite.** The broker, the Sprite, and the connector gateway should each be independently defensible. Studio's compromise should reduce to "an attacker can authorize actions on their own behalf," not "an attacker can escape the sandbox."

This is the same principle the AWS SaaS Factory guidance established: the control plane is powerful but should not be the isolation mechanism. The isolation mechanism is the execution plane's job. Studio is the control plane.

---

## What the broker review teaches when applied to Sprites

The broker review in `prompt4_plan_review.md` refined two properties that *apply directly* to Sprites — and got them right in a way that clarifies the layering.

### Property 1 — Owner-side state outside the agent's reach

The review said: *"StateDir, EnvFile, build pins and authorization state must be outside **all agent-accessible and agent-writable paths**, not merely outside the repository."*

On a developer workstation, this means *"not in /tmp, not in the repo, not in any directory the agent's Unix user can write to."* The broker's state goes into a directory owned by `studio-gw` that `studio-agent` cannot read.

On a Sprite, the equivalent has a stronger answer: **owner-side state goes to Studio's tenant-scoped database, not to any directory on the Sprite at all.** The Sprite should hold only what the agent legitimately needs. The broker's authoritative records — capability grants, audit logs, disarmed flags, digest pins — live in the control plane, unreachable from the Sprite.

This is the correct application of the review's mandate. "Outside the agent's reach" is stronger than "outside the repo" — and on a Sprite, "outside the Sprite" is stronger than "outside the repo directory on the Sprite." The right answer uses the strongest available boundary.

The implication for the Muse broker's design: the design's *principle* (owner-side state outside agent-reachable paths) applies verbatim to Sprites. The design's *implementation* (a directory owned by `studio-gw` on the same filesystem) is the workstation-specific version. The Sprite-specific implementation is "state on the control plane," because the control plane is the only place truly outside the Sprite's reach.

### Property 2 — Two snapshot domains (INFRA vs RUN)

The plan distinguished *INFRA inputs* (baked into the image at build time, digest-pinned, immutable) from *RUN inputs* (the whole repo, snapshotted per run, provenance frozen, owner-diff-reviewed).

On Sprites, the same distinction applies but with Sprites-native mechanisms:

| Domain | On a workstation | On a Sprite |
|---|---|---|
| INFRA inputs | Baked into the Docker image, digest-pinned | Frozen in the Sprite checkpoint at provision, or in the artifact digest per Phase E |
| RUN inputs | Repo mounted read-only as per-job snapshot | Workspace checkout snapshotted via Sprite checkpoint per build |
| Immutability mechanism | Docker image ID | Sprite checkpoint ID, or Fly artifact digest |
| Provenance record | Tree hash + git HEAD/status on the workstation | Checkpoint ID + tree hash in Studio's DB |

The *principle* — separate the two domains, pin both, snapshot the mutable one per run — applies. The *mechanism* changes from Docker image digests to Sprite checkpoint IDs. This is exactly what the broker review meant by *"architecture stays the same even when the implementation changes."*

---

## What Sprites teach back to the broker pattern

The relationship is bidirectional. Three things Sprites get right that the Muse broker had to construct:

### 1. Isolation is not the broker's job

On the developer workstation, the broker is the security boundary. If the broker is compromised, so is the host. The broker had to construct its own isolation: Unix user separation, signed protocols, queue-as-untrusted-input, digest-pinned builds. That work is necessary but expensive, and the security depends on the broker being *correct*.

On a Sprite, the broker is a layer *within* an isolation boundary that already exists. The Firecracker boundary is provided by Fly and does not depend on Studio's broker code being correct. The broker's job is capability governance, not isolation. This means:

- The broker on a Sprite is *less critical* to overall safety than the broker on a workstation.
- If the Sprite broker has a bug, the blast radius is bounded by the Firecracker boundary.
- If the workstation broker has a bug, the blast radius is the whole developer machine.

**Implication for Studio:** The Muse broker's rigor should still apply to the Sprite broker (typed operations, digest pinning, honest UNKNOWN, scoped authorization), but the failure mode is more contained. Studio's broker has *more* margin for error than the Muse broker did — and should still not use it.

### 2. Credentials are handled by the gateway, not the broker

On the developer workstation, the broker has to design its own credential handling: env-file paths, mode bits, no DSN in results, no secret in logs. That work was necessary and correct, and it took a lot of the review's attention.

On a Sprite, credentials never enter the guest. The connector gateway holds them. The broker on the Sprite passes capability identifiers, not secrets. This means the broker *cannot leak what it does not have*.

**Implication for Studio:** The credential-isolation parts of the Muse broker review are, on Sprites, handled by Sprites' own design. Studio's broker should still enforce that no secret-shaped value appears in its own audit records, but the primary defense is upstream. This is a genuine simplification, not a place to skip rigor.

### 3. Checkpoint handles the disk recovery case

On a workstation, the broker's "sweep and honest UNKNOWN" pattern had to reconstruct disk state after a crash. The review's U3 handling of "sweep validates residue, never overwrites terminal results" is the correct pattern for a system without checkpoint.

On a Sprite, checkpoint/restore handles disk state. The broker's UNKNOWN pattern is still needed for *external* effects (did the API call actually go through?), but it is no longer needed for *disk* state (the checkpoint is the disk state).

**Implication for Studio:** The broker's sweep becomes narrower. It reconciles external effects and capability state; it does not reconcile filesystem state. This is a real reduction in broker complexity on Sprites, and a real reason not to build the Muse broker's full disk-sweep machinery into the Sprite broker.

---

## What this means for the Miniapp Studio architecture

Five concrete architectural implications:

### 1. The broker is per-Sprite, not per-host

On a workstation, one broker controls one Docker daemon on one machine. On Sprites, each Sprite has its own broker instance (or its own slice of a shared broker's capability table). The Muse broker's "single-flight via lockfile" pattern becomes "single-flight per Sprite per operation type" — a scale change, not a pattern change.

### 2. The authorization boundary is in Studio, not on the Sprite

The Muse broker's auto-disarm-after-workflow pattern is a one-time authorization tied to one workflow. On Sprites, the pattern generalizes: **Studio holds the authorization table; the Sprite broker holds the capability table that enforces it.** When Studio revokes a capability, the Sprite broker's next job for that capability fails with `capability-revoked`, not because the agent was asked politely but because the capability is no longer in the table.

This is the correct layering. Studio does not have to be reachable at every operation — the Sprite broker can serve locally-authorized operations during a brief Studio outage. But Studio's revocation is authoritative, and the Sprite broker reconciles against Studio on every batch.

### 3. The Sprite broker is a controlled capability surface, not a Docker CLI wrapper

The Muse broker's "typed operations only, fixed arguments, no shell interpolation" pattern applies verbatim. But on a Sprite, the "operations" are different:

| On a workstation (Muse broker) | On a Sprite (Studio broker) |
|---|---|
| `docker network create` | `sprite.service.start` (specific service) |
| `docker run` (with pinned image) | `sprite.checkpoint` (specific checkpoint), `sprite.service.invoke` |
| `docker inspect` | `connector.call` (specific provider, specific paths) |
| `docker rm` | `sprite.checkpoint.delete` (specific checkpoint) |

The specific operations differ. The pattern — typed, fixed-argument, allowlisted, digest-pinned — is identical.

### 4. The dual-user pattern on Sprites is the same pattern with a different substrate

The Muse broker review established: `studio-gw` vs `studio-agent` Unix users, HMAC keys readable only by `studio-gw`, agent receives `EACCES`.

The Sprite architecture (per `writeup2.md`) specifies the same pattern: `studio-gw` vs `studio-agent` Unix users, HMAC key paths unreadable by the agent, file handoff via `studio-io` group. The pattern is identical because the threat is identical: **the agent process and the supervisor process share a filesystem, and the supervisor holds key material the agent must not read.**

The review's mandate that provisioning fails closed if the OS separation cannot be enforced applies verbatim to Sprite provisioning. This is not a coincidence — it is the same architectural problem.

### 5. The Muse broker's threat model clarifies the Sprite broker's threat model

The Muse broker review's threat model assumptions:

- The agent is untrusted input.
- The agent can write to the workspace.
- The agent can construct malformed jobs.
- The agent can race filesystem operations.
- The agent can crash the broker.
- The agent may deliberately attempt argument injection.

Every one of these assumptions holds on a Sprite. The broker's defensive design is not Muse-specific — it is **the correct defensive design for any agent supervisor that the agent can attempt to attack**.

The implication: the Sprite broker should be reviewed with the same adversarial discipline as the Muse broker. The `prompt3.md`-style review (threat model, mandatory adversarial tests, one remediation pass) should be the template for every capability surface Studio exposes to a Sprite.

---

## The honest answer to "does Miniapp Studio sort of the security layer for Sprites"

**No, and framing it that way would be a mistake that produces concrete security and architectural debt.** The relationship is more accurate (and more defensible) stated as:

- **Sprites provide isolation.** Their security guarantees (Firecracker, network policy, connector gateway, checkpoint) hold independently of what Studio does.
- **The broker pattern provides capability governance.** The pattern (typed ops, fixed args, scoped authorization, honest UNKNOWN, owner-side state) is what makes "the agent can do X but only X" enforceable.
- **Miniapp Studio provides control-plane orchestration.** It decides who can grant what capability to which agent on which Sprite, records the decision, and enforces it at provisioning and deployment time.
- **The combination provides what none of them provides alone:** a governed, auditable, reversible system in which an untrusted agent can be given consequential capabilities under human authority.

The correct sentence is:

> **Sprites make agents isolated. The broker pattern makes capabilities typed and scoped. Miniapp Studio makes the whole thing governable. None of these is the other's security layer — they are three independent layers whose composition is the product.**

The safety of the product depends on the layers remaining independent. If Studio becomes the "security layer" for Sprites, the failure modes collapse and the architecture loses the property it was designed to provide: **a compromise of one layer is contained.**

The Muse broker review's core insight — *"Trust the authorization boundary and verified execution evidence, never the agent's unrestricted access or promises"* — applies to each layer independently. The layer that provides isolation trusts its substrate. The layer that provides capability governance trusts the isolation layer below it. The layer that provides orchestration trusts both. None of them trust the layer above it. That asymmetry is the design.

That is the honest reading of the review in the context of Sprites. The broker pattern did not invent a new kind of security — it named the discipline that a capability surface requires. Sprites provide one class of substrate; the broker provides another; Studio provides a third. They are complementary, not substitutable.