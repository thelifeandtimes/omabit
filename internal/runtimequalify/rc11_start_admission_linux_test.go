//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestRC11StartReadmitsAfterPreparationAndRemainsRetryable(t *testing.T) {
	for _, mode := range []string{"bare-image", "legacy-pill", "comet-probe", "owned-probe", "replacement-image", "replacement-probe"} {
		for _, root := range []string{"runtime", "default-pier", "named"} {
			for _, fault := range []string{"RAM", "disk", "unavailable"} {
				t.Run(mode+"/"+root+"/"+fault, func(t *testing.T) {
					fixture := managedFixture
					if mode == "comet-probe" || mode == "owned-probe" {
						fixture = managedCometFixture
					}
					f, state, persist := fixture(t)
					if mode == "bare-image" {
						state.Bootstrap = &protocol.Bootstrap{Kind: "bare"}
					}
					if mode == "owned-probe" {
						state.NativeShip, state.Comet.Ship = "~sampel-palnet", "~sampel-palnet"
						state.Storage.Path = "livenet/planets/sampel-palnet/pier"
						state.Boot, state.AmesPort = &ManagedBoot{}, 34343
						key := "0w" + strings.Repeat("a", 128)
						state.NativeKeyDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
						dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/keys")
						if err := os.Mkdir(dir, 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(dir, state.ID+".key"), []byte(key), 0600); err != nil {
							t.Fatal(err)
						}
						f.d.hostIdentity = func(context.Context, string) (string, error) { return "~sampel-palnet", nil }
					}
					storageRoot := ""
					if root == "named" {
						storageRoot = t.TempDir()
						if _, err := pierstore.RegisterRoot(f.d.home, "ssd", storageRoot); err != nil {
							t.Fatal(err)
						}
						state.Storage.Path = "@ssd/" + state.Storage.Path
					}
					if root == "default-pier" {
						pier, err := pierstore.Absolute(f.d.home, state.Storage.Path)
						if err != nil {
							t.Fatal(err)
						}
						storageRoot = filepath.Dir(pier)
					}
					if strings.HasPrefix(mode, "replacement-") {
						for _, action := range []string{"start", "stop"} {
							var err error
							state, _, err = applyManaged(context.Background(), action, state, persist, f.d)
							if err != nil {
								t.Fatal(err)
							}
						}
					}
					originalID, originalLocation, originalKey := state.ID, *state.Storage, state.NativeKeyDigest
					initialized := state.Initialized
					minimumRAM, minimumDisk := uint64(5<<30), uint64(8<<30)
					if initialized {
						minimumRAM, minimumDisk = 1<<30, 2<<30
					}
					exhausted, late := false, false
					f.d.admission = func(path string) (uint64, uint64, error) {
						if !exhausted || storageRoot != "" && path != storageRoot {
							return 5 << 30, 8 << 30, nil
						}
						late = true
						switch fault {
						case "RAM":
							return minimumRAM - 1, 8 << 30, nil
						case "disk":
							return 5 << 30, minimumDisk - 1, nil
						default:
							return 5 << 30, 8 << 30, errors.New("capacity read failed")
						}
					}
					execute, download := f.d.execute, f.d.download
					switch {
					case strings.HasSuffix(mode, "image"):
						f.editImage = func(map[string]any) { exhausted = true }
					case mode == "legacy-pill":
						f.d.download = func(ctx context.Context, path string) (candidate.Artifact, error) {
							artifact, err := download(ctx, path)
							exhausted = true
							return artifact, err
						}
					default:
						f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
							out, err := execute(ctx, args, env)
							if err == nil && args[4] == "container" && args[5] == "rm" {
								exhausted = true // after the live probe has been removed
							}
							return out, err
						}
					}
					var durable []byte
					save := func(s ManagedState) error {
						if err := persist(s); err != nil {
							return err
						}
						var err error
						durable, err = json.Marshal(s)
						return err
					}
					before := len(f.events)
					state, _, err := applyManaged(context.Background(), "start", state, save, f.d)
					if err == nil || !strings.Contains(err.Error(), "immediately before dispatch") || !exhausted || !late || state.Initialized != initialized || state.UncertainProvisioning() || state.UncertainMaintenance() || len(f.containers) != 0 {
						t.Fatalf("late capacity did not prevent runtime dispatch cleanly: %+v %v (late=%t)", state, err, late)
					}
					for _, event := range f.events[before:] {
						if (strings.HasPrefix(event, "create-") || strings.HasPrefix(event, "start-")) && !strings.HasSuffix(event, "-probe") && event != "create-network" {
							t.Fatal("runtime dispatched with inadequate capacity:", event)
						}
					}
					// Reopen the last durable state. No initial-identity fence or key
					// consumption is allowed until the final admission succeeds.
					state = ManagedState{}
					if err := json.Unmarshal(durable, &state); err != nil {
						t.Fatal(err)
					}
					if state.ID != originalID || state.Storage.Path != originalLocation.Path || state.NativeKeyDigest != originalKey || state.Comet != nil && state.Comet.StartedAt != nil || initialized && *state.Storage != originalLocation {
						t.Fatal("preparation refusal changed identity, key or existing pier")
					}
					f.editImage, f.d.execute, f.d.download = nil, execute, download
					exhausted = false
					state, obs, err := observeManaged(context.Background(), state, f.d)
					if err != nil || obs.RecoveryRequired || state.UncertainProvisioning() {
						t.Fatalf("refusal required manual recovery after reload: %+v %v", obs, err)
					}
					state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
					if err != nil || !state.Initialized || obs.Health != "healthy" {
						t.Fatalf("capacity recovery did not allow retry: %+v %v", obs, err)
					}
				})
			}
		}
	}
}
