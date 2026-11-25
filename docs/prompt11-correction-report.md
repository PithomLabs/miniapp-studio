# Correction report — prompt11 Phase B Part 1 (2026-11-25)

Governing review: `plans/adv_review_prompt10.md`. Directive:
`plans/prompt11.md` as amended by `plans/prompt11_plan_review.md`
(A1 indeterminate claims, A2 failure rules + cross-process lock,
A3 identity/filesystem spec). Execution: `plans/prompt11_plan.md`
Phases 0–9, uninterrupted pass.

**Part 2 was NOT executed.** No live Sprite was provisioned or
deleted, no Fly API mutation ran, no OAuth flow started, no gateway
was installed anywhere, no external infrastructure was touched, no
Miniapp code changed, F1/G-01/P5 untouched, no commits or pushes.
All verification below is local/fake-backed against Docker PG19
(19.0) plus real local processes. This report stops at correction
evidence per the directive.

## 1. Findings fixed (root cause → correction)

### B1 — No OS-identity separation (BLOCKER)

Root cause: identity existed only as setup-script user creation; no
code path applied it. The supervisor spawned with no `Credential`,
the Sprite service API has no user field, provisioning set none —
so the agent inherited the gateway uid with full key access, and no
uid assignment could make the file-drop design work (single-user
exposes keys; dual-user broke Q&A both directions).

Correction (`gateway/supervisor.go`, `gateway/proc_linux.go`,
`gateway/proc_other.go`, `cmd/gateway/main.go`,
`tools/provisiondata/setup.sh`, `tools/build_provision.go`):
every agent spawn drops via `SysProcAttr.Credential` to configured
agent uid/gid/groups; any spawn failure (including unprivileged
EPERM) refuses the command — fail-closed by construction, no
same-user path. Unconfigured or non-allowed same-uid spawns refuse
(`AllowSameUID` exists for tests/dev only; serve never sets it).
Different-uid spawns pin explicit non-empty groups
(`NoSetGroups` only for same-uid — Go setgroups unconditionally,
verified in the fork/exec source). Inbox model enforced at runtime:
gateway-owned dir `studio-gw:studio-io` 2730, `.studio` 2750,
workdirs 2770, answers 0640, agent umask pinned 007 via the spawn
wrapper (drops land 0660). Staging locked to 0700/0600 at setup
start, key copies deleted after install (verified absent).
Service PATH pinned to root-owned dirs; setup asserts its
prerequisites; serve resolves agent/io identity by name and refuses
boot on failure.

Regression: refusal (unconfigured / different-uid-unprivileged /
same-uid-unallowed / missing io membership), plumbing (`id`
output), umask-pin (0660), mode assertions (2730/2750/2770/0640),
service PATH args, setup.sh static assertions + `sh -n`.

Evidence: unit (refusal/plumbing/modes) + integration (real
processes). The real different-uid drop + key EACCES are live-only
(runbook §7b gate). Residual: co-located builds share the agent
uid, so a compromised agent can read same-app sibling workdirs
(same tenant+app blast radius; per-build agent uids are a
follow-up, not a prompt11 finding).

### F2/F3 — Symlink file-drop exploits

Root cause: question read and answer write were path-based
(`ReadFile`/`WriteFile` follow symlinks) with no type, ownership,
or existence discipline — a confused deputy read primitive (key
exfil into callbacks) and write primitive (clobber at agent-chosen
paths).

Correction (`gateway/filedrop.go`, adapter wiring): OpenRoot
dir-relative operations; question consume opens `O_NOFOLLOW` and
validates the descriptor (regular, agent-owned, io-group,
group-readable, ≤64KB), unlinks BEFORE reading (crash loses the
question for natural re-yield instead of risking stale yield),
malformed-but-owned still yields leniently; answer creation is
`O_CREAT|O_EXCL|O_NOFOLLOW` with ownership-defined EEXIST handling
(symlink/non-regular → remove; gateway-owned → atomic temp+rename;
agent-owned → remove+recreate; foreign → refuse) + explicit 0640
+ stale-temp sweep; Continue clears the question slot first (fresh
turn, no stale yield); present-but-invalid drops fail the run
honestly even on exit 0.

