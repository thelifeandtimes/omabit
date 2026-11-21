package guidance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bundleFixture(t *testing.T, third bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	m := manifest{provenance: provenance{Version: "test-dev", SourceCommit: strings.Repeat("a", 40)}, Format: 1, Skills: []string{"omabit-operator", "omabit-plugin-developer"}, Files: map[string]string{}}
	if third {
		m.Skills = append(m.Skills, "omabit-app-developer")
	}
	for _, name := range m.Skills {
		path := "skills/" + name + "/SKILL.md"
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0700); err != nil {
			t.Fatal(err)
		}
		data := []byte("# " + name + "\nBounded fixture guidance.\n")
		if err := os.WriteFile(filepath.Join(root, path), data, 0600); err != nil {
			t.Fatal(err)
		}
		m.Files[path] = hash(data)
	}
	build, _ := json.Marshal(m.provenance)
	if err := os.WriteFile(filepath.Join(root, "BUILD.json"), build, 0600); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(root, "SKILLS.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return root, hash(data)
}
func TestVerifiedGuidanceLegacyAndThirdSkill(t *testing.T) {
	for _, third := range []bool{false, true} {
		root, pin := bundleFixture(t, third)
		bundle, err := Open(root, pin)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range bundle.Index().Files {
			doc, err := bundle.Read(entry.Path)
			if err != nil || doc.SHA256 != entry.SHA256 || hash([]byte(doc.Text)) != entry.SHA256 || doc.ManifestSHA256 != pin {
				t.Fatal(doc, err)
			}
		}
		if _, err := bundle.Read("../../BUILD.json"); err == nil {
			t.Fatal("path escaped manifest")
		}
		path := bundle.Index().Files[0].Path
		if err := os.WriteFile(filepath.Join(root, path), []byte("changed"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := bundle.Read(path); err == nil {
			t.Fatal("changed resource accepted")
		}
	}
}
func TestGuidanceRefusesUnpinnedChangedLinkedAndExtraResources(t *testing.T) {
	for _, scenario := range []string{"pin", "build", "manifest", "symlink", "extra", "executable", "hardlink"} {
		t.Run(scenario, func(t *testing.T) {
			root, pin := bundleFixture(t, false)
			path := filepath.Join(root, "skills/omabit-operator/SKILL.md")
			switch scenario {
			case "pin":
				pin = strings.Repeat("0", 64)
			case "build":
				if err := os.WriteFile(filepath.Join(root, "BUILD.json"), []byte(`{"version":"other"}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "manifest":
				f, err := os.OpenFile(filepath.Join(root, "SKILLS.json"), os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				f.WriteString(" ")
				f.Close()
			case "symlink":
				outside := filepath.Join(t.TempDir(), "outside.md")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(outside, data, 0600); err != nil {
					t.Fatal(err)
				}
				if err = os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			case "extra":
				if err := os.WriteFile(filepath.Join(root, "skills/unlisted.md"), []byte("unlisted"), 0600); err != nil {
					t.Fatal(err)
				}
			case "executable":
				if err := os.Chmod(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, filepath.Join(t.TempDir(), "linked.md")); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Open(root, pin); err == nil {
				t.Fatal("unsafe guidance accepted")
			}
		})
	}
}
