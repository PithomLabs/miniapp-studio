#!/bin/sh
# Integration-test runner entrypoint (Option 2). Builds the control plane
# from the mounted repo with the mounted toolchain, starts the loopback
# relay, then runs one mode: netcheck | run. Fails fast and loud; the
# container exit code is the verdict (0 only when every gate passed).
set -eu

REPO="${REPO:-/repo}"
OUT="${RESULTS:-/results}"
PG_TARGET="${PG_TARGET:-pg-test:5432}"

# Rotation guard: the e2e provisioner rotates credentials when these are
# set. They must never be set in a test runner.
if [ -n "${BOUNDARY_DSN_FILE:-}${BOUNDARY_DIR:-}" ]; then
	echo "refusing: BOUNDARY_* is set (credential-rotation risk)" >&2
	exit 2
fi
if [ -z "${TEST_DATABASE_URL:-}" ]; then
	echo "refusing: TEST_DATABASE_URL is unset" >&2
	exit 2
fi

# Toolchain comes from read-only mounts at broker-passed paths (never
# hardcoded: the same entrypoint runs on any machine the broker
# configures). Missing or dangling paths fail loudly, never silently.
: "${GOROOT:?broker must pass GOROOT}"
: "${GOMODCACHE:?broker must pass GOMODCACHE}"
[ -d "$GOROOT" ] || { echo "missing GOROOT dir: $GOROOT" >&2; exit 2; }
[ -d "$GOMODCACHE" ] || { echo "missing GOMODCACHE dir: $GOMODCACHE" >&2; exit 2; }
export PATH="$GOROOT/bin:${PATH}"
export GOCACHE="${GOCACHE:-/tmp/gocache}"
export GOPROXY=off
export GOFLAGS=-buildvcs=false
mkdir -p "$GOCACHE" "$OUT"
cd "$REPO"

echo "building control plane from $REPO/testrunner"
go build -o /tmp/testrunner ./testrunner

echo "starting loopback relay → $PG_TARGET"
/tmp/testrunner relay -listen 127.0.0.1:5434 -target "$PG_TARGET" >"$OUT/relay.log" 2>&1 &
RELAY_PID=$!
trap 'kill $RELAY_PID 2>/dev/null' EXIT INT TERM
/tmp/testrunner gate -addr 127.0.0.1:5434 -timeout 60s

mode="${1:-run}"
if [ "$mode" = "netcheck" ]; then
	exec /tmp/testrunner netcheck -relay 127.0.0.1:5434 -pg "$PG_TARGET"
fi
if [ "$mode" != "run" ]; then
	echo "unknown mode $mode (want run|netcheck)" >&2
	exit 2
fi

# Run mode mirrors the CI gates; first failure ends the run with the
# failing gate's evidence preserved.
fail() {
	echo "$1" >"$OUT/exitcode"
	echo "FAILED: $2 (see $OUT)"
	exit "$1"
}
echo "== gofmt" | tee "$OUT/summary.txt"
out=$(gofmt -l .) || fail 3 "gofmt execution failed"
[ -z "$out" ] || { echo "$out" >"$OUT/gofmt-dirty.txt"; fail 3 "gofmt dirty: $out"; }
echo "== vet" | tee -a "$OUT/summary.txt"
go vet ./... >"$OUT/vet.log" 2>&1 || fail 4 "go vet failed"
echo "== build" | tee -a "$OUT/summary.txt"
go build ./... >"$OUT/build.log" 2>&1 || fail 5 "go build failed"
echo "== test" | tee -a "$OUT/summary.txt"
set +e
go test -p 1 -count=1 -json ./... >"$OUT/gotest.json" 2>&1
code=$?
set -e
grep -E '"Action":"(pass|fail|skip)"' "$OUT/gotest.json" | grep -v '"Test":' >>"$OUT/summary.txt" || true
echo "$code" >"$OUT/exitcode"
if [ "$code" -ne 0 ]; then
	echo "FAILED: suite exit $code"
	exit "$code"
fi
echo "ALL GATES GREEN"
