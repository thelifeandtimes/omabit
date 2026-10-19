#!/bin/sh
# Review artifact: only the installed dedicated-account service executes this check.
set -eu
PATH=@SAFE_PATH@
LC_ALL=C
export PATH LC_ALL

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

# Native packages are authenticated by the OS-authorized installer. This boot
# guard checks local root trust, not publisher signatures or dpkg/MD5 metadata.
trusted() {
	[ ! -L "$1" ] || fail 'Unexpected symlink in trusted engine path.'
	[ "$(stat -c %u "$1")" = 0 ] || fail 'Engine paths must be root-owned.'
	permissions=$(stat -c %a "$1") || fail 'Cannot inspect trusted permissions.'
	[ "$((0$permissions & 022))" -eq 0 ] || fail 'Engine paths must not be group/world writable.'
}
for directory in / /usr /usr/bin /usr/local /usr/local/lib /usr/local/lib/urbit-runner /usr/local/lib/urbit-runner/rootless '@INSTALL_DIR@'; do
	[ -d "$directory" ] || fail 'Missing trusted engine directory.'
	trusted "$directory"
done
native_path() {
	native=/usr/bin/$1
	if [ ! -e "$native" ] && [ ! -L "$native" ]; then
		trusted /usr/sbin
		native=/usr/sbin/$1
	fi
	resolved=$native
	depth=0
	while [ -L "$resolved" ]; do
		[ "$depth" -lt 6 ] || fail 'Native executable alias loop.'
		depth=$((depth + 1))
		[ "$(stat -c %u "$resolved")" = 0 ] || fail 'Native aliases must be root-owned.'
		target=$(readlink "$resolved") || fail 'Cannot read native alias.'
		case "$target" in
			/*) ;;
			*[!a-zA-Z0-9_.-]*) fail 'Unreviewed relative native alias.' ;;
			*) target=${resolved%/*}/$target ;;
		esac
		case "$target" in
			/usr/bin/"$1") trusted /usr/bin ;;
			/usr/sbin/"$1") trusted /usr/sbin ;;
			/etc/alternatives/iptables) [ "$1" = iptables ] || fail 'Unreviewed native alias.'; trusted /etc; trusted /etc/alternatives ;;
			/usr/bin/iptables-nft|/usr/bin/iptables-legacy|/usr/bin/xtables-nft-multi|/usr/bin/xtables-legacy-multi) [ "$1" = iptables ] || fail 'Unreviewed native alias.'; trusted /usr/bin ;;
			/usr/sbin/iptables-nft|/usr/sbin/iptables-legacy|/usr/sbin/xtables-nft-multi|/usr/sbin/xtables-legacy-multi) [ "$1" = iptables ] || fail 'Unreviewed native alias.'; trusted /usr/sbin ;;
			*) fail 'Native alias leaves reviewed distribution paths.' ;;
		esac
		resolved=$target
	done
	[ -f "$resolved" ] && [ -x "$resolved" ] || fail 'Missing native executable.'
	trusted "$resolved"
}
for helper in docker rootlesskit slirp4netns newuidmap newgidmap iptables; do
	native_path "$helper"
done
native_path dockerd
dockerd=$native
version=$("$dockerd" --version) || fail 'Cannot read the Docker engine version.'
case "$version" in
	'Docker version '*', build '?*) version=${version#Docker version }; version=${version%%, build *} ;;
	*) fail 'Malformed Docker engine version.' ;;
esac
@ENGINE_VERSION_POLICY@
engine_version_supported "$version" || fail 'Docker engine is outside supported stable API compatibility families.'

launcher='@INSTALL_DIR@/dockerd-rootless.sh'
[ -f "$launcher" ] || fail 'Missing pinned launcher.'
trusted "$launcher"
printf '%s  %s\n' '@LAUNCHER_SHA256@' "$launcher" | sha256sum --check --status - || fail 'Launcher SHA-256 mismatch.'
