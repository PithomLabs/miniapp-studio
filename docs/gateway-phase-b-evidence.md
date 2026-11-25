# Phase B evidence record — Part 1: fake-backed gateway (2026-11-25)

Implementation of `plans/prompt9_imp.md` Part 1 (authorized; Part 2 live
work explicitly NOT started). No live Sprite, Fly API mutation, OAuth,
or external action was performed: every proof below runs against
in-process fakes, scripted doubles, real local processes, or Docker
Postgres. Boundaries held: F1/G-01/P5 frozen, M5–M8 carried, Miniapp
untouched, no commits.

## Gates executed (Part 1 pass)

- `gofmt -l .` → no output (exit 0); `go vet ./...` → exit 0;
  `go build ./...` → exit 0.
- Static gateway binary: `CGO_ENABLED=0 GOOS=linux go build
  ./cmd/gateway` → `ELF 64-bit LSB executable, x86-64, ... statically
  linked` (verified via `file`).
- `go test -p 1 ./...` on PG19 (127.0.0.1:5434): 13/13 packages ok,
  0 failures (root: no test files).
- `go test -race -p 1` on `gateway`/`store`/`tools`/`server`/
  `cmd/studio`/`cmd/gateway`: all ok, exit 0.
- `go test -p 1 ./...` on PG17 (127.0.0.1:5435): 13/13 packages ok,
  0 failures. (Historical: prompt11 + AGENTS.md §8.3 mandate PG19-only
  verification afterwards; no PG17 runs in the correction pass.)
- Tree state at gate time: `git status` all-untracked, zero commits
  (no-commit boundary held); Miniapp + frozen docs untouched.

## What was built

- `gateway/`: v1 command protocol (`SignCommand`/`VerifyCommand`),
  Studio `Client` (sign once, redeliver identical bytes, bounded
  retry/backoff), command `Server` (auth, per-build dedup with stored
  replay, assign/continue/cancel/status/fetch_artifact, event pump to
  outbox), durable `Outbox` + `Deliverer` (backoff, terminal 4xx,
  25-attempt cap), `Supervisor` (spawn/signal/reap, never restarts),
  `SessionManager` (atomic on-disk records, escape-proof keys),
  `OpenCodeAdapter` (NDJSON translator, file-drop Q&A, collect),
  `FileKeys`/`LoadKeysDir`/`KDFProvider` key boundary, boot
  `RecoverSessions` (identity-checked orphan reap).
- `tools/build_provision.go`: `build.provision` (per-application
  deterministic builders, attach-on-retry, token-only URL, staged
  install bundle incl. per-build HMAC key file, idempotent setup exec,
  service refresh + running verification, row update) + embedded
  `provisiondata/{opencode.json,builder.md,setup.sh}` (pinned,
  structure-tested).
- `server/`: `STUDIO_GATEWAY_ORG_SECRET` config, KDF wiring with
  unconfigured warning, `Options` seams, exported `RunTool` for
  operator commands.
- `cmd/gateway`: sprite binary (flags, keys dir, recovery, serve).
- `cmd/studio gateway provision|command`: operator CLI for the live
  runbook (fake-tested; performs nothing without invocation + live env).

## Capability matrix v1 (honest states)

