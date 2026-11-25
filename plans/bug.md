## prompt

give more background story into the issue, high level overview of the problem and how do you plan to address the underlying issue (not band-aid fix), do a technical writeup as plans/writeup6.md wait for next instruction

## muse

# Writeup 6 — Why the Boundary B Browser Matrix Keeps Circling, and the Root-Cause Fixes

## 1. Background story

Boundary B's U5 phase requires a headless-Chrome acceptance matrix
against a live Studio server: login, conversations, Line messages,
sender dispatch to a stub gateway, signed gateway callbacks,
clarification replies, approvals, stale-answer 409s, all seven
Console scenes, density toggle, offline/reconnect, server-restart
resume, tenant isolation, mobile layout, and kill-switch behavior —
with database cross-checks at each step.

The matrix script (`/tmp/boundary-b/drive.py`, CDP-driven Chrome) has
now run four times. Each run passes further than the last — login,
gallery, messaging, live SSE dispatch receipts, callback-driven
questions, UI replies with frozen turn 2, continue acks, stale-answer
409s, approval and rejection decisions now all pass repeatedly — but
every run has died at the *next* uncovered step, and every fix has
required a full reset ritual: kill browsers, kill server, drop and
recreate the database, re-run migrations, re-provision operators,
restart the server, rerun from step one. Around U5's edges the same
sessions also surfaced three genuine product defects (a sender
`settleTx` lock-order deadlock, dispatch rows missing from the
stream, the evidence feed drowning in SSE-poll audits), each fixed
with regression tests. Those fixes are done and green; what remains
is finishing the matrix itself.

## 2. High-level overview of the problem

The circling is not one bug. It is a verification architecture with
three underlying technical causes plus one process cause. Each cycle
burns 10–20 minutes and yields exactly one finding because every
iteration restarts the world from zero.

**Cause 1 — Test and live environments share mutable global state:
the `studio_app` password.** `store/provision.go:EnsureAppRole`
unconditionally executes `ALTER ROLE studio_app WITH PASSWORD ...`
on every call. Every Go test-suite boot (`server`, `tools`,
`store`, `cmd/studio` harnesses) calls it with a fresh random
password. PostgreSQL roles are cluster-global, so each suite run
rotates the credential out from under any live server on the same
PG19 cluster — the Boundary B live server has died this way three
times, always presenting as a confusing `password authentication
failed` minutes after a green test run. Any workflow that
interleaves `go test` with a live server is broken by construction.

