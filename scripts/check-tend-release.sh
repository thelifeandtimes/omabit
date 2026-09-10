#!/usr/bin/env bash
set -euo pipefail

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work_dir=$(mktemp -d)
trap 'chmod -R u+rwX "$work_dir"; rm -rf "$work_dir"' EXIT
mkdir -p "$work_dir/build-a" "$work_dir/build-b"
archive=$($repo_dir/scripts/build-tend-release.sh "$work_dir/build-a")
second_archive=$($repo_dir/scripts/build-tend-release.sh "$work_dir/build-b")
cmp "$archive" "$second_archive"
sha256sum --check "$archive.sha256"
mkdir -p "$work_dir/extracted"
tar -xzf "$archive" -C "$work_dir/extracted"
release_dir=$(find "$work_dir/extracted" -mindepth 1 -maxdepth 1 -type d -name 'omabit-tend-*' -print -quit)
[[ -n "$release_dir" ]] || { echo "Release root is missing" >&2; exit 1; }
(
  cd "$release_dir"
  sha256sum --check SHA256SUMS
)

omarchy plugin validate "$release_dir/omarchy-plugin"
python3 -m py_compile "$release_dir/omarchy-plugin/transport/eyre_client.py" "$release_dir/bin/omabit"
bash -n "$release_dir/install.sh"
[[ "$(cat "$release_dir/VERSION")" == "$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$release_dir/omarchy-plugin/manifest.json")" ]]
"$release_dir/bin/omabit" --version | grep -q "$(cat "$release_dir/VERSION")"

install_root="$work_dir/install-root"
HOME="$install_root/home" XDG_CONFIG_HOME="$install_root/config" \
  "$release_dir/install.sh" --plugin-dir "$install_root/plugin" --bin-dir "$install_root/bin"
cmp "$release_dir/omarchy-plugin/manifest.json" "$install_root/plugin/manifest.json"
cmp "$release_dir/bin/omabit" "$install_root/bin/omabit"

if "$release_dir/install.sh" --plugin-dir "$install_root/plugin" --bin-dir "$install_root/bin" >/dev/null 2>&1; then
  echo "Installer unexpectedly overwrote an existing installation without --force" >&2
  exit 1
fi

HOME="$install_root/home" XDG_CONFIG_HOME="$install_root/config" \
  "$release_dir/install.sh" --force --plugin-dir "$install_root/plugin" --bin-dir "$install_root/bin"
find "$install_root" -maxdepth 1 -type d -name 'plugin.backup.*' -print -quit | grep -q .
find "$install_root/bin" -maxdepth 1 -type f -name 'omabit.backup.*' -print -quit | grep -q .
echo "Tend release verification passed"
