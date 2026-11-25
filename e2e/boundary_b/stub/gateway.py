#!/usr/bin/env python3
"""Boundary B stub gateway: accepts sender commands over loopback HTTP and
records them. NOT an agent — it never emits callbacks or runs anything.
Responses mimic the real gateway's 200 shapes (unsigned: the Studio
client does not verify response signatures).

Usage: gateway.py <port> <call-log-path>
"""
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

LOG = sys.argv[2] if len(sys.argv) > 2 else "/tmp/boundary-b/stub_calls.jsonl"
turns = {}  # build_id -> last accepted turn


class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def do_POST(self):
        if self.path != "/gw/v1/command":
            self.send_response(404)
            self.end_headers()
            return
        raw = self.rfile.read(int(self.headers.get("Content-Length", 0)))
        try:
            env = json.loads(raw)
        except Exception:
            self.send_response(400)
            self.end_headers()
            return
        name = env.get("name")
        build = env.get("build_id")
        payload = env.get("payload") or {}
        with open(LOG, "a") as f:
            f.write(json.dumps({"name": name, "build": build,
                                "message_id": env.get("message_id"),
                                "payload": payload}) + "\n")
        if name == "assign":
            turns[build] = 1
            out = {"session_ref": "stub-sess-" + str(build)[:8]}
            code = 200
        elif name == "continue":
            # Mirror fencing truthfully: wrong turn is a 409 like the real server.
            want = turns.get(build, 0) + 1
            got = payload.get("turn")
            if got != want:
                out = {"error": "stale turn", "code": "stale_turn"}
                code = 409
            else:
                turns[build] = got
                out = {"resumed": True}
                code = 200
        elif name == "cancel":
            out = {"cancelled": True}
            code = 200
        elif name == "status":
            if build in turns:
                out = {"turn": turns[build], "state": "running",
                       "session_ref": "stub-sess-" + str(build)[:8]}
                code = 200
            else:
                out = {"error": "unknown build"}
                code = 404
        else:
            out = {"error": "unknown command"}
            code = 400
        body = json.dumps(out).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 18093
    open(LOG, "w").close()
    HTTPServer(("127.0.0.1", port), H).serve_forever()
