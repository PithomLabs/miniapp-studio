# Phase B live runbook — Builder Gateway installation & recovery (OWNER-RUN)

**Status: DRAFT — requires explicit Part 2 GO before ANY step executes.**
Nothing in this document has run against live infrastructure. Part 1
(fake-backed implementation) is complete and green; this runbook is its
prescribed live verification. Do not improvise commands: every live
action below is exact, and every observation point says what to record.

Conventions: `$REPO` is the miniapp-studio checkout (this working tree —
see §2); all `studio`/`gateway` commands run from `$REPO` unless noted.
`BUILD` is a Studio build id from the conversation UI.

## §0 How to use this runbook

1. Collect the §1 decisions first. Nothing proceeds without all of them.
2. Work top to bottom; each section ends with evidence to record.
3. On any unexpected output, go to §12 (failure interpretation) before
   retrying or improvising.
4. Finish with §13 (cleanup or explicit retention) no matter what.

## §1 Owner decisions required (all blocking)

| # | Decision | Options | Default if deferred |
|---|---|---|---|
| D1 | Fly organization for builders | name it | — (blocking) |
| D2 | Model provider + auth for OpenCode | provider; owner-provisioned auth | — (blocking for agent runs) |
| D3 | Studio callback reachability | (a) owner machine + tunnel, (b) Fly-hosted Studio | — (blocking; see §9) |
| D4 | Builder retention after proof | destroy to zero, or retain named builder | destroy to zero |
| D5 | System packages (node≥20, bun, git, curl) | preinstall in image, or apt-install live | discover in §7, decide then |
| D6 | Pin drift | re-verify per §2; any drift needs explicit re-pin approval | hold pins |

D3 note: the gateway POSTs callbacks to Studio's `/gw/v1/callback`, so
Studio must be reachable FROM the sprite during live proofs. Pick (a) or
(b) before §9; the steps branch on it. There is no third path that
doesn't weaken the proof — do not substitute a request-catcher.

## §2 Pins and tree state (re-verify, record, do not float)

```bash
git status --porcelain            # record verbatim (repo has no commits yet: expect all-untracked tree)
go version                         # record
curl -fsSL https://registry.npmjs.org/@flydotio/sprites-opencode-plugin \
  && echo "NPM-PUBLISHED (stop: needs re-pin decision)" || echo "npm-404-as-expected"
curl -fsSL https://api.github.com/repos/anomalyco/opencode/releases?per_page=10 \
  | grep -o '"tag_name":"v1[^"]*"' | sort -u    # expect: v1.19.6 only
```

Expected pins (Part 1 values; any difference stops the run for a re-pin
decision): OpenCode v1.19.6, asset `opencode-linux-x64.tar.gz` sha256
`47734e08…09a057` (full: `47734e08968b30cc22a0bffb6d9feeed76ad4aa743213a12122040392d09a057`),
plugin commit `f63a85a7da9fa2f42b21877ab58b35d18ccaa682`, sprites-go
v0.2.2, Miniapp `73351a8`.

Record: full command outputs + `git status` in the evidence.

## §3 Pre-live gates (prove the tree before touching live infra)

```bash
gofmt -l .                # expect: no output
go vet ./...
go build ./...
go test -p 1 ./...        # expect: all ok (needs TEST_DATABASE_URL)
```

Record: suite tail. Any failure stops the run.

## §4 Build the gateway binary

```bash
CGO_ENABLED=0 GOOS=linux go build -o /tmp/studio-gateway ./cmd/gateway
file /tmp/studio-gateway  # expect: ELF 64-bit LSB executable, x86-64, statically linked
```

Record: the `file` line. Copy the binary path for §6.

## §5 Environment for operator commands

The CLI loads the standard Studio environment plus the org secret:

```bash
export STUDIO_OPERATORS_FILE=... STUDIO_DATABASE_URL=... STUDIO_OWNER_DATABASE_URL=...
export MINIAPP_KEYS_FILE=... MINIAPP_GRANTS_FILE=...
export STUDIO_FLY_ORG='<D1>'
export STUDIO_SPRITES_TOKEN_FILE=/path/to/org-token  # 0600, owner-held
export STUDIO_GATEWAY_ORG_SECRET_FILE=/path/to/org-secret  # 0600, owner-held; see below
```

Org secret: 32+ random bytes, generated once (`head -c 32 /dev/urandom
| base64 > file; chmod 600 file`), stored 0600. It derives ALL build
HMAC keys — compromise rotates every builder. Never print it; the CLI
never echoes it (verify: no secret in any output below).

## §6 Provision the builder

