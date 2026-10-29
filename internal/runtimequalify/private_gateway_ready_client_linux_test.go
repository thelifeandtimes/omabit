//go:build linux && amd64

package runtimequalify

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The first member factory consumes a validated ready gateway. Reusing its
// original client must accept that lifecycle phase without writing metadata or
// calling Docker. The factory separately validates complete readiness evidence.
func TestPrivateGatewayReadyClientReusesExactMetadata(t *testing.T) {
	for _, mode := range []string{"retained", "missing", "foreign-selection", "changed-marker", "fresh-refused"} {
		t.Run(mode, func(t *testing.T) {
			q, g := privateClientFixture(t)
			calls := 0
			q.d.execute = func(context.Context, []string, []string) (string, error) { calls++; return "", nil }
			var marker string
			var before []byte
			var info os.FileInfo
			if mode != "missing" {
				c, err := openPrivateGatewayClient(context.Background(), q, g, true)
				if err != nil {
					t.Fatal(err)
				}
				marker = filepath.Join(q.r.Directory, "session.json")
				if err = c.Close(); err != nil {
					t.Fatal(err)
				}
				if mode == "changed-marker" {
					if err = os.WriteFile(marker, []byte("changed"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				before, err = os.ReadFile(marker)
				if err != nil {
					t.Fatal(err)
				}
				info, err = os.Stat(marker)
				if err != nil {
					t.Fatal(err)
				}
			}
			g.Phase = "ready"
			if mode == "foreign-selection" {
				g.Selection.Binding.ProjectID = "private-project-other"
			}
			c, err := openPrivateGatewayClient(context.Background(), q, g, mode == "fresh-refused")
			if mode == "retained" {
				if err != nil || c == nil {
					t.Fatalf("ready gateway cannot reuse its client: %v", err)
				}
				if c.current() != nil {
					t.Fatal("reused metadata differs")
				}
				if err = c.Close(); err != nil {
					t.Fatal(err)
				}
			} else if err == nil || c != nil {
				t.Fatal("unowned ready client admitted")
			}
			if calls != 0 {
				t.Fatal("client reuse reached engine")
			}
			if marker != "" {
				after, err := os.ReadFile(marker)
				if err != nil {
					t.Fatal(err)
				}
				current, err := os.Stat(marker)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) || !os.SameFile(info, current) || info.Mode() != current.Mode() || info.ModTime() != current.ModTime() {
					t.Fatal("reuse changed retained metadata")
				}
			} else if _, err := os.Lstat(filepath.Join(q.base, "private-gateway-client-"+g.Selection.OperationID)); !os.IsNotExist(err) {
				t.Fatal("missing ready client was created")
			}
		})
	}
}
