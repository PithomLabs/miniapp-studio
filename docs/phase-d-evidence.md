# Phase D Evidence (running record)

Per-unit completion evidence. Authorizing review:
`plans/prompt22_plan_review.md` (D0 GO, D1–D6 gated sequence).
Plan: `plans/phase-d-implementation-plan.md`.

## D0 — Contract and boundaries (done)

Frozen: `docs/artifact-contract-v1.md` + `artifact/` package
(doc, errors, contract, identity, layout, toolchain) + table
tests + `gateway/collector.go` cap single-sourcing.

- Gate A resolved: bundle claim = `bundle` entry + member
  entries; root binds contract/tenant/build/app/toolchain/
  spec/minifest/member digests; no self-digest in the
  envelope (refused); envelope digest joins the root file
  list (acyclic by construction).
- Spec lifecycle resolved: save-at-verify via existing
  `spec.save`; approval cites + re-checks `(version, hash)`.
- Transport constraints frozen: single-file fetch commands
  only; bundle assembly Studio-side.
- Security requirements frozen: §D of the freeze doc
  (non-root, cap-drop ALL, no-new-privs, seccomp/apparmor,
  no priv/host, ro source mount, scratch mounts, bounded
  tree, cleanup, re-hash + immutable source).
- Tests: root determinism + golden
  (`sha256:17bef013…ecee1a2`), per-member tamper (EDigest),
  toolchain binding, self-digest/envelope strictness,
  path/file-set tables, cap values, module pattern, banned
  imports, stdlib heuristic, toolchain checks, go.mod pin
  match.
- Gates: gofmt/vet/build clean; `go test -p 1 -count=1
  ./...` 16/16 ok; bootstrap 1/1 builds before/after.

## D1 — Retrieval and admission (done)

- Migration `000021_artifact_pipeline` (up/down): blobs,
  verifications, test runs, approvals digest columns +
  partial unique, storage_ref nullable + FK, RLS + grants.
  Two red-before-green catches: missing ENABLE (the 000004
  lesson, caught by the RLS catalog test) and the retained
  `''` storage_ref default (violated the new FK; dropped).
- `testdb` mirror + `TestPrivilegeParity` +
  `TestRoleAttributes` extended; `phaseATables` deliberately
  not extended (version-pinned list; comment corrected).
- Ingest logs one `fetch_artifact` row per bundle member
  (coherent-or-nothing; bounds: 512 members, 50 MB
  declared); duplicates re-drive (terminal recovery);
  non-bundle/malformed claims log zero rows.
- Sender `fetch_artifact` case: fail-fast malformed/vanished
  (no burn, no dispatch); turnless classify guards (retry,
  never supersede); ack parses + verifies digest/size;
  bytes + ack commit atomically in settleTx.
- Gate B resolved by measurement: client response cap was
  1 MB both directions — max fetch needs ~6.7 MB. Fix:
  per-command cap (fetch → 8 MB derived bound + drift-guard
  test; all else stays 1 MB; server inbound unchanged —
  fetch requests are ~37 B; callback cap untouched —
  events only). Proof: 5,242,880-byte file round-trips
  intact (6,990,508 B b64) through the real client/server
  pair.
- Tests: migration behavioral (blobs idempotency/CHECK/FK,
  partial unique, app-role reach); fetch wire/parse tables;
  sender round-trip/tamper/fail-fast; ingest logging/
  coherent-or-nothing/re-drive; transport cap + max-file
  round trip.
- Gates: gofmt/vet/build clean; `go test -p 1 -count=1
  ./...` 16/16 ok; bootstrap 1/1 builds before/after; run
  DBs reaped.

## D2 — Artifact verification (done)

- `artifact.verify` tool (registered, agent-callable
  read-only): identity re-hash, contract/pins, spec
  validation + owner bind, go.mod/go.sum/imports admission,
  secrets, spec↔manifest↔code binding, ownership; evidence
  row + `verified_at` flip atomic in the tool tx;
  idempotent re-verify returns the original evidence id.
- Findings fixed during implementation (all red-first):
  `modfile.File.Toolchain` is a nilable pointer (nil
  panic); `FileEntry` needed lowercase JSON tags (claim
  round-trip) — moved the D0 golden root once, before any
  external reliance; claim/envelope member-set split
  (claim lists the envelope, root input excludes it);
  framework ClassA+river refused before Studio's check,
  so N2 uses ClassB+river; missing artifact errors
  (no evidence parent) instead of verifying false.
