# Phase D Implementation Plan — Miniapp Artifact Pipeline and Verification

Status: proposed, planning only. No code, migrations, provisioning,
commits, or pushes were performed to produce it. Source prompt:
`plans/prompt22.md`. Authorizing plan: `plans/prompt22_plan.md`
(owner-approved S0–S3). Phase D implementation requires a separate
explicit GO after architectural acceptance of this document.

Evidence labels: `[VERIFIED path]` read in the current checkout;
`[PROPOSAL]` new design; `[ASSUMPTION]` believed true with deferred
proof named; `[OWNER]` needs an explicit owner decision.

## 1. Executive architectural decision

Phase D builds the first complete, verifiable Miniapp artifact
lifecycle for the Invoice Review Desk, ending at a durable
digest-bound human decision. Three load-bearing decisions:

1. **Artifact identity is a sha256 digest over canonical bytes,
   enforced by re-hash at every use.** Transport reuses the
   existing manifest fetch protocol (per-file digests from
   gateway `Collect`); no new archive format is introduced, so
   the zip-slip/bomb class cannot occur. Bytes land in a
   content-addressed store; verify, test, and decide each
   re-hash from stored bytes and compare against the pinned
   identity before acting (the `runner.Admit`/`Revalidate`
   pattern lifted from specs to whole artifacts). Any byte
   change yields a new identity and restarts the pipeline.
2. **Untrusted Go executes only in a disposable Docker
   container or not at all.** Docker is present on this machine
   and already a project dependency (PG19 testing). The test
   runner uses a pinned toolchain image, no network, PID/memory/
   CPU limits, read-only rootfs, scrubbed environment, timeout
   with process-tree kill, capped redacted output. When Docker
   is unavailable the result is an honest blocked state, never
   fabricated success; only Studio-authored fixtures may run on
   the host. No new sandbox is invented.
3. **Approval is a human-only control-plane operation bound to
   the exact digest.** `artifact.decide` is deliberately not a
   registry tool (the `console_stop.go` kill-switch precedent):
   Console POST with operator session + Origin check, tenant
   derived server-side, idempotent per artifact, unique decision
   per artifact digest. Agents cannot reach it through tools,
   MCP, or Gateway commands; no test success or agent claim
   implies authorization.

The D0–D6 sequence from the prompt is adopted unchanged: each
unit's dependency is evidenced in §14. No Phase E work.

## 2. Current repository inventory with exact paths

Product code (all paths repo-relative, verified at HEAD
`5942fef`):

- `spec/` — MiniappSpec v1 (`types.go`, `SchemaVersion =
  "miniappspec/v1"`), deterministic strict validator
  (`validate.go`, 13-code taxonomy, secret-field scan, declared-
  hash check), canonical JSON + `ContentHash` (`hash.go`).
- `runner/` — admission boundary (`admission.go`: `Admit`
  validates + tenant-binds + hash-pins + retains raw bytes;
  `Revalidate` replays before execution), strict interpreter
  (`interpreter.go`), fixture dry-run (`dryrun.go`).
- `tools/` — 25 registered Studio tools (`register.go` manifest
  with JSON schemas, entity keys, ClassA/SubstrateGo,
  TrustAutonomous, profile digests); spec/app persistence
  (`appspecs.go` → `studio.app_specs` versioned + hashed);
  conversation/build/question flows; gateway event ingest
  (`ingest.go`, writes claimed artifact rows); idempotency
  key + payload-fingerprint machinery; secret scrubbing
  (`scrub.go`).
- `gateway/` — Builder-side service: signed command server
  (`server.go`, incl. `fetch_artifact` handler),
  hash-pinned artifact collector (`collector.go`: `os.OpenRoot`,
  no-follow, regular-file-only, 5 MB/file + 50 MB total caps),
  supervisor/session/outbox/adapter/filedrop, Studio client
  (`client.go`, no fetch call).
- `server/` — session auth + CSRF/Origin, Focus Console
  (`handlers_console.go`, `templates/focus/`, placeholder map
  for spec/validate/connect/deploy), Q&A decide + kill
  operator-only handlers, SSE + resume, sender dispatch loop
  (`sender.go`: assign/continue/cancel; fetch falls through to
  terminal "does not drive"), gateway callbacks, tenant pools.
- `store/` — migrations `000001`–`000020`, RLS FORCE
  everywhere, `studio_app`/`studio_lookup` roles, session
  SECURITY DEFINER functions, schema gate.
- `apps/` — slug validation + deployment state machine
  (P3-era, fake-backed; Phase E territory, untouched).
