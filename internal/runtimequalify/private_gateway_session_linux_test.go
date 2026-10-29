//go:build linux && amd64

package runtimequalify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func TestPrivateGatewaySessionFixedInputSelection(t *testing.T) {
	h := strings.Repeat("a", 64)
	got, err := privateGatewayInputPath("/owned/managed", h)
	if err != nil || got != "/owned/managed/private-runtime-inputs/"+h {
		t.Fatal(got, err)
	}
	for _, bad := range []string{"", "../escape", strings.Repeat("0", 64), strings.Repeat("A", 64), strings.Repeat("b", 63)} {
		if _, err = privateGatewayInputPath("/owned/managed", bad); err == nil {
			t.Fatal("invalid manifest admitted")
		}
	}
	for _, bad := range []string{"relative", "/owned/../other", "/path,options", "/"} {
		if _, err = privateGatewayInputPath(bad, h); err == nil {
			t.Fatal("invalid path admitted")
		}
	}
}

func TestPrivateGatewaySessionRefusesWrongOrChangedStateRoot(t *testing.T) {
	for _, mode := range []string{"wrong", "alias", "mode", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			q, _ := privateClientFixture(t)
			root := q.base
			ctx := context.Background()
			switch mode {
			case "wrong":
				root = filepath.Dir(root)
			case "alias":
				if err := os.Rename(root, root+"-retained"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(root+"-retained", root); err != nil {
					t.Fatal(err)
				}
			case "mode":
				if err := os.Chmod(root, 0755); err != nil {
					t.Fatal(err)
				}
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if got, err := privateGatewaySession(ctx, root, q.d); err == nil || got != nil {
				t.Fatal("foreign session admitted")
			}
		})
	}
}

func TestPrivateGatewaySessionInvalidSelectionHasNoMetadataOrEngineEffect(t *testing.T) {
	for _, mode := range []string{"workspace", "manifest", "start-phase", "start-inputs", "recovery-phase", "recovery-operation"} {
		t.Run(mode, func(t *testing.T) {
			q, g := privateClientFixture(t)
			p, w, _, _, _ := privateNetworkFixture(t)
			calls := 0
			q.d.execute = func(context.Context, []string, []string) (string, error) {
				calls++
				t.Error("invalid selection reached engine")
				return "", nil
			}
			var err error
			switch mode {
			case "workspace":
				w.Phase = "planned"
				_, _, err = inspectPrivateGatewayInputs(context.Background(), q.base, "/absent", p, w, g.Selection.Inputs.ManifestSHA256, q.d)
			case "manifest":
				_, _, err = inspectPrivateGatewayInputs(context.Background(), q.base, "/absent", p, w, "../bad", q.d)
			case "start-phase":
				g.Phase = "started"
				_, err = openManagedPrivateGateway(context.Background(), q.base, "/absent", p, w, g, q.d)
			case "start-inputs":
				g.Selection.Inputs.RuntimeAuthorized = true
				_, err = openManagedPrivateGateway(context.Background(), q.base, "/absent", p, w, g, q.d)
			case "recovery-phase":
				_, err = openManagedPrivateGatewayObserver(context.Background(), q.base, "/absent", p, w, g, "recovery", q.d)
			case "recovery-operation":
				g.Phase = "started"
				g.NetworkID = strings.Repeat("a", 64)
				g.ContainerID = strings.Repeat("b", 64)
				_, err = openManagedPrivateGatewayObserver(context.Background(), q.base, "/absent", p, w, g, g.Selection.OperationID, q.d)
			}
			if err == nil || calls != 0 {
				t.Fatal("invalid operation admitted", err, calls)
			}
			entries, e := os.ReadDir(q.base)
			if e != nil || len(entries) != 0 {
				t.Fatal("invalid selection created state", entries, e)
			}
		})
	}
}

func TestPrivateGatewaySessionObserverInterfaceHasNoEffects(t *testing.T) {
	var o any = &privateSessionObserver{}
	if _, ok := o.(privatechain.GatewayDriver); ok {
		t.Fatal("observer exposes Start/create methods")
	}
}
