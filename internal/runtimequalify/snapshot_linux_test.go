//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func snapshotOptions(args []string) []string {
	var out []string
	for i, arg := range args {
		if arg == "--snap-time" && i+1 < len(args) {
			out = append(out, args[i+1])
		}
	}
	return out
}

func TestSnapshotDefaultAppliesToRuntimeCommandsOnly(t *testing.T) {
	for _, kind := range []string{"boot", "restart", "http-boot", "http-restart", "post-pack", "post-meld", "import-resume", "import-replacement", "epoch-restart-1", "post-prune", "prefix-restart-1", "comet-boot", "comet-resume", "comet-verify", "comet-post-pack", "comet-post-meld", "comet-epoch-restart-1", "comet-prefix-recovered-1"} {
		_, args := command(kind)
		if !reflect.DeepEqual(snapshotOptions(args), []string{"60"}) {
			t.Fatal(kind, args)
		}
	}
	for _, kind := range []string{"probe", "pack", "meld", "roll", "roll-1", "epoch-recovery", "prefix-recovery-1", "comet-pack", "comet-meld", "comet-roll", "comet-roll-1", "comet-prefix-recovery-1"} {
		_, args := command(kind)
		if len(snapshotOptions(args)) != 0 {
			t.Fatal("offline utility received a runtime timer", kind, args)
		}
	}
}

func TestSnapshotPolicyIsJournaledAndStrictlyVerified(t *testing.T) {
	for _, networked := range []bool{false, true} {
		f, state, persist := managedFixture(t)
		if networked {
			f, state, persist = managedCometFixture(t)
		}
		seen := false
		state, _, err := applyManaged(context.Background(), "start", state, func(s ManagedState) error {
			for _, c := range s.Runtime.Containers {
				if c.Kind == "http-boot" || c.Kind == "comet-boot" {
					seen = true
					if c.SnapshotMinutes != 60 {
						t.Fatal("runtime intent lacks the snapshot policy")
					}
				}
			}
			return persist(s)
		}, f.d)
		if err != nil || !seen {
			t.Fatal("new bootstrap", err)
		}
		for _, mutation := range []string{"omit", "change", "duplicate"} {
			f.editInspect = func(kind string, doc map[string]any) {
				if kind == "probe" {
					return
				}
				config := mapAt(doc, "Config")
				raw := config["Cmd"].([]any)
				var args []any
				for i := 0; i < len(raw); i++ {
					if raw[i] == "--snap-time" {
						if mutation == "omit" {
							i++
							continue
						}
						if mutation == "change" {
							args = append(args, "--snap-time", "2")
							i++
							continue
						}
					}
					args = append(args, raw[i])
				}
				if mutation == "duplicate" {
					args = append([]any{"--snap-time", "60"}, args...)
				}
				config["Cmd"] = args
			}
			if _, _, err = observeManaged(context.Background(), state, f.d); err == nil {
				t.Fatal("altered snapshot command accepted", mutation)
			}
		}
	}
}

func TestExistingContainerKeepsItsLaunchContractUntilReplacement(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	// Model an already-running container created without an explicit timer flag.
	for n := range state.Runtime.Containers {
		state.Runtime.Containers[n].SnapshotMinutes = 0
	}
	for _, c := range f.containers {
		config := mapAt(c.doc, "Config")
		raw := config["Cmd"].([]string)
		var args []string
		for i := 0; i < len(raw); i++ {
			if raw[i] == "--snap-time" {
				i++
				continue
			}
			args = append(args, raw[i])
		}
		config["Cmd"] = args
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var reopened ManagedState
	if err = json.Unmarshal(data, &reopened); err != nil {
		t.Fatal(err)
	}
	state, obs, err := observeManaged(context.Background(), reopened, f.d)
	if err != nil || obs.Health != "healthy" {
		t.Fatal("manager upgrade invalidated existing runtime", err)
	}
	for _, c := range state.Runtime.Containers {
		if c.Kind == "http-boot" && c.SnapshotIntervalMinutes() != 2 {
			t.Fatal("wrong effective existing interval")
		}
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" {
		t.Fatal(err)
	}
	for _, c := range state.Runtime.Containers {
		if c.Kind == "http-restart" && c.SnapshotMinutes != 60 {
			t.Fatal("replacement kept the old interval")
		}
	}
	for _, c := range f.containers {
		if !reflect.DeepEqual(snapshotOptions(mapAt(c.doc, "Config")["Cmd"].([]string)), []string{"60"}) {
			t.Fatal("replacement omitted the new default")
		}
	}
}