- `testdb/` — run-scoped test databases, `studio_app_test`
  least-privilege mirror, endpoint allowlist (loopback + PG19).
- `e2e/boundary_b/` — ledger-driven browser matrix (phases +
  sabotage gate), reusable for D6 rehearsal shape.
- Framework (read-only, pin `73351a8`): `contract/` manifest /
  `Digest` (`sha256:<64hex>`) / `ProfileDigest` /
  `ReleaseEnvelope.Matches`; `Service.Invoke/Status/
  DecideApproval` (`runtime/`); three substrates admitted
  (`go`, `river`, `dbos`) — Studio must enforce Go-only.

Key boundary precedents to reuse verbatim: `runner.Admit` /
`Revalidate` (verify-at-admission + re-verify-at-use);
`gateway.Collect` (descriptor-pinned, capped collection);
`server/console_stop.go` (operator-only by non-registration);
`testdb` guards (fail-closed allowlists).

## 3. Existing schema and capability assessment

Present and reusable (all tenant-scoped, RLS ENABLED + FORCE):

- `studio.artifacts` (000014): `(tenant_id, id)` PK,
  `UNIQUE (tenant_id, build_id, digest)`, kind CHECK
  (`diff|spec|test_report|bundle`), `manifest` JSONB,
  `verified_at` (always NULL today — no setter exists),
  `storage_ref` (never written; bytes have no home).
- `studio.approvals` (000014): question-grant rows written by
  `question.answer` (`tools/question.go:114`); `artifact_id`
  exists but is never set; no digest, FK, or evidence columns.
- `studio.app_specs`: versioned canonical specs with
  `spec_hash` (`spec.save/get` tools).
- `studio.builds/turns/questions/commands/events`: lifecycle
  + Q&A + sender retry source of truth; `commands.kind`
  admits `fetch_artifact` but nothing sends it.

Discrepancies between prior plans and source (logged, not
reopened — none violates an invariant):

- Gateway plan §7 `messages.seq`/`events.seq` cursors →
  superseded by D1 `conversation_stream` (recorded in 000014).
- Gateway plan §7/§15/§19 `deployments_gateway` → never
  created; `studio.deployments` (P3) stands instead.
- Gateway plan §13/§6 "approvals reuse framework
  `DecideApproval`" → Studio uses native `studio.approvals`
  rows; `DecideApproval` is uncalled in Studio. Phase D
  builds on the native rows (digest binding is Studio-side).
- Gateway plan §19 `server/review.go`,
  `tools/build_review.go`, finalized fetch → absent; this is
  Phase D scope, not drift.
- `roadmap4.md` status/scope text (§Status, §4 NEXT, §7, §8)
  predates accepted C/Boundary B/U5.12; its Phase D product
  intent and invariants still govern.
- `prompt21-boundary-b.md` U5.12-HOLD notes are historical;
  U5.12 + U5.12.1 are accepted.
- Ingest records artifact rows from agent CLAIMS
  (`tools/ingest.go` `persistArtifacts`, malformed entries
  skipped, manifest kept in the event row); the code comment
  already requires "unverified until proven, and the UI must
  say so." Phase D defines the proof.

Gaps Phase D must close: byte storage; verified_at setter
semantics; fetch sender; verification/test evidence rows;
digest-bound decision writer + human-only enforcement.

## 4. End-to-end artifact lifecycle

One build, one artifact lineage. States derive from rows, never
memory:

1. **Claim.** Agent emits `agent.artifact_ready` (manifest of
   path/kind/digest/size). Ingest persists one `artifacts` row
   per well-formed entry (`verified_at` NULL). Malformed
   entries skip with the manifest preserved forensically.
2. **Fetch.** Studio sender dispatches `fetch_artifact` with
   the claimed manifest. Gateway `Collect`s (hash-pinned) and
   returns bytes+digests. Studio re-hashes every file, rejects
   any mismatch, stores bytes content-addressed. Retry/backoff
   and dedup ride the existing sender machinery.
3. **Verify.** `artifact.verify` (deterministic, zero code
   execution): identity, contract version, spec validity,
   Go/Miniapp compliance, structure, dependency policy, path
   safety, tenant/build consistency. Pass sets `verified_at`
   and writes structured evidence; fail writes evidence and
   leaves the row unverified. Only verified rows advance.
4. **Test.** `artifact.test` builds + tests the pinned bytes
   in a disposable container (or records honest blocked).
   Machine-readable report persisted; pass/fail is explicit.
