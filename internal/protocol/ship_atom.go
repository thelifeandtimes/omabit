package protocol

import (
	"encoding/binary"
	"errors"
	"math/bits"
	"strings"
)

// ShipFromAtom formats Vere's little-endian public identity atom as canonical @p.
// The concealment and syllable grouping follow ++fein/++pat-p in hoon.hoon and
// urbit/urbit-ob. Only the 128-bit ship domain is accepted (never key material).
func ShipFromAtom(atom []byte) (string, error) {
	if len(atom) > 16 {
		return "", errors.New("ship identity exceeds 128 bits")
	}
	var bytes [16]byte
	copy(bytes[:], atom)
	lo, hi := binary.LittleEndian.Uint64(bytes[:8]), binary.LittleEndian.Uint64(bytes[8:])
	if hi == 0 {
		low := uint64(uint32(lo))
		if low >= 65536 {
			conceal := func(n uint64) uint64 {
				left, right := n%65535, n/65535
				for j, seed := range []uint32{0xb76d5eed, 0xee281300, 0x85bcae01, 0x4b387af7} {
					modulus := uint64(65535 + j%2)
					left, right = right, (left+uint64(shipHash(seed, uint16(right))))%modulus
				}
				if right == 65535 {
					return 65535*right + left
				}
				return 65535*left + right
			}
			value := conceal(low - 65536)
			if value >= 0xffffffff {
				value = conceal(value)
			}
			lo = lo&0xffffffff00000000 | (65536 + value)
		}
		binary.LittleEndian.PutUint64(bytes[:8], lo)
	}
	if hi == 0 && lo < 256 {
		return "~" + shipSuffixes[3*lo:3*lo+3], nil
	}
	last := 7
	for last > 0 && binary.LittleEndian.Uint16(bytes[2*last:]) == 0 {
		last--
	}
	var name strings.Builder
	name.WriteByte('~')
	for i := last; i >= 0; i-- {
		if i < last {
			if (i+1)%4 == 0 {
				name.WriteString("--")
			} else {
				name.WriteByte('-')
			}
		}
		prefix, suffix := int(bytes[2*i+1])*3, int(bytes[2*i])*3
		name.WriteString(shipPrefixes[prefix : prefix+3])
		name.WriteString(shipSuffixes[suffix : suffix+3])
	}
	return name.String(), nil
}

// MurmurHash3 x86_32 for the two-byte little-endian Feistel round input.
func shipHash(seed uint32, value uint16) uint32 {
	k := uint32(value) * 0xcc9e2d51
	k = bits.RotateLeft32(k, 15) * 0x1b873593
	h := seed ^ k ^ 2
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	return h ^ h>>16
}
