#!/usr/bin/env bash
# Provision one acceptance run: isolated database, secrets, policy files.
# Usage: provision_run.sh <run-dir> <db-name>
# Requires $BOUNDARY_HOME/owner.dsn (operator-bootstrapped once per
# cluster, 0600). Writes only into <run-dir>. Never echoes secrets:
# no xtrace in this file, ever.
set -euo pipefail

RUN_DIR="$1"
DB="$2"
HOME_DIR="${BOUNDARY_HOME:-/tmp/boundary-b}"
REPO="$(git rev-parse --show-toplevel)"

if [[ ! "$DB" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
	echo "refusing unsafe database name: $DB" >&2
	exit 1
fi
if [ ! -f "$HOME_DIR/owner.dsn" ]; then
	echo "missing $HOME_DIR/owner.dsn (bootstrap it once per cluster)" >&2
	exit 1
fi

mkdir -p "$RUN_DIR"/{logs,shots,chromeA,chromeB}
chmod 700 "$RUN_DIR"
exec >"$RUN_DIR/provision.log" 2>&1

echo "run dir: $RUN_DIR"
echo "database: $DB"
MAINT_DSN="$(cat "$HOME_DIR/owner.dsn")"
export PGPASSWORD
PGPASSWORD="$(python3 -c "import sys,urllib.parse; print(urllib.parse.urlparse(sys.argv[1]).password or '')" "$MAINT_DSN")"

echo "== drop/create database"
psql "$MAINT_DSN" -c "DROP DATABASE IF EXISTS $DB" -c "CREATE DATABASE $DB"

echo "== run database owner DSN"
python3 - "$HOME_DIR/owner.dsn" "$RUN_DIR/owner.dsn" "$DB" <<'EOF'
import sys, urllib.parse
maint = open(sys.argv[1]).read().strip()
p = urllib.parse.urlparse(maint)
open(sys.argv[2], "w").write(urllib.parse.urlunparse(
    ("postgres", p.netloc, "/" + sys.argv[3], "", "", "")))
EOF
chmod 600 "$RUN_DIR/owner.dsn"

echo "== run secrets"
python3 -c "
import secrets
d = '$RUN_DIR'
open(d + '/tokenA', 'w').write(secrets.token_hex(24))
open(d + '/tokenB', 'w').write(secrets.token_hex(24))
open(d + '/approle', 'w').write(secrets.token_hex(24))
open(d + '/orgsecret', 'w').write(secrets.token_hex(32))"
chmod 600 "$RUN_DIR"/tokenA "$RUN_DIR"/tokenB "$RUN_DIR"/approle "$RUN_DIR"/orgsecret

echo "== schema + operators"
cd "$REPO"
BOUNDARY_DSN_FILE="$RUN_DIR/owner.dsn" BOUNDARY_DIR="$RUN_DIR" \
	go test ./e2e/boundary_b/ -run TestProvisionRun -count=1 -v

echo "== provision CLI (operators file)"
go run ./cmd/studio provision --tenant t-demo --binding demo-op \
	--framework-credential svc-demo --login-token-file "$RUN_DIR/tokenA" \
	--operators-file "$RUN_DIR/ops.json" --dsn-file "$RUN_DIR/owner.dsn"
go run ./cmd/studio provision --tenant t-demo-b --binding demo-op-b \
	--framework-credential svc-demo-b --login-token-file "$RUN_DIR/tokenB" \
	--operators-file "$RUN_DIR/ops.json" --dsn-file "$RUN_DIR/owner.dsn"

echo "== policy files + app DSN"
python3 - <<EOF
import json
d = "$RUN_DIR"
json.dump({
  "svc-demo": {"agent": "demo-agent", "kind": "service", "id": "svc", "tenant": "t-demo"},
  "svc-demo-b": {"agent": "demo-agent-b", "kind": "service", "id": "svc-b", "tenant": "t-demo-b"},
}, open(d + "/keys.json", "w"))
json.dump({
  "t-demo": {"studio-runner": "autonomous", "studio": "autonomous"},
  "t-demo-b": {"studio-runner": "autonomous", "studio": "autonomous"},
}, open(d + "/grants.json", "w"))
EOF
chmod 600 "$RUN_DIR"/keys.json "$RUN_DIR"/grants.json
APW="$(cat "$RUN_DIR/approle")"
python3 -c "
import urllib.parse
dsn = open('$RUN_DIR/owner.dsn').read().strip()
p = urllib.parse.urlparse(dsn)
netloc = 'studio_app:$APW@' + p.hostname + ':' + str(p.port)
open('$RUN_DIR/app.dsn','w').write(urllib.parse.urlunparse(('postgres', netloc, p.path, '', '', '')))"
chmod 600 "$RUN_DIR/app.dsn"

echo "PROVISIONED $RUN_DIR $DB"
