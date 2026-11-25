# Residual-closure report — prompt12.1 (2026-11-25)

Bounded residual-closure pass per `plans/prompt12.1.md`, against the
verdict **ACCEPTED WITH BOUNDED RESIDUALS** in
`plans/adv_review_prompt12.md`. This pass closes N1, N4, R-DB, and
R-NET, and tracks N2–N10 with operational restrictions. Architecture
not reopened; no broad re-review performed.

## 1. Executive summary and acceptance state

- N1 (service privilege + identity spec + boot self-test): CLOSED —
  contract explicit in three documents, fail-closed self-test
  implemented and tested, matrix/runbook uid-agnostic.
- N4 (M5 collision + v1 banner): CLOSED — M5-STDERR/M5-SSE mapping
  applied across current docs, v1 bannered superseded, completion
  language disambiguated.
- R-DB (PG19 verification): CLOSED — full suite, six-package race,
  and all named F7/M14/M15/migration/RLS targets executed green
  against PostgreSQL 19.0 Docker, zero skips in DB-named runs.
- R-NET (run.ts provenance): CLOSED — tag, commit, tree entry, and
  both hashes independently re-verified; all match the vendored
  record.
- N2–N10 (except N4): TRACKED with a residual register here and in
  the evidence doc; N2 carries a live-upgrade prohibition now in
  the runbook.
- One pre-existing test flake observed twice (`TestAdapterStatus`
  TempDir-cleanup race: once in a verbose full run, once in six
  package runs), root-caused to prompt11-era adapter-test
  lifecycle code untouched by this pass; disclosed in §10, suite
  green on re-run. No code changed for it (scope).

## 2. Exact scope and tree state

Code changed (N1 + N4-hygiene only):
`cmd/gateway/main.go` (self-test), `cmd/gateway/main_test.go`
(7 new tests), `gateway/protocol.go` (two stale comments).
Docs changed: `docs/prompt11-correction-report.md` (§7 contract +
matrix, §2 N4 language), `docs/gateway-phase-b-evidence.md`
(N1 contract, N4 mapping + v1 banner, N2–N10 register),
`docs/gateway-phase-b-live-runbook.md` (§7a, §7b rework, §6.5 N2
banner, M5-SSE), `plans/prompt11_plan.md` (M5-STDERR + note).
Created: this report. Untouched: `plans/adv_review_prompt12.md`,
all historical prompts/reviews, Miniapp, frozen plans, F1/G-01/P5.

Tree-state correction (read carefully — the assumed boundary
evolved outside this session): the tree is NO LONGER
all-untracked/zero-commit. It now holds two owner commits
(`2d020a3` "Initial commit" 08:39 and `22ce9c4` "prompt12.md"
12:05, both authored by the owner, not by this agent). This
session ran zero git write commands (no commit/push/stage;
`git diff --cached` verified empty): my work is exactly the 7
modified + 1 created files above, all uncommitted. Note:
`plans/prompt12.1.md` was committed EMPTY and filled (522 lines)
outside this session — I executed against the working-tree
content as found. Untracked `plans/prompt13*.md` files also
predate my session and were left untouched.

## 3. N1 root cause, correction, code/test evidence

Root cause (review §10): the Credential drop needs a privileged
service, but no document stated, pinned, or verified that
requirement, while runbook §7b and the report matrix asserted
`studio-gw`-owned gateway-created files — impossible under the
only working configuration (privileged service creates
service-uid-owned files; `SecureDir`/`writeFileComplete` chgrp/
chmod but never chown owner — verified by code walk).

Correction: (a) privilege contract + four-identity distinction in
report §7, evidence doc, runbook §7a; (b) boot self-test
(`checkAgentDistinct` + `checkAgentSpawn`) inline in `serve()`
before recovery/listening, using the production supervisor and
exact credential (probe `true`; probe binary parameterized for
tests only); (c) uid-agnostic matrix + runbook (derive service
EUID from `/proc/<pid>/status`, compare numerically with `stat`).

Tests (`cmd/gateway`, 7 new, all passing unprivileged except the
root-only real-drop case which correctly skips): distinctness
refusal/pass, spawn success, EPERM refusal with privilege
diagnostic, root-only genuine drop, child identity observation,
resolveAgent never sets AllowSameUID, end-to-end `run()` refusal
for same-uid and unprivileged boots — proving no readiness path
bypasses the checks.

## 4. Updated service privilege contract

Reference deployment runs the Gateway service privileged (root).
Non-root service identities are acceptable only when the boot
self-test passes and live §7b proves the complete uid+gid+groups
transition. A `studio-gw` service without drop privilege must
never be assumed capable. Production never falls back to
same-UID launch. Any insufficient privilege fails closed (boot
refusal / command refusal) with a non-secret diagnostic. The
Sprite service API is unchanged (no user field invented).

