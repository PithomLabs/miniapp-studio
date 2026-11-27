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

0. Confirm the authorization class of every step in §0b before running
   it. Read-only steps run under the Part 2 GO; each C-class step needs
   its own explicit owner GO at execution time.
1. Collect the §1 decisions first. Nothing proceeds without all of them.
2. Work top to bottom; each section ends with evidence to record.
3. On any unexpected output, go to §12 (failure interpretation) before
   retrying or improvising.
4. Finish with §13 (cleanup or explicit retention) no matter what.

## §0b Authorization map (read-only vs consequential)

| Class | Steps | Authorization |
|---|---|---|
| R (read-only) | §2 pins/tree, §3 gates, §4 build, §5 env assembly, §6 verify-only reads (`info`, `exec` reads, `status`), §7 discovery reads, §7a/§7b reads, §9 route probes, §11 observations | Covered by the Part 2 GO; stop on any surprise per §12 |
| C0 | Fly-hosted Studio deployment (D3, owner-established outside this runbook) | Prerequisite: must exist + pass §5b before §6 |
| C1 | §6 provision (creates Sprite, installs service, writes keys) | Separate owner GO at execution |
| C2 | §7/D5 package installation on the builder | Separate owner GO at execution |
| C3 | §8 OAuth ceremony (Fly grants) | Separate owner GO at execution |
| C4 | §8b provider key entry (OpenRouter credential) | Separate owner GO at execution |
| C5 | §10 assign + §11 resume turns (live agent runs, inference spend) | Separate owner GO at execution |
| C6 | §13 destroy (irreversible) or D4 retention | Separate owner GO at execution |

§6.5 upgrade stays PROHIBITED live until N2 closes (§6.5) — no GO
may authorize it. A GO for one class never covers another.

## §1 Owner decisions (recorded)

| # | Decision | Resolution | Status |
|---|---|---|---|
| D1 | Fly organization for builders | `isagani-mendoza` (exact slug; no new org) | DECIDED |
| D2 | Model provider + auth for OpenCode | Primary `opencode` (default `deepseek-v4.1-flash-free`, alt `mimo-v2.6-flash-free`); OpenRouter same-model fallback; owner key | DECIDED (§8b) |
| D3 | Studio callback reachability | (b) Fly-hosted Studio through Fly Proxy | STRATEGY DECIDED (deployment outstanding; verified by §5b pre-provision + §9 pre-agent — both must pass, no tunnel substitution without a new decision) |
| D4 | Builder retention after proof | destroy to zero | DEFAULT ACCEPTED |
| D5 | System packages (node≥20, bun, git, curl) | preinstall in image, or apt-install live | AT EXECUTION (§7 discovery) |
| D6 | Pin drift | hold existing pins; any drift needs STOP + separate re-pin decision | DECIDED |

D2 auth note: credentials must be available to the supervised
OpenCode process inside the Builder — dashboard or dev-machine
possession does not establish that. Delivery mechanism: the §8b
owner ceremony into OpenCode's native credential store (no
Gateway injection API; owner ceremony + §10 proof required
before any agent run counts as evidence). Model selection rides
the assign-payload `model` field; no automatic failover exists.

D3 note: the gateway POSTs callbacks to Studio's `/gw/v1/callback`, so
Studio must be reachable FROM the sprite during live proofs. There
is no third path that doesn't weaken the proof — do not substitute
a request-catcher. §5b gates deployment readiness (record, listener,
`/readyz`) before any provisioning; §9 proves the callback route
(400 + 401 probes) before any agent run.

## §2 Pins and tree state (re-verify, record, do not float)

```bash
git branch --show-current         # record: must be the owner-approved branch
git rev-parse HEAD                # record: exact HEAD SHA under test (no hardcoded expectation)
git status --porcelain            # record verbatim
go version                         # record
curl -fsSL https://registry.npmjs.org/@flydotio/sprites-opencode-plugin \
  && echo "NPM-PUBLISHED (stop: needs re-pin decision)" || echo "npm-404-as-expected"
release_json="$(curl -fsSL 'https://api.github.com/repos/anomalyco/opencode/releases?per_page=10')" || {
  echo "STOP: failed to retrieve OpenCode releases"
  exit 1
}

v1_tags="$(printf '%s\n' "$release_json" \
  | grep -oE '"tag_name"[[:space:]]*:[[:space:]]*"v1[^"]*"' \
  | sed -E 's/.*"(v1[^"]*)".*/\1/' \
  | sort -u)"

if [ -z "$v1_tags" ]; then
  echo "STOP: no v1 release tags found"
  exit 1
fi

printf '%s\n' "$v1_tags"

if [ "$v1_tags" != "v1.19.6" ]; then
  echo "STOP: unexpected v1 release set; explicit pin review required"
  exit 1
fi
# expect: prints exactly v1.19.6, exit 0
```

