#!/usr/bin/env python3
"""Boundary B U5 acceptance orchestrator.

Runs one phase (resumable, against existing state) or all phases in
order. A phase failure stops the run but destroys nothing; dependent
phases report blocked. Usage:
  run.py --phase N --ledger <path> [--db <name>]
  run.py --all --ledger <path> [--db <name>] [--fresh-db]
"""
import argparse
import importlib
import os
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from lib.ledger import (load_ledger, new_ledger, record_phase,  # noqa: E402
                        save_ledger)
from lib.serverctl import rebuild_restart, start_all  # noqa: E402

PHASES = [
    "phase1_environment",
    "phase2_conversation_dispatch",
    "phase3_clarification",
    "phase4_approvals_evidence",
    "phase5_console",
    "phase6_recovery_isolation",
]

DEPENDS = {
    "phase1_environment": [],
    "phase2_conversation_dispatch": ["phase1_environment"],
    "phase3_clarification": ["phase2_conversation_dispatch"],
    "phase4_approvals_evidence": ["phase3_clarification"],
    "phase5_console": ["phase1_environment"],
    "phase6_recovery_isolation": ["phase2_conversation_dispatch"],
}


def run_one(name, ledger_path, db, env):
    ledger = load_ledger(ledger_path, expect_database=db)
    for dep in DEPENDS[name]:
        if ledger["phases"].get(dep, {}).get("status") != "passed":
            record_phase(ledger, name, "blocked",
                         error=f"dependency {dep} has not passed")
            save_ledger(ledger_path, ledger)
            print(f"BLOCKED {name}: dependency {dep} has not passed")
            return False
    mod = importlib.import_module(f"phases.{name}")
    started = time.time()
    try:
        evidence = mod.run(ledger_path, db, env)
    except Exception as e:  # noqa: BLE001 - harness reports, never hides
        ledger = load_ledger(ledger_path, expect_database=db)
        record_phase(ledger, name, "failed", error=f"{type(e).__name__}: {e}")
        save_ledger(ledger_path, ledger)
        print(f"FAIL {name}: {type(e).__name__}: {e}")
        return False
    ledger = load_ledger(ledger_path, expect_database=db)
    ledger.setdefault("epochs", {})[name] = {
        "started": started, "finished": time.time()}
    record_phase(ledger, name, "passed", evidence=evidence or [])
    save_ledger(ledger_path, ledger)
    print(f"PASS phase {name}")
    return True


def do_init(ledger_path, db, env):
    """Provision a fresh run end to end: database, secrets, policy
    files, stub + server processes, initial ledger. Idempotent only
    in the sense that re-running with a new --db starts a new world;
    rerunning against an existing ledger is refused."""
    import subprocess
    here = os.path.dirname(os.path.abspath(__file__))
    if os.path.basename(ledger_path) != "ledger.json":
        raise SystemExit("ledger path must end with /ledger.json")
    run_dir = os.path.dirname(os.path.abspath(ledger_path))
    run_id = os.path.basename(run_dir)
    if os.path.exists(ledger_path):
        raise SystemExit(f"refusing to re-init existing ledger: {ledger_path}")
    if not db:
        raise SystemExit("--init requires --db <name>")
    script = os.path.join(here, "scripts", "provision_run.sh")
    p = subprocess.run(["bash", script, run_dir, db], capture_output=True,
                       text=True, timeout=600)
    print(p.stdout[-2000:])
    if p.returncode != 0:
        raise SystemExit(f"provision_run.sh failed:\n{p.stdout}\n{p.stderr}")
    identity = start_all(run_dir, env["base"], repo=env["repo"])
    ledger = new_ledger(
        run_id, {"name": db, "cluster": "PG19-5434"}, identity,
        [{"tenant": "t-demo", "binding": "demo-op"},
         {"tenant": "t-demo-b", "binding": "demo-op-b"}])
    save_ledger(ledger_path, ledger)
    print(f"PASS init run {run_id} on database {db}")
    return 0


def do_rebuild_restart(ledger_path, db, env, reason):
    """Rebuild the run binary from current source and restart onto it.
    The only sanctioned mid-run binary change; records old→new shas
    plus the reason in the ledger. Used when the matrix itself finds
    a product defect that must be fixed to proceed."""
    import time
    ledger = load_ledger(ledger_path, expect_database=db)
    run_dir = os.path.dirname(os.path.abspath(ledger_path))
    old = ledger.get("server", {}).get("binary_sha256", "")
    identity = rebuild_restart(run_dir, env["base"], repo=env["repo"])
    ledger = load_ledger(ledger_path, expect_database=db)
    ledger.setdefault("server", {}).update(identity)
    ledger["server"].setdefault("rebuilds", []).append({
        "at": int(time.time()), "from_sha": old,
        "to_sha": identity["binary_sha256"], "reason": reason})
    save_ledger(ledger_path, ledger)
    print(f"PASS rebuild-restart {old[:12]}… -> {identity['binary_sha256'][:12]}… ({reason})")
    return 0


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--phase", type=int, choices=range(1, 7))
    ap.add_argument("--all", action="store_true")
    ap.add_argument("--init", action="store_true",
                    help="provision a fresh run (db, secrets, servers, ledger)")
    ap.add_argument("--rebuild-restart", dest="rebuild_restart",
                    metavar="REASON",
                    help="rebuild run binary from source, restart, record in ledger")
    ap.add_argument("--ledger", required=True)
    ap.add_argument("--db", default=None)
    args = ap.parse_args()
    env = {
        "base": os.environ.get("STUDIO_BASE", "http://127.0.0.1:18082"),
        "run_dir": os.environ.get("BOUNDARY_RUN_DIR", "/tmp/boundary-b"),
        "repo": os.path.abspath(os.path.join(
            os.path.dirname(os.path.abspath(__file__)), "..", "..")),
    }
    if args.init:
        if args.phase or args.all:
            ap.error("--init takes no --phase/--all")
        return do_init(args.ledger, args.db, env)
    if args.rebuild_restart:
        if args.phase or args.all:
            ap.error("--rebuild-restart takes no --phase/--all")
        return do_rebuild_restart(args.ledger, args.db, env, args.rebuild_restart)
    if bool(args.phase) == bool(args.all):
        ap.error("pass exactly one of --phase N or --all")
    names = [PHASES[args.phase - 1]] if args.phase else list(PHASES)
    ok = True
    for name in names:
        if not run_one(name, args.ledger, args.db, env):
            ok = False
            if args.all:
                continue
            break
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
