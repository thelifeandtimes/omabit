//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func inputTreeFixture(t *testing.T) (string, []RuntimeInputFile) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "bundle")
	if err := os.MkdirAll(filepath.Join(root, "tools", "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	data := map[string]string{"manifest.json": "{}", "tools/nested/empty": "", "tools/code": "#!/bin/sh\necho THIS MUST NEVER EXECUTE\n"}
	var files []RuntimeInputFile
	for name, body := range data {
		mode := os.FileMode(0400)
		if name == "tools/code" {
			mode = 0500
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), mode); err != nil {
			t.Fatal(err)
		}
		if name != "manifest.json" {
			files = append(files, RuntimeInputFile{Path: name, Bytes: int64(len(body)), SHA256: inputSHA([]byte(body)), Executable: mode == 0500})
		}
	}
	for _, p := range []string{root, filepath.Join(root, "tools"), filepath.Join(root, "tools", "nested")} {
		if err := os.Chmod(p, 0500); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_ = filepath.Walk(root, func(p string, i os.FileInfo, e error) error {
			if e == nil && i.IsDir() {
				_ = os.Chmod(p, 0700)
			}
			return nil
		})
	})
	return root, files
}

func TestRuntimeInputTreeReadOnlyVerification(t *testing.T) {
	directory, files := inputTreeFixture(t)
	root, err := openInputRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := verifyInputTree(context.Background(), root, files); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(directory, "tools/code"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(before), "MUST NEVER EXECUTE") {
		t.Fatal("input changed")
	}
	if _, err := os.Stat(filepath.Join(directory, "result")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unexpected effect")
	}
	for _, f := range files {
		i, err := root.Lstat(f.Path)
		if err != nil || i.Mode().Perm()&0200 != 0 {
			t.Fatal("input permissions changed", err)
		}
	}
}

func TestRuntimeInputTreeRefusesChangedAndIndirectFiles(t *testing.T) {
	cases := map[string]func(*testing.T, string, []RuntimeInputFile){
		"missing": func(t *testing.T, p string, f []RuntimeInputFile) {
			mustInput(t, os.Chmod(filepath.Join(p, "tools"), 0700))
			mustInput(t, os.Remove(filepath.Join(p, "tools/code")))
			mustInput(t, os.Chmod(filepath.Join(p, "tools"), 0500))
		},
		"extra": func(t *testing.T, p string, f []RuntimeInputFile) {
			mustInput(t, os.Chmod(p, 0700))
			mustInput(t, os.WriteFile(filepath.Join(p, "secret"), []byte("PRIVATE-SENTINEL"), 0400))
			mustInput(t, os.Chmod(p, 0500))
		},
		"extra-directory": func(t *testing.T, p string, f []RuntimeInputFile) {
			mustInput(t, os.Chmod(p, 0700))
			mustInput(t, os.Mkdir(filepath.Join(p, "unreviewed"), 0500))
			mustInput(t, os.Chmod(p, 0500))
		},
		"same-size-corruption": func(t *testing.T, p string, f []RuntimeInputFile) {
			q := filepath.Join(p, "tools/code")
			data, e := os.ReadFile(q)
			mustInput(t, e)
			data[0] = '!'
			mustInput(t, os.Chmod(q, 0700))
			mustInput(t, os.WriteFile(q, data, 0700))
			mustInput(t, os.Chmod(q, 0500))
		},
		"writable-file": func(t *testing.T, p string, f []RuntimeInputFile) {
			mustInput(t, os.Chmod(filepath.Join(p, "tools/code"), 0700))
		},
		"writable-directory": func(t *testing.T, p string, f []RuntimeInputFile) {
			mustInput(t, os.Chmod(filepath.Join(p, "tools"), 0700))
		},
		"hardlink": func(t *testing.T, p string, f []RuntimeInputFile) {
			mustInput(t, os.Link(filepath.Join(p, "tools/code"), filepath.Join(filepath.Dir(p), "another-name")))
		},
		"symlink": func(t *testing.T, p string, f []RuntimeInputFile) {
			mustInput(t, os.Chmod(filepath.Join(p, "tools"), 0700))
			q := filepath.Join(p, "tools/code")
			mustInput(t, os.Remove(q))
			mustInput(t, os.Symlink("/etc/passwd", q))
			mustInput(t, os.Chmod(filepath.Join(p, "tools"), 0500))
		},
		"fifo": func(t *testing.T, p string, f []RuntimeInputFile) {
			mustInput(t, os.Chmod(filepath.Join(p, "tools"), 0700))
			q := filepath.Join(p, "tools/code")
			mustInput(t, os.Remove(q))
			mustInput(t, syscall.Mkfifo(q, 0500))
			mustInput(t, os.Chmod(filepath.Join(p, "tools"), 0500))
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			directory, files := inputTreeFixture(t)
			change(t, directory, files)
			root, err := openInputRoot(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			err = verifyInputTree(context.Background(), root, files)
			if err == nil || strings.Contains(err.Error(), "PRIVATE-SENTINEL") {
				t.Fatal("invalid tree accepted or reflected", err)
			}
		})
	}
}

func mustInput(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeInputRootRefusalsAndCancellation(t *testing.T) {
	directory, files := inputTreeFixture(t)
	link := filepath.Join(filepath.Dir(directory), "linked")
	mustInput(t, os.Symlink(directory, link))
	for _, p := range []string{link, link + "/.", "relative", "", directory + "/../bundle"} {
		if r, err := openInputRoot(p); err == nil {
			r.Close()
			t.Fatal("noncanonical root admitted", p)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := InspectRuntimeInputs(ctx, "/does-not-exist", "", ""); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation must precede filesystem access", err)
	}
	r, err := openInputRoot(directory)
	mustInput(t, err)
	defer r.Close()
	if err := verifyInputTree(ctx, r, files); !errors.Is(err, context.Canceled) {
		t.Fatal("tree cancellation ignored", err)
	}
	if _, err := InspectRuntimeInputs(context.Background(), directory, strings.Repeat("a", 64), strings.Repeat("b", 64)); err == nil {
		t.Fatal("unreviewed manifest admitted")
	}
}
