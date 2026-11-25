#!/usr/bin/env python3
"""Per-phase context and prerequisite helpers.

Every phase starts by loading its context (ledger + secrets + env)
and asserting its preconditions. Anything missing fails fast with a
PREREQ: diagnostic — never a downstream DOM assert.
"""
import os
import urllib.request

from lib.harness import psql
from lib.ledger import load_ledger


def load_context(ledger_path, db, env):
    """Load ledger (validating) and resolve run-scoped paths/secrets."""
    ledger = load_ledger(ledger_path, expect_database=db)
    run_dir = os.path.dirname(os.path.abspath(ledger_path))

    def secret(name):
        with open(os.path.join(run_dir, name)) as f:
            return f.read().strip()

    with open(os.path.join(run_dir, "owner.dsn")) as f:
        dsn = f.read().strip()
    return {
        "ledger": ledger,
        "ledger_path": os.path.abspath(ledger_path),
        "run_dir": run_dir,
        "run_id": ledger["run_id"],
        "shortid": ledger["run_id"][:8],
        "db": db,
        "dsn": dsn,
        "base": env["base"],
        "repo": env["repo"],
        "tokens": {"t-demo": secret("tokenA"), "t-demo-b": secret("tokenB")},
        "chrome_ports": (19223, 19224),
        "stub_base": "http://127.0.0.1:18093",
    }


def check_readyz(ctx):
    try:
        urllib.request.urlopen(ctx["base"] + "/readyz", timeout=5)
    except Exception as e:
        raise RuntimeError(f"PREREQ: server not ready at {ctx['base']}: {e}")


def check_db(ctx):
    try:
        name = psql(ctx["dsn"], "SELECT current_database()")
    except Exception as e:
        raise RuntimeError(f"PREREQ: database unreachable: {e}")
    if name != ctx["db"]:
        raise RuntimeError(
            f"PREREQ: connected database {name} != ledger {ctx['db']}")


def require(ledger, *path):
    """Fetch a ledger-recorded value or fail with a PREREQ diagnostic."""
    cur = ledger
    for key in path:
        if not isinstance(cur, dict) or key not in cur:
            raise RuntimeError(
                f"PREREQ: ledger lacks {'.'.join(path)} — run the owning phase first")
        cur = cur[key]
    return cur


def record_ids(ledger, section, mapping):
    """Merge verified identifiers into a ledger section."""
    ledger.setdefault(section, {}).update(mapping)


# Assertion deadline for sender dispatch, derived — not tuned: the 30s
# sender hold (server/sender.go:defaultSenderWait) plus 15s scheduling
# plus 15s dispatch allowance. A shorter window races the hold the
# product deliberately implements.
DISPATCH_DEADLINE_S = 60
# Live SSE assertions (no sender scheduling involved).
SSE_LIVE_TIMEOUT_S = 20


def update_ledger(ledger_path, db, fn):
    """Load, mutate via fn(ledger), and atomically save. Phases record
    verified outputs through this; run.py reloads afterwards, so no
    write is ever lost."""
    from lib.ledger import load_ledger, save_ledger
    ledger = load_ledger(ledger_path, expect_database=db)
    fn(ledger)
    save_ledger(ledger_path, ledger)


def next_nonce(ledger_path, db, key):
    """Durable per-run attempt counter. Callback/event IDs that must be
    unique per phase attempt incorporate this, so reruns neither
    collide with nor silently reuse prior attempts."""
    box = {}

    def _bump(ledger):
        ledger.setdefault("nonces", {})
        ledger["nonces"][key] = ledger["nonces"].get(key, 0) + 1
        box["n"] = ledger["nonces"][key]

    update_ledger(ledger_path, db, _bump)
    return box["n"]
