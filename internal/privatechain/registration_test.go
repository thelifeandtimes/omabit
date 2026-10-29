package privatechain

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRegistrationDecodingRejectsAmbiguityAndSecretFields(t *testing.T) {
	raw, err := os.ReadFile("testdata/qualified-foundation.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := DecodeFoundation(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Build(Request{Label: "Development", Foundation: f, ConfirmSharedAdministration: true, Members: []MemberRequest{{InstanceID: "dev-zod", Ship: "~zod", HTTPPort: 18381}, {InstanceID: "dev-nec", Ship: "~nec", HTTPPort: 18382}}}, Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := p.Digest()
	r := RegisterRequest{RequestID: "register", Label: p.Label, Foundation: f, ConfirmSharedAdministration: true, ConfirmReservation: true, ReviewDigest: digest}
	for _, m := range p.Members {
		r.Members = append(r.Members, m.MemberRequest)
	}
	good, _ := json.Marshal(r)
	for _, kind := range []string{"valid", "duplicate", "case-alias", "unknown-key", "unknown-start", "trailing", "oversized", "altered-chain", "no-consent", "nested-duplicate"} {
		t.Run(kind, func(t *testing.T) {
			data := bytes.Clone(good)
			switch kind {
			case "duplicate":
				data = append(data[:len(data)-1], []byte(`,"request_id":"other"}`)...)
			case "case-alias":
				data = append(data[:len(data)-1], []byte(`,"Request_ID":"other"}`)...)
			case "unknown-key":
				data = append(data[:len(data)-1], []byte(`,"private_key":"DO-NOT-REFLECT-THIS"}`)...)
			case "unknown-start":
				data = append(data[:len(data)-1], []byte(`,"start":true}`)...)
			case "trailing":
				data = append(data, []byte(` {}`)...)
			case "oversized":
				data = append(data, bytes.Repeat([]byte(" "), MaxFoundationBytes)...)
			case "altered-chain":
				data = bytes.Replace(data, []byte(`"chain_id":1337`), []byte(`"chain_id":1`), 1)
			case "no-consent":
				data = bytes.Replace(data, []byte(`"confirm_reservation":true`), []byte(`"confirm_reservation":false`), 1)
			case "nested-duplicate":
				data = bytes.Replace(data, []byte(`"chain_id":1337`), []byte(`"chain_id":1337,"chain_id":1337`), 1)
			}
			got, err := DecodeRegistration(data)
			if kind == "valid" {
				if err != nil || got.RequestID != r.RequestID {
					t.Fatal(err)
				}
				return
			}
			if err == nil || got.RequestID != "" || strings.Contains(err.Error(), "DO-NOT-REFLECT-THIS") {
				t.Fatal("invalid review accepted or reflected", kind, err)
			}
		})
	}
}
