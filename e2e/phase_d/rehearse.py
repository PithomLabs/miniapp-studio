#!/usr/bin/env python3
"""Phase D rehearsal: one honest claim-to-decide run through a live server.

Flow: provision an isolated run DB, start studio + stub gateway, drive a
real browser (login, conversation, Line message), play the Builder through
the genuine signed-callback + fetch path with a generated coherent bundle,
then review in the browser: verified tag, Run tests, Approve. Screenshots
at every stage land under <run-dir>/shots/.

The Builder side is scripted (callbacks + fixture bytes); everything
Studio-side is production: ingest, sender fetch, auto-verify, container
test run, decide, and all Console rendering.

Usage: rehearse.py [--run-dir DIR] [--db NAME] [--keep]
"""
import argparse
import asyncio
import json
import os
import subprocess
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, os.path.join(REPO, "e2e", "boundary_b"))

from lib.harness import launch, login, psql, run_callbacks, submit_and_wait  # noqa: E402
from lib.serverctl import start_all, stop_all  # noqa: E402

BASE = "http://127.0.0.1:18082"
TENANT = "t-demo"


def sh(args, **kw):
    p = subprocess.run(args, capture_output=True, text=True, timeout=kw.pop("timeout", 300), **kw)
    if p.returncode != 0:
        raise RuntimeError(f"{' '.join(args)} failed:\n{p.stdout}\n{p.stderr}")
    return p.stdout.strip()


def wait_psql(dsn, sql, want="1", timeout=300, every=2):
    end = time.time() + timeout
    while time.time() < end:
        if psql(dsn, sql) == want:
            return True
        time.sleep(every)
    raise RuntimeError(f"wait timeout: {sql} never became {want!r}")


async def _open_session(base, run_dir):
    """First browser session: conversation, Line message, app row."""
    token = open(os.path.join(run_dir, "tokenA")).read().strip()
    c = await launch(19223, os.path.join(run_dir, "chromeA"))
    await c.call("Page.enable")
    await c.call("Runtime.enable")
    await login(c, base, token)

    # Conversation + Line message through the real forms.
    await c.nav(f"{base}/console?scene=gallery")
    await c.wait_js("!!document.querySelector('input[name=title]')")
    await c.js("document.querySelector('input[name=title]').value = 'Invoice Review Desk'")
    await submit_and_wait(c, "document.querySelector('form[action=\"/console/conversations\"]').submit()")
    conv = await c.js("new URL(location.href).searchParams.get('conversation')")
    assert conv, "gallery submit did not focus a conversation"
    await c.wait_js("!!document.querySelector('#lineInput')")
    await c.js("document.querySelector('#lineInput').value = 'Build the invoice review desk'")
    await submit_and_wait(c, "document.querySelector('#lineForm').submit()")
    print("PASS browser opened conversation + build:", conv)

    # The operator registers the app first (real product surface); the
    # Builder learns its id the way a real one would — the app page —
    # and binds the bundle to it.
    await c.nav(f"{base}/apps/new")
    await c.wait_js("!!document.querySelector('input[name=slug]')")
    await c.js("document.querySelector('input[name=slug]').value = 'invoice-review'")
    await c.js("document.querySelector('input[name=name]').value = 'Invoice Review Desk'")
    await submit_and_wait(c, "document.querySelector('form[action=\"/apps\"]').submit()")
    app_id = await c.js("location.pathname.split('/').pop()")
    assert app_id, "app creation did not land on the app page"
    print("PASS browser registered app:", app_id)
    return conv, app_id


async def _review_session(base, run_dir, conv, digest):
    """Second browser session: verify → test → approve with shots."""
    token = open(os.path.join(run_dir, "tokenA")).read().strip()
    c = await launch(19223, os.path.join(run_dir, "chromeA"))
    await c.call("Page.enable")
    await c.call("Runtime.enable")
    await login(c, base, token)
    await _review(c, base, run_dir, conv, digest)