**Cause 2 — The framework audits every invocation, including
1Hz SSE polls, and the receipts ledger had no way to say "not
that".** The audit trail (`app.audit_events`, written by the
miniapp runtime's invoke path, not by Studio) records dispatched
*and* succeeded rows for every tool call — including each SSE
poll. A live browser therefore appends ~120 audit rows per minute.
The Evidence scene's `LIMIT 50` window filled with `conversation.
stream` polls within ~25 seconds, burying real receipts. This was
found live, not predicted: the matrix's evidence step timed out on
a receipt that had aged out of the window.

**Cause 3 — The sender wakes by schedule only; provisioning
completion has no signal.** When a build has no gateway URL the
sender holds the row for the 30s wait interval. Anything that
provisions the URL between polls (the matrix's SQL scaffolding,
a real Part 2 provision) then waits out the remainder of the hold
before dispatching. Correct and convergent — but every live
orchestration inherits a timing race, and the matrix burned two
full cycles on it (a 25s assertion window vs. a 30s hold).

**Cause 4 (process) — The matrix is one monolithic sequential
script with absolute assertions.** Kill-count ("Stopped 2 builds"),
"latest row" lookups, and epoch-second ID tags all assume a
pristine database, so any mid-run script bug forces a wipe and a
restart from step one. Script bugs (a truthiness check on a DOM
element, an approve/reject form mixup, attribute-vs-`innerText`
assertions) each cost a full cycle despite having nothing to do
with the product.

## 3. Addressing the underlying issues (not band-aids)

For each cause: what a band-aid would look like, and what the
structural fix is.

### 3.1 Role/password separation (Cause 1)

*Band-aids (rejected):* "don't run tests during live passes" (the
current workaround — process duct-tape around a design flaw);
fixed test passwords (still rotates the shared role; changes
nothing).

*Root fix:* separate role creation from rotation, and separate
test roles from the live role. Concretely: split `EnsureAppRole`
into create-if-missing plus grants (no password touch) and an
explicit rotate used only by the `provision` CLI; give test
harnesses their own cluster role (e.g. `studio_app_test`) granted
on test databases, so no suite run can ever mutate the live
credential again. Blast radius is real (all four test harnesses,
the provision CLI, runbook wording), so this is proposed as a
bounded follow-up work item with its own red-before/green-after
proof (live server survives a concurrent suite run) — *not*
smuggled into U5.

### 3.2 Receipts vs. transport audit (Cause 2)

*Band-aid (rejected):* display-side filtering after `LIMIT`
(first attempted — still starves under load) or raising the limit
(still time-bounded by design).

*Root fix (implemented):* the exclusion belongs in the query, so
the limit counts kept rows. `activity.list` gained an `exclude`
parameter (validated, max 16, schema-pinned in the tool
manifest); the Evidence scene passes `exclude:
["conversation.stream"]` and discloses it in the UI copy. This is
the correct layer, not a workaround: what gets audited is owned
by the frozen miniapp runtime (`app.*` schema, `runtime.Invoke`
path), which Studio must not fork for a display need — while
*which audit rows constitute receipts* is legitimately a Studio
read-model decision. Shipped with tool-level tests (exclusion,
unknown tools, oversized/invalid input) and a handler test. The
deeper alternative — framework audit categories (read vs.
mutate) — is miniapp-owned and correctly out of scope.

### 3.3 Provisioning completion signal (Cause 3)

*Band-aid (rejected):* shortening hold intervals (still polling,
still racy) or matrix-side `next_attempt_at` pokes as a permanent
pattern (test scaffolding masquerading as product behavior).

*Position:* polling is correct — it always converges — so no
product change is required for Boundary B. The structural
improvement, if the owner wants it later, is a completion signal:
a trigger on `builds.gateway_url` + `LISTEN/NOTIFY` (or a
provisioning outbox row) that wakes only the affected build's
rows due immediately. That is new machinery with its own failure
modes (missed notifies, restart replay) and is explicitly
deferred; the matrix keeps its documented SQL wake as labeled
scaffolding, with hold→dispatch-on-due covered at Go level.

### 3.4 Matrix structure (Cause 4, immediate)

*Band-aid (rejected):* more reruns of the monolith, longer
timeouts, looser assertions.

*Root fix (this week, before resuming):* restructure the matrix
so a script bug costs seconds, not a full reset —
(1) split `drive.py` into per-phase scripts runnable against
existing state (each phase sets up only what it asserts);
(2) replace absolute assertions with relative ones (live-build
count before/after kill instead of "Stopped 2"; per-phase unique
titles instead of "latest row" lookups);
(3) keep wipe-and-rerun for exactly one final clean-slate pass
at the end, quoted as the acceptance evidence.
No product-code change is involved; the Go suite already covers
the logic under every matrix step, so restructuring the browser
layer cannot weaken the gate.

## 4. What is *not* changing

- No F1/G-01/P5/miniapp-framework changes; no Sprite, Fly, live
  OpenCode, Part 2/C5, or Phase D/E work; no commits without
  authorization.
- F7 (SSE poll cost/lifetime) stays a tracked residual per the
  C6 authorization; the audit-exclusion work above treats one of
  its symptoms at the right layer without claiming the gate.
- The browser matrix keeps its full scope (all seven scenes, all
  eight UI states, concurrency, isolation, keyboard, mobile) —
  restructured, not trimmed — unless the owner directs otherwise.

## 5. Status and next step

Product code for U1–U4 is complete with full gates green (13/13
packages, plus `-race` on the concurrency tests). The U5 matrix
passes through approval/rejection decisions and honest
placeholders; remaining live items are evidence receipts,
density, offline/reconnect, restart resume, black-hole retrying,
keyboard kill, tenant-B isolation, mobile sheet, and
screenshots. Next action on approval: restructure the matrix per
§3.4, run it to green, then file the Boundary B report and stop.