- New exports (additive): `spec.ScanSecretsJSON`,
  `tools.ContainsSecretSpan`, `artifact.MiniappGoSum*`
  (freshness-tested vs repo go.sum). New dep:
  `golang.org/x/mod` (cached, offline) for go.mod parsing.
- Tests: valid bundle (verify + evidence + re-verify
  convergence); negatives N1–N5, N7, N9, N11–N13a each
  refusing with its exact code; tamper (N4), missing
  inputs, wrapper unit tests; registry-wide tenant/auth/
  malformed suites extended to the new tool (26 tools).
  N6 covered at collection (existing symlink tests);
  N13b/N14–N20 belong to D3/D4.
- Gates: gofmt/vet/build clean; `go test -p 1 -count=1
  ./...` 16/16 ok (rc=0); bootstrap 1/1 builds
  before/after; run DBs reaped.

## D3 — Isolated test execution (done)

- Toolchain image `miniapp-phase-d-toolchain:local`
  (`sandbox/testdata/toolchain/Dockerfile`): base
  `golang:1.27.3@sha256:e519d3b0…f0d1e` pinned by digest,
  miniapp closure pre-seeded (offline builds), non-root
  tester (65532), fail-closed go.sum grep in the build.
  Built image id `sha256:b3269ef4…840310` (recorded, not
  pinned — rebuilds legitimately change it; base digest +
  behavioral content checks are the pins).
- `sandbox/` runner: disposable containers with the full
  gate-C flag set (unit-pinned in `TestContainerArgs`):
  no network, uid 65532, cap-drop ALL, no-new-privs,
  default seccomp/apparmor (daemon-verified), pids 256,
  1g/1cpu, read-only rootfs + ro source bind, tmpfs /tmp,
  rw scratch bind, scrubbed explicit env incl.
  `GOPROXY=off`, `-mod=readonly`, `CGO_ENABLED=0`,
  pinned module cache; timeout with kill + rm; output
  caps with measured truncation; secret redaction at the
  capture boundary (single choke point); egress-marker
  quarantine with re-prove.
- Design corrections with rationale: staging dirs are
  world-accessible (container uid must traverse; random
  names under sticky /tmp; alternative — Studio uid —
  would hand an escape the host credentials); /tmp stays
  executable (Go test binaries execute from the build
  cache); `GOFLAGS=-mod=readonly` not `-mod=mod`
  (verified artifacts arrive complete); probe go.mod
  inherits the artifact's require set verbatim.
- Report parser fails closed (truncation/garbage/missing
  verdict/nonzero exit all refuse green). Store:
  claim/start/finish lifecycle, advisory-lock
  serialization, stale reconcile; no global janitor
  (per-artifact reconcile + stale-aware display contract).
- N13b conformance: Studio-authored harness (scratch
  module, replace→/work/app) registers minimapp.json
  against the artifact's exported `Implementation()`.
  Convention frozen here (completing v1's binding
  requirement; no version bump — the requirement
  predates, this is its mechanism).
- Tests (all executed, none skipped here): image
  content; valid/failing runs with parsed reports (N8);
  canary (N18); env scrub; fs ro/rw; pids bomb with
  pre-asserted flags; timeout+cleanup with leftover
  assertion; secret redaction (N10); conformance ±
  (N13b, negative names the missing export); blocked
  without docker / unverifiable image (N19 shape);
  quarantine unit test; parser tables; store lifecycle.
  Red-first catches: signal-exit timeout masking, probe
  require completeness, vacuous negatives hardened.

## D4 — Human authorization (done)

- `POST /console/artifact-decide` (kill-switch precedent:
  unregistered, session+Origin, server-side tenant).
  Single tx: lock artifact row; require bundle+verified;
  re-hash bytes vs digest (corruption → 500, nothing
  decided); replay-or-409 on existing rows; terminal
  test-run gate (approve additionally requires passed);
  save-or-cite spec version; digest-bound decision row
  with human binding + evidence refs.
- Corrections with rationale: no auto-continue on
  rejection (would violate sticky terminality — rework
  enters via a new build carrying the brief, which the
  redirect notice surfaces); envelope↔spec app check
  added to verify too (N13c); shared `store.LoadBundle`
  (verify refactored onto it, D2 suite re-greened);
  `SaveSpecVersion`/`FindSpecVersion` extracted (specSave
  delegates, existing tests green).
- Tests: approve round trip (row + cited v1 + re-verify
  replay); conflict; gating matrix (404/400/unverified/
  failed-approve-refused/failed-reject-allowed/no-run);
  concurrent same→all-302 + mixed→{302,409} with one
  row; tampered-bytes 500 with zero rows; second
  approval cites (no duplicate spec version); N17
  registry absence + unknown-tool refusal.
