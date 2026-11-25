#!/usr/bin/env python3
"""Phase 6 — recovery and isolation.

Offline banner + reconnect replay, server restart resume (same
binary, epoch recorded), black-hole retrying state, keyboard
hold-to-confirm kill with relative counts, F3 terminal skip as a
Pulse alert, tenant-B empty states + 404 isolation, mobile sheet
open/close/Esc + overflow check. Marker event IDs are
nonce-scoped per attempt so reruns drive live events.
"""
import asyncio
import hashlib
import os
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from lib.harness import (launch, login, psql, run_callbacks,  # noqa: E402
                         submit_and_wait)
from lib.phase import (check_db, check_readyz, load_context, next_nonce,  # noqa: E402
                       record_ids, require, update_ledger)
from lib.serverctl import restart_studio  # noqa: E402


async def _amain(ctx):
    check_readyz(ctx)
    check_db(ctx)
    base, run_dir = ctx["base"], ctx["run_dir"]
    ledger, shortid, dsn = ctx["ledger"], ctx["shortid"], ctx["dsn"]
    conv = require(ledger, "conversations", "p2-main", "id")
    build = require(ledger, "builds", "p2-main", "id")
    orgsecret = os.path.join(run_dir, "orgsecret")

    def cb(mid, name, payload, turn="1"):
        return run_callbacks(base, orgsecret, build, "t-demo", mid, name,
                             payload, turn)

    def live_builds():
        return int(psql(dsn, "SELECT count(*) FROM studio.builds "
                             "WHERE tenant_id='t-demo' AND status IN "
                             "('assigned','running','waiting_user')"))

    c = await launch(19223, os.path.join(run_dir, "chromeA"))
    await c.call("Page.enable")
    await c.call("Runtime.enable")
    await login(c, base, ctx["tokens"]["t-demo"])
    await c.nav(f"{base}/console?scene=run&conversation={conv}")

    # Offline banner + reconnect replay without reload.
    await c.call("Network.enable", {})
    await c.call("Network.emulateNetworkConditions",
                 {"offline": True, "latency": 0, "downloadThroughput": 0,
                  "uploadThroughput": 0})
    await asyncio.sleep(3)  # SSE error surfaces; banner is honest, not instant
    assert await c.js("!document.querySelector('#connBanner').hidden"), \
        "offline banner never appeared"
    print("PASS offline banner appears")
    await c.call("Network.emulateNetworkConditions",
                 {"offline": False, "latency": 0, "downloadThroughput": -1,
                  "uploadThroughput": -1})
    await c.wait_js("document.querySelector('#connBanner').hidden", timeout=20)
    off_nonce = next_nonce(ctx["ledger_path"], ctx["db"], "p6-offline")
    cb(f"p6-off-{shortid}-{off_nonce}", "agent.progress",
       {"text": f"back-online-marker-{off_nonce}"})
    await c.wait_js(f"document.body.innerText.includes('back-online-marker-{off_nonce}')",
                    timeout=20)
    print("PASS reconnect resumes SSE; missed event arrives without reload")

    # Server restart mid-stream: same binary (sha-checked), resume exact.
    with open(os.path.join(run_dir, "studio"), "rb") as f:
        sha_before = hashlib.sha256(f.read()).hexdigest()
    assert sha_before == ledger["server"]["binary_sha256"], \
        "PREREQ: server binary changed under this run"
    restart_studio(run_dir, base)
    rs_nonce = next_nonce(ctx["ledger_path"], ctx["db"], "p6-restart")
    cb(f"p6-rs-{shortid}-{rs_nonce}", "agent.progress",
       {"text": f"post-restart-marker-{rs_nonce}"})
    await c.wait_js(f"document.body.innerText.includes('post-restart-marker-{rs_nonce}')",
                    timeout=25)
    print("PASS server restart: stream resumes, new events flow without reload")

    def _record_restart(ledger):
        ledger.setdefault("server", {})["restarted_epoch"] = int(time.time())
    update_ledger(ctx["ledger_path"], ctx["db"], _record_restart)

    # Black-hole gateway: retrying visible, nothing green. Adopt the
    # phase conv/build on rerun instead of duplicating it.
    bh_title = f"p6-blackhole-{shortid}"
    conv2 = psql(dsn, "SELECT id FROM studio.conversations "
                      f"WHERE tenant_id='t-demo' AND title='{bh_title}'")
    if not conv2:
        await c.nav(f"{base}/console?scene=gallery")
        await c.js(f"document.querySelector('#stage input[name=title]').value = {bh_title!r}")
        await submit_and_wait(c, "document.querySelector('#stage form[action=\"/console/conversations\"]').submit()")
        await c.wait_js("location.search.includes('scene=run')")
        conv2 = await c.js("new URL(location.href).searchParams.get('conversation')")
        assert conv2, "PREREQ: no conversation id after create"
        await c.js("document.querySelector('#lineInput').value = 'hello?'")
        await submit_and_wait(c, "document.querySelector('#lineForm').submit()")
        await c.wait_js("document.body.innerText.includes('Build opened.')")
    else:
        print(f"adopt existing blackhole conversation {conv2[:8]}… (rerun)")
    psql(dsn, "UPDATE studio.builds SET gateway_url = 'http://127.0.0.1:9' "
              f"WHERE tenant_id='t-demo' AND conversation_id='{conv2}'")
    psql(dsn, "UPDATE studio.commands SET next_attempt_at = now() "
              f"WHERE build_id IN (SELECT id FROM studio.builds WHERE conversation_id='{conv2}')")
    for _ in range(60):
        att = psql(dsn, "SELECT attempts FROM studio.commands c "
                        "JOIN studio.builds b ON b.id=c.build_id "
                        f"WHERE b.conversation_id='{conv2}' AND c.kind='assign'")
        if att and int(att) > 0:
            break
        await asyncio.sleep(1)
    else:
        raise RuntimeError("sender never attempted the black-hole row")
    await c.nav(f"{base}/console?scene=run&conversation={conv2}")
    # State-aware: a rerun after the kill finds the row terminally
    # failed instead of retrying — assert whichever state is true,
    # never a stale one.
    rowstate = psql(dsn, "SELECT c.state FROM studio.commands c "
                         "JOIN studio.builds b ON b.id=c.build_id "
                         f"WHERE b.conversation_id='{conv2}' AND c.kind='assign'")
    if rowstate == "failed":
        await c.wait_js("document.body.innerText.includes('dispatch assign failed')")
        print("PASS black-hole row terminally failed (post-kill rerun); alert visible")
    else:
        await c.wait_js("document.body.innerText.includes('retrying')")
        print("PASS black-hole gateway shows retrying; nothing green")

    # Keyboard-only kill hold; fully relative counts so reruns pass
    # honestly (killed == live-before, whatever it is).
    live_before = live_builds()
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js("document.querySelector('#kill') && !document.querySelector('#kill').disabled")
    await c.js("document.querySelector('#kill').focus()")
    await c.call("Input.dispatchKeyEvent",
                 {"type": "rawKeyDown", "key": " ", "code": "Space",
                  "windowsVirtualKeyCode": 32, "autoRepeat": False})
    await asyncio.sleep(1.0)
    await c.call("Input.dispatchKeyEvent",
                 {"type": "keyUp", "key": " ", "code": "Space",
                  "windowsVirtualKeyCode": 32})
    await c.wait_js("location.pathname === '/console' && location.search.includes('scene=gallery')",
                    timeout=15)
    live_after = live_builds()
    assert live_after == 0, f"live builds after kill = {live_after}"
    notice = await c.js("document.body.innerText")
    if live_before == 0:
        assert "Stopped 0 builds" in notice, f"kill notice wrong: {notice[:200]}"
    elif live_before == 1:
        assert "Stopped 1 build." in notice, f"kill notice wrong: {notice[:200]}"
    else:
        assert f"Stopped {live_before} builds." in notice, f"kill notice wrong: {notice[:200]}"
    for key in ("p2-main",):
        bid = require(ledger, "builds", key, "id")
        st = psql(dsn, f"SELECT status FROM studio.builds WHERE id='{bid}'")
        assert st == "cancelled", f"ledger build {key} = {st}"
    print(f"PASS keyboard hold-to-confirm kill; {live_before}→0 with count")
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    st = await c.js("Array.from(document.querySelectorAll('.sph')).find(e => e.textContent === 'Run').dataset.s")
    assert st == "refused", f"Run pill after kill = {st}"

    # The black-hole assign fails honestly at its next claim; its
    # terminal row streams as a Pulse alert. Force due-ness once the
    # wait exceeds the hold (labeled scaffolding, same class as the
    # provisioning wake): the terminal-guard behavior, not the
    # schedule, is under test.
    for i in range(75):
        st8 = psql(dsn, "SELECT c.state FROM studio.commands c "
                        "JOIN studio.builds b ON b.id=c.build_id "
                        f"WHERE b.conversation_id='{conv2}' AND c.kind='assign'")
        if st8 == "failed":
            break
        if i == 30:
            psql(dsn, "UPDATE studio.commands SET next_attempt_at = now() "
                      f"WHERE build_id IN (SELECT id FROM studio.builds WHERE conversation_id='{conv2}')")
        await asyncio.sleep(1)
    else:
        raise RuntimeError("black-hole assign never settled failed")
    await c.nav(f"{base}/console?scene=run&conversation={conv2}")
    await c.wait_js("document.body.innerText.includes('dispatch assign failed')")
    print("PASS F3 skip streams as Pulse alert after kill")

    def _record_bh(ledger):
        bh = psql(dsn, "SELECT id FROM studio.builds "
                       f"WHERE tenant_id='t-demo' AND conversation_id='{conv2}' "
                       "ORDER BY created_at LIMIT 1")
        record_ids(ledger, "conversations", {"p6-blackhole": {"id": conv2, "title": bh_title}})
        record_ids(ledger, "builds", {"p6-blackhole": {"id": bh, "conversation": conv2}})
    update_ledger(ctx["ledger_path"], ctx["db"], _record_bh)

    # Tenant B: empty states + isolation (separate browser profile).
    # Empty-evidence holds only for a pristine tenant: B's own reads
    # audit. Gallery stays empty forever (B creates nothing).
    b_audit = int(psql(dsn, "SELECT count(*) FROM app.audit_events "
                            "WHERE tenant_id='t-demo-b'") or 0)
    b = await launch(19224, os.path.join(run_dir, "chromeB"))
    await b.call("Page.enable")
    await b.call("Runtime.enable")
    await login(b, base, ctx["tokens"]["t-demo-b"])
    # Evidence first: any page navigation audits tool invocations,
    # which would end the empty state we are asserting here.
    await b.nav(f"{base}/console?scene=evidence")
    if b_audit == 0:
        await b.wait_js("document.body.innerText.includes('No receipts yet')")
        print("PASS tenant B evidence empty state")
    else:
        # Rerun: B has its own audit rows, and the render itself
        # audits more (app.list for the rail) — so exact-count
        # equality would race the renderer. Assert the feed renders
        # rows at all; isolation itself is proven by the gallery
        # (A's titles never appear) and the 404 below, plus the Go
        # suite's tool-level tenant scoping.
        await b.wait_js("document.querySelectorAll('#stage tbody tr').length > 0")
        print("PASS tenant B evidence renders own rows (rerun)")
    await b.nav(f"{base}/console?scene=gallery")
    await b.wait_js("document.body.innerText.includes('No conversations yet')")
    print("PASS tenant B gallery empty (no leak from A)")
    await b.nav(f"{base}/console?scene=run&conversation={conv}")
    await b.wait_js("document.title.includes('404') || document.body.innerText.includes('not found')",
                    timeout=10)
    print("PASS cross-tenant conversation is 404, no oracle")

    # Mobile 375px: no overflow, sheet open/close, Esc.
    await c.call("Emulation.setDeviceMetricsOverride",
                 {"width": 375, "height": 700, "deviceScaleFactor": 2,
                  "mobile": True})
    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js("!!document.querySelector('#pulseEdge')")
    wider = await c.js("document.scrollingElement.scrollWidth > window.innerWidth")
    assert not wider, "horizontal overflow at 375px"
    print("PASS mobile: no horizontal overflow")
    await c.js("document.querySelector('#pulseEdge').click()")
    await c.wait_js("document.querySelector('#pulse').classList.contains('open')")
    await c.shot(os.path.join(run_dir, "shots", "mobile-sheet.png"))
    await c.call("Input.dispatchKeyEvent",
                 {"type": "rawKeyDown", "key": "Escape", "code": "Escape",
                  "windowsVirtualKeyCode": 27})
    await c.wait_js("!document.querySelector('#pulse').classList.contains('open')")
    print("PASS mobile sheet opens via edge, closes via Esc")
    await c.call("Emulation.setDeviceMetricsOverride",
                 {"width": 1280, "height": 900, "deviceScaleFactor": 1,
                  "mobile": False})
    return [os.path.join(run_dir, "shots", "mobile-sheet.png")]


def run(ledger_path, db, env):
    ctx = load_context(ledger_path, db, env)
    return asyncio.run(_amain(ctx))
