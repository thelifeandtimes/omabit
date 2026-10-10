package runtimequalify

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"os"
	"path/filepath"
	"testing"
)

func TestHandoffArtifactUsesOwnSizeOwnershipAndLinkBounds(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "artifact.tar")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	// Sparse metadata validates the 100 GB stream boundary without writing or
	// transferring a pier of that size (including the separate framing allowance).
	if err = f.Truncate(protocol.MaxHandoffBytes); err != nil {
		t.Fatal(err)
	}
	if err = f.Chmod(0400); err != nil {
		t.Fatal(err)
	}
	r, err := OpenHandoffArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	link := filepath.Join(dir, "link.tar")
	if err = os.Link(path, link); err != nil {
		t.Fatal(err)
	}
	if r, err = OpenHandoffArchive(path); err == nil {
		r.Close()
		t.Fatal("shared artifact accepted")
	}
	os.Remove(link)
	if err = os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if r, err = OpenHandoffArchive(link); err == nil {
		r.Close()
		t.Fatal("symlink accepted")
	}
	if err = f.Truncate(protocol.MaxHandoffBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if r, err = OpenHandoffArchive(path); err == nil {
		r.Close()
		t.Fatal("oversized artifact accepted")
	}
}