5. **Review.** Console Spec/Validate/Evidence scenes render
   spec + verification + test evidence from rows. Unified
   diff for `diff` kinds; refusal reasons verbatim.
6. **Decide.** Human approves/rejects the exact digest via
   the operator-only Console action. One decision per
   artifact; approval records digest + approver + evidence
   refs. Rejection returns reasons as a `continue` payload
   (rework loop); new bytes restart at Claim with a new id.
7. **Terminal.** Approved digests wait for Phase E. Nothing
   in Phase D deploys, provisions Runtimes, or executes the
   app outside the test sandbox. A test asserts no route or
   tool deploys from Phase D state.

## 5. Versioned artifact contract

`[PROPOSAL]` Contract version `studio-artifact/v1`, carried in
every manifest and pinned in code. No archive format: transport
is the existing manifest fetch (path/kind/digest/size per file,
bytes base64 in the response). This reuses the proven
`gateway.Collect` threat model and excludes archive classes
(zip-slip, bombs, link entries, duplicate paths) by
construction. Rejected alternative: tar/zip bundle — adds an
extraction threat surface for no capability gain.

Bundle layout (path conventions inside the manifest):

- `miniappspec.json` — the canonical MiniappSpec document
  (required for `bundle`; the `spec` kind is this file alone).
- `app/go.mod`, `app/**/*.go` — Go module + sources.
- `app/**/*_test.go` — Go tests (required: an untested
  bundle cannot report green).
- `manifest.json` — Studio envelope: contract version,
  tenant/build/conversation/turn correlation, provenance
  (source revision, builder identity), member file list
  with digests, toolchain declaration.
- `minimapp.json` — the framework `MiniappManifest`
  (tool list with substrates, schemas, trust modes),
  validated by `contract.Manifest.Validate`. Distinct
  from the Studio envelope by role: the envelope binds
  the bundle, the framework manifest declares the app.
- `diff/*`, `test_report/*` — `diff` and `test_report`
  kinds follow the same pinning.

Identity: artifact id = `sha256:` hex over canonical JSON of
`{contract, tenant_id, build_id, files:[{path, digest}], }`
with paths sorted — a Merkle-style root over gateway-pinned
file digests. Studio recomputes it from stored bytes; any
mismatch refuses. File digests use the `contract.Digest`
shape (`sha256:<64hex>`, validated).

Row model: one decision unit = one `bundle`-kind row whose
digest is the root. Member files live as bytes in the blob
(§10), evidenced per file; they are not rows, so the kind
enum needs no extension. Standalone `spec`/`diff`/
`test_report` rows remain auxiliary evidence — only
`bundle` rows are decidable (§9).

Spec↔source binding (single truth, no competition):
MiniappSpec is canonical for WHAT the app is; Go source is the
implementation. `artifact.verify` enforces the declared correspondence:
`app.id` equal across spec, both manifests, and module path
(`studio-artifact/<tenant>/<app>` pattern); every
`capabilities.operations[].id` named by a `minimapp.json`
tool of equal name; `Substrate == "go"` on every declared
tool (the framework admits `river`/`dbos` — Studio refuses
them here); `appVersion` consistent; embedded spec hash
equals a `studio.app_specs` saved version (or the bundle
carries the new version being proposed, which verify
validates and the review flow saves on approval — decided
at implementation with the D1 evidence). The behavioral
half — declared tools actually registered by the built
code — is proven at test time (§8 registration
conformance), never statically.

Toolchain/dependencies: `go.mod` must declare exactly the
allowlist — Go stdlib plus `github.com/PithomLabs/miniapp`
at the pinned SHA (+ its go.sum closure, pinned). No
`replace` directives, no second module, no non-Go sources
(`*.py`, `*.js`, `package.json`, `node_modules`, etc.
refuse). Size caps reuse the proven 5 MB/file, 50 MB/total
(`gateway/collector.go`).

Reproducibility: identity + toolchain pin + dependency pins
fully determine the bytes; `artifact.test` re-hashes inputs
before running and records input digests in the report.

## 6. Artifact retrieval and admission security design

Threat posture: agent, manifest, bytes, and paths are
untrusted. Studio trusts only what it re-hashes and
re-validates.

