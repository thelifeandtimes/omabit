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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

func gatewayMemberStateFixture(t *testing.T, count int) (string, Plan, Workspace, Gateway, string, []GatewayMemberPier) {
	t.Helper()
	d, p, w, g, root, _ := memberKeyFixture(t)
	var members []GatewayMemberPier
	for _, m := range p.Members[:count] {
		pier := filepath.Join(root, "piers", m.InstanceID, strings.TrimPrefix(m.Ship, "~"))
		mustInput(t, os.MkdirAll(filepath.Join(pier, ".urb/log"), 0755))
		mustInput(t, os.Mkdir(filepath.Join(pier, ".urb/chk"), 0755))
		mustInput(t, os.WriteFile(filepath.Join(pier, ".urb/log/synthetic-event"), []byte("inert state"), 0600))
		id, err := pierstore.Identify(pier)
		mustInput(t, err)
		members = append(members, GatewayMemberPier{InstanceID: m.InstanceID, Identity: id})
	}
	return d, p, w, g, root, members
}
func TestGatewayMemberStateReadsOnlyRetainedLayout(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		t.Run(string(rune('0'+count)), func(t *testing.T) {
			d, p, w, g, root, members := gatewayMemberStateFixture(t, count)
			v, err := OpenMemberGatewayStateVerifier(context.Background(), d, p, w, g, members)
			mustInput(t, err)
			defer v.Close()
			before := workspaceFiles(t, d)
			o, err := v.Verify(context.Background())
			mustInput(t, err)
			if !o.Private || len(o.Mounts) != 2 || !reflect.DeepEqual(before, workspaceFiles(t, d)) {
				t.Fatal("changed read-only layout")
			}
			if count > 0 {
				pier := filepath.Join(root, "piers", members[0].InstanceID, strings.TrimPrefix(p.Members[0].Ship, "~"))
				// Payload is intentionally opaque: even a FIFO cannot block the gateway
				// reader. It is not a claim that this pier is safe to stop/resume.
				mustInput(t, unix.Mkfifo(filepath.Join(pier, ".urb/log/opaque-payload"), 0600))
				mustInput(t, os.WriteFile(filepath.Join(pier, ".vere.lock"), []byte("synthetic running lock"), 0600))
				mustInput(t, os.WriteFile(filepath.Join(pier, ".urb/log/synthetic-event"), []byte("updated inert state"), 0600))
				members[0].Identity.Inode++
				g.Observation.Chain.Inode++
				w.Observation.Mounts[0].Inode++
				p.Members[0].Ship = "~bud"
				_, err = v.Verify(context.Background())
				mustInput(t, err)
			}
			mustInput(t, v.Close())
			mustInput(t, v.Close())
			if _, err = v.Verify(context.Background()); err == nil {
				t.Fatal("closed reader accepted")
			}
		})
	}
}
func TestGatewayMemberStateRejectsUnboundOrIncompleteSelection(t *testing.T) {
	for _, kind := range []string{"phase", "project", "unknown", "duplicate", "too-many", "omitted", "wrong-pier", "wrong-urb", "shared", "reserved", "zero", "other-device", "chain-identity", "secrets-identity", "alias"} {
		t.Run(kind, func(t *testing.T) {
			d, p, w, g, _, members := gatewayMemberStateFixture(t, 2)
			switch kind {
			case "phase":
				g.Phase = "started"
				g.Observation = nil
			case "project":
				g.Selection.Binding.ProjectID = "private-project-other"
			case "unknown":
				members[0].InstanceID = "unselected"
			case "duplicate":
				members[1] = members[0]
			case "too-many":
				members = append(members, members[0])
			case "omitted":
				members = members[:1]
			case "wrong-pier":
				members[0].Identity.Inode++
			case "wrong-urb":
				members[0].Identity.UrbInode++
			case "shared":
				members[1].Identity = members[0].Identity
			case "reserved":
				members[0].Identity.Inode = w.Root.Inode
				members[0].Identity.Birth = w.Root.Birth
			case "zero":
				members[0].Identity = pierstore.Identity{}
			case "other-device":
				members[0].Identity.Device++
			case "chain-identity":
				g.Observation.Chain.Inode++
			case "secrets-identity":
				g.Observation.Secrets.Inode++
			case "alias":
				alias := filepath.Join(filepath.Dir(d), "alias")
				mustInput(t, os.Symlink(d, alias))
				d = alias
			}
			before := workspaceFiles(t, d)
			if v, err := OpenMemberGatewayStateVerifier(context.Background(), d, p, w, g, members); err == nil {
				v.Close()
				t.Fatal("unbound selection accepted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, d)) {
				t.Fatal("refusal changed state")
			}
		})
	}
}
func TestGatewayMemberStateRejectsChangedObjectsAndUnselectedWrites(t *testing.T) {
	for _, kind := range []string{"pier", "urb", "chain", "secrets", "extra-member-child", "extra-peer-child", "extra-secret", "public-key", "symlink-urb"} {
		t.Run(kind, func(t *testing.T) {
			d, p, w, g, root, members := gatewayMemberStateFixture(t, 1)
			v, err := OpenMemberGatewayStateVerifier(context.Background(), d, p, w, g, members)
			mustInput(t, err)
			defer v.Close()
			mount := filepath.Join(root, "piers", members[0].InstanceID)
			pier := filepath.Join(mount, strings.TrimPrefix(p.Members[0].Ship, "~"))
			if kind == "extra-member-child" {
				mustInput(t, os.Mkdir(filepath.Join(mount, "unexpected"), 0700))
			} else if kind == "extra-peer-child" {
				mustInput(t, os.WriteFile(filepath.Join(root, "piers", p.Members[1].InstanceID, "unexpected"), []byte("retained"), 0600))
			} else if kind == "extra-secret" {
				mustInput(t, os.WriteFile(filepath.Join(root, "secrets/unexpected"), []byte("retained"), 0600))
			} else if kind == "public-key" {
				mustInput(t, os.Chmod(filepath.Join(root, "secrets/zod.key"), 0644))
			} else {
				target := map[string]string{"pier": pier, "urb": filepath.Join(pier, ".urb"), "chain": filepath.Join(root, "chain"), "secrets": filepath.Join(root, "secrets"), "symlink-urb": filepath.Join(pier, ".urb")}[kind]
				kept := filepath.Join(t.TempDir(), "retained")
				mustInput(t, os.Rename(target, kept))
				if kind == "symlink-urb" {
					mustInput(t, os.Symlink(kept, target))
				} else {
					mustInput(t, os.Mkdir(target, 0700))
					entries, err := os.ReadDir(kept)
					mustInput(t, err)
					for _, entry := range entries {
						mustInput(t, os.Rename(filepath.Join(kept, entry.Name()), filepath.Join(target, entry.Name())))
					}
				}
			}
			before := workspaceFiles(t, d)
			if _, err = v.Verify(context.Background()); err == nil {
				t.Fatal("changed state accepted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, d)) {
				t.Fatal("reader repaired state")
			}
		})
	}
}
func TestGatewayMemberStateCloseAndCancellation(t *testing.T) {
	d, p, w, g, _, members := gatewayMemberStateFixture(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenMemberGatewayStateVerifier(ctx, d, p, w, g, members); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	v, err := OpenMemberGatewayStateVerifier(context.Background(), d, p, w, g, members)
	mustInput(t, err)
	if _, err = v.Verify(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = v.Verify(context.Background()); _ = v.Close() }()
	}
	wg.Wait()
	mustInput(t, v.Close())
}
