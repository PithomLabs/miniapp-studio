#!/usr/bin/env python3
"""Phase 5 — console behavior.

All seven scenes: future phases honestly gray with phase notes,
settings states kill scope, OPS/DEV density flips detail visibility
over the identical DOM, and the no-JS baseline (curl: no JS engine
at all) proves server-rendered completeness. Fully read-only apart
from navigation: rerunnable by construction.
"""
import asyncio
import os
import subprocess
import sys
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from lib.harness import launch, login  # noqa: E402
from lib.phase import check_db, check_readyz, load_context, require  # noqa: E402


async def _amain(ctx):
    check_readyz(ctx)
    check_db(ctx)
    base, run_dir = ctx["base"], ctx["run_dir"]
    conv = require(ctx["ledger"], "conversations", "p2-main", "id")

    c = await launch(19223, os.path.join(run_dir, "chromeA"))
    await c.call("Page.enable")
    await c.call("Runtime.enable")
    await login(c, base, ctx["tokens"]["t-demo"])

    for scene, phase in [("spec", "Phase D"), ("validate", "Phase D"),
                         ("connect", "Phase D"), ("deploy", "Phase E")]:
        await c.nav(f"{base}/console?scene={scene}&conversation={conv}")
        await c.wait_js("document.body.innerText.includes('Not built yet')")
        assert await c.js(f"document.body.innerText.includes('{phase}')"), scene
        n = await c.js("document.querySelectorAll('.sph[data-s=\"proven\"]').length")
        assert n == 1, f"{scene}: proven pills = {n} (Describe only)"
    print("PASS spec/validate/connect/deploy honest placeholders; spine stays truthful")

    await c.nav(f"{base}/console?scene=settings&conversation={conv}")
    await c.wait_js("document.body.innerText.includes('only this tenant')")
    print("PASS settings states kill scope")

    await c.nav(f"{base}/console?scene=run&conversation={conv}")
    await c.wait_js("!!document.querySelector('#pDev')")
    before = await c.js("getComputedStyle(document.querySelector('.devd')).display")
    assert before == "none", f"ops devd display = {before}"
    await c.js("document.querySelector('#pDev').click()")
    await c.wait_js("getComputedStyle(document.querySelector('.devd')).display !== 'none'")
    assert await c.js("document.documentElement.dataset.persona") == "dev"
    await c.js("document.querySelector('#pOps').click()")
    await c.wait_js("getComputedStyle(document.querySelector('.devd')).display === 'none'")
    print("PASS OPS/DEV density toggle flips detail visibility, same DOM")

    # No-JS baseline via curl: no JS engine exists in this check, so
    # whatever renders is server-side by construction.
    token_file = os.path.join(run_dir, "tokenA")
    with open(token_file) as f:
        token = f.read().strip()
    jar = os.path.join(run_dir, "curl-cookies.txt")
    subprocess.run(
        ["curl", "-s", "-c", jar, "-o", "/dev/null", "-H", f"Origin: {base}",
         "--data-urlencode", f"token={token}", "--data-urlencode", "next=/",
         f"{base}/login"], check=True, timeout=30)
    page = subprocess.run(
        ["curl", "-s", "-b", jar,
         f"{base}/console?scene=run&conversation={conv}"],
        capture_output=True, text=True, check=True, timeout=30).stdout
    for marker in ["Builder run", 'id="lineForm"', 'name="body"',
                   "dispatch assign acked", "spine", "Pulse"]:
        assert marker in page, f"no-JS page lacks {marker}"
    print("PASS no-JS baseline: content + forms fully server-rendered")
    return []


def run(ledger_path, db, env):
    ctx = load_context(ledger_path, db, env)
    return asyncio.run(_amain(ctx))