Retrieval (D1): extend `server/sender.go` with a
`fetch_artifact` case (today it terminal-settles "does not
drive kind"). Payload = manifest entries from the CLAIMED
artifact rows. Retry/backoff/dedup/reconcile reuse the sender
loop (rejected: a dedicated fetch worker — it would duplicate
retry/dedup/reconcile the sender already owns); gateway
`Collect` re-pins bytes server-side. Studio re-hashes each
file against both the gateway digest and the ingest claim;
any mismatch fails the fetch honestly (retry re-fetches; no
partial bundle advances). Bytes land in the content store
keyed by digest; the artifact row flips from claimed to
fetched by byte presence (no new status enum —
derived state).

Admission (pre-execution, all inside `artifact.verify`, zero
code execution):

- Paths: rooted, cleaned, no absolute/traversal/empty;
  refused set includes dotfiles outside the layout, control
  characters, case-collision pairs, and any path outside the
  §5 layout.
- No links: fetch carries no link entries; verify re-checks
  stored file metadata for non-regular types.
- Sizes: per-file and total caps re-checked on stored bytes
  (gateway caps are defense in depth, not authority).
- `go.mod`: single module, required path pattern, exactly
  one direct require (miniapp at the pin; indirect requires are
  tidy's normal output and carry no authority), no replace, no
  retract ambiguity; `go.sum` must carry the two pinned miniapp
  lines (D2 correction: static transitive-closure pinning adds
  no control beyond the requires-allowlist, the import check,
  and the offline sandbox build, so it was dropped).
- Imports: static walk with `go/parser` over stored sources
  (never `go list` on untrusted code — no toolchain
  execution pre-verify); every import must resolve to stdlib
  or the pinned miniapp module. `unsafe`, `plugin`, `net/*`
  server listeners, `os/exec`, and cgo (`import "C"`) refuse:
  the test sandbox would contain them, but policy refuses
  them first so a sandbox bug cannot be the only control.
- Secrets: `spec.scanSecrets` rules plus `tools/scrub.go`
  patterns over every file and the manifest; bearer-shaped
  values refuse (admission, not redaction — secrets must
  never enter the artifact). Both helpers are currently
  unexported — D2 exports or relocates them; hand-duplicated
  rule lists are forbidden.
- Tenant/build ownership: manifest correlation must equal
  the row owners; mismatches refuse before any byte is
  trusted.
- Digest binding: recomputed artifact id must equal the
  claimed digest; `UNIQUE (tenant_id, build_id, digest)`
  makes re-fetch idempotent.

Anything failing admission writes structured evidence and
stops. Malformed vs. malicious is not distinguished for
advancement — both refuse — but evidence records the class
for the operator.

## 7. `artifact.verify` contract

`[PROPOSAL]` Deterministic Studio tool, registered in
`tools/register.go` (agent-callable: read-only self-service —
an agent may pre-check its own artifact; harmless because
verification confers no authority and is tenant/build-scoped).
Input: `{artifact_id}` (+ idempotency key per form rules).
Effects, in order:

1. Load the artifact row + stored bytes; re-hash; require the
   recomputed identity to equal the pinned digest (else:
   `E_DIGEST`, terminal for this input).
2. Require contract `studio-artifact/v1` (else `E_CONTRACT`).
3. Validate the embedded spec with `spec.Validate`
   (existing taxonomy surfaces verbatim) + hash-bind to
   `studio.app_specs` per §5.
4. Enforce §6 admission (structure, go.mod, imports,
   secrets, paths, ownership).
5. On pass: set `verified_at`, write the verification
   evidence row (checks with per-check pass/fail + digests).
   On fail: write evidence, leave `verified_at` NULL.

Output: machine-readable `{verified, artifact_id, digest,
checks[], evidence_id}`. No code execution, no network, no
side effects beyond evidence rows. Concurrent duplicate
verifies are idempotent (UNIQUE evidence per artifact +
check-set version); verify-after-new-bytes is impossible —
new bytes are a new artifact id.

## 8. `artifact.test` isolation architecture

`[PROPOSAL]` NOT a registry tool: operator/build-initiated
only (Console review action; cost + sandbox access are not
agent-self-service). Input: verified `artifact_id` only —
unverified rows refuse before any execution. Flow:

1. Re-hash stored bytes against the pinned digest; load the
   toolchain declaration; require it to equal the pinned
   toolchain (else refuse: evidence, no run).
2. Materialize bytes into a fresh tmpfs workdir (0600/0700,
   Studio-owned), never onto shared or host paths.
3. Run in a disposable Docker container, all mandatory:
   `--network=none`; `--pids-limit` + `--memory` + `--cpus`;
   read-only rootfs with only the workdir writable;
   environment fully scrubbed (no host env, no `HOME`
   secrets, `GOPROXY=off`, `GOFLAGS=-mod=mod`,
   `GOCACHE` on tmpfs); image pinned by digest and
   pre-seeded with exactly the toolchain + miniapp pin +
   go.sum closure; wall-clock timeout with container kill
   and process-tree reaping; stdout/stderr capped
   (per-stream byte cap + total cap, truncation noted);
   output passed through secret redaction before
   persistence.
4. Parse `go test -json` into the machine-readable report
   (pass/fail counts, per-test outcomes, failures listed);
   any parse failure, timeout, OOM/kill, or non-zero exit
   records FAILED with the evidence — never green.
   Registration conformance runs here too: the harness
   registers the built artifact in-container and requires
   the registered tool names to cover the spec's declared
   operations (the behavioral half of §5 binding).
5. Record the test-run row (status lifecycle
   queued→running→passed/failed/blocked, never an
   indeterminate terminal without its cause), then destroy
   the container and wipe the workdir — including on crash
   paths (deferred cleanup + orphan reaper keyed by run id).

Honest blocked state (hard boundary): if Docker is absent,
the image digest cannot be verified, or container creation
fails, the run records `blocked` with the cause and the
Console renders blocked with the cause. Trusted repository
fixtures (Studio-authored, reviewed, pinned in-tree) may run
on the host `go test` as the local capability proof; agent
bytes never execute on the host under any flag.

Rejected alternative: bubblewrap/userns sandbox — present
on this machine but every namespace/seccomp/cgroup rule
would be Studio-authored; Docker's flags are battle-tested
and Docker is already a project dependency. Revisit only
with measured cause.

Machine facts (read-only survey, nothing started): docker
present, bwrap present, unshare present, firejail absent,
host Go 1.27.3. `[ASSUMPTION]` Docker daemon reachable from
the Studio runtime with container-create rights — deferred
proof: D3 negative tests (no-network, no-host-fs, env-scrub,
cap-enforcement, kill-cleanup) must pass on the real
daemon, else the blocked state is the shipped behavior.

## 9. `artifact.decide` authorization and persistence contract

`[PROPOSAL]` Human-only Studio control-plane operation
following `server/console_stop.go` exactly:

- Not in `tools/register.go` (the manifest is the
  agent-callable surface); no Gateway command; no MCP path.
  A test asserts registry absence.
- Console `POST /console/artifact-decide` (distinct from
  `POST /console/decide`, the Q&A path) with
  `requireSession(requireOrigin(...))`, tenant derived from
  the operator session, never from input.
- Input: `{artifact_id, decision, note?}` + per-form
  idempotency key (fingerprint-bound, P4 pattern). The
  handler re-loads the artifact row, requires kind
  `bundle`, requires `verified_at` IS NOT NULL, re-hashes
  stored bytes against the pinned digest, and requires a
  terminal test run. Approve additionally requires the run
  passed (failed/blocked approve refuses, per gateway plan
  §14 rule 5); reject is allowed on any terminal run.
- Persistence: one `studio.approvals` row per artifact
  (`UNIQUE (tenant_id, artifact_id)`): decision,
  `decided_by` = operator binding (human identity, not an
  agent id), digest, verification + test-run evidence refs,
  timestamp. Concurrent duplicate decisions converge to one
  winner (constraint + idempotent replay); a second decision
  with a different outcome conflicts (409), never
  overwrites. Rejection writes reasons and returns them as
  a `continue` payload (rework); new bytes require a new
  artifact and a new decision — approvals never transfer.
- An agent proposal, completed build, or green test report
  changes nothing about authorization: only this handler
  writes artifact decisions, and only a human session
  reaches it.

## 10. Migration and RLS impact

`[PROPOSAL]` One migration, `000021_artifact_pipeline`
(up/down), on top of 000020. No conversation/Q&A table is
touched; no durable functionality is duplicated.

- `studio.artifact_blobs (tenant_id, digest, bytes BYTEA,
  size_bytes, created_at)` — content-addressed byte store,
  one blob per bundle holding the canonical bundle encoding
  (sorted members with boundaries; the exact encoding is a
  D0 deliverable), PK `(tenant_id, digest)`, `CHECK
  (size_bytes <= 52428800)` total-cap guardrail (per-file
  5 MB enforced in code where file boundaries exist).
  `studio.artifacts.storage_ref` becomes NULLABLE with a
  `''`→NULL backfill, then FK → `(tenant_id, digest)`
  (NULL skips the check; pre-pipeline claims stay NULL,
  never trusted).
- `studio.artifact_verifications (id, tenant_id,
  artifact_id, check_set_version, passed, checks JSONB,
  created_at)` — UNIQUE `(tenant_id, artifact_id,
  check_set_version)` for idempotent re-verify; FK to
  artifacts CASCADE.
- `studio.artifact_test_runs (id, tenant_id, artifact_id,
  status, input_digest, report JSONB, evidence_id,
  created_at, finished_at)` — status CHECK
  (`queued|running|passed|failed|blocked`); FK CASCADE.
  Cleanup/audit columns ride the report JSONB, not new
  columns.
- `studio.approvals` ALTER: ADD `artifact_digest TEXT NOT
  NULL DEFAULT ''`, `verification_id TEXT NOT NULL DEFAULT
  ''`, `test_run_id TEXT NOT NULL DEFAULT ''`;
  ADD `UNIQUE (tenant_id, artifact_id)` (question-grant
  rows keep `artifact_id = ''` — the constraint must be
  partial: `WHERE artifact_id <> ''`, else the two grant
  shapes collide).
- RLS: ENABLE + FORCE + tenant_isolation policies on all
  three new tables (mirror 000014); `GRANT SELECT, INSERT,
  UPDATE, DELETE` to `studio_app`; extend the `testdb`
  mirror + `TestPrivilegeParity` in the same change.
- Retention: evidence rows are permanent review records
  (CASCADE with the build); blobs CASCADE with artifacts.
  No auto-delete; storage bounded by caps.

Concurrency/crash: blob write is idempotent (PK on
digest); verification evidence is idempotent per check-set
version; test-run status advances queued→running→terminal
under the run-id claim (crash mid-run reconciles to
failed-with-cause on next touch, never silently green);
decision uniqueness converges concurrent submits.

## 11. Focus Console changes

Minimal scene deltas on `server/handlers_console.go`,
`templates/focus/`, `static/focus/console.js`; spine
honesty preserved (`buildSpine` gains proven/refused only
from the §10 rows):

- **Spec scene** (replaces placeholder): canonical spec
  from `studio.app_specs` (plain-language sections; DEV
  `<details>` raw JSON + content hash + version). Empty
  state until a spec version exists.
- **Validate scene**: verification result (per-check
  pass/fail, digests) + test-run status with the
  machine-readable report (counts, failures listed) +
  refusal evidence verbatim. Blocked renders blocked with
  the cause. Nothing green without rows.
- **Connect scene**: the app's declared capability→
  connection bindings from the verified spec (catalog +
  scope display only). G-01 preserved: no production
  credentials, no live integration actions; any refused
  scope renders refused with the policy reason.
- **Evidence scene**: gate ledger bound to real rows —
  claim/fetched/verified/tested/decided per artifact with
  digest prefixes, approver, timestamps (the G0–G12 shape
  from the prototype, populated only from evidence rows;
  gates without rows render pending, never verified).
- **Pulse**: artifact_ready / verified / test
  passed-failed-blocked / decided receipts; approval
  request card with inline Approve/Reject bound to the
  exact digest (DEV shows digest + evidence ids).
- **New action**: `POST /console/artifact-decide`
  (decide form with idempotency key); rejection note
  field; conflict (409) and stale-digest renders.
- Deploy stays a Phase E placeholder; Run keeps its
  Boundary B honest-activity rendering (builder activity
  only, product Run unproven until Phase E). OPS/DEV
  remain density modes over identical data. Mobile
  slide-over and keyboard/a11y patterns follow the
  Boundary B baseline.

## 12. Threat model and failure/recovery analysis

Actors: operator (trusted, authenticated); Studio backend
(trusted); agent/Builder (untrusted execution, trusted
nothing); artifact bytes/manifests/paths (adversarial
input); test container (hostile code, contained).

Boundaries and controls:

- Agent→Studio: signed protocol only; claims never trusted
  (§6 re-hash + re-validate); tenant/build re-derived.
- Studio→sandbox: verified bytes only; scrubbed env; no
  network; caps; timeout+kill; redacted capped output.
- Human→Studio: session + Origin + server-side tenant;
  digest-bound idempotent decisions.
- Secrets: refused at admission (§6); redacted in test
  output; never in specs, logs, manifests, or approvals.

Failure dispositions (prompt22 §10, each with durable
honest semantics):

- Upload interrupted / retrieval failure / gateway down →
  no agent→Studio upload path exists (bytes move only via
  fetch); the fetch row stays unacked, the sender retries,
  no partial bundle advances, digest-keyed bytes make
  resume idempotent.
- Invalid/malicious archive → class excluded (§5, no
  archives); malformed manifest entries skip with forensic
  preservation (existing ingest rule).
- Digest mismatch (gateway vs claim, stored vs pinned) →
  terminal refusal for that input + evidence; operator
  re-drives with a fresh claim.
- Bytes changed between verify and test → impossible by
  construction (content store immutable; test re-hashes);
  a mismatch (storage corruption) fails closed with
  corruption evidence, never proceeds.
- Verify crash → no `verified_at`, no evidence row:
  re-verify converges (idempotent).
- Test startup failure / image unverifiable / Docker
  absent → `blocked` with cause (the honest state).
- Network isolation failure (container egress observed by
  canary) → run failed, incident evidence, sandbox
  quarantined from further runs until re-proven.
- Timeout / OOM / kill → FAILED with cause; workdir wiped.
- Output truncation → noted in the report; truncation of a
  closing summary fails the parse → FAILED, not green.
- Cleanup failure → orphan reaper keyed by run id retries;
  residue never blocks new runs; alert surfaces.
- Duplicate invocations → idempotency keys (verify/test
  evidence UNIQUE; decide 409-on-conflict).
- Concurrent verify/test → converge on UNIQUE rows; test
  runs serialize per artifact (second run while running
  returns the running run id).
- Approval racing a new version → decisions bind digests;
  new bytes need a new decision; the old decision never
  transfers.
- Rejection → rework via `continue` with reasons; build
  returns to running per gateway plan §15.
- DB failure mid-write → transaction rollback; no
  half-written lifecycle (single-tx transitions).
- Studio restart mid-run → test rows in `running` past
  their lease reconcile to failed-with-cause on next
  touch; verify/decision rows are idempotent replays.

## 13. Negative test matrix

Every case: red-before/green-after regression test, exact
expected result. Prompt-only restrictions satisfy none.

| # | Case | Expected |
|---|---|---|
| N1 | Non-Go implementation (`*.py` present) | verify refuses `E_LANG` |
| N2 | Alternate substrate (`river`/`dbos` tool) | verify refuses `E_SUBSTRATE` |
| N3 | Invalid MiniappSpec | existing `E_*` taxonomy surfaces |
| N4 | Tampered bytes / wrong digest | fetch or verify refuses `E_DIGEST` |
| N5 | Traversal/absolute/case-collision path | verify refuses `E_PATH` |
| N6 | Symlink/non-regular entry | verify refuses `E_PATH` |
| N7 | Oversize file / oversize total | fetch refuses at caps; verify re-checks |
| N8 | Failing `go test` in sandbox | report FAILED; approval refuses |
| N9 | Secret-bearing file content | verify refuses `E_SECRET` |
| N10 | Secret-bearing test output | redacted in report; secret class flagged |
| N11 | `go.mod` replace / unapproved require | verify refuses `E_DEPS` |
| N12 | Banned imports (`unsafe`, `os/exec`, cgo…) | verify refuses `E_DEPS` |
| N13a | Spec/declared-manifest capability mismatch | verify refuses `E_BINDING` |
| N13b | Declared/registered tool mismatch | test conformance fails |
| N14 | Approve unverified artifact | decide refuses (no `verified_at`) |
| N15 | Approve with failing/blocked tests | approve refuses; reject allowed |
| N16 | Concurrent conflicting decisions | one winner; loser 409s |
| N17 | Agent invoking decide (registry/MCP/gateway) | unreachable: absence test + 401 probes |
| N18 | Sandbox egress attempt (canary) | run failed; sandbox quarantined |
| N19 | Docker absent | run `blocked` with cause; Console renders it |
| N20 | Post-verify byte swap (corruption sim) | test re-hash refuses, corruption evidence |

## 14. Sequenced implementation units and dependencies

The prompt's D0–D6 is adopted — each dependency evidenced:

- **D0: contract + boundaries.** Freeze §5/§6 (§5 identity,
  layout, allowlists; §6 admission rules) as constants +
  doc + table tests with fixtures. Depends on: this plan's
  acceptance. Gates everything: verify/test/decide/persist
  all key off identity.
- **D1: retrieval + admission.** Sender `fetch_artifact`
  case; content store writes; claim→fetched pipeline;
  ingest claim compatibility. Depends on: D0 (needs the
  identity + layout to store). Independently testable with
  the stub gateway + tampered fixtures.
- **D2: `artifact.verify`.** §7 implementation + evidence
  rows + `verified_at` semantics. Depends on: D1 (needs
  stored bytes). Pure/deterministic; the largest negative
  matrix share (N1–N7, N9, N11–N13a).
- **D3: `artifact.test`.** §8 runner + sandbox + blocked
  state + report parsing. Depends on: D2 (only verified
  bytes execute — cost and safety). Needs the real Docker
  daemon for N18/N19-class proofs; else ships blocked.
- **D4: `artifact.decide`.** §9 handler + persistence +
  registry-absence + rework loop. Depends on: D2 (needs
  verified rows + evidence refs); test-run gating needs
  D3's row shape (stable contract agreed at D0).
- **D5: Console integration.** §11 scenes + spine + Pulse
  + decide form. Depends on: D2–D4 rows (no UI before
  backend state — hard rule).
- **D6: rehearsal + acceptance.** Invoice Review Desk
  fixture artifact end to end (claim→decide) + full
  negative matrix + browser evidence + report. Depends on:
  D5. May reuse the `e2e/boundary_b` harness shape for
  ledger-driven browser proof.

No unit starts before its dependency's acceptance rows
exist. D2 and the D3 sandbox-probe track can overlap in
time but D3 executes only D2-verified fixtures.

## 15. Concrete acceptance criteria

- Contract: `studio-artifact/v1` pinned; identity
  recomputation tested against tampered fixtures (N4/N20).
- Retrieval: stub-gateway round trip with digest mismatch
  and oversize fixtures refusing honestly (N4/N7).
- Verify: all §7 checks green on a valid fixture bundle;
  N1–N7/N9/N11–N13a red-before/green-after.
- Test: valid fixture passes in-container with a parsed
  report; N8/N10/N13b/N18/N19/N20 evidenced (N18/N19 on
  the real daemon or the blocked state ships instead).
- Decide: human approve/reject round trip on the exact
  digest; N14–N17 evidenced; rework returns reasons.
- Console: Spec/Validate/Evidence scenes render from rows
  (empty/loading/error/blocked states covered); no green
  without proof rows; placeholders remain only for Deploy
  (Phase E).
- Rehearsal: one Invoice Review Desk fixture artifact
  traverses claim→decide with browser evidence; report
  filed per §7.3 of the agent guidelines.
- Gates: `gofmt`, `go vet`, `go build`, `go test -p 1`,
  race subset per guidelines §8.4; PG19 `:5434` verified;
  no-deploy-from-D-state test green.

## 16. Risks, unresolved questions, and blockers

- **Sandbox daemon rights [ASSUMPTION].** Container-create
  rights for the Studio runtime are assumed; D3 proofs
  decide between Docker execution and shipped-blocked.
  Not a blocker: both outcomes are shippable.
- **New-spec-version flow.** Whether a bundle may propose
  a new `app_specs` version saved at approval, or must
  reference a pre-saved version, is deferred to D1
  evidence (flagged in §5). Either preserves the hash
  binding.
- **Image provenance.** The pinned toolchain image build
  (Dockerfile, miniapp seed, digest pinning, rebuild
  cadence) is D3 design detail; the plan requires the pin
  but does not author the Dockerfile.
- **No structural blockers found.** No Boundary B decision
  needs reopening; no framework change is proposed; no
  Phase E work is required.

## 17. Local-first validation and environment constraints

- No Sprite CLI (including reads), Fly operations, live
  OpenCode/Codex/Claude/Antigravity/Muse, Part 2/C5,
  Runtime provisioning, deployment, or commits — same
  boundary as the planning task, carried into
  implementation until separately authorized.
- PG19 `127.0.0.1:5434` only, via the accepted `testdb`
  endpoint guard; isolated run databases; concurrent
  live-server invariant preserved (U5.12).
- Fixture artifacts are Studio-authored and live in-tree
  under `testdata/`; agent-shaped fixtures are clearly
  labeled adversarial inputs, never trusted code.
- Evidence labels from the agent guidelines (§1.3) on
  every acceptance claim; sandbox claims require
  daemon-executed proofs, never fake-backed assertions.

## 18. Explicit Phase E boundary

Phase D ends when a digest-bound human decision row exists.
Out of scope, explicitly not designed here: Runtime Sprite
provisioning, approved-digest transfer, production
connector binding, health verification, deployment
records, rollback, runtime operations views, Deploy/Run
scene wiring, fleet/quota/billing, second-agent adapters,
marketplace/canvas/workflows, and any live business
integration. The no-deploy-from-D-state test (§4.7, §15)
is the executable boundary: if it ever fails, Phase D is
not done — and nothing in Phase D may fix it by adding
deployment.
