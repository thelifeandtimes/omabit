package protocol

import (
	"fmt"
	"strings"
)

// The standard @p syllable alphabet (arvo/sys/hoon.hoon, also published in
// https://github.com/urbit/urbit-ob/blob/master/src/internal/co.js).
const shipPrefixes = "dozmarbinwansamlitsighidfidlissogdirwacsabwissib" +
	"rigsoldopmodfoglidhopdardorlorhodfolrintogsilmir" +
	"holpaslacrovlivdalsatlibtabhanticpidtorbolfosdot" +
	"losdilforpilramtirwintadbicdifrocwidbisdasmidlop" +
	"rilnardapmolsanlocnovsitnidtipsicropwitnatpanmin" +
	"ritpodmottamtolsavposnapnopsomfinfonbanmorworsip" +
	"ronnorbotwicsocwatdolmagpicdavbidbaltimtasmallig" +
	"sivtagpadsaldivdactansidfabtarmonranniswolmispal" +
	"lasdismaprabtobrollatlonnodnavfignomnibpagsopral" +
	"bilhaddocridmocpacravripfaltodtiltinhapmicfanpat" +
	"taclabmogsimsonpinlomrictapfirhasbosbatpochactid" +
	"havsaplindibhosdabbitbarracparloddosbortochilmac" +
	"tomdigfilfasmithobharmighinradmashalraglagfadtop" +
	"mophabnilnosmilfopfamdatnoldinhatnacrisfotribhoc" +
	"nimlarfitwalrapsarnalmoslandondanladdovrivbacpol" +
	"laptalpitnambonrostonfodponsovnocsorlavmatmipfip"
const shipSuffixes = "zodnecbudwessevpersutletfulpensytdurwepserwylsun" +
	"rypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnex" +
	"lunmeplutseppesdelsulpedtemledtulmetwenbynhexfeb" +
	"pyldulhetmevruttylwydtepbesdexsefwycburderneppur" +
	"rysrebdennutsubpetrulsynregtydsupsemwynrecmegnet" +
	"secmulnymtevwebsummutnyxrextebfushepbenmuswyxsym" +
	"selrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpel" +
	"syptermebsetdutdegtexsurfeltudnuxruxrenwytnubmed" +
	"lytdusnebrumtynseglyxpunresredfunrevrefmectedrus" +
	"bexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermer" +
	"tenlusnussyltecmexpubrymtucfyllepdebbermughuttun" +
	"bylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmyl" +
	"wedducfurfexnulluclennerlexrupnedlecrydlydfenwel" +
	"nydhusrelrudneshesfetdesretdunlernyrsebhulryllud" +
	"remlysfynwerrycsugnysnyllyndyndemluxfedsedbecmun" +
	"lyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

func hasSyllable(table, syllable string) bool {
	if len(syllable) != 3 {
		return false
	}
	for i := 0; i < len(table); i += 3 {
		if table[i:i+3] == syllable {
			return true
		}
	}
	return false
}

// Validate spelling/grouping through 128-bit ship names. Vere remains the final
// identity parser, and the running /~/host must equal the registered name.
func ValidFakeShip(ship string) bool {
	if len(ship) < 4 || len(ship) > 57 || ship[0] != '~' {
		return false
	}
	name := ship[1:]
	if len(name) == 3 {
		return hasSyllable(shipSuffixes, name)
	}
	words := strings.Split(strings.ReplaceAll(name, "--", "-"), "-")
	if len(words) > 8 {
		return false
	}
	var canonical strings.Builder
	for i, word := range words {
		if len(word) != 6 || !hasSyllable(shipPrefixes, word[:3]) || !hasSyllable(shipSuffixes, word[3:]) {
			return false
		}
		if i == 0 && (word == "dozzod" || len(words) == 1 && word[:3] == "doz") {
			return false
		}
		if i > 0 {
			if (len(words)-i)%4 == 0 {
				canonical.WriteString("--")
			} else {
				canonical.WriteByte('-')
			}
		}
		canonical.WriteString(word)
	}
	return canonical.String() == name
}

func ValidPierName(name string) bool {
	return ValidID(name) && ((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z') || (name[0] >= '0' && name[0] <= '9'))
}

// ShipNameProblem describes a public-name error without inspecting key material.
func ShipNameProblem(ship string) string {
	if ValidFakeShip(ship) {
		return ""
	}
	if len(ship) < 4 || len(ship) > 57 || !strings.HasPrefix(ship, "~") {
		return "Enter a valid Urbit ID beginning with ~."
	}
	name := ship[1:]
	if len(name) == 3 {
		return fmt.Sprintf("Unknown Urbit suffix syllable %q; check the generated ID.", name)
	}
	words := strings.Split(strings.ReplaceAll(name, "--", "-"), "-")
	for _, word := range words {
		if len(word) != 6 {
			return "Urbit IDs use six-letter words separated by hyphens (or a three-letter galaxy name)."
		}
		if !hasSyllable(shipPrefixes, word[:3]) {
			return fmt.Sprintf("Unknown Urbit prefix syllable %q in %q; check the generated ID.", word[:3], word)
		}
		if !hasSyllable(shipSuffixes, word[3:]) {
			return fmt.Sprintf("Unknown Urbit suffix syllable %q in %q; check the generated ID.", word[3:], word)
		}
	}
	return "Urbit ID has noncanonical leading syllables or hyphen grouping; use the exact generated ID."
}

func ShipClass(ship string) string {
	if !ValidFakeShip(ship) {
		return ""
	}
	n := len(strings.ReplaceAll(strings.TrimPrefix(ship, "~"), "-", ""))
	switch {
	case n == 3:
		return "galaxy"
	case n == 6:
		return "star"
	case n == 12:
		return "planet"
	case n <= 24:
		return "moon"
	default:
		return "comet"
	}
}
