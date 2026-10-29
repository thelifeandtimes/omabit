package hostinstall

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"strings"
	"testing"
)

func TestGuidedUpgradeIdentityBeforeHostEffects(t *testing.T) {
	for _, scenario := range []string{"match", "different-runner", "busy", "unavailable", "pending", "changed-target", "foreign-journal", "unknown-stage"} {
		t.Run(scenario, func(t *testing.T) {
			f, p, _ := ownerFixture(t)
			binary := []byte("new runner")
			want := "runner-original"
			probe := func(context.Context, string) (protocol.Hello, bool, error) {
				h := protocol.Hello{RunnerID: want, Mode: "development-managed"}
				if scenario == "different-runner" {
					h.RunnerID = "runner-other"
				}
				if scenario == "unavailable" || scenario == "pending" {
					return h, false, errors.New("down")
				}
				return h, scenario == "busy", nil
			}
			if scenario == "pending" || scenario == "changed-target" || scenario == "foreign-journal" || scenario == "unknown-stage" {
				j := activationJournal{Version: 1, Owner: p.Name, UID: p.UID, RunnerID: want, Token: strings.Repeat("a", 64), Stage: "starting", Target: executableGeneration{SHA256: digestBytes(binary), MinSchema: 11, Schema: 47}}
				if scenario == "changed-target" {
					j.Target.SHA256 = strings.Repeat("b", 64)
				}
				if scenario == "foreign-journal" {
					j.RunnerID = "runner-other"
				}
				if scenario == "unknown-stage" {
					j.Stage = "invented"
				}
				data, _ := json.Marshal(j)
				f.put(activationJournalPath, string(data), 0600)
			}
			before := len(f.calls)
			err := f.i.checkUpgradeIdentity(context.Background(), p, want, binary, probe)
			if (err == nil) != (scenario == "match" || scenario == "pending") || len(f.calls) != before {
				t.Fatal(scenario, err, f.calls)
			}
		})
	}
}

func TestGuidedOwnerUpdateUsesIdentityFenceUnderSetupLock(t *testing.T) {
	f, p, probe := ownerFixture(t)
	first, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("old runner"), p, probe, nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.i.readInstalled(context.Background(), activationJournalPath, 16384)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("new runner"), p, probe, nil, "runner-unrelated"); err == nil {
		t.Fatal("accepted unrelated runner")
	}
	after, _ := f.i.readInstalled(context.Background(), activationJournalPath, 16384)
	if string(after) != string(before) {
		t.Fatal("replaced activation journal for unrelated runner")
	}
	second, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("new runner"), p, probe, nil, first.RunnerID)
	if err != nil || !second.Ready || second.RunnerID != first.RunnerID {
		t.Fatal(second, err)
	}
}
