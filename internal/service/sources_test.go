package service

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func TestSourceAuthorityIsPinnedAndConfined(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "source")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "sample.txt"), []byte("reviewed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := OpenSourceRoot("repo", directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	s, err := New(Options{Sources: []*SourceRoot{root}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(directory, directory+"-retained"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "rebound.txt"), []byte("wrong\n"), 0600); err != nil {
		t.Fatal(err)
	}
	selected, err := s.source(SourceSelection{Root: "repo", Directory: "."})
	if err != nil {
		t.Fatal(err)
	}
	defer selected.Close()
	manifest, err := deskwork.SnapshotRoot(selected)
	if err != nil || len(manifest.Files) != 1 || manifest.Files[0].Path != "sample.txt" {
		t.Fatal(manifest, err)
	}
	var bundle bytes.Buffer
	if err := deskwork.WriteBundleRoot(context.Background(), selected, manifest, &bundle); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bundle.Bytes(), []byte("rebound")) || !bytes.Contains(bundle.Bytes(), []byte("reviewed")) {
		t.Fatal("root path replacement rebound source")
	}
	if err := os.Symlink(parent, filepath.Join(directory+"-retained", "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".", filepath.Join(directory+"-retained", "internal")); err != nil {
		t.Fatal(err)
	}
	for _, selection := range []SourceSelection{
		{Root: "missing", Directory: "."}, {Root: "repo", Directory: "../source"},
		{Root: "repo", Directory: directory}, {Root: "repo", Directory: "escape"},
		{Root: "repo", Directory: "internal"}, {Root: "repo", Directory: "escape/source"},
		{Root: "repo", Directory: "a\\b"}, {Root: "repo", Directory: ""},
	} {
		opened, err := s.source(selection)
		if err == nil {
			opened.Close()
			t.Fatalf("unapproved selection accepted %+v", selection)
		}
	}
	if _, err := deskwork.SnapshotRoot(selected); err == nil {
		t.Fatal("link in source payload accepted")
	}
	if _, err := New(Options{Sources: []*SourceRoot{root, root}}); err == nil {
		t.Fatal("duplicate root accepted")
	}
}

func TestSourceSelectionSyntaxIsRejectedBeforeRegistryAccess(t *testing.T) {
	s, err := New(Options{ConfigDir: filepath.Join(t.TempDir(), "absent"), EnableMutations: true})
	if err != nil {
		t.Fatal(err)
	}
	result := s.Call(context.Background(), "desk.plan-stage.v1", []byte(`{"name":"dev","runner_id":"runner","instance_id":"ship","desk":"sample","source":{"root":"repo","directory":"../private"}}`))
	if result.Error == nil || result.Error.Code != "invalid_input" || result.Error.Dispatch != "not-dispatched" {
		t.Fatal(result)
	}
}