- Gates: gofmt/vet/build clean; `go test -p 1 -count=1
  ./...` 17/17 ok (rc=0); bootstrap 1/1 builds
  before/after; run DBs reaped.
- Note: `plans/faq.md` shows 424 added lines dated
  mid-session from another author (owner-side FAQ
  entries); untouched by this work, left as found.

## D5 — Console review (done)

- `artifact.status` extended to the full review bundle: per
  artifact created/decided timestamps, latest checks JSON,
  latest run report, app/spec identity (parsed from the
  envelope correlation via `artifact.EnvelopeFileName`, never
  hardcoded), plus top-level `canonical_spec` (latest saved
  version + hash + document for the newest bundle's app, nil
  before any decision saves one).
- New scenes `scene_spec.html` (canonical version/hash/pretty
  document, honest "No saved spec"), `scene_validate.html`
  (per-artifact digest/verified/checks-table/run-report with
  failures/run-test/approve/reject+note forms), and
  `scene_connect.html` (connector list or honest unconfigured).
  Evidence gains the artifact ledger (digest × verified ×
  tests × decision); only Deploy still placeholders.
- One read-only status call per focused console page feeds
  spine + scenes; per-artifact opkeys (test/approve/reject
  distinct, same-key-fingerprint precedent). Spine: Validate
  active→proven on verified, Spec proven on saved version,
  Evidence active→proven on decision; Run guardrail intact;
  Connect stays gray on the spine (live tool read only on
  its own scene — no claimed-but-unloaded state).
- Corrections with rationale: seeded bundles lacked turn 1
  (`conversation.get` requires it; production births every
  build with one in `createBuildTx`) — seed fixed, not the
  view; template evidence shapes corrected to the real JSON
  (`check`/`pass`/`message`, report `pass`/`total`/`failed`
  + `failures[]`, conformance `pass`); stale placeholder
  tests + Boundary B phase5 updated to the shipped contract
  (only Deploy waits).
- Tests (`server/console_phased_test.go` + spine table):
  validate shows bundle/checks/forms; empty states; decide
  forms approve + reject end to end with ledger assertions;
  spec absent-before/present-after decision; connect
  unconfigured; Pulse carries `artifact_decided`; status
  tool review-bundle shape incl. canonical-spec gating.
- Gates: gofmt/vet/build clean; serial suite 17/17 ok
  (rc=0, sandbox 532s incl. daemon tests); server+tools
  re-greened on the final tree; run DBs reaped, zero
  stray `phase-d-run-*` containers.

## D5b — Fetch→Verify link (done)

- Gap found designing the D6 rehearsal: nothing in
  production invoked `artifact.verify` — fetched bundles sat
  unverified forever while test and decide both refuse
  unverified rows. Root cause, not wiring: the lifecycle
  plan lists Verify after Fetch but no trigger existed.
- Fix: verify core extracted to
  `tools.VerifyBundleTx(ctx, tx, tenant, artifactID)` (tool
  wrapper delegates; D2 suite re-greened unchanged); the
  sender runs it in the settle tx on first fetch completion
  (`maybeAppendFetched`), so bytes, verified_at, evidence,
  and both receipts commit atomically. Passed=false is an
  outcome (bytes kept with failure evidence), never a
  settlement error.
- Tests: claim→fetch→verify through the real sender
  (`TestSenderFetchCompletionVerifies`: verified_at set,
  passed evidence, exactly one `artifact_fetched` +
  `artifact_verified` receipt); tampered-bytes failure keeps
  all blobs with failed evidence
  (`TestSenderFetchCompletionFailureKeepsBytes`);
  `buildCoherentBundle` extracted for DB-free byte building.
- Gates: covered by the 23.1 serial suite below (17/17);
  D2 negative matrix + D4 decide suite re-greened
  unchanged after the `VerifyBundleTx` extraction.

## 23.1 — D6 abort, D5/D5b close (in progress)

- D6 browser rehearsal ABORTED per prompt 23.1 before any
  review evidence: claim ingested (1 artifact row, digest
  `sha256:d51ab8e6…826de`), 7 fetch rows queued, 0 fetch
  dispatches (stub call log empty), 0 verified, 0 test
  runs, 0 decisions, 0 screenshots. No D6 evidence exists;
  nothing from it is cited.
