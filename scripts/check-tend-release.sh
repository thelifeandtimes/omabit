#!/usr/bin/env bash
set -euo pipefail

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work_dir=$(mktemp -d)
trap 'chmod -R u+rwX "$work_dir"; rm -rf "$work_dir"' EXIT
mkdir -p "$work_dir/build-a" "$work_dir/build-b"
archive=$($repo_dir/scripts/build-tend-release.sh "$work_dir/build-a")
second_archive=$($repo_dir/scripts/build-tend-release.sh "$work_dir/build-b")
cmp "$archive" "$second_archive"
checksum_file="$archive.sha256"
checksum_entry=$(cut -d ' ' -f 3- "$checksum_file")
[[ "$checksum_entry" == "$(basename "$archive")" ]] || {
  echo "Release checksum must name only the portable archive basename" >&2
  exit 1
}
(
  cd "$(dirname "$archive")"
  sha256sum --check "$(basename "$checksum_file")"
)
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
  "$release_dir/omarchy-plugin/TendPanel.qml" \
  "$release_dir/omarchy-plugin/BarWidget.qml" \
  "$release_dir/omarchy-plugin/ReminderDetail.qml" \
  "$release_dir/omarchy-plugin/TendButton.qml" \
  "$release_dir/omarchy-plugin/TendCheck.qml" \
  "$release_dir/omarchy-plugin/TendDropdown.qml" \
  "$release_dir/omarchy-plugin/TendNumber.qml" \
  "$release_dir/omarchy-plugin/Service.qml"
python3 -m py_compile "$release_dir/omarchy-plugin/transport/eyre_client.py" "$release_dir/bin/omabit"
python3 -m py_compile "$release_dir/scripts/check-tend-multiship.py"
bash -n "$release_dir/install.sh"
bash -n "$release_dir/scripts/check-tend-migrations.sh"
[[ "$(cat "$release_dir/VERSION")" == "$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$release_dir/omarchy-plugin/manifest.json")" ]]
"$release_dir/bin/omabit" --version | grep -q "$(cat "$release_dir/VERSION")"

for desk_dependency in \
  desk.docket-0 \
  app/tend.html \
  mar/bill.hoon \
  mar/docket-0.hoon \
  mar/html.hoon \
  mar/mime.hoon \
  mar/json.hoon \
  lib/default-agent.hoon \
  lib/docket.hoon \
  lib/server.hoon \
  lib/skeleton.hoon \
  lib/strand.hoon \
  sur/docket.hoon \
  sur/spider.hoon
do
  [[ -f "$release_dir/desk/$desk_dependency" ]] || {
    echo "Release desk is missing required Urbit dependency: $desk_dependency" >&2
    exit 1
  }
done
if grep -q 'strandio' "$release_dir/desk/ted/tend-migrations.hoon"; then
  echo "Migration thread must not vendor userspace-specific strandio" >&2
  exit 1
fi

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
plugin_backup_root="$install_root/config/omarchy/plugin-backups"
find "$plugin_backup_root" -maxdepth 1 -type d -name 'plugin.backup.*' -print -quit | grep -q .
if find "$(dirname "$install_root/plugin")" -maxdepth 1 -type d -name 'plugin.backup.*' -print -quit | grep -q .; then
  echo "Installer left a plugin backup beside the live plugin target" >&2
  exit 1
fi
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
cmp "$release_dir/desk/app/tend.html" "$mounted_desk/app/tend.html"
cmp "$release_dir/desk/desk.docket-0" "$mounted_desk/desk.docket-0"
cmp "$release_dir/desk/mar/bill.hoon" "$mounted_desk/mar/bill.hoon"
cmp "$release_dir/desk/mar/docket-0.hoon" "$mounted_desk/mar/docket-0.hoon"
cmp "$release_dir/desk/mar/html.hoon" "$mounted_desk/mar/html.hoon"
cmp "$release_dir/desk/mar/mime.hoon" "$mounted_desk/mar/mime.hoon"
cmp "$release_dir/desk/mar/json.hoon" "$mounted_desk/mar/json.hoon"
cmp "$release_dir/desk/lib/default-agent.hoon" "$mounted_desk/lib/default-agent.hoon"
cmp "$release_dir/desk/lib/docket.hoon" "$mounted_desk/lib/docket.hoon"
cmp "$release_dir/desk/lib/server.hoon" "$mounted_desk/lib/server.hoon"
cmp "$release_dir/desk/sur/docket.hoon" "$mounted_desk/sur/docket.hoon"
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
