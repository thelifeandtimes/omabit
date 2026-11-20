//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestProjectShipActionDependenciesKeepJournalBeforeSignalAndExactProcess(t *testing.T) {
	for _, mode := range []string{"clean", "save-refused", "changed-start", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			f, input, _ := resumedStopAdapterFixture(t, false, 2)
			s, err := SeedProjectExpansion(ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]})
			if err != nil {
				t.Fatal(err)
			}
			f.f.d.readBootID = func() (string, error) { return f.ns.BootID, nil }
			index := -1
			for i, member := range s.Members {
				if member.Member.InstanceID == input.Stop.InstanceID {
					index = i
				}
			}
			if index < 0 {
				t.Fatal("selected fixture member missing")
			}
			id := s.Members[index].Creator.Container.ID
			started, err := time.Parse(time.RFC3339Nano, f.containers[id].State.StartedAt)
			if err != nil {
				t.Fatal(err)
			}
			peers := map[string][]byte{}
			for cid, c := range f.containers {
				if cid != id {
					peers[cid], _ = json.Marshal(c)
				}
			}
			signalsBefore := f.signals
			var saved []ProjectExpansion
			saveErr := errors.New("synthetic journal refusal")
			save := func(value ProjectExpansion) error {
				if mode == "save-refused" {
					return saveErr
				}
				// Persistence serializes at the callback boundary. Avoid aliases
				// making an earlier durable phase look like the final in-memory one.
				raw, err := json.Marshal(value)
				if err != nil {
					return err
				}
				var copy ProjectExpansion
				if err = json.Unmarshal(raw, &copy); err != nil {
					return err
				}
				saved = append(saved, copy)
				return nil
			}
			originalSignal := f.signal
			f.signal = func(ctx context.Context, selected string) (string, error) {
				if selected != id || len(saved) != 1 || saved[0].Members[index].Phase != "stop-dispatched" {
					t.Fatal("signal preceded durable exact dispatch")
				}
				intent := saved[0].Members[index].StopIntent
				if intent == nil || intent.OperationID != "op-native-timing" || intent.ContainerID != id || !intent.StartedAt.Equal(started) {
					t.Fatal("signal lost exact operation/process binding")
				}
				out, err := originalSignal(ctx, selected)
				if mode == "changed-start" {
					f.containers[id].State.StartedAt = started.Add(time.Second).Format(time.RFC3339Nano)
				}
				return out, err
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			err = runProjectShipActionWithDependencies(ctx, &s, input.Stop.InstanceID, "stop", "op-native-timing", save, f.f.d)
			switch mode {
			case "clean":
				if err != nil || f.signals != signalsBefore+1 || len(saved) != 2 || saved[1].Members[index].Phase != "stopped" || !reflect.DeepEqual(saved[0].Members[index].StopIntent, saved[1].Members[index].StopIntent) {
					t.Fatal("native action lost durable stop order or exact single dispatch", err, f.signals, len(saved))
				}
			case "save-refused":
				if !errors.Is(err, saveErr) || f.signals != signalsBefore || len(saved) != 0 {
					t.Fatal("refused journal allowed a signal", err)
				}
			case "changed-start":
				if err == nil || f.signals != signalsBefore+1 || len(saved) != 1 {
					t.Fatal("replacement process became completion or repeated signal", err)
				}
			case "cancelled":
				if err == nil || f.signals != signalsBefore || len(saved) != 0 {
					t.Fatal("cancelled action dispatched", err)
				}
			}
			for cid, before := range peers {
				after, _ := json.Marshal(f.containers[cid])
				if string(before) != string(after) {
					t.Fatal("unselected peer changed")
				}
			}
		})
	}
}
