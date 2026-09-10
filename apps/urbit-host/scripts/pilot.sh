#!/usr/bin/env bash
# Run on laptop AFTER CLI/host installation and an SSH host alias are working.
# Creates ONE disposable fake zod. Never reads a real ship's networking key.
set -euo pipefail
host=${1:-workstation}
command -v jq >/dev/null || { echo 'jq is required by this pilot script' >&2; exit 2; }
cli=(urbitctl --host "$host")
"${cli[@]}" doctor
stamp=$(date +%s)
group=pilot-$stamp
label=pilot-zod-$stamp
"${cli[@]}" dev group create "$group"
create=$(urbitctl --host "$host" --request-id "pilot-create-$stamp" dev create --group "$group" --fake zod --label "$label")
printf '%s\n' "$create"
op=$(jq -er '.result.id' <<<"$create")
ship=$(jq -er '.result.instance_id' <<<"$create")
"${cli[@]}" operation wait "$op"
"${cli[@]}" ship inspect "$ship"
printf '\nCreated disposable instance %s in group %s.\n' "$ship" "$group"
printf 'Operation completion is not a claim that Urbit has finished booting.\n'
printf 'Inspect: urbitctl --host %q ship inspect %q\n' "$host" "$ship"
printf 'Logs:    urbitctl --host %q ship logs %q\n' "$host" "$ship"
printf 'Browser: urbitctl --host %q ship open %q\n' "$host" "$ship"
printf 'Code:    urbitctl --host %q ship code --clipboard %q\n' "$host" "$ship"
printf 'Dojo:    urbitctl --host %q ship dojo %q\n' "$host" "$ship"
printf '\nNo archive or purge is performed by this script. Follow docs/PILOT.md.\n'
