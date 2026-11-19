//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordingSessionAllocationSurvivesClaimClosureWithoutInventingCapture(t *testing.T) {
	s, root, job, m, worker := recordingJobsFixture(t, 8)
	digest, err := s.Create(job)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadSession(job.ID, digest); !errors.Is(err, ErrRecordingSessionUnknown) {
		t.Fatal("missing allocation fabricated", err)
	}
	lease, err := s.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	if err := lease.AllocateSession(m.SessionID, at); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, job.ID, "session.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.AllocateSession(strings.Repeat("3", 32), at); err == nil {
		t.Fatal("allocated session replaced")
	}
	lease.Close()
	for pass := 0; pass < 2; pass++ {
		allocation, err := s.ReadSession(job.ID, digest)
		if err != nil || allocation.SessionID != m.SessionID || !allocation.AllocatedAt.Equal(at) || allocation.JobSHA256 != digest {
			t.Fatal("saved allocation lost binding", err)
		}
		if _, err := s.ReadReady(job.ID, digest); !errors.Is(err, ErrRecordingJobNotReady) {
			t.Fatal("allocation became capture/readiness", err)
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			t.Fatal("lookup rewrote allocation")
		}
	}
}

func TestRecordingSessionAllocationRefusesChangedClaimAndMalformedOrReboundFiles(t *testing.T) {
	for _, scenario := range []string{"claim", "partial", "unknown-field", "session-binding", "hardlink"} {
		t.Run(scenario, func(t *testing.T) {
			s, root, job, m, worker := recordingJobsFixture(t, 8)
			digest, err := s.Create(job)
			if err != nil {
				t.Fatal(err)
			}
			lease, err := s.Claim(job.ID, digest, worker)
			if err != nil {
				t.Fatal(err)
			}
			defer lease.Close()
			if err := lease.AllocateSession(m.SessionID, time.Now()); err != nil {
				t.Fatal(err)
			}
			if err := lease.Ready(m); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, job.ID, "session.json")
			raw, _ := os.ReadFile(path)
			switch scenario {
			case "claim":
				claimPath := filepath.Join(root, job.ID, "claim.json")
				data, _ := os.ReadFile(claimPath)
				var claim recordingJobClaim
				json.Unmarshal(data, &claim)
				claim.Worker.StartTicks++
				data, _ = json.Marshal(claim)
				if err := os.WriteFile(claimPath, data, 0600); err != nil {
					t.Fatal(err)
				}
			case "partial":
				raw = []byte("partial")
			case "unknown-field":
				raw = append(append(bytes.Clone(raw[:len(raw)-1]), []byte(`,"unknown":true`)...), '}')
			case "session-binding":
				var allocation RecordingJobSession
				json.Unmarshal(raw, &allocation)
				allocation.SessionID = strings.Repeat("3", 32)
				raw, _ = json.Marshal(allocation)
			case "hardlink":
				if err := os.Link(path, filepath.Join(root, "retained-allocation")); err != nil {
					t.Fatal(err)
				}
			}
			if scenario != "claim" && scenario != "hardlink" {
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadFile(path)
			if _, err := s.ReadReady(job.ID, digest); err == nil {
				t.Fatal("changed allocation/claim gained readiness", scenario)
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("lookup repaired malformed original")
			}
		})
	}
}
