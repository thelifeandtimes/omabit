package credentials

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialOutputLimitSurvivesIOCopyFastPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized-output")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", limit+1)), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out cappedBuffer
	if _, err := io.Copy(&out, f); !errors.Is(err, ErrStore) || out.Len() > limit {
		t.Fatal("io.Copy bypassed private output limit", err, out.Len())
	}
}
