package bootpill

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestPillURLFileAndHashBoundPublication(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	data := strings.Repeat("fixture-pill", 1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, data) }))
	defer server.Close()
	file, meta, err := Resolve(context.Background(), server.URL, dir)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Bytes != int64(len(data)) || meta.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(data))) {
		t.Fatal(meta)
	}
	local, again, err := Resolve(context.Background(), file, dir)
	if err != nil || again != meta {
		t.Fatal(again, err)
	}
	f, _ := Open(local)
	dest := filepath.Join(dir, "boot.pill")
	err = Publish(context.Background(), f, meta, dest)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(context.Background(), dest, meta); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Stage(context.Background(), dir, strings.NewReader("wrong"), &meta); err == nil {
		t.Fatal("mismatched pill published")
	}
	bad := protocol.BootPill{SHA256: meta.SHA256, Bytes: meta.Bytes + 1}
	if err := Verify(context.Background(), dest, bad); err == nil {
		t.Fatal("size mismatch accepted")
	}
}

func TestPillFileAndDescriptorAccept4GBAndRejectOverflow(t *testing.T) {
	// Sparse metadata checks the boundary without allocating or hashing 4 GB.
	path := filepath.Join(t.TempDir(), "large.pill")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, size := range []int64{0, 4_000_000_000, 4_000_000_001} {
		if err = f.Truncate(size); err != nil {
			t.Fatal(err)
		}
		opened, err := Open(path)
		if opened != nil {
			opened.Close()
		}
		valid := size == 4_000_000_000
		meta := protocol.BootPill{SHA256: strings.Repeat("a", 64), Bytes: size}
		if (err == nil) != valid || protocol.ValidBootPill(&meta) != valid {
			t.Fatalf("file/descriptor boundary mismatch at %d bytes: %v", size, err)
		}
	}
}
