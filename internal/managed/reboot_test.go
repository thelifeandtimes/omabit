package managed

import (
	"context"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const nextBootID = "00000000-0000-0000-0000-000000000002"

func seedHealthyBoot(t *testing.T, s *store) storedInstance {
	t.Helper()
	v := seedStartup(t, s, "alpha", 18080)
	s.bootID = testBootID
	s.recordHealthyBoot(&v)
	saveStartupFixture(t, s, v)
	return v
}

func TestUncleanResumeRequiresPreviousHealthyBoot(t *testing.T) {
	for _, scenario := range []string{"reboot-255", "reboot-137", "same-boot", "legacy", "known-failure", "intentional-stop", "moved", "partial-import"} {
		t.Run(scenario, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			v := seedHealthyBoot(t, s)
			boot := nextBootID
			exit := 255
			switch scenario {
			case "same-boot":
				boot = testBootID
			case "legacy":
				v.Startup = nil
			case "known-failure":
				v.Public.RecoveryRequired = true
			case "intentional-stop":
				v.Public.DesiredState = "stopped"
			case "moved":
				v.Runtime.Handoff = &protocol.Handoff{Direction: "out", State: "moved"}
			case "partial-import":
				v.Runtime.Storage = nil
				v.Runtime.Initialized = false
			case "reboot-137":
				exit = 137
			}
			saveStartupFixture(t, s, v)
			if err := s.close(); err != nil {
				t.Fatal(err)
			}
			b := &startupBackend{exitCode: exit}
			r, err := newServerWithStartup(dir, b, 5*time.Millisecond, true, boot)
			if err != nil {
				t.Fatal(err)
			}
			want := strings.HasPrefix(scenario, "reboot-")
			if want {
				op := startupOperations(t, r, 1, "succeeded")[0]
				if !strings.Contains(op.Message, "host reboot") {
					t.Fatal(op)
				}
			} else {
				time.Sleep(80 * time.Millisecond)
			}
			if err := r.Close(); err != nil {
				t.Fatal(err)
			}
			b.mu.Lock()
			calls := b.calls
			b.mu.Unlock()
			if want && calls != 1 || !want && calls != 0 {
				t.Fatal("unexpected attempts", calls)
			}
			// A rejected same-boot failure cannot carry authorization into a later reboot.
			if scenario == "same-boot" {
				r, err = newServerWithStartup(dir, b, 5*time.Millisecond, true, nextBootID)
				if err != nil {
					t.Fatal(err)
				}
				time.Sleep(80 * time.Millisecond)
				if err := r.Close(); err != nil {
					t.Fatal(err)
				}
				b.mu.Lock()
				calls = b.calls
				b.mu.Unlock()
				if calls != 0 {
					t.Fatal("same-boot failure became reboot retry")
				}
			}
		})
	}
}

func TestRebootCandidateSurvivesPollingAndManagementRestart(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	seedHealthyBoot(t, s)
	s.bootID = nextBootID
	candidates, err := s.startupCandidates()
	if err != nil || len(candidates) != 1 || !candidates[0].Startup.AllowUnclean {
		t.Fatal(candidates, err)
	}
	b := &startupBackend{exitCode: 255}
	v := candidates[0]
	state, observation, err := b.Observe(context.Background(), v.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.observe(v, state, observation); err != nil {
		t.Fatal(err)
	}
	v, err = loadInstance(s.db, "alpha")
	if err != nil || !v.Public.RecoveryRequired || v.Startup.PendingBootID != nextBootID {
		t.Fatal(v, err)
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	r, err := newServerWithStartup(dir, b, 5*time.Millisecond, true, nextBootID)
	if err != nil {
		t.Fatal(err)
	}
	startupOperations(t, r, 1, "succeeded")
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	v, err = loadInstance(s.db, "alpha")
	if err != nil || v.Startup == nil || v.Startup.HealthyBootID != nextBootID || v.Startup.PendingBootID != "" || v.Runtime.StartupResume != nil {
		t.Fatal(v, err)
	}
}

func TestRebootAttemptDoesNotLoopAfterFailureOrInterruption(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		dir := privateDir(t)
		s, err := testOpenStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		seedHealthyBoot(t, s)
		if err := s.close(); err != nil {
			t.Fatal(err)
		}
		b := &startupBackend{exitCode: 255, fail: true}
		if interrupted {
			b.entered = make(chan struct{}, 1)
			b.release = make(chan struct{})
		}
		r, err := newServerWithStartup(dir, b, 5*time.Millisecond, true, nextBootID)
		if err != nil {
			t.Fatal(err)
		}
		if interrupted {
			select {
			case <-b.entered:
			case <-time.After(3 * time.Second):
				t.Fatal("worker not dispatched")
			}
		} else {
			startupOperations(t, r, 1, "failed")
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		for _, boot := range []string{nextBootID, "00000000-0000-0000-0000-000000000003"} {
			r, err = newServerWithStartup(dir, b, 5*time.Millisecond, true, boot)
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(60 * time.Millisecond)
			if err := r.Close(); err != nil {
				t.Fatal(err)
			}
		}
		b.mu.Lock()
		calls := b.calls
		b.mu.Unlock()
		if calls != 1 {
			t.Fatal("failed reboot attempt repeated", calls)
		}
	}
}

func TestRebootClaimRejectsReusedNameAndStoppedCandidate(t *testing.T) {
	for _, scenario := range []string{"delete", "reuse", "stop"} {
		s := testStore(t)
		seedHealthyBoot(t, s)
		s.bootID = nextBootID
		candidates, err := s.startupCandidates()
		if err != nil || len(candidates) != 1 {
			t.Fatal(candidates, err)
		}
		v := candidates[0]
		b := &startupBackend{exitCode: 255}
		state, observation, err := b.Observe(context.Background(), v.Runtime)
		if err != nil {
			t.Fatal(err)
		}
		switch scenario {
		case "delete":
			if _, err := s.db.Exec("DELETE FROM instances WHERE id='alpha'"); err != nil {
				t.Fatal(err)
			}
		case "reuse":
			replacement := v
			replacement.Runtime.ID = strings.Repeat("a", 32)
			saveStartupFixture(t, s, replacement)
		case "stop":
			if _, _, err := s.accept("operator", "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "stop-reboot", ExpectedRevision: 1}); err != nil {
				t.Fatal(err)
			}
		}
		if _, queued, err := s.claimStartup(v, state, observation, nextBootID); err != nil || queued {
			t.Fatal("stale reboot claim accepted", scenario, err)
		}
	}
}

func TestSchema14UpgradePreservesRecordsWithoutInventingBootEvidence(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	seedStartup(t, s, "alpha", 18080)
	var before string
	if err := s.db.QueryRow("SELECT body FROM instances WHERE id='alpha'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("PRAGMA user_version=14"); err != nil {
		t.Fatal(err)
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	s, err = testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	var after string
	var version int
	if err := s.db.QueryRow("SELECT body FROM instances WHERE id='alpha'").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if before != after || version != 15 {
		t.Fatal("migration changed runtime evidence", version)
	}
}
