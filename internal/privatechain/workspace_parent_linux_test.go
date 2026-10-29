//go:build linux

package privatechain

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceParentInspectionIsReadOnlyAndIdentityBound(t *testing.T) {
	base := t.TempDir()
	directory := filepath.Join(base, "private-projects")
	if _, err := InspectWorkspaceParent(context.Background(), directory); err == nil {
		t.Fatal("missing parent admitted")
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("inspection created parent", err)
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	id, err := InspectWorkspaceParent(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	again, err := InspectWorkspaceParent(context.Background(), directory)
	if err != nil || id != again {
		t.Fatal("stable parent identity changed", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("inspection modified directory", err)
	}
	if err = os.Rename(directory, directory+"-retained"); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	replacement, err := InspectWorkspaceParent(context.Background(), directory)
	if err != nil || id.SameObject(replacement) {
		t.Fatal("replacement identity not distinguished", err)
	}
}
func TestWorkspaceParentInspectionRejectsAliasPermissionsAndCancellation(t *testing.T) {
	for _, kind := range []string{"symlink", "ancestor-link", "public", "file", "wrong-name", "relative", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			base := t.TempDir()
			directory := filepath.Join(base, "private-projects")
			ctx := context.Background()
			switch kind {
			case "file":
				if err := os.WriteFile(directory, []byte("retained"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(base, directory); err != nil {
					t.Fatal(err)
				}
			default:
				if err := os.Mkdir(directory, 0700); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "ancestor-link":
				link := base + "-link"
				if err := os.Symlink(base, link); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Remove(link) })
				directory = filepath.Join(link, "private-projects")
			case "public":
				if err := os.Chmod(directory, 0755); err != nil {
					t.Fatal(err)
				}
			case "wrong-name":
				directory = base
			case "relative":
				directory = "private-projects"
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, err := InspectWorkspaceParent(ctx, directory); err == nil {
				t.Fatal("unsafe inspection admitted")
			}
		})
	}
}
