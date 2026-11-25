#!/usr/bin/env python3
"""Phase 2 — conversation and dispatch.

Creates the run's main conversation (or adopts it by phase-unique
title on rerun), sends the Line message (or verifies the build),
provisions the stub gateway, and proves the assign ack streams live
with DB cross-checks. Mutating steps are ledger-gated: a rerun
verifies completed work instead of duplicating it.
"""
import asyncio
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from lib.harness import launch, login, psql, submit_and_wait  # noqa: E402
from lib.phase import (DISPATCH_DEADLINE_S, check_db, check_readyz,  # noqa: E402
                       load_context, record_ids, update_ledger)
from lib.serverctl import STUB_PORT  # noqa: E402


async def _amain(ctx):
    check_readyz(ctx)
    check_db(ctx)
    base, run_dir = ctx["base"], ctx["run_dir"]
    ledger, shortid = ctx["ledger"], ctx["shortid"]
    title = f"p2-invoice-{shortid}"
    dsn = ctx["dsn"]

    c = await launch(19223, os.path.join(run_dir, "chromeA"))
    await c.call("Page.enable")
    await c.call("Runtime.enable")
    await login(c, base, ctx["tokens"]["t-demo"])

    # Conversation: adopt by phase-unique title, else create via UI.
    conv = psql(dsn, "SELECT id FROM studio.conversations "
                     f"WHERE tenant_id='t-demo' AND title='{title}'")
    if conv:
        print(f"adopt existing conversation {conv[:8]}… (rerun)")
        await c.nav(f"{base}/console?scene=run&conversation={conv}")
    else:
        await c.nav(f"{base}/console?scene=gallery")
        await c.js(f"document.querySelector('#stage input[name=title]').value = {title!r}")
        await submit_and_wait(c, "document.querySelector('#stage form[action=\"/console/conversations\"]').submit()")
        await c.wait_js("location.pathname === '/console' && location.search.includes('scene=run')")
        conv = await c.js("new URL(location.href).searchParams.get('conversation')")
        assert conv, "PREREQ: no conversation id after create"
        print("PASS conversation created:", title)

    # Message: adopt if a build already exists, else send via Line.
    build = psql(dsn, f"SELECT id FROM studio.builds WHERE tenant_id='t-demo' "
                      f"AND conversation_id='{conv}' ORDER BY created_at LIMIT 1")
    if not build:
        await c.nav(f"{base}/console?scene=run&conversation={conv}")
        await c.js("document.querySelector('#lineInput').value = 'Review invoice INV-2024-001'")
        await submit_and_wait(c, "document.querySelector('#lineForm').submit()")
        await c.wait_js("document.body.innerText.includes('Build opened.')")
        build = psql(dsn, f"SELECT id FROM studio.builds WHERE tenant_id='t-demo' "
                          f"AND conversation_id='{conv}' ORDER BY created_at LIMIT 1")
        assert build, "PREREQ: no build after Line message"
        print("PASS message sent; build opened")
    else:
        print(f"adopt existing build {build[:8]}… (rerun)")
        await c.nav(f"{base}/console?scene=run&conversation={conv}")

    # Spine honesty (guardrail 1): exactly Describe proven, Run active.
    proven = await c.js("document.querySelectorAll('.sph[data-s=\"proven\"]').length")
    assert proven == 1, f"proven pills = {proven}, want 1 (Describe only)"
    run_state = await c.js("Array.from(document.querySelectorAll('.sph')).find(e => e.textContent === 'Run').dataset.s")
    assert run_state in ("active", "refused"), f"Run pill = {run_state}"
    assert run_state != "proven", "Run pill must never go green from builder state"
    print("PASS spine honest: Describe proven, Run active-not-green")

    # Provision stub gateway via SQL (provisioning surface is banned
    # scope) + documented SQL wake (labeled scaffolding; the hold is
    # covered at Go level). Idempotent on rerun.
    stub = f"http://127.0.0.1:{STUB_PORT}"
    psql(dsn, f"UPDATE studio.builds SET gateway_url = '{stub}' WHERE id = '{build}'")
    psql(dsn, f"UPDATE studio.commands SET next_attempt_at = now() WHERE build_id = '{build}'")
    await c.wait_js("document.body.innerText.includes('dispatch assign acked')",
                    timeout=DISPATCH_DEADLINE_S)
    print("PASS assign ack receipt streamed live via SSE (no reload)")
    row = psql(dsn, f"SELECT state, gateway_turn FROM studio.commands c "
                    f"JOIN studio.builds b ON b.id=c.build_id "
                    f"WHERE c.build_id='{build}' AND c.kind='assign'")
    # The assign row stays acked forever; the build's tracked turn only
    # grows as later phases dispatch continues. Exact turn==1 holds
    # solely before any continue exists — derived from DB state, not
    # from run history, so reruns after later phases stay honest.
    state, _, turn = row.partition("|")
    assert state == "acked", f"assign row = {row}"
    conts = psql(dsn, f"SELECT count(*) FROM studio.commands "
                      f"WHERE build_id='{build}' AND kind='continue'")
    if int(conts or 0) == 0:
        assert turn == "1", f"tracked turn with no continues = {turn}, want 1"
        print("PASS DB: assign acked, gateway_turn=1")
    else:
        assert int(turn) >= 1, f"tracked turn went backwards: {row}"
        print(f"PASS DB: assign acked (tracked turn now {turn} after continues)")

    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js("document.body.innerText.includes('dispatch assign acked')")
    await c.shot(os.path.join(run_dir, "shots", "desktop-run.png"))

    def _record(ledger):
        record_ids(ledger, "conversations", {"p2-main": {"id": conv, "title": title}})
        record_ids(ledger, "builds", {"p2-main": {"id": build, "conversation": conv}})
    update_ledger(ctx["ledger_path"], ctx["db"], _record)
    return [os.path.join(run_dir, "shots", "desktop-run.png")]


def run(ledger_path, db, env):
    ctx = load_context(ledger_path, db, env)
    return asyncio.run(_amain(ctx))
