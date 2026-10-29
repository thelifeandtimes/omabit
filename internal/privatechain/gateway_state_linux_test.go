//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sys/unix"
)

func gatewayStateFixture(t *testing.T) (string, Plan, Workspace, Gateway, string) {
	t.Helper()
	directory, p, w := readyWorkspaceFixture(t)
	i := RuntimeInputInspection{Format: RuntimeInputFormat, ManifestSHA256: strings.Repeat("1", 64), FoundationDigest: p.FoundationDigest, Files: 12352, Bytes: 834036904}
	g, err := NewGateway(p, w, "gateway-state-operation", "engine-private-fixture", i)
	if err != nil {
		t.Fatal(err)
	}
	g.Phase = "start-dispatched"
	g.NetworkID = strings.Repeat("a", 64)
	g.ContainerID = strings.Repeat("b", 64)
	root := filepath.Join(directory, w.Binding.ProjectID)
	for _, name := range []string{"home", "chain", "secrets"} {
		mustInput(t, os.Mkdir(filepath.Join(root, name), 0700))
	}
	for _, name := range []string{"marker", "guest-namespaces.json", "gateway-ready.json", "chain/CURRENT", "secrets/0.generation-output", "secrets/1.generation-output", "secrets/zod.key", "secrets/nec.key"} {
		mustInput(t, os.WriteFile(filepath.Join(root, name), []byte("synthetic fixture only\n"), 0600))
	}
	return directory, p, w, g, root
}
func TestGatewayStateReadsMetadataAndPreservesState(t *testing.T) {
	d, p, w, g, root := gatewayStateFixture(t)
	before := workspaceFiles(t, d)
	v, err := OpenGatewayStateVerifier(context.Background(), d, p, w, g)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	v.filesystem.syncFile = func(*os.File) error { t.Fatal("unexpected sync"); return errors.New("forbidden") }
	p.Members[0].Ship = "~bud"
	w.Observation.Mounts[0].Inode++
	g.Phase = "planned"
	for range 2 {
		o, err := v.Verify(context.Background())
		if err != nil || !o.Private || len(o.Mounts) != 2 || !o.Root.SameObject(v.workspace.Root) || o.Chain.SameObject(o.Secrets) {
			t.Fatal(o, err)
		}
		o.Mounts[0].Inode++
	}
	if !reflect.DeepEqual(before, workspaceFiles(t, d)) {
		t.Fatal("changed tree")
	}
	data, err := os.ReadFile(filepath.Join(root, "secrets/zod.key"))
	if err != nil || string(data) != "synthetic fixture only\n" {
		t.Fatal("changed synthetic bytes")
	}
	if _, ok := any(v).(WorkspaceDriver); ok {
		t.Fatal("verifier has creation authority")
	}
	mustInput(t, v.Close())
	mustInput(t, v.Close())
	if _, err := v.Verify(context.Background()); err == nil {
		t.Fatal("closed handle accepted")
	}
}
func TestGatewayStateRefusesMissingUnsafeOrUnexpectedObjects(t *testing.T) {
	for _, kind := range []string{"empty-chain", "missing-key", "empty-key", "large-key", "large-tree", "symlink-key", "hardlink-key", "fifo", "public-file", "public-dir", "extra-root", "extra-secret", "nested-secret", "member-data", "missing-marker", "large-ready"} {
		t.Run(kind, func(t *testing.T) {
			d, p, w, g, root := gatewayStateFixture(t)
			key := filepath.Join(root, "secrets/zod.key")
			switch kind {
			case "empty-chain":
				mustInput(t, os.Remove(filepath.Join(root, "chain/CURRENT")))
			case "missing-key":
				mustInput(t, os.Remove(key))
			case "empty-key":
				mustInput(t, os.Truncate(key, 0))
			case "large-key":
				mustInput(t, os.Truncate(key, (1<<20)+1))
			case "large-tree":
				mustInput(t, os.Truncate(filepath.Join(root, "chain/CURRENT"), (1<<30)+1))
			case "symlink-key":
				mustInput(t, os.Remove(key))
				mustInput(t, os.Symlink("nec.key", key))
			case "hardlink-key":
				mustInput(t, os.Link(key, filepath.Join(root, "home/alias")))
			case "fifo":
				mustInput(t, unix.Mkfifo(filepath.Join(root, "home/pipe"), 0600))
			case "public-file":
				mustInput(t, os.Chmod(key, 0644))
			case "public-dir":
				mustInput(t, os.Chmod(filepath.Join(root, "chain"), 0755))
			case "extra-root":
				mustInput(t, os.WriteFile(filepath.Join(root, "unexpected"), []byte("retained"), 0600))
			case "extra-secret":
				mustInput(t, os.WriteFile(filepath.Join(root, "secrets/extra"), []byte("retained"), 0600))
			case "nested-secret":
				mustInput(t, os.Remove(key))
				mustInput(t, os.Mkdir(key, 0700))
			case "member-data":
				mustInput(t, os.WriteFile(filepath.Join(root, "piers", w.Mounts[0].InstanceID, "unexpected"), []byte("retained"), 0600))
			case "missing-marker":
				mustInput(t, os.Remove(filepath.Join(root, "marker")))
			case "large-ready":
				mustInput(t, os.Truncate(filepath.Join(root, "gateway-ready.json"), 65537))
			}
			before := workspaceFiles(t, d)
			if v, err := OpenGatewayStateVerifier(context.Background(), d, p, w, g); err == nil {
				v.Close()
				t.Fatal("accepted unsafe state")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, d)) {
				t.Fatal("refusal repaired state")
			}
		})
	}
}
func TestGatewayStateRefusesIdentityReplacement(t *testing.T) {
	for _, kind := range []string{"parent", "root", "piers", "member", "chain", "secrets"} {
		t.Run(kind, func(t *testing.T) {
			d, p, w, g, root := gatewayStateFixture(t)
			v, err := OpenGatewayStateVerifier(context.Background(), d, p, w, g)
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			target := map[string]string{"parent": d, "root": root, "piers": filepath.Join(root, "piers"), "member": filepath.Join(root, "piers", w.Mounts[0].InstanceID), "chain": filepath.Join(root, "chain"), "secrets": filepath.Join(root, "secrets")}[kind]
			retained := filepath.Join(filepath.Dir(d), "retained-"+kind)
			mustInput(t, os.Rename(target, retained))
			mustInput(t, os.Mkdir(target, 0700))
			// Preserve the full original contents under a same-shaped replacement.
			names, err := os.ReadDir(retained)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range names {
				mustInput(t, os.Rename(filepath.Join(retained, entry.Name()), filepath.Join(target, entry.Name())))
			}
			before := workspaceFiles(t, d)
			if _, err := v.Verify(context.Background()); err == nil {
				t.Fatal("adopted replacement")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, d)) {
				t.Fatal("changed replacement")
			}
			if kind != "chain" && kind != "secrets" {
				if reopened, err := OpenGatewayStateVerifier(context.Background(), d, p, w, g); err == nil {
					reopened.Close()
					t.Fatal("adopted retained workspace replacement")
				}
			}
		})
	}
}
func TestGatewayStateParentAliasAndCancelledPhaseRefuse(t *testing.T) {
	d, p, w, g, _ := gatewayStateFixture(t)
	for _, phase := range []string{"planned", "gateway-created", "ready"} {
		changed := g
		changed.Phase = phase
		if v, err := OpenGatewayStateVerifier(context.Background(), d, p, w, changed); err == nil {
			v.Close()
			t.Fatal("invalid phase accepted")
		}
	}
	alias := filepath.Join(filepath.Dir(d), "alias")
	mustInput(t, os.Symlink(d, alias))
	if v, err := OpenGatewayStateVerifier(context.Background(), alias, p, w, g); err == nil {
		v.Close()
		t.Fatal("alias accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenGatewayStateVerifier(ctx, d, p, w, g); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	v, err := OpenGatewayStateVerifier(context.Background(), d, p, w, g)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	if _, err := v.Verify(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestGatewayStateMetadataReadsAndCloseSerialize(t *testing.T) {
	d, p, w, g, _ := gatewayStateFixture(t)
	v, err := OpenGatewayStateVerifier(context.Background(), d, p, w, g)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 2 {
				if _, err := v.Verify(context.Background()); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	mustInput(t, v.Close())
}
