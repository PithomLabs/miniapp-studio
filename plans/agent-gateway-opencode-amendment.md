# Agent Gateway — OpenCode Amendment (separate review doc)

**Status:** Proposed 2026-11-24, awaiting owner review and explicit merge
approval. **Parent:** `plans/agent-gateway-implementation-plan.md` (the
gateway plan), which this document does **not** modify. Source prompt:
`plans/prompt_opencode.md`; review direction:
`plans/roadmap3_plan_review.md` (separate amendment first, merge via
explicit section-by-section change list).

**Document-only:** no implementation, install, authentication,
provisioning, commit, or push was performed. All external claims carry
source + version + access date in §J; anything not verifiable from here is
labeled `[ASSUMPTION]` with its prescribed live proof.

## Central finding

The official plugin's outside-topology and the gateway plan's
inside-topology are **compatible without changing any locked decision**
(see §D). The plugin assumes OpenCode is outside each sprite it *manages
via MCP* — and that stays true: an OpenCode process inside Builder Sprite
B still manages every MCP-targeted sprite remotely (including lifecycle
operations on B itself by name), while doing repository work in B with its
native local tools. No plugin fork, no invented install path, no
non-interactive auth workaround is required. Three frictions need explicit
handling, and this amendment handles them: (1) the plugin's conflicting
system guidance is disabled via its supported `guidance: false` option and
replaced with verified Builder instructions (§D); (2) the `question` tool's
behavior under `opencode run` is undocumented and gates the critical
acceptance (§G); (3) headless OAuth bootstrap needs a verified ceremony
(§E). None of them moves a locked decision.

## A. Adopt the official integration

OpenCode is the designated initial and primary coding agent. The
integration is `@flydotio/sprites-opencode-plugin` **version 0.1.0**
(package.json, unpublished — see below), loaded as an OpenCode 1 `plugin`
entry:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": [
    [
      "file:///absolute/path/to/sprites-opencode-plugin",
      { "guidance": false }
    ]
  ]
}
```

(The tuple form is the documented OpenCode 1 options shape; `guidance:
false` is correction 1 — see §D. MCP, commands, and permission defaults
stay enabled.)

- **npm is not available (verified 2026-11-24):** the npm registry returns
  404 for the package, the repository has no GitHub releases, and its
  release workflow awaits first publication. The `file://` local checkout
  is therefore **required**, exactly as the README and ecosystem page
  document — not a workaround. Re-check npm at implementation time; if
  published, prefer the tarball + integrity hash over the checkout.
- **Pin the checkout to commit
  `f63a85a7da9fa2f42b21877ab58b35d18ccaa682`** (verified genuine commit:
  merge PR #8 "Point the default MCP endpoint at mcp.fly.io",
  2026-11-13, valid signature; all plugin sources in §J read at this
  commit). Record the SHA in the provisioning manifest and assert it at
  install time — never a floating branch, per repo discipline.
- **OpenCode 1 only for the first implementation:** pin **v1.19.6**
  (latest 1.x tag observed 2026-11-24; repo now `anomalyco/opencode` —
  the `sst/opencode` name redirects). Prefer the release asset +
  published SHA256. Re-verify the install method against official docs at
  implementation (third-party sources converge on `opencode.ai/install`
  for v1 with a `--version` pin flag, but that is not yet officially
  verified — do not treat it as fact).
