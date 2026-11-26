# Artifact Contract v1 — D0 Freeze

Status: frozen. Contract version `studio-artifact/v1`, implemented
in `artifact/` with table tests. Any change to identity, layout,
pins, or caps is a contract-version bump, never a silent edit.
Source review: `plans/prompt22_plan_review.md` (gates A–C + spec
lifecycle). Phase D plan: `plans/phase-d-implementation-plan.md`.

## A. Bundle identity and structure (review gate A — resolved)

- A bundle claim is one `bundle`-kind entry plus its member
  entries in a single `agent.artifact_ready` payload. Ingest
  persists one row per well-formed entry (existing behavior);
  the bundle row's digest is the CLAIMED root.
- Members associate to the bundle through the bundle
  envelope's member list (`manifest.json` → `files[]`), matched
  against stored bytes by path. Member files are bytes in the
  content store, not rows; no kind-enum extension.
- The root binds, over canonical JSON with sorted paths:
  contract version; tenant/build/conversation/turn;
  app id/version; the FULL toolchain declaration (Go version
  + Miniapp version); the spec file digest; the framework
  manifest digest; and every member's path, digest, and size.
  A changed toolchain declaration changes the identity
  (pinned by `TestRootBindsToolchain`).
- No circular hashing: the envelope lists member digests but
  must not carry any self-digest key (`digest`,
  `artifactDigest`, `artifact_digest`, `root`, `id` —
  refused with EContract). The envelope's own digest is
  computed over its bytes and added to the root file list by
  `ComputeRoot`. Flow is one-directional: envelope → member
  digests → root.
- Recomputation rule: verify, test, and decide each recompute
  from stored bytes and compare against the pinned digest
  before acting. Any mismatch refuses with EDigest.

## B. Spec version lifecycle (resolved)

Save-at-verify: when verification passes a bundle carrying a
new spec version, the review flow persists it via the existing
`spec.save` tool (versioned, hashed). Approval never creates
versions — it CITES the persisted `(version, spec_hash)` and
re-checks the embedded-spec hash at decide time, so approving
one version while persisting another is impossible. Abandoned
proposed versions are honest immutable history (superseded,
never approved). Rationale over atomic-with-approval: review
renders from the same row the decision will cite, and the
decide handler stays single-purpose.

## C. Transport constraints (frozen for D1)

- Fetch commands address SINGLE files (`fetch_artifact` with
  one manifest entry): at most 5 MB → ~6.7 MB base64 per
  command, never a 50 MB whole-bundle command. Bundle
  assembly is Studio-side from the content store.
- D1 must evidence HTTP body limits, sender payload handling,
  and DB row effects against a max-size single-file fetch,
  plus retry/idempotency across a multi-file bundle, before
  the transport is accepted. Whole-bundle commands are
  forbidden by this freeze.

## D. Execution security requirements (frozen for D3)

The test runner MUST enforce all of: disposable container;
image pinned by digest with exactly the toolchain + miniapp
pin + go.sum closure; `--network=none`; `--pids-limit`,
`--memory`, `--cpus`; read-only rootfs; non-root user;
`--cap-drop=ALL`; `no-new-privileges`; default seccomp and
AppArmor/SELinux confinement where available; no privileged
mode, host namespaces, host devices, or Docker socket;
artifact source mounted read-only; separate disposable
writable scratch/cache mounts; fully scrubbed environment
(`GOPROXY=off`, tmpfs `GOCACHE`); wall-clock timeout with
container kill and process-tree reaping; capped output with
secret redaction; re-hash of inputs before execution.
Failure of any requirement yields the honest `blocked`
result. Trusted repository fixtures alone may run on host
`go test`.

## E. Implementation map

- `artifact/doc.go`, `errors.go` (E_* taxonomy),
  `contract.go` (version, layout, allowlists, caps, banned
  imports), `identity.go` (envelope + root), `layout.go`
  (path + file-set rules), `toolchain.go` (pins).
- `gateway/collector.go` caps single-sourced from
  `artifact.MaxFileBytes/MaxTotalBytes`.
- Tests: determinism + golden root, per-member tamper
  (EDigest), toolchain binding, self-digest refusal,
  envelope strictness, path/file-set tables, cap values,
  module pattern, banned imports, stdlib heuristic,
  toolchain checks, go.mod pin match.
