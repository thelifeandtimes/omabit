//go:build linux

package pierstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTreeReviewLargeLocalPier(t *testing.T) {
	pier := t.TempDir()
	name := filepath.Join(pier, "sparse-test")
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, size := range []int64{100_000_000_001, 512 << 30, MaxLocalPierBytes} {
		if err = f.Truncate(size); err != nil {
			t.Fatal(err)
		}
		_, files, bytes, err := TreeReview(context.Background(), pier, true)
		if err != nil || files != 1 || bytes != size {
			t.Fatalf("size %d: files %d bytes %d err %v", size, files, bytes, err)
		}
	}
	if err = os.WriteFile(filepath.Join(pier, "overflow"), []byte{1}, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = TreeReview(context.Background(), pier, true); err == nil {
		t.Fatal("aggregate bound ignored")
	}
}
