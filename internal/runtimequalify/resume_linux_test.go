//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestProbeRequiresSignalForwardingInit(t *testing.T) {
	for _, target := range []string{"probe", "boot"} {
		t.Run(target, func(t *testing.T) {
			f := fixture(t)
			f.editInspect = func(kind string, doc map[string]any) {
				if kind != target {
					return
				}
				if kind == "probe" {
					delete(mapAt(doc, "HostConfig"), "Init")
				} else {
					mapAt(doc, "HostConfig")["Init"] = true
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete {
				t.Fatalf("unexpected init topology accepted: %+v %v", r, err)
			}
			for _, event := range f.events {
				if event == "start-"+target {
					t.Fatalf("started %s with wrong PID-1 topology", target)
				}
			}
		})
	}
}

func TestManagedResumeUsesSamePierWithoutBootstrap(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	directory := state.Runtime.Directory
	canary := filepath.Join(managedPier(f, state), ".urb/persistence-test")
	if err := os.WriteFile(canary, []byte("existing persistent state"), 0600); err != nil {
		t.Fatal(err)
	}
	f.d.download = func(context.Context, string) (candidate.Artifact, error) {
		t.Fatal("resume requested a bootstrap pill")
		return candidate.Artifact{}, errors.New("unexpected")
	}
	start := len(f.calls)
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.State != "running" || state.Runtime.Directory != directory {
		t.Fatalf("resume: %+v %v", obs, err)
	}
	resumes, probes := 0, 0
	for _, args := range f.calls[start:] {
		if len(args) < 2 || args[0] != "container" || args[1] != "create" {
			continue
		}
		entry := ""
		imageAt := -1
		for i, arg := range args {
			if arg == "--entrypoint" && i+1 < len(args) {
				entry = args[i+1]
			}
			if arg == state.Runtime.ImageID {
				imageAt = i
			}
		}
		if imageAt < 0 {
			t.Fatal("missing pinned image")
		}
		if entry == "/bin/sleep" {
			probes++
			foundInit := false
			for _, arg := range args[:imageAt] {
				foundInit = foundInit || arg == "--init"
			}
			if !foundInit {
				t.Fatal("probe lacks an init to forward SIGTERM")
			}
			continue
		}
		resumes++
		want := []string{"-t", "--snap-time", "60", "-L", "--no-conn", "--no-dock", "--loom", "31", "-b", "0.0.0.0", "--http-port", "8080", "/urbit/pier"}
		if entry != "/bin/urbit" || !reflect.DeepEqual(args[imageAt+1:], want) {
			t.Fatalf("resume must open the existing pier, not initialize: %q", args)
		}
		mountFound := false
		for _, arg := range args[:imageAt] {
			if arg == "type=bind,src="+filepath.Dir(managedPier(f, state))+",dst=/urbit,bind-propagation=rprivate" {
				mountFound = true
			}
			if strings.Contains(arg, "boot.pill") || arg == "--init" {
				t.Fatalf("unexpected resume topology/bootstrap artifact: %q", arg)
			}
		}
		if !mountFound {
			t.Fatal("resume changed the pier mount")
		}
	}
	if resumes != 1 || probes != 1 {
		t.Fatalf("unexpected execution: resumes=%d probes=%d", resumes, probes)
	}
	data, err := os.ReadFile(canary)
	if err != nil || string(data) != "existing persistent state" {
		t.Fatal("existing pier state was replaced")
	}
}
