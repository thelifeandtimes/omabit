//go:build linux

package piercopy

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestLargeLocalPierDoesNotExpandHandoffStream(t *testing.T) {
	s, _, source, _ := tree(t)
	if err := os.Truncate(filepath.Join(s, ".urb/chk/image.bin"), protocol.MaxPierBytes+1); err != nil {
		t.Fatal(err)
	}
	var stream bytes.Buffer
	if _, err := WriteArchive(context.Background(), source, &stream); err == nil || stream.Len() != 0 {
		t.Fatal("large handoff emitted data", err, stream.Len())
	}
	if _, err := inventory(context.Background(), source); err != nil {
		t.Fatal("local metadata refused", err)
	}
}