Pick a build first: in the Studio UI, open any conversation and send one
message (creates conversation + build `BUILD`; record both ids).

```bash
go run ./cmd/studio gateway provision --tenant '<TENANT>' --build '<BUILD>' \
  --gateway-binary /tmp/studio-gateway --studio-url '<STUDIO_URL>'
```

`<STUDIO_URL>` is the D3 decision (tunnel or Fly-hosted base URL —
the provisioned service boots with it as its callback target; without
it the service cannot start, so there is no default).

Expected: pretty JSON `{"build_id": ..., "sprite_name":
"studio-bld-<t8>-<a8>", "url": "https://..."}`. Record it verbatim.

Then verify the sprite directly (independent oracle, not our own output):

```bash
sprite info -s '<SPRITE>'        # expect: URL matches, auth token-only
sprite exec -s '<SPRITE>' -- ls -la /srv/studio-gateway /srv/studio-staging
```

Expected: gateway binary, `builder.md`, service files present;
`opencode --version` prints `1.19.6...`; plugin dir at the pinned
commit (`git -C /srv/studio-agent/plugin rev-parse HEAD` equals
`f63a85a…`); `/srv/studio-gateway/keys/` holds `keys-<build12>.json`
0600 `studio-gw`-owned. Record every output.

Staging hygiene (B1 — verify, do not assume): staging key copies must
be locked down and then removed by setup:

```bash
sprite exec -s '<SPRITE>' -- ls -la /srv/studio-staging/ | grep -c 'keys-.*\.json' ; echo "expect: 0 (cleanup ran)"
sprite exec -s '<SPRITE>' -- stat -c '%a %U %n' /srv/studio-staging /srv/studio-gateway/keys /srv/studio-gateway/keys/*
```

Expected: no `keys-*.json` remains under staging; staging is `700`,
keys dir is `700 studio-gw`, each key file is `600 studio-gw`.
Anything else stops the run (setup regression — report verbatim).

Provisioning a second build on the same builder refreshes keys WITHOUT
restarting the service (create-if-absent + digest-verified refresh).
Verify once: after §10 completes, provision a second build on the same
builder and confirm the first build's session is undisrupted
(`status` on the first build still answers; no new `agent.failed`).
Record the observation.

## §6.5 Gateway upgrade (explicit replacement only)

Provisioning NEVER replaces a running gateway binary. When the binary
must change (drift refusal names this op), replace it explicitly:

```bash
go run ./cmd/studio gateway upgrade --tenant '<TENANT>' --build '<BUILD>' \
  --gateway-binary /tmp/studio-gateway --studio-url '<STUDIO_URL>'
```

Expected: JSON with `"upgraded": true`, `"gateway_sha256"`, and
`"idle_checked"` (every build on the builder verified without a live
agent run). A live run refuses without `--force`; `--force` replaces
anyway and reports `"force": true` — use only after draining agent
runs deliberately. After upgrade, re-run §6 verification (service
running, keys present) and §9 (status answers).

## §7 Privilege and package discovery (honest unknowns — observe, decide)

Run and record verbatim:

```bash
sprite exec -s '<SPRITE>' -- id -u; echo "---"; sprite exec -s '<SPRITE>' -- 'command -v node bun git curl; node --version; bun --version'
```

- If `id -u` ≠ 0 and no sudo path exists: STOP. Setup cannot create the
  dual-user layout; the fail-closed refusal is the correct outcome.
  Report back — do not weaken to single-user.
- If node/bun/git/curl missing: install per D5 (record exact commands),
  then re-run provision (idempotent: attaches, re-installs, refreshes
  service) and re-verify §6.
- Re-run provision once more regardless: expect success with the SAME
  sprite name (attach path), proving retry convergence.

## §7b Identity and inbox proofs (B1 live gate — observe, record)

The fake-backed suite proves refusal and plumbing only. These checks
prove the real boundary on the builder. Prefer direct observation
via an agent-uid channel (`runuser` below needs a root exec channel —
R5 decides that); if the exec channel cannot assume the agent uid,
use the agent self-report fallback and record the weaker evidence
explicitly (it relies on agent compliance, so it supplements but does
not replace a kernel-observed denial):

```bash
AGENT_UID=$(sprite exec -s '<SPRITE>' -- id -u studio-agent)
sprite exec -s '<SPRITE>' -- ps -o user=,args= -C studio-gateway  # gateway uid (expect root/service)
sprite exec -s '<SPRITE>' -- runuser -u studio-agent -- cat /srv/studio-gateway/keys/keys-<build12>.json; echo "expect: EACCES (exit ≠ 0)"
sprite exec -s '<SPRITE>' -- runuser -u studio-agent -- ls /srv/studio-gateway/keys/; echo "expect: EACCES (exit ≠ 0)"
```

