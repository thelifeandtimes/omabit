#!/usr/bin/env bash
# Installs only into this user's home; no sudo, Docker changes or Tailscale setup.
set -euo pipefail
umask 077
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
host_name=''
if (($#)); then
  [[ $# == 2 && $1 == --host ]] || { echo 'usage: scripts/install.sh [--host NAME]' >&2; exit 2; }
  host_name=$2
fi
case $(uname -m) in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; *) echo 'Only Linux amd64/arm64 are packaged' >&2; exit 2;; esac
[[ $(uname -s) == Linux ]] || { echo 'Linux required' >&2; exit 2; }
mkdir -p "$HOME/.local/bin"
staging=$(mktemp "$HOME/.local/bin/.urbitctl.XXXXXXXX")
trap 'rm -f "$staging"' EXIT
if [[ -f "$root/dist/urbitctl-linux-$arch" ]]; then
  (cd "$root" && sha256sum --check --ignore-missing SHA256SUMS)
  cp "$root/dist/urbitctl-linux-$arch" "$staging"
else
  command -v go >/dev/null || { echo 'Go 1.23 or newer is required to build source' >&2; exit 2; }
  (cd "$root" && CGO_ENABLED=0 go build -trimpath -o "$staging" ./cmd/urbitctl)
fi
chmod 0755 "$staging"
mv -f "$staging" "$HOME/.local/bin/urbitctl"
data=${XDG_DATA_HOME:-$HOME/.local/share}/omarchy-urbit
mkdir -p "$data/docs" "$data/skills"
cp -R "$root/docs/." "$data/docs/"
cp -R "$root/skills/." "$data/skills/"
"$HOME/.local/bin/urbitctl" version
if [[ -n "$host_name" ]]; then
  command -v docker >/dev/null || { echo 'Docker is not installed; configure it separately, then rerun this installer.' >&2; exit 2; }
  docker info >/dev/null || { echo 'This user cannot access the Docker daemon. No permissions were changed.' >&2; exit 2; }
  config=${XDG_CONFIG_HOME:-$HOME/.config}/omarchy-urbit/host.json
  if [[ ! -f "$config" ]]; then "$HOME/.local/bin/urbitctl" host init --name "$host_name"; fi
  # %h in systemd intentionally follows HOME, not an arbitrary executable path.
  units=${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user
  mkdir -p "$units"
  cp "$root/packaging/omarchy-urbit.service" "$units/omarchy-urbit.service"
  systemctl --user daemon-reload
  systemctl --user enable --now omarchy-urbit.service
  ready=false
  for _ in {1..50}; do
    if report=$("$HOME/.local/bin/urbitctl" --host local doctor); then printf '%s\n' "$report"; ready=true; break; fi
    sleep 0.1
  done
  [[ "$ready" == true ]] || { echo 'Service did not become available; inspect journalctl --user -u omarchy-urbit.service.' >&2; exit 2; }
  printf '\nFor workstation operation after logout, inspect: loginctl show-user %q -p Linger\n' "$USER"
  printf 'Enable lingering deliberately when appropriate: loginctl enable-linger %q\n' "$USER"
fi
printf '\nCLI installed at %s/.local/bin/urbitctl\n' "$HOME"