Regression: question-symlink→key refused (P2), answer-symlink→victim
refused (P3), pre-created regular safely replaced, oversized
refused, malformed yields leniently, concurrent replace hammers
clean, temp sweep (stale removed, live preserved).

Evidence: unit + concurrency (race ×3). Dual-user live proof is
runbook §7b.

### F4 — Concurrent duplicate execution

Root cause: check-then-execute-then-record with no in-progress
claim — concurrent identical deliveries both dispatched, and any
crash between execution and Record re-executed.

Correction (`gateway/dedup.go`, `gateway/server.go`,
`gateway/recover.go`, adapter sentinels): durable claims persisted
before dispatch (message id, payload digest, build/session/turn,
invocation id) in the existing store file (no second mechanism);
per-(build,message) call table shares one execution (digest
mismatches 409 without joining); per-build semaphore serializes
distinct commands through run end; crashes recover claims as
INDETERMINATE (plan-review A1 — reap orphan, keep stable-id events,
never relaunch; deliberate recovery is F6-as-new-turn); same-id
redelivery of indeterminate 409s naming continue; `Drop` only on
proven no-child (Runner contract: Start error ⇒ no child);
`Complete`-persist failure aborts the run honestly; busy (live run)
drops + transient 409, retryable.

Regression: N identical → 1 execution (P1), N distinct → serialized,
digest mismatch → immediate 409, emit-then-crash → no relaunch +
event exactly once + orphan reaped, pidless windows → indeterminate,
busy → drop → retry succeeds, claim-failure → zero dispatch,
abort semantics. Guarantee framed honestly as at-most-once dispatch
with explicit recovery.

Evidence: unit + concurrency + crash simulation (reopen).

### F5 — Premature terminal failure

Root cause: the translator mapped mid-run `error` lines to
`EventFailed`, so a recoverable error terminal-failed builds that
then exited 0.

Correction (`gateway/events.go`): error lines become
not-ok progress with diagnostic detail; terminality comes from the
exit code + question-drop only (drop still beats exit 0; nonzero
stays honest failure). Vendored run.ts confirms error lines are
mid-run (stream continues to idle) with exit 1 set at finish —
the rule coincides there and stays robust if the coupling loosens.

Regression: error-then-success `[progress message done]` (P4),
error-then-failure, yield-beats-exit, malformed-stream,
killed-run, plus corrected `TestAdapterFailureHonest`/translator
cases (old semantics were the pinned bug).

Evidence: unit (scripted transcripts).

### F6 — No continuation after recovery

Root cause: recovery terminalized records while the server 409'd
both re-assign and continue — recovered work was unusable and the
runbook's resume step unimplementable.

Correction (`gateway/server.go`, `gateway/opencode.go`): failed
sessions accept explicit authorized continue as a new fenced turn;
done/cancelled stay 409; the adapter verifies lineage (resumed
stream must report the requested ref — mismatch kills + reports
stray ambiguity); `resumed` emits only after verified launch;
launch failure restores the pre-resume state (failed stays failed);
no auto-restart, nothing fabricated; turn fencing M11 (record
tracks turn; absent → 400, mismatch → 409); CLI `--turn` +
runbook §11.5 rewritten to the implemented behavior.

Regression: full e2e recover → continue → resumed → completed
(server + real adapter, lineage + workdir + turn asserted),
done/cancelled 409s, stale/absent/next turns, lineage mismatch.

Evidence: integration (server+adapter, scripted agent process).

### F7 — Disruptive provisioning on shared builders

Root cause: every provision unconditionally deleted + recreated the
gateway service, killing co-located builds' agents (which then
wedged via F6) — plus a found boot-args gap (below).

Correction (`tools/build_provision.go`, `tools/build_upgrade.go`,
`fly/fake.go`, `tools/provisiondata/setup.sh`, `gateway/keys.go`,
`cmd/studio/gateway.go`, `cmd/gateway/main.go`): inspect-first
ordering (absent → install; running → digest-verified key refresh
with NO restart; unhealthy/inspection-error → fail closed, never
touch); drift refuses naming the explicit upgrade; upgrade is a
separate idle-guarded tool (all builds on the sprite verified idle
unless `--force`) + CLI op; per-sprite PG advisory lock (two-key
form) serializes serve + operator CLI across processes; M14 digest
printed + echoed in the tool result; `ReloadingKeys` lets the live
gateway pick up refreshed keys without restart; setup installs keys
atomically (temp + rename) with stale-temp sweep.
FOUND IN PASS: the provisioned service carried no binary args, so
the gateway would exit on usage error at boot. Fixed: service args
now pin `--state-dir/--work-root/--studio-url/--keys-dir`;
`studio_url` is a required, transport-validated provision/upgrade
input (explicit D3 decision); setup creates the state dir; CLI and
runbook carry `--studio-url`.

