package omabittheme

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

type Transport interface {
	Scry(context.Context, string, string) (json.RawMessage, error)
	Poke(context.Context, string, string, json.RawMessage) (eyre.PokeResult, error)
}

var ErrMalformed = errors.New("existing Talon settings are malformed; nothing was replaced")

type Entries struct {
	Themes map[string]any
	Accent map[string]any
}

func Decode(raw json.RawMessage) (Entries, error) {
	value, err := strictJSON(raw)
	body, ok := value.(map[string]any)
	if err != nil || !ok || body == nil {
		return Entries{}, ErrMalformed
	}
	if nested, ok := body["desk"]; ok {
		var good bool
		body, good = nested.(map[string]any)
		if !good {
			return Entries{}, ErrMalformed
		}
	}
	for _, bucket := range body {
		if _, ok := bucket.(map[string]any); !ok {
			return Entries{}, ErrMalformed
		}
	}
	bucket := map[string]any{}
	if value, ok := body["ui-prefs"]; ok {
		bucket = value.(map[string]any)
	}
	entry := func(key string) (map[string]any, error) {
		value, ok := bucket[key]
		if !ok {
			return map[string]any{}, nil
		}
		if text, ok := value.(string); ok {
			var err error
			value, err = strictJSON([]byte(text))
			if err != nil {
				return nil, ErrMalformed
			}
		}
		out, ok := value.(map[string]any)
		if !ok || out == nil {
			return nil, ErrMalformed
		}
		return out, nil
	}
	themes, err := entry("themes")
	if err != nil {
		return Entries{}, err
	}
	accent, err := entry("accent")
	if err != nil {
		return Entries{}, err
	}
	if id, ok := themes["activeId"]; ok && id != nil {
		if _, ok := id.(string); !ok {
			return Entries{}, ErrMalformed
		}
	}
	if value, ok := themes["themes"]; ok {
		list, ok := value.([]any)
		if !ok {
			return Entries{}, ErrMalformed
		}
		ids := map[string]bool{}
		for _, value := range list {
			theme, ok := value.(map[string]any)
			if !ok {
				return Entries{}, ErrMalformed
			}
			id, ok := theme["id"].(string)
			if !ok || id == "" || ids[id] {
				return Entries{}, ErrMalformed
			}
			ids[id] = true
			if name, ok := theme["name"].(string); !ok || strings.TrimSpace(name) == "" {
				return Entries{}, ErrMalformed
			}
			if _, ok := theme["dark"].(bool); !ok {
				return Entries{}, ErrMalformed
			}
			for _, key := range []string{"primary", "secondary", "tertiary", "background", "surface"} {
				color, ok := theme[key].(string)
				if !ok || !colorPattern.MatchString("#"+strings.TrimPrefix(strings.TrimSpace(color), "#")) {
					return Entries{}, ErrMalformed
				}
			}
		}
	}
	if enabled, ok := accent["enabled"]; ok && enabled != nil {
		if _, ok := enabled.(bool); !ok {
			return Entries{}, ErrMalformed
		}
	}
	for _, key := range []string{"mode", "customHex"} {
		if value, ok := accent[key]; ok && value != nil {
			if _, ok := value.(string); !ok {
				return Entries{}, ErrMalformed
			}
		}
	}
	return Entries{Themes: themes, Accent: accent}, nil
}

func Merge(raw json.RawMessage, p Palette) (Entries, error) {
	if err := p.Validate(); err != nil {
		return Entries{}, err
	}
	entries, err := Decode(raw)
	if err != nil {
		return entries, err
	}
	data, _ := json.Marshal(p)
	var palette map[string]any
	json.Unmarshal(data, &palette)
	list, _ := entries.Themes["themes"].([]any)
	found := false
	for _, value := range list {
		theme := value.(map[string]any)
		if theme["id"] == ThemeID {
			for k, v := range palette {
				theme[k] = v
			}
			found = true
		}
	}
	if !found {
		list = append(list, palette)
	}
	entries.Themes["themes"], entries.Themes["activeId"] = list, ThemeID
	entries.Accent["enabled"] = false
	return entries, nil
}

// Publish preserves unrelated entry fields and themes. Each real publication
// resends both entries and requires Gall acknowledgements plus readback. Matching
// preexisting state never proves an unacknowledged resend was delivered.
func Publish(ctx context.Context, c Transport, p Palette) (bool, error) {
	raw, err := c.Scry(ctx, "settings", "/desk/talon")
	if err != nil {
		return false, err
	}
	desired, err := Merge(raw, p)
	if err != nil {
		return false, err
	}
	cleanup := true
	for _, key := range []string{"themes", "accent"} {
		if key == "accent" {
			raw, err = c.Scry(ctx, "settings", "/desk/talon")
			if err != nil {
				return cleanup, err
			}
			current, e := Decode(raw)
			if e != nil {
				return cleanup, e
			}
			desired.Accent = current.Accent
			desired.Accent["enabled"] = false
		}
		value := desired.Themes
		if key == "accent" {
			value = desired.Accent
		}
		encoded, _ := json.Marshal(value)
		payload, _ := json.Marshal(map[string]any{"put-entry": map[string]any{"desk": "talon", "bucket-key": "ui-prefs", "entry-key": key, "value": string(encoded)}})
		result, err := c.Poke(ctx, "settings", "settings-event", payload)
		cleanup = cleanup && result.CleanupConfirmed
		if err != nil {
			return cleanup, err
		}
		if !result.Acknowledged {
			return cleanup, eyre.ErrUncertain
		}
		readback, err := c.Scry(ctx, "settings", "/desk/talon")
		if err != nil {
			return cleanup, err
		}
		actual, err := Decode(readback)
		if err != nil {
			return cleanup, err
		}
		got := actual.Themes
		if key == "accent" {
			got = actual.Accent
		}
		if !reflect.DeepEqual(got, value) {
			return cleanup, errors.New("Talon settings changed during publication; reconciliation required")
		}
	}
	raw, err = c.Scry(ctx, "settings", "/desk/talon")
	if err != nil {
		return cleanup, err
	}
	actual, err := Decode(raw)
	if err != nil {
		return cleanup, err
	}
	if !reflect.DeepEqual(actual, desired) {
		return cleanup, errors.New("Talon settings changed during publication; reconciliation required")
	}
	return cleanup, nil
}
