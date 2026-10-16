package piermeta

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Minimal independently specified LMDB v1 pages: two commit slots, MAIN's
// named META descriptor, and public atoms stored in a leaf. No event data.
func metadataFixture() []byte {
	return metadataAtoms(nil, nil)
}
func metadataAtoms(fake, who []byte) []byte {
	data := make([]byte, 4*4096)
	for n := 0; n < 2; n++ {
		p := data[n*4096:]
		binary.LittleEndian.PutUint64(p, uint64(n))
		binary.LittleEndian.PutUint16(p[10:], 8)
		binary.LittleEndian.PutUint32(p[16:], 0xbeefc0de)
		binary.LittleEndian.PutUint32(p[20:], 1)
		binary.LittleEndian.PutUint32(p[40:], 4096)
		binary.LittleEndian.PutUint64(p[128:], 2)
		binary.LittleEndian.PutUint64(p[136:], 3)
		binary.LittleEndian.PutUint64(p[144:], uint64(n+1))
	}
	leaf := func(number int, values map[string][]byte, flag uint16) {
		p := data[number*4096 : (number+1)*4096]
		binary.LittleEndian.PutUint64(p, uint64(number))
		binary.LittleEndian.PutUint16(p[10:], 2)
		keys := []string{}
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		upper := 4096
		for i, key := range keys {
			value := values[key]
			length := 8 + len(key) + len(value)
			upper -= (length + 1) &^ 1
			node := p[upper:]
			binary.LittleEndian.PutUint32(node, uint32(len(value)))
			binary.LittleEndian.PutUint16(node[4:], flag)
			binary.LittleEndian.PutUint16(node[6:], uint16(len(key)))
			copy(node[8:], key)
			copy(node[8+len(key):], value)
			binary.LittleEndian.PutUint16(p[16+2*i:], uint16(upper))
		}
		binary.LittleEndian.PutUint16(p[12:], uint16(16+2*len(keys)))
		binary.LittleEndian.PutUint16(p[14:], uint16(upper))
	}
	descriptor := make([]byte, 48)
	binary.LittleEndian.PutUint64(descriptor[40:], 3)
	leaf(2, map[string][]byte{"META": descriptor}, 2)
	leaf(3, map[string][]byte{"fake": fake, "life": {1}, "version": {3}, "who": who}, 0)
	return data
}

func TestReadLiveIdentityAndBranchSnapshot(t *testing.T) {
	atom := make([]byte, 4)
	binary.LittleEndian.PutUint32(atom, 9896704)
	data := metadataAtoms([]byte{1}, atom)
	// Move MAIN's leaf to page 4 and make page 2 a branch pointing there.
	data = append(data, data[2*4096:3*4096]...)
	binary.LittleEndian.PutUint64(data[4*4096:], 4)
	for n := 0; n < 2; n++ {
		binary.LittleEndian.PutUint64(data[n*4096+136:], 4)
	}
	branch := data[2*4096 : 3*4096]
	clear(branch)
	binary.LittleEndian.PutUint64(branch, 2)
	binary.LittleEndian.PutUint16(branch[10:], 1)
	binary.LittleEndian.PutUint16(branch[12:], 18)
	binary.LittleEndian.PutUint16(branch[14:], 4088)
	binary.LittleEndian.PutUint16(branch[16:], 4088)
	binary.LittleEndian.PutUint32(branch[4088:], 4)
	// The old committed slot is not authoritative.
	binary.LittleEndian.PutUint64(data[128:], 999)
	got, err := Read(bytes.NewReader(data), int64(len(data)))
	if err != nil || got.Ship != "~poldec-tonteg" || got.Network != "livenet" || got.Kind != "planet" {
		t.Fatal(got, err)
	}
	binary.LittleEndian.PutUint32(branch[4088:], 2)
	if _, err := Read(bytes.NewReader(data), int64(len(data))); err == nil {
		t.Fatal("branch cycle accepted")
	}
}

type changingMetadata struct {
	*bytes.Reader
	reads int
}

func (r *changingMetadata) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.Reader.ReadAt(p, off)
	if off == 0 {
		r.reads++
		if r.reads > 1 && n > 144 {
			p[144]++
		}
	}
	return n, err
}
func TestReadRejectsChangingSnapshot(t *testing.T) {
	data := metadataFixture()
	if _, err := Read(&changingMetadata{Reader: bytes.NewReader(data)}, int64(len(data))); err == nil {
		t.Fatal("changing metadata snapshot accepted")
	}
}
func FuzzReadMetadata(f *testing.F) {
	f.Add(metadataFixture())
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}
		_, _ = Read(bytes.NewReader(data), int64(len(data)))
	})
}

func TestReadPublicMetadataAndRejectMalformedPages(t *testing.T) {
	data := metadataFixture()
	got, err := Read(bytes.NewReader(data), int64(len(data)))
	if err != nil || got.Ship != "~zod" || got.Network != "fakenet" || got.Kind != "fake" {
		t.Fatal(got, err)
	}
	for _, mutate := range []func([]byte){
		func(p []byte) { p[16] = 0 },
		func(p []byte) { binary.LittleEndian.PutUint64(p[4096+136:], 1<<60) },
		func(p []byte) { binary.LittleEndian.PutUint64(p[4096+128:], 999) },
		func(p []byte) { binary.LittleEndian.PutUint16(p[2*4096+10:], 4) },
		func(p []byte) { binary.LittleEndian.PutUint16(p[3*4096+16:], 0) },
		func(p []byte) { binary.LittleEndian.PutUint16(p[3*4096+12:], 4095) },
	} {
		p := append([]byte(nil), data...)
		mutate(p)
		if _, err := Read(bytes.NewReader(p), int64(len(p))); err == nil {
			t.Fatal("accepted corrupted metadata")
		}
	}
	for _, length := range []int{0, 16, 151, 4096, 8192, 12000} {
		if _, err := Read(bytes.NewReader(data[:length]), int64(length)); err == nil {
			t.Fatal("accepted truncated metadata", length)
		}
	}
}

func TestReadStoppedVereFixture(t *testing.T) {
	pier := os.Getenv("OMABIT_TEST_STOPPED_PIER")
	if pier == "" {
		t.Skip("set OMABIT_TEST_STOPPED_PIER and OMABIT_TEST_SHIP for a read-only real Vere fixture")
	}
	if _, err := os.Lstat(filepath.Join(pier, ".vere.lock")); !os.IsNotExist(err) {
		t.Fatal("fixture must be stopped with no Vere lock")
	}
	f, err := os.Open(filepath.Join(pier, ".urb/log/data.mdb"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Read(f, info.Size())
	if err != nil || got.Ship != os.Getenv("OMABIT_TEST_SHIP") {
		t.Fatal(got, err)
	}
	t.Logf("read-only metadata: %+v", got)
}