Tree-state rule (B1): the branch, HEAD SHA, and every working-tree
change must be owner-recognized — reconcile them against the
owner-approved checkout before proceeding. Do not assume zero
commits or an all-untracked tree, and no SHA is hardcoded here so
this gate cannot go stale. If anything is unexpected or cannot be
reconciled: STOP and report. Purpose: establish exactly which
source tree is being used before live operations.

Expected pins (Part 1 values; any difference stops the run for a re-pin
decision): OpenCode v1.19.6, asset `opencode-linux-x64.tar.gz` sha256
`47734e08…09a057` (full: `47734e08968b30cc22a0bffb6d9feeed76ad4aa743213a12122040392d09a057`),
plugin commit `f63a85a7da9fa2f42b21877ab58b35d18ccaa682`, sprites-go
v0.2.2, Miniapp `73351a8`.

Record: full command outputs + `git status` in the evidence.

## §3 Pre-live gates (prove the tree before touching live infra)

Class R. Full-suite live verification stays DEFERRED to the
pre-demo gate (Card 001 decision) — this section is the smallest
sufficient B2 gate, not the full suite:

```bash
gofmt -l .                # expect: no output
go vet ./...
go build ./...
env -u TEST_DATABASE_URL go test -p 1 ./...   # repo-local: live tests skip honestly, expect: all ok
command -v sprite         # expect: a path (CLI presence; login state is owner-held)
```

The `env -u` is load-bearing: an exported `TEST_DATABASE_URL`
would turn this gate into a live full-suite run. Record the suite
tail + skip count. Any failure stops the run.

Live DB/migration/privilege surface: cite Card 001 (seven-test
Neon gate GREEN at commit `0c027ae`, `test-run-card001.log`,
468.5s) instead of re-running — IF the §2 HEAD equals `0c027ae`
with a reconciled tree. Otherwise re-run exactly:

```bash
set -o pipefail
time task dev:test -- -- -run 'TestRoleAttributes|TestPrivilegeParity|TestStudioLookupSetRepair|TestMigrationDownUpBounded|TestPoolResetsTenantSetting|TestPoolDiscardsUnresettable|TestTenantResetHookTimesOut' -timeout 15m -v -count=1 2>&1 | tee test-run-card002-gate.log
```

