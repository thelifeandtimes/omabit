#!/usr/bin/bash
# Source-checkout desktop install. Host changes happen only in the setup flow.
set -euo pipefail
if [[ $EUID == 0 || $# != 0 ]]; then
  printf '%s\n' 'Run this installer as your desktop user, with no arguments.' >&2
  exit 2
fi
ROOT="$(dirname "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")")"
for tool in go python3 omarchy; do
  if ! command -v "$tool" >/dev/null; then
    printf 'Missing %s. Install Go 1.25+, Python and the Omarchy shell before continuing.\n' "$tool" >&2
    exit 1
  fi
done
printf '%s\n' 'Building Omabit, the collector and the official-runtime runner…'
mkdir -p "$ROOT/build"
go -C "$ROOT" build -o "$ROOT/build/" ./cmd/...
python3 -B "$ROOT/scripts/install-panel.py" --replace --enable
printf '\n%s\n' 'Open the ~+ menu → Setup → Set up this computer.'
printf 'Terminal setup is also available: %q --format text setup\n' "$ROOT/build/omabit"
