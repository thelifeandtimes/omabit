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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func legacyAmesFixture(t *testing.T, ship, tier string, verified, named bool) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, state, persist := managedCometFixture(t)
	state.NativeShip, state.Comet.Ship = ship, ship
	state.Loom, state.NativeControl = 29, true // a supported historical contract
	rel := "livenet/" + tier + "/legacy/" + strings.TrimPrefix(ship, "~")
	if named {
		if _, err := pierstore.RegisterRoot(f.d.home, "ssd", t.TempDir()); err != nil {
			t.Fatal(err)
		}
		rel = "@ssd/" + rel
	}
	path, err := pierstore.Parents(f.d.home, rel)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, ".urb/chk"), 0700); err != nil {
		t.Fatal(err)
	}
	writePierEpochPath(t, path, "0i0", false)
	identity, err := pierstore.Identify(path)
	if err != nil {
		t.Fatal(err)
	}
	mount, err := pierstore.IdentifyMount(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	state.Storage = &pierstore.Location{Path: rel, Identity: identity, MountIdentity: mount}
	state.Adoption = &ManagedAdoption{Fingerprint: pierstore.Fingerprint(identity), Uncertain: true}
	state, _, err = applyManaged(context.Background(), "enroll-pier", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	f.d.hostIdentity = func(context.Context, string) (string, error) { return ship, nil }
	if verified {
		for _, action := range []string{"start", "stop"} {
			state, _, err = applyManaged(context.Background(), action, state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	state.ExternalImport = &ManagedExternalImport{Stage: "complete", Source: pierstore.ExternalSource{Path: path, Identity: identity, Fingerprint: strings.Repeat("e", 64)}}
	return f, state, persist
}

func TestAmesPublicationLegacyEnrollmentLaunchesExistingPier(t *testing.T) {
	for _, tc := range []struct{ ship, tier string }{{"~sampel-palnet", "planets"}, {"~doznec-dozzod-sampel-palnet", "moons"}, {fixtureComet, "comets"}} {
		for _, verified := range []bool{false, true} {
			name := tc.tier + "/unverified"
			if verified {
				name = tc.tier + "/verified"
			}
			t.Run(name, func(t *testing.T) {
				f, state, persist := legacyAmesFixture(t, tc.ship, tc.tier, verified, true)
				before, err := json.Marshal(state)
				if err != nil {
					t.Fatal(err)
				}
				calls, downloads := len(f.calls), f.downloads
				if err := validateAmesPublication(context.Background(), state, f.d); err != nil {
					t.Fatal(err)
				}
				proposed, err := AmesPublicationState(state, 34567)
				if err != nil {
					t.Fatal(err)
				}
				after, _ := json.Marshal(state)
				if string(before) != string(after) || proposed.Boot == nil || proposed.Boot.Pill != nil || !proposed.Initialized || proposed.NativeKeyDigest != "" || proposed.Loom != 29 || !proposed.NativeControl || !reflect.DeepEqual(proposed.Storage, state.Storage) || !reflect.DeepEqual(proposed.Adoption, state.Adoption) || !reflect.DeepEqual(proposed.ExternalImport, state.ExternalImport) {
					t.Fatal("publication changed identity, settings, source provenance or original state")
				}
				for _, call := range f.calls[calls:] {
					if call[0] != "info" && call[1] != "inspect" && call[1] != "ls" {
						t.Fatal("publication performed a runtime mutation", call)
					}
				}
				// Serialize the proposed contract as the runner does at commit.
				data, _ := json.Marshal(proposed)
				proposed = ManagedState{}
				if err := json.Unmarshal(data, &proposed); err != nil {
					t.Fatal(err)
				}
				calls = len(f.calls)
				proposed, obs, err := applyManaged(context.Background(), "start", proposed, persist, f.d)
				if err != nil || obs.Health != "healthy" || !proposed.Adoption.Verified || f.downloads != downloads || !reflect.DeepEqual(proposed.Storage, state.Storage) {
					t.Fatalf("migrated next Start failed: %+v %v", obs, err)
				}
				resume, verify := false, false
				for _, call := range f.calls[calls:] {
					if len(call) < 2 || call[0] != "container" || call[1] != "create" {
						continue
					}
					joined := " " + strings.Join(call, " ") + " "
					for _, forbidden := range []string{" -c ", " -F ", " -B ", " -k ", " -w ", "comet-boot"} {
						if strings.Contains(joined, forbidden) && !(forbidden == " -c " && strings.Contains(joined, "umask 077;")) {
							t.Fatal("existing pier was bootstrapped", call)
						}
					}
					if strings.Contains(joined, "comet-verify") {
						verify = true
						if strings.Contains(joined, "/udp") || !strings.Contains(joined, " -L ") {
							t.Fatal("identity verification enabled Ames", call)
						}
					}
					if strings.Contains(joined, "comet-resume") {
						resume = true
						if !strings.Contains(joined, "0.0.0.0:34567:34567/udp") || !strings.Contains(joined, " --ames-port 34567 ") || strings.Contains(joined, " -L ") || !strings.Contains(joined, " --loom 29 ") {
							t.Fatal("incorrect migrated launch contract", call)
						}
					}
				}
				if !resume || verify == verified {
					t.Fatal("identity verification/resume ordering changed", verify, resume)
				}
			})
		}
	}
}

func TestAmesPublicationRefusesMissingForeignBusyOrUncertainEvidence(t *testing.T) {
	for _, fault := range []string{"missing-adoption", "foreign-fingerprint", "foreign-run", "foreign-image", "foreign-pill", "foreign-source", "missing-pier", "replacement-pier", "lock", "late-lock", "retained-container", "uncertain-create", "uncertain-network", "maintenance", "provisioning", "daemon-container", "engine-error", "canceled"} {
		t.Run(fault, func(t *testing.T) {
			f, state, _ := legacyAmesFixture(t, "~sampel-palnet", "planets", false, false)
			path, err := pierstore.Absolute(f.d.home, state.Storage.Path)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch fault {
			case "missing-adoption":
				state.Adoption = nil
			case "foreign-fingerprint":
				state.Adoption.Fingerprint = strings.Repeat("f", 64)
			case "foreign-run":
				state.Runtime.RunID = strings.Repeat("f", 32)
			case "foreign-image":
				state.Runtime.Image = "foreign"
			case "foreign-pill":
				state.Runtime.PillSHA256 = strings.Repeat("f", 64)
			case "foreign-source":
				state.ExternalImport.Source.Identity.Inode = 0
			case "missing-pier", "replacement-pier":
				if err := os.Rename(path, path+"-preserved"); err != nil {
					t.Fatal(err)
				}
				if fault == "replacement-pier" {
					if err := os.MkdirAll(filepath.Join(path, ".urb"), 0700); err != nil {
						t.Fatal(err)
					}
				}
			case "lock":
				if err := os.WriteFile(filepath.Join(path, ".vere.lock"), []byte("fixture"), 0600); err != nil {
					t.Fatal(err)
				}
			case "late-lock":
				f.editInfo = func(map[string]any) {
					if err := os.WriteFile(filepath.Join(path, ".urb/lock"), []byte("fixture"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			case "retained-container":
				state.Runtime.Containers = []ContainerRecord{{Kind: "comet-resume", CreateDispatched: true}}
			case "uncertain-create":
				state.Runtime.Containers = []ContainerRecord{{Kind: "comet-resume", Removed: true, CreateUncertain: true}}
			case "uncertain-network":
				state.Runtime.Network = &NetworkRecord{CreateUncertain: true}
			case "maintenance":
				state.MaintenanceUncertain = true
			case "provisioning":
				state.Storage.Uncertain = true
			case "daemon-container":
				execute := f.d.execute
				f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					out, err := execute(ctx, args, env)
					if err == nil && args[4] == "container" && args[5] == "ls" {
						return "unexpected-managed-runtime\n", nil
					}
					return out, err
				}
			case "engine-error":
				f.d.execute = func(context.Context, []string, []string) (string, error) {
					return "", errors.New("fixture engine unavailable")
				}
			case "canceled":
				cancel()
			}
			before, _ := json.Marshal(state)
			if err := validateAmesPublication(ctx, state, f.d); err == nil {
				t.Fatal("unsafe publication admitted")
			}
			after, _ := json.Marshal(state)
			if string(before) != string(after) || state.Boot != nil || len(f.containers) != 0 {
				t.Fatal("failed validation mutated registration or runtime")
			}
		})
	}
}