Expect: all seven PASS. Any red stops the run (triage at root
cause; no live proofs on a red gate).

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
export STUDIO_FLY_ORG='isagani-mendoza'  # D1 decided
export STUDIO_SPRITES_TOKEN_FILE=/path/to/org-token  # 0600, owner-held
export STUDIO_GATEWAY_ORG_SECRET_FILE=/path/to/org-secret  # 0600, owner-held; see below
```

Org secret: 32+ random bytes, generated once (`head -c 32 /dev/urandom
| base64 > file; chmod 600 file`), stored 0600. It derives ALL build
HMAC keys — compromise rotates every builder. Never print it; the CLI
never echoes it (verify: no secret in any output below).

Model auth (D2 decided: OpenCode primary, OpenRouter fallback).
Model selection has an implemented path: assign-payload `model`
(`provider/model` identifier) flows to the agent's `--model`
flag. No Gateway credential-injection API exists by design:
the service API carries no environment (our `ServiceRequest`
and sprites-go v0.2.2 both expose only Cmd/Args/HTTPPort), the
adapter launches OpenCode with no extra env (it inherits the
gateway process environment), and the provisioned bundle
carries no model/auth configuration. Credential delivery is
the §8b owner ceremony into OpenCode's native store:
OpenRouter key REQUIRED (fallback); OpenCode credential NOT
required while step-4 catalog confirms zero input cost for
both primary models. Owner action required before any agent
run: complete §8b, then prove it — the §10 no-op turn
succeeding is the end-to-end proof (a model call works); an
auth failure there means STOP, not evidence. Hard rules: keys
are never committed, never included in any application
artifact, never passed in process arguments, never printed in
logs, evidence, or agent transcripts. Keep the two mechanisms
separate: provider keys are for model inference only; Sprites
MCP OAuth is for the Builder's authorized Sprite capabilities
only. Fallback cost controls (owner-confirmed spending limit,
no auto-recharge, bounded attempts, per-build usage evidence)
must be in place before any fallback use.

D3 note: Fly-hosted Studio deployment itself is owner-established
(no in-tree Studio-on-Fly deploy path exists); §5b verifies the
deployed result before any provisioning, and §9 proves the
callback route before any agent run. Neither section deploys.

## §5b Studio readiness gate (C0 prerequisite — must pass BEFORE §6)

Class R (reads against owner-established infra). Provisioning a
builder whose callbacks go nowhere proves nothing — do not run §6
until every line below passes. No tunnel substitution: D3 decided
Fly-hosted, and only a new owner decision changes that.

1. Record the deployment: Fly app name, image revision/commit,
   public base URL (`<base-url>` below), internal port.
2. Studio serves with the Fly Proxy listener (the loopback
   default receives nothing behind the proxy — this fails
   silently at the network layer):

   | D3 deployment | Required `STUDIO_LISTEN` |
   |---|---|---|
   | (b) Fly Proxy (decided) | `0.0.0.0:<internal_port>` |
   | (a) Owner machine + tunnel (fallback, needs a new decision) | `127.0.0.1:8080` (default) |
   | Fly private 6PN (not selected) | `fly-local-6pn:<port>` |

   ```bash
   go run ./cmd/studio serve   # takes no flags; reads STUDIO_LISTEN
   ```
3. Prove readiness (`/readyz`, not just `/healthz`: the former
   checks database connectivity and exact schema version and
   answers 503 `{"status":"unavailable"}` while migrating,
   unreachable, dirty, or behind; the latter only proves the
   handler responds):

   ```bash
   curl <base-url>/healthz   # expect: HTTP 200 {"status":"ok",...}
   curl <base-url>/readyz    # expect: HTTP 200 {"status":"ready"}
   ```

Record: app name, revision, base URL, listener, both outputs.
Anything but double-200 stops the run (C0 unmet — fix the
deployment, do not provision against an unready Studio).

## §6 Provision the builder (C1 provision + R verification)

Pick a build first: in the Studio UI, open any conversation and send one
message (creates conversation + build `BUILD`; record both ids).

```bash
go run ./cmd/studio gateway provision --tenant '<TENANT>' --build '<BUILD>' \
  --gateway-binary /tmp/studio-gateway --studio-url '<STUDIO_URL>'
```

`<STUDIO_URL>` is the §5b-verified Fly-hosted base URL — the
provisioned service boots with it as its callback target; without
it the service cannot start, so there is no default. (Class C1:
this command needs its own owner GO.)

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

> **N8 RESTRICTION — re-provisioning permission window (F-R1).**
> Staging is recreated with permissive modes on every provision
> and locked down only when setup runs, so re-provisioning while
> an untrusted sibling agent process is live exposes a transient
> window. This race is NOT fixed in provisioning code. Before
> every re-provision on a builder with running agents:
>
> ```bash
> sprite exec -s '<SPRITE>' -- stat -c '%a %n' /srv/studio-staging  # expect: 700 before you start
> sprite exec -s '<SPRITE>' -- ps -o user=,args= -C opencode        # expect: no live untrusted sibling agents
> ```
>
> - Verify staging is mode `0700` before re-provisioning.
> - Verify no live untrusted sibling agent processes exist that
>   could exploit staging recreation.
> - If either condition cannot be established, drain the relevant
>   agent activity first (cancel runs, let turns complete), then
>   re-verify, then re-provision. Record what was drained and why.

## §6.5 Gateway upgrade (explicit replacement only)

> **N2 PROHIBITION — DO NOT INVOKE LIVE.** The upgrade idle check has
> a time-of-check/time-of-use window (an assign racing the upgrade
> can start a run after the scan but before replacement) and skips
> builds with empty URLs instead of refusing them. Until N2 is fixed
> and tested, **no live `gateway upgrade` operation is authorized**.
> If binary drift is detected during a future live run, STOP and
> report rather than bypassing the guard. The procedure below is
> preserved for use after N2 closure.

Provisioning NEVER replaces a running gateway binary. When the binary
must change (drift refusal names this op), replace it explicitly
(after N2 closure — see prohibition above):

```bash
go run ./cmd/studio gateway upgrade --tenant '<TENANT>' --build '<BUILD>' \
  --gateway-binary /tmp/studio-gateway --studio-url '<STUDIO_URL>'
