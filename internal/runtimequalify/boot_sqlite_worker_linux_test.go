//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The managed SQLite integration suite drives this test-only child over pipes.
// Every runtime call is the actual driver with the existing isolated OS/engine
// fixtures. Persistence callbacks synchronously cross into the parent's real
// store; no production injection API or live Docker/service is involved.
type bootSQLiteWire struct {
	Kind        string             `json:"kind"`
	Action      string             `json:"action,omitempty"`
	Cut         string             `json:"cut,omitempty"`
	Directory   string             `json:"directory,omitempty"`
	State       ManagedState       `json:"state"`
	Observation ManagedObservation `json:"observation"`
	Error       string             `json:"error,omitempty"`
	Events      []string           `json:"events,omitempty"`
}

type bootSQLiteCrash struct{}

func TestBootSQLiteDriverWorker(t *testing.T) {
	mode := os.Getenv("OMABIT_TEST_BOOT_SQLITE_WORKER")
	if mode == "" {
		t.Skip("test-only driver for the managed SQLite integration suite")
	}
	fixture := managedCometFixture
	if mode == "fake" {
		fixture = managedFixture
	} else if mode != "comet" && mode != "owned" {
		t.Fatal("unknown worker fixture")
	}
	f, _, report := fixture(t)
	f.d.logs = func(context.Context, string, string) (string, error) { return "fixture logs", nil }
	encode, decode := json.NewEncoder(os.Stdout), json.NewDecoder(os.Stdin)
	send := func(message bootSQLiteWire) {
		if err := encode.Encode(message); err != nil {
			t.Fatal(err)
		}
	}
	send(bootSQLiteWire{Kind: "ready", Directory: filepath.Join(f.d.home, ".local/share/urbit-runner/managed")})
	execute, identity := f.d.execute, f.d.hostIdentity
	for {
		var request bootSQLiteWire
		if err := decode.Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Kind == "close" {
			return
		}
		f.d.hostPort = request.State.Port
		f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
			if mode == "fake" {
				return "~zod", f.d.hostProbe(ctx, endpoint)
			}
			ship, err := identity(ctx, endpoint)
			if mode == "owned" {
				ship = request.State.NativeShip
			}
			return ship, err
		}
		f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
			bootCreate := len(args) > 7 && args[4] == "container" && args[5] == "create" &&
				(strings.HasSuffix(args[7], "-comet-boot") || strings.HasSuffix(args[7], "-http-boot"))
			out, err := execute(ctx, args, env)
			if err == nil && bootCreate && request.Cut == "create-response" {
				return "", errors.New("fixture actual create response lost")
			}
			return out, err
		}
		result := func() (response bootSQLiteWire) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if _, ok := recovered.(bootSQLiteCrash); !ok {
						panic(recovered)
					}
					// Abort the runner call without returning a result/finish. The
					// fixture engine survives just as an independent daemon would.
					response = bootSQLiteWire{Kind: "crashed"}
				}
			}()
			var state ManagedState
			var observed ManagedObservation
			var err error
			if request.Kind == "observe" {
				state, observed, err = observeManaged(context.Background(), request.State, f.d)
			} else if request.Kind == "apply" {
				state, observed, err = applyManaged(context.Background(), request.Action, request.State, func(state ManagedState) error {
					if err := report(state); err != nil {
						return err
					}
					send(bootSQLiteWire{Kind: "persist", State: state})
					var reply bootSQLiteWire
					if err := decode.Decode(&reply); err != nil {
						t.Fatal(err)
					}
					switch reply.Kind {
					case "ok":
						return nil
					case "error":
						return errors.New("fixture managed resource journal error")
					case "crash":
						panic(bootSQLiteCrash{})
					default:
						t.Fatal("unknown persistence response")
						return errors.New("unknown persistence response")
					}
				}, f.d)
			} else {
				t.Fatal("unknown driver command")
			}
			response = bootSQLiteWire{Kind: "result", State: state, Observation: observed}
			if err != nil {
				response.Error = err.Error()
			}
			return response
		}()
		result.Events = append([]string(nil), f.events...)
		send(result)
	}
}
