//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const correctedFixtureShip = "~sampel-sampel-sampel-palnet"

func identityRepairFixture(t *testing.T) (*fakeEngine, ManagedState, string, func(context.Context, string, string, string) error) {
	t.Helper()
	f, s, journal := bootAdmissionFixture(t, true)
	// Match the incident's moon + selected Recommended Apps pill launch shape.
	s.NativeShip, s.Comet.Ship = "~sampel-palnet-sampel-palnet", "~sampel-palnet-sampel-palnet"
	s.Storage.Path = "livenet/moons/sampel-palnet-sampel-palnet/pier"
	pill := []byte("synthetic recommended apps pill")
	p := protocol.BootPill{SHA256: fmt.Sprintf("%x", sha256.Sum256(pill)), Bytes: int64(len(pill))}
	s.Boot.Pill = &p
	base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
	if err := os.Mkdir(filepath.Join(base, "boot-pills"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bootpill.CachePath(base, p), pill, 0400); err != nil {
		t.Fatal(err)
	}
	identity := f.d.hostIdentity
	f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
		_, err := identity(ctx, endpoint)
		return correctedFixtureShip, err
	}
	s, _, err := applyManaged(context.Background(), "start", s, journal.save, f.d)
	if err == nil || s.Initialized || !s.UncertainProvisioning() {
		t.Fatal("fixture did not reproduce a quarantined keyed boot", err)
	}
	boot, ok := bootAdmissionRecord(s)
	if !ok {
		t.Fatal("no boot")
	}
	mapAt(f.containers[boot.Name].doc, "State")["StartedAt"] = time.Now().UTC().Format(time.RFC3339Nano)
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "container kill") || strings.Contains(joined, "container stop") {
			t.Fatal("repair escalated to a signal")
		}
		if strings.Contains(joined, `"dojo":"our"`) {
			return `"` + correctedFixtureShip + `"`, nil
		}
		return execute(ctx, args, env)
	}
	absent := func(_ context.Context, _, id, name string) error {
		for _, c := range f.containers {
			if c.id == id || c.name == name {
				return errors.New("still present")
			}
		}
		return nil
	}
	return f, s, boot.ID, absent
}

func TestBootIdentityRepairInterruptionSamePierAndNormalResume(t *testing.T) {
	for _, cut := range []string{"none", "before-verified", "after-verified", "before-removing", "after-removing", "after-rm-before-ack", "after-removed"} {
		t.Run(cut, func(t *testing.T) {
			f, s, id, absent := identityRepairFixture(t)
			original, _ := json.Marshal(s)
			path := filepath.Join(managedPier(f, s), "retained-user-data")
			if err := os.WriteFile(path, []byte("same-pier-marker"), 0600); err != nil {
				t.Fatal(err)
			}
			var saved []byte
			cutOnce := false
			persist := func(j BootIdentityRepair) error {
				boot, _ := bootAdmissionRecord(j.State)
				point := j.Stage
				if j.Stage == "removing" && boot.Removed {
					point = "rm-before-ack"
				}
				if !cutOnce && cut == "before-"+point {
					cutOnce = true
					return errors.New("disk unavailable")
				}
				if !cutOnce && cut == "after-rm-before-ack" && point == "rm-before-ack" {
					cutOnce = true
					return errors.New("lost removal acknowledgement")
				}
				saved, _ = json.Marshal(j)
				if !cutOnce && cut == "after-"+point {
					cutOnce = true
					return errors.New("simulated process loss after commit")
				}
				return nil
			}
			result, err := repairBootIdentity(context.Background(), s, s.NativeShip, correctedFixtureShip, id, nil, persist, f.d, absent)
			if cut != "none" {
				if err == nil || !cutOnce {
					t.Fatal("interruption was not exercised", cut, err)
				}
				var resume *BootIdentityRepair
				if len(saved) > 0 {
					if err := json.Unmarshal(saved, &resume); err != nil {
						t.Fatal(err)
					}
				}
				result, err = repairBootIdentity(context.Background(), s, s.NativeShip, correctedFixtureShip, id, resume, persist, f.d, absent)
			}
			if err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(s)
			if string(after) != string(original) {
				t.Fatal("original evidence was mutated")
			}
			if !result.MaintenanceQualified() || result.UncertainProvisioning() || result.NativeShip != correctedFixtureShip || result.Storage.Path != s.Storage.Path || result.Storage.MountIdentity != s.Storage.MountIdentity || result.ID != s.ID {
				t.Fatal("incomplete/wrong-object repair")
			}
			var receipt BootIdentityRepair
			if err := json.Unmarshal(saved, &receipt); err != nil {
				t.Fatal(err)
			}
			if receipt.State.NativeShip != s.NativeShip || receipt.Stage != "removed" {
				t.Fatal("journal lost old launch identity")
			}
			// The existing manager's normal Start must resume, never bootstrap again.
			f.d.execute = f.execute
			result, obs, err := applyManaged(context.Background(), "start", result, func(state ManagedState) error {
				b, _ := json.Marshal(state.Runtime)
				return os.WriteFile(filepath.Join(state.Runtime.Directory, "report.json"), b, 0600)
			}, f.d)
			if err != nil || obs.State != "running" || obs.Health != "healthy" || obs.RecoveryRequired || result.NativeKeyDigest != "" {
				t.Fatalf("normal resume failed: %+v %v", obs, err)
			}
			if bootAdmissionEventCount(f, "start-comet-boot") != 1 || bootAdmissionEventCount(f, "start-comet-resume") != 1 {
				t.Fatal("repair bootstrapped a replacement")
			}
			b, err := os.ReadFile(path)
			if err != nil || string(b) != "same-pier-marker" {
				t.Fatal("existing pier data lost", err)
			}
		})
	}
}

