#!/bin/sh
# Review artifact: only the installed dedicated-account service executes this check.
set -eu
PATH=/usr/bin:/bin
export PATH

fail() {
	printf '%s\n' "$1" >&2
	exit 1
}

[ "$(id -u)" != 0 ] || fail 'The engine must not run as root.'
[ "$(id -un)" = '@ACCOUNT@' ] || fail 'The engine requires the dedicated @ACCOUNT@ account.'
[ "${HOME:-}" = '@HOME@' ] || fail 'Unexpected dedicated-account home.'
account=$(getent passwd '@ACCOUNT@') || fail 'Cannot resolve the dedicated account.'
IFS=: read -r name password uid gid gecos home shell <<EOF
$account
EOF
[ "$home" = '@HOME@' ] || fail 'Unexpected passwd home for the dedicated account.'

[ -f /usr/bin/dockerd ] && [ -x /usr/bin/dockerd ] || fail 'Missing /usr/bin/dockerd.'
[ "$(stat -Lc %u /usr/bin/dockerd)" = 0 ] || fail '/usr/bin/dockerd must be root-owned.'
permissions=$(stat -Lc %a /usr/bin/dockerd) || fail 'Cannot inspect dockerd permissions.'
[ "$((0$permissions & 022))" -eq 0 ] || fail '/usr/bin/dockerd must not be group/world writable.'
version=$(/usr/bin/dockerd --version) || fail 'Cannot read the Docker engine version.'
case "$version" in
	'Docker version @ENGINE_VERSION@, build '?*) ;;
	*) fail 'Expected exactly Docker version @ENGINE_VERSION@.' ;;
esac

launcher='@INSTALL_DIR@/dockerd-rootless.sh'
[ -f "$launcher" ] || fail 'Missing pinned launcher.'
[ "$(stat -Lc %u "$launcher")" = 0 ] || fail 'The installed launcher must be root-owned.'
printf '%s  %s\n' '@LAUNCHER_SHA256@' "$launcher" | sha256sum --check --status - || fail 'Launcher SHA-256 mismatch.'
