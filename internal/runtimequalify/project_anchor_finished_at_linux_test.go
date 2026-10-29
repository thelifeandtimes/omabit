//go:build linux && amd64

package runtimequalify

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestRecoveryAnchorRetainsHistoricalFinish(t *testing.T) {
	c, n, v := resourceContractFixture(t)
	runningAnchorFixture(&v, &n)
	start := time.Now().UTC().Add(-time.Second)
	v.State.StartedAt = start.Format(time.RFC3339Nano)
	v.State.FinishedAt = start.Add(-time.Second).Format(time.RFC3339Nano)
	if c.validateAnchor(v, v.ID, n.ID, true) == nil {
		t.Fatal("initial Start accepted restart history")
	}
	if err := c.validateRecoveryAnchor(v, v.ID, n.ID); err != nil {
		t.Fatal("Docker restart retains its earlier FinishedAt", err)
	}
	for _, finish := range []string{start.Format(time.RFC3339Nano), start.Add(time.Second).Format(time.RFC3339Nano), "malformed"} {
		v.State.FinishedAt = finish
		if c.validateRecoveryAnchor(v, v.ID, n.ID) == nil {
			t.Fatal("invalid running anchor finish accepted", finish)
		}
	}
	v.State.FinishedAt = "0001-01-01T00:00:00Z"
	if c.validateRecoveryAnchor(v, v.ID, n.ID) != nil {
		t.Fatal("initial running state no longer observable")
	}
}

// Reads only a supplied retained guest report. It invokes no Docker, filesystem
// ownership adapter, live ship, or user credential path.
func TestProjectRecordedRestartedAnchorRecovery(t *testing.T) {
	path := os.Getenv("OMABIT_TEST_ANCHOR_REVIEW_RECEIPT")
	if path == "" {
		t.Skip("requires exact retained disposable review receipt")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 2<<20 {
		t.Fatal("missing/excessive retained receipt", err)
	}
	var report struct {
		Members []struct {
			Review *struct {
				Selection     ProjectAnchorRecoveryReviewSelection `json:"selection"`
				Restarted     []container                          `json:"restarted_inspection"`
				StoppedReview json.RawMessage                      `json:"stopped_review"`
			} `json:"anchor_recovery_review"`
		} `json:"member_launch"`
	}
	if err = json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range report.Members {
		if m.Review == nil {
			continue
		}
		if found {
			t.Fatal("duplicate review")
		}
		found = true
		r := m.Review
		if len(r.Restarted) != 1 || len(r.Selection.Launches) != 2 {
			t.Fatal("missing recorded selection")
		}
		v := r.Restarted[0]
		p := r.Selection.Launches[0].Anchor.Prepared
		c, err := newProjectResourceContract(r.Selection.Project, p.EngineID, v.Image)
		if err != nil {
			t.Fatal(err)
		}
		if c.validateAnchor(v, v.ID, p.Bridge.ID, true) == nil {
			t.Fatal("initial validator lost strict initial-Start semantics")
		}
		if err = c.validateRecoveryAnchor(v, v.ID, p.Bridge.ID); err != nil {
			t.Fatal("retained restarted Docker anchor rejected", err)
		}
	}
	if !found {
		t.Fatal("recorded review absent")
	}
}
