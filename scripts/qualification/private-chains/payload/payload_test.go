package payload

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func archive(t *testing.T, bad *tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	for n := 0; n < 10; n++ {
		if err := w.WriteHeader(&tar.Header{Name: fmt.Sprintf("src/file%d", n), Mode: 0644, Size: 1}); err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("x"))
	}
	if bad != nil {
		if err := w.WriteHeader(bad); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	return b.Bytes()
}
func TestPayloadRegularFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "new")
	if err := Unpack(bytes.NewReader(archive(t, nil)), root); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "src/file9"))
	if err != nil || string(b) != "x" {
		t.Fatal(err)
	}
}
func TestPayloadRefusesEscapesAndSpecials(t *testing.T) {
	for _, h := range []tar.Header{{Name: "../escape"}, {Name: "/absolute"}, {Name: "src/../../escape"}, {Name: "src/link", Typeflag: tar.TypeSymlink, Linkname: "/"}, {Name: "src/hard", Typeflag: tar.TypeLink, Linkname: "src/file0"}, {Name: "src/device", Typeflag: tar.TypeChar}, {Name: "src/file0"}, {Name: "unapproved/file"}, {Name: "src/huge", Size: 1024 * 1024 * 1024}} {
		t.Run(h.Name, func(t *testing.T) {
			if err := Unpack(bytes.NewReader(archive(t, &h)), filepath.Join(t.TempDir(), "new")); err == nil {
				t.Fatal("unsafe entry accepted")
			}
		})
	}
}
func TestPayloadRefusesExistingDestination(t *testing.T) {
	if err := Unpack(bytes.NewReader(archive(t, nil)), t.TempDir()); err == nil {
		t.Fatal("existing root accepted")
	}
}

func TestRunnerReplacementBindsBothRoles(t *testing.T) {
	old, new := []byte("previous runner fixture"), []byte("candidate runner fixture")
	hash := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
	if err := verifyRunner(old, new, hash(old), hash(new)); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name                string
		previous, candidate []byte
		oldSHA, newSHA      string
	}{
		{"changed-original", []byte("changed"), new, hash(old), hash(new)},
		{"changed-candidate", old, []byte("changed"), hash(old), hash(new)},
		{"swapped-roles", new, old, hash(old), hash(new)},
		{"missing-pins", old, new, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if verifyRunner(test.previous, test.candidate, test.oldSHA, test.newSHA) == nil {
				t.Fatal("unbound replacement accepted")
			}
		})
	}
	// A newly prepared base may already contain the same qualified binary.
	if err := verifyRunner(new, new, hash(new), hash(new)); err != nil {
		t.Fatal(err)
	}
}
func TestInstallRefusesHostBeforeFilesystemAccess(t *testing.T) {
	if os.Getpid() == 1 && os.Geteuid() == 0 {
		t.Skip("not a host-process test environment")
	}
	if err := Install(Config{}); err == nil || !strings.Contains(err.Error(), "guest PID1 required") {
		t.Fatal("host entry was not refused")
	}
}