## 5. Corrected filesystem permission matrix

Ownership is creator-based: setup-created paths carry setup-pinned
owners (`0600 studio-gw` keys, root binary); gateway-created paths
(state `0700`, workdirs `2770`, `.studio` `2750`, inbox `2730`,
answers `0640`) carry the service EUID with group `studio-io`;
agent drops are `0660 studio-agent`. Runbook §7b verifies numeric
owner == §7a service EUID, group == `studio-io` GID, exact modes,
plus live access probes (agent write/delete-own succeed;
key reads and answer overwrite/remove fail). Full matrix with
per-row proof lives in correction report §7 as amended.

## 6. N4 identifier mapping and matrix v1 supersession record

- M5-STDERR: bounded stderr ring (prompt10-review minor),
  implemented + tested in prompt11 correction.
- M5-SSE: SSE stream lifetime + concurrent-stream budget
  (prompt10 §7 carried item), still deferred (runbook §15).
- Collision cause: the prompt10 review numbered its own minors
  M1–M16 independently of prompt10's carried items. Mapping note
  added to the evidence doc; all current-doc references renamed
  (report, evidence incl. v1 annotations, runbook, prompt11_plan);
  historical prompt/review documents preserved untouched.
- Matrix v1 carries an unmistakable SUPERSEDED banner (historical
  pre-correction record; matrix v2 + correction report govern;
  present-tense v1 claims must not be read as current); the one
  present-tense straggler ("Live P-G3 now proves…") was
  explicitly scoped to past tense in place.
- "M1–M16 all fixed" now reads "prompt11 bundle all fixed"
  with an explicit M5-SSE exclusion wherever completion is
  claimed. M16 stale Phase-A comments refreshed.

## 7. R-DB exact environment and gate results

- Endpoint: 127.0.0.1:5434, container `miniapp-p0-pg`, database
  `miniapp_p0` (project test database; no user data).
- `SHOW server_version` → `19.0 (Debian 19.0-1.pgdg13+1)`;
  `SELECT version()` → `PostgreSQL 19.0 ... x86_64-pc-linux-gnu`.
- Access: established escalated-shell Docker path (container env
  inspection + `docker exec psql`); credentials derived in-shell,
  never printed. No permission/group/config changes.
- Isolation: suites create disposable per-test databases
  (`studio_migcheck_<rand>`, dropped after) or truncate
  test-schema tables within the test database; nothing outside
  the test database is touched.
- `gofmt -l .` empty (0); `go vet ./...` 0; `go build ./...` 0.
- `go test -p 1 -count=1 ./...` → 13/13 ok (exit 0).
- `go test -race -p 1 -count=1` six packages → all ok (exit 0).
- Targeted, all executed green with zero skips: 14 F7/provision/
  upgrade/bundle tests (incl. cross-session lock proof,
  digest echoes), `TestCallbackKDFWiring` +
  `TestCallbackKDFRejectsWrongSecret` + `TestDeriveKey` (M15),
  15 migration/RLS/tenant tests (`TestMigration*`,
  `TestRLSEnable000016`, `TestPhaseARLSEnforced`,
  `TestTenant*`, cross-tenant reference/cascade, pool reset).
- Full-suite verbose audit: 354 pass, 1 skip
  (`TestCheckAgentSpawnRealDrop` — requires root by design); the
  single verbose-run failure was the §10 `TestAdapterStatus`
  flake (suite green on re-run).

## 8. R-NET upstream provenance verification

Independent re-fetch (no trust in the vendored bytes):
- Tag `v1.19.6` → lightweight tag → commit
  `f4e363c8295fa90a03dddc451882fffe8fbce357`.
- Tree entry `packages/opencode/src/cli/cmd/run.ts`: blob
  `944aec17abd75cc3272688e89a140b56cd2bb2d3`, 31940 bytes.
- Downloaded 31940 bytes; SHA-256
  `e040e90d4c81cdd59b9568ee90a53c0209cd9ddf1b8b207d1a8685b2e26d57d0`
  ✓ matches record; Git blob SHA-1 over `blob <len>\0`+bytes =
  `944aec17abd75cc3272688e89a140b56cd2bb2d3` ✓ matches record
  and tree entry. No upstream code copied into the repo.

## 9. N2–N10 residual register and operational restrictions