Regression: second provision preserves service (no delete/recreate,
keys refresh), unhealthy/inspection-error fail-closed, drift
refuses naming upgrade, digest echo, cross-session lock proof
(max in-flight 1, one shared builder), upgrade idle/live/force/
refusal paths, service-args shape, setup assertions.

Evidence: unit (fakes + httptest gateway) + concurrency.

### F8/F9 — Terminal outbox loss + stale callbacks

Root cause: terminally failed items had no backstop (the §16 poll
comment was fiction; only the CLI sent commands), envelopes were
signed once at emit while redelivered for hours (any item older
than 5 minutes 401'd into terminal failure), and pump enqueue
failures logged-and-dropped events.

Correction (`gateway/protocol.go`, `gateway/deliver.go`,
`gateway/outbox.go`, `gateway/server.go`, CLIs): per-attempt
re-signing with fresh timestamps under stable message ids
(stored bytes immutable); Studio skew check untouched;
per-build signed `outbox_inspect`/`outbox_requeue` commands
(scoped reads, ownership-checked requeue preserving identity +
recording the action); pump failures mark sessions degraded
(durable, status-surfaced) and drain undispatched (no wedge);
direct-emit failures abort + degrade; §16 comment replaced with
the requeue + Phase C sender-reconciliation requirement; M2
component-level attempt timeout; M12 https-except-loopback both
directions (+ serve boot validation); M13 CLI allowlist + fetch
truncation + `--output`.

Regression: aged envelope redelivers + drains with stable id,
stale-forged bytes stay 401, Resign identity unit, inspect
scoping + cross-build 404 + live-item 409, requeue-then-ack
drain, enqueue-failure → degraded + drained + status-visible,
hanging-Studio timeout, insecure-transport terminal, client
refuse/allow, CLI allowlist + formatter units.

Evidence: unit + integration (real HTTP stubs).

## 2. M1–M16 dispositions (all fixed, none carried)

M1 dir-fsync on all three atomic stores + helper test. M2
per-attempt deliverer timeout (component-level, hanging-stub
test — stronger than wiring-only). M3 SIGTERM + spawn decoupled
from request ctx + adapter Shutdown (cancel-spared + shutdown
tests). M4 process-group spawn/kill + group reap with verified
non-leader fallback (grandchild + fallback tests). M5 bounded
stderr ring (wedge + tail-cap tests). M6 partial recovery listing
+ emit-before-mark + deterministic ids + already-queued proceeds
(skip + idempotence tests). M7 argv[0]-basename + starttime reaper
(decoy + PID-reuse + comm-parser tests). M8 fd-pinned collector
(inside-symlink refusal + swap-hammer test). M9 scanner errors
surfaced (overlong-line truncation test). M10 64KB question cap
(oversize test). M11 turn fencing (absent/stale/next tests +
CLI/runbook). M12 https-except-loopback client + deliverer +
boot validation (refuse/allow/terminal tests). M13 CLI allowlist
+ fetch truncation + `--output` (allowlist + formatter tests).
M14 digest print + tool-result record (echo tests both tools).
M15 KDF negative — wrong-secret 401 (wiring test pre-existed;
review's zero-reference claim was stale). M16 header refresh +
`turn_id` populated inside the signed bytes + shortID
hex-validate-or-hash (mapping test).

## 3. Regression tests and results

Every finding reproduced red on the unmodified tree before its
fix (P1–P4 via `/tmp/b1probe`; per-phase reds in-suite), then
proved green without weakening any invariant. Final: 349 test
functions (gateway 99), full suite + race green on PG19,
probes P1–P4 green (see §5). No test was deleted to pass; three
tests pinning the old wrong semantics were corrected per the
review's explicit order (`TestAdapterFailureHonest`, two
translator cases) with their replacements asserting the corrected
contract.

## 4. Verification details (PG19 Docker only)

- Endpoint 127.0.0.1:5434 (container miniapp-p0-pg),
  `SHOW server_version` → `19.0 (Debian 19.0-1.pgdg13+1)`,
  confirmed before DB-backed work. No PG16/17 substitution
  anywhere in this pass; no skips.
- `gofmt -l .` empty; `go vet ./...` 0; `go build ./...` 0.
- `go test -p 1 ./...` → 13/13 ok.
- `go test -race -p 1` on the six required packages → all ok.
- Static `cmd/gateway` → ELF 64-bit x86-64 statically linked.
- `GOOS=darwin go build ./gateway/ ./cmd/gateway/` → ok.
- `sh -n tools/provisiondata/setup.sh` → ok.
- DSNs/credentials never emitted (derived in-shell, used, dropped).

## 5. Probe re-run (corrected tree)

- P1: `codes=[200 200] adapter.Continue calls=1`.
- P2: `kinds=[message failed]`, no key material exfiltrated.
- P3: victim intact, answer slot real + 0640.
- P4: `kinds=[progress progress message done]`.

## 6. Docs updated (same change as behavior)

- `docs/gateway-phase-b-evidence.md`: correction record appended
  (gates, probe outputs, five-state matrix v2, G1–G8 + M1–M16
  dispositions, deviations, residuals).
- `docs/gateway-phase-b-live-runbook.md`: §6 staging checks +
  refresh note, §6.5 upgrade procedure, §7b identity/inbox live
  gate (+ agent-report fallback), §11.5 resume rewrite, §12 new
  rows (indeterminate/stale-turn/degraded/requeue/stderr).
- `docs/opencode-run-v1.19.6.md`: vendored hash-verified run.ts
  audit record (blob SHA-1 matches the tag tree).
- `tools/provisiondata/{builder.md,setup.sh}`: contract section,
  staging hygiene, atomic keys, state dir.
- `plans/prompt11_plan.md`: A1/A2/A3 amendments (this pass's
  governing deltas).

## 7. Threat model and permission matrix (B1/F2/F3)

Trusted: gateway process (service uid — privileged in the
reference install), setup.sh (root, install time), Studio
control plane. Untrusted: agent processes (studio-agent), agent-
writable paths, all envelope payloads pre-verify. Protected:
per-build HMAC keys, tenant builds/sessions, gateway state,
co-located builds' files, callback integrity.

| Actor | Resource | Read | Write | Delete/replace | Denial proved by |
|---|---|---|---|---|---|
| gateway | keys `0600 studio-gw` | yes | rotation via provision | authorized | — |
| agent | keys dir/files | NO (EACCES) | NO | NO | live §7b (fake: modes) |
| agent | inbox `2730` | traverse | own drops | own only (sticky) | live §7b + unit modes |
| agent | answer `0640` | yes (group) | NO | NO | live §7b + EEXIST tests |
| gateway | question `0660 agent` | yes (group) | consume | yes (dir owner) | unit ownership tests |
| agent | workdir `2770` | yes | yes | yes | scoped to app builder |
| agent | gateway state `0700` | NO | NO | NO | modes |
| anyone | staging post-setup | — | — | keys removed | runbook §6 check |

Residual (disclosed, out of prompt11 scope): co-located builds
share the agent uid, so a compromised agent can reach same-app
sibling workdirs. Cross-tenant builders do not exist, bounding
blast radius to one tenant+app. Per-build agent identities are
the follow-up hardening.

## 8. Remaining live-only unknowns (R1–R10, unchanged home)

Warm/cold wake, wake latency, exit codes (P-G1), part internals
(P-G0), agent compliance (P-G3), real drop + EACCES (§7b),
exec privilege/service identity (§7), OAuth ceremony (P-E),
plugin behavior, egress/token handling — all assigned to Part 2
runbook checkpoints. Nothing in this pass claims them.

## 9. Stop point

Correction evidence complete. No commits made (tree all-untracked,
zero commits). No further general review cycle per the directive —
this package awaits correction verification and the separate Part 2
authorization decision. **Part 2 was not executed.**