```

Expected: JSON with `"upgraded": true`, `"gateway_sha256"`, and
`"idle_checked"` (every build on the builder verified without a live
agent run). A live run refuses without `--force`; `--force` replaces
anyway and reports `"force": true` — use only after N2 closure and
deliberate draining of agent runs (never to bypass the prohibition
above). After upgrade, re-run §6 verification (service
running, keys present) and §9 (status answers).

## §7 Privilege and package discovery (R discovery + C2 install)

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

### §7a Gateway SERVICE privilege (N1 — distinct from exec privilege)

The §7 `id -u` above is the *exec channel*. The gateway *service*
needs its own privilege to drop agent processes to `studio-agent`
(uid + gid + supplementary groups). After the service is running
(§6/§9), derive its actual identity from the live process — never
assume it:

```bash
sprite exec -s '<SPRITE>' -- ps -o pid=,user=,args= -C studio-gateway
GW_PID=<pid from above>
sprite exec -s '<SPRITE>' -- awk '/^Uid:/ {print "euid="$3}' /proc/<GW_PID>/status
```

Record the service EUID verbatim. Contract:

- Reference: service EUID 0 (privileged) — the credential drop is
  expected to work, and §7b proves it. The gateway also self-tests
  at boot (spawns `true` as the agent user; a service that cannot
  drop refuses to start — check service logs for `boot refused`
  on any early failure).
- Non-root service EUID: acceptable ONLY if the §7b drop proofs
  pass (agent runs as `studio-agent`, keys denied). The boot
  self-test is the first gate — if it refuses, the configuration
  lacks privilege: STOP and report, do not weaken to single-user.
- A service running as `studio-gw` without drop privilege must
  never be assumed capable of launching `studio-agent`.

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
GW_EUID=$(sprite exec -s '<SPRITE>' -- awk '/^Uid:/ {print $3}' /proc/<GW_PID>/status)  # from §7a
IO_GID=$(sprite exec -s '<SPRITE>' -- getent group studio-io | cut -d: -f3)
echo "service euid=$GW_EUID agent uid=$AGENT_UID io gid=$IO_GID"
sprite exec -s '<SPRITE>' -- runuser -u studio-agent -- cat /srv/studio-gateway/keys/keys-<build12>.json; echo "expect: EACCES (exit ≠ 0)"
sprite exec -s '<SPRITE>' -- runuser -u studio-agent -- ls /srv/studio-gateway/keys/; echo "expect: EACCES (exit ≠ 0)"
```

Expected: every agent read of the keys dir or files fails with
permission denied (record exit codes + stderr). The keys stay
setup-owned `0600 studio-gw` regardless of service uid — only the
gateway (owner or privileged) reads them.
Then the inbox matrix on a provisioned build workdir
(`/srv/studio-agent/builds/<...>/.studio/inbox`, created at first
assign — run §10 first if needed). Gateway-created paths carry the
SERVICE EUID (N1 — never assume the literal owner name `studio-gw`):

```bash
sprite exec -s '<SPRITE>' -- stat -c '%a %u %g %n' <workdir>/.studio <workdir>/.studio/inbox
sprite exec -s '<SPRITE>' -- runuser -u studio-agent -- touch <workdir>/.studio/inbox/agent-write-probe && echo WRITE_OK
sprite exec -s '<SPRITE>' -- runuser -u studio-agent -- rm <workdir>/.studio/inbox/agent-write-probe && echo DELETE_OWN_OK
```

