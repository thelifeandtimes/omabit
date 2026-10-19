package hostsetup

import (
	"sort"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func subids(text, uid string, hostIDs []uint64) (string, string) {
	return ValidateSubIDs(rootless.AccountName, text, uid, hostIDs)
}

func ValidateSubIDs(name, text, uid string, hostIDs []uint64) (string, string) {
	type interval struct {
		start, end uint64
		target     bool
	}
	var ranges []interval
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) != 3 || f[0] == "" || strings.ContainsAny(f[0], " \t\r") {
			return "unknown", "Subordinate-ID database is malformed."
		}
		owner := f[0]
		if strings.Trim(owner, "0123456789") == "" {
			n, ok := number(owner)
			if !ok || owner != strconv.FormatUint(n, 10) {
				return "unknown", "Numeric subordinate-ID owner is invalid or noncanonical; exact matching is required."
			}
		}
		start, sok := number(f[1])
		count, cok := number(f[2])
		if !sok || !cok || count == 0 || start+count > 1<<32-1 {
			return "unknown", "Subordinate-ID range is malformed or overflows usable IDs."
		}
		ranges = append(ranges, interval{start, start + count, owner == name || (uid != "" && owner == uid)})
	}
	if hostIDs == nil {
		return "unknown", "Complete local host ID inventory is unavailable or malformed; subordinate-ID safety is unknown."
	}
	hostIDs = append([]uint64(nil), hostIDs...)
	sort.Slice(hostIDs, func(i, j int) bool { return hostIDs[i] < hostIDs[j] })
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	var allEnd, targetEnd, runEnd, runSize, largest uint64
	// Sorted high-water marks detect even nested overlaps without quadratic work.
	for _, r := range ranges {
		if (r.target && r.start < allEnd) || r.start < targetEnd {
			return "blocked", "Overlapping subordinate-ID ranges involve the dedicated account."
		}
		allEnd = max(allEnd, r.end)
		if r.target {
			i := sort.Search(len(hostIDs), func(i int) bool { return hostIDs[i] >= r.start })
			if r.start == 0 || (i < len(hostIDs) && hostIDs[i] < r.end) {
				return "blocked", "Dedicated-account subordinate-ID range overlaps root or a known local host ID."
			}
			targetEnd = r.end
			if r.start != runEnd {
				runSize = 0
			}
			runSize += r.end - r.start
			runEnd, largest = r.end, max(largest, runSize)
		}
	}
	if uid == "" {
		return "unknown", "Target UID is unknown; numeric subordinate-ID ownership cannot be resolved."
	}
	if largest < 65536 {
		return "missing", "Dedicated account needs at least 65536 contiguous, non-overlapping subordinate IDs."
	}
	return "pass", "At least 65536 contiguous dedicated-account subordinate IDs found, with no target overlaps or known local host ID collisions."
}

func (p probe) engineVersion() (string, string) {
	return engineVersionCheck(hostcompat.Inspect(p.files, p.goos, p.goarch))
}

// Preserve the original check ID/status vocabulary for older CLI consumers.
// A pass here describes version compatibility only, never payload authentication.
func engineVersionCheck(report hostcompat.Report) (string, string) {
	if report.InstalledEngineVersion == "" {
		return "unknown", "Native package engine version is unresolved; review compatibility checks and platform guidance. No executable was run."
	}
	switch hostcompat.EngineVersionStatus(report.InstalledEngineVersion) {
	case "compatible", "qualified", "compatible-unqualified":
		return "pass", "Native package engine version meets compatibility policy; payload authenticity, running engine and full-stack qualification are separate checks."
	default:
		return "blocked", "Native package engine version is outside the supported compatibility policy; review platform guidance."
	}
}
