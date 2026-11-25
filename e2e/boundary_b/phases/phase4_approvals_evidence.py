#!/usr/bin/env python3
"""Phase 4 — approvals and Evidence.

Authorization approve + reject via Pulse cards with DB checks that
clarifications authorize nothing; Evidence receipts live with the
gate-ledger and transport-noise disclosures; measured audit write
rate over a 60s window with an open stream (feeds the F7 residual).
Approval questions are nonce-scoped per attempt, so reruns drive
live decisions instead of reusing decided ones.
"""
import asyncio
import os
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from lib.harness import (launch, login, psql, run_callbacks,  # noqa: E402
                         submit_and_wait)
from lib.phase import (check_db, check_readyz, load_context, next_nonce,  # noqa: E402
                       record_ids, require, update_ledger)


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
            f"PREREQ: phase-2 build {build[:8]}… is '{status or 'missing'}'")
    orgsecret = os.path.join(run_dir, "orgsecret")

    def cb(mid, name, payload, turn="1"):
        return run_callbacks(base, orgsecret, build, "t-demo", mid, name,
                             payload, turn)

    c = await launch(19223, os.path.join(run_dir, "chromeA"))
    await c.call("Page.enable")
    await c.call("Runtime.enable")
    await login(c, base, ctx["tokens"]["t-demo"])

    ap_nonce = next_nonce(ctx["ledger_path"], ctx["db"], "p4-approvals")
    before = psql(dsn, f"SELECT count(*) FROM studio.approvals "
                       f"WHERE tenant_id='t-demo' AND build_id='{build}'")
    for qid, body, decision, btn in [
            (f"p4-ap1-{shortid}-{ap_nonce}", "Ship it?", "approved", "primary"),
            (f"p4-ap2-{shortid}-{ap_nonce}", "Delete all?", "rejected", "danger")]:
        cb(f"p4-{qid}", "agent.authorization_request",
           {"question_id": qid, "body": body})
        await c.nav(f"{base}/console?scene=run&conversation={conv}")
        await c.wait_js(f"document.body.innerText.includes({body!r})")
        assert await c.js("document.body.innerText.includes('Approval required')")
        await submit_and_wait(c, f"""Array.from(document.querySelectorAll('#stream form[action="/console/decide"]'))
          .find(f => f.querySelector('input[name=question_id]').value === {qid!r} &&
                     f.querySelector('input[name=decision]').value === {decision!r})
          .querySelector('button.{btn}').click()""")
        await c.wait_js(f"document.body.innerText.includes({decision!r})")
        print(f"PASS {decision} decided via Pulse; decision rendered")
    after = psql(dsn, f"SELECT count(*) FROM studio.approvals "
                      f"WHERE tenant_id='t-demo' AND build_id='{build}'")
    assert int(after) - int(before) == 2, f"approvals {before} -> {after}"
    clar = psql(dsn, "SELECT count(*) FROM studio.approvals a "
                     "JOIN studio.questions q ON q.id = a.question_id "
                     f"AND q.tenant_id = a.tenant_id WHERE a.tenant_id='t-demo' "
                     f"AND a.build_id='{build}' AND q.kind <> 'authorization_request'")
    assert clar == "0", f"non-approval questions authorized: {clar}"
    print("PASS DB: exactly +2 approvals; clarifications authorize nothing")

    await c.nav(f"{base}/console?scene=evidence&conversation={conv}")
    await c.wait_js("document.body.innerText.includes('question.answer')")
    assert await c.js("document.body.innerText.includes('lands in Phase D')")
    assert await c.js("document.body.innerText.includes('hidden as transport noise')")
    print("PASS evidence receipts live; gate-ledger note present")
    await c.shot(os.path.join(run_dir, "shots", "evidence.png"))

    # Audit write rate, measured — not estimated. One open stream for
    # 60s; count stream-tool audit rows in the window.
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js("!!document.querySelector('#stream')")
    t0 = time.time()
    n0 = int(psql(dsn, "SELECT count(*) FROM app.audit_events "
                       "WHERE tenant_id='t-demo' AND tool='conversation.stream'") or 0)
    await asyncio.sleep(60)
    n1 = int(psql(dsn, "SELECT count(*) FROM app.audit_events "
                       "WHERE tenant_id='t-demo' AND tool='conversation.stream'") or 0)
    dt = time.time() - t0
    per_min = (n1 - n0) / dt * 60
    print(f"PASS audit measurement: {n1 - n0} stream rows in {dt:.0f}s "
          f"= {per_min:.0f}/min (~{per_min * 24:.0f}/stream/day)")

    def _record(ledger):
        record_ids(ledger, "questions", {
            f"p4-ap1-{ap_nonce}": {"id": f"p4-ap1-{shortid}-{ap_nonce}",
                                  "decided": "approved"},
            f"p4-ap2-{ap_nonce}": {"id": f"p4-ap2-{shortid}-{ap_nonce}",
                                  "decided": "rejected"}})
        ledger.setdefault("measurements", {})["audit_stream_rows_per_min"] = round(per_min, 1)
    update_ledger(ctx["ledger_path"], ctx["db"], _record)
    return [os.path.join(run_dir, "shots", "evidence.png")]


def run(ledger_path, db, env):
    ctx = load_context(ledger_path, db, env)
    return asyncio.run(_amain(ctx))
