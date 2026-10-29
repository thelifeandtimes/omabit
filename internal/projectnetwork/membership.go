package projectnetwork

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// NetworkContainer is nonsecret metadata from one complete, fresh, engine-bound
// container inventory. Names resolve references only; they do not authorize
// adoption. Docker bridge endpoint lists omit container:-namespace participants.
type NetworkContainer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	NetworkMode string `json:"network_mode"`
}

var dockerNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,254}$`)
var shortDockerIDPattern = regexp.MustCompile(`^[0-9a-f]{1,63}$`)

// NamespaceMembers resolves direct and indirect container: joins to an exact
// anchor ID. Missing/ambiguous references and cycles make the inventory
// insufficient. Its caller must bracket collection against engine/inventory
// changes and inspect full contracts separately before claiming completeness.
func NamespaceMembers(anchorID string, all []NetworkContainer) ([]string, error) {
	refuse := func() ([]string, error) {
		return nil, errors.New("container namespace inventory is incomplete or ambiguous")
	}
	if !digestPattern.MatchString(anchorID) {
		return refuse()
	}
	byID := map[string]NetworkContainer{}
	byName := map[string]string{}
	for _, v := range all {
		name := strings.TrimPrefix(v.Name, "/")
		if !digestPattern.MatchString(v.ID) || !dockerNamePattern.MatchString(name) || v.NetworkMode == "" || len(v.NetworkMode) > 512 || strings.IndexFunc(v.NetworkMode, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
			return refuse()
		}
		if _, ok := byID[v.ID]; ok {
			return refuse()
		}
		if _, ok := byName[name]; ok {
			return refuse()
		}
		byID[v.ID] = v
		byName[name] = v.ID
	}
	anchor, ok := byID[anchorID]
	if !ok || strings.HasPrefix(anchor.NetworkMode, "container:") {
		return refuse()
	}
	resolve := func(ref string) (string, bool) {
		matches := map[string]bool{}
		if _, ok := byID[ref]; ok {
			matches[ref] = true
		}
		if id, ok := byName[ref]; ok {
			matches[id] = true
		}
		if shortDockerIDPattern.MatchString(ref) {
			for id := range byID {
				if strings.HasPrefix(id, ref) {
					matches[id] = true
				}
			}
		}
		if len(matches) != 1 {
			return "", false
		}
		for id := range matches {
			return id, true
		}
		return "", false
	}
	parents := map[string]string{}
	for id, v := range byID {
		if ref, shared := strings.CutPrefix(v.NetworkMode, "container:"); shared {
			parent, ok := resolve(ref)
			if !ok {
				return refuse()
			}
			parents[id] = parent
		}
	}
	// Each node has at most one parent. Memoization bounds repeated traversal and
	// detects cycles even in unrelated retained container groups.
	roots := map[string]string{}
	visiting := map[string]bool{}
	var root func(string) (string, bool)
	root = func(id string) (string, bool) {
		if value, ok := roots[id]; ok {
			return value, true
		}
		if visiting[id] {
			return "", false
		}
		visiting[id] = true
		value := id
		if parent := parents[id]; parent != "" {
			var ok bool
			value, ok = root(parent)
			if !ok {
				return "", false
			}
		}
		visiting[id] = false
		roots[id] = value
		return value, true
	}
	members := []string{}
	for id := range byID {
		value, ok := root(id)
		if !ok {
			return refuse()
		}
		if id != anchorID && value == anchorID {
			members = append(members, id)
		}
	}
	slices.Sort(members)
	return members, nil
}