Expected: gateway runs privileged; every agent read of the keys dir
or files fails with permission denied (record exit codes + stderr).
Then the inbox matrix on a provisioned build workdir
(`/srv/studio-agent/builds/<...>/.studio/inbox`, created at first
assign — run §10 first if needed):

```bash
sprite exec -s '<SPRITE>' -- stat -c '%a %U %G %n' <workdir>/.studio <workdir>/.studio/inbox
sprite exec -s '<SPRITE>' -- runuser -u studio-agent -- touch <workdir>/.studio/inbox/agent-write-probe && echo WRITE_OK
sprite exec -s '<SPRITE>' -- runuser -u studio-agent -- rm <workdir>/.studio/inbox/agent-write-probe && echo DELETE_OWN_OK
```

Expected: `.studio` = `2750 studio-gw studio-io`, inbox = `2730
studio-gw studio-io`; agent writes + deletes its own drop. After a
yield + continue cycle (§10 extended or a second turn): `answer.json`
is `0640 studio-gw studio-io`, the agent can `cat` it, and agent
overwrite/remove attempts fail (record each). Any deviation stops the
run — do not weaken the boundary to proceed.

Fallback (only if no agent-uid exec channel exists): assign two
supervised turns on a scratch build — (a) "run `id -u`, `id -g`,
`id -G` and report the output verbatim", expecting the studio-agent
uid/gid/groups and NOT the gateway/service uid; (b) "try to read
`/srv/studio-gateway/keys/` and report the exact error", expecting a
permission-denied failure. Record transcripts verbatim and mark the
evidence agent-reported.

## §8 OAuth ceremony (owner-driven, P-E1 → P-E2 → P-E3 ladder)

Per the merged amendment §E. Attempt in order; record which rung worked:

- P-E1: `sprite exec -s '<SPRITE>' -- opencode mcp auth sprites` (as the
  agent user if the channel allows; else Fly console). Expect a printed
  authorization URL; open it in your browser, pick org D1 WITH a
  restricted grant (builder-name prefix, creation limit 0); paste any
  callback URL back if prompted. Verify: `opencode mcp list` shows
  `sprites`; `get_fly_context` (via a status run) shows the expected
  prefix + zero creation limit.
- P-E2 (if P-E1's callback can't complete): port-forward the callback
  port per the amendment; record ports used.
- P-E3 (last resort): complete OAuth locally, transfer ONLY the
  `mcp-auth.json` as a handled 0600 secret (document every hop).
- Forbidden: inventing tokens, CLI/API workarounds, second MCP servers.
  If all three rungs fail: STOP and report — do not invent a fourth.

Record: rung used, grant parameters, `get_fly_context` output (redact
any token material before pasting into evidence).

## §9 Status round trip

Studio must be serving (D3): start it per the standard deploy, confirm
its public/tunneled base URL, and record it. Then:

```bash
go run ./cmd/studio gateway command --tenant '<TENANT>' --build '<BUILD>' --cmd status
```

Expected: JSON with `state: running`, `agent_alive`, outbox depths.
Record verbatim + wall-clock latency (first command may absorb a wake).

## §10 No-op turn (supervised agent proof)

Send the assignment (exact prompt — do not embellish):

```bash
go run ./cmd/studio gateway command --tenant '<TENANT>' --build '<BUILD>' --cmd assign \
  --payload '{"prompt":"Reply with exactly the single line BUILDER-ALIVE. Do not use any tools.","workspace_id":"<TENANT>"}'
```

Expected, in order (record each with timestamps): `agent.started`
callback lands (check Studio conversation stream); progress/message
callbacks stream; run goes idle; `agent.completed` lands. Then:

```bash
go run ./cmd/studio gateway command --tenant '<TENANT>' --build '<BUILD>' --cmd fetch_artifact \
  --payload '{"manifest":{"entries":[]}}'
```

Expected: 400 (empty manifest refused) — proves the path is reachable
and strict. (Real artifact fetch is exercised in Phase D.)

## §11 Sleep, wake, and resume observation

