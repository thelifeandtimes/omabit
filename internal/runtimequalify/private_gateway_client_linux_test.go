//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func privateClientFixture(t *testing.T) (*qualifier, privatechain.Gateway) {
	t.Helper()
	f, _, _ := managedFixture(t)
	_, _, g, _, _ := privateNetworkFixture(t)
	q := &qualifier{d: f.d, base: filepath.Join(f.d.home, ".local/share/urbit-runner/managed"), phase: "private-client-fixture"}
	return q, g
}
func TestPrivateGatewayClientExclusiveCreationAndReadOnlyReopen(t *testing.T) {
	q, g := privateClientFixture(t)
	c, err := openPrivateGatewayClient(context.Background(), q, g, true)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if q.config != filepath.Join(q.r.Directory, "docker-config") || c.current() != nil {
		t.Fatal("unbound client paths")
	}
	before, err := os.ReadFile(filepath.Join(q.r.Directory, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = openPrivateGatewayClient(context.Background(), q, g, true); err == nil {
		t.Fatal("adopted existing client")
	}
	g.Phase = "started"
	g.NetworkID = strings.Repeat("a", 64)
	g.ContainerID = strings.Repeat("b", 64)
	q2, _ := privateClientFixture(t)
	q2.base = q.base
	c2, err := openPrivateGatewayClient(context.Background(), q2, g, false)
	if err != nil {
		t.Fatal(err)
	}
	if c2.current() != nil {
		t.Fatal("recovery client differs")
	}
	if err = c2.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(q.r.Directory, "session.json"))
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("recovery changed marker")
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil || c.current() == nil {
		t.Fatal("closed handle remained usable")
	}
	if _, err = os.Stat(q.r.Directory); err != nil {
		t.Fatal("Close removed retained evidence")
	}
}

func TestPrivateGatewayClientRefusesDriftWithoutRepair(t *testing.T) {
	for _, mode := range []string{"client-replaced", "config-replaced", "marker-replaced", "marker-changed", "client-mode", "config-mode", "marker-mode", "extra-config", "extra-root", "config-link", "marker-link", "marker-hardlink", "base-replaced"} {
		t.Run(mode, func(t *testing.T) {
			q, g := privateClientFixture(t)
			c, err := openPrivateGatewayClient(context.Background(), q, g, true)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			config := q.config
			marker := filepath.Join(q.r.Directory, "session.json")
			switch mode {
			case "client-replaced", "config-replaced", "base-replaced":
				p := q.r.Directory
				if mode == "config-replaced" {
					p = config
				}
				if mode == "base-replaced" {
					p = q.base
				}
				if err = os.Rename(p, p+"-retained"); err != nil {
					t.Fatal(err)
				}
				if err = os.Mkdir(p, 0700); err != nil {
					t.Fatal(err)
				}
			case "marker-replaced":
				if err = os.Rename(marker, marker+"-retained"); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(marker, c.marker, 0600); err != nil {
					t.Fatal(err)
				}
			case "marker-changed":
				if err = os.WriteFile(marker, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "client-mode":
				err = os.Chmod(q.r.Directory, 0750)
			case "config-mode":
				err = os.Chmod(config, 0750)
			case "marker-mode":
				err = os.Chmod(marker, 0640)
			case "extra-config":
				err = os.WriteFile(filepath.Join(config, "config.json"), []byte("{}"), 0600)
			case "extra-root":
				err = os.WriteFile(filepath.Join(q.r.Directory, "unexpected"), nil, 0600)
			case "config-link":
				if err = os.Rename(config, config+"-retained"); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(config+"-retained", config)
			case "marker-link":
				if err = os.Rename(marker, marker+"-retained"); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(marker+"-retained", marker)
			case "marker-hardlink":
				err = os.Link(marker, marker+"-alias")
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.current() == nil {
				t.Fatal("changed client metadata accepted")
			}
		})
	}
}

func TestPrivateGatewayClientNeverAdoptsMissingPartialOrForeignSelection(t *testing.T) {
	for _, mode := range []string{"missing-recovery", "partial-fresh", "different-selection", "planned-recovery", "started-fresh", "invalid-operation", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			q, g := privateClientFixture(t)
			fresh := true
			ctx := context.Background()
			path := filepath.Join(q.base, "private-gateway-client-"+g.Selection.OperationID)
			switch mode {
			case "missing-recovery":
				fresh = false
				g.Phase = "started"
			case "partial-fresh":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "different-selection":
				c, err := openPrivateGatewayClient(ctx, q, g, true)
				if err != nil {
					t.Fatal(err)
				}
				c.Close()
				g.Selection.Binding.ProjectID = "private-project-other"
				g.Phase = "started"
				fresh = false
			case "planned-recovery":
				fresh = false
			case "started-fresh":
				g.Phase = "started"
			case "invalid-operation":
				g.Selection.OperationID = "../bad"
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if c, err := openPrivateGatewayClient(ctx, q, g, fresh); err == nil || c != nil {
				t.Fatal("invalid setup acquired client")
			}
			if mode != "partial-fresh" && mode != "different-selection" {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("invalid setup created state", err)
				}
			}
		})
	}
}

func TestPrivateGatewayClientChecksEveryCommandAndRetainsKnownID(t *testing.T) {
	for _, changedAfter := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "after"}[changedAfter], func(t *testing.T) {
			q, g := privateClientFixture(t)
			calls := 0
			id := strings.Repeat("a", 64)
			q.d.execute = func(context.Context, []string, []string) (string, error) {
				calls++
				if changedAfter {
					if err := os.WriteFile(filepath.Join(q.config, "config.json"), nil, 0600); err != nil {
						t.Fatal(err)
					}
				}
				return id, nil
			}
			c, err := openPrivateGatewayClient(context.Background(), q, g, true)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			if !changedAfter {
				if err = os.WriteFile(filepath.Join(q.config, "config.json"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := q.dockerCreateID(context.Background(), q.d.times.command, "network", "create", "fixture")
			if err == nil {
				t.Fatal("changed config accepted")
			}
			if changedAfter {
				if got != id || calls != 1 {
					t.Fatal("known outcome lost", got, calls)
				}
			} else if got != "" || calls != 0 {
				t.Fatal("changed config reached command")
			}
		})
	}
}

func TestPrivateGatewayClientCloseAndChecksSerialize(t *testing.T) {
	q, g := privateClientFixture(t)
	c, err := openPrivateGatewayClient(context.Background(), q, g, true)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for n := 0; n < 6; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 10; n++ {
				_ = c.current()
			}
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); _ = c.Close() }()
	wg.Wait()
	if c.current() == nil {
		t.Fatal("closed client accepted")
	}
}