| Capability | State | Evidence / next proof |
|---|---|---|
| Command sign/verify, skew, strict envelope | IMPLEMENTED+TESTED | protocol + server tests |
| Client send/retry/same-bytes/backoff | IMPLEMENTED+TESTED | client tests vs real server |
| Command dedup + stored replay, persisted | IMPLEMENTED+TESTED | assign flow + reopen test |
| assign/continue/cancel/status/fetch flows | IMPLEMENTED+TESTED (fake adapter) | server tests incl. 409/404 |
| Outbox durable + deliverer retry/terminal | IMPLEMENTED+TESTED | outbox tests + joint Studio test |
| Supervisor spawn/signal/reap/exit codes | IMPLEMENTED+TESTED (real processes) | supervisor tests |
| Sessions CRUD/persist/escape-proof | IMPLEMENTED+TESTED | session tests |
| NDJSON translator (verified shapes) | IMPLEMENTED+TESTED (canned) | events tests; part internals live-confirm (P-G0) |
| OpenCode start/continue/cancel/collect/status | IMPLEMENTED+TESTED (doubles) | adapter tests |
| File-drop Q&A handshake | IMPLEMENTED+TESTED (doubles); AGENT COMPLIANCE unverified | live P-G3 |
| Native `question` tool under run | UNSUPPORTED BY DESIGN (verified: CLI denies it) | run.ts @ v1.19.6 |
| Boot recovery (reap+failed+callback) | IMPLEMENTED+TESTED (fake /proc) | recover tests |
| Provision tool + install bundle | IMPLEMENTED+TESTED (fakes) | provision tests |
| Pinned opencode.json/plugin/config | SPECIFIED+STRUCTURE-TESTED | bundle test; live behavior open |
| setup.sh + dual-user layout | SPECIFIED, UNEXECUTED | live install |
| serve binary + static build | IMPLEMENTED+TESTED | boot test + ELF check |
| Operator CLI provision/command | IMPLEMENTED+TESTED (fakes/local) | cmd tests |
| Real OpenCode no-op turn | UNVERIFIED (live) | Part 2 |
| Sleep→wake→resume + warm/cold observation | UNVERIFIED (live) | Part 2 (P-G2) |
| Exit-code semantics (0 idle-ok / 1 error) | SPECIFIED from source | live P-G1 |
| NDJSON part internals | SPECIFIED from source, tolerant parse | live P-G0 |
| OAuth bootstrap ladder | SPECIFIED (P-E1–P-E3) | live, owner ceremony |
| Sprite exec privilege (root for setup) | UNKNOWN | live discovery, fail-closed |
| Wake-on-command latency | UNKNOWN | live timing |
| Studio sender loop (DB dispatch) | NOT BUILT — deferred to Phase C | matrix record (this line) |
| SSE stream lifetime (M5) | NOT BUILT — live-segment checklist | runbook §L |

## Deviations and findings (with reasons)

1. **P-G3 primary path refuted by source inspection** (run.ts @ v1.19.6:
   non-interactive sessions hard-deny the `question` tool, and the run
   loop has no question-reply path). The file-drop handshake is therefore
   the REQUIRED mechanism, not a fallback. Live P-G3 now proves agent
   compliance with the file convention instead.
2. **Adapter returns event channels from Start/Continue** (no separate
   Events call): a fast run could otherwise finish before subscription,
   losing its terminal event. Race eliminated by construction.
3. **One session record per build, adapter-owned creation**: fixes a
   found race (pump updating before creation) and a real double-assign
   path (409 + no stray agent).
4. **Outbox as atomic whole-file rewrite**, not append-log+compaction:
   bounded queue size makes rewrites trivially cheap; same crash story.
5. **Per-application builders** per §10 (tenant/app-hashed names), with
   per-build key files (no read-modify-write) and empty-app default
   pooling for the MVP.
6. **Warm/cold wake reports are third-party and unverified**: the design
   is safe under both (warm = nothing restarts; cold = recovery marks
   failed, Studio re-drives), and the runbook observes and records which
   holds instead of assuming.
7. **M5 deferred to the live-segment checklist** (operational hardening
   with no fake-backed signal); M6 restated below; M7 stays Phase D; M8
   counts in this file are exact.

## Standing disclosures (carried)

- M6: the pinned framework persists full tool inputs in
  `app.invocations`; tenant-RLS'd, no Studio read path, activity scrub
  on read. Phase B contract: never select invocation inputs into any
  response (no gateway code path does).
- F1/G-01/P5 unchanged. Frozen docs and Miniapp untouched. No commits.

## Pending (Part 2, explicitly unauthorized here)

Live install proof, no-op turn, sleep/wake/resume observation, OAuth
ceremony, wake timings, exec-privilege discovery, builder retention
decision. See `docs/gateway-phase-b-live-runbook.md`.

---

# Correction pass record — prompt11 (2026-11-25)

Bounded correction of the adversarial review (`plans/adv_review_prompt10.md`):
B1 + F2–F9 at root cause, M1–M16 dispositioned, G1–G8 evidence
corrected, §7 contract reinforced. Local/fake-backed only — Part 2 live
work NOT executed. Boundaries held: F1/G-01/P5 frozen, Miniapp
untouched, frozen docs untouched, no commits, PG19-only verification.

## Gates executed (correction pass)

- PG19 confirmed first: `SHOW server_version` → `19.0
  (Debian 19.0-1.pgdg13+1)` on 127.0.0.1:5434 (container miniapp-p0-pg).
- `gofmt -l .` → no output; `go vet ./...` → 0; `go build ./...` → 0.
- `go test -p 1 ./...` (PG19): 13/13 ok, 0 failures, 0 skips.
- `go test -race -p 1` on the six required packages: all ok.
- Static `cmd/gateway`: ELF 64-bit x86-64 statically linked (re-verified).
- `GOOS=darwin go build ./gateway/ ./cmd/gateway/` → ok (non-linux
  fail-closed stubs compile).
