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
omarchy_root=${OMARCHY_PATH:-"$HOME/.local/share/omarchy"}
qmllint -I "$omarchy_root/shell" \
  "$release_dir/omarchy-plugin/Overlay.qml" \
  "$release_dir/omarchy-plugin/Service.qml"
python3 -m py_compile "$release_dir/omarchy-plugin/transport/eyre_client.py" "$release_dir/bin/omabit"
python3 -m py_compile "$release_dir/scripts/check-tend-multiship.py"
bash -n "$release_dir/install.sh"
bash -n "$release_dir/scripts/check-tend-migrations.sh"
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

mounted_desk="$work_dir/mounted-tend"
mkdir -p "$mounted_desk/mar" "$work_dir/mounted-originals"
for mark in hoon kelvin noun txt; do
  printf '%s\n' "base-$mark" > "$mounted_desk/mar/$mark.hoon"
  cp "$mounted_desk/mar/$mark.hoon" "$work_dir/mounted-originals/$mark.hoon"
done
printf '%s\n' '[%zuse 408]' > "$mounted_desk/sys.kelvin"
cp "$mounted_desk/sys.kelvin" "$work_dir/mounted-originals/sys.kelvin"

desk_install_root="$work_dir/desk-install-root"
HOME="$desk_install_root/home" XDG_CONFIG_HOME="$desk_install_root/config" \
  "$release_dir/install.sh" \
  --desk-path "$mounted_desk" \
  --plugin-dir "$desk_install_root/plugin" \
  --bin-dir "$desk_install_root/bin"

for mark in hoon kelvin noun txt; do
  cmp "$work_dir/mounted-originals/$mark.hoon" "$mounted_desk/mar/$mark.hoon"
done
cmp "$work_dir/mounted-originals/sys.kelvin" "$mounted_desk/sys.kelvin"
cmp "$release_dir/desk/app/tend.hoon" "$mounted_desk/app/tend.hoon"
mounted_backup=$(find "$work_dir" -maxdepth 1 -type d -name 'mounted-tend.backup.*' -print -quit)
[[ -n "$mounted_backup" ]] || { echo "Mounted desk backup is missing" >&2; exit 1; }
cmp "$work_dir/mounted-originals/hoon.hoon" "$mounted_backup/mar/hoon.hoon"

printf '%s\n' 'locally-modified' > "$mounted_desk/app/tend.hoon"
conflict_install_root="$work_dir/conflict-install-root"
if HOME="$conflict_install_root/home" XDG_CONFIG_HOME="$conflict_install_root/config" \
  "$release_dir/install.sh" \
  --desk-path "$mounted_desk" \
  --plugin-dir "$conflict_install_root/plugin" \
  --bin-dir "$conflict_install_root/bin" >/dev/null 2>&1; then
  echo "Installer unexpectedly overwrote conflicting Tend source without --force" >&2
  exit 1
fi
[[ ! -e "$conflict_install_root/plugin" ]]
[[ ! -e "$conflict_install_root/bin/omabit" ]]

HOME="$desk_install_root/home" XDG_CONFIG_HOME="$desk_install_root/config" \
  "$release_dir/install.sh" --force \
  --desk-path "$mounted_desk" \
  --plugin-dir "$desk_install_root/plugin" \
  --bin-dir "$desk_install_root/bin"
cmp "$release_dir/desk/app/tend.hoon" "$mounted_desk/app/tend.hoon"
cmp "$work_dir/mounted-originals/sys.kelvin" "$mounted_desk/sys.kelvin"

broken_desk="$work_dir/broken-tend"
broken_install_root="$work_dir/broken-install-root"
mkdir -p "$broken_desk"
if HOME="$broken_install_root/home" XDG_CONFIG_HOME="$broken_install_root/config" \
  "$release_dir/install.sh" \
  --desk-path "$broken_desk" \
  --plugin-dir "$broken_install_root/plugin" \
  --bin-dir "$broken_install_root/bin" >/dev/null 2>&1; then
  echo "Installer unexpectedly accepted a mounted desk without its base files" >&2
  exit 1
fi
[[ ! -e "$broken_install_root/plugin" ]]
[[ ! -e "$broken_install_root/bin/omabit" ]]
echo "Tend release verification passed"
