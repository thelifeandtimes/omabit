package protocol

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func TestShipAtomReferenceVectors(t *testing.T) {
	// Independent public reference vectors from urbit-ob/test/co.test.js.
	for value, name := range map[uint64]string{0: "~zod", 200: "~lex", 512: "~binzod", 1024: "~samzod", 9896704: "~poldec-tonteg", 15663360: "~nidsut-tomdun", 3108299008: "~morlyd-mogmev", 479733505: "~fipfes-morlyd"} {
		var atom [8]byte
		binary.LittleEndian.PutUint64(atom[:], value)
		if got, err := ShipFromAtom(atom[:]); err != nil || got != name {
			t.Fatal(value, got, name, err)
		}
	}
	atom, _ := hex.DecodeString("ffffffffffffffffffffffffffffffff")
	if got, err := ShipFromAtom(atom); err != nil || got != "~fipfes-fipfes-fipfes-fipfes--fipfes-fipfes-fipfes-fipfes" {
		t.Fatal(got, err)
	}
	if _, err := ShipFromAtom(make([]byte, 17)); err == nil {
		t.Fatal("accepted an oversized identity")
	}
}
