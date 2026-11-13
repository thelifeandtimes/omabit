package deskwork

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalFileInventoryIsBoundedAndHasStableDigests(t *testing.T) {
	files := []CanonicalFile{{Path: []string{"hello", "txt"}, Hash: "0xa.bcd"}}
	a, err := CanonicalManifest(files)
	if err != nil || a.Files[0].SHA256 != strings.Repeat("0", 60)+"abcd" || a.Files[0].Bytes != 0 {
		t.Fatal(a, err)
	}
	for _, bad := range [][]CanonicalFile{nil, {{Path: []string{"hello", "txt"}, Hash: "0x"}}, {{Path: []string{"hello", "txt"}, Hash: "0xnot-hex"}}, {{Path: []string{"..", "hello", "txt"}, Hash: "0x1"}}, {files[0], files[0]}} {
		if _, err := CanonicalManifest(bad); err == nil {
			t.Fatal("invalid canonical manifest accepted")
		}
	}
	for _, bad := range []SourceFile{{Path: "../hello.txt"}, {Path: "hello..txt'"}} {
		if _, err := NormalizeFiles("~zod", "base", 2, []SourceFile{bad}); err == nil {
			t.Fatal("unsafe source accepted")
		}
	}
	if _, err := ClayFiles("~zod", "base", 0); err == nil {
		t.Fatal("mutable revision")
	}
	if _, err := NormalizeFiles("~zod", "base", 2, make([]SourceFile, 129)); err == nil {
		t.Fatal("unbounded batch")
	}
}
func TestCanonicalSnapshotDoesNotReportCleanAfterChangedSource(t *testing.T) {
	for _, scenario := range []string{"ok", "changed", "replaced-root", "normalize-error", "wrong-files", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			parent := t.TempDir()
			dir := filepath.Join(parent, "source")
			os.Mkdir(dir, 0700)
			file := filepath.Join(dir, "hello.txt")
			os.WriteFile(file, []byte("hello\n"), 0600)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if scenario == "cancelled" {
				cancel()
			}
			normalize := func(_ context.Context, files []SourceFile) (Manifest, error) {
				if len(files) != 1 || string(files[0].Bytes) != "hello\n" {
					t.Fatal("wrong source", files)
				}
				switch scenario {
				case "normalize-error":
					return Manifest{}, errors.New("unsupported mark")
				case "changed":
					os.WriteFile(file, []byte("edited during read"), 0600)
				case "replaced-root":
					os.Rename(dir, dir+"-old")
					os.Mkdir(dir, 0700)
					os.WriteFile(file, []byte("hello\n"), 0600)
				}
				name := "hello.txt"
				if scenario == "wrong-files" {
					name = "other.txt"
				}
				hash := sha256.Sum256([]byte("canonical fixture hello"))
				return NewManifest([]File{{Path: name, SHA256: hex.EncodeToString(hash[:])}})
			}
			raw, canonical, err := SnapshotCanonical(ctx, dir, normalize)
			if scenario == "ok" {
				if err != nil || raw.Digest == canonical.Digest || len(canonical.Files) != 1 {
					t.Fatal(raw, canonical, err)
				}
			} else if err == nil || raw.Version != 0 || canonical.Version != 0 {
				t.Fatal("invalid comparison accepted", raw, canonical, err)
			}
		})
	}
}
func TestCanonicalSnapshotBatchesAndRejectsSymlinks(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 130; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file-%03d.txt", i)), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	counts := []int{}
	normalize := func(_ context.Context, files []SourceFile) (Manifest, error) {
		counts = append(counts, len(files))
		out := []File{}
		for _, f := range files {
			out = append(out, File{Path: f.Path, SHA256: strings.Repeat("a", 64)})
		}
		return NewManifest(out)
	}
	raw, canonical, err := SnapshotCanonical(context.Background(), dir, normalize)
	if err != nil || len(raw.Files) != 130 || len(canonical.Files) != 130 || len(counts) != 2 || counts[0] != 128 || counts[1] != 2 {
		t.Fatal(counts, err)
	}
	os.Symlink("file-001.txt", filepath.Join(dir, "link.txt"))
	if _, _, err = SnapshotCanonical(context.Background(), dir, normalize); err == nil {
		t.Fatal("symlink source accepted")
	}
}

func TestCommitDeltaCannotSelectAnotherDeskOrInjectCode(t *testing.T) {
	baseline := Observation{Exists: true, Revision: 3, ClayHash: "0v123", Publication: "private"}
	for _, delta := range [][]DeltaFile{nil, {{Path: "../secret.txt"}}, {{Path: "hello.txt"}, {Path: "hello.txt"}}, {{Path: "hello.txt", Delete: true, Bytes: []byte("bad")}}} {
		if _, err := CommitFiles("~zod", "example", baseline, delta); err == nil {
			t.Fatal("invalid delta accepted")
		}
	}
	bad := baseline
	bad.ClayHash = "0v123) code"
	if _, err := CommitFiles("~zod", "example", bad, []DeltaFile{{Path: "hello.txt"}}); err == nil {
		t.Fatal("hash injection accepted")
	}
	r, err := CommitFiles("~zod", "example", baseline, []DeltaFile{{Path: "hello.txt", Bytes: []byte("raw ' bytes \n |commit")}})
	if err != nil || strings.Contains(r.expression, "raw ' bytes") || !strings.Contains(r.expression, "[%c %info %example") || r.app != "hood" || r.mark != "helm-pass" {
		t.Fatal("wrong target/transport", err)
	}
}
