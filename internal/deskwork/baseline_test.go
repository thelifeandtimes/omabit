package deskwork

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceStatusRejectsInventedComparisonEvidence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	clay, _ := NewManifest([]File{{Path: "hello.txt", SHA256: strings.Repeat("a", 64)}})
	s, err := InspectSource(context.Background(), &sourceReaderFixture{clay: clay, scenario: "different"}, dir, "example", nil)
	if err != nil || s.Validate(nil) != nil {
		t.Fatal(s, err, s.Validate(nil))
	}
	for _, change := range []func(*SourceStatus){
		func(s *SourceStatus) { s.Comparison.State = "synchronized" },
		func(s *SourceStatus) { s.Differences = []string{} },
		func(s *SourceStatus) { s.Autocommit = "disabled" },
		func(s *SourceStatus) { s.Observation = Observation{} },
		func(s *SourceStatus) { s.Clay = s.Unix },
		func(s *SourceStatus) {
			s.CanonicalUnix, _ = NewManifest([]File{{Path: "other.txt", SHA256: strings.Repeat("b", 64)}})
		},
	} {
		var copy SourceStatus
		body, _ := json.Marshal(s)
		json.Unmarshal(body, &copy)
		change(&copy)
		if copy.Validate(nil) == nil {
			t.Fatal("invented comparison accepted", copy)
		}
	}
	if s.Validate(&clay) == nil {
		t.Fatal("unclassified result accepted against a known baseline")
	}
}
