#!/usr/bin/env python3
"""Shared harness library for Boundary B U5 acceptance phases.

CDP client, login, psql helper, callback signer caller, Chrome
launcher, screenshot helper. No assertions here — phases own those.
"""
import asyncio
import base64
import json
import subprocess
import time
import urllib.request

import websockets


class CDP:
    def __init__(self, ws):
        self.ws = ws
        self.mid = 0
        self.pending = []

    def _result(self, method, msg):
        if "error" in msg:
            raise RuntimeError(f"{method}: {msg['error']}")
        return msg.get("result", {})

    async def call(self, method, params=None):
        self.mid += 1
        await self.ws.send(json.dumps({"id": self.mid, "method": method,
                                       "params": params or {}}))
        for i, msg in enumerate(self.pending):
            if msg.get("id") == self.mid:
                del self.pending[i]
                return self._result(method, msg)
        while True:
            msg = json.loads(await self.ws.recv())
            if msg.get("id") == self.mid:
                return self._result(method, msg)
            self.pending.append(msg)

    async def wait_event(self, method, timeout=15):
        """Wait for a CDP session event (e.g. Page.loadEventFired),
        buffering unrelated traffic instead of dropping it."""
        end = time.time() + timeout
        while time.time() < end:
            for i, msg in enumerate(self.pending):
                if msg.get("method") == method:
                    del self.pending[i]
                    return msg.get("params", {})
            try:
                raw = await asyncio.wait_for(self.ws.recv(), timeout=0.25)
            except asyncio.TimeoutError:
                continue
            msg = json.loads(raw)
            if msg.get("method") == method:
                return msg.get("params", {})
            self.pending.append(msg)
        raise RuntimeError(f"wait timeout for {method}")

    async def js(self, expr):
        r = await self.call("Runtime.evaluate",
                            {"expression": expr, "returnByValue": True})
        if r.get("exceptionDetails"):
            raise RuntimeError(f"js: {expr[:100]}: {r['exceptionDetails']}")
        return r["result"].get("value")

    def _purge(self, method):
        self.pending = [m for m in self.pending if m.get("method") != method]

    async def nav(self, url):
        self._purge("Page.loadEventFired")
        await self.call("Page.navigate", {"url": url})
        await self.wait_event("Page.loadEventFired", timeout=15)

    async def wait_js(self, expr, timeout=15):
        end = time.time() + timeout
        while time.time() < end:
            if await self.js(expr):
                return True
            await asyncio.sleep(0.25)
        raise RuntimeError(f"wait timeout: {expr[:100]}")

    async def shot(self, path):
        r = await self.call("Page.captureScreenshot", {"format": "png"})
        with open(path, "wb") as f:
            f.write(base64.b64decode(r["data"]))
        print("PASS screenshot", path)


async def launch(port, profile):
    # Rerunnable phases must not collide with a previous run's browser.
    subprocess.run(["fuser", "-k", f"{port}/tcp"], capture_output=True,
                   timeout=30)
    await asyncio.sleep(1)
    subprocess.Popen(
        ["google-chrome", "--headless=new", "--disable-gpu", "--no-sandbox",
         f"--remote-debugging-port={port}", f"--user-data-dir={profile}",
         "about:blank"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    for _ in range(100):
        try:
            targets = json.load(urllib.request.urlopen(
                f"http://127.0.0.1:{port}/json/list"))
            pages = [t for t in targets if t["type"] == "page"]
            if pages:
                ws = await websockets.connect(
                    pages[0]["webSocketDebuggerUrl"], max_size=8 << 20)
                return CDP(ws)
        except Exception:
            pass
        await asyncio.sleep(0.2)
    raise RuntimeError("chrome debug endpoint never came up")


async def submit_and_wait(c, submit_js, timeout=15):
    """Submit a form (or click a submit control) and wait for the
    resulting full navigation to load. Never assert on pathname
    alone: the old page still matches while the POST is in flight,
    and a follow-up navigate would cancel the submission."""
    c._purge("Page.loadEventFired")
    await c.js(submit_js)
    await c.wait_event("Page.loadEventFired", timeout=timeout)


async def login(c, base, token):
    await c.nav(f"{base}/login")
    # Reruns reuse the profile: a valid session may already have us
    # past the login page; only submit the form when it rendered.
    on_login = await c.js("location.pathname.startsWith('/login')")
    if on_login:
        await c.js(f"document.querySelector('input[name=token]').value = {token!r}")
        await c.js("document.querySelector('form[action=\"/login\"]').submit()")
        await c.wait_js("!location.pathname.startsWith('/login')")


def psql(dsn, sql, timeout=30):
    p = subprocess.run(["psql", dsn, "-tAc", sql], capture_output=True,
                       text=True, timeout=timeout)
    if p.returncode != 0:
        raise RuntimeError(f"psql failed: {p.stderr[:300]}")
    return p.stdout.strip()


def run_callbacks(base, orgsecret_path, build, workspace, mid,
                  name, payload, turn="1"):
    """Sign and POST one gateway callback; the server's 200 is the check."""
    import hashlib
    import hmac
    from datetime import datetime, timezone
    orgsecret = open(orgsecret_path).read().strip().encode()
    key = hmac.new(orgsecret, b"miniapp-gateway/v1:" + build.encode(),
                   hashlib.sha256).digest()
    env = {"protocol": "miniapp-gateway/1", "message_id": mid,
           "build_id": build, "turn_id": turn, "workspace_id": workspace,
           "sent_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
           "signature": "", "kind": "event", "name": name,
           "payload": {k: payload[k] for k in sorted(payload)}}
    body = json.dumps(env, separators=(",", ":")).encode()
    env["signature"] = hmac.new(key, body, hashlib.sha256).hexdigest()
    raw = json.dumps(env, separators=(",", ":")).encode()
    req = urllib.request.Request(f"{base}/gw/v1/callback", data=raw,
                                 headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            out = r.read().decode()
            print(f"   callbacks: {name} {r.status} {out[:160]}")
            return out
    except urllib.error.HTTPError as e:
        out = e.read().decode()
        raise RuntimeError(f"callback {name} failed: {e.code} {out[:200]}")