| ID | Root issue / component | Impact | Scope | Durable correction + test | Owner action |
|---|---|---|---|---|---|
| N2 | Upgrade idle-check TOCTOU + empty-URL skip (`tools/build_upgrade.go` verify) | Racing assign's run killed at replace (honest failed callback); unscannable builds silently pass | **Gates live upgrade only** | Re-verify idle immediately before delete/create; refuse unscannable builds; racing-assign + empty-URL tests | **PROHIBITED live until fixed** (runbook §6.5 banner); risk acceptance alone insufficient |
| N3 | Recovery-guidance text misdirects (`gateway/server.go` abort/indeterminate messages) | Operator confusion (retry vs new-turn vs fresh-assign) | Narrow | Per-case next-action wording + message assertions in existing tests | None (track) |
| N4 | M5 collision + unmarked v1 | Misread completion claims | — | **CLOSED this pass** | None |
| N5 | Claim pid recorded post-capture (`server.go` RecordLaunch) | Crash in window → pidless claim (correctly indeterminate) + possibly live unidentifiable orphan | Part 2 recovery reliance | Persist pid immediately post-`Runner.Start`; kill-after-start-pre-capture test | Disclose before depending on automated orphan recovery; Part 2 runbook must record pre-restart PIDs + verify absence |
| N6 | Launch capture has no timeout (`opencode.go` head loop) | Hung binary holds build sem; redeliveries 503 | Part 2 recovery reliance | Bounded capture timeout → kill + honest failure; hanging-runner test | Do not claim bounded startup recovery |
| N7 | Deliverer transition errors ignored (`deliver.go` pass) | Dead disk spins silently (safety-neutral via dedup) | Narrow | Log errors + deliverer-error counter in status; fault-injection test | None (track) |
| N8 | Staging permissive-mode window (`fly/sprites.go` WriteFile 0755/0644 + setup ordering) | Transient world-readable key bytes if staging is recreated beside running siblings | Part 2 provisioning | Restrictive modes at write time for keys-* (or verify-and-fix dir mode first); fake-fs mode test | **Do not re-provision beside untrusted siblings** until fixed; owner acknowledges constraint |
| N9 | Adapter Continue check-then-act (`opencode.go` live-check) | Unreachable via serialized server; future direct callers re-open F4 | Narrow | Per-session mutex spanning check→launch; direct-adapter concurrency test | None (server sem remains the relied-upon boundary) |
| N10 | Silent IOGID zero-fallback (`server.go` egid default) | Masks misconfiguration (fail-closed but confusing) | Narrow | Refuse explicit misconfiguration; zero-IOGID refusal test | None (track) |

Nothing here is promoted to fixed; nothing dismissed.

## 10. Exact list of checks not run and why

- Live proofs (R1–R10, EACCES, setup.sh, OpenCode, OAuth, wake):
  unauthorized (Part 2) — not run by design.
- `TestCheckAgentSpawnRealDrop`: skipped unprivileged by design
  (requires root for a genuine uid transition); refusal halves
  executed instead.
- One pre-existing flake observed twice: `TestAdapterStatus`
  TempDir-cleanup `ENOTEMPTY` (once in a verbose full run, once
  in six gateway-package runs). Root-caused to prompt11-era code
  untouched by this pass (test cancels a hung scripted run and
  returns while the adapter pump's async session write can land
  mid-cleanup). Left unmodified per scope; suite green on
  re-run (13/13).
- G4-style upstream comparison beyond R-NET: covered by R-NET.
- No general adversarial re-audit (out of directive scope).
- Tree-history change outside this session (see §2): two owner
  commits exist; the "all-untracked, zero-commit" boundary in the
  directive described review-time state. No action required, but
  future directives should re-baseline the boundary (and gain a
  real red/green baseline now that history exists).

## 11. Part 2 statement

**Part 2 was not executed or authorized in this pass.** No Sprite
was created, destroyed, provisioned, or modified; no Fly API
mutation ran; no OAuth ceremony started; OpenCode was not
installed or run on live infrastructure; no live runbook section
was executed; no production credentials were used or requested;
Miniapp, F1, G-01, P5, and frozen plans are untouched; this pass
performed no commits, pushes, stages, or history changes (the two
owner commits in §2 predate/are outside this session).

## 12. Final recommendation

N1, N4, R-DB, and R-NET are independently closed with executed
evidence; N2–N10 are explicitly tracked with safe operational
restrictions (notably the N2 live-upgrade prohibition and the N8
re-provisioning constraint). The remaining Part 2 prerequisites
are the live gates themselves (R-LIVE) plus N2/N5/N8 handling at
their point of use.

**READY FOR SEPARATE PART 2 AUTHORIZATION**

This status is a recommendation only. It does not authorize Part 2.
