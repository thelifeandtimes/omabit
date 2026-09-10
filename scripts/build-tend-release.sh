#!/usr/bin/env bash
set -euo pipefail

repo_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
version=$(tr -d '[:space:]' < "$repo_dir/VERSION")
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "VERSION must be semantic x.y.z" >&2; exit 1; }

output_dir=${1:-"$repo_dir/dist"}
mkdir -p "$output_dir"
work_dir=$(mktemp -d)
trap 'chmod -R u+rwX "$work_dir"; rm -rf "$work_dir"' EXIT
release_name="omabit-tend-$version"
release_dir="$work_dir/$release_name"
mkdir -p "$release_dir/bin" "$release_dir/docs"

cp -a "$repo_dir/desk" "$release_dir/desk"
cp -a "$repo_dir/omarchy-plugin" "$release_dir/omarchy-plugin"
find "$release_dir" -type d -name __pycache__ -prune -exec rm -rf {} +
find "$release_dir" -type f -name '*.pyc' -delete
cp "$repo_dir/bin/omabit" "$release_dir/bin/omabit"
cp "$repo_dir/packaging/tend/install.sh" "$release_dir/install.sh"
cp "$repo_dir/VERSION" "$release_dir/VERSION"
cp "$repo_dir/docs/TEND_INSTALL.md" "$release_dir/README.md"
cp "$repo_dir/docs/TEND_INSTALL.md" "$release_dir/docs/TEND_INSTALL.md"
cp "$repo_dir/docs/TEND_OPERATIONS.md" "$release_dir/docs/TEND_OPERATIONS.md"
cp "$repo_dir/docs/TEND_PROTOCOL.md" "$release_dir/docs/TEND_PROTOCOL.md"
cp "$repo_dir/docs/TEND_SECURITY.md" "$release_dir/docs/TEND_SECURITY.md"
cp "$repo_dir/docs/TEND_ACCESSIBILITY.md" "$release_dir/docs/TEND_ACCESSIBILITY.md"
cp "$repo_dir/docs/TEND_BACKUP.md" "$release_dir/docs/TEND_BACKUP.md"
cp "$repo_dir/docs/TEND_PERFORMANCE.md" "$release_dir/docs/TEND_PERFORMANCE.md"
cp "$repo_dir/docs/TEND_RELEASE_CHECKLIST.md" "$release_dir/docs/TEND_RELEASE_CHECKLIST.md"
chmod 0755 "$release_dir/bin/omabit" "$release_dir/install.sh"

(
  cd "$release_dir"
  find . -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 sha256sum > SHA256SUMS
)

archive="$output_dir/$release_name.tar.gz"
tar --sort=name --mtime='UTC 1970-01-01' --owner=0 --group=0 --numeric-owner -czf "$archive" -C "$work_dir" "$release_name"
sha256sum "$archive" > "$archive.sha256"
printf '%s\n' "$archive"
