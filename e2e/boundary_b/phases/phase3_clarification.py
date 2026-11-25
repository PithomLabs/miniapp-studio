#!/usr/bin/env python3
"""Phase 3 — clarification lifecycle.

Started + question callbacks stream live; reply submitted via UI with
frozen-turn DB check; continue ack streams; stale answer 409s naming
asked + current; sequential double-submit conflicts honestly.
Callback resends are idempotent (event-id dedup); the reply step is
ledger-gated (verify-instead-of-resubmit on rerun); stale and
double-submit steps re-assert 409s idempotently.
"""
import asyncio
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from lib.harness import (launch, login, psql, run_callbacks,  # noqa: E402
                         submit_and_wait)
from lib.phase import (SSE_LIVE_TIMEOUT_S, check_db, check_readyz,  # noqa: E402
                       load_context, next_nonce, record_ids, require,
                       update_ledger)


async def _amain(ctx):
    check_readyz(ctx)
    check_db(ctx)
    base, run_dir = ctx["base"], ctx["run_dir"]
    ledger, shortid, dsn = ctx["ledger"], ctx["shortid"], ctx["dsn"]
    conv = require(ledger, "conversations", "p2-main", "id")
    build = require(ledger, "builds", "p2-main", "id")
    status = psql(dsn, f"SELECT status FROM studio.builds WHERE id='{build}'")
    if status in ("succeeded", "failed", "cancelled", ""):
        raise RuntimeError(
            f"PREREQ: phase-2 build {build[:8]}… is '{status or 'missing'}'; "
            "clarification needs a live build (rerun phase 2 first on a fresh run)")
    orgsecret = os.path.join(run_dir, "orgsecret")

    def cb(mid, name, payload, turn="1"):
        return run_callbacks(base, orgsecret, build, "t-demo", mid, name,
                             payload, turn)

    q1 = f"p3-q1-{shortid}"
    cb(f"p3-started-{shortid}", "agent.started", {"agent_session_id": "sess-p3"})
    cb(f"p3-q1-{shortid}", "agent.question",
       {"question_id": q1, "body": "Which cost center?"})

    c = await launch(19223, os.path.join(run_dir, "chromeA"))
    await c.call("Page.enable")
    await c.call("Runtime.enable")
    await login(c, base, ctx["tokens"]["t-demo"])
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js("document.body.innerText.includes('Which cost center?')",
                    timeout=SSE_LIVE_TIMEOUT_S)
    print("PASS question streamed (live or server-rendered)")

    # Resumption is driven by DB state, not by the ledger: a failed
    # phase never records, so the ledger cannot distinguish "never
    # attempted" from "attempted then crashed". The ledger records
    # verified outcomes; the database is ground truth.
    q1state = psql(dsn, f"SELECT status FROM studio.questions WHERE id='{q1}'")
    if q1state != "answered":
        await c.wait_js(f"!!document.querySelector('[data-reply-qid=\"{q1}\"]')")
        await c.js(f"document.querySelector('[data-reply-qid=\"{q1}\"]').click()")
        await c.wait_js("!document.querySelector('#replyChip').hidden")
        await c.js("document.querySelector('#lineInput').value = 'CC-42'")
        await submit_and_wait(c, "document.querySelector('#lineForm').submit()")
        print("PASS UI reply submitted")
    else:
        print("reply already applied; verifying instead (rerun)")
    ans = psql(dsn, f"SELECT status, answer FROM studio.questions WHERE id='{q1}'")
    assert ans.startswith("answered|CC-42"), f"answer row = {ans}"
    frozen = psql(dsn, f"SELECT payload->>'turn' FROM studio.commands "
                       f"WHERE build_id='{build}' AND kind='continue' "
                       f"AND payload->>'question_id' = '{q1}'")
    assert frozen == "2", f"frozen turn = {frozen}"
    print("PASS UI reply recorded; continue froze turn 2")
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js("document.body.innerText.includes('dispatch continue acked')",
                    timeout=SSE_LIVE_TIMEOUT_S)
    print("PASS continue ack visible")

    # Stale pair: supersede, then answer the stale one → 409 naming
    # asked + current. Nonce-scoped IDs make every attempt (including
    # reruns) drive a live supersede pair instead of reusing dead ones.
    stale_nonce = next_nonce(ctx["ledger_path"], ctx["db"], "p3-stale")
    q2, q3 = f"p3-q2-{shortid}-{stale_nonce}", f"p3-q3-{shortid}-{stale_nonce}"
    cb(f"p3-q2-{shortid}-{stale_nonce}", "agent.question",
       {"question_id": q2, "body": "Second?"})
    cb(f"p3-q3-{shortid}-{stale_nonce}", "agent.question",
       {"question_id": q3, "body": "Third?"})
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js(f"!!document.querySelector('#lineForm input[name=opkey]')")
    await submit_and_wait(c, f"""(async () => {{
      const f = document.createElement('form');
      f.method = 'post'; f.action = '/console/line';
      const op = document.querySelector('#lineForm input[name=opkey]').value;
      const vals = {{opkey: op, conversation_id: {conv!r}, question_id: {q2!r}, body: 'x'}};
      for (const [k, v] of Object.entries(vals)) {{
        const i = document.createElement('input');
        i.type = 'hidden'; i.name = k; i.value = v; f.appendChild(i);
      }}
      document.body.appendChild(f); f.submit();
      return 'submitted';
    }})()""")
    await c.wait_js("document.body.innerText.includes('no longer open')")
    assert await c.js(f"document.body.innerText.includes({q2!r})"), "409 lacks asked qid"
    assert await c.js("document.body.innerText.includes('Third?')"), "409 lacks current"
    print("PASS stale answer → 409 naming asked + current")
    await c.shot(os.path.join(run_dir, "shots", "conflict-409.png"))

    # Sequential double-submit on a fresh live question: first applies,
    # second 409s — never double-applies. True concurrency stays covered
    # by the Go -race suite.
    dbl_nonce = next_nonce(ctx["ledger_path"], ctx["db"], "p3-double")
    qd = f"p3-qd-{shortid}-{dbl_nonce}"
    cb(f"p3-qd-{shortid}-{dbl_nonce}", "agent.question",
       {"question_id": qd, "body": "Double?"})
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js(f"!!document.querySelector('[data-reply-qid=\"{qd}\"]')")
    await c.js(f"document.querySelector('[data-reply-qid=\"{qd}\"]').click()")
    await c.wait_js("!document.querySelector('#replyChip').hidden")
    await c.js("document.querySelector('#lineInput').value = 'answer-d'")
    await submit_and_wait(c, "document.querySelector('#lineForm').submit()")
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js(f"!!document.querySelector('#lineForm input[name=opkey]')")
    await submit_and_wait(c, f"""(async () => {{
      const f = document.createElement('form');
      f.method = 'post'; f.action = '/console/line';
      const op = document.querySelector('#lineForm input[name=opkey]').value;
      const vals = {{opkey: op, conversation_id: {conv!r}, question_id: {qd!r}, body: 'again'}};
      for (const [k, v] of Object.entries(vals)) {{
        const i = document.createElement('input');
        i.type = 'hidden'; i.name = k; i.value = v; f.appendChild(i);
      }}
      document.body.appendChild(f); f.submit();
      return 'submitted';
    }})()""")
    await c.wait_js("document.body.innerText.includes('no longer open')")
    kept = psql(dsn, f"SELECT answer FROM studio.questions WHERE id='{qd}'")
    assert kept == "answer-d", f"second submit overwrote the answer: {kept}"
    print("PASS double-submit: first applies, second 409s, answer kept")

    def _record(ledger):
        record_ids(ledger, "questions", {
            "p3-q1": {"id": q1, "answered": True},
            "p3-q2-stale": {"id": q2}, "p3-q3": {"id": q3},
            "p3-qd": {"id": qd, "answered": True}})
    update_ledger(ctx["ledger_path"], ctx["db"], _record)
    return [os.path.join(run_dir, "shots", "conflict-409.png")]


def run(ledger_path, db, env):
    ctx = load_context(ledger_path, db, env)
    return asyncio.run(_amain(ctx))
