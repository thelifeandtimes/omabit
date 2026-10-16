// Package piermeta reads only the public identity/fake metadata of a stopped
// Vere pier. It never opens an LMDB environment, starts Vere, replays events,
// acquires a writer lock, or changes a snapshot/database.
package piermeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var invalid = errors.New("unsupported or inconsistent Vere identity metadata; source was not changed")

type Identity struct {
	Ship    string `json:"ship"`
	Network string `json:"network"`
	Kind    string `json:"kind"`
}

// Read supports the 64-bit little-endian LMDB v1 format used by qualified Vere
// 4.6. It follows only MAIN -> META -> {version,who,fake}, with bounded page reads.
// Source exclusivity and filesystem identities are checked by the caller.
func Read(input io.ReaderAt, size int64) (Identity, error) {
	var out Identity
	first := make([]byte, 152)
	if _, err := input.ReadAt(first, 0); err != nil {
		return out, invalid
	}
	pageSize := int(binary.LittleEndian.Uint32(first[40:]))
	if pageSize < 512 || pageSize > 32768 || pageSize&(pageSize-1) != 0 || size < int64(2*pageSize) {
		return out, invalid
	}
	second := make([]byte, 152)
	if _, err := input.ReadAt(second, int64(pageSize)); err != nil {
		return out, invalid
	}
	validMeta := func(data []byte, page uint64) bool {
		return binary.LittleEndian.Uint64(data) == page && binary.LittleEndian.Uint16(data[10:]) == 8 && binary.LittleEndian.Uint32(data[16:]) == 0xbeefc0de && binary.LittleEndian.Uint32(data[20:]) == 1 && int(binary.LittleEndian.Uint32(data[40:])) == pageSize
	}
	if !validMeta(first, 0) || !validMeta(second, 1) {
		return out, invalid
	}
	meta := first
	if binary.LittleEndian.Uint64(second[144:]) > binary.LittleEndian.Uint64(first[144:]) {
		meta = second
	}
	lastPage := binary.LittleEndian.Uint64(meta[136:])
	if lastPage >= uint64(size/int64(pageSize)) || lastPage < 2 {
		return out, invalid
	}
	reads := 0
	lookup := func(root uint64, key string) ([]byte, uint16, error) {
		seen := map[uint64]bool{}
		for depth := 0; depth < 16; depth++ {
			reads++
			if reads > 64 || root < 2 || root > lastPage || seen[root] {
				return nil, 0, invalid
			}
			seen[root] = true
			page := make([]byte, pageSize)
			if _, err := input.ReadAt(page, int64(root)*int64(pageSize)); err != nil || binary.LittleEndian.Uint64(page) != root {
				return nil, 0, invalid
			}
			flags := binary.LittleEndian.Uint16(page[10:])
			lower, upper := int(binary.LittleEndian.Uint16(page[12:])), int(binary.LittleEndian.Uint16(page[14:]))
			if (flags != 1 && flags != 2) || lower < 18 || lower%2 != 0 || lower > upper || upper > pageSize {
				return nil, 0, invalid
			}
			var next uint64
			previous := ""
			for index := 16; index < lower; index += 2 {
				offset := int(binary.LittleEndian.Uint16(page[index:]))
				if offset < upper || offset > pageSize-8 {
					return nil, 0, invalid
				}
				node := page[offset:]
				length := int(binary.LittleEndian.Uint16(node[6:]))
				if length > len(node)-8 {
					return nil, 0, invalid
				}
				name := string(node[8 : 8+length])
				if index > 16 && name <= previous {
					return nil, 0, invalid
				}
				previous = name
				dataSize := uint64(binary.LittleEndian.Uint32(node))
				nodeFlags := binary.LittleEndian.Uint16(node[4:])
				if flags == 1 {
					if index == 16 && name != "" {
						return nil, 0, invalid
					}
					if name <= key {
						next = dataSize | uint64(nodeFlags)<<32
					}
				} else if name == key {
					// The public atoms and named DB descriptor fit inline. Never
					// traverse overflow/event pages or duplicate-value databases.
					if dataSize > 48 || dataSize > uint64(len(node)-8-length) || (nodeFlags != 0 && nodeFlags != 2) {
						return nil, 0, invalid
					}
					return node[8+length : 8+length+int(dataSize)], nodeFlags, nil
				}
			}
			if flags == 2 || next == 0 {
				return nil, 0, invalid
			}
			root = next
		}
		return nil, 0, invalid
	}
	// MDB_db is 48 bytes on this supported platform, root page at offset 40.
	if binary.LittleEndian.Uint16(meta[92:]) != 0 {
		return out, invalid
	}
	db, flags, err := lookup(binary.LittleEndian.Uint64(meta[128:]), "META")
	if err != nil || flags != 2 || len(db) != 48 || binary.LittleEndian.Uint16(db[4:]) != 0 {
		return out, invalid
	}
	root := binary.LittleEndian.Uint64(db[40:])
	values := map[string][]byte{}
	for _, key := range []string{"version", "who", "fake"} {
		value, flags, err := lookup(root, key)
		if err != nil || flags != 0 {
			return out, invalid
		}
		values[key] = value
	}
	// Vere strips trailing zero bytes; c3y (fake) is zero, c3n is one.
	version, fake := values["version"], values["fake"]
	if len(version) != 1 {
		return out, invalid
	}
	if version[0] != 3 {
		return out, fmt.Errorf("unsupported Vere disk metadata version %x; expected Vere 4.6 disk version 3", version)
	}
	if len(fake) > 1 || len(fake) == 1 && fake[0] > 1 {
		return out, invalid
	}
	out.Ship, err = protocol.ShipFromAtom(values["who"])
	if err != nil {
		return Identity{}, invalid
	}
	out.Network, out.Kind = "livenet", protocol.ShipClass(out.Ship)
	if len(fake) == 0 || fake[0] == 0 {
		out.Network, out.Kind = "fakenet", "fake"
	}
	// Reject a changing database instead of mixing committed snapshots.
	for number, before := range [][]byte{first, second} {
		after := make([]byte, len(before))
		if _, err := input.ReadAt(after, int64(number*pageSize)); err != nil || !bytes.Equal(before, after) {
			return Identity{}, invalid
		}
	}
	return out, nil
}
