package deskwork

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestTypedRequestsRejectInjection(t *testing.T) {
	for _, bad := range []string{"", "../base", "%base", "base, =auto &", "base\n|exit", "base/test", "Base"} {
		if _, e := Create("~zod", bad); e == nil {
			t.Fatal(bad)
		}
		if _, e := Commit(bad); e == nil {
			t.Fatal(bad)
		}
		if _, e := PublishSource(bad); e == nil {
			t.Fatal(bad)
		}
	}
	for _, bad := range []string{"zod", "~zod /base", "~zod\n|exit", "~zod'"} {
		if _, e := Inventory(bad, "base"); e == nil {
			t.Fatal(bad)
		}
	}
	r, e := Commit("omabit-test")
	if e != nil {
		t.Fatal(e)
	}
	b, e := r.Body()
	if e != nil {
		t.Fatal(e)
	}
	var got map[string]any
	if json.Unmarshal(b, &got) != nil {
		t.Fatal("invalid JSON")
	}
	if !strings.Contains(string(b), "kiln-commit") || strings.Contains(string(b), "%.y") {
		t.Fatal(string(b))
	}
	for _, r := range []LensRequest{r, func() LensRequest { v, _ := Inventory("~zod", "omabit-test"); return v }(), func() LensRequest { v, _ := Create("~zod", "omabit-test"); return v }()} {
		if strings.Count(r.expression, "(") != strings.Count(r.expression, ")") || strings.Count(r.expression, "[") != strings.Count(r.expression, "]") {
			t.Fatal(r.expression)
		}
	}
}
func TestObservationDecoding(t *testing.T) {
	encode := func(s string) []byte {
		raw := []byte(s)
		for i, j := 0, len(raw)-1; i < j; i, j = i+1, j-1 {
			raw[i], raw[j] = raw[j], raw[i]
		}
		b, _ := json.Marshal("0x" + hex.EncodeToString(raw) + "\n")
		return b
	}
	var out struct {
		Exists   bool   `json:"exists"`
		Revision uint64 `json:"revision"`
	}
	if e := DecodeJSONOutput(encode(`{"exists":true,"revision":2}`), &out); e != nil || !out.Exists || out.Revision != 2 {
		t.Fatal(out, e)
	}
	for _, bad := range [][]byte{[]byte(`">="`), []byte(`"0xg"`), encode(`{"exists":true} trailing`), encode(`{"extra":true}`), encode(`{} {}`), encode(`{"revision":-1}`)} {
		if DecodeJSONOutput(bad, &out) == nil {
			t.Fatal(string(bad))
		}
	}
}

func TestComparisonTreatsBytesAsDataAndGroupsHoonNumbers(t *testing.T) {
	raw := []byte("' |exit\n\x00\xff")
	r, e := MarkEquivalent("~zod", "example", 1234567, "web/foo.min.js", raw)
	if e != nil {
		t.Fatal(e)
	}
	b, e := r.Body()
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(r.expression, "|exit") || !strings.Contains(r.expression, "1.234.567") || !strings.Contains(string(b), `"data":`) {
		t.Fatal(string(b))
	}
	if strings.Count(r.expression, "(") != strings.Count(r.expression, ")") || strings.Count(r.expression, "[") != strings.Count(r.expression, "]") {
		t.Fatal(r.expression)
	}
	for n, want := range map[uint64]string{0: "0", 999: "999", 1000: "1.000", 1234567: "1.234.567"} {
		if got := hoonDecimal(n); got != want {
			t.Fatal(got, want)
		}
	}
	if _, e = MarkEquivalent("~zod", "example", 0, "hello.txt", nil); e == nil {
		t.Fatal("unfenced revision accepted")
	}
	if _, e = MarkEquivalent("~zod", "example", 1, "../secret.txt", nil); e == nil {
		t.Fatal("escaping file accepted")
	}
}

func TestActualRuntimeObservationFixture(t *testing.T) {
	body, e := os.ReadFile("testdata/runtime-observations.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixtures map[string]string
	if e = json.Unmarshal(body, &fixtures); e != nil {
		t.Fatal(e)
	}
	for name, value := range fixtures {
		raw, _ := json.Marshal(value)
		var observed Observation
		if e = DecodeJSONOutput(raw, &observed); e != nil {
			t.Fatal(name, e)
		}
		if e = observed.Validate(); e != nil {
			t.Fatal(name, e)
		}
		if name == "typed-inventory" && (observed.Revision != 1 || observed.Publication != "private") {
			t.Fatal(observed)
		}
		if name == "inventory-published" && (observed.Revision != 3 || observed.Publication != "published") {
			t.Fatal(observed)
		}
	}
}
