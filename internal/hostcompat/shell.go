package hostcompat

import "fmt"

// EngineVersionShell returns the POSIX-shell version predicate used by reviewed
// boot guards. It embeds only policy constants, never caller text, and has no
// host side effects. API, containment and provenance checks remain independent.
func EngineVersionShell() string {
	return fmt.Sprintf(`engine_version_supported() (
	[ "$#" -eq 1 ] || return 1
	[ "${#1}" -le %d ] || return 1
	engine_version=$1
	case "$engine_version" in
		*+*) engine_build=${engine_version#*+}; engine_version=${engine_version%%%%+*}
			case "$engine_build" in ''|*[!a-zA-Z0-9.-]*|.*|*.|*..*) return 1 ;; esac ;;
	esac
	case "$engine_version" in ''|*[!0-9.]*|.*|*.|*..*) return 1 ;; esac
	IFS=.
	set -f
	set -- $engine_version
	[ "$#" -eq 3 ] || return 1
	for component do
		case "$component" in ''|0[0-9]*) return 1 ;; esac
		[ "${#component}" -le %d ] || return 1
	done
	[ "$1" -ge %d ] && [ "$1" -lt %d ] || return 1
	[ "$1" -gt %d ] || [ "$2" -ge %d ]
)`, maxEngineVersionBytes, maxVersionComponentDigits, minEngineMajor, maxEngineMajorExclusive, minEngineMajor, minEngineMinor)
}
