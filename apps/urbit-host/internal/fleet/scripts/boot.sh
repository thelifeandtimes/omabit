#!/bin/bash
set -euo pipefail
umask 077
while [[ ! -f /tmp/runner/go ]]; do sleep 0.1; done
kind=$1; identity=$2; http=$3; ames=$4; loom=$5; pill=$6
args=(--http-port "$http" -p "$ames" --loom "$loom")
# A held advisory lock prevents two manager containers opening this bind mount.
# Do not delete .vere.lock or silently replace an incomplete pier.
exec 9>/urbit/.manager.lock
if command -v flock >/dev/null; then flock -n 9 || { echo 'pier already locked' >&2; exit 73; }; else echo 'flock missing from runtime image' >&2; exit 69; fi
if [[ -d /urbit/pier/.urb ]]; then
  args+=(/urbit/pier)
elif [[ -e /urbit/pier ]]; then
  echo 'incomplete pier exists: inspect or archive it; refusing an implicit fresh boot' >&2; exit 65
else
  [[ -z "$pill" ]] || args+=(-u "$pill")
  case "$kind" in
    fake) args+=(-F "$identity" /urbit/pier) ;;
    comet) args+=(-c /urbit/pier) ;;
    keyed) [[ -s /boot-secret/network.key ]] || { echo 'networking key unavailable; resubmit with a fresh instance' >&2; exit 66; }; args+=(-w "$identity" -k /boot-secret/network.key -c /urbit/pier) ;;
    *) exit 64 ;;
  esac
fi
printf '%s\n' "$$" > /tmp/runner/vere.pid
exec urbit "${args[@]}"
