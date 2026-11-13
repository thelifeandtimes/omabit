package eyre

import (
	"bytes"
	"errors"
	"testing"
)

func TestValidateStoredSessionRequiresExactBindingWithoutNetwork(t *testing.T) {
	const valid = `{"origin":"https://unused.invalid","ship":"~zod","cookie_name":"urbauth-~zod","cookie_value":"synthetic","expires":"2000-01-01T00:00:00Z"}`
	for _, kind := range []string{"expired", "origin", "ship", "unknown", "duplicate", "missing", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			data, origin, ship := []byte(valid), "https://unused.invalid", "~zod"
			switch kind {
			case "origin":
				origin = "https://other.invalid"
			case "ship":
				ship = "~nec"
			case "unknown":
				data = append([]byte(`{"extra":"value",`), data[1:]...)
			case "duplicate":
				data = append([]byte(`{"ship":"~zod",`), data[1:]...)
			case "missing":
				data = []byte(`{}`)
			case "malformed":
				data = []byte(`PRIVATE malformed`)
			}
			err := ValidateStoredSession(origin, ship, data)
			if kind == "expired" && err != nil || kind != "expired" && !errors.Is(err, ErrStoredSession) {
				t.Fatal(err)
			}
			if !bytes.Equal(data, make([]byte, len(data))) {
				t.Fatal("retained secret input")
			}
		})
	}
}