- `sh -n tools/provisiondata/setup.sh` → ok.
- Probes P1–P4 re-run green (see below).
- Test counts (exact, counted): gateway 99, store 23, tools 58,
  server 56, cmd/studio 9, cmd/gateway 3, fly 16, apps 5, connector 3,
  render 11, runner 29, spec 34, spike 3 — 349 total.
- Tree state: all-untracked, zero commits (no-commit boundary held).

## Probe re-run (corrected tree)

- P1: `codes=[200 200] adapter.Continue calls=1` — duplicates share
  one execution.
- P2: `kinds=[message failed]`, no key material — symlink refused,
  run fails honestly.
- P3: victim intact (`clobbered=false`), answer slot real + 0640.
- P4: `kinds=[progress progress message done]` — no premature terminal.

## Capability matrix v2 (five states)

States: (1) implemented+tested; (2) implemented, insufficiently tested;
(3) specified, not implemented; (4) explicitly not built;
(5) live behavior not yet verified. Fake-backed rows never claim live
proof; live rows never claim fake proof.

| Capability | State | Evidence / next proof |
|---|---|---|
| Command sign/verify, skew, strict envelope | (1) | protocol tests; Resign unit |
| Client send/retry/backoff + https rule | (1) | round-trip + M12 allow/refuse tests |
| Durable claims + digest conflicts + indeterminate | (1) | claim lifecycle + A1 adversarial tests |
| Call table + per-build run serialization | (1) | N-identical/N-distinct/mismatch tests |
| assign/continue/cancel/status/fetch + outbox cmds | (1) | server + command tests |
| Turn fencing (400 absent, 409 stale) | (1) | fencing tests; CLI/runbook carry turns |
| Outbox durable + inspect/requeue | (1) | requeue-drain + scoping tests |
| Deliverer: backoff, re-sign, timeout, transport guard | (1) | aged-redelivery, hang, insecure tests |
| Supervisor: credential drop + fail-closed | (1) fake / (5) live | refusal + plumbing + `id` tests live: real drop + EACCES |
| Supervisor: umask, pgroups, stderr ring, WithoutCancel, SIGTERM | (1) | real-process tests |
| Sessions + partial list + degraded + turn | (1) | session + recovery tests |
| Translator (error→progress, defensive) | (1) canned | events tests; P-G0 live |
| Adapter incl. lineage check + shutdown | (1) doubles | adapter + shutdown + mismatch tests |
| File-drop: OpenRoot, ownership, atomic replace | (1) single-user / (5) dual-user live | symlink/oversize/mode/race/restart matrix |
| Collector fd-pinned | (1) | refusal + swap-hammer tests |
| Boot recovery: indeterminate + emit-first + precise reaper | (1) | fake-/proc + real-proc tests |
| Provision: lock, create-if-absent, refresh, fail-closed, digest | (1) fakes | F7 matrix incl. cross-session lock proof |
| Upgrade tool + CLI (idle-guarded) | (1) fakes + httptest | upgrade tests |
| ReloadingKeys (no-restart refresh) | (1) | pickup + malformed tests |
| setup.sh + dual-user layout + staging hygiene | (3) | structure-tested (`sh -n` + assertions); unexecuted |
| Pinned bundle (opencode.json/plugin/builder) | (2) | structure tests; behavior live |
| Gateway binary + static build + serve loop | (1) | boot test + ELF re-verify |
| serve() identity/boot wiring | (2) | resolveAgent unit-tested; full boot live-only |
| Operator CLI (provision/command/upgrade) | (1) | usage + round-trip + formatter tests |
| run.ts claims (denial/shapes/exit/flags) | (1) code-verified | vendored hash-verified excerpt |
| Real OpenCode execution (any turn) | (5) | Part 2 P-G3 transcript |
| Agent file-convention compliance | (5) | Part 2 P-G3 |
| Warm/cold wake + resume behavior | (5) | Part 2 §11 observations |
| Exit codes live (P-G1) | (5) | Part 2 forced-error + clean runs |
| Part internals live (P-G0) | (5) | Part 2 captured transcript |
| OAuth ceremony (P-E1–P-E3) | (5) | owner ceremony record |
| Exec privilege + service identity | (5) | Part 2 §7 discovery (fail-closed) |
| Wake latency | (5) | Part 2 timing table |
| Dual-user inbox + key EACCES proof | (5) | Part 2 §7 checks |
| Studio sender loop + reconciliation | (4) | Phase C requirement (documented) |
| SSE stream lifetime | (4) | live-segment checklist (unchanged) |
| Artifact-admission pipeline | (4) | Phase D (contract reinforced only) |

