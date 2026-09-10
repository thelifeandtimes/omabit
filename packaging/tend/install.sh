#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: install.sh [--enable] [--force] [--desk-path PATH] [--plugin-dir PATH] [--bin-dir PATH]

Installs the Tend Omarchy plugin and the omabit CLI into user-owned paths.
When --desk-path is supplied, it also installs the %tend desk into that exact
mounted-desk directory. Existing targets are refused unless --force is used;
forced replacements are moved to timestamped backups first.
EOF
}

enable_plugin=false
force=false
desk_path=""
plugin_dir="${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/io.omabit.tend"
bin_dir="$HOME/.local/bin"

while (($#)); do
  case "$1" in
    --enable) enable_plugin=true ;;
    --force) force=true ;;
    --desk-path)
      shift
      [[ $# -gt 0 ]] || { usage >&2; exit 2; }
      desk_path="$1"
      ;;
    --plugin-dir)
      shift
      [[ $# -gt 0 ]] || { usage >&2; exit 2; }
      plugin_dir="$1"
      ;;
    --bin-dir)
      shift
      [[ $# -gt 0 ]] || { usage >&2; exit 2; }
      bin_dir="$1"
      ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

release_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
timestamp=$(date -u +%Y%m%dT%H%M%SZ)

install_tree() {
  local source_dir=$1
  local target_dir=$2
  local label=$3
  local parent_dir
  parent_dir=$(dirname "$target_dir")

  [[ ! -L "$target_dir" ]] || { echo "Refusing symbolic-link $label target: $target_dir" >&2; exit 1; }
  mkdir -p "$parent_dir"
  if [[ -e "$target_dir" ]]; then
    if [[ "$force" != true ]]; then
      echo "$label already exists at $target_dir; rerun with --force to back it up and replace it" >&2
      exit 1
    fi
    mv "$target_dir" "$target_dir.backup.$timestamp"
    echo "Backed up $label to $target_dir.backup.$timestamp"
  fi

  local staging_dir
  staging_dir=$(mktemp -d "$parent_dir/.tend-install.XXXXXX")
  cp -a "$source_dir/." "$staging_dir/"
  mv "$staging_dir" "$target_dir"
  echo "Installed $label at $target_dir"
}

install_tree "$release_dir/omarchy-plugin" "$plugin_dir" "Omarchy plugin"

mkdir -p "$bin_dir"
cli_target="$bin_dir/omabit"
[[ ! -L "$cli_target" ]] || { echo "Refusing symbolic-link CLI target: $cli_target" >&2; exit 1; }
if [[ -e "$cli_target" ]]; then
  if [[ "$force" != true ]]; then
    echo "CLI already exists at $cli_target; rerun with --force to back it up and replace it" >&2
    exit 1
  fi
  mv "$cli_target" "$cli_target.backup.$timestamp"
  echo "Backed up CLI to $cli_target.backup.$timestamp"
fi
install -m 0755 "$release_dir/bin/omabit" "$cli_target"
echo "Installed CLI at $cli_target"

if [[ -n "$desk_path" ]]; then
  case "$desk_path" in
    /|"$HOME"|"$HOME/") echo "Refusing unsafe desk target: $desk_path" >&2; exit 1 ;;
  esac
  install_tree "$release_dir/desk" "$desk_path" "%tend desk"
  echo "Commit and start the mounted desk from dojo: |commit %tend  then  |rein %tend [& %tend]"
fi

if [[ "$enable_plugin" == true ]]; then
  if command -v omarchy-shell >/dev/null 2>&1; then
    omarchy-shell -q shell rescanPlugins
  fi
  if command -v omarchy >/dev/null 2>&1; then
    omarchy plugin enable io.omabit.tend
  else
    echo "Omarchy CLI was not found; enable io.omabit.tend after Omarchy is installed" >&2
  fi
else
  echo "Review the unsandboxed plugin, then enable it with: omarchy plugin enable io.omabit.tend"
fi

if [[ -z "$desk_path" ]]; then
  echo "The desk was not installed. See TEND_INSTALL.md for mounted-desk instructions."
fi
