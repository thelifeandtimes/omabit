package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordingArchivePublicationIsPrivateFreshAndFailurePreserving(t *testing.T) {
	for _, variant := range []string{"ok", "existing", "partial", "failure", "symlink-parent", "public-parent", "racing-output"} {
		t.Run(variant, func(t *testing.T) {
			parent := t.TempDir()
			if err := os.Chmod(parent, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(parent, "recording.zip")
			if variant == "existing" {
				if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if variant == "partial" {
				if err := os.WriteFile(path+".partial", []byte("older partial"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if variant == "public-parent" {
				if err := os.Chmod(parent, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if variant == "symlink-parent" {
				link := parent + "-link"
				if err := os.Symlink(parent, link); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Remove(link) })
				path = filepath.Join(link, "recording.zip")
			}
			calls := 0
			digest, err := writeRecordingExport(path, func(w io.Writer) error {
				calls++
				if _, err := w.Write([]byte("private archive bytes")); err != nil {
					return err
				}
				if variant == "failure" {
					return errors.New("injected transfer failure")
				}
				if variant == "racing-output" {
					return os.WriteFile(path, []byte("racing owner output"), 0600)
				}
				return nil
			})
			if variant == "ok" {
				if err != nil || len(digest) != 64 || calls != 1 {
					t.Fatal("fresh export failed", err)
				}
				info, err := os.Stat(path)
				if err != nil || info.Mode().Perm() != 0600 {
					t.Fatal("permissions", err)
				}
				if _, err := os.Lstat(path + ".partial"); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("success kept partial alias")
				}
				return
			}
			if err == nil || digest != "" {
				t.Fatal("refused/partial output published")
			}
			switch variant {
			case "failure":
				if calls != 1 || !strings.Contains(err.Error(), ".partial") {
					t.Fatal(err)
				}
				data, err := os.ReadFile(path + ".partial")
				if err != nil || !bytes.Equal(data, []byte("private archive bytes")) {
					t.Fatal("failure lost partial bytes", err)
				}
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("partial gained final name")
				}
			case "racing-output":
				data, err := os.ReadFile(path)
				if err != nil || string(data) != "racing owner output" {
					t.Fatal("racing output overwritten", err)
				}
			case "existing":
				data, _ := os.ReadFile(path)
				if string(data) != "existing" || calls != 0 {
					t.Fatal("existing output changed")
				}
			case "partial":
				data, _ := os.ReadFile(path + ".partial")
				if string(data) != "older partial" || calls != 0 {
					t.Fatal("existing partial changed")
				}
			default:
				if calls != 0 {
					t.Fatal("unsafe directory began writing")
				}
			}
		})
	}
}
