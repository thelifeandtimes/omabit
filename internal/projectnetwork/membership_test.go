package projectnetwork

import (
	"reflect"
	"strings"
	"testing"
)

func TestNamespaceMembersResolveDirectIndirectNamesAndIDs(t *testing.T) {
	a, b, c, d := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)
	all := []NetworkContainer{{a, "/anchor", "owned-bridge"}, {b, "/direct", "container:" + a}, {c, "/indirect", "container:direct"}, {d, "/other", "none"}}
	original := append([]NetworkContainer(nil), all...)
	got, err := NamespaceMembers(a, all)
	if err != nil || !reflect.DeepEqual(got, []string{b, c}) || !reflect.DeepEqual(all, original) {
		t.Fatal(got, err)
	}
	all[1].NetworkMode = "container:anchor"
	if got, err = NamespaceMembers(a, all); err != nil || len(got) != 2 {
		t.Fatal(got, err)
	}
	all[1].NetworkMode = "container:" + a[:12]
	if got, err = NamespaceMembers(a, all); err != nil || len(got) != 2 {
		t.Fatal(got, err)
	}
	if got, err = NamespaceMembers(a, all[:1]); err != nil || len(got) != 0 {
		t.Fatal(got, err)
	}
}
func TestNamespaceMembersRefusePartialAmbiguousAndCyclicInventory(t *testing.T) {
	a, b, c := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
	base := []NetworkContainer{{a, "anchor", "bridge"}, {b, "member", "container:" + a}, {c, "other", "none"}}
	cases := map[string]func([]NetworkContainer) []NetworkContainer{
		"missing-anchor":     func(v []NetworkContainer) []NetworkContainer { return v[1:] },
		"duplicate-id":       func(v []NetworkContainer) []NetworkContainer { v[2].ID = b; return v },
		"duplicate-name":     func(v []NetworkContainer) []NetworkContainer { v[2].Name = "/member"; return v },
		"dangling-reference": func(v []NetworkContainer) []NetworkContainer { v[2].NetworkMode = "container:missing"; return v },
		"self-cycle":         func(v []NetworkContainer) []NetworkContainer { v[2].NetworkMode = "container:other"; return v },
		"cycle": func(v []NetworkContainer) []NetworkContainer {
			v[1].NetworkMode = "container:other"
			v[2].NetworkMode = "container:member"
			return v
		},
		"shared-anchor":     func(v []NetworkContainer) []NetworkContainer { v[0].NetworkMode = "container:other"; return v },
		"empty-mode":        func(v []NetworkContainer) []NetworkContainer { v[2].NetworkMode = ""; return v },
		"invalid-name":      func(v []NetworkContainer) []NetworkContainer { v[2].Name = "/path/other"; return v },
		"name-id-ambiguity": func(v []NetworkContainer) []NetworkContainer { v[2].Name = a; return v },
		"short-id-ambiguity": func(v []NetworkContainer) []NetworkContainer {
			v[2].ID = a[:12] + strings.Repeat("c", 52)
			v[1].NetworkMode = "container:" + a[:12]
			return v
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			all := mutate(append([]NetworkContainer(nil), base...))
			if got, err := NamespaceMembers(a, all); err == nil || got != nil {
				t.Fatal("ambiguous inventory claimed membership", got, err)
			}
		})
	}
}
