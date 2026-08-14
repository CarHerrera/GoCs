#!/usr/bin/env bash
# Runs the Go server and the Vite client dev servers together.
# Ctrl+C stops both.

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
    echo
    echo "Stopping dev servers..."
    for pid in "${SERVER_PID:-}" "${CLIENT_PID:-}"; do
        [[ -z "$pid" ]] && continue
        pkill -P "$pid" 2>/dev/null
        kill -9 "$pid" 2>/dev/null
    done
    wait 2>/dev/null
}
trap cleanup EXIT

echo "Starting Go server (server/) on :4000 ..."
(cd "$ROOT_DIR/server" && go run .) &
SERVER_PID=$!

echo "Starting Vite client (client/) on :6969 ..."
(cd "$ROOT_DIR/client" && npm run dev) &
CLIENT_PID=$!

# Wait for whichever exits first (a crash or Ctrl+C), then the EXIT trap
# above cleans up the other one.
wait -n "$SERVER_PID" "$CLIENT_PID"