Expected: `.studio` = `2750 $GW_EUID $IO_GID`, inbox = `2730
$GW_EUID $IO_GID` (numeric owner must equal the §7a service EUID,
group must equal the `studio-io` GID); agent writes + deletes its
own drop. After a yield + continue cycle (§10 extended or a second
turn): `answer.json` is `0640 $GW_EUID $IO_GID`, the agent can
`cat` it, and agent overwrite/remove attempts fail (record each).
Any deviation stops the run — do not weaken the boundary to
proceed. (N1 correction: earlier revisions hardcoded `studio-gw`
as the file owner, which contradicts privileged-service
operation — the file creator's EUID is the correct expectation.)

Fallback (only if no agent-uid exec channel exists): assign two
supervised turns on a scratch build — (a) "run `id -u`, `id -g`,
`id -G` and report the output verbatim", expecting the studio-agent
uid/gid/groups and NOT the gateway/service uid; (b) "try to read
`/srv/studio-gateway/keys/` and report the exact error", expecting a
permission-denied failure. Record transcripts verbatim and mark the
evidence agent-reported.

## §8 OAuth ceremony (C3 — owner-driven, P-E1 → P-E2 → P-E3 ladder)

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

## §8b Provider authentication ceremony (C4 — OpenRouter + OpenCode)

Owner-driven, after §7b identity proofs, before §10 (no agent run
may precede it — §10 is the first model call). No Gateway code
changes: this uses OpenCode v1.19.6's native credential store.

Model mapping (owner-decided; STOP on any catalog mismatch):

| Role | Provider | Model ID |
|---|---|---|
| Default | `opencode` | `opencode/deepseek-v4.1-flash-free` |
| Approved alternative | `opencode` | `opencode/mimo-v2.6-flash-free` |
| Fallback for DeepSeek | `openrouter` | `openrouter/deepseek/deepseek-v4.1-flash` |
| Fallback for MiMo | `openrouter` | `openrouter/xiaomi/mimo-v2.6-flash` |

Fallback preserves the same model family, never another model or
provider. OpenRouter routes are paid (owner-recorded 2026-11-25:
DeepSeek $0.051/$0.567, MiMo $0.14/$0.28 per M in/out tokens —
recheck at execution). No automatic failover exists in the
Gateway: selection via `--model` is manual per run. Configured
credentials ≠ proven failover.

1. Open the interactive console (owner-verified Sprite command):
   ```bash
   sprite console -s '<SPRITE>'
   ```
2. Become the agent identity — never root or `studio-gw` — and
   confirm it matches the §7b agent uid:
   ```bash
   su - studio-agent
   id -u  # expect: <AGENT_UID from §7b>
   ```
3. HOME/XDG/data-dir gate (critical-path check — a mismatch
   means STOP, do not enter any key). The Gateway child
   inherits the service environment with no override
   (`Supervisor.Start`: `cmd.Env = append(os.Environ(), ...)`
   with nil extras), so the supervised agent reads
   credentials from the *service* env's data dir, not
   necessarily this login shell's. Compare both:
   ```bash
   echo "shell: HOME=$HOME XDG_DATA_HOME=${XDG_DATA_HOME:-<unset>}"
   tr '\0' '\n' < /proc/<GW_PID>/environ | grep -E '^(HOME|XDG_DATA_HOME)='  # service env (<GW_PID> from §7a)
   getent passwd studio-agent  # fallback identity if service env lacks HOME
   ```
   Compute each side's OpenCode data dir (`$XDG_DATA_HOME/
   opencode`, else `$HOME/.local/share/opencode`; unset HOME
   resolves via passwd lookup by euid). Proceed ONLY if both
   sides agree byte-for-byte. On any divergence: STOP and
   report the two environments as the root cause — do not
   enter the key into the wrong home and do not work around it.
4. Verify the catalog from the pinned CLI (still as
   `studio-agent`), and confirm zero input cost for both
   OpenCode models plus the presence of both fallback IDs:
   ```bash
   opencode models opencode --verbose --refresh
   opencode models openrouter --verbose --refresh
   ```
   Expect: `deepseek-v4.1-flash-free` and `mimo-v2.6-flash-free`
   listed with zero input cost; `deepseek/deepseek-v4.1-flash`
   and `xiaomi/mimo-v2.6-flash` present under openrouter.
   Record full output (redact nothing — catalogs carry no
   secrets). Any missing ID or nonzero primary cost: STOP.
5. Authenticate OpenRouter (REQUIRED for fallback) via the
   native hidden prompt. No separate OpenCode credential is
   needed provided step 4 confirmed zero input cost (pinned
   `provider.ts` retains zero-cost models under the `public`
   placeholder and filters paid ones without a credential):
   ```bash
   opencode auth login --provider openrouter
   ```
   Enter the owner's dedicated, spending-limited key at the
   hidden prompt. Never pass it as an argument, never print
   it, never run `opencode auth export`, never copy it into
   the Builder workspace.
6. Verify without displaying secrets:
   ```bash
   opencode auth list  # expect: openrouter configured (no values shown)
   stat -c '%U %a %n' <data-dir-from-step-3>/auth.json  # expect: studio-agent 600
   stat -c '%U %a %n' <data-dir-from-step-3>            # record parent perms; must not be world-writable
   ```
   Never `cat` the credential file. Any other owner/mode:
   STOP and report.
7. Mechanism separation (restated, enforced by distinct
   stores): the OpenRouter key authenticates model inference
   only; Sprites MCP OAuth (§8) authorizes Builder Sprite
   capabilities only. Neither substitutes for the other.
8. End-to-end proof is §10: the supervised no-op assignment
   must explicitly select `opencode/deepseek-v4.1-flash-free`
   and succeed. Record provider ID, model ID, OpenCode
   version, and attempt outcome. Auth failure there means
   STOP, not evidence.

Same-UID exposure warning: `0600` protects `auth.json` from
other OS users, NOT from the agent's own tools — the supervised
agent runs as `studio-agent` and can read its own credential
file. This is inherent to the approved design, not a
misconfiguration. Mitigations are procedural: dedicated key,
strict spending limit, no auto-recharge, revoke after the
proof unless continued use is explicitly authorized, never
use a personal/broad key. Record the key's spending limit
and revocation plan in the evidence; do not record the key.

Studio readiness was proven pre-provision in §5b; the callback
route proof runs in §9 after this ceremony. Do not run any agent
turn before §9 passes.

## §9 Callback route proof (must pass BEFORE §10)

Class R (rejected probes change no state — the 400/401 paths
ingest nothing). Uses the §5b base URL, the §6 sprite + build,
and on-sprite `curl` (a D5 dependency, present after §7).

Probe 1 — route + handler (bare POST carries no envelope, so the
handler answers 400, not 401):

```bash
sprite exec -s '<SPRITE>' -- curl -s -o /dev/null -w "%{http_code}" -X POST <base-url>/gw/v1/callback  # expect: 400
```

Probe 2 — auth enforced (well-formed envelope, valid build id,
bogus everything else — parses, resolves keys, fails Verify):

```bash
sprite exec -s '<SPRITE>' -- curl -s -o /dev/null -w "%{http_code}" -X POST <base-url>/gw/v1/callback -H 'Content-Type: application/json' -d "{\"build_id\":\"<BUILD>\"}"  # expect: 401
```

Anything else stops the run (§12): 404 = wrong base URL or a
Studio too old for the route; 503 = Studio keys unconfigured;
timeout/refused = the D3 path is broken. The full signed
round-trip proof completes in §10 when the `agent.started`
callback lands in the Studio conversation stream. Then:

```bash
go run ./cmd/studio gateway command --tenant '<TENANT>' --build '<BUILD>' --cmd status
```

Expected: JSON with `state: running`, `agent_alive`, outbox depths.
Record verbatim + wall-clock latency (first command may absorb a wake).

## §10 No-op turn (C5 — supervised agent proof)

Send the assignment (exact prompt — do not embellish):

```bash
go run ./cmd/studio gateway command --tenant '<TENANT>' --build '<BUILD>' --cmd assign \
  --payload '{"prompt":"Reply with exactly the single line BUILDER-ALIVE. Do not use any tools.","workspace_id":"<TENANT>","model":"opencode/deepseek-v4.1-flash-free"}'
```

The `model` field explicitly selects the D2 default; do not omit
it or substitute another model. Expected, in order (record each
with timestamps): `agent.started` callback lands (check Studio
conversation stream); progress/message callbacks stream; run goes
idle; `agent.completed` lands. Record provider ID, model ID,
OpenCode version, and attempt outcome — this success is the
authoritative proof of inference. A separate controlled test
proves fallback behavior only if automatic failover is
implemented and accepted; it is neither today, so no fallback
claim may be made from this section. Then:

```bash
go run ./cmd/studio gateway command --tenant '<TENANT>' --build '<BUILD>' --cmd fetch_artifact \
  --payload '{"manifest":{"entries":[]}}'
```

Expected: 400 (empty manifest refused) — proves the path is reachable
and strict. (Real artifact fetch is exercised in Phase D.)

## §11 Sleep, wake, and resume observation (C5 resume + R observations)

1. Note the gateway service PID + agent session ref (status command),
   and record the live agent PIDs (F-R2/N5 — needed for the step 5
   absence check):
   ```bash
   sprite exec -s '<SPRITE>' -- ps -o pid=,user=,args= -C opencode  # record every PID verbatim
   ```
2. Idle the sprite: no commands for 90s (idle timeout reported ~30s,
   third-party — observe, don't assume).
3. `sprite info` / status: record whether the sprite slept
   (state/status fields) and the wall time.
4. Send `status` again: record wake latency + whether the same PIDs
   survived (warm) or the gateway relaunched (cold). BOTH outcomes are
   acceptable; record which occurred.
5. If cold (gateway relaunched): expect an `agent.failed` recovery
   callback in Studio ("gateway restarted…"). Then verify orphan
   cleanup explicitly (F-R2/N5) — do NOT claim cleanup succeeded
   solely because the callback arrived:
   ```bash
   sprite exec -s '<SPRITE>' -- ps -o pid=,user=,args= -C opencode  # expect: none of the step-1 PIDs
   for p in <PID_FROM_STEP_1>...; do sprite exec -s '<SPRITE>' -- ps -p $p -o pid=,args=; echo "pid $p expect: no such process"; done
   ```
   Every pre-restart agent PID must be absent. Report any remaining
   process, any PID that now belongs to a different command
   (PID reuse — record the new argv), or any ambiguous identity.
   Only then resume. Recovery never
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
| provision: binary differs, run gateway upgrade | staged binary drifted from deployed | do NOT re-provision; STOP and report — §6.5 upgrade is prohibited live until N2 closes |
| provision: service unhealthy / inspection failed | needs diagnosis | STOP touching it; diagnose and report (upgrade path needs N2 closure first) |
| upgrade: live agent run | guard held — live upgrade prohibited (N2) | STOP and report. Live `gateway upgrade` remains prohibited until N2 is fixed and tested, as stated in §6.5. Do not use `--force` to bypass the prohibition. |
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
| §5b: `/readyz` ≠ 200 | Studio unready (migrating, unreachable, dirty, or behind) | fix the deployment (C0 unmet); do not provision |
| §9 probe 1 ≠ 400 | route/handler not reached | 404 = wrong base URL or Studio too old; timeout/refused = D3 path broken; fix, re-probe |
| §9 probe 2 ≠ 401 | auth path unexpected | 503 = Studio keys unconfigured (check org secret); 404 = unknown build (wrong BUILD id); fix, re-probe |
| status: outbox_failed > 0 | callbacks parked terminally | inspect per build (`gateway command --cmd outbox_inspect`), fix the cause, requeue (`gateway command --cmd outbox_requeue --payload '{"item_id":"..."}'`); record ids + acks |
| status: degraded true | pump lost callbacks (sick outbox) | treat build as backlogged, not complete; diagnose disk; Phase C reconciles; record |
| agent run hangs, no events | gateway-side (stderr wedge impossible — supervisor drains) or prompt/agent issue | check gateway status detail (corrupt/truncated tallies) + service logs first; then cancel via CLI, capture transcript, report |

## §13 Cleanup (C6 — always runs)

```bash
# Retire via sprite CLI (exact names from §6 output).
# Class C6: needs its own owner GO. The CLI prompts for
# confirmation — confirm deliberately; do NOT add --force.
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
from this runbook with reasons. Always include: the §2 pin/tree record,
the §3 gate outputs (or the Card 001 citation with HEAD), the §5b
deployment record (app, revision, base URL, listener, both health
outputs), the §9 probe pair, and the per-class GO record (which C-class
steps ran under which explicit authorization).

## §15 Phase B live-segment backlog (must close before Phase B exit)

- M5-SSE: SSE max stream lifetime (+ concurrent-stream budget decision).
- Serve-time `STUDIO_GATEWAY_BINARY` wiring (CLI flag exists; serve
  currently refuses without it).
- Studio sender loop stays Phase C (recorded, not forgotten).
- Any pin drift found in §2 needs explicit re-pin approval (§14 rules).
