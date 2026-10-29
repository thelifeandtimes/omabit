package main

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestResumeReceiptReaderRejectsIndirectAndSpecialFiles(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink", "fifo", "oversize", "directory", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "selected.json")
			other := filepath.Join(root, "other.json")
			if e := os.WriteFile(other, []byte(`{"ok":true}`), 0600); e != nil {
				t.Fatal(e)
			}
			var err error
			switch kind {
			case "symlink":
				err = os.Symlink(other, path)
			case "hardlink":
				err = os.Link(other, path)
			case "fifo":
				err = unix.Mkfifo(path, 0600)
			case "oversize":
				err = os.WriteFile(path, []byte(`{}`), 0600)
				if err == nil {
					err = os.Truncate(path, 4*1024*1024+1)
				}
			case "directory":
				err = os.Mkdir(path, 0700)
			case "malformed":
				err = os.WriteFile(path, []byte(`not-json`), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if recover() == nil {
					t.Fatal("unsafe receipt accepted")
				}
			}()
			readPublicReceipt(path)
		})
	}
}

func TestResumeReceiptReaderBindsExactBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	if e := os.WriteFile(path, []byte("{\"ok\":true}\n"), 0600); e != nil {
		t.Fatal(e)
	}
	value, before := readPublicReceipt(path)
	if value["ok"] != true || len(before) != 64 {
		t.Fatal("receipt projection differs")
	}
	if e := os.WriteFile(path, []byte("{\"ok\": true}\n"), 0600); e != nil {
		t.Fatal(e)
	}
	_, after := readPublicReceipt(path)
	if before == after {
		t.Fatal("receipt hash does not bind exact bytes")
	}
}
