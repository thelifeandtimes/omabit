#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: install.sh [--enable] [--force] [--desk-path PATH] [--plugin-dir PATH] [--bin-dir PATH]

Installs the Tend Omarchy plugin and the omabit CLI into user-owned paths.
When --desk-path is supplied, it overlays Tend source into that exact mounted
desk while preserving its base marks and sys.kelvin. Existing desktop targets
and conflicting Tend source are refused unless --force is used. Replacements
are moved to timestamped backups first.
EOF
}

enable_plugin=false
force=false
desk_path=""
plugin_dir="${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/io.omabit.tend"
plugin_backup_dir="${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugin-backups"
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
active_staging_dir=""

cleanup() {
  if [[ -n "$active_staging_dir" && -d "$active_staging_dir" ]]; then
    chmod -R u+rwX "$active_staging_dir" 2>/dev/null || true
    rm -rf -- "$active_staging_dir"
  fi
}
trap cleanup EXIT

next_backup_path() {
  local target=$1
  local candidate="$target.backup.$timestamp"
  local suffix=1

  while [[ -e "$candidate" || -L "$candidate" ]]; do
    candidate="$target.backup.$timestamp.$suffix"
    ((suffix += 1))
  done
  printf '%s\n' "$candidate"
}

preflight_replace_target() {
  local target=$1
  local label=$2

  [[ ! -L "$target" ]] || { echo "Refusing symbolic-link $label target: $target" >&2; exit 1; }
  if [[ -e "$target" && "$force" != true ]]; then
    echo "$label already exists at $target; rerun with --force to back it up and replace it" >&2
    exit 1
  fi
}

preflight_desk_overlay() {
  local source_dir=$1
  local target_dir=$2
  local required_path
  local source_file
  local relative_path
  local conflict=""
  local symlink_path=""

  case "$target_dir" in
    /*) ;;
    *) echo "The mounted %tend desk path must be absolute: $target_dir" >&2; exit 1 ;;
  esac
  case "$target_dir" in
    /|"$HOME"|"$HOME/") echo "Refusing unsafe desk target: $target_dir" >&2; exit 1 ;;
  esac

  [[ ! -L "$target_dir" ]] || { echo "Refusing symbolic-link %tend desk target: $target_dir" >&2; exit 1; }
  [[ -d "$target_dir" ]] || {
    echo "Mounted %tend desk not found at $target_dir; run |new-desk %tend and |mount %tend first" >&2
    exit 1
  }

  symlink_path=$(find "$target_dir" -type l -print -quit)
  [[ -z "$symlink_path" ]] || {
    echo "Refusing mounted %tend desk containing a symbolic link: $symlink_path" >&2
    exit 1
  }

  for required_path in \
    mar/hoon.hoon \
    mar/kelvin.hoon \
    mar/noun.hoon \
    mar/txt.hoon \
    sys.kelvin
  do
    [[ -f "$target_dir/$required_path" ]] || {
      echo "Mounted %tend desk is missing base file $required_path at $target_dir" >&2
      echo "Restore the mounted desk from its backup, or recreate it with |new-desk %tend and |mount %tend" >&2
      exit 1
    }
  done

  while IFS= read -r -d '' source_file; do
    relative_path=${source_file#"$source_dir/"}
    [[ "$relative_path" != "sys.kelvin" ]] || continue
    if [[ -e "$target_dir/$relative_path" ]]; then
      [[ -f "$target_dir/$relative_path" ]] || {
        echo "Refusing non-file Tend source target: $target_dir/$relative_path" >&2
        exit 1
      }
      if ! cmp -s "$source_file" "$target_dir/$relative_path"; then
        conflict=$relative_path
        break
      fi
    fi
  done < <(find "$source_dir" -type f -print0)

  if [[ -n "$conflict" && "$force" != true ]]; then
    echo "Mounted %tend desk has different Tend source at $conflict; rerun with --force to back up the desk and update it" >&2
    exit 1
  fi
}

install_tree() {
  local source_dir=$1
  local target_dir=$2
  local label=$3
  local backup_parent=${4:-}
  local parent_dir
  parent_dir=$(dirname "$target_dir")

  mkdir -p "$parent_dir"
  if [[ -e "$target_dir" ]]; then
    local backup_dir
    if [[ -n "$backup_parent" ]]; then
      [[ ! -L "$backup_parent" ]] || { echo "Refusing symbolic-link backup directory: $backup_parent" >&2; exit 1; }
      mkdir -p "$backup_parent"
      backup_dir=$(next_backup_path "$backup_parent/$(basename "$target_dir")")
    else
      backup_dir=$(next_backup_path "$target_dir")
    fi
    mv "$target_dir" "$backup_dir"
    echo "Backed up $label to $backup_dir"
  fi

  active_staging_dir=$(mktemp -d "$parent_dir/.tend-install.XXXXXX")
  cp -a "$source_dir/." "$active_staging_dir/"
  mv "$active_staging_dir" "$target_dir"
  active_staging_dir=""
  echo "Installed $label at $target_dir"
}

install_desk_overlay() {
  local source_dir=$1
  local target_dir=$2
  local parent_dir
  local source_file
  local relative_path
  local backup_dir
  parent_dir=$(dirname "$target_dir")
  backup_dir=$(next_backup_path "$target_dir")

  active_staging_dir=$(mktemp -d "$parent_dir/.tend-desk-install.XXXXXX")
  cp -a "$target_dir/." "$active_staging_dir/"
  while IFS= read -r -d '' source_file; do
    relative_path=${source_file#"$source_dir/"}
    [[ "$relative_path" != "sys.kelvin" ]] || continue
    mkdir -p "$(dirname "$active_staging_dir/$relative_path")"
    cp -a "$source_file" "$active_staging_dir/$relative_path"
  done < <(find "$source_dir" -type f -print0)

  mv "$target_dir" "$backup_dir"
  if ! mv "$active_staging_dir" "$target_dir"; then
    mv "$backup_dir" "$target_dir"
    echo "Failed to install %tend source; restored the original mounted desk" >&2
    exit 1
  fi
  active_staging_dir=""
  echo "Backed up mounted %tend desk to $backup_dir"
  echo "Installed Tend source at $target_dir (preserved mounted base marks and sys.kelvin)"
}

cli_target="$bin_dir/omabit"
preflight_replace_target "$plugin_dir" "Omarchy plugin"
preflight_replace_target "$cli_target" "CLI"
if [[ -n "$desk_path" ]]; then
  preflight_desk_overlay "$release_dir/desk" "$desk_path"
fi

if [[ -n "$desk_path" ]]; then
  install_desk_overlay "$release_dir/desk" "$desk_path"
  echo "Commit and install the mounted desk from dojo: |commit %tend  then  |install our %tend"
  echo "If %tend already exists but is suspended, run: |revive %tend"
fi

install_tree "$release_dir/omarchy-plugin" "$plugin_dir" "Omarchy plugin" "$plugin_backup_dir"

mkdir -p "$bin_dir"
if [[ -e "$cli_target" ]]; then
  cli_backup=$(next_backup_path "$cli_target")
  mv "$cli_target" "$cli_backup"
  echo "Backed up CLI to $cli_backup"
fi
install -m 0755 "$release_dir/bin/omabit" "$cli_target"
echo "Installed CLI at $cli_target"

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
