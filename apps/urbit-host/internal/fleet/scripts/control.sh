#!/bin/bash
# Read the exact pier's loopback port, never guess 12321 in a shared namespace.
set -euo pipefail
ports=/urbit/pier/.http.ports
[[ -f "$ports" ]] || { echo 'runtime HTTP port file not available' >&2; exit 69; }
port=$(awk '$2 == "insecure" && $3 == "loopback" && $1 ~ /^[0-9]+$/ {print $1; exit}' "$ports")
[[ "$port" =~ ^[0-9]+$ ]] && ((port > 0 && port <= 65535)) || { echo 'no loopback port discovered; inspect runtime compatibility' >&2; exit 69; }
# Input/output are private to docker exec, not the container console log.
exec curl --silent --show-error --fail --max-time 20 --noproxy '*' -H 'Content-Type: application/json' --data-binary @- "http://127.0.0.1:$port"
