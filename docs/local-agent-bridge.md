# Local Agent Bridge (B0 manual procedure)

Studio manages the task; a human operates the local coding agent.
This bridge is temporary scaffolding with durable state: every step
below leaves rows in Studio. Nothing about the workflow changes when
later slices automate the handoffs.

## Procedure

1. **Describe.** Open a conversation in the Console gallery, title it
   with the task objective, and send one Line message stating scope
   and acceptance criteria. This opens the build (the work item).
2. **Collect the packet.** Open the run scene. The Execution packet
   card carries packet version `v0`: build/conversation ids,
   workspace identity, objective, task messages, branch hint, prior
   attempts, and evidence references. Copy what the agent needs.
3. **Execute locally.** Invoke the authorized local coding agent in
   the workspace checkout with the packet content. The operator
   chooses the branch; Studio does not launch anything in B0.
4. **Return the result.** On the run scene, submit Record attempt
   result: branch actually used, status digest, commits (if any),
   dirty-tree note, whether tests passed + summary, the agent
   report, and your verdict (accept/reject/rework) with a note.
5. **Evaluate.** The attempt lands in the Pulse and Evidence.
   Rejection or rework keeps the build open: run the agent again
   and record attempt N+1, which links its parent automatically.
   Acceptance closes the build; the receipt stays queryable.

## Rules

- Packet format `v0` is defined by the `build.packet` tool output.
  Consumers refuse unknown versions; format changes bump it.
- One form render = one operation key. Re-submitting the same form
  replays the recorded attempt; a fresh render records a new one.
- Reports, notes, and summaries scrub secret spans before storage.
  Never paste credentials into any field; digests and SHAs only.
- A terminal build refuses new attempts: continue via a new build.
- Attempt numbers come from the database, never the browser.
- The held gateway `assign` row on manual builds is expected: the
  sender waits without burning attempts or touching the network
  (residual, revisited when execution location drives dispatch).

## Limitations (B0)

Single project, single local workspace, no checkpoints/resume, no
remote handoff, no Preview, no workbench panel, no GitHub. Each
arrives in its own slice without changing this loop's records.

## B1 addendum: local loop with checkpoints

Set `STUDIO_WORKSPACE_ROOT` to the checkout (absolute path); without
it, Git-backed actions refuse honestly and the UI shows unavailable
states instead of guessing.

Extended loop: **start attempt** (opens attempt N) → work →
**capture checkpoint** (server inspects the worktree: branch, HEAD,
content digests, modes) → optionally **mark interrupted** → record
the attempt result as in B0 → **resume** a later attempt from the
latest complete checkpoint.

Rules added in B1:

- One open attempt per build; a second start conflicts (409).
- Checkpoints attach to the open attempt; numbering is server-side.
- Resume compares the live worktree (branch, HEAD, manifest digest)
  against the checkpoint. Any difference without explicit reconcile
  (+ note) refuses with the discrepancy named. Nothing is ever
  reset, cleaned, or overwritten.
- Resume honesty: B1 reconstructs context from durable records
  ("reconstructed"); it never claims a live agent session survived.
- Checkpoint capture inspects with read-only Git commands
  (`rev-parse`, `branch`, `status`, `ls-files`); bounds fail closed
  (oversized file/tree refuses the checkpoint, never a partial one).
- Progress (started/checkpoint/interrupted/resumed) streams to the
  Pulse as durable events; attempts link parents and source
  checkpoints automatically.
- Gateway-owned builds (waiting_user) are not eligible for local
  operations: answer or cancel the gateway question first. The
  refusal names the state; local checkpoint/resume never apply to
  gateway-driven execution.

## B1 boundaries the operator must know

- **Ignored files are outside the manifest.** Git status never
  reports them, so checkpoints cannot bind them. When ignored
  configuration (`.env`, local overrides) may matter to a resume,
  verify it by hand and record the outcome in the resume
  reconcile note — the reconcile path exists exactly for
  operator-verified state the manifest cannot see. Never paste
  secret values into notes to "prove" them; digests and SHAs only.
- **The recorded workspace path is provenance, not identity.** A
  checkpoint names the checkout it was captured from, but resume
  compares branch, HEAD, and manifest digest — never the path
  string. Moving a byte-identical tree and resuming there is
  legitimate in B1; workspace identity arrives in B2.

## Known residual risks (accepted, B1-R1)

- Concurrent parent-directory swap during hashing (TOCTOU below
  the per-component open): excluded by mechanism for final and
  intermediate components (O_NOFOLLOW throughout); a privileged
  local attacker racing directory replacement between the
  top-level resolution and the first open remains theoretically
  possible and is accepted as out of scope for a local-operator
  tool. No silent wrong-hash outcome is known: every observed
  failure mode refuses or notes.
- Sparse checkouts (skip-worktree) and assume-unchanged flags
  refuse capture/resume explicitly; supported if B2 needs them.
- Nested submodules refused explicitly; uninitialized, dirty, or
  mismatched submodules refuse explicitly.
- Ignored-file contents invisible to manifests (see above).
