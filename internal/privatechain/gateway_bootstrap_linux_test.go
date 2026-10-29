//go:build linux

package privatechain

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func bootstrapFixture(t *testing.T) (string, Plan, Workspace, Gateway, *GatewayBootstrap) {
	t.Helper()
	directory, p, w := readyWorkspaceFixture(t)
	inputs := RuntimeInputInspection{Format: RuntimeInputFormat, ManifestSHA256: strings.Repeat("1", 64), FoundationDigest: p.FoundationDigest, Files: 12352, Bytes: 834036904}
	g, err := NewGateway(p, w, "gateway-bootstrap-operation", "engine-private-fixture", inputs)
	if err != nil {
		t.Fatal(err)
	}
	b, err := OpenGatewayBootstrap(context.Background(), directory, p, w, g)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return directory, p, w, g, b
}
func bootstrapDispatched(g Gateway) Gateway {
	g.Phase = "start-dispatched"
	g.NetworkID = strings.Repeat("a", 64)
	g.ContainerID = strings.Repeat("b", 64)
	return g
}
func bootstrapMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGatewayBootstrapCreatesOnlyJournalledStartupMetadata(t *testing.T) {
	directory, p, w, g, b := bootstrapFixture(t)
	before := workspaceFiles(t, directory)
	bootstrapMust(t, b.VerifyEmpty(context.Background()))
	if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
		t.Fatal("opening/checking changed workspace")
	}
	// An opened handle retains its own plan/workspace copies.
	p.Members[0].InstanceID = "caller-change"
	w.Observation.Mounts[0].Inode++
	g = bootstrapDispatched(g)
	bootstrapMust(t, b.Initialize(context.Background(), g))
	root := filepath.Join(directory, g.Selection.Binding.ProjectID)
	after := workspaceFiles(t, directory)
	want := map[string]os.FileMode{}
	for name, mode := range before {
		want[name] = mode
	}
	for name, mode := range map[string]os.FileMode{"home": os.ModeDir | 0700, "marker": 0600, "guest-namespaces.json": 0600} {
		want[filepath.Join(g.Selection.Binding.ProjectID, name)] = mode
	}
	if !reflect.DeepEqual(after, want) {
		t.Fatal("bootstrap changed unexpected entries", after, want)
	}
	marker, err := os.ReadFile(filepath.Join(root, "marker"))
	bootstrapMust(t, err)
	if string(marker) != gatewayBootstrapMarker {
		t.Fatal("marker differs")
	}
	data, err := os.ReadFile(filepath.Join(root, "guest-namespaces.json"))
	bootstrapMust(t, err)
	var ns map[string]string
	bootstrapMust(t, json.Unmarshal(data, &ns))
	if len(ns) != 3 {
		t.Fatal("unexpected namespace fields")
	}
	for _, kind := range []string{"net", "pid", "mnt"} {
		name, err := os.Readlink("/proc/self/ns/" + kind)
		bootstrapMust(t, err)
		if ns[kind] != name {
			t.Fatal("namespace not observed from current process")
		}
	}
	if err = b.Initialize(context.Background(), g); !errors.Is(err, ErrGatewayUnresolved) {
		t.Fatal("initialization repeated", err)
	}
	if err = b.VerifyEmpty(context.Background()); err == nil {
		t.Fatal("initialized tree represented as empty")
	}
	fresh := g
	fresh.Phase = "planned"
	fresh.NetworkID = ""
	fresh.ContainerID = ""
	if other, err := OpenGatewayBootstrap(context.Background(), directory, b.plan, b.workspace, fresh); err == nil {
		other.Close()
		t.Fatal("existing startup adopted")
	}
	if !reflect.DeepEqual(after, workspaceFiles(t, directory)) {
		t.Fatal("refusal changed retained startup")
	}
	bootstrapMust(t, b.Close())
	bootstrapMust(t, b.Close())
	if err = b.Initialize(context.Background(), g); err == nil {
		t.Fatal("closed handle initialized")
	}
}

