package routines

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUpdateRequiresExplicitFalseAndZeroFields(t *testing.T) {
	good := Update{ExpectedInstanceRevision: 1, Config: Config{Action: "pack", Weekdays: []int{0}, WindowMinutes: 5}}
	data, err := json.Marshal(good)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Update
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal("explicit zero/false rejected", err)
	}
	for _, field := range []string{"enabled", "hour_utc", "minute_utc", "stop_running", "resume"} {
		var value map[string]any
		json.Unmarshal(data, &value)
		delete(value["config"].(map[string]any), field)
		bad, _ := json.Marshal(value)
		if json.Unmarshal(bad, &decoded) == nil {
			t.Fatal("omitted policy field accepted", field)
		}
	}
}

func TestUTCSlotsNoCatchupDuplicatesOrRetroactiveEnable(t *testing.T) {
	now := time.Date(2026, 9, 29, 23, 59, 0, 0, time.UTC)
	p := Policy{Config: Config{Enabled: true, Action: "pack", Weekdays: []int{2}, HourUTC: 23, MinuteUTC: 59, WindowMinutes: 15}, UpdatedAt: now.Add(-time.Hour)}
	if err := Validate(&p.Config); err != nil {
		t.Fatal(err)
	}
	slot, due := Due(p, now.Add(2*time.Minute))
	if !due || !slot.Equal(now) {
		t.Fatal("cross-midnight dispatch window lost")
	}
	p.LastSlot = &slot
	if _, due := Due(p, now.Add(-time.Minute)); due {
		t.Fatal("clock reversal repeated slot")
	}
	if _, due := Due(p, now.Add(3*time.Minute)); due {
		t.Fatal("same slot repeated")
	}
	next := Next(p, now)
	if next == nil || !next.Equal(now.AddDate(0, 0, 7)) {
		t.Fatal("wrong next UTC slot")
	}
	p.LastSlot = nil
	p.UpdatedAt = now.Add(time.Second)
	if _, due := Due(p, now.Add(time.Minute)); due {
		t.Fatal("enable caught up past slot")
	}
	p.Config.Enabled = false
	if Next(p, now) != nil {
		t.Fatal("disabled schedule has next slot")
	}
}

func TestPolicyRejectsUnqualifiedOrAmbiguousConfiguration(t *testing.T) {
	good := Config{Action: "meld", Weekdays: []int{1}, HourUTC: 2, MinuteUTC: 30, WindowMinutes: 15}
	for _, change := range []func(*Config){func(c *Config) { c.Action = "prune-epochs" }, func(c *Config) { c.Weekdays = []int{1, 1} }, func(c *Config) { c.Weekdays = []int{7} }, func(c *Config) { c.HourUTC = 24 }, func(c *Config) { c.MinuteUTC = 60 }, func(c *Config) { c.WindowMinutes = 0 }, func(c *Config) { c.Resume = true }} {
		c := good
		change(&c)
		if Validate(&c) == nil {
			t.Fatal("invalid policy accepted", c)
		}
	}
}
