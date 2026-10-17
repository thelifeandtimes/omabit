package managed

// Atomic completion/crash tests adapted from wundrellama's PR #3.
import (
	"context"
	"errors"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func completedRecovery(v storedInstance) runtimequalify.ManagedState {
	state := v.Runtime
	adoption, intent := *state.Adoption, *state.ExternalImport
	state.Adoption, state.ExternalImport = &adoption, &intent
	now := time.Now().UTC()
	state.Initialized = true
	intent.Stage = "complete"
	adoption.Uncertain, adoption.FinishedAt = false, &now
	state.Runtime.Stage = "stopped"
	return state
}

func TestRecoveryPublication(t *testing.T) {
	for _, outcome := range []string{"crash-before-finish", "terminal-write-failure", "backend-error", "canceled-before-finish", "success"} {
		t.Run(outcome, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { s.close() }()
			seedRecovery(t, s)
			req := protocol.OperationRequest{Kind: "recover-import", RequestID: "recover", ExpectedRevision: 1}
			op, _, err := s.accept("operator", "one", req)
			if err != nil {
				t.Fatal(err)
			}
			op, v, err := s.beginOperation(op.ID)
			if err != nil {
				t.Fatal(err)
			}
			completed := completedRecovery(v)
			if err = s.runtimeUpdate("one", op.ID, completed); err != nil {
				t.Fatal(err)
			}
			stopped := runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}
			switch outcome {
			case "terminal-write-failure":
				if _, err = s.db.Exec(`CREATE TRIGGER fail_recovery_receipt BEFORE UPDATE OF body ON operations WHEN json_extract(NEW.body,'$.status')='succeeded' BEGIN SELECT RAISE(ABORT,'injected terminal journal failure'); END`); err != nil {
					t.Fatal(err)
				}
				if err = s.finish(op, completed, stopped, nil, false); err == nil {
					t.Fatal("terminal failure not injected")
				}
				if _, err = s.db.Exec(`DROP TRIGGER fail_recovery_receipt`); err != nil {
					t.Fatal(err)
				}
			case "backend-error":
				if err = s.finish(op, completed, stopped, errors.New("late validation failure"), false); err != nil {
					t.Fatal(err)
				}
			case "canceled-before-finish":
				if err = s.finish(op, completed, stopped, nil, true); err != nil {
					t.Fatal(err)
				}
			case "success":
				if err = s.finish(op, completed, stopped, nil, false); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			s, err = testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			receipt, replayed, err := s.accept("operator", "one", req)
			if err != nil || !replayed || receipt.ID != op.ID {
				t.Fatal("lost replay", receipt, err)
			}
			got, err := loadInstance(s.db, "one")
			if err != nil {
				t.Fatal(err)
			}
			if outcome == "success" {
				if receipt.Status != "succeeded" || got.Runtime.UncertainProvisioning() || !got.Runtime.Initialized || got.Public.RecoveryRequired || got.Public.ObservedState != "stopped" || got.Public.DesiredState != "stopped" || !got.Public.IdentityPending || got.Public.ActiveOperationID != "" {
					t.Fatal("completion not atomic", receipt, got.Public)
				}
				return
			}
			want := "interrupted"
			if outcome == "backend-error" {
				want = "failed"
			}
			if receipt.Status != want || !got.Runtime.ImportRecoveryEligible() || !got.Public.RecoveryRequired {
				t.Fatal("completion escaped without success receipt", receipt, got.Public)
			}
			state, obs, err := runtimequalify.ObserveManaged(context.Background(), got.Runtime)
			if err != nil || !obs.RecoveryRequired {
				t.Fatal("poll lost fence", obs, err)
			}
			if err = s.observe(got, state, obs); err != nil {
				t.Fatal(err)
			}
			stop, _, err := s.accept("operator", "one", protocol.OperationRequest{Kind: "stop", RequestID: "stop", ExpectedRevision: 1})
			if err != nil {
				t.Fatal(err)
			}
			stop, got, err = s.beginOperation(stop.ID)
			if err != nil {
				t.Fatal(err)
			}
			stopped.RecoveryRequired = got.Runtime.UncertainProvisioning()
			if err = s.finish(stop, got.Runtime, stopped, nil, false); err != nil {
				t.Fatal(err)
			}
			got, err = loadInstance(s.db, "one")
			if err != nil || !got.Runtime.ImportRecoveryEligible() || !got.Public.RecoveryRequired {
				t.Fatal("Stop cleared fence", got.Public, err)
			}
			_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
			requireFault(t, err, "recovery_required")
			again, replayed, err := s.accept("operator", "one", req)
			if err != nil || !replayed || again.Status != want {
				t.Fatal("poll/Stop changed receipt", again, err)
			}
		})
	}
}

func TestRecoveryRefusesIncompleteSuccess(t *testing.T) {
	s := testStore(t)
	seedRecovery(t, s)
	op, _, err := s.accept("operator", "one", protocol.OperationRequest{Kind: "recover-import", RequestID: "recover", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	op, v, err := s.beginOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "stopped"}, nil, false); err == nil {
		t.Fatal("incomplete state published as success")
	}
	still, err := loadInstance(s.db, "one")
	if err != nil || !still.Runtime.ImportRecoveryEligible() || still.Public.ActiveOperationID != op.ID {
		t.Fatal("failed completion lost ownership/fence", err)
	}
}
