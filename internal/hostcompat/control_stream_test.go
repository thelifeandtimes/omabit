package hostcompat

import (
	"strings"
	"testing"
)

func TestRepositoryControlLargeUnselectedFields(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"Ubuntu Provides", "Provides: " + strings.Repeat("p", 70841) + "\n", true},
		{"oversized unrelated line", "Provides: " + strings.Repeat("p", 1<<20) + "\n", false},
		{"oversized folded field", "Description: " + strings.Repeat("d", 700000) + "\n " + strings.Repeat("d", 700000) + "\n", false},
		{"oversized selected value", "Filename: " + strings.Repeat("f", 4097) + "\n", false},
		{"folded selected value", "SHA256: abc\n def\n", false},
		{"duplicate selected field", "Size: 1\nSize: 2\n", false},
		{"duplicate ignored field", "Provides: a\nProvides: b\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			visits := 0
			err := scanControl(strings.NewReader("Package: target\nVersion: 1\nArchitecture: amd64\n"+tc.body+"\n"), 2, packageIndexFields, func(m map[string]string) error {
				visits++
				if m["Provides"] != "" || m["Description"] != "" {
					t.Fatal("unselected field retained")
				}
				return nil
			})
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
			if tc.valid && visits != 1 {
				t.Fatal("missing record")
			}
		})
	}
}
