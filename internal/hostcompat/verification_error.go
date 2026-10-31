package hostcompat

import (
	"errors"
	"sort"
	"strings"
)

func packageVerificationError(report Report) error {
	names := map[string]bool{}
	for _, p := range report.Packages {
		if p.Provenance != "verified" && token(p.Name) {
			names[p.Name] = true
		}
	}
	unresolved := make([]string, 0, len(names))
	for name := range names {
		unresolved = append(unresolved, name)
	}
	sort.Strings(unresolved)
	if len(unresolved) > 16 {
		unresolved = unresolved[:16]
	}
	if len(unresolved) == 0 {
		return errors.New("native runtime package authentication is incomplete; trusted package inventory is unavailable")
	}
	return errors.New("could not authenticate installed runtime packages: " + strings.Join(unresolved, ", ") + "; matching trusted repository payload evidence is unavailable or invalid")
}
