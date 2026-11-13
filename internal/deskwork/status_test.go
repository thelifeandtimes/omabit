package deskwork

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type sourceReaderFixture struct {
	reads     int
	scenario  string
	directory string
	clay      Manifest
}

func (f *sourceReaderFixture) Observe(context.Context, string) (Observation, error) {
	f.reads++
	if f.scenario == "unavailable" {
		return Observation{}, errors.New("unavailable")
	}
	rev := uint64(3)
	if f.reads > 1 {
		if f.scenario == "clay-changed" {
			rev++
		}
		if f.scenario == "unix-changed" {
			os.WriteFile(filepath.Join(f.directory, "hello.txt"), []byte("new edit"), 0600)
		}
	}
	return Observation{Exists: true, Revision: rev, ClayHash: "0v123", Publication: "private"}, nil
}
func (f *sourceReaderFixture) ClayFiles(context.Context, string, uint64) (Manifest, error) {
	return f.clay, nil
}
func (f *sourceReaderFixture) NormalizeFiles(_ context.Context, _ string, _ uint64, files []SourceFile) (Manifest, error) {
	if f.scenario == "unknown-mark" {
		return Manifest{}, errors.New("unknown mark")
	}
	if len(files) != 1 || files[0].Path != "hello.txt" {
		return Manifest{}, errors.New("unexpected source")
	}
	hash := strings.Repeat("a", 64)
	if f.scenario == "different" || f.scenario == "diverged" {
		hash = strings.Repeat("b", 64)
	}
	return NewManifest([]File{{Path: "hello.txt", SHA256: hash}})
}
func (f *sourceReaderFixture) AutocommitActive(context.Context) (bool, error) {
	if f.scenario == "unknown-auto" {
		return false, errors.New("no timer evidence")
	}
	return f.scenario == "active-auto", nil
}
func TestSourceInspectionDistinguishesPendingDivergedAndUnknown(t *testing.T) {
	for _, scenario := range []string{"same", "different", "diverged", "clay-changed", "unix-changed", "unknown-mark", "unavailable", "unknown-auto", "active-auto"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\n"), 0600)
			clay, _ := NewManifest([]File{{Path: "hello.txt", SHA256: strings.Repeat("a", 64)}})
			f := &sourceReaderFixture{scenario: scenario, directory: dir, clay: clay}
			var baseline *Manifest
			if scenario == "diverged" {
				b, _ := NewManifest([]File{{Path: "hello.txt", SHA256: strings.Repeat("c", 64)}})
				baseline = &b
			}
			out, err := InspectSource(context.Background(), f, dir, "example", baseline)
			switch scenario {
			case "clay-changed", "unix-changed", "unknown-mark", "unavailable":
				if err == nil || out.Version != 0 {
					t.Fatal("false successful observation", out, err)
				}
			default:
				expected := "synchronized"
				if scenario == "different" {
					expected = "different"
				}
				if scenario == "diverged" {
					expected = "diverged"
				}
				if err != nil || out.Comparison.State != expected {
					t.Fatal(out, err)
				}
				if scenario == "unknown-auto" && out.Autocommit != "unknown" || scenario == "active-auto" && out.Autocommit != "active" {
					t.Fatal("unknown auto treated as inactive", out)
				}
				if scenario == "different" || scenario == "diverged" {
					if len(out.Differences) != 1 {
						t.Fatal(out)
					}
				}
			}
		})
	}
}
