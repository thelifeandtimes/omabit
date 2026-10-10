//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestPrivateSocketModes(t *testing.T) {
	for _, mode := range []os.FileMode{
		0600, 0660, os.ModeSticky | 0600, os.ModeSticky | 0660,
		0666, 0664, 0606, 0640, 0760, os.ModeSticky | 0666,
		os.ModeSetuid | 0660, os.ModeSetgid | 0660,
		os.ModeSticky | os.ModeSetuid | 0660, os.ModeSticky | os.ModeSetgid | 0660,
	} {
		t.Run(mode.String(), func(t *testing.T) {
			f := fixture(t)
			stat := f.d.stat
			f.d.stat = func(path string) (node, error) {
				n, err := stat(path)
				if path == filepath.Join(f.d.runtimeDir, rootless.SocketName) {
					n.mode = os.ModeSocket | mode
				}
				return n, err
			}
			r, err := run(context.Background(), nil, f.d)
			if mode == 0600 || mode == 0660 || mode == os.ModeSticky|0600 || mode == os.ModeSticky|0660 {
				if err != nil || !r.CheckpointPassed {
					t.Fatalf("installed/private socket refused: %v", err)
				}
				assertReport(t, r)
			} else if err == nil || !reflect.DeepEqual(r, Result{}) || len(f.calls) != 0 || !strings.Contains(err.Error(), "socket requires mode") {
				t.Fatalf("unsafe socket accepted or unclear error: %+v %v", r, err)
			} else if !strings.Contains(err.Error(), (os.ModeSocket | mode).String()) {
				t.Fatalf("rejection omitted observed mode: %v", err)
			}
		})
	}
	for _, fault := range []string{"socket gid", "parent owner", "parent mode"} {
		t.Run(fault, func(t *testing.T) {
			f := fixture(t)
			stat := f.d.stat
			f.d.stat = func(path string) (node, error) {
				n, err := stat(path)
				if fault == "socket gid" && path == filepath.Join(f.d.runtimeDir, rootless.SocketName) {
					n.gid++
				}
				if path == f.d.runtimeDir {
					if fault == "parent owner" {
						n.uid++
					} else if fault == "parent mode" {
						n.mode = os.ModeDir | 0750
					}
				}
				return n, err
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || !reflect.DeepEqual(r, Result{}) || len(f.calls) != 0 {
				t.Fatal("private socket ownership/parent protection bypassed")
			}
		})
	}
}

