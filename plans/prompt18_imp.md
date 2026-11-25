 Boundary A reached: Phase C backend (C1–C4, local-first) is implemented, green, and stopped. Full evidence:
  docs/prompt18-boundary-a.md.

  Delivered. Durable sender (server/sender.go) driving assign/continue/cancel with stable row ids, exact turn binding,
  stale/indeterminate reconcile-once recovery, 25-attempt budget, prerequisite waits that never burn, crash sweep, per-tenant
  RLS polling. Gateway gained SendWithID, status turn, and a six-code refusal taxonomy. Submit-time turn freeze,
  assigned-join handoff, sender running-flip, terminal callback mapping. SSE kill-and-resume, backlog drain, and concurrent
  no-gap proofs. Migrations 000017/000018.

  Evidence. 39 new tests, all green; gofmt/vet/build clean; full serial suite passes all 13 packages; -race clean on every
  concurrency test. Real Chrome pass against a live server: login → message → signed callbacks → question streamed with no
  reload → UI answer → frozen turn 2, with DB cross-checks. Three genuine reds fixed at root cause: a pre-existing
  send-vs-ingest deadlock (PG-log proven, fixed by lock ordering in appendStream), my own binding-vs-tenant workspace
  mismatch (plan says tenant; would have 404'd all callbacks), and a 500-on-duplicate-question retry poison (now 400).

  Needs your ratification. Yield-on-assigned now parks instead of absorbing — the old pin wedged builds assigned forever on
  lost started with no recovery. Test rewritten with rationale inline.

  Residuals. C5 live proof not run (gated); gateway binary is OpenCode-only so no live gateway process exists; dispatch state
  has no UI until C6; test boots rotate the cluster-global app password (broke my live server twice — re-provision after
  suites).

  No commits, no pushes, no Sprite/Fly/live-OpenCode operations; tree uncommitted; pass DB dropped; secrets never left /tmp
  0600 files. Next dependency on your GO: C6 Console (U4).

◆ Recap: Local-first C1–C4 backend is done and green with a real-browser pass. Next action is the owner's decision on the
         contract change and C5 authorization.
