package hostinstall

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func ownerEngineState(t *testing.T, f *fixture) ownerEngineActivation {
	t.Helper()
	data, err := f.i.readInstalled(context.Background(), ownerEngineActivationPath, 8192)
	if err != nil {
		t.Fatal(err)
	}
	var state ownerEngineActivation
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestOwnerFirstEngineActivationResumesAcrossProfileAndServiceInterruptions(t *testing.T) {
	for _, boundary := range []string{"owner-profile", "loginctl", "user-manager", "engine"} {
		t.Run(boundary, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			if boundary == "owner-profile" {
				f.i.publishFile = func(from, to string) error {
					if to == f.i.path(ownerhost.Path) {
						return errors.New("interrupted owner publication")
					}
					return os.Rename(from, to)
				}
			} else {
				f.fail = func(c call) error {
					if boundary == "loginctl" && c.path == "/usr/bin/loginctl" ||
						boundary == "user-manager" && slices.Contains(c.args, "user@942.service") && slices.Contains(c.args, "start") ||
						boundary == "engine" && slices.Contains(c.args, rootless.UnitName) && slices.Contains(c.args, "--now") {
						return errors.New("interrupted initial engine activation")
					}
					return nil
				}
			}
			result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
			if err == nil || result.Ready || !result.Partial {
				t.Fatal(result, err)
			}
			if state := ownerEngineState(t, f); state.Owner != p || state.Stage != "prepared" {
				t.Fatal(state)
			}
			_, profileErr := os.Stat(f.i.path(ownerhost.Path))
			if (profileErr == nil) != (boundary != "owner-profile") {
				t.Fatal(profileErr)
			}
			if _, err := os.Stat(f.i.path(testSocket)); !os.IsNotExist(err) {
				t.Fatal("fixture manufactured engine startup", err)
			}
			f.fail, f.i.publishFile, f.calls, f.i.mutated = nil, nil, nil, false
			result, err = f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
			if err != nil || !result.Ready {
				t.Fatal("first installation did not resume", result, err)
			}
			linger, manager, engine := 0, 0, 0
			for _, c := range f.calls {
				if c.path == "/usr/bin/loginctl" && slices.Equal(c.args, []string{"enable-linger", p.Name}) {
					linger++
				}
				if c.path == "/usr/bin/systemctl" && slices.Equal(c.args, []string{"start", "user@942.service"}) {
					manager++
				}
				if c.path == "/usr/bin/systemctl" && slices.Contains(c.args, rootless.UnitName) && slices.Contains(c.args, "--now") {
					engine++
				}
			}
			if linger != 1 || manager != 1 || engine != 1 {
				t.Fatal("missing explicit initial activation", linger, manager, engine)
			}
			if state := ownerEngineState(t, f); state.Owner != p || state.Stage != "complete" {
				t.Fatal(state)
			}
			info, err := os.Stat(f.i.path(ownerEngineActivationPath))
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatal("activation journal permissions", err)
			}
		})
	}
}

func TestEstablishedStoppedOwnerEngineNeverRestartsOnInstallRetry(t *testing.T) {
	for _, generation := range []string{"recorded", "legacy-without-record"} {
		t.Run(generation, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
				t.Fatal(err)
			}
			if generation == "legacy-without-record" {
				if err := os.Remove(f.i.path(ownerEngineActivationPath)); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Remove(f.i.path(testSocket)); err != nil {
				t.Fatal(err)
			}
			delete(f.metadata, testSocket)
			f.calls, f.i.mutated = nil, false
			before := tree(t, f.i.root)
			result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
			if err == nil || result.Ready {
				t.Fatal("stopped engine was revived or treated as ready", result, err)
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/loginctl" || slices.Contains(c.args, "start") || slices.Contains(c.args, "restart") || slices.Contains(c.args, "--now") {
					t.Fatal("established stopped engine startup", c)
				}
			}
			if !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("stopped installation changed")
			}
		})
	}
}

func TestOwnerEngineCompletionPublicationRetryDoesNotRepeatActivation(t *testing.T) {
	f, p, probe := ownerFixture(t)
	interrupted := false
	f.i.syncDirectory = func(path string) error {
		if path == f.i.path("/etc/urbit-runner") && !interrupted {
			data, _ := os.ReadFile(f.i.path(ownerEngineActivationPath))
			if strings.Contains(string(data), `"stage":"complete"`) {
				interrupted = true
				return errors.New("lost completion sync acknowledgement")
			}
		}
		return nil
	}
	if result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err == nil || result.Ready || !interrupted {
		t.Fatal(result, err)
	}
	if state := ownerEngineState(t, f); state.Stage != "complete" {
		t.Fatal(state)
	}
	f.calls, f.i.syncDirectory = nil, nil
	if result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil || !result.Ready {
		t.Fatal(result, err)
	}
	for _, c := range f.calls {
		if c.path == "/usr/bin/loginctl" || slices.Contains(c.args, "user@942.service") || slices.Contains(c.args, rootless.UnitName) && slices.Contains(c.args, "--now") {
			t.Fatal("repeated completed engine activation", c)
		}
	}
}

func TestOwnerEngineJournalRejectsUnboundOrUnsafeEvidence(t *testing.T) {
	for _, fault := range []string{"owner", "version", "stage", "unknown-field", "writable", "symlink", "stale-prepared"} {
		t.Run(fault, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
				t.Fatal(err)
			}
			state := ownerEngineState(t, f)
			switch fault {
			case "owner":
				state.Owner.UID++
			case "version":
				state.Version++
			case "stage":
				state.Stage = "unknown"
			case "stale-prepared":
				state.Stage = "prepared"
			}
			data, _ := json.Marshal(state)
			if fault == "unknown-field" {
				data = append([]byte(`{"future":true,`), data[1:]...)
			}
			mode := os.FileMode(0600)
			if fault == "writable" {
				mode = 0666
			}
			f.put(ownerEngineActivationPath, string(data)+"\n", mode)
			if fault == "symlink" {
				if err := os.Rename(f.i.path(ownerEngineActivationPath), f.i.path(ownerEngineActivationPath+".saved")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(ownerEngineActivationPath+".saved", f.i.path(ownerEngineActivationPath)); err != nil {
					t.Fatal(err)
				}
			}
			f.calls, f.i.mutated = nil, false
			before := tree(t, f.i.root)
			if result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err == nil || result.Ready {
				t.Fatal(result, err)
			}
			if len(f.calls) != 0 || !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("unsafe journal caused host changes")
			}
		})
	}
}