func TestGatewayBootstrapRequiresExactCommittedStart(t *testing.T) {
	for _, kind := range []string{"planned", "gateway-created", "started", "foreign-operation", "foreign-inputs", "missing-container", "same-resource", "canceled", "closed"} {
		t.Run(kind, func(t *testing.T) {
			directory, _, _, g, b := bootstrapFixture(t)
			before := workspaceFiles(t, directory)
			g = bootstrapDispatched(g)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "planned":
				g.Phase = "planned"
				g.NetworkID = ""
				g.ContainerID = ""
			case "gateway-created", "started":
				g.Phase = kind
			case "foreign-operation":
				g.Selection.OperationID = "foreign-operation"
			case "foreign-inputs":
				g.Selection.Inputs.ManifestSHA256 = strings.Repeat("2", 64)
			case "missing-container":
				g.ContainerID = ""
			case "same-resource":
				g.ContainerID = g.NetworkID
			case "canceled":
				cancel()
			case "closed":
				bootstrapMust(t, b.Close())
			}
			if err := b.Initialize(ctx, g); err == nil {
				t.Fatal("unselected startup admitted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
				t.Fatal("rejected startup wrote files")
			}
		})
	}
}

func TestGatewayBootstrapKeepsPartialWritesAndNeverRetries(t *testing.T) {
	for _, kind := range []string{"home-sync", "marker-sync", "namespace-sync", "root-sync", "cancel-after-marker", "replace-root"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w, g, b := bootstrapFixture(t)
			g = bootstrapDispatched(g)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			root := filepath.Join(directory, w.Binding.ProjectID)
			retained := filepath.Join(filepath.Dir(directory), "retained-root")
			calls := 0
			err := b.initialize(ctx, g, func(f *os.File) error {
				calls++
				if kind == "replace-root" && calls == 1 {
					bootstrapMust(t, os.Rename(root, retained))
					bootstrapMust(t, os.Mkdir(root, 0700))
					return f.Sync()
				}
				if kind == "cancel-after-marker" && calls == 2 {
					cancel()
					return f.Sync()
				}
				failAt := map[string]int{"home-sync": 1, "marker-sync": 2, "namespace-sync": 3, "root-sync": 4}[kind]
				if calls == failAt {
					return errors.New("synthetic durability failure")
				}
				return f.Sync()
			})
			if !errors.Is(err, ErrGatewayUnresolved) {
				t.Fatal("partial failure lost uncertainty", err)
			}
			if kind == "cancel-after-marker" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation identity lost")
			}
			before := workspaceFiles(t, filepath.Dir(directory))
			if err = b.Initialize(context.Background(), g); !errors.Is(err, ErrGatewayUnresolved) {
				t.Fatal("partial initialization retried", err)
			}
			planned := g
			planned.Phase = "planned"
			planned.NetworkID = ""
			planned.ContainerID = ""
			if fresh, err := OpenGatewayBootstrap(context.Background(), directory, p, w, planned); err == nil {
				fresh.Close()
				t.Fatal("partial startup adopted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, filepath.Dir(directory))) {
				t.Fatal("retry changed retained evidence")
			}
			if kind == "replace-root" {
				entries, err := os.ReadDir(root)
				bootstrapMust(t, err)
				if len(entries) != 0 {
					t.Fatal("wrote into replacement root")
				}
				entries, err = os.ReadDir(retained)
				bootstrapMust(t, err)
				if len(entries) != 2 {
					t.Fatal("partial original not preserved")
				}
			} else {
				info, err := os.Stat(filepath.Join(root, "home"))
				bootstrapMust(t, err)
				if info.Mode() != os.ModeDir|0700 {
					t.Fatal("partial private home not retained")
				}
			}
		})
	}
}

func TestGatewayBootstrapRejectsNonemptyOrReplacedWorkspace(t *testing.T) {
	for _, kind := range []string{"foreign-file", "symlink", "permissions", "member-replacement"} {
		t.Run(kind, func(t *testing.T) {
			directory, _, w, g, b := bootstrapFixture(t)
			root := filepath.Join(directory, w.Binding.ProjectID)
			switch kind {
			case "foreign-file":
				bootstrapMust(t, os.WriteFile(filepath.Join(root, "marker"), []byte("retain me"), 0600))
			case "symlink":
				bootstrapMust(t, os.Symlink("piers", filepath.Join(root, "home")))
			case "permissions":
				bootstrapMust(t, os.Chmod(root, 0755))
			case "member-replacement":
				member := filepath.Join(root, "piers", w.Mounts[0].InstanceID)
				bootstrapMust(t, os.Rename(member, filepath.Join(filepath.Dir(directory), "retained-member")))
				bootstrapMust(t, os.Mkdir(member, 0700))
			}
			before := workspaceFiles(t, filepath.Dir(directory))
			if err := b.Initialize(context.Background(), bootstrapDispatched(g)); err == nil {
				t.Fatal("changed workspace initialized")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, filepath.Dir(directory))) {
				t.Fatal("refusal repaired tree")
			}
		})
	}
}

