package omabittheme

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

func palette() Palette {
	return Palette{ID: ThemeID, Name: "miasma", Dark: true, Primary: "#78824B", Secondary: "#78824B", Tertiary: "#5F875F", Background: "#222222", Surface: "#2C2C2C"}
}

type fakeSettings struct {
	body  map[string]any
	pokes int
	fail  bool
}

func (f *fakeSettings) Scry(context.Context, string, string) (json.RawMessage, error) {
	data, err := json.Marshal(f.body)
	return data, err
}
func (f *fakeSettings) Poke(_ context.Context, app, mark string, data json.RawMessage) (eyre.PokeResult, error) {
	f.pokes++
	if f.fail {
		return eyre.PokeResult{}, eyre.ErrUncertain
	}
	if app != "settings" || mark != "settings-event" {
		return eyre.PokeResult{}, errors.New("wrong target")
	}
	var msg map[string]map[string]string
	json.Unmarshal(data, &msg)
	put := msg["put-entry"]
	f.body["ui-prefs"].(map[string]any)[put["entry-key"]] = put["value"]
	return eyre.PokeResult{Acknowledged: true, CleanupConfirmed: true}, nil
}
func TestPublicationPreservesSettingsAndResends(t *testing.T) {
	f := &fakeSettings{body: map[string]any{"ui-prefs": map[string]any{"themes": `{"themes":[],"activeId":null,"future":"keep"}`, "accent": `{"enabled":true,"mode":"custom","customHex":"#123456","future":17}`, "unrelated": "keep"}}}
	for range 2 {
		if _, err := Publish(context.Background(), f, palette()); err != nil {
			t.Fatal(err)
		}
	}
	if f.pokes != 4 {
		t.Fatal("matching entries were not resent")
	}
	data, _ := f.Scry(context.Background(), "", "")
	entries, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if entries.Themes["future"] != "keep" || entries.Accent["mode"] != "custom" || entries.Accent["customHex"] != "#123456" || f.body["ui-prefs"].(map[string]any)["unrelated"] != "keep" {
		t.Fatal("unrelated settings changed")
	}
	f.fail = true
	if _, err := Publish(context.Background(), f, palette()); !errors.Is(err, eyre.ErrUncertain) {
		t.Fatal("preexisting matching state falsely confirmed a resend")
	}
}
func TestMalformedThemeDoesNotWrite(t *testing.T) {
	f := &fakeSettings{body: map[string]any{"ui-prefs": map[string]any{"themes": `{"themes":"bad"}`}}}
	if _, err := Publish(context.Background(), f, palette()); !errors.Is(err, ErrMalformed) || f.pokes != 0 {
		t.Fatalf("malformed state overwritten: %v", err)
	}
}

func TestUnknownNumbersAndDuplicateKeys(t *testing.T) {
	raw := json.RawMessage(`{"ui-prefs":{"themes":"{\"themes\":[],\"future\":9007199254740993}"}}`)
	merged, err := Merge(raw, palette())
	if err != nil {
		t.Fatal(err)
	}
	if merged.Themes["future"] != json.Number("9007199254740993") {
		t.Fatal("rounded unrelated settings")
	}
	if _, err := Decode(json.RawMessage(`{"ui-prefs":{"themes":{"themes":[],"themes":[]}}}`)); !errors.Is(err, ErrMalformed) {
		t.Fatal("duplicate settings keys accepted")
	}
}
func TestLegacyPaletteFingerprint(t *testing.T) {
	if got := palette().Fingerprint(); got != "f028f5fd22d52693fa32c76448b966c53f95333ba23e1fe03afce1800c13749a" {
		t.Fatalf("legacy fingerprint changed: %s", got)
	}
}
