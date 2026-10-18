//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func withRuntimeSettings(t *testing.T, s ManagedState, loom int, native bool) ManagedState {
	t.Helper()
	b, _ := json.Marshal(s)
	var wire map[string]json.RawMessage
	json.Unmarshal(b, &wire)
	wire["loom"] = json.RawMessage(fmt.Sprint(loom))
	wire["native_control"] = json.RawMessage(fmt.Sprint(native))
	b, _ = json.Marshal(wire)
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func assertSettingsArgs(t *testing.T, args []string, loom string, native bool) {
	t.Helper()
	i := slices.Index(args, "--loom")
	if i < 0 || i+1 >= len(args) || args[i+1] != loom {
		t.Fatalf("loom propagation missing: %v; want %s", args, loom)
	}
	runtime := slices.Contains(args, "-t")
	if runtime && slices.Contains(args, "--no-conn") == native {
		t.Fatalf("native control opt-in mismatch: %v want enabled=%t", args, native)
	}
	if runtime && !slices.Contains(args, "--no-dock") {
		t.Fatal("undocked boundary lost", args)
	}
}

func TestRuntimeSettingsLaunchStopStartAndUtilities(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprint(native), func(t *testing.T) {
			f, state, persist := managedFixture(t)
			state = withRuntimeSettings(t, state, 34, native)
			saved := false
			save := func(s ManagedState) error {
				for _, c := range s.Runtime.Containers {
					if c.Kind == "probe" {
						continue
					}
					b, _ := json.Marshal(c)
					var wire map[string]any
					json.Unmarshal(b, &wire)
					if wire["loom"] != float64(34) {
						t.Fatal("launch intent lacks selected loom before create", string(b))
					}
					if c.Kind == "http-boot" && native && wire["native_control"] != true {
						t.Fatal("launch intent lacks native opt-in before create", string(b))
					}
					saved = true
				}
				return persist(s)
			}
			state, _, err := applyManaged(context.Background(), "start", state, save, f.d)
			if err != nil || !saved {
				t.Fatal("start", err)
			}
			if native {
				for _, c := range f.containers {
					cfg := mapAt(c.doc, "Config")
					if c.kind != "probe" && (cfg["Entrypoint"].([]string)[0] != "/bin/sh" || !slices.Contains(cfg["Cmd"].([]string), `umask 077; exec /bin/urbit "$@"`)) {
						t.Fatal("native launch lacks fixed private umask wrapper", cfg)
					}
				}
			}
			for _, c := range f.containers {
				if c.kind != "probe" {
					assertSettingsArgs(t, mapAt(c.doc, "Config")["Cmd"].([]string), "34", native)
				}
			}
			state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			for _, kind := range []string{"pack", "meld", "roll"} {
				q, err := managedSession(&state, persist, f.d)
				if err != nil {
					t.Fatal(err)
				}
				_, args := q.command(kind)
				assertSettingsArgs(t, args, "34", false)
				if slices.Contains(args, "--no-conn") {
					t.Fatal("utility got runtime-only option")
				}
			}
			state, _, err = applyManaged(context.Background(), "pack", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			state, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range f.containers {
				assertSettingsArgs(t, mapAt(c.doc, "Config")["Cmd"].([]string), "34", native)
			}
		})
	}
}

func TestRuntimeSettingsLegacyLaunchNotRelabeled(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	// Strip fields exactly as a pre-settings JSON launch record lacks them.
	b, _ := json.Marshal(state)
	var wire map[string]any
	json.Unmarshal(b, &wire)
	runtime := wire["runtime"].(map[string]any)
	for _, raw := range runtime["containers"].([]any) {
		c := raw.(map[string]any)
		delete(c, "loom")
		delete(c, "native_control")
	}
	b, _ = json.Marshal(wire)
	json.Unmarshal(b, &state)
	state = withRuntimeSettings(t, state, 34, true)
	state, obs, err := observeManaged(context.Background(), state, f.d)
	if err != nil || obs.Health != "healthy" {
		t.Fatal("desired settings invalidated legacy launch", err, obs)
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
		assertSettingsArgs(t, mapAt(c.doc, "Config")["Cmd"].([]string), "34", true)
	}
}

