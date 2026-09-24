#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 PIER" >&2
  exit 64
fi

pier=$1
socket="$pier/.urb/conn.sock"
[[ -S "$socket" ]] || { echo "Tend migration gate requires a running pier: $socket" >&2; exit 66; }
command -v urbit >/dev/null || { echo "urbit is required" >&2; exit 69; }
command -v nc >/dev/null || { echo "nc is required" >&2; exit 69; }

request='[0 %fyrd %tend %tend-migrations %noun %noun ~]'
expected='[0 %avow 0 %noun %tend-migrations 11 0]'
result=$(printf '%s\n' "$request" \
  | urbit eval -jn 2>/dev/null \
  | nc -U -W 30 "$socket" \
  | urbit eval -cn 2>/dev/null)

if [[ "$result" != "$expected" ]]; then
  echo "Tend migration gate failed: $result" >&2
  exit 1
fi

echo "Tend saved-state migrations %0 through %11 passed"
