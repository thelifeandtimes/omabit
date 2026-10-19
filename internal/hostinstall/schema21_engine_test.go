package hostinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

// Service/process/artifact evidence is synthetic, as in the RC11 activation
// fixtures. The installer, files and activation journal are real. Original-writer
// schema20 SQLite migration is covered by the managed historical-upgrade tests.
func TestSchema21EngineOnlineUpgradeFrom20(t *testing.T) {
	if managed.StateSchemaVersion <= 20 {
		t.Fatal("engine evidence requires a schema boundary after published schema20")
	}
	f, p, probe := ownerFixture(t)
	ctx := context.Background()
	old, target := []byte("synthetic-schema20-generation"), []byte("synthetic-current-schema-generation")
	initial, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), old, p, probe, nil)
	if err != nil || !initial.Ready {
		t.Fatal(initial, err)
	}
	// Model the completed schema20 installation's executable metadata and RPC.
	// No database marker or historical-writer fixture is changed here.
	j := retryJournal(t, f)
	j.Target.Schema = 20
	if err := f.i.saveActivation(ctx, &j, "complete"); err != nil {
		t.Fatal(err)
	}
	contract, err := json.Marshal(j.Target)
	if err != nil {
		t.Fatal(err)
	}
	f.put(filepath.Dir(generationPath(j.Target))+"/contract.json", string(contract)+"\n", 0644)
	rpc := f.i.updateRPC
	acquired, released := false, false
	f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
		status, err := rpc(ctx, socket, uid, pid, req)
		if status.ExecutableSHA256 == digestBytes(old) {
			status.StateSchema = 20
			acquired = acquired || req != nil && req.Action == "acquire" && err == nil && status.Quiesced
		} else if status.ExecutableSHA256 == digestBytes(target) {
			released = released || req != nil && req.Action == "release" && err == nil && !status.Quiesced && status.StateSchema == managed.StateSchemaVersion
		}
		return status, err
	}
	preserved := map[string][]byte{}
	for _, path := range []string{ownerhost.Path, access.DefaultKeys, hostKeyPath, managerUnit, unitPath} {
		data, err := os.ReadFile(f.i.path(path))
		if err != nil {
			t.Fatal(err)
		}
		preserved[path] = data
	}
	f.calls = nil
	result, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), target, p, probe, nil)
	if err != nil || !result.Ready || result.RunnerID != initial.RunnerID || result.HostPublicKey != initial.HostPublicKey || result.Owner != initial.Owner || result.UID != initial.UID || result.PierRoot != initial.PierRoot {
		t.Fatal("online update changed identity or did not activate", result, err)
	}
	j = retryJournal(t, f)
	if !acquired || !released || j.Offline || j.Stage != "complete" || j.Previous == nil || j.Previous.Schema != 20 || j.Target.Schema != managed.StateSchemaVersion {
		t.Fatal("normal online schema boundary was not exercised", j, acquired, released)
	}
	for _, generation := range []executableGeneration{*j.Previous, j.Target} {
		if _, err := f.i.retainedBytes(ctx, generation); err != nil {
			t.Fatal("verified generation lost", err)
		}
	}
	for path, before := range preserved {
		after, err := os.ReadFile(f.i.path(path))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("online update changed retained host identity/profile", path, err)
		}
	}
	for _, c := range f.calls {
		if slices.Contains(c.args, "stop") && !slices.Contains(c.args, "urbit-runner.service") || slices.Contains(c.args, rootless.UnitName) && (slices.Contains(c.args, "start") || slices.Contains(c.args, "restart") || slices.Contains(c.args, "--now")) {
			t.Fatal("online update changed engine availability", c)
		}
	}
	if _, err := f.i.rollbackManagement(ctx, "fixture", &j, func(...string) error {
		t.Fatal("incompatible rollback dispatched")
		return nil
	}); err == nil {
		t.Fatal("schema20 generation accepted as successor-schema rollback target")
	}
	f.calls = nil
	if r, err := f.i.installOwner(ctx, "/bundle", hostPublic(1), target, p, probe, nil); err != nil || !r.Ready {
		t.Fatal(r, err)
	}
	for _, c := range f.calls {
		if slices.Contains(c.args, "stop") {
			t.Fatal("completed update replay stopped management", c)
		}
	}
}
