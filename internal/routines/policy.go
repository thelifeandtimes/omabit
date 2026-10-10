// Package routines defines explicit UTC maintenance policy and durable run intent.
// Calendar decisions have no filesystem, network or runtime side effects.
package routines

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"time"
)

type Config struct {
	Enabled       bool   `json:"enabled"`
	Action        string `json:"action"`
	Weekdays      []int  `json:"weekdays"`
	HourUTC       int    `json:"hour_utc"`
	MinuteUTC     int    `json:"minute_utc"`
	WindowMinutes int    `json:"window_minutes"`
	StopRunning   bool   `json:"stop_running"`
	Resume        bool   `json:"resume"`
}

type Policy struct {
	ID               string     `json:"id"`
	InstanceID       string     `json:"instance_id"`
	Revision         uint64     `json:"revision"`
	InstanceRevision uint64     `json:"instance_revision"`
	Owner            string     `json:"owner"`
	Config           Config     `json:"config"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastSlot         *time.Time `json:"last_slot,omitempty"`
	LastOperationID  string     `json:"last_operation_id,omitempty"`
	NextSlot         *time.Time `json:"next_slot,omitempty"`
}

type Update struct {
	ExpectedRevision         uint64 `json:"expected_revision"`
	ExpectedInstanceRevision uint64 `json:"expected_instance_revision"`
	Config                   Config `json:"config"`
}

// Consent and calendar defaults must not be inferred from omitted JSON fields.
func (u *Update) UnmarshalJSON(data []byte) error {
	var wire struct {
		Revision         *uint64 `json:"expected_revision"`
		InstanceRevision *uint64 `json:"expected_instance_revision"`
		Config           *struct {
			Enabled       *bool   `json:"enabled"`
			Action        *string `json:"action"`
			Weekdays      *[]int  `json:"weekdays"`
			HourUTC       *int    `json:"hour_utc"`
			MinuteUTC     *int    `json:"minute_utc"`
			WindowMinutes *int    `json:"window_minutes"`
			StopRunning   *bool   `json:"stop_running"`
			Resume        *bool   `json:"resume"`
		} `json:"config"`
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&wire); err != nil {
		return errors.New("invalid routine update schema")
	}
	c := wire.Config
	if wire.Revision == nil || wire.InstanceRevision == nil || c == nil || c.Enabled == nil || c.Action == nil || c.Weekdays == nil || c.HourUTC == nil || c.MinuteUTC == nil || c.WindowMinutes == nil || c.StopRunning == nil || c.Resume == nil {
		return errors.New("routine update requires explicit revisions and every timing/consent field")
	}
	*u = Update{ExpectedRevision: *wire.Revision, ExpectedInstanceRevision: *wire.InstanceRevision, Config: Config{Enabled: *c.Enabled, Action: *c.Action, Weekdays: *c.Weekdays, HourUTC: *c.HourUTC, MinuteUTC: *c.MinuteUTC, WindowMinutes: *c.WindowMinutes, StopRunning: *c.StopRunning, Resume: *c.Resume}}
	return nil
}

type Run struct {
	PolicyID       string     `json:"policy_id"`
	PolicyRevision uint64     `json:"policy_revision"`
	Slot           time.Time  `json:"slot"`
	Deadline       time.Time  `json:"deadline"`
	Action         string     `json:"action"`
	StopRunning    bool       `json:"stop_running"`
	Resume         bool       `json:"resume"`
	WasRunning     bool       `json:"was_running"`
	Stage          string     `json:"stage"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	Succeeded      bool       `json:"succeeded"`
}

func Validate(c *Config) error {
	if c.Action != "pack" && c.Action != "meld" {
		return errors.New("routine action must be pack or meld")
	}
	if len(c.Weekdays) < 1 || len(c.Weekdays) > 7 || c.HourUTC < 0 || c.HourUTC > 23 || c.MinuteUTC < 0 || c.MinuteUTC > 59 || c.WindowMinutes < 5 || c.WindowMinutes > 180 || c.Resume && !c.StopRunning {
		return errors.New("require UTC weekdays/time, a 5..180 minute dispatch window, and explicit downtime consent before resume")
	}
	seen := map[int]bool{}
	for _, d := range c.Weekdays {
		if d < 0 || d > 6 || seen[d] {
			return errors.New("weekdays must be unique values 0 (Sunday) through 6 (Saturday)")
		}
		seen[d] = true
	}
	c.Weekdays = append([]int(nil), c.Weekdays...)
	sort.Ints(c.Weekdays)
	return nil
}

func onDay(c Config, day time.Weekday) bool {
	for _, d := range c.Weekdays {
		if d == int(day) {
			return true
		}
	}
	return false
}

// Due returns the most recent slot only. Old windows are recorded as skipped;
// there is no catch-up queue, and backwards clock movement cannot repeat a slot.
func Due(p Policy, now time.Time) (time.Time, bool) {
	if !p.Config.Enabled {
		return time.Time{}, false
	}
	now = now.UTC()
	for day := 0; day <= 7; day++ {
		d := now.AddDate(0, 0, -day)
		if !onDay(p.Config, d.Weekday()) {
			continue
		}
		slot := time.Date(d.Year(), d.Month(), d.Day(), p.Config.HourUTC, p.Config.MinuteUTC, 0, 0, time.UTC)
		if slot.After(now) {
			continue
		}
		if !slot.After(p.UpdatedAt) || p.LastSlot != nil && !slot.After(*p.LastSlot) {
			return time.Time{}, false
		}
		return slot, true
	}
	return time.Time{}, false
}

func Next(p Policy, now time.Time) *time.Time {
	if !p.Config.Enabled {
		return nil
	}
	now = now.UTC()
	if p.UpdatedAt.After(now) {
		now = p.UpdatedAt
	}
	if p.LastSlot != nil && p.LastSlot.After(now) {
		now = *p.LastSlot
	}
	for day := 0; day <= 7; day++ {
		d := now.AddDate(0, 0, day)
		if !onDay(p.Config, d.Weekday()) {
			continue
		}
		slot := time.Date(d.Year(), d.Month(), d.Day(), p.Config.HourUTC, p.Config.MinuteUTC, 0, 0, time.UTC)
		if slot.After(now) {
			return &slot
		}
	}
	return nil
}