- **OpenCode 2 (v2.4.1) is not recommended:** beta plugin API (the
  plugin's own CONTRIBUTING says to expect v2 contract churn), renamed
  config keys, and a separate entry point. The gateway's §9 adapter
  contract stays version-agnostic so a v2 adapter can follow without
  touching Studio.
- **Runtimes:** Node.js ≥ 20 (package `engines`), Bun (plugin auto-install
  at OpenCode startup). Both must be present in the Builder Sprite image
  before OpenCode first starts.

## B. What the plugin provides (verified from source)

The plugin connects OpenCode to hosted Fly MCP at `https://mcp.fly.io`
under server name `sprites` (tool prefix `sprites_*`). From the README,
`src/shared.js`, and `src/v1.js`/`src/v2.js` structure:

- Sprite listing/creation, remote exec, remote file tools (`file_read`,
  `file_write`, `file_list`, `file_delete`, `file_rename`, `file_copy`),
  service management, checkpoints, network policy, URL/status reporting.
- `/sprites-status` (read-only connectivity check) and `/sprites-smoke`
  (list → create → exec).
- Workflow/safety guidance injected into Sprites-related sessions,
  inherited by subagents; compaction preservation of sprite names,
  service state, checkpoint IDs, pending approvals (v1 hook).
- Approval guards (`ask`) on `destroy_sprite`, `checkpoint_restore`,
  `policy_network_update` — overridable only by exact rules or
  `permissions: false`.
- OAuth via OpenCode's browser flow; restricted connector tokens with a
  configurable name prefix (default `mcp-`) and creation limits; full
  access removes all restrictions.

Do not recreate any of this in Studio. In particular, Studio keeps using
`sprites-go` (the proven `fly` boundary) for its own control-plane calls;
the plugin is OpenCode's path, not Studio's.

## C. OpenCode vs plugin vs gateway (preserved distinction)

- **OpenCode** (pinned v1.19.6): the coding agent. Reads/writes/tests the
  checkout, runs builds, asks questions, produces artifacts.
- **Official plugin** (pinned checkout): gives OpenCode Fly MCP tools for
  sprite-lifecycle operations plus safety guidance. It knows nothing of
  Studio, builds, or the business conversation.
- **Agent Gateway** (ours): owns the Studio conversation bridge — build
  assignments, session supervision via the §G adapter, §8-protocol event
  translation, authenticated callbacks, artifact collection.

The plugin's MCP connection is never the business-chat channel. Gateway↔
Studio traffic stays on the gateway plan's signed HTTPS protocol (§8);
OpenCode↔Fly traffic stays on MCP. The two channels meet only inside the
gateway process, which translates — never forwards raw streams.

## D. Execution topology (discrepancy resolved)

### The discrepancy

The plugin's own guidance text (the source of truth per the review)
states: *"OpenCode is outside each Sprite: the local workspace and shell
are not the Sprite filesystem"* (`src/shared.js`, `systemGuidance`). The
gateway plan places the gateway *and* the supervised agent process inside
the Builder Sprite. The review correctly flags: these are not obviously
interchangeable.

### Resolution

They compose. The guidance sentence scopes to sprites OpenCode touches
**via MCP tools** — and nothing in the plugin is location-dependent: it
registers a remote HTTPS MCP server, prompt guidance, commands, and
permission hooks. MCP over HTTPS works identically from inside a Sprite
(egress permitting). An OpenCode process inside Builder Sprite B therefore:

1. **Runs in B** (installed under the sprite user's home — exact path
   re-verified at implementation alongside the install method; config +
   sessions on B's persistent filesystem).
2. **Loads the plugin in B** (Bun auto-install at startup; `file://`
   checkout deployed to B at provisioning).
3. **Does repository work LOCALLY** with native tools (`read`/`edit`/
   `bash`/LSP) — its workspace *is* B's filesystem at the build checkout.
4. **Uses `sprites_*` MCP tools for lifecycle operations**: checkpoint B
   itself by name before risky work, `/sprites-status` checks, and —
   future phases only — operations on other sprites. It never round-trips
   repo files through remote file tools for work it can do locally.
5. **Is supervised by the gateway in B** (spawn/kill/reap the `opencode
   run` process; consume its NDJSON events; see §G).
6. **Persists sessions in B** (OpenCode session DB on B's filesystem),
   so post-sleep resume is a fresh process reopening the persisted
   session — consistent with the gateway plan's §11 invariant.

Every sprite OpenCode manages *via MCP* is still remote to it, so the
guidance stays true where it applies.

### Rejected alternative: OpenCode outside, Builder remote

Running OpenCode on the Studio host (or owner laptop) and driving B purely
through MCP remote tools is rejected: (a) the Studio host must not run
agent workloads (trusted-infra boundary, §3A); (b) an owner laptop is not
always-on infrastructure; (c) every file operation becomes a remote call
(latency + failure surface); (d) session state would live outside the
persistent builder, splitting the recovery story. The plugin supports this
arrangement, but our architecture must not adopt it.

### Friction 1 (handled): disable the conflicting guidance, supply our own

The plugin injects outside-topology guidance into OpenCode's system
context via the `experimental.chat.system.transform` hook (verified in
`src/v1.js` @ f63a85a: the hook appends `systemGuidance()` for activated
sessions, plus a compaction context). Overriding that text with competing
instructions would leave two authorities fighting in the system prompt —
so the plugin's guidance is **disabled**, not out-argued:

- Set the supported `guidance: false` plugin option (tuple form, §A).
  Verified effect in `src/v1.js`: disables session-activation tracking,
  the system-transform injection, and the compaction context — while MCP
  server registration, `/sprites-status` + `/sprites-smoke` commands, and
  destructive-tool permission defaults stay active (each gated by its own
  independent option).
- Supply Builder-specific instructions through OpenCode's verified
  `instructions` config option (array of instruction-file paths/globs,
  `opencode.ai/v1/docs/config/`): B ships a `builder.md` instruction file
  stating that the local workspace IS the builder checkout, local tools
  do repo work, and `sprites_*` tools are for lifecycle operations on
  named sprites only — plus the build assignment prompt per turn.
- No plugin fork: both halves use supported configuration surfaces.

Phase B acceptance asserts the resolved configuration (`opencode debug
config` — the same command the plugin's own integration tests use) shows
the plugin loaded with `guidance` disabled and our instruction file
active; the behavioral assertion stands (the agent works locally, no
stray sprite creations), backstopped by the per-builder OAuth scope (§E).

### Implications

- **Gateway**: supervises a local process (simpler than remote supervision;
  no change to §9 roles).
- **Session control/resume**: CLI-native (`--session`), DB-backed (§G).
- **Conversation bridging**: unchanged — the adapter translates NDJSON
  events to §8; Studio never sees MCP.
- **Filesystem access**: the collector reads local paths (co-resident
  benefit: no transfer protocol needed for collection; hashing before
  upload unchanged).
- **Recovery**: sleep kills `opencode run` like any non-service process;
  resume reopens the persisted session. No MCP-session continuity is
  assumed across sleep (MCP is per-invocation HTTPS + stored OAuth
  tokens).

## E. Authentication and credential boundaries

Three credential planes; each is owner-provisioned, each stays in its lane:

1. **Fly MCP OAuth** (OpenCode ↔ `https://mcp.fly.io`): browser flow,
   tokens stored by OpenCode at `~/.local/share/opencode/mcp-auth.json`
   — inside B, hence sprite-persistent across sleep/wake. Token
   expiry/refresh behavior is undocumented → `[ASSUMPTION]` with deferred
   proof (idle-then-use test in Phase B acceptance).
2. **Gateway↔Studio HMAC build keys**: unchanged from the gateway plan
   (§12). Never conflated with MCP OAuth.
3. **Model provider auth** (OpenCode ↔ its LLM): owner-provisioned at
   provisioning time (method re-verified on the pinned binary at
   implementation — e.g. `opencode auth` behavior; not yet officially
   verified). Provider choice is an `[OWNER]` decision. Keys never enter
   repos, prompts, logs, or artifacts.

### Headless OAuth bootstrap (the hard part, resolved honestly)

B has no browser; MCP OAuth is a browser flow with a loopback callback.
Verified mechanism class: OpenCode prints the authorization URL and its
`mcp auth` prompt *waits* — over SSH the documented recovery is to open
the URL in the owner's browser, authorize, and paste the dead-callback
URL (`http://127.0.0.1:<port>/callback?code=…&state=…`) back into the
waiting prompt (reported for OpenCode 2.3.0; **V1 behavior must be
confirmed on the pinned binary — prescribed proof P-E1**).

Prescribed verification ladder at implementation (stop at the first that
works; all use supported mechanisms only):
1. **P-E1:** on headless Linux with pinned v1.19.6, run `opencode mcp auth
   sprites`; record whether it prints a URL and accepts a pasted
   callback. If yes → the bootstrap ceremony is: provisioning pauses at
   the OAuth step, the owner completes the browser half, pastes back via
   the provisioning channel, tokens land in B. Owner-held, once per
   sprite.
2. **P-E2 (fallback):** port-forward the callback port (owner's
   localhost → B, via Sprite port proxy) with a fixed `callbackPort`, and
   complete the flow in the owner's browser. Supported composition, no
   invented tokens.
3. **P-E3 (last resort):** owner completes OAuth on their own machine and
   transfers `mcp-auth.json` to B as a handled secret (0600, documented,
   owner-executed). Verify the transferred grant works remotely — token
   binding behavior is undocumented.

Forbidden (unchanged): inventing tokens, CLI/API workarounds, second MCP
servers, stuffing secrets into prompts or repos. If P-E1–P-E3 all fail,
return to the owner — do not invent a fourth option.

### Scoping (correction 2: per-builder, creationless)

A broad shared prefix would let one builder's agent see and touch every
builder sharing it. The grant is therefore scoped to the assigned Builder
Sprite, enforced at two independent layers:

- **Grant layer (consent screen):** name prefix set to the assigned
  Builder's full deterministic name (names carry a tenant + random suffix
  with separator discipline, so the prefix matches exactly that sprite —
  verified semantic: the grant's prefix + label *filter* list/operate
  access, `mcp.fly.io` scope docs); Sprite creation limit
  (`max_sprites_total`) set to **zero** — Studio provisions builders, the
  agent never does. Full access is forbidden for builders.
- **Config layer (B's `opencode.json` exact rules):** `sprites_create_sprite
  → deny` (defense in depth under the zero-creation grant),
  `sprites_destroy_sprite → deny`, `sprites_policy_network_update → deny`,
  `sprites_checkpoint_restore → deny` (all restores are Studio-recorded
  via the `fly` path, per gateway §11 — OpenCode must not self-restore
  around Studio's books). `checkpoint create` stays allowed: the agent
  checkpoints B before risky work and reports the checkpoint ID as build
  evidence.
- **Scope proof (positive):** `get_fly_context` shows the connection's
  authorized bindings — the test asserts the expected prefix, the zero
  creation limit, and the consented org.
- **Negative tests (required):** from the agent's context — list shows no
  other builder or runtime sprite; operate-on-other-sprite calls fail;
  `create_sprite` is refused at both layers; destroy/restore/policy calls
  are refused. A confused or hostile agent is confined to its own builder
  by the grant itself, not by prompts.
- Which Fly org the builders live under is an `[OWNER]` decision, made at
  the OAuth consent screen during bootstrap. Ending access = disconnect
  the client + revoke the backing token (revocation also kills refresh —
  verified in the scope docs), which doubles as the builder-offboarding
  primitive.

### Gateway process identity and HMAC-key isolation (correction 3b)

The gateway plan's §12 ("key files 0600 outside the checkout") is
insufficient as stated: a `0600` file does not isolate a secret from an
agent process running **as the same Unix user** as the gateway. This
amendment corrects the boundary:

- **Requirement:** the agent's process identity must be unable to read
  gateway HMAC signing keys — by kernel-enforced discretion, not by path
  convention. Observably: an agent-context read attempt fails with
  `EACCES`.
- **Mechanism:** separate Unix users. The gateway service runs as a
  dedicated user (e.g. `studio-gw`); the agent runs as an unprivileged
  user (e.g. `studio-agent`) that shares no group with the gateway. Key
  material lives in a `0700 studio-gw`-owned directory as `0600` files,
  delivered via the provisioning channel (Studio's owner-held path —
  never through agent-visible locations). The gateway spawns agent
  processes with privilege drop and supervises them across the user
  boundary.
- **Fail-closed provisioning:** the builder's install smoke test includes
  the negative access test (agent-identity read of the key path must
  fail); if the sprite cannot provide user separation (mechanism —
  service-user support, `setpriv`/`su-exec`, or equivalent — confirmed at
  implementation against the live platform), provisioning **refuses to
  start the agent**. No single-user fallback that leaves keys readable:
  without the boundary there is no supervised agent, only an explicit
  owner exception.
- **Handling rules:** keys live in gateway memory at runtime and are never
  logged, never enter tool inputs or callback payloads, and never appear
  in diagnostics; rotation = re-provisioning (new derived keys, same
  ceremony). Cross-identity file exchange (the §G file-drop fallback, if
  adopted) uses a shared group (`studio-io` containing both users) on a
  `0770` inbox directory with `0660` files — group-scoped, never
  world-readable, never by widening key-path permissions.

This supersedes the §12 key-file sentence on merge (M7).

## F. Versioning and compatibility (pinned)

| Component | Pin (observed 2026-11-24) | Source | Re-verify rule |
|---|---|---|---|
| OpenCode | **v1.19.6** (latest 1.x tag) | release tags | confirm tag still latest 1.x; use asset SHA256 |
| Plugin | **0.1.0 @ `f63a85a7da9fa2f42b21877ab58b35d18ccaa682`** via `file://` checkout | package.json; commit verified (§A); no npm, no releases | re-check npm at implementation; if published, use tarball + integrity |
| Node.js | ≥ 20 | package `engines` | — |
| Bun | present (auto-install + smoke) | README/CONTRIBUTING | — |
| MCP endpoint | `https://mcp.fly.io`, server `sprites` | README + `shared.js` | — |
| OpenCode 2 | v2.4.1 observed; **not adopted** | release tags | revisit only by owner decision |

Additional repo note: `sst/opencode` now redirects to `anomalyco/opencode`
(verified via the API redirect this session). Install/clone references must
use the live canonical name at implementation time. The plugin's npm
dev-dependencies (`@opencode-ai/*`) are unaffected.

## G. OpenCode adapter contract (method-by-method mapping)

Basis: the **stable V1 CLI surface** (`opencode run/serve/session/mcp`,
all flags verified in `opencode.ai/v1/docs/cli/`). The HTTP API is
documented as *experimental* and is therefore the fallback, not the
foundation. No invented flags; no terminal scraping — the event source is
`--format json` ("raw JSON events"), whose exact schema is captured live
at implementation (prescribed proof P-G0: record a full NDJSON transcript
of a scripted turn and freeze the event-shape table).

| §9 method | OpenCode 1 mechanism (verified) | Gaps / proofs |
|---|---|---|
| `Start` | `opencode run --dir <checkout> --model <p/m> --agent <a> --title <t> --format json "<assignment>"`; no `--auto` (exact permission rules instead — see below) | exit-code semantics undocumented → P-G1: capture codes for success / question-yield / error / denied-tool |
| `Continue` | `opencode run --session <sessionRef> --format json "<answer>"` (`-s`/`-c` verified; sessions DB-backed, survive process death) | post-sleep resume → P-G2 (live, Phase B); whether a resumed message satisfies a pending `question` call → P-G3 (the critical proof) |
| `Events` | NDJSON stream on stdout (`--format json`); assistant text, tool calls, and results arrive as events | exact shapes → P-G0; `question`-tool event shape → P-G0b (same transcript, must show a `question` invocation) |
| `Status` | `opencode session list --format json` + supervisor process liveness; read-only, no side effects | — |
| `Cancel` | SIGTERM to the supervised `run` process, SIGKILL after grace; session stays resumable-or-failed honestly | verify no orphaned child servers → P-G4 |
| `Collect` | local filesystem reads (co-resident) + `git diff`; test reports as files the assignment requires | — |

**Deliberately not used at first:** `opencode serve` + `--attach` (avoids
per-run MCP cold boot, but adds a second supervised process with its own
sleep/recovery story). Start cold-boot-per-run; adopt serve+attach only if
measured cold-boot latency binds — a performance optimization, never a
correctness dependency. Serve's basic-auth flags (`--password`,
`OPENCODE_SERVER_PASSWORD`) are then the documented mechanism, verified in
the CLI reference.

**Permission posture for builds** (exact rules, no `--auto` blanket):
`bash`/`edit`/`read`/dev-tools allowed within the checkout;
`external_directory` denied outside it; `sprites_destroy_sprite`,
`sprites_checkpoint_restore`, `sprites_policy_network_update` denied;
`sprites_checkpoint_create` allowed (evidence). `--auto` is rejected for
builder runs because it auto-approves anything not explicitly denied —
the opposite of least privilege; exact rules fail closed on new tools.

### Friction 2 (the critical unknown): `question` under `run`

The `question` tool exists and is documented (header + text + options;
user picks or types custom). What is **not** documented is what happens
when it fires inside non-interactive `opencode run`: does the run exit
with the question in its events, block forever, or error? The entire
question→answer→resume acceptance hinges on this.

- **Primary path (P-G3):** if the run surfaces the `question` invocation
  as JSON events and then exits (or waits killably), the adapter ends the
  turn, emits `agent.question`, and resumes with the user's answer via
  `--session`. Proof must show the resumed run *consuming* the answer as
  the question's resolution — not merely continuing the transcript.
- **Fallback path (file-drop handshake):** if the primary fails, the
  assignment instructs the agent to write questions as JSON to the
  group-scoped inbox (§E: `0770`, `studio-io` group — the two process
  identities exchange here, never through key paths) at
  `question.<turn>.json` and exit 0 (`completed-awaiting-input` is
  reported as `agent.question`, never as success); Studio answers land
  in `answer.<turn>.json` via the gateway, and resume tells the agent to
  read the file first. Deterministic, testable, no parsing of prose.
  This is a *tested adapter boundary*, not stdout scraping: fixed paths,
  JSON schemas, turn binding.
- **Stop rule:** if neither path proves out live, stop and return to the
  owner. Do not parse free prose for questions, do not invent flags, do
  not switch silently to the experimental HTTP API or PTY driving.

## H. Acceptance proof (OpenCode track)

The gateway plan's Phase C 15-step bridge stays the bar; this track makes
it OpenCode-concrete. Each step names its mechanism and proof artifact:

1. Studio creates a build assignment (existing §7 rows).
2. Studio wakes/provisions Builder Sprite B (existing `fly` path).
3. B starts pinned OpenCode v1.19.6 (`opencode --version` recorded).
4. The pinned plugin checkout loads (`file://` config; `opencode mcp
   list` shows `sprites`; guidance present).
5. OpenCode verifies authenticated connectivity (`/sprites-status` or
   `list_sprites` via a scripted run; empty list accepted as success).
6. OpenCode inspects the assigned repo in B **locally** (native tools;
   no stray sprite creations — asserted).
7. Progress events reach Studio (`agent.progress` via NDJSON → §8).
8. OpenCode asks a real clarification question (`question` tool or
   file-drop per §G; `agent.question` in Studio).
9. Studio displays it in the original conversation (parked turn row).
10. The user answers (idempotent answer post).
11. Studio forwards to the same session (`continue` → `--session`).
12. OpenCode resumes and consumes the answer (**the critical condition**:
    transcript must show resolution, not mere continuation).
13. OpenCode generates a small artifact and runs tests in B (exit codes
    recorded; failures honest).
14. Result + evidence reach Studio (`artifact_ready` + verified fetch).
15. Studio displays the result in the original conversation.

Plus the prescribed proofs: P-G0/G0b (event shapes), P-G1 (exit codes),
P-G2 (post-sleep resume), P-G3 (question→answer→resume), P-G4 (clean
cancel), P-E1–P-E3 (OAuth bootstrap), and the interruption/sleep-wake/
duplicate/reconnect matrix from gateway §11 — now executed against the
real OpenCode binary. A passing `/sprites-status` alone proves nothing
about the bridge and is explicitly insufficient.

Review-correction acceptance additions (all required before merge sign-off
converts to implementation authorization):

- **C1 (guidance):** `opencode debug config` on B shows the plugin loaded
  with `guidance: false` and the Builder instruction file active; a
  scripted turn touching `sprites_*` tools produces no outside-topology
  guidance in the effective system prompt (asserted from the recorded
  transcript/config, not from agent self-report).
- **C2 (scope):** `get_fly_context` shows the expected per-builder prefix,
  zero creation limit, and consented org; negative tests prove the agent
  sees and touches only its assigned builder (no other builder/runtime in
  listings; operate/create/destroy refused at both layers).
- **C3a (pin):** the deployed checkout's HEAD equals `f63a85a…` (asserted
  by the provisioning manifest check).
- **C3b (identity boundary):** agent-identity read of the HMAC key path
  fails `EACCES`; gateway-identity operations succeed; provisioning
  refuses to start the agent when user separation is unavailable
  (fail-closed path demonstrated, not just coded).

## I. Future agent portability (unchanged)

OpenCode is the first and primary implementation. The §9 adapter contract
is untouched by this amendment: a second agent adds a second
implementation of the same interface, with its own capability matrix and
proof track. No second adapter and no multi-agent framework are planned.
The file-drop fallback (§G), if adopted, is specified per-adapter — it
must not harden into a pseudo-standard that later agents are forced into.

## Scope preservation (review requirement 6)

This amendment changes no standing position: F1 stays deferred (gateway
mutual auth remains purpose-built HMAC, operator sessions unchanged);
G-01 stays open (business-effect tools stay gated; OpenCode work is
orchestration + artifacts only); P5 scope is untouched; Invoice Review
Desk remains the golden path; Lead-to-Booking remains the candidate
second demonstration. No locked gateway decision is altered — M1 records
consistency, not change.

## Merge list (exact; applied only on owner approval)

| # | Gateway plan target | Change |
|---|---|---|
| M1 | §3 locked decisions | Append: OpenCode designated initial agent (this amendment); inside-topology composition (§D) recorded as consistent with locked §2C — no locked-decision change. |
| M2 | §4 verified capabilities | Append: OpenCode/plugin verified facts table (§F) + repo-rename note. |
| M3 | §5 components | Name `gateway/opencode_adapter.go` as the first §9 implementation (replacing `first_adapter.go`). |
| M4 | §9 adapter contract | Append the §G method mapping + capability-matrix seed (verified cells filled, P-proofs listed as unknown); file-drop inbox is group-scoped per §E. |
| M5 | §10 provisioning | Append: OpenCode install (pinned), plugin checkout deploy **at `f63a85a…` (asserted)**, `opencode.json` contents (exact permission rules incl. `create_sprite → deny`, `guidance: false` tuple, Builder instruction file), OAuth bootstrap ceremony (§E ladder), dual-user creation + fail-closed smoke. |
| M6 | §11 recovery | Append: session-DB resume specifics, serve-not-used rationale, OAuth-token persistence + idle proof. |
| M7 | §12 security | Append: three credential planes, **per-builder creationless grant + negative tests**, destructive-deny rules, model-auth owner decision; **supersede the key-file sentence with the separate-user HMAC boundary + `EACCES` test**. |
| M8 | §§18–19 acceptance | Append the §H OpenCode proof track to Phases B/C/D criteria; add P-E/P-G proofs **and C1–C3b correction proofs** to the required-evidence lists. |
| M9 | §20 dependencies | Resolve item 3 (first agent: OpenCode v1.19.6 + plugin checkout @ `f63a85a…`); update item 4/5 notes to point at §H proofs. |
| M10 | §21 risks | Add: `question`-under-`run` unknown (with stop rule), OAuth bootstrap ladder, npm-unpublished/plugin-drift, repo rename, sprite multi-user support (fail-closed). |
| M11 | New changelog entry | Text below. |

## Proposed changelog text

> **2026-11-24 — OpenCode amendment merged (owner-approved):** OpenCode
> v1.19.6 designated the initial coding agent with official plugin
> checkout 0.1.0 @ `f63a85a…` (asserted); inside-Builder topology confirmed
> compatible with the plugin's outside-assumption, with plugin guidance
> disabled (`guidance: false`) and Builder instructions supplied via the
> verified `instructions` option (§D); adapter mapped to the stable V1 CLI
> with prescribed proofs P-G0–P-G4 and P-E1–P-E3; OAuth bootstrap ceremony,
> per-builder creationless grants, and exact permission rules specified;
> gateway/agent split into separate Unix users with an `EACCES` negative
> test (§E); question→answer→resume round trip named the critical
> acceptance with a file-drop fallback and stop rule. No locked decision
> changed. (Re-verified install facts filled at merge.)

## J. Sources (all accessed 2026-11-24)

Official (normative):
- Plugin README @ main (blob `a3e1b75`, 12234 B) — install, OAuth,
  commands, guards, options, restricted tokens, run/ask rule.
- Plugin `package.json` 0.1.0 (blob `15ac243`) — exports, engines,
  devDeps.
- Plugin `src/shared.js` (blob `9b06bcc`) — guidance text (outside-topology
  sentence), tool names, risky tools, option parsing.
- Plugin `src/v1.js` @ `f63a85a…` — `guidance` option gates the
  `experimental.chat.system.transform` injection, session activation, and
  compaction context; MCP/commands/permissions independently gated.
- Commit `f63a85a…` verified genuine via the commits API (merge PR #8,
  2026-11-13, valid signature).
- `mcp.fly.io` scope guide — consent-screen name prefix + Sprite limit,
  `get_fly_context` bindings, prefix/label filtering, no-expansion on
  retry, revocation kills refresh.
- `opencode.ai/v1/docs/config/` — `instructions` option (instruction-file
  paths/globs); config precedence.
- Plugin CONTRIBUTING.md (blob `0302eb3`) — layout, tests, release
  process (first publication pending).
- npm registry `GET @flydotio/sprites-opencode-plugin` → **404**
  (unpublished); GitHub releases → `[]` (none).
- `fly.io/sprites/ecosystem/` — file:// fallback until npm; Bun
  auto-install; browser OAuth steps.
- `opencode.ai/v1/docs/cli/` — `run` flags (incl. `--format json`,
  `--session`, `--continue`, `--auto`, `--attach`), `serve` + basic
  auth, `session list/delete`, `mcp` commands.
- `opencode.ai/v1/docs/tools/` — `question` tool contract.
- `opencode.ai/v1/docs/permissions/` — allow/ask/deny, granular rules,
  `--auto`.
- `opencode.ai/v1/docs/mcp-servers/` — OAuth auto-flow, tokens at
  `~/.local/share/opencode/mcp-auth.json`, `mcp auth/logout`.
- `opencode.ai/docs/cli/` (V2) — background service model, `debug paths
  db`, uninstall preserves sessions.
- `opencode.ai/docs/api` (V2) — HTTP API labeled **experimental** (178
  ops; session/inbox/permission tags noted, not adopted).
- Release tags via API (`anomalyco/opencode`, redirect from
  `sst/opencode` observed): latest 1.x **v1.19.6**, latest 2.x v2.4.1;
  assets carry SHA256 digests.

Third-party (corroborative only, never normative):
- everpod.ai OpenCode 2.3.0 MCP guide — SSH copy-back passage; install
  lines (`/install` = v1) — the latter re-verified-or-dropped at
  implementation.
- openworklabs/ravenna/openstatus MCP guides — loopback-callback
  mechanics, `callbackPort`/`redirectUri` options (confirm on pinned V1).
- Agent skill repos — convergent `run`/`serve` flag listings (all
  re-verified against official V1 CLI docs above before use).
