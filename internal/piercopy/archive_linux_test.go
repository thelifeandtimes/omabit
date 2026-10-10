//go:build linux

package piercopy

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"testing"
)

func TestHandoffArchiveRoundTripAndRefusesOverwrite(t *testing.T) {
	_, _, source, target := tree(t)
	var archive bytes.Buffer
	want, err := WriteArchive(context.Background(), source, &archive)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadArchive(context.Background(), bytes.NewReader(archive.Bytes()), target)
	if err != nil || got != want {
		t.Fatal(got, want, err)
	}
	if _, err = ReadArchive(context.Background(), bytes.NewReader(archive.Bytes()), target); err == nil {
		t.Fatal("overwrote populated destination")
	}
}

func TestHandoffArchiveRejectsUnsafeEntries(t *testing.T) {
	for _, h := range []tar.Header{{Name: "../escape", Typeflag: tar.TypeReg}, {Name: "/escape", Typeflag: tar.TypeReg}, {Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/tmp"}, {Name: "hard", Typeflag: tar.TypeLink, Linkname: "."}, {Name: "device", Typeflag: tar.TypeChar}, {Name: ".", Typeflag: tar.TypeDir}} {
		t.Run(h.Name+string(h.Typeflag), func(t *testing.T) {
			var b bytes.Buffer
			w := tar.NewWriter(&b)
			if err := w.WriteHeader(&tar.Header{Name: ".", Mode: 0700, Typeflag: tar.TypeDir}); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteHeader(&h); err != nil {
				t.Fatal(err)
			}
			w.Close()
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if _, err = ReadArchive(context.Background(), &b, root); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}