## G1–G8 dispositions

- G1 (test count): corrected — 99 gateway / 349 total, counted above.
- G2 (file-drop directionality): corrected — directionality is now
  PROVEN single-user (refusal matrix) with dual-user live proofs
  assigned to runbook §7, not assumed.
- G3 (P-G3 wording): corrected — future-tense throughout; "the actual
  OpenCode binary was never exercised" stated in the matrix.
- G4 (run.ts unauditable): corrected — vendored hash-verified excerpt
  at `docs/opencode-run-v1.19.6.md`; code citations point at it.
- G5 (outbox terminal loss): corrected — requeue/inspect path
  implemented + tested; matrix notes it.
- G6 (dedup concurrency): corrected — durable claims + serialization
  implemented + tested; guarantee framed as at-most-once dispatch.
- G7 (recovery dead end): corrected — F6 resume implemented + tested
  end to end; runbook §11.5 rewritten to match.
- G8 (flyErr assertion): corrected — comment downgraded to
  capped-but-unverified with the required live inspection.

## M1–M16 dispositions (all fixed, none carried)

M1 dir-fsync (3 stores + helper test); M2 per-attempt timeout
(component-level, hanging-stub test); M3 SIGTERM + WithoutCancel +
Shutdown (signal/cancel tests); M4 process groups + group reap with
non-leader fallback (grandchild tests); M5 stderr ring (wedge +
bounded-tail tests); M6 partial list + emit-before-mark + deterministic
ids (skip + idempotence tests); M7 argv[0] + starttime reaper (decoy +
reuse + parser tests); M8 fd-pinned collector (refusal + swap tests);
M9 scanner errors surfaced (overlong-line test); M10 64KB drop cap
(oversize test); M11 turn fencing (absent/stale/next tests); M12
https-except-loopback both directions (refuse/allow tests); M13 CLI
allowlist + fetch truncation + --output (allowlist + formatter tests);
M14 digest print + tool-result record (echo test); M15 KDF negative
(wrong-secret 401; wiring test pre-existed); M16 header refresh +
turn_id populated in signed bytes + shortID hash fallback (mapping
test).

## Deviations and findings (with reasons)

- Same-uid spawns need explicit `AllowSameUID` (tests/dev only;
  serve never sets it); Go applies setgroups for ANY Credential
  (verified in the fork/exec source), so same-uid skips it via
  `NoSetGroups` while different-uid pins explicit groups.
- F4 recovery is indeterminate-first (plan-review A1): at-most-once
  dispatch with explicit F6 recovery, never auto-drop.
- The pump marks degraded immediately (no retry): outbox persist
  failures are not transient; fast drain beats slow hammering.
- Group kill falls back to direct kill on ESRCH (verified orphans
  only) for non-leader pids.
- `abortRun` marks failed+degraded (commutes with pump marking —
  no lost update between them).
- Recovery emits before marking under deterministic ids; an
  already-queued id proceeds to marking (not an error).
- Continue restores the pre-resume state on launch failure (failed
  stays failed, never rewritten to yielded).
- serve() resolves agent/io identity by NAME (no uid plumbing
  through provisioning); resolution failure refuses the boot.
- M2 implemented as a component-level attempt timeout (unit-testable)
  rather than wiring-only.
- Vendored run.ts shows error lines always precede exit 1 in v1.19.6:
  F5's exit-decides rule coincides there and stays robust if the
  coupling loosens; error lines are confirmed mid-run (stream
  continues to idle).
- The collector refuses ALL symlink entries, including
  inside-resolving ones (manifest names real files).
- Unknown self-ticks degrade the reaper to argv[0]-exact with a loud
  boot warning (documented residual).
- `shortID` hashes non-hex ids (defense in depth; Studio mints hex).

## Residuals and live unknowns (carried, not hidden)

R1–R10 unchanged (all assigned to Part 2 checkpoints); catastrophic
disk (log-only when the degraded mark itself fails); two gateways on
one state dir (operator error → honest 503); full `serve()` boot
unexecuted (needs real users); `setup.sh` unexecuted; umask wrapper
requires `/bin/sh` on the builder (setup already requires sh);
co-located builds share the agent uid (compromised agent reaches
same-app sibling workdirs; cross-tenant builders don't exist, so
blast radius is one tenant+app — per-build agent identities are a
follow-up, not a prompt11 finding).
