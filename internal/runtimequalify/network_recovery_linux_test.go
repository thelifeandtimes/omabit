//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestManagedStopReconcilesOnlyJournaledOrphanEndpoint(t *testing.T) {
	for _, mode := range []string{"recover", "foreign-name", "wrong-endpoint", "name-reused", "id-present", "list-error", "disconnect-error", "retained"} {
		t.Run(mode, func(t *testing.T) {
			f, s, save := managedFixture(t)
			var err error
			s, _, err = applyManaged(context.Background(), "start", s, save, f.d)
			if err != nil {
				t.Fatal(err)
			}
			s, _, err = applyManaged(context.Background(), "stop", s, save, f.d)
			if err != nil {
				t.Fatal(err)
			}
			c := s.Runtime.Containers[len(s.Runtime.Containers)-1]
			n := f.networks[s.Runtime.Network.Name]
			endpoint := strings.Repeat("b", 64)
			name := c.Name
			if mode == "foreign-name" {
				name = "unrelated"
			}
			key := "ep-" + endpoint
			if mode == "wrong-endpoint" {
				key = "ep-" + strings.Repeat("c", 64)
			}
			mapAt(n.doc, "Containers")[key] = map[string]any{"Name": name, "EndpointID": endpoint}
			base := f.d.execute
			disconnects := 0
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				if len(a) > 1 && a[0] == "container" && a[1] == "ls" {
					if mode == "list-error" {
						return "", errors.New("inventory unavailable")
					}
					if mode == "name-reused" && strings.Contains(strings.Join(a, " "), "name=") || mode == "id-present" && strings.Contains(strings.Join(a, " "), "id=") {
						return strings.Repeat("d", 64), nil
					}
					return "", nil
				}
				if len(a) > 1 && a[0] == "network" && a[1] == "disconnect" {
					disconnects++
					if len(a) != 5 || a[2] != "--force" || a[3] != n.id || a[4] != c.Name {
						t.Fatal("wrong recovery target", a)
					}
					if mode == "disconnect-error" {
						return "", errors.New("lost reply")
					}
					if mode != "retained" {
						delete(mapAt(n.doc, "Containers"), key)
					}
					return "", nil
				}
				return base(ctx, args, env)
			}
			_, before, err := observeManaged(context.Background(), s, f.d)
			if err == nil || !before.RecoveryRequired || disconnects != 0 {
				t.Fatal("observation repaired or trusted dangling endpoint", before, err)
			}
			next, obs, err := applyManaged(context.Background(), "stop", s, save, f.d)
			if mode == "recover" {
				if err != nil || obs.RecoveryRequired || disconnects != 1 {
					t.Fatal(obs, err, disconnects)
				}
				_, obs, err = observeManaged(context.Background(), next, f.d)
				if err != nil || obs.State != "stopped" || obs.RecoveryRequired {
					t.Fatal("recovery did not survive polling", obs, err)
				}
			} else if err == nil || !obs.RecoveryRequired {
				t.Fatal("unsafe recovery accepted", mode, obs, err)
			}
			if mode != "recover" && mode != "disconnect-error" && mode != "retained" && disconnects != 0 {
				t.Fatal("foreign or unverified attachment changed")
			}
		})
	}
}