func TestGatewayBootstrapOpenRefusesRecoveryAndConcurrentClose(t *testing.T) {
	directory, p, w, g, b := bootstrapFixture(t)
	if other, err := OpenGatewayBootstrap(context.Background(), directory, p, w, bootstrapDispatched(g)); err == nil {
		other.Close()
		t.Fatal("recovery reopened mutation authority")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if other, err := OpenGatewayBootstrap(ctx, "/does-not-exist", p, w, g); !errors.Is(err, context.Canceled) {
		if other != nil {
			other.Close()
		}
		t.Fatal("canceled open did access", err)
	}
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() { _ = b.VerifyEmpty(context.Background()) })
	}
	wg.Go(func() { _ = b.Close() })
	wg.Wait()
	if err := b.Initialize(context.Background(), bootstrapDispatched(g)); err == nil {
		t.Fatal("closed handle retained authority")
	}
}

// The controller uses real bootstrap filesystem effects and an inert engine.
// This exercises the commit boundary; it does not claim Docker readiness.
type bootstrapControllerDriver struct {
	*gatewayHarness
	bootstrap *GatewayBootstrap
	failSync  bool
}

func (d *bootstrapControllerDriver) VerifyPreparation(ctx context.Context, g Gateway) error {
	if err := d.gatewayHarness.VerifyPreparation(ctx, g); err != nil {
		return err
	}
	return d.bootstrap.VerifyEmpty(ctx)
}

func (d *bootstrapControllerDriver) StartGateway(ctx context.Context, g Gateway) error {
	if d.saved.Phase != "start-dispatched" || d.saved != g {
		d.t.Fatal("filesystem initialization preceded recorded startup intent")
	}
	syncFile := (*os.File).Sync
	if d.failSync {
		syncFile = func(*os.File) error { return errors.New("synthetic bootstrap sync failure") }
	}
	if err := d.bootstrap.initialize(ctx, g, syncFile); err != nil {
		return err
	}
	return d.gatewayHarness.StartGateway(ctx, g)
}

func (d *bootstrapControllerDriver) ObserveGateway(context.Context, Gateway) (GatewayObservation, error) {
	return GatewayObservation{}, errors.New("inert driver has no runtime observation")
}

func TestGatewayBootstrapControllerKeepsJournalBeforeFilesystemEffects(t *testing.T) {
	for _, kind := range []string{"commit-failure", "lost-commit-reply", "bootstrap-failure", "start-failure", "lost-started-commit", "started"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w, g, b := bootstrapFixture(t)
			before := workspaceFiles(t, directory)
			h := &gatewayHarness{t: t, p: p, w: w, saved: g.Clone()}
			driver := &bootstrapControllerDriver{gatewayHarness: h, bootstrap: b}
			switch kind {
			case "commit-failure", "lost-commit-reply":
				h.fail = "commit-start-dispatched"
				h.loseCommit = kind == "lost-commit-reply"
			case "bootstrap-failure":
				driver.failSync = true
			case "start-failure":
				h.fail = "start-gateway"
			case "lost-started-commit":
				h.fail, h.loseCommit = "commit-started", true
			}
			_, err := ExecuteGateway(context.Background(), p, w, g, driver, h.commit, time.Now)
			if err == nil {
				t.Fatal("inert driver claimed readiness")
			}
			after := workspaceFiles(t, directory)
			switch kind {
			case "commit-failure", "lost-commit-reply":
				if h.startCalls != 0 || !reflect.DeepEqual(before, after) {
					t.Fatal("failed/lost commit performed startup effects")
				}
			case "bootstrap-failure":
				if h.startCalls != 0 || h.saved.Phase != "start-dispatched" || len(after) != len(before)+1 {
					t.Fatal("partial bootstrap not retained before container Start")
				}
			default:
				if h.startCalls != 1 || len(after) != len(before)+3 {
					t.Fatal("startup metadata/container Start count differs")
				}
			}
			want := "start-dispatched"
			if kind == "commit-failure" {
				want = "gateway-created"
			} else if kind == "lost-started-commit" || kind == "started" {
				want = "started"
			}
			if h.saved.Phase != want {
				t.Fatal("retained intent differs", h.saved.Phase, want)
			}
			starts := h.startCalls
			if _, err := ExecuteGateway(context.Background(), p, w, h.saved, driver, h.commit, time.Now); err == nil || h.startCalls != starts || !reflect.DeepEqual(after, workspaceFiles(t, directory)) {
				t.Fatal("retained uncertain startup repeated effects")
			}
		})
	}
}