1. Note the gateway service PID + agent session ref (status command).
2. Idle the sprite: no commands for 90s (idle timeout reported ~30s,
   third-party — observe, don't assume).
3. `sprite info` / status: record whether the sprite slept
   (state/status fields) and the wall time.
4. Send `status` again: record wake latency + whether the same PIDs
   survived (warm) or the gateway relaunched (cold). BOTH outcomes are
   acceptable; record which occurred.
5. If cold (gateway relaunched): expect an `agent.failed` recovery
   callback in Studio ("gateway restarted…"). Recovery never
   auto-restarts: resume deliberately with a continue as a NEW fenced
   turn (the next turn number after the failed turn — turn 2 if the
   §10 assign was turn 1):
   ```bash
   go run ./cmd/studio gateway command --tenant '<TENANT>' --build '<BUILD>' --cmd continue --turn 2 \
     --payload '{"question_id":"wake-probe","kind":"clarification","answer":"proceed"}'
   ```
   Expected: `agent.resumed` (emitted only after the fresh supervised
   run launches with the preserved workdir + session lineage) and then
   a completed run. A `409 stale turn` means the turn number is wrong
   (use `status` + the conversation stream to find the last executed
   turn, then send last+1); a `409 session is terminal` after
   done/cancelled is correct (those never resume); a `500 resume
   ambiguous` means the launch was tainted (recovery indeterminate —
   send the next turn instead of retrying the same command).
6. If warm (processes intact): expect `status` to show the same live
   session with no recovery callback.
7. Repeat once after a 10-minute idle to probe warm-vs-cold boundary;
   record both runs in a timing table.

The adapter design is safe under either outcome; this section exists to
replace reports with observations.

## §12 Failure interpretation

| Symptom | Meaning | Action |
|---|---|---|
| provision: non-assigned/already-provisioned | build state guard | use a fresh build; re-check |
| provision: sprite exists, install fails | partial state | fix cause, re-run (attach converges) |
| provision: binary differs, run gateway upgrade | staged binary drifted from deployed | do NOT re-provision; use §6.5 upgrade |
| provision: service unhealthy / inspection failed | needs diagnosis | STOP touching it; diagnose, then explicit upgrade |
| upgrade: live agent run | guard held | drain runs, or repeat with --force deliberately |
| setup.sh: "must run as root" | no privilege channel | STOP, report (§7) |
| setup.sh: missing prerequisite | image lacks packages | D5 install, re-run |
| setup.sh: checksum mismatch | asset drift/tamper | STOP, report; do not override |
| OAuth: no URL printed / flow hangs | headless ceremony gap | descend P-E ladder, record |
| status: timeout on first call | wake latency | retry once, record both timings |
| status: 401 | key mismatch (wrong secret/build) | verify env + keys dir, never print keys |
| assign: 409 | build already assigned | use a fresh build |
| continue: 400 turn required | turn fencing (M11) | resend with --turn <last+1> |
| continue: 409 stale turn | wrong turn number | find last executed turn, send last+1 |
| continue: 409 session is terminal | done/cancelled never resume | start a new build if needed |
| command: 409 indeterminate | crash before completion | recover with continue as a new turn, never retry the id |
| No callbacks in Studio | deliverer failing OR Studio unreachable | check gateway logs + D3 path; record |
| status: outbox_failed > 0 | callbacks parked terminally | inspect per build (`gateway command --cmd outbox_inspect`), fix the cause, requeue (`gateway command --cmd outbox_requeue --payload '{"item_id":"..."}'`); record ids + acks |
| status: degraded true | pump lost callbacks (sick outbox) | treat build as backlogged, not complete; diagnose disk; Phase C reconciles; record |
| agent run hangs, no events | gateway-side (stderr wedge impossible — supervisor drains) or prompt/agent issue | check gateway status detail (corrupt/truncated tallies) + service logs first; then cancel via CLI, capture transcript, report |

## §13 Cleanup (always runs)

```bash
# Retire via sprite CLI (exact names from §6 output):
sprite destroy -s '<SPRITE>'
sprite list | grep '<SPRITE>' ; echo "expect: no match (exit 1)"
```

Record: destroy output + empty-list proof. If D4 retains the builder,
record its name + reason + who owns its ongoing cost instead — and still
verify nothing ELSE remains (`sprite list` otherwise empty of studio
builders).

## §14 Evidence to report back

Timings (provision, install, first wake, no-op turn, sleep observes),
full transcripts (redacted), D1–D6 resolutions, warm/cold observations,
any §12 rows hit, cleanup proof or retention record, and every deviation
from this runbook with reasons.

## §15 Phase B live-segment backlog (must close before Phase B exit)

- M5: SSE max stream lifetime (+ concurrent-stream budget decision).
- Serve-time `STUDIO_GATEWAY_BINARY` wiring (CLI flag exists; serve
  currently refuses without it).
- Studio sender loop stays Phase C (recorded, not forgotten).
- Any pin drift found in §2 needs explicit re-pin approval (§14 rules).
