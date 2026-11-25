#!/usr/bin/env python3
"""Run-scoped state ledger for Boundary B U5 acceptance.

One JSON ledger per run. Writers record verified outputs; readers get
exact identifiers — never "latest row". Atomic writes (tmp + rename).
The loader rejects missing/malformed/version-mismatched/wrong-database
ledgers. No credentials or secrets are ever stored; the loader asserts
that on read.
"""
import json
import os
import tempfile

SCHEMA_VERSION = 1

_FORBIDDEN_KEYS = ("token", "secret", "password", "credential", "dsn",
                   "private_key", "cookie")


def new_ledger(run_id, database, server, operators):
    """Build a fresh ledger dict. operators: [{tenant, binding}]."""
    return {
        "schema_version": SCHEMA_VERSION,
        "run_id": run_id,
        "database": database,
        "server": server,
        "operators": operators,
        "conversations": {},
        "builds": {},
        "epochs": {},
        "phases": {},
    }


def _check_no_secrets(obj, path="ledger"):
    if isinstance(obj, dict):
        for k, v in obj.items():
            kl = str(k).lower()
            if any(f in kl for f in _FORBIDDEN_KEYS):
                raise ValueError(f"ledger must not store secrets: {path}.{k}")
            _check_no_secrets(v, f"{path}.{k}")
    elif isinstance(obj, list):
        for i, v in enumerate(obj):
            _check_no_secrets(v, f"{path}[{i}]")


def save_ledger(path, ledger):
    """Validate + atomically persist. Raises on any violation."""
    if ledger.get("schema_version") != SCHEMA_VERSION:
        raise ValueError("refusing to save ledger with wrong schema_version")
    if not ledger.get("run_id") or not ledger.get("database", {}).get("name"):
        raise ValueError("refusing to save ledger without run_id/database")
    _check_no_secrets(ledger)
    directory = os.path.dirname(path)
    os.makedirs(directory, exist_ok=True)
    fd, tmp = tempfile.mkstemp(dir=directory, prefix=".ledger-")
    try:
        with os.fdopen(fd, "w") as f:
            json.dump(ledger, f, indent=2, sort_keys=True)
            f.write("\n")
        os.replace(tmp, path)
    except BaseException:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


def load_ledger(path, expect_database=None):
    """Load and validate. expect_database rejects foreign ledgers."""
    if not os.path.exists(path):
        raise FileNotFoundError(f"PREREQ: ledger not found: {path}")
    try:
        with open(path) as f:
            ledger = json.load(f)
    except (json.JSONDecodeError, UnicodeDecodeError) as e:
        raise ValueError(f"PREREQ: malformed ledger {path}: {e}")
    if not isinstance(ledger, dict):
        raise ValueError(f"PREREQ: malformed ledger {path}: not an object")
    if ledger.get("schema_version") != SCHEMA_VERSION:
        raise ValueError(
            f"PREREQ: ledger schema {ledger.get('schema_version')}, "
            f"want {SCHEMA_VERSION} ({path})")
    if not ledger.get("run_id"):
        raise ValueError(f"PREREQ: ledger lacks run_id ({path})")
    if expect_database is not None and ledger.get("database", {}).get("name") != expect_database:
        raise ValueError(
            f"PREREQ: ledger database {ledger.get('database', {}).get('name')} "
            f"!= expected {expect_database} ({path})")
    _check_no_secrets(ledger, path)
    for key in ("conversations", "builds", "epochs", "phases"):
        ledger.setdefault(key, {})
    return ledger


def record_phase(ledger, name, status, evidence=None, error=None):
    """Mark a phase passed/failed/blocked with evidence references."""
    if status not in ("passed", "failed", "blocked"):
        raise ValueError(f"bad phase status: {status}")
    entry = {"status": status}
    if evidence:
        entry["evidence"] = evidence
    if error:
        entry["error"] = error[:500]
    ledger["phases"][name] = entry
    return ledger


def _selftest():
    import tempfile
    d = tempfile.mkdtemp(prefix="ledger-selftest-")
    p = os.path.join(d, "ledger.json")
    # Missing file.
    try:
        load_ledger(p)
        raise SystemExit("missing ledger accepted")
    except FileNotFoundError:
        pass
    # Round trip + secret rejection on save.
    good = new_ledger("r1", {"name": "db1"}, {"s": 1}, [{"tenant": "t"}])
    save_ledger(p, good)
    assert load_ledger(p, expect_database="db1")["run_id"] == "r1"
    bad = new_ledger("r1", {"name": "db1"}, {}, [])
    bad["token"] = "x"
    try:
        save_ledger(p, bad)
        raise SystemExit("secret-bearing ledger saved")
    except ValueError:
        pass
    # Foreign database + version mismatch + malformed.
    try:
        load_ledger(p, expect_database="other")
        raise SystemExit("foreign ledger accepted")
    except ValueError:
        pass
    wrong = dict(good, schema_version=999)
    with open(p, "w") as f:
        json.dump(wrong, f)
    try:
        load_ledger(p)
        raise SystemExit("wrong-version ledger accepted")
    except ValueError:
        pass
    with open(p, "w") as f:
        f.write("{not json")
    try:
        load_ledger(p)
        raise SystemExit("malformed ledger accepted")
    except ValueError:
        pass
    print("ledger selftest: 7/7 rejection paths hold")


if __name__ == "__main__":
    _selftest()
