package hostsetup

import (
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func subids(text, uid string, hostIDs []uint64) (string, string) {
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
		ranges = append(ranges, interval{start, start + count, owner == rootless.AccountName || (uid != "" && owner == uid)})
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
	info, err := fs.Stat(p.files, "var/lib/pacman/local")
	if err != nil || !info.IsDir() {
		return "unknown", "Pacman package database is unavailable or not a directory; Docker version is unknown."
	}
	entries, err := fs.ReadDir(p.files, "var/lib/pacman/local")
	if err != nil {
		return "unknown", "Pacman package metadata is unavailable; Docker version is unknown and no program was executed."
	}
	version := ""
	for _, entry := range entries {
		suffix, match := strings.CutPrefix(entry.Name(), "docker-")
		// Only engine version directories, not compose/buildx/rootless packages.
		if !match || suffix == "" || suffix[0] < '0' || suffix[0] > '9' {
			continue
		}
		text, err := p.read("var/lib/pacman/local/" + entry.Name() + "/desc")
		if err != nil {
			return "unknown", "Docker package metadata is unreadable or invalid."
		}
		name, candidate := "", ""
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			if line != "%NAME%" && line != "%VERSION%" {
				continue
			}
			if i+1 >= len(lines) || lines[i+1] == "" || (i+2 < len(lines) && lines[i+2] != "") {
				return "unknown", "Docker package metadata is malformed."
			}
			field := &name
			if line == "%VERSION%" {
				field = &candidate
			}
			if *field != "" {
				return "unknown", "Docker package metadata has duplicate fields."
			}
			*field = lines[i+1]
		}
		if name == "" || candidate == "" || len(candidate) > 128 {
			return "unknown", "Docker package metadata is incomplete or invalid."
		}
		if name != "docker" {
			continue
		}
		if version != "" {
			return "unknown", "Multiple Docker engine package records are ambiguous."
		}
		if epoch, rest, found := strings.Cut(candidate, ":"); found {
			if _, ok := number(epoch); !ok {
				return "unknown", "Docker package epoch is malformed."
			}
			candidate = rest
		}
		release := strings.LastIndex(candidate, "-")
		if release < 1 {
			return "unknown", "Docker package release is malformed."
		}
		for _, part := range strings.Split(candidate[release+1:], ".") {
			if _, ok := number(part); !ok {
				return "unknown", "Docker package release is malformed."
			}
		}
		version = candidate[:release]
		parts := strings.Split(version, ".")
		if len(parts) != 3 {
			return "unknown", "Docker upstream version is malformed or unrecognized."
		}
		for _, part := range parts {
			if _, ok := number(part); !ok {
				return "unknown", "Docker upstream version is malformed or unrecognized."
			}
		}
	}
	if version == "" {
		return "missing", "No Docker engine package record found in the local Pacman database."
	}
	if version != rootless.EngineVersion {
		return "blocked", "Docker package upstream version " + version + " does not match required " + rootless.EngineVersion + "."
	}
	return "pass", "Docker package metadata matches upstream " + rootless.EngineVersion + "; installed binaries and runtime remain unverified."
}
