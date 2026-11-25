# Adversarial Code Review — Agent Gateway, Phase B Part 1

- Scope: `gateway/`, `cmd/gateway`, `tools/build_provision.go` +
  `tools/provisiondata/`, Studio gateway CLI + KDF wiring,
  `docs/gateway-phase-b-evidence.md`, `docs/gateway-phase-b-live-runbook.md`.
- Read-only review; no repository file created, modified, or deleted except
  this report. No live operations performed (no Sprites, no OAuth, no Fly
  mutations, no external contact).
- Date: 2026-11-25. Studio root: `/home/chaschel/Documents/go/miniapp-studio`.
- Note: prompt §10 names the output `plans/adv_review_phase_b_part1.md`;
  the review was ordered as `plans/adv_review_prompt10.md`. This file is
  that report under the ordered name.

## A. Executive verdict

**NOT ACCEPTED — one blocker prevents acceptance.**

The blocker (B1): **there is no OS-identity separation between the gateway
and the agent anywhere in the implementation**, violating binding invariant
7 ("Gateway HMAC signing keys must be inaccessible to the agent through an
operating-system-enforced identity boundary"). `setup.sh` creates the
`studio-gw`/`studio-agent` users and 0600 key files, but no code path uses
them: the supervisor spawns agent processes with no credential drop
(`gateway/supervisor.go:45-75`), the Sprite service request cannot express
a user (`fly/interfaces.go:62-66`), and provisioning sets none
(`tools/build_provision.go:181-183`). Whatever uid the service runs as, the
agent inherits it — so the agent can read the build HMAC keys (0600
owner-readable, plus world-readable 0644 copies lingering in
`/srv/studio-staging`), forge gateway→Studio callbacks, and reach every
co-located build's keys and files on shared per-application builders. The
failure is structural, not a hardening wish: worse, **no uid assignment
makes the current file-drop design work** — single-user exposes keys,
while dual-user breaks Q&A both directions (gateway-written 0600
`answer.json` unreadable by the agent; gateway-created 0755 inbox
unwritable by the agent). The design cannot be installed securely as
written, which fails this review's primary question directly.

The rest of the foundation is genuinely strong — protocol, client retry
semantics, atomic persistence design, translator defensiveness, collector
confinement, provisioning idempotency — and every other finding below has
a bounded, mostly fake-testable correction. Four of my adversarial probes
demonstrate live exploits (concurrent double-continue, question-symlink
key exfiltration, answer-symlink file clobber, premature terminal failure
on a mid-run error line); the recovery story contradicts the runbook
(resume-after-restart is unimplementable against the current 409s); and
provisioning restarts the gateway service on shared builders, killing
co-located agents. None of these requires a redesign — §J lists the single
bounded correction bundle. I do **not** authorize Part 2; after the §J
corrections land and their named regression tests pass, acceptance needs
only correction verification, not another general review.

## B. Architecture and invariant assessment

| # | Invariant | State |
|---|---|---|
| 1 | Studio PG is the authoritative record | Holds (gateway persists execution state only; callbacks are events, Studio owns transitions) |
| 2 | Builder-local files are execution state | Holds (session/outbox/dedup/inbox are runtime state; no business record lives only on the builder) |
| 3 | Builder/Runtime separation | Holds (no runtime-target code in Part 1 paths) |
| 4 | User talks to Studio, never Sprite terminal | Holds (operator CLI drives Studio tools/commands; agent contact is file+protocol only) |
| 5 | Authenticated HTTPS commands/callbacks, no permanent socket | Holds with limitations (HMAC both directions verified; client does not enforce https scheme — minor M12) |
| 6 | Gateway is a persistent service; agents recoverable after sleep/restart | Holds with limitations (recovery honestly marks failed; but no re-drive path exists — F6; sleep/wake unverified — R1) |
| 7 | HMAC keys inaccessible to agent via OS identity boundary | **Violated (B1)** — users exist, enforcement does not |
| 8 | Clarification/proposal/authorization distinct; no self-authorization | Holds (strict answer matching Studio-side; gateway passes kind through; agent cannot forge Studio-side decisions — though it can disregard answer files, see §D) |
| 9 | F1 deferred, G-01 open, consequential writes disabled | Holds (no new effect paths; unconfigured backends intact) |
| 10 | No production connector credentials to builders | Holds (nothing in bundle/provisioning carries connector credentials; OAuth ceremony is owner-held) |
| 11 | No framework changes without process | Holds (Miniapp untouched; public APIs only) |
| 12 | No commits without authorization | Holds (tree all-untracked, zero commits) |
| App | Miniapp-exclusive, Go-exclusive application output | Holds (no contradiction; instructions neutral — recommend reinforcement before artifact-bearing turns, §K) |

## C. Findings

| ID | Severity | Component | Exact location | Reproduction/evidence | Violated invariant | Required durable correction |
|---|---|---|---|---|---|---|
| B1 | BLOCKER | identity boundary | `gateway/supervisor.go:45-75` (no Credential); `fly/interfaces.go:62-66` (no user field); `tools/build_provision.go:181-183` (no user set); `tools/provisiondata/setup.sh:41-60` (users unused at runtime) | Repo-wide search: zero privilege-drop primitives; all tests single-user; `builder.md:40` falsely promises unreadability | Invariant 7 | Spawn agent as `studio-agent` (service runs privileged or setuid helper — owner decision); inbox perm design (group `studio-io` + sticky bit, 0640 answers, gateway-owned dir); staging 0600 + key cleanup; runbook live uid/read-denied checks |
| F2 | MAJOR | file-drop read path | `gateway/opencode.go:338-344` (`ReadFile` follows symlinks) | Probe P2: symlinked `question.json` → key file yields Body containing key material | §B key-material exclusion; §H directionality | `O_NOFOLLOW` + ownership/type check + size cap on question read; regression test with symlink + oversized drop |
| F3 | MAJOR | file-drop write path | `gateway/opencode.go:154` (`WriteFile` truncates through symlinks, no `O_EXCL`) | Probe P3: pre-created symlink `answer.json` → victim file clobbered with answer JSON | §H directional integrity; file integrity | `O_CREAT\|O_EXCL\|O_NOFOLLOW` write (+ safe replace protocol for retries); regression test with pre-created symlink/regular file |
| F4 | MAJOR | command dedup concurrency | `gateway/server.go:84-95` (check-then-act, no claim); `gateway/opencode.go:133-158` (check-then-launch) | Probe P1: two concurrent identical continues → both 200, adapter invoked twice | §C: concurrent deliveries execute once | Per-(build,message) singleflight or per-session claim mutex spanning check→record; regression test with N concurrent identical + distinct continues asserting exactly-one launch |
| F5 | MAJOR | translator terminality | `gateway/events.go:78-79` (`error` → `EventFailed`); `gateway/opencode.go:287-329` (pump forwards immediately) | Probe P4: recoverable mid-run error + exit 0 emits `failed` then `done`; pinned as designed by `TestAdapterFailureHonest` | Accurate terminal state (Studio would wrongly terminal-fail) | Terminality from exit code + question-drop only; mid-run errors → progress-with-detail; regression test: error-then-clean-exit yields single `done`, no `failed` |
| F6 | MAJOR | recovery re-drive | `gateway/recover.go:41-48` (marks failed); `gateway/server.go:153-155,199-201` (409s block re-assign and continue) | Code path walk: post-recovery build accepts neither assign nor continue; runbook §11.5 expects `agent.resumed` — unimplementable; no test covers recover→continue | Invariant 6 (recoverable agents); runbook/code consistency | Resume-from-failed path (fresh run, same workdir/lineage, honest `resumed` event) + runbook §11.5 fix + regression test recover→continue→resumed→completed |
| F7 | MAJOR | provisioning vs shared builders | `tools/build_provision.go:180-185` (unconditional service delete/create) + per-app/default pooling (`builderSpriteName`) | Code walk: each new build's provision restarts the service, orphaning co-located builds' agents (which then wedge via F6) | Availability; multi-build isolation on shared builders | Create-if-absent + key refresh without restart, or enforced single-build-per-builder; regression test: provision B on B's builder leaves A's live session untouched |
| F8 | MAJOR | outbox terminal loss | `gateway/deliver.go:14-17,95-108` (25-attempt/4xx → terminal Failed, kept only); comment cites nonexistent §16 poll | Only `Client.Send` caller is the operator CLI (verified by search); no requeue path; no automatic poller (sender loop deferred to Phase C) | At-least-once callback delivery; honest terminality | Operator requeue/inspect path + status surfacing of failed depth to Studio + comment correction + Phase C reconciliation requirement; regression test requeue-then-ack |
| F9 | MAJOR | callback freshness | `gateway/server.go:352-379` (sign-at-emit) + `gateway/deliver.go:91-96` (verbatim redelivery, 4xx terminal) + Studio 5-min skew verify | Code-path proof: envelope `SentAt` fixed at emit; any item undelivered >5 min (attempt ~8 under default backoff) gets Studio 401 → terminal Failed. A ~6-minute Studio outage permanently kills queued callbacks | At-least-once delivery; replay-window coherence | Re-sign at each delivery attempt (stable `MessageID` preserves dedup identity); regression test: aged envelope redelivers successfully after re-sign, stale-forged bytes still 401 |
| M1 | MINOR | durability (fsync) | `gateway/dedup.go:71-99`, `gateway/outbox.go:187-216`, `gateway/session.go:164-193` (no dir fsync after rename) | Code inspection: OS-crash window can lose the rename (process-crash safe) | Crash durability completeness | fsync parent dir after rename in all three stores; no new test needed (unit-testable with fault injection if desired) |
| M2 | MINOR | deliverer wiring | `cmd/gateway/main.go:88` (zero `Client` → `http.DefaultClient`, no timeout) | Code inspection: hung Studio stalls the single-threaded loop until shutdown | Head-of-line availability | Set `&http.Client{Timeout: …}` in wiring; regression test with hanging stub asserting pass returns |
| M3 | MINOR | shutdown behavior | `cmd/gateway/main.go:55` (SIGINT only); no agent signaling on shutdown | Code inspection: SIGTERM kills without drain; agents orphaned until next boot's recovery | Graceful lifecycle | Handle SIGTERM; signal tracked agents on shutdown; test shutdown path signals runs |
| M4 | MINOR | process groups | `gateway/supervisor.go:54,105-118` (no `Setpgid`; signals hit direct child only); `gateway/recover.go:31-40` (single-PID kill) | Code inspection: grandchildren survive cancel and recovery reap | Cancellation completeness | Process-group spawn + group kill on cancel/reap; regression test with grandchild sleeper |
| M5 | MINOR | stderr drain | `gateway/supervisor.go:61-65` + adapter never reads stderr during runs | Code inspection: >64KB stderr wedges the agent; runbook §12 misattributes such hangs to prompt/agent | Liveness; runbook accuracy | Drain stderr to bounded discard/log; correct runbook §12 row |
| M6 | MINOR | recovery robustness | `gateway/recover.go:21-25` (`List` error aborts all recovery); `:41-47` (Update-then-emit crash window; emit failure silent) | Code inspection: one corrupt record blocks every build's recovery; crash between mark and emit loses the failed callback | Recovery completeness | Per-record skip-and-continue; emit-before-mark or mark-dirty retry; regression tests for corrupt-record-skip |
| M7 | MINOR | reaper precision | `gateway/recover.go:62-76` (substring match on any argv element) | Code inspection: PID reuse + `opencode` in args (e.g. editor) kills a foreign process | Kill safety | Match argv[0] basename + start-time check; regression test with decoy cmdline |
| M8 | MINOR | collector TOCTOU | `gateway/collector.go:40-61` (resolve/stat then re-resolving read) | Code inspection: concurrent agent can swap file→outside-symlink between check and read | Artifact confinement | Open `O_NOFOLLOW`, `fstat`, read from fd; regression test with swapping fixture (best-effort) or code-review sign-off |
| M9 | MINOR | translator robustness | `gateway/opencode.go:309-313` (`scanner.Err()` ignored; 4MB line death ends observation); `events.go` zero counters on truncation | Code inspection: overlong line silently truncates the event stream mid-run | Observability completeness | Log `scanner.Err()` into corrupt tally + status detail; regression test with overlong line |
| M10 | MINOR | drop size caps | `gateway/opencode.go:338-351` (unbounded `ReadFile`) | Code inspection: giant `question.json` OOMs the gateway | Resource bounds | Size cap with clean refusal; regression test with oversized drop |
| M11 | MINOR | turn fencing (gateway layer) | `gateway/server.go:194-223` (terminal check only); `Turn` fields parsed but unused (`opencode.go:87-90`, adapter ignores) | Code inspection: stale-turn continue enforced Studio-side only; gateway obeys blindly | Defense in depth (§C) | Track turn in session record; refuse mismatched continue; regression test stale-turn continue → 409 |
| M12 | MINOR | transport scheme | `gateway/client.go:59` (no https enforcement); provision records API-issued URL verbatim | Code inspection: prompts (possibly secret-bearing) would cross plaintext HTTP if a URL ever were | Secret transport | Enforce https (or document private-network exception with a test pinning it) |
| M13 | MINOR | CLI sharp edges | `cmd/studio/gateway.go:83-131` (unvalidated `--cmd`; full response incl. 50MB `data_b64` printed) | Code inspection: typos fail only at gateway; huge terminal dumps | Operator safety/UX | Client-side command allowlist; `--output`/truncation for fetch responses |
| M14 | MINOR | binary provenance | `cmd/studio/gateway.go:48-51` (deploys whatever path is given, no digest) | Code inspection: stale/wrong binary deploys silently; "running" check passes for any serving binary | Deployment identity | Print + record sha256 at provision; runbook compares with local `sha256sum`; regression test asserting digest echo |
| M15 | MINOR | KDF path untested | `server/server.go:162-165` wired; all callback tests use `StaticProvider` | Search evidence: zero `KDFProvider` references in any test | Production-auth-path coverage | End-to-end callback test through `KDFProvider`+`DeriveKey` round trip |
| M16 | MINOR | misc consistency | `gateway/protocol.go:1-6` (stale Phase-A header); `Envelope.TurnID` never populated (`server.go:352-379`); `shortID` unsanitized in key filenames (`tools/build_provision.go:196-201`) | Code inspection | Code/doc hygiene | Header refresh; populate or remove `turn_id`; hex-validate build ids in filenames (defense in depth; Studio mints hex today) |

Risks (live-unverified; each names its proof — all already assigned to runbook checkpoints except R6's auditability ask):

| ID | Risk | Proof needed |
|---|---|---|
| R1 | Wake-on-command latency/behavior (warm vs cold) | Runbook §11 timing table (both idles) |
| R2 | Exit-code semantics (0 idle-ok / 1 error) | Runbook P-G1: forced-error run + clean run, exit codes recorded |
| R3 | NDJSON part internals | Runbook P-G0: captured transcript vs translator expectations |
| R4 | Agent file-drop compliance (drop shape, last-act discipline, answer-file reading) | Runbook P-G3 against real OpenCode with transcript |
| R5 | Sprite exec privilege (root for setup) | Runbook §7 discovery (fail-closed refusal is an acceptable outcome) |
| R6 | CLI flag shapes (`run --format/--dir/--model/--agent/--session`) + `run.ts` claims unauditable from repo (no vendored source) | Vendor the source excerpt or its digest for auditability; P-G0 live-confirm |
| R7 | Plugin behavior + `guidance:false` semantics | Live §6 plugin verification + P-E ladder outputs |
| R8 | OAuth ceremony (P-E1–P-E3) | Owner ceremony record (§8) |
| R9 | Sleep/wake PID survival + recovery-callback honesty | §11 warm/cold observations (both outcomes acceptable, must be recorded) |
| R10 | Egress/network policy + MCP token handling | P-E outputs; token-redaction discipline in evidence |

## D. File-drop Q&A assessment

**Not trustworthy as implemented. The mechanism is well-shaped but its
security properties do not hold under any runnable uid assignment, and two
concrete symlink attacks succeed today.**

How it works (verified in code): the agent writes
`<workdir>/.studio/inbox/question.json` (`{kind, body, options[]}`); at run
end the gateway reads + deletes it (`readQuestionDrop`), yields a question
event, and marks the session yielded. On Studio continue, the gateway
writes `answer.json` (`{question_id, kind, answer}`) and relaunches
`opencode run --session <ref>` with instructions to read the answer first.
Single-slot overwrite gives deterministic supersede semantics with no turn
numbers to desynchronize — a sound core idea.

Directionality (the decisive question): **broken three ways.**

1. **No usable permission assignment exists.** Inbox and workdir are created
   by the gateway (`os.MkdirAll … 0755`); answers are written 0600; the
   agent runs as the gateway's own uid (B1). Single-user: the agent reads
   keys and can delete/replace `answer.json` (writable dir). Hypothetical
   dual-user: the agent could neither write questions (non-writable dir)
   nor read answers (0600). `setup.sh` sets no inbox permissions at all.
   The shared `studio-io` group is currently decorative for the inbox.
2. **Agent→gateway file-read primitive (F2, proven).** `readQuestionDrop`
   follows symlinks with no type/ownership check: my probe symlinked
   `question.json` to a key file and the yielded question body contained
   the key material — a confused-deputy exfiltration of gateway secrets
   into Studio-bound callbacks.
3. **Agent→gateway file-write primitive (F3, proven).** `Continue` writes
   `answer.json` with truncating `WriteFile` (no `O_EXCL`/`O_NOFOLLOW`):
   my probe pre-created the path as a symlink and the continue clobbered
   the target with answer JSON. Pre-creating a *regular* file does not
   forge answers (the gateway overwrites), and a resumed agent that
   ignores the answer file is non-compliance no permission scheme can
   fix — but clobbering gateway files at agent-chosen paths is a genuine
   integrity break (keys file included).

Turn binding: absent gateway-side by design (single-slot overwrite;
`Turn` fields decorative). Deterministic and adequate *given* Studio-side
fencing (answers apply to open questions only) — but a stale/fabricated
Studio continue is obeyed blindly (M11). Crash behavior is mostly sound:
question consumed exactly once per run end (tested); crash between read
and delete cannot double-yield because recovery terminalizes instead of
re-pumping; answer write is non-atomic (torn file on crash — minor,
recovery terminalizes anyway).

**Real OpenCode behavior was NOT demonstrated — only simulated.** Every
adapter assertion runs through `ScriptedRunner` canned transcripts; the
supervisor tests use real generic processes (sleep/echo-class), never the
OpenCode binary; no Sprite execution occurred. The `run.ts` @ v1.19.6
citations (question-tool denial, NDJSON shapes, exit semantics) reference
source that is not vendored or hashed anywhere in the repo, so they are
unverifiable in review. P-G3 as redefined (agent compliance with the file
convention) is the right live proof and must run against real OpenCode
with a recorded transcript — the current evidence honestly labels it
future work.

## E. Durability and recovery assessment

Command deduplication: the persisted store itself is well built (atomic
temp+fsync+rename, mutex-guarded, corrupt→fail-closed, reopen-tested) —
but it records only *after* full execution with no in-progress claim, so
**crash between execution and Record re-executes** (assign converges to a
divergent 409; cancel converges to a divergent already-true; continue
re-resumes), and **concurrent identical deliveries double-execute** (F4,
proven: two 200s, two adapter invocations). Recording every status
including 500s is correct dedup semantics (same id → same response) with
one operational consequence the docs should state: transient-failure 500s
replay until Studio mints a fresh id. Untested crash windows: dispatch→
Record (above); `Start` launch-ok→session-Update-fail (stray running
agent + broken record); `Record` persist-failure ignored (disk-full
re-execution). The store grows unboundedly (no eviction — fine at
command volumes, note for years-long builders).

Outbox: the durable core is sound (atomic whole-file rewrites, sending→
queued reset on open, id-keyed dedup with Studio, backoff+jitter, 429
retried, 4xx terminal, attempt cap). Holes: **terminal failure has no
backstop** (F8 — the §16 poll comment is fiction; only the CLI sends
commands); **pump drops events when enqueue fails** (log-and-continue —
disk-full/corrupt loses even terminal events); single-threaded deliverer
with an **untimeouted HTTP client** stalls head-of-line on a hung Studio
(M2); no dir-fsync (M1); failed items accumulate with no requeue path
(F8). Retries preserve envelope bytes, so Studio dedup holds; ordering
is not preserved across backoff, which Phase A absorb semantics tolerate.

Supervisor/sessions: spawn/reap-once/exit-codes correct and really tested
against live processes; never-restarts honored; session records atomic
with escape-proof keys. Gaps: no process groups (M4 — cancel/recovery
leave grandchildren); stderr undrained (M5 — 64KB wedge risk);
`exec.CommandContext` with the request context lets client disconnects
kill agents (use `WithoutCancel` for spawn lifetime — fold into M3);
`List` aborts whole recovery on one corrupt record (M6); reaper argv
matching overbroad (M7).

Restart behavior: boot order is right (recover → deliver → listen) and
recovery is honest (failed + callback, no fake resume). But recovery is a
dead end: **no re-drive path exists** (F6), the Update→emit window can go
silent (M6), and SIGTERM/shutdown orphans agents until the next boot
(M3).

## F. Security assessment

Tenant isolation: sound at every enforcement boundary I could reach.
Per-build KDF keys (domain-separated) mean cross-build command replay
fails MAC; unknown builds 404 without oracle both directions; workspace
echoes from signed assign payloads and is re-validated by Studio's
tenant-scoped build lookup (operator workspace mismatch fails closed —
verified in `server/gateway_callback.go:50-69`); sessions keyed by build;
workdirs per build with sanitized segments; turn/state machines
server-side. The isolation that FAILS is intra-builder file/process
isolation (B1/F2/F3/F7): on shared per-application builders, builds are
separated by convention only.

HMAC/secrets: protocol core is disciplined (canonical JSON, constant-time
compare, 5-min skew both directions, strict envelope + trailing-data
rejection, empty-key refusal). Key lifecycle: generation = KDF
(un-stored Studio-side ✓); provisioning delivery = staged key file;
storage = 0600 + 0700 dir (good) **plus 0644 staging copies that linger
forever** (B1 evidence — `fly/sprites.go:179`); runtime loading refuses
bad modes ✓; rotation = re-provision (workable, no prune of stale keys —
record under credential-revocation operations); cleanup = absent
(staging). StaticProvider is structurally contained (defined once, zero
production references, unconfigured → warn + 503). Secret redaction:
`flyErr` caps at 300 chars but *asserts* API errors never carry
credentials — WriteFile-failure paths stage key bytes, so an echoing
error would persist key material in invocation records; I could not
verify sprites-go error shapes offline — treat as a live-evidence item
(inspect one staged failure output), minor.

Filesystem/permissions: keys dir 0700 ✓ (as specified); gateway binary
root-owned 0755 ✓; opencode in root-owned `/usr/local/bin` (shadowing
requires PATH games — service env PATH should be pinned and asserted,
fold into B1 correction); workdir 0755 world-readable (agent outputs
visible to any uid — acceptable single-purpose host, note); inbox
unusable-or-unsafe as analyzed (B1/F2/F3); collector confinement solid
except TOCTOU (M8); session/dedup/outbox state dirs 0700 via MkdirAll —
created by the service user, consistent.

Callback trust: gateway→Studio callbacks carry key-derived MACs under
per-build keys; Studio verifies + tenant-scopes; replay within skew
dedups on envelope id. **Signing-time vs delivery-time confusion (F9):**
envelopes are signed once at emit, but the deliverer redelivers identical
bytes across a backoff spanning hours — any item still undelivered 5
minutes past signing gets Studio 401 → deliverer marks terminally Failed.
A mere ~6-minute Studio outage (backoff crosses the skew window around
attempt 8) thus permanently kills queued callbacks, compounding F8.
Correction: re-sign at each attempt (stable MessageID preserves dedup
identity); do not weaken Studio's skew check for redeliveries.
## G. Evidence and capability-matrix corrections

Line-by-line against `docs/gateway-phase-b-evidence.md` (Part 1 claims
only; the runbook is assessed after):

1. **"Thirty-six gateway tests" — wrong, and M8 claims exact counts.**
   The package holds 39 `Test` functions (counted). Trivial by itself;
   it matters because M8 explicitly vouches for count accuracy.
2. **File-drop row: "IMPLEMENTED+TESTED (doubles); AGENT COMPLIANCE
   unverified" — materially incomplete.** The doubles prove the happy
   path only. Missing from the matrix: no directionality tests exist
   (agent overwrite/symlink attempts untested — F2/F3 demonstrate live
   breaks), and the dual-user permission design the row implicitly
   assumes is specified nowhere executable (B1). Correct state:
   IMPLEMENTED+TESTED (happy path, doubles); DIRECTIONALITY UNPROVEN
   (now disproven — see this review); agent compliance unverified.
3. **"live P-G3" (§5 question — answered precisely).** The phrase occurs
   only as forward reference ("live P-G3 now proves agent compliance",
   matrix "live P-G3"). No live Sprite work occurred; **the actual
   OpenCode binary was never exercised anywhere** — every adapter
   assertion runs through `ScriptedRunner` canned transcripts, and the
   supervisor tests use real *generic* processes, never `opencode`.
   No test ran inside a real Sprite. The wording is awkward ("now
   proves" in present tense for future work) but the state column is
   honest. Correction: rephrase to "live P-G3 *will* prove…", and
   record that P-G3 must run against real OpenCode with a transcript.
4. **"verified: CLI denies it" / "SPECIFIED from source" rows — source
   unauditable.** `run.ts` @ v1.19.6 is cited in five places but is not
   vendored, hashed, or linked anywhere in the repo. The claims may be
   true, but no reviewer can check them here. Correction: vendor the
   excerpt or pin its digest; until then these rows are
   SPECIFIED-FROM-UNPINNED-SOURCE, and P-G0/P-G1 carry the full weight.
5. **Outbox row omits the terminal-loss paths.** "Outbox durable +
   deliverer retry/terminal" is listed IMPLEMENTED+TESTED with no note
   that terminal Failed items have no backstop or requeue (F8), that
   >5-minute-old items can never deliver (F9), or that pump enqueue
   failures drop events. The unit behavior is tested; the
   *durability story* is overstated. Correct with F8/F9 notes.
6. **Dedup row omits the concurrency hole.** "Command dedup + stored
   replay, persisted" is tested only sequentially (`TestDedupSurvives
   Reopen`); concurrent duplicate execution (F4, proven) contradicts
   the unqualified claim. Correct with F4 note.
7. **Boot-recovery row omits the dead end.** "Boot recovery (reap+
   failed+callback)" is tested, but recovery leaves builds with no
   re-drive path (F6) — the row should state the terminal consequence.
8. **`flyErr` "never credentials" is asserted, not proven.**
   `tools/build_provision.go:205-208` claims API errors echo context,
   never credentials — but staged WriteFile payloads *are* key bytes,
   and sprites-go error shapes were not inspected here. Downgrade to
   "capped; credential-echo unverified — inspect one staged failure
   output live" (minor).
9. **"No false success based solely on HTTP 200" (provisioning)** —
   accepted: service-state check exists. But no gateway health proof
   follows (behavioral proof deferred to runbook §9 — acceptable,
   accurately labeled as the live check).
10. **Static-binary row — accepted as written** (re-verified: ELF
    64-bit x86-64 statically linked; note the binary ships unstripped
    with debug info — fine, aids live debugging).
11. **Gate claims — accepted** (reproduced on PG16 instead of PG19/17 —
    environment deviation recorded in §J; all green incl. race on all
    six packages).

Runbook consistency: §§0–5, 7–10, 12–14 are careful, exact, and honest
about unknowns (D1–D6 all present; cleanup-or-retention with
empty-list proof; pin-drift policy; forbidden fourth rungs). Two
defects: **§11.5 contradicts the code** (continue-after-recovery 409s —
F6; must be rewritten around whatever resume path the correction adds),
and **§12's "agent run hangs" row misattributes** stderr-pipe wedges to
prompt/agent issues (M5 — add the gateway-side check). §6 should add a
staging-cleanup/0600 verification once B1's correction lands, and §7
should gain the uid/read-denied proofs (agent cannot read keys dir;
agent can use the inbox) as explicit observe-and-record checks.

## H. Accepted design decisions

Do not redesign these; corrections in §J extend them:

- v1 envelope + canonical JSON + HMAC + 5-min skew + strict parse
  (both directions); per-build KDF keys with domain separation.
- `FileKeys` mode refusal; nil-keys fail-closed (503 Studio-side,
  boot refusal gateway-side); StaticProvider structurally test-only.
- Client same-bytes redelivery with 4xx-terminal / 429+5xx-retry
  classification and bounded backoff+jitter.
- Atomic whole-file persistence (outbox/dedup/sessions) with
  corrupt-fail-closed and reopen-tested recovery semantics.
- Outbox state machine (queued/sending/acked/failed) with attempt
  counting and terminal 4xx handling.
- Supervisor never-restarts + reap-exactly-once + honest exit codes.
- Translator defensive posture (skip unknown/corrupt loudly-counted,
  foreign-session skip) — modulo F5's terminality fix.
- Collector confinement model (clean+anchor+symlink-resolve+allowlist+
  caps) — modulo M8's fd hardening.
- Single-slot file-drop handshake shape (question/answer files,
  deterministic overwrite-supersede, no turn-number sync) — modulo
  B1/F2/F3 permission and symlink hardening.
- Provisioning idempotency (deterministic names, attach-on-retry,
  token-only URL lock, staged bundle, setup-then-verify) — modulo F7's
  no-restart refresh.
- Operator CLI explicit targeting, tenant-bound URL lookup, fresh keys
  per invocation, no secret echo.
- Callback 404-without-oracle both directions; tenant re-validation by
  scoped build lookup.
- Miniapp/Go exclusivity posture: no contradiction introduced; OpenCode
  remains behind the adapter contract. Before artifact-bearing turns,
  add the application contract (Go + Miniapp + MiniappSpec + runtime)
  to `builder.md`/assignment construction so the default is not
  "whatever stack the agent prefers" — a Phase D prerequisite, not a
  Part 1 defect.

## I. Deferred requirements

Valid issues that belong to later phases (do not gate Part 1):

- M5 (SSE stream lifetime + connection budgets) → live-segment
  checklist, unchanged.
- M6 (framework `app.invocations` residual) → carried; verified no
  gateway code path selects invocation inputs (gateway has no DB
  access at all).
- M7 (artifact-manifest sensitivity + Phase D read-path scrubbing) →
  Phase D; add M8-collector and answer/command-payload sensitivity to
  the same scrub-on-read requirement.
- Builder lifecycle/retirement, compromise quarantine, credential
  revocation (+ stale-key pruning), resource accounting, quotas and
  concurrent-build limits, callback/outbox observability → multi-tenant
  operations / Phase C+ (F8's requeue is the one piece pulled forward).
- Studio sender loop (DB dispatch) → Phase C as recorded; it must
  include reconciliation for outbox-failed items (F8 follow-through)
  and the formerly-§16 backstop semantics.
- OAuth ceremony, wake-timing tables, exec-privilege discovery →
  Part 2 runbook as written.
- R1–R10 live proofs → Part 2 checkpoints (each already mapped above).

## J. Final acceptance conditions

**The bounded correction task (no general re-review after it):**

1. **B1 — enforce the identity boundary.** Decide: privileged gateway
   service + `setuid`-to-`studio-agent` spawn (`SysProcAttr.Credential`
   or `setpriv`), or setuid helper — then implement: inbox dir owned
   by gateway, group `studio-io`, mode 2730 (group-writable + sticky);
   `answer.json` 0640 `studio-gw:studio-io`; question drops
   agent-writable but gateway-consumed with F2 checks; staging key
   files 0600 + deleted after install; service env PATH pinned to
   root-owned dirs. Tests: mode assertions on every created path;
   symlink-attack tests (P2/P3 inverted to refusals); runbook §7 gains
   live uid/read-denied proofs. Done when: tests pass + a same-tree
   adversarial re-probe (my four probes, re-run) shows P2/P3 refused.
2. **F2/F3 — symlink-safe drops.** `O_NOFOLLOW` + file-type/ownership
   checks + size caps on question reads; `O_CREAT|O_EXCL|O_NOFOLLOW`
   answer writes with a safe replace protocol. Tests: symlink +
   oversized + pre-created-file cases. Done when: P2/P3 probes refuse.
3. **F4 — deduplication claim.** Per-(build,message) singleflight (or
   per-session mutex) spanning Check→dispatch→Record; same for
   adapter Continue. Test: N concurrent identical continues ⇒ exactly
   one adapter invocation; N distinct ⇒ N ordered runs. Done when: P1
   probe shows one execution.
4. **F5 — honest terminality.** Translator maps mid-run `error` lines
   to progress-with-detail; terminal callbacks only from exit code +
   question-drop. Update `TestAdapterFailureHonest` to the corrected
   semantics + add error-then-clean-exit test. Done when: P4 probe
   shows `[progress, progress, message, done]`, no `failed`.
5. **F6 — resume after recovery.** Continue-from-failed starts a fresh
   supervised run reusing workdir/session lineage and emits honest
   `resumed`; rewrite runbook §11.5 to match. Test: recover →
   continue → resumed → completed end to end. Done when: that test
   passes and §11.5 describes it exactly.
6. **F7 — non-disruptive provision refresh.** Create-if-absent service
   + key refresh without restart (or code-enforced single-build
   occupancy with a clean refusal). Test: second provision on a live
   builder leaves the first session running. Done when: test passes.
7. **F8/F9 — callback durability honesty.** Re-sign envelopes at each
   delivery attempt (stable MessageID); add operator outbox
   inspect/requeue path; correct the §16 comment; record Phase C
   reconciliation. Tests: aged-envelope redelivery succeeds;
   requeue-then-ack drains a failed item. Done when: both pass.
8. **Evidence corrections G1–G8** applied to the evidence doc and
   runbook (§11.5, §12, §6, §7 additions).
9. **Minors M1–M16** fixed or explicitly carried with owner sign-off
   (each is small; none may silently drop).

**Completion criteria:** the nine items above landed, their named
regression tests green, `gofmt`/`vet`/`build`/full suite/race green,
and my four probes re-run against the corrected tree showing
P1 single-execution, P2/P3 refused, P4 honest. Verification of that
correction — not another general review — is then sufficient to accept
Part 1.

**What Part 1 has proved (durable):** the v1 command/callback protocol
with per-build authenticated envelopes; a correct client retry model; a
sound atomic-persistence pattern used consistently; an honest,
never-restarting supervisor; a defensive translator and a confining
collector; idempotent provisioning with fail-closed setup; an operator
CLI that cannot cross tenants by accident. **What remains unverified
before Part 2:** everything in R1–R10 — no real OpenCode process has
run, no Sprite exists, no wake/sleep behavior is observed, and the
`run.ts` citations are unaudited. Part 2 authorization additionally
requires the §J corrections, because several (B1, F2/F3, F6) change
code the live procedure installs.

**Verification record (this review):** `gofmt -l .` clean;
`go vet ./...` exit 0; `go build ./...` exit 0;
`CGO_ENABLED=0 GOOS=linux go build ./cmd/gateway` → ELF 64-bit x86-64
statically linked (re-verified); `go test -p 1 ./...` 13/13 ok;
`go test -race` on `gateway`/`store`/`tools`/`server`/`cmd/studio`/
`cmd/gateway` all ok; adversarial probes P1–P4 (concurrent continue,
question-symlink exfil, answer-symlink clobber, mid-run error
terminality) all reproduced as reported — probe code at `/tmp/b1probe`
(outside the repo, temp dirs only). Environment deviation: disposable
PostgreSQL **16** (user-space `initdb`, trust auth, destroyed after each
run) substitutes for the evidence's PG19/PG17, whose ports are
unreachable from this sandbox; the exercised surface (DDL/RLS/pgx) has
no version-sensitive constructs. No live infrastructure was contacted;
Miniapp and frozen docs untouched; tree remains all-untracked, zero
commits.

