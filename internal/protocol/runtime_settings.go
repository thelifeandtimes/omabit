package protocol

// The pinned Vere 4.6-5a7614f main.c _main_read_loom accepts 20..u3a_bits_max.
// pkg/noun/allocate.h defines that maximum as 8*sizeof(c3_w)+u3a_vits;
// c3_w is uint32_t and u3a_vits is 2, giving 34. These are byte exponents,
// not resident-memory budgets. The managed container retains its 4 GiB cap.
const (
	MinLoom                      = 20
	MaxLoom                      = 34
	DefaultLoom                  = 31
	RuntimeSettingsFeature       = "runtime-settings"
	NativeControlAllShipsFeature = "native-control-all-ships"
)

// NativeControlWarning is shared by collector and runner help. Keep its wording
// aligned with the owner-facing RuntimeSettingsFields warning.
const NativeControlWarning = `Primarily for development. Leave disabled for routine use: it is unnecessary and adds risk.

This pinned Vere build has a known crash bug when a client disconnects with replies pending. Keep clients connected until every reply has been received. Do not kill clients or use short timeouts.

conn.sock grants full ship-control authority, not health-only access.`

func ValidLoom(loom uint8) bool { return loom >= MinLoom && loom <= MaxLoom }

// EffectiveLoom preserves pre-settings private runtime records. Public requests
// still require an explicit valid value; zero is not a request-side default.
func EffectiveLoom(loom uint8) uint8 {
	if loom == 0 {
		return DefaultLoom
	}
	return loom
}
