#!/usr/bin/env python3
"""Process lifecycle for acceptance runs: Studio server + stub gateway.

All processes are detached (setsid) with logs under the run dir so a
matrix run survives the launching shell; stop() kills by port. No
secrets ever appear in argv, logs, or errors here — env files only.
"""
import hashlib
import os
import subprocess
import time
import urllib.request

STUDIO_PORT = 18082
STUB_PORT = 18093
CHROME_PORTS = (19223, 19224)


def _repo_root():
    return subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True).strip()


def build_server(repo, out_path):
    """Compile the exact binary under test; returns its sha256."""
    p = subprocess.run(["go", "build", "-o", out_path, "./cmd/studio"],
                       cwd=repo, capture_output=True, text=True, timeout=300)
    if p.returncode != 0:
        raise RuntimeError(f"go build failed:\n{p.stdout}\n{p.stderr}")
    h = hashlib.sha256()
    with open(out_path, "rb") as f:
        h.update(f.read())
    return h.hexdigest()


def _wait_readyz(base, timeout=60):
    end = time.time() + timeout
    while time.time() < end:
        try:
            urllib.request.urlopen(base + "/readyz", timeout=2)
            return
        except Exception:
            time.sleep(1)
    raise RuntimeError(f"server never ready at {base}")


def start_all(run_dir, base, repo=None):
    """Build, start stub + studio, wait for health. Returns identity dict.
    Kills previous holders first: a stale server answering readyz would
    silently invalidate the whole run."""
    repo = repo or _repo_root()
    stop_port(STUDIO_PORT)
    stop_port(STUB_PORT)
    time.sleep(2)
    if _port_holders(STUDIO_PORT) or _port_holders(STUB_PORT):
        raise RuntimeError("PREREQ: ports still held after kill; investigate")
    binary = studio_binary(run_dir)
    sha = build_server(repo, binary)
    stub_log = open(os.path.join(run_dir, "logs", "stub.log"), "a")
    subprocess.Popen(
        ["setsid", "python3",
         os.path.join(repo, "e2e", "boundary_b", "stub", "gateway.py"),
         str(STUB_PORT), os.path.join(run_dir, "stub_calls.jsonl")],
        stdin=subprocess.DEVNULL, stdout=stub_log, stderr=subprocess.STDOUT,
        start_new_session=True, cwd=repo)
    _serve(run_dir, base, binary, repo)
    _wait_readyz(base)
    _assert_fresh_listener(STUDIO_PORT)
    _assert_fresh_listener(STUB_PORT)
    return {"binary_sha256": sha, "started_epoch": int(time.time()),
            "listen": base, "stub_port": STUB_PORT}


def stop_port(port):
    p = subprocess.run(["fuser", "-k", f"{port}/tcp"],
                       capture_output=True, text=True, timeout=30)
    return p.returncode == 0


def _port_holders(port):
    p = subprocess.run(["fuser", f"{port}/tcp"],
                       capture_output=True, text=True, timeout=30)
    holders = set()
    for tok in p.stderr.split() + p.stdout.split():
        if tok.isdigit():
            holders.add(int(tok))
    return holders


def _assert_fresh_listener(port, max_age_s=120):
    """Prove the listener is the process we just started: exactly one
    holder, recently started. Without this, a stale server from a
    previous run answers readyz and the new run silently tests it
    (observed live)."""
    holders = _port_holders(port)
    if len(holders) != 1:
        raise RuntimeError(
            f"PREREQ: port {port} has {len(holders)} holders, want exactly 1")
    pid = next(iter(holders))
    p = subprocess.run(["ps", "-o", "etimes=", "-p", str(pid)],
                       capture_output=True, text=True, timeout=30)
    try:
        age = int(p.stdout.strip())
    except ValueError:
        raise RuntimeError(f"PREREQ: cannot age listener on {port}")
    if age > max_age_s:
        raise RuntimeError(
            f"PREREQ: listener on {port} is {age}s old — stale server?")


def studio_binary(run_dir):
    """The single server binary for a run. Built only by build_server
    (at init) and rebuild_restart (explicit, ledger-recorded) — never
    by hand into a second path, or restarts silently resurrect stale
    code (observed live: restart revived a pre-fix binary)."""
    return os.path.join(run_dir, "studio")


def start_existing(run_dir, base, repo=None):
    """Start the run's existing binary without rebuilding. Used by
    restarts, which must preserve binary identity."""
    repo = repo or _repo_root()
    binary = studio_binary(run_dir)
    if not os.path.exists(binary):
        raise RuntimeError(f"PREREQ: no built server at {binary}")
    _serve(run_dir, base, binary, repo)
    _wait_readyz(base)


def _serve(run_dir, base, binary, repo):
    env = dict(os.environ,
               STUDIO_LISTEN=f"127.0.0.1:{STUDIO_PORT}",
               STUDIO_DATABASE_URL_FILE=os.path.join(run_dir, "app.dsn"),
               STUDIO_OWNER_DATABASE_URL_FILE=os.path.join(run_dir, "owner.dsn"),
               STUDIO_OPERATORS_FILE=os.path.join(run_dir, "ops.json"),
               MINIAPP_KEYS_FILE=os.path.join(run_dir, "keys.json"),
               MINIAPP_GRANTS_FILE=os.path.join(run_dir, "grants.json"),
               STUDIO_SENDER="1",
               STUDIO_GATEWAY_ORG_SECRET_FILE=os.path.join(run_dir, "orgsecret"))
    studio_log = open(os.path.join(run_dir, "logs", "studio.log"), "a")
    subprocess.Popen(
        ["setsid", binary, "serve"],
        stdin=subprocess.DEVNULL, stdout=studio_log, stderr=subprocess.STDOUT,
        env=env, start_new_session=True, cwd=repo)


def restart_studio(run_dir, base, repo=None):
    """Kill studio by port and start it again with identical wiring and
    the identical binary. Restarts must never rebuild: the phase
    asserting resume behavior must know exactly what came back."""
    stop_port(STUDIO_PORT)
    time.sleep(2)
    if _port_holders(STUDIO_PORT):
        raise RuntimeError("PREREQ: studio port still held after kill")
    start_existing(run_dir, base, repo=repo)
    _assert_fresh_listener(STUDIO_PORT)


def rebuild_restart(run_dir, base, repo=None):
    """Rebuild the run's server binary from current source and restart
    onto it. The ONLY sanctioned mid-run binary change (for product
    fixes found by the matrix itself); callers must record the new
    sha in the ledger with a reason. Returns the new identity dict."""
    repo = repo or _repo_root()
    binary = studio_binary(run_dir)
    sha = build_server(repo, binary)
    stop_port(STUDIO_PORT)
    time.sleep(2)
    if _port_holders(STUDIO_PORT):
        raise RuntimeError("PREREQ: studio port still held after kill")
    _serve(run_dir, base, binary, repo)
    _wait_readyz(base)
    _assert_fresh_listener(STUDIO_PORT)
    return {"binary_sha256": sha, "started_epoch": int(time.time())}


def stop_all():
    stop_port(STUDIO_PORT)
    stop_port(STUB_PORT)
