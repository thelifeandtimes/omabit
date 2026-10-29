package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectoryDiagnosticExportsOnlyFixedMetadata(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "private-name")
	if err := os.Mkdir(selected, 0700); err != nil {
		t.Fatal(err)
	}
	value := directoryMetadata(selected)
	if value["directory"] != true || value["canonical"] != true || value["mode"] != uint32(0700) {
		t.Fatal("directory metadata differs")
	}
	if err := os.Chmod(selected, 0755); err != nil {
		t.Fatal(err)
	}
	if directoryMetadata(selected)["mode"] != uint32(0755) {
		t.Fatal("mode mismatch not visible")
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(selected, link); err != nil {
		t.Fatal(err)
	}
	value = directoryMetadata(link)
	raw, _ := json.Marshal(value)
	if value["symlink"] != true || value["directory"] != false || value["canonical"] != false || strings.Contains(string(raw), "private-name") || strings.Contains(string(raw), root) {
		t.Fatal("diagnostic followed or exposed target")
	}
	if directoryMetadata(filepath.Join(root, "absent"))["present"] != false {
		t.Fatal("absent path admitted")
	}
}