func TestStickySocketMetadata(t *testing.T) {
	f := fixture(t)
	path := filepath.Join(f.d.runtimeDir, rootless.SocketName)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	// Reproduce Docker's StickRuntimeDirContents using a real temporary Unix
	// socket. No connections are accepted and no Docker process is executed.
	if err := os.Chmod(path, os.ModeSticky|0660); err != nil {
		t.Fatal(err)
	}
	actual, err := statNode(path)
	if err != nil || actual.mode != os.ModeSocket|os.ModeSticky|0660 {
		t.Fatalf("could not reproduce Docker socket metadata: %+v, %v", actual, err)
	}
	stat := f.d.stat
	f.d.stat = func(name string) (node, error) {
		if name == path {
			n, err := statNode(name)
			// Ownership follows the synthetic account used by the fake engine;
			// the complete FileMode comes from the real Linux socket above.
			n.uid, n.gid = uint32(f.i.uid), uint32(f.i.gid)
			return n, err
		}
		return stat(name)
	}
	r, err := run(context.Background(), nil, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete {
		t.Fatalf("valid sticky runtime socket refused: %+v, %v", r, err)
	}
	after, err := statNode(path)
	if err != nil || after.mode != actual.mode {
		t.Fatal("qualification changed socket permissions")
	}
	assertReport(t, r)
}

func TestImageBackendsUseOnlyVerifiedActualID(t *testing.T) {
	for _, id := range []string{candidate.ConfigDigest, candidate.ManifestDigest} {
		t.Run(id, func(t *testing.T) {
			for _, fault := range []string{"", "wrong repo", "wrong platform", "container uses other backend ID"} {
				t.Run(fault, func(t *testing.T) {
					f := fixture(t)
					f.imageID = id
					f.editImage = func(doc map[string]any) {
						if fault == "wrong repo" {
							doc["RepoDigests"] = []string{"other/vere@" + candidate.ManifestDigest}
						} else if fault == "wrong platform" {
							doc["Os"] = "windows"
						}
					}
					if fault == "container uses other backend ID" {
						f.editInspect = func(_ string, doc map[string]any) {
							other := candidate.ManifestDigest
							if id == other {
								other = candidate.ConfigDigest
							}
							doc["Image"] = other
							mapAt(doc, "Config")["Image"] = other
						}
					}
					r, err := run(context.Background(), nil, f.d)
					if fault == "" {
						if err != nil || !r.CheckpointPassed || r.ImageID != id || r.RepoDigest == "" || !strings.Contains(r.Phases[1].Detail, id) {
							t.Fatalf("backend identity evidence missing: %+v %v", r, err)
						}
					} else if err == nil {
						t.Fatal("backend ID bypassed pin or container identity checks")
					} else if fault != "container uses other backend ID" && (r.ImageID != "" || r.RepoDigest != "" || f.downloads != 0 || len(r.Containers) != 0) {
						t.Fatal("unverified image ID recorded/used as evidence")
					}
					assertReport(t, r)
				})
			}
		})
	}
}

func TestLateCreatePublicationAndDurableUncertainty(t *testing.T) {
	for _, publish := range []bool{true, false} {
		t.Run(fmt.Sprint(publish), func(t *testing.T) {
			f := fixture(t)
			f.d.times.cleanup = 80 * time.Millisecond
			execute := f.d.execute
			var pending *fakeContainer
			emptyLists := 0
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				out, err := execute(ctx, a, e)
				if len(a) > 5 && a[5] == "create" {
					for name, c := range f.containers {
						pending = c
						delete(f.containers, name)
					}
					return "", context.DeadlineExceeded // daemon accepted, not visible yet
				}
				if pending != nil && len(a) > 5 && a[5] == "ls" && out == "" {
					emptyLists++
					if publish {
						f.containers[pending.name] = pending // AFTER this empty response
					}
				}
				return out, err
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || emptyLists == 0 || len(r.Containers) != 1 {
				t.Fatalf("create uncertainty not exercised: %+v %v", r, err)
			}
			c := r.Containers[0]
			if !c.CreateDispatched || c.CreateUncertain == publish || r.CleanupComplete != publish || c.Removed != publish {
				t.Fatalf("empty listing resolved an in-flight request: %+v", r)
			}
			assertReport(t, r)
			if publish {
				if !containerIDPattern.MatchString(c.ID) || len(f.containers) != 0 {
					t.Fatal("late owned container was not recovered and removed")
				}
				return
			}
			before, err := os.ReadFile(r.ReportPath)
			if err != nil {
				t.Fatal(err)
			}
			f.d.execute = execute
			f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
			for _, visible := range []bool{false, true} {
				if visible {
					f.containers[pending.name] = pending
				}
				calls := len(f.calls)
				next, err := run(context.Background(), nil, f.d)
				if err == nil || !reflect.DeepEqual(next, Result{}) || len(f.calls) != calls || !strings.Contains(err.Error(), "manual resolution") {
					t.Fatalf("prior intent was ignored (visible=%v): %+v %v", visible, next, err)
				}
			}
			after, err := os.ReadFile(r.ReportPath)
			if err != nil || string(before) != string(after) {
				t.Fatal("later run rewrote unresolved prior evidence")
			}
		})
	}
}

func TestCleanupRetriesLockedAcceptedStartAndControlErrors(t *testing.T) {
	for _, fault := range []string{"inspection timeout", "stop error", "remove error", "foreign after timeout"} {
		t.Run(fault, func(t *testing.T) {
			f := fixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			execute := f.d.execute
			cleaning, failed := false, false
			var boot *fakeContainer
			f.d.execute = func(c context.Context, a, e []string) (string, error) {
				verb := a[5]
				if cleaning && !failed && ((verb == "inspect" && (fault == "inspection timeout" || fault == "foreign after timeout")) || (verb == "stop" && fault == "stop error") || (verb == "rm" && fault == "remove error")) {
					failed = true
					if verb == "inspect" {
						<-c.Done() // model Docker's lock outliving the request client
					}
					if fault == "foreign after timeout" {
						mapAt(mapAt(boot.doc, "Config"), "Labels")[runLabel] = "foreign"
					}
					return "", context.DeadlineExceeded
				}
				out, err := execute(c, a, e)
				if verb == "start" {
					for _, container := range f.containers {
						if container.kind == "boot" {
							boot = container
							cleaning = true
							cancel()
							return "", context.Canceled
						}
					}
				}
				return out, err
			}
			r, err := run(ctx, nil, f.d)
			if err == nil || !failed || r.CheckpointPassed {
				t.Fatalf("accepted-start failure not exercised: %+v %v", r, err)
			}
			foreign := fault == "foreign after timeout"
			if r.CleanupComplete == foreign {
				t.Fatalf("transient recovery/ownership outcome wrong: %+v", r)
			}
			joined := strings.Join(f.events, ",")
			if strings.Contains(joined, "stop-boot") == foreign || strings.Contains(joined, "rm-boot") == foreign {
				t.Fatalf("cleanup controls wrong after retry: %s", joined)
			}
			assertReport(t, r)
		})
	}
}

func TestCompletedHistoryIgnoresPierContentsAndHoldsLock(t *testing.T) {
	f := fixture(t)
	old, err := run(context.Background(), nil, f.d)
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(old.Directory, "data")
	if err := os.Symlink("/unreadable-target", filepath.Join(data, "retained-link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(data, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(data, 0700) })
	stat := f.d.stat
	locked := false
	f.d.stat = func(path string) (node, error) {
		if path == data || strings.HasPrefix(path, data+"/") {
			t.Fatal("prior data was inspected")
		}
		if path == old.ReportPath {
			file, err := os.OpenFile(filepath.Join(filepath.Dir(old.Directory), ".lock"), os.O_RDWR, 0)
			if err != nil {
				t.Fatal(err)
			}
			err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			_ = file.Close()
			if !errors.Is(err, syscall.EWOULDBLOCK) {
				t.Fatalf("report read without exclusive account lock: %v", err)
			}
			locked = true
		}
		return stat(path)
	}
	f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
	next, err := run(context.Background(), nil, f.d)
	if err != nil || !next.CheckpointPassed || next.Directory == old.Directory || !locked {
		t.Fatalf("retained pier blocked fresh run: %+v %v", next, err)
	}
	assertReport(t, old)
	assertReport(t, next)
}

func TestPriorMetadataIsBoundedAndFailsClosed(t *testing.T) {
	for _, fault := range []string{"malformed", "oversize", "missing report", "report symlink", "report writable", "run symlink", "missing outcome", "missing containers", "missing profile", "uncertain even if removed", "unremoved", "foreign name", "wrong run ID", "missing dispatch markers"} {
		t.Run(fault, func(t *testing.T) {
			f := fixture(t)
			old, err := run(context.Background(), nil, f.d)
			if err != nil {
				t.Fatal(err)
			}
			bytes, err := os.ReadFile(old.ReportPath)
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(bytes, &doc); err != nil {
				t.Fatal(err)
			}
			first := doc["containers"].([]any)[0].(map[string]any)
			switch fault {
			case "malformed":
				bytes = []byte("{broken-private-metadata")
			case "oversize":
				bytes = make([]byte, maxPriorReportBytes+1)
			case "missing outcome":
				delete(doc, "cleanup_complete")
			case "missing containers":
				delete(doc, "containers")
			case "missing profile":
				delete(doc, "profile")
			case "uncertain even if removed":
				first["create_uncertain"] = true
			case "unremoved":
				first["removed"] = false
			case "foreign name":
				first["name"] = "unrelated-container"
			case "wrong run ID":
				doc["run_id"] = strings.Repeat("c", 32)
			case "missing dispatch markers":
				for _, c := range doc["containers"].([]any) {
					delete(c.(map[string]any), "create_dispatched")
					delete(c.(map[string]any), "create_uncertain")
				}
			}
			if fault != "malformed" && fault != "oversize" {
				bytes = []byte(encode(t, doc))
			}
			if err := os.WriteFile(old.ReportPath, bytes, 0600); err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "missing report", "report symlink":
				if err := os.Remove(old.ReportPath); err != nil {
					t.Fatal(err)
				}
				if fault == "report symlink" {
					if err := os.Symlink(filepath.Join(old.Directory, "data"), old.ReportPath); err != nil {
						t.Fatal(err)
					}
				}
			case "report writable":
				if err := os.Chmod(old.ReportPath, 0660); err != nil {
					t.Fatal(err)
				}
			case "run symlink":
				target := filepath.Join(f.d.home, "retained")
				if err := os.Rename(old.Directory, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, old.Directory); err != nil {
					t.Fatal(err)
				}
			}
			f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
			calls := len(f.calls)
			r, err := run(context.Background(), nil, f.d)
			if err == nil || !reflect.DeepEqual(r, Result{}) || len(f.calls) != calls || !strings.Contains(err.Error(), "manual resolution") {
				t.Fatalf("unsafe history accepted: %+v %v", r, err)
			}
			if strings.Contains(err.Error(), "broken-private-metadata") {
				t.Fatal("report contents leaked in error")
			}
		})
	}
	t.Run("entry cap", func(t *testing.T) {
		f := fixture(t)
		base := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification")
		if err := os.MkdirAll(base, 0700); err != nil {
			t.Fatal(err)
		}
		for i := 0; i <= maxPriorRuns; i++ {
			if err := os.Mkdir(filepath.Join(base, fmt.Sprintf("%032x", i)), 0700); err != nil {
				t.Fatal(err)
			}
		}
		r, err := run(context.Background(), nil, f.d)
		if err == nil || !reflect.DeepEqual(r, Result{}) || len(f.calls) != 0 || !strings.Contains(err.Error(), "entry limit") {
			t.Fatalf("unbounded history scan: %+v %v", r, err)
		}
	})
}

func TestExposeMetadataIsNotPublication(t *testing.T) {
	for _, publish := range []bool{false, true} {
		t.Run(fmt.Sprint(publish), func(t *testing.T) {
			f := fixture(t)
			f.editInspect = func(_ string, doc map[string]any) {
				ports := map[string]any{"80/tcp": nil, "34343/udp": nil}
				if publish {
					ports["80/tcp"] = []any{map[string]any{"HostIp": "0.0.0.0", "HostPort": "80"}}
				}
				mapAt(doc, "NetworkSettings")["Ports"] = ports
			}
			r, err := run(context.Background(), nil, f.d)
			if (err == nil) == publish || r.CheckpointPassed == publish {
				t.Fatalf("publication evidence mishandled: %+v %v", r, err)
			}
			assertReport(t, r)
		})
	}
}
