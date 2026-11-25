#!/usr/bin/env python3
"""Phase 1 — environment and authentication.

Verifies: server health, database identity vs. ledger, operator login
for both tenants, tenant context in the Console. No fixtures created.
Rerunnable: pure verification, no ledger writes except epochs.
"""
import asyncio
import os
import sys
import urllib.error
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from lib.harness import launch, login  # noqa: E402
from lib.phase import check_db, check_readyz, load_context  # noqa: E402


async def _amain(ctx):
    check_readyz(ctx)
    check_db(ctx)
    base, run_dir = ctx["base"], ctx["run_dir"]
    # Stub gateway answers (400 on empty body proves it is alive).
    req = urllib.request.Request(f"http://127.0.0.1:18093/gw/v1/command",
                                 data=b"{}", method="POST")
    try:
        urllib.request.urlopen(req, timeout=10)
        raise RuntimeError("PREREQ: stub gateway accepted an empty envelope")
    except urllib.error.HTTPError as e:
        if e.code != 400:
            raise RuntimeError(f"PREREQ: stub gateway answered {e.code}")
    print("PASS environment: server ready, database matches ledger, stub alive")

    c = await launch(19223, os.path.join(run_dir, "chromeA"))
    await c.call("Page.enable")
    await c.call("Runtime.enable")
    await login(c, base, ctx["tokens"]["t-demo"])
    print("PASS login tenant t-demo ->", await c.js("location.pathname"))
    await c.nav(f"{base}/console?scene=gallery")
    await c.wait_js("document.body.innerText.includes('t-demo')")
    print("PASS tenant context t-demo visible in Console")

    b = await launch(19224, os.path.join(run_dir, "chromeB"))
    await b.call("Page.enable")
    await b.call("Runtime.enable")
    await login(b, base, ctx["tokens"]["t-demo-b"])
    print("PASS login tenant t-demo-b")
    return []


def run(ledger_path, db, env):
    ctx = load_context(ledger_path, db, env)
    return asyncio.run(_amain(ctx))