func TestRuntimeSettingsStartupAndRoutine(t *testing.T) {
	f, s, p := managedFixture(t)
	s = withRuntimeSettings(t, s, 34, true)
	s, _, err := applyManaged(context.Background(), "start", s, p, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.containers {
		mapAt(c.doc, "State")["Running"] = false
	}
	s, obs, err := applyManaged(context.Background(), "startup-resume", s, p, f.d)
	if err != nil || obs.State != "running" {
		t.Fatal("startup", obs, err)
	}
	for _, c := range f.containers {
		assertSettingsArgs(t, mapAt(c.doc, "Config")["Cmd"].([]string), "34", true)
	}
	s.Routine = &routines.Run{Action: "meld", StopRunning: true, Resume: true, WasRunning: true, Deadline: time.Now().Add(time.Minute)}
	before := len(f.calls)
	s, obs, err = applyManaged(context.Background(), "routine-meld", s, p, f.d)
	if err != nil || obs.State != "running" || !s.Routine.Succeeded {
		t.Fatal("routine", obs, err)
	}
	seen := false
	for _, call := range f.calls[before:] {
		if len(call) > 1 && call[0] == "container" && call[1] == "create" && !slices.Contains(call, "/bin/sleep") {
			assertSettingsArgs(t, call, "34", true)
			if slices.Contains(call, "meld") {
				seen = true
				if slices.Contains(call, "urbit-native-control") {
					t.Fatal("offline meld enabled native control", call)
				}
			}
		}
	}
	if !seen {
		t.Fatal("routine never dispatched offline meld")
	}
}

func TestRuntimeSettingsContainmentRejectsDrift(t *testing.T) {
	for _, native := range []bool{false, true} {
		f, s, p := managedFixture(t)
		s = withRuntimeSettings(t, s, 34, native)
		s, _, err := applyManaged(context.Background(), "start", s, p, f.d)
		if err != nil {
			t.Fatal(err)
		}
		for _, change := range []string{"loom", "conn", "wrapper", "memory"} {
			f.editInspect = func(kind string, doc map[string]any) {
				if kind == "probe" {
					return
				}
				cfg := mapAt(doc, "Config")
				args := cfg["Cmd"].([]any)
				switch change {
				case "loom":
					for n := range args {
						if args[n] == "--loom" {
							args[n+1] = "31"
						}
					}
				case "conn":
					if native {
						args = append(args, "--no-conn")
					} else {
						for n := range args {
							if args[n] == "--no-conn" {
								args = append(args[:n], args[n+1:]...)
								break
							}
						}
					}
				case "wrapper":
					if native {
						args[1] = `umask 022; exec /bin/urbit "$@"`
					} else {
						cfg["Entrypoint"] = []any{"/bin/sh"}
					}
				case "memory":
					mapAt(doc, "HostConfig")["Memory"] = float64(8 << 30)
				}
				cfg["Cmd"] = args
			}
			if _, _, err := observeManaged(context.Background(), s, f.d); err == nil {
				t.Fatal("launch drift accepted", native, change)
			}
		}
	}
}

func TestRuntimeSettingsPrivateSocketMask(t *testing.T) {
	if path := os.Getenv("OMABIT_SOCKET_MASK_FIXTURE"); path != "" {
		listener, err := net.Listen("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		info, err := os.Lstat(path)
		if err != nil || info.Mode().Perm()&0077 != 0 {
			t.Fatal("socket was accessible to group/other", info, err)
		}
		return
	}
	f, s, p := managedFixture(t)
	s = withRuntimeSettings(t, s, 34, true)
	q, err := managedSession(&s, p, f.d)
	if err != nil {
		t.Fatal(err)
	}
	entry, args := q.command("http-restart")
	if entry != "/bin/sh" || len(args) < 3 || args[1] != `umask 077; exec /bin/urbit "$@"` {
		t.Fatal("native control missing fixed private launcher", entry, args)
	}
	// Exercise the exact mask and argument-forwarding prefix with an inert Go
	// socket producer instead of Vere. No runtime, Docker or real pier is used.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := strings.Replace(args[1], "/bin/urbit", `"$OMABIT_MASK_EXECUTABLE"`, 1)
	cmd := exec.Command(entry, "-c", script, args[2], "-test.run=^TestRuntimeSettingsPrivateSocketMask$")
	cmd.Env = append(os.Environ(), "OMABIT_MASK_EXECUTABLE="+exe, "OMABIT_SOCKET_MASK_FIXTURE="+filepath.Join(t.TempDir(), "conn.sock"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("private mask fixture: %s %v", out, err)
	}
}

func TestRuntimeSettingsExactCommands(t *testing.T) {
	for _, native := range []bool{false, true} {
		f, s, p := managedFixture(t)
		s = withRuntimeSettings(t, s, 34, native)
		q, err := managedSession(&s, p, f.d)
		if err != nil {
			t.Fatal(err)
		}
		for _, kind := range []string{"http-boot", "http-restart", "pack", "meld", "roll"} {
			wantEntry := "/bin/urbit"
			var want []string
			if kind == "http-boot" || kind == "http-restart" {
				want = []string{"-t", "--snap-time", "60", "-L"}
				if !native {
					want = append(want, "--no-conn")
				}
				want = append(want, "--no-dock", "--loom", "34", "-b", "0.0.0.0", "--http-port", "8080")
				if kind == "http-boot" {
					want = append(want, "-F", "zod", "-B", "/artifacts/boot.pill", "-c")
				}
				want = append(want, "/urbit/pier")
				if native {
					wantEntry = "/bin/sh"
					want = append([]string{"-c", `umask 077; exec /bin/urbit "$@"`, "urbit-native-control"}, want...)
				}
			} else {
				want = []string{kind, "--loom", "34", "/urbit/pier"}
			}
			entry, args := q.command(kind)
			if entry != wantEntry || !slices.Equal(args, want) {
				t.Fatalf("%s native=%t exact launch: %s %q; want %s %q", kind, native, entry, args, wantEntry, want)
			}
		}
	}
}

func TestRuntimeSettingsInvalidAdmission(t *testing.T) {
	for _, loom := range []int{19, 35, 255} {
		f, s, p := managedFixture(t)
		s = withRuntimeSettings(t, s, loom, false)
		if _, _, err := applyManaged(context.Background(), "start", s, p, f.d); err == nil {
			t.Fatal("invalid loom launched", loom)
		}
		if len(f.calls) != 0 {
			t.Fatal("invalid settings reached engine", f.calls)
		}
	}
	f, s, p := managedCometFixture(t)
	s, _, err := applyManaged(context.Background(), "start", s, p, f.d)
	if err != nil {
		t.Fatal(err)
	}
	q, err := managedSession(&s, p, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"probe", "comet-pack", "comet-meld", "comet-roll"} {
		s.Runtime.Containers[0].Kind = kind
		s.Runtime.Containers[0].Name = q.resourceName(kind)
		s.Runtime.Containers[0].NativeControl = true
		before := len(f.calls)
		if _, err := managedSession(&s, p, f.d); err == nil || err.Error() != "invalid recorded runtime settings" {
			t.Fatal("utility record native-control guard failed", kind, err)
		}
		if len(f.calls) != before {
			t.Fatal("invalid record reached engine")
		}
	}
}

func TestHistoricalSmallLoomObservationAndReplacement(t *testing.T) {
	f, state, persist := managedFixture(t)
	state.Loom = 29 // Existing RC.10 configuration; no new registration request.
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range state.Runtime.Containers {
		if c.Kind != "probe" && c.Loom != 29 {
			t.Fatal("historical launch relabeled", c)
		}
	}
	state.Loom = 31 // A new desired selection must not invalidate the old launch.
	state, obs, err := observeManaged(context.Background(), state, f.d)
	if err != nil || obs.Health != "healthy" {
		t.Fatal("historical runtime became unmanageable", obs, err)
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range state.Runtime.Containers {
		if c.Kind != "probe" && c.Loom != 31 {
			t.Fatal("replacement lost new selection", c)
		}
	}
}
