package protocol

import (
	"strings"
	"testing"
)

func TestFakeShipSpellingAndPierNames(t *testing.T) {
	if len(shipPrefixes) != 768 || len(shipSuffixes) != 768 {
		t.Fatal("incomplete @p alphabet")
	}
	for _, ship := range []string{"~zod", "~bus", "~rus", "~marzod", "~sampel-palnet", "~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod"} {
		if !ValidFakeShip(ship) {
			t.Fatal("valid fake name rejected", ship)
		}
	}
	for _, ship := range []string{"zod", "~foo", "~BUS", "~dozzod", "~doznec", "~sampel--palnet", "~sampel/../palnet", "~nolpel-modsyd-livsep-novmeg-sallen-tassul-padlud-litzod"} {
		if ValidFakeShip(ship) {
			t.Fatal("invalid fake name accepted", ship)
		}
	}
	for _, name := range []string{"bus-pier", "local_bus", "pier1"} {
		if !ValidPierName(name) {
			t.Fatal(name)
		}
	}
	for _, name := range []string{"", ".", "..", "../pier", "/outside", "a/b", "-flag", "two words"} {
		if ValidPierName(name) {
			t.Fatal("invalid directory", name)
		}
	}
}

func TestGeneratedMoonNameAndTransposedPrefixDiagnostic(t *testing.T) {
	const moon = "~dibpen-fogweg-dozzod-sarlev"
	if ShipClass(moon) != "moon" || ShipNameProblem(moon) != "" {
		t.Fatal("valid generated moon rejected")
	}
	const typo = "~dipben-fogweg-dozzod-sarlev"
	if ValidFakeShip(typo) || !strings.Contains(ShipNameProblem(typo), `prefix syllable "dip"`) {
		t.Fatal("typo must identify the invalid public syllable")
	}
}
