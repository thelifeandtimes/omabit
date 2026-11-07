package deskwork

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestBundleRoundTripAndIncompleteUploads(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "desk.bill"), []byte("~\n"), 0600)
	os.Mkdir(filepath.Join(src, "app"), 0700)
	os.WriteFile(filepath.Join(src, "app/test.hoon"), []byte(":: app\n"), 0600)
	m, e := Snapshot(src)
	if e != nil {
		t.Fatal(e)
	}
	var b bytes.Buffer
	if e = WriteBundle(context.Background(), src, m, &b); e != nil {
		t.Fatal(e)
	}
	parent := t.TempDir()
	os.Chmod(parent, 0700)
	dest, got, e := StageBundle(context.Background(), parent, bytes.NewReader(b.Bytes()), m.Digest)
	if e != nil || got.Digest != m.Digest {
		t.Fatal(e)
	}
	check, e := Snapshot(dest)
	if e != nil || check.Digest != m.Digest {
		t.Fatal(check, e)
	}
	os.RemoveAll(dest)
	badHash := append([]byte{}, b.Bytes()...)
	badHash[len(badHash)-1] ^= 1
	tooBig := append([]byte{}, b.Bytes()...)
	binary.BigEndian.PutUint32(tooBig[len(bundleMagic):], MaxManifestBytes+1)
	for _, raw := range [][]byte{b.Bytes()[:2], b.Bytes()[:len(b.Bytes())-1], append(append([]byte{}, b.Bytes()...), 1), badHash, tooBig} {
		if _, _, err := StageBundle(context.Background(), parent, bytes.NewReader(raw), m.Digest); err == nil {
			t.Fatal("invalid bundle accepted")
		}
		entries, _ := os.ReadDir(parent)
		if len(entries) != 0 {
			t.Fatal("partial upload retained", entries)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e = StageBundle(ctx, parent, bytes.NewReader(b.Bytes()), m.Digest); e == nil {
		t.Fatal("cancelled upload accepted")
	}
	os.WriteFile(filepath.Join(src, "desk.bill"), []byte("changed"), 0600)
	if WriteBundle(context.Background(), src, m, &bytes.Buffer{}) == nil {
		t.Fatal("changed source transferred")
	}
}
