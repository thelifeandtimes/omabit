//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func TestBootstrapUsesPermanentMountAndNeverStopsOrMovesTheShip(t *testing.T) {
	f, state, persist := managedFixture(t)
	state.Storage = &pierstore.Location{Path: "fakeships/project-one/zod/pier"}
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || state.Storage.Identity.Inode == 0 || state.Storage.Uncertain {
		t.Fatal(obs, err)
	}
	p, err := pierstore.Check(f.d.home, *state.Storage)
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(f.d.home, "piers/fakeships/project-one/zod/pier") {
		t.Fatal(p)
	}
	if _, err := os.Stat(filepath.Join(state.Runtime.Directory, "data")); !os.IsNotExist(err) {
		t.Fatal("obsolete staging directory created")
	}
	for _, c := range f.containers {
		args := mapAt(c.doc, "Config")["Cmd"].([]string)
		if args[len(args)-1] != "/urbit/pier" || fakeMountRoot(c) != filepath.Dir(p) {
			t.Fatal("permanent bootstrap mount", args)
		}
	}
	for _, e := range f.events {
		if e == "stop-http-boot" || e == "rm-http-boot" || e == "start-http-restart" {
			t.Fatal("bootstrap recycled the ship", e)
		}
	}
	identity := *state.Storage
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || *state.Storage != identity {
		t.Fatal("resume changed permanent location", err)
	}
	if f.downloads != 1 {
		t.Fatal("pill downloaded on replacement")
	}
}

func TestOwnedKeyBootUsesPermanentPierAndCleansUpKeyWithoutRestart(t *testing.T) {
	f, state, persist := managedCometFixture(t)
	state.NativeShip = "~sampel-palnet"
	state.Comet.Ship = state.NativeShip
	state.Storage = &pierstore.Location{Path: "livenet/planets/sampel-palnet/pier"}
	key := "0w" + strings.Repeat("a", 128)
	state.NativeKeyDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/keys")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, state.ID+".key")
	if err := os.WriteFile(file, []byte(key), 0600); err != nil {
		t.Fatal(err)
	}
	f.d.hostIdentity = func(context.Context, string) (string, error) { return "~sampel-palnet", nil }
	state, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || state.Storage.Identity.Inode == 0 || state.NativeKeyDigest != "" {
		t.Fatal(obs, err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("boot key retained after initialized replacement")
	}
	boot := false
	for _, call := range f.calls {
		joined := " " + strings.Join(call, " ") + " "
		if strings.Contains(joined, key) {
			t.Fatal("key in arguments")
		}
		if strings.Contains(joined, " -w ~sampel-palnet ") {
			boot = true
			if !strings.Contains(joined, " -k /artifacts/boot.key ") {
				t.Fatal("key file flag absent")
			}
		}
	}
	if !boot {
		t.Fatal("owned bootstrap was not dispatched")
	}
	for _, e := range f.events {
		if e == "stop-comet-boot" || e == "start-comet-resume" {
			t.Fatal("boot required a stop/restart", e)
		}
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.containers {
		cmd := mapAt(c.doc, "Config")["Cmd"].([]string)
		for _, arg := range cmd {
			if arg == "-k" || arg == "-w" || arg == "-c" {
				t.Fatal("resume repeated identity flags")
			}
		}
	}
}

func TestOwnedEnrollmentVerifiesWithAmesDisabledBeforeNetworkedResume(t *testing.T) {
	f, state, persist := managedCometFixture(t)
	state.NativeShip = "~sampel-palnet"
	state.Comet.Ship = state.NativeShip
	rel := "livenet/planets/sampel-palnet/pier"
	p, err := pierstore.Parents(f.d.home, rel)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(p, ".urb/chk"), 0700); err != nil {
		t.Fatal(err)
	}
	writePierEpochPath(t, p, "0i0", false)
	identity, err := pierstore.Identify(p)
	if err != nil {
		t.Fatal(err)
	}
	mountIdentity, err := pierstore.IdentifyMount(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	state.Storage = &pierstore.Location{Path: rel, Identity: identity, MountIdentity: mountIdentity}
	state.Adoption = &ManagedAdoption{Fingerprint: pierstore.Fingerprint(identity), Uncertain: true}
	state, obs, err := applyManaged(context.Background(), "enroll-pier", state, persist, f.d)
	if err != nil || obs.State != "stopped" || state.MaintenanceQualified() {
		t.Fatal(obs, err)
	}
	if _, _, err := applyManaged(context.Background(), "pack", state, persist, f.d); err == nil {
		t.Fatal("maintenance preceded identity verification")
	}
	f.d.hostIdentity = func(context.Context, string) (string, error) { return "~sampel-palnet", nil }
	state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || !state.Adoption.Verified {
		t.Fatal(obs, err)
	}
	verified, resumed := false, false
	for _, call := range f.calls {
		if len(call) < 2 || call[0] != "container" || call[1] != "create" {
			continue
		}
		joined := " " + strings.Join(call, " ") + " "
		if strings.Contains(joined, " -c ") || strings.Contains(joined, " -w ") || strings.Contains(joined, " -k ") {
			t.Fatal("import used creation/key flags")
		}
		if strings.Contains(joined, "comet-verify") {
			verified = strings.Contains(joined, " -L ")
		}
		if strings.Contains(joined, "comet-resume") {
			resumed = !strings.Contains(joined, " -L ") && verified
		}
	}
	if !verified || !resumed {
		t.Fatal("identity handover ordering", verified, resumed)
	}
}

func TestMovingExistingPierPreservesEpochsAndMissingDoesNotRecreate(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	for _, name := range []string{"0i0", "0i100", "0i200", "0i300"} {
		writePierEpochPath(t, managedPier(f, state), name, name != "0i0")
	}
	old := managedPier(f, state)
	identity, err := pierstore.Identify(old)
	if err != nil {
		t.Fatal(err)
	}
	state.MoveTarget = "fakeships/project/zod/pier"
	state, obs, err := applyManaged(context.Background(), "move-pier", state, persist, f.d)
	if err != nil || obs.State != "stopped" || state.Storage.Identity != identity {
		t.Fatal(obs, err)
	}
	state, _, err = applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil || len(state.Epochs.Entries) != 4 {
		t.Fatal("moved inventory", err)
	}
	state.PruneIntent = &ManagedPruneIntent{Cutoff: "0i100", InventoryDigest: state.Epochs.Digest, Confirmed: true}
	state, _, err = applyManaged(context.Background(), "prune-epochs", state, persist, f.d)
	if err != nil || len(state.Epochs.Entries) != 2 {
		t.Fatal("moved pruning", err)
	}
	p := filepath.Join(pierstore.Root(f.d.home), state.Storage.Path)
	if err := os.Rename(filepath.Dir(p), filepath.Dir(p)+"-moved"); err != nil {
		t.Fatal(err)
	}
	_, obs, err = observeManaged(context.Background(), state, f.d)
	if err != nil || obs.State != "missing" || !obs.RecoveryRequired {
		t.Fatal(obs, err)
	}
	before := len(f.calls)
	if _, _, err = applyManaged(context.Background(), "start", state, persist, f.d); err == nil {
		t.Fatal("recreated missing pier")
	}
	if len(f.calls) != before {
		t.Fatal("missing pier dispatched Docker")
	}
}