- Cleanup: rehearsal studio (pid 1472343) + stub
  (pid 1472342) killed; `phase_d_rehearsal` DB dropped;
  no `phase-d-run-*` containers left by it. Owner's
  OpenCode crashpad process left untouched. Run files
  preserved at `/tmp/phase-d/runs/20261126-095802`
  (provision.log, studio.log, fixture + claim.json).
- D6 harness files remain in-tree, unused, for the
  planned redesign: `e2e/phase_d/cmd/genfixture/main.go`,
  `e2e/phase_d/rehearse.py`, the additive `fetch_artifact`
  branch in `e2e/boundary_b/stub/gateway.py` (inert
  without `PHASE_D_FIXTURE_DIR`). The phase5 placeholder
  update is REQUIRED (D5 shipped those scenes) and stays.
- D5/D5b verification set (all in green server+tools
  runs): automatic verification on fetch completion
  (pass + tamper-failure-keeps-bytes link tests); digest
  integrity (per-file claim re-hash, identity
  recompute-vs-pinned, tamper refusals); idempotency
  (decide replay-or-409, claim digest collapse, evidence
  upsert); retries (sender backoff, fetch re-drive);
  failure handling (honest blocked runs, decide gating
  matrix, corruption 500s with zero rows); durable
  evidence (verification/test-run/approval rows plus
  `artifact_fetched`/`artifact_verified`/`artifact_decided`
  stream receipts, all asserted).
- Environment note: mid-session the sandboxed shell lost
  host TCP (`127.0.0.1:5434` refused) and Docker socket
  access although `miniapp-p0-pg` (5434) stayed Up;
  diagnosis + all DB/Docker steps since ran via the
  sanctioned one-time escalated route with approval. No
  escalated command wrote outside test DBs, run scratch,
  and the dropped rehearsal DB.
- Suite gate: GREEN. `go test -p 1 -count=1 ./...`
  rc=0, 17/17 packages ok (sandbox 532.3s, server 37.8s,
  tools 13.7s, rest <8s). gofmt/vet clean. Targeted
  `TestDaemonTimeoutCleanup` rerun before it FAILED
  (leftover container); the suite's own run of the same
  test PASSED. History: fail→pass→fail→pass across
  suite/isolated/targeted/suite — intermittent, in
  untouched D3 cleanup code (`defer rm -f` swallows its
  error in `sandbox/runner.go`; file mtimes 08:13–09:18
  predate the D5 session). D3 left unmodified per 23.1;
  the flake stands as a documented residual, not a waiver:
  any future red on it fails the gate again.
- Changed files (D5/D5b/23.1 only; prior accepted Phase D
  work untouched): `tools/artifact_status.go` (review
  bundle), `tools/artifact_verify.go` (`VerifyBundleTx`
  extraction, tool delegates), `server/sender_fetch.go`
  (first-completion verify hook), `server/handlers_console.go`
  (scenes/spine/shaping), `server/artifact_decide_test.go`
  (turn-1 seed coherence, `buildCoherentBundle`,
  `digestHex`), `server/handlers_console_test.go`
  (spine/status/deploy-only placeholders),
  `server/sender_fetch_test.go` (Fetch→Verify link pair),
  `server/console_phased_test.go` (new: scene/form/pulse/
  status tests), `server/templates/focus/scene_{spec,
  validate,connect}.html` (new), `scene_evidence.html`
  (ledger), `e2e/boundary_b/phases/phase5_console.py`
  (shipped-scene assertions), `e2e/boundary_b/stub/
  gateway.py` (additive fetch branch, inert by default),
  `e2e/phase_d/{cmd/genfixture/main.go,rehearse.py}`
  (new, unused after the abort), `docs/phase-d-evidence.md`,
  `plans/prompt23.1.md` (new).
- Residuals: (1) sandbox timeout-cleanup flake above;
  (2) D6 rehearsal harness unused, kept for the redesign;
  (3) toolchain image digest recorded, not code-pinned
  (rebuilds change it); (4) `phase_d_rehearsal` DB
  dropped, run files kept at
  `/tmp/phase-d/runs/20261126-095802`; (5) `plans/faq.md`
  still carries concurrent owner-side edits, untouched.
- Git status at filing: no commits (per standing rule);
  65 modified+untracked entries, all Phase D scope or
  plans/docs; `genfixture` binary + `__pycache__` litter
  removed; no stash entries created.

## 23.1 — verdict: D5/D5b DONE, D6 not started

D5 Console review and the D5b Fetch→Verify link are
implemented, tested, and green on a full serial suite
with an honest accounting of the one intermittent
residual. The D6 rehearsal was aborted before producing
evidence and is not claimed. Stopping here per 23.1.