async def _review(c, base, run_dir, conv, digest):
    shots = os.path.join(run_dir, "shots")

    async def shot(name, scene):
        await c.nav(f"{base}/console?scene={scene}&conversation={conv}")
        await c.wait_js("!!document.querySelector('.scene')")
        await c.shot(os.path.join(shots, name))

    # Validate shows the verified bundle with checks and forms.
    await c.nav(f"{base}/console?scene=validate&conversation={conv}")
    await c.wait_js("document.body.innerText.includes('verified')")
    assert await c.js(f"document.body.innerText.includes({digest!r})"), "validate lacks digest"
    await c.shot(os.path.join(shots, "validate_verified.png"))
    print("PASS validate shows the verified bundle")

    # Run tests through the real button; the container run takes minutes.
    await submit_and_wait(
        c, "document.querySelector('form[action=\"/console/artifact-test\"]').submit()",
        timeout=600)
    await c.wait_js("document.body.innerText.includes('Tests ')")
    print("PASS test run finished:",
          await c.js("document.querySelector('.banner').innerText"))
    await c.shot(os.path.join(shots, "validate_tested.png"))

    # Spec + Evidence carry the canonical version and the ledger.
    await shot("spec.png", "spec")
    await shot("evidence.png", "evidence")

    # Approve through the real button.
    await c.nav(f"{base}/console?scene=validate&conversation={conv}")
    await submit_and_wait(
        c, "document.querySelector('form[action=\"/console/artifact-decide\"]"
        " input[value=\"approved\"]').form.submit()")
    await c.wait_js("document.body.innerText.includes('Approved artifact')")
    print("PASS browser approved:",
          await c.js("document.querySelector('.banner').innerText"))
    await shot("evidence_decided.png", "evidence")
    await shot("run_pulse.png", "run")
    assert await c.js("document.body.innerText.includes('artifact_decided')"), \
        "pulse lacks the decision receipt"


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--run-dir", default="")
    ap.add_argument("--db", default="phase_d_rehearsal")
    ap.add_argument("--keep", action="store_true",
                    help="leave studio + stub running after the rehearsal")
    args = ap.parse_args()
    run_dir = args.run_dir or f"/tmp/phase-d/runs/{time.strftime('%Y%m%d-%H%M%S')}"
    fixture_dir = os.path.join(run_dir, "fixture")
    os.makedirs(fixture_dir, exist_ok=True)

    prov = os.path.join(REPO, "e2e", "boundary_b", "scripts", "provision_run.sh")
    print("== provision", run_dir, args.db)
    sh([prov, run_dir, args.db])
    dsn = open(os.path.join(run_dir, "owner.dsn")).read().strip()

    os.environ["PHASE_D_FIXTURE_DIR"] = fixture_dir
    print("== start studio + stub")
    ident = start_all(run_dir, BASE, repo=REPO)
    print("   binary", ident["binary_sha256"][:16])

    conv, app_id = asyncio.run(_open_session(BASE, run_dir))
    build = psql(dsn, "SELECT id FROM studio.builds WHERE conversation_id = "
                 f"'{conv}' ORDER BY created_at DESC LIMIT 1")
    turn = psql(dsn, "SELECT id FROM studio.turns WHERE build_id = "
                f"'{build}' ORDER BY n DESC LIMIT 1")
    assert build and turn, "build/turn rows missing"

    print("== generate coherent bundle")
    digest = sh(["go", "run", "./e2e/phase_d/cmd/genfixture",
                 "--spec", "spec/testdata/valid_invoice_review.json",
                 "--out", fixture_dir, "--tenant", TENANT,
                 "--conv", conv, "--build", build, "--turn", turn,
                 "--app-id", app_id], cwd=REPO)
    claim = json.load(open(os.path.join(fixture_dir, "claim.json")))
    assert claim["digest"] == digest, "claim/digest mismatch"
    print("   digest", digest)

    print("== claim callback (agent.completed, signed)")
    out = run_callbacks(BASE, os.path.join(run_dir, "orgsecret"), build,
                        TENANT, "rh-claim-1", "agent.completed",
                        {"artifacts": [claim]})
    assert '"transitioned":true' in out.replace(" ", ""), f"claim did not transition: {out}"

    print("== wait for fetch + auto-verify")
    wait_psql(dsn, "SELECT count(*) FROM studio.artifacts "
              f"WHERE digest = '{digest}' AND verified_at IS NOT NULL")
    passed = psql(dsn, "SELECT passed FROM studio.artifact_verifications WHERE artifact_id = "
                  f"(SELECT id FROM studio.artifacts WHERE digest = '{digest}')")
    assert passed == "t", "verification evidence did not pass"
    nfetch = psql(dsn, "SELECT count(*) FROM studio.commands WHERE build_id = "
                  f"'{build}' AND kind = 'fetch_artifact' AND state = 'acked'")
    print(f"PASS fetched + verified ({nfetch} fetch rows acked, evidence passed)")

    asyncio.run(_review_session(BASE, run_dir, conv, digest))
    print("== rehearsal complete:", run_dir)
    if not args.keep:
        stop_all()
        subprocess.run(["fuser", "-k", "19223/tcp"], capture_output=True, timeout=30)


if __name__ == "__main__":
    main()
