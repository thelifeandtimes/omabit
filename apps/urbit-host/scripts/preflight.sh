#!/usr/bin/env bash
# Read-only diagnostics. No environment dump, public IP lookup or credentials.
set -uo pipefail
printf '== Linux / CLI ==\n'
uname -srm
command -v urbitctl >/dev/null && urbitctl version
printf '\n== Required executables ==\n'
for tool in docker ssh tailscale systemctl jq wl-copy xdg-open; do
  if command -v "$tool" >/dev/null; then printf '%s: present\n' "$tool"; else printf '%s: missing\n' "$tool"; fi
done
printf '\n== Docker access ==\n'
docker info --format 'server={{.ServerVersion}} architecture={{.Architecture}} driver={{.Driver}}' 2>&1
printf '\n== Tailscale ==\n'
tailscale version 2>&1 | head -n 1
# We deliberately do not print the tailnet's device/user inventory.
if command -v jq >/dev/null && command -v tailscale >/dev/null; then
  tailscale status --json 2>/dev/null | jq '{BackendState,SelfOnline:.Self.Online}'
fi
printf '\n== User service ==\n'
systemctl --user is-active omarchy-urbit.service 2>&1
loginctl show-user "$USER" -p Linger 2>&1
