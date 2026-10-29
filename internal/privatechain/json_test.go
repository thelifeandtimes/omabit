package privatechain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFoundationDecodeRejectsAmbiguityAndPrivateInputs(t *testing.T) {
	data, _ := json.Marshal(foundation(t))
	if _, err := DecodeFoundation(data); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"duplicate root":   `{"version":1,` + string(data[1:]),
		"case alias":       strings.Replace(string(data), `"version":1`, `"Version":1`, 1),
		"duplicate nested": strings.Replace(string(data), `"name":"Azimuth"`, `"name":"Azimuth","name":"Azimuth"`, 1),
		"private field":    `{"private_key":"do-not-reflect",` + string(data[1:]),
		"trailing object":  string(data) + `{}`, "null": `null`, "too deep": strings.Repeat("[", 10) + "1" + strings.Repeat("]", 10),
		"oversize": strings.Repeat(" ", MaxFoundationBytes+1), "key instead of JSON": "0w.do-not-reflect",
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeFoundation([]byte(value)); err == nil || strings.Contains(err.Error(), "do-not-reflect") {
				t.Fatal("malformed or private input accepted/reflected", err)
			}
		})
	}
}