func TestBootIdentityRepairRefusesUnverifiedAndChangedRuntime(t *testing.T) {
	for _, mode := range []string{"public", "private", "class", "container", "containment", "stopped", "mount", "maintenance", "oom", "nonzero", "generation", "missing-without-removal", "absence-unavailable"} {
		t.Run(mode, func(t *testing.T) {
			f, s, id, absent := identityRepairFixture(t)
			boot, _ := bootAdmissionRecord(s)
			ship := correctedFixtureShip
			var saved *BootIdentityRepair
			persist := func(j BootIdentityRepair) error {
				raw, _ := json.Marshal(j)
				saved = nil
				return json.Unmarshal(raw, &saved)
			}
			switch mode {
			case "public":
				f.d.hostIdentity = func(context.Context, string) (string, error) { return s.NativeShip, nil }
			case "private":
				execute := f.d.execute
				f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					if strings.Contains(strings.Join(args, " "), `"dojo":"our"`) {
						return `"~zod"`, nil
					}
					return execute(ctx, args, env)
				}
			case "class":
				ship = "~zod"
			case "container":
				id = strings.Repeat("f", 64)
			case "containment":
				mapAt(f.containers[boot.Name].doc, "Config")["Cmd"] = []string{"wrong"}
			case "stopped":
				mapAt(f.containers[boot.Name].doc, "State")["Running"] = false
			case "mount":
				s.Storage.MountIdentity.Inode++
			case "maintenance":
				s.MaintenanceUncertain = true
			case "oom":
				f.oom = true
			case "nonzero":
				f.stopExit = 1
			case "generation", "missing-without-removal", "absence-unavailable":
				_, err := repairBootIdentity(context.Background(), s, s.NativeShip, ship, id, nil, func(j BootIdentityRepair) error {
					if mode == "absence-unavailable" && j.Stage != "removed" {
						return persist(j)
					}
					_ = persist(j)
					return errors.New("interrupted")
				}, f.d, absent)
				if err == nil {
					t.Fatal("did not interrupt")
				}
				if mode == "generation" {
					mapAt(f.containers[boot.Name].doc, "State")["StartedAt"] = time.Now().UTC().Add(time.Second).Format(time.RFC3339Nano)
				}
				if mode == "missing-without-removal" {
					delete(f.containers, boot.Name)
				}
				if mode == "absence-unavailable" {
					absent = func(context.Context, string, string, string) error { return errors.New("engine unavailable") }
				}
			}
			beforeStops := bootAdmissionEventCount(f, "stop-comet-boot")
			_, err := repairBootIdentity(context.Background(), s, s.NativeShip, ship, id, saved, persist, f.d, absent)
			if err == nil {
				t.Fatal("unsafe repair accepted", mode)
			}
			if mode != "oom" && mode != "nonzero" && bootAdmissionEventCount(f, "stop-comet-boot") != beforeStops {
				t.Fatal("refused correction stopped the runtime")
			}
			if bootAdmissionEventCount(f, "rm-comet-boot") != 0 && mode != "absence-unavailable" {
				t.Fatal("failed correction removed runtime evidence")
			}
		})
	}
}

func TestBootIdentityRepairAbsenceRequiresBothDaemon404s(t *testing.T) {
	for _, codes := range [][]int{{404, 404}, {404, 200}, {500, 404}, {301, 404}} {
		t.Run(strings.Join([]string{http.StatusText(codes[0]), http.StatusText(codes[1])}, "-"), func(t *testing.T) {
			socket := filepath.Join(t.TempDir(), "docker.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			var paths []string
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				w.WriteHeader(codes[len(paths)-1])
			})}
			done := make(chan struct{})
			go func() { _ = srv.Serve(listener); close(done) }()
			err = removedBootContainer(context.Background(), socket, strings.Repeat("a", 64), "exact-boot")
			_ = srv.Close()
			<-done
			want := reflect.DeepEqual(codes, []int{404, 404})
			if (err == nil) != want || want && (len(paths) != 2 || !strings.HasSuffix(paths[1], "/exact-boot/json")) {
				t.Fatal(paths, err)
			}
		})
	}
}
