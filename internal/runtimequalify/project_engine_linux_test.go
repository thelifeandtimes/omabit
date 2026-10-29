//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestProjectDockerAdapterUsesExactIDsAndJournalOrdering(t *testing.T) {
	for _, mode := range []string{"complete", "lost-create-reply", "id-with-client-error", "id-with-cancellation", "malformed-id-error", "engine-replaced", "anchor-volume", "foreign-attachment"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f, _, _ := managedFixture(t)
			contract, n, v := resourceContractFixture(t)
			retained := contract.Preparation
			f.engine.ID = retained.EngineID
			q := &qualifier{d: f.d, base: filepath.Join(f.d.home, ".local/share/urbit-runner/managed"), phase: "project-preparation"}
			q.r.Directory = filepath.Join(q.base, "project-fixture")
			q.r.ImageID = contract.ImageID
			q.config = filepath.Join(q.r.Directory, "docker-config")
			if err := os.MkdirAll(q.config, 0700); err != nil {
				t.Fatal(err)
			}
			var calls [][]string
			q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if _, ok := ctx.Deadline(); !ok || len(args) < 6 || args[0] != "--host" || args[1] != "unix://"+q.socket || args[2] != "--config" || args[3] != q.config {
					t.Fatal("unbounded or misdirected Docker command", args)
				}
				actual := append([]string(nil), args[4:]...)
				calls = append(calls, actual)
				switch {
				case reflect.DeepEqual(actual, contract.bridgeArgs()):
					if retained.Bridge.Phase != "dispatched" {
						t.Fatal("bridge create preceded intent")
					}
					if mode == "lost-create-reply" {
						return "", errors.New("synthetic client disconnect")
					}
					if mode == "id-with-client-error" {
						return n.ID + "\n", errors.New("synthetic client failure after ID")
					}
					if mode == "id-with-cancellation" {
						cancel()
						return n.ID + "\n", nil
					}
					if mode == "malformed-id-error" {
						return n.ID + " unexpected output", errors.New("synthetic client failure")
					}
					if mode == "engine-replaced" {
						f.engine.ID = "other-engine"
					}
					return n.ID + "\n", nil
				case reflect.DeepEqual(actual, contract.anchorArgs(n.ID)):
					if retained.Anchor.Phase != "dispatched" || retained.Bridge.Phase != "observed" {
						t.Fatal("anchor create preceded intent/bridge verification")
					}
					return v.ID + "\n", nil
				case reflect.DeepEqual(actual, []string{"network", "inspect", n.ID}):
					if retained.Bridge.ID != n.ID || retained.Bridge.Phase != "created" && retained.Bridge.Phase != "observed" {
						t.Fatal("bridge inspected before exact ID committed")
					}
					if mode == "foreign-attachment" {
						n.Containers[strings.Repeat("e", 64)] = json.RawMessage(`{}`)
					}
					data, _ := json.Marshal([]bridge{n})
					return string(data), nil
				case reflect.DeepEqual(actual, []string{"container", "inspect", v.ID}):
					if retained.Anchor.Phase != "created" || retained.Anchor.ID != v.ID {
						t.Fatal("anchor inspected before exact ID committed")
					}
					if mode == "anchor-volume" {
						v.HostConfig.Mounts = nil
					}
					data, _ := json.Marshal([]container{v})
					return string(data), nil
				default:
					t.Fatal("unexpected command or name-based lookup", actual)
					return "", errors.New("unexpected")
				}
			}
			engine := &projectDockerEngine{q: q, contract: contract}
			commit := func(sequence uint64, kind, phase, id string) error {
				next, err := retained.Advance(contract.Selection.Plan, sequence, kind, phase, id)
				if err == nil {
					retained = next
				}
				return err
			}
			got, err := projectnetwork.ExecutePreparation(ctx, contract.Selection.Plan, contract.Preparation, engine, commit)
			if (err == nil) != (mode == "complete") || !reflect.DeepEqual(got, retained) {
				t.Fatal(mode, got, err)
			}
			if mode == "complete" {
				if got.Sequence != 6 || got.Anchor.Phase != "observed" || len(calls) != 6 {
					t.Fatal(got, calls)
				}
			} else {
				if got.Anchor.Phase == "observed" {
					t.Fatal("untrusted resource completed")
				}
				if mode == "anchor-volume" && (got.Anchor.Phase != "created" || got.Anchor.ID != v.ID) {
					t.Fatal("untrusted mount lost anchor ID")
				}
				if mode == "engine-replaced" && (got.Bridge.Phase != "created" || len(calls) != 1) {
					t.Fatal("replacement daemon received follow-up commands", got, calls)
				}
				if (mode == "id-with-client-error" || mode == "id-with-cancellation") && (got.Bridge.Phase != "created" || got.Bridge.ID != n.ID || len(calls) != 1) {
					t.Fatal("valid returned ID was lost or followed by inspection", got, calls)
				}
				if (mode == "lost-create-reply" || mode == "malformed-id-error") && (got.Bridge.Phase != "dispatched" || len(calls) != 1) {
					t.Fatal(got, calls)
				}
			}
			count := len(calls)
			if _, err = projectnetwork.ExecutePreparation(context.Background(), contract.Selection.Plan, retained, engine, commit); err == nil || len(calls) != count {
				t.Fatal("adapter replayed prior preparation")
			}
		})
	}
}
