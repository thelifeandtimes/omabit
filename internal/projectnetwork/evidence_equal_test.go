package projectnetwork

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestEvidenceEqualityPreservesExactFieldsAcrossTimeMetadata(t *testing.T) {
	type receipt struct {
		At     time.Time
		Values []string
		Owner  string
		Ack    bool
	}
	now := time.Now()
	original := receipt{At: now, Values: []string{"a"}, Owner: "owner", Ack: true}
	for _, zone := range []*time.Location{time.UTC, time.FixedZone("local-zero", 0), time.FixedZone("east", 19800), time.FixedZone("west", -25200)} {
		other := original
		other.At = now.In(zone)
		if !equalEvidence(original, other) {
			t.Fatal("timezone or monotonic metadata changed the instant")
		}
		other.At = other.At.Add(time.Nanosecond)
		if equalEvidence(original, other) {
			t.Fatal("changed instant was accepted")
		}
	}
	for _, change := range []func(*receipt){
		func(r *receipt) { r.Owner = "another" },
		func(r *receipt) { r.Ack = false },
		func(r *receipt) { r.Values = nil },
		func(r *receipt) { r.Values = []string{} },
		func(r *receipt) { r.Values = []string{"b"} },
	} {
		other := original
		change(&other)
		if equalEvidence(original, other) {
			t.Fatal("changed evidence was accepted")
		}
	}
	if equalEvidence([]string(nil), []string{}) || equalEvidence(nil, (*receipt)(nil)) {
		t.Fatal("nil distinctions lost")
	}
}

// Re-express every timestamp in a private fixture copy without changing an
// instant. This exercises real nested journal validation, not only the helper.
func evidenceFixtureZone(v reflect.Value, zone *time.Location) {
	if v.Type() == evidenceTimeType {
		v.Set(reflect.ValueOf(v.Interface().(time.Time).In(zone)))
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			evidenceFixtureZone(v.Elem(), zone)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			evidenceFixtureZone(v.Field(i), zone)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			evidenceFixtureZone(v.Index(i), zone)
		}
	}
}

func TestRecoveryJournalTimezoneRoundTripKeepsExactCheckpoint(t *testing.T) {
	h, ready := journalReadinessFixture(t, false)
	trace := replayRecoveryJournal(t, h, ready, false)
	for _, zone := range []*time.Location{time.FixedZone("local-zero", 0), time.FixedZone("east", 19800), time.FixedZone("west", -25200)} {
		for _, checkpoint := range trace {
			changed := checkpoint.Clone()
			evidenceFixtureZone(reflect.ValueOf(&changed).Elem(), zone)
			body, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			var reopened AnchorRecoveryJournal
			if err = json.Unmarshal(body, &reopened); err != nil {
				t.Fatal(err)
			}
			if err = reopened.Validate(h); err != nil {
				t.Fatal(zone, checkpoint.Phase, err)
			}
			if !RecoveryJournalContainsCheckpoint(h, reopened, checkpoint) || !RecoveryJournalContainsCheckpoint(h, checkpoint, reopened) {
				t.Fatal("same checkpoint lost after timezone round trip", zone, checkpoint.Phase)
			}
		}
	}
}
