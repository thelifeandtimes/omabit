//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func completeReplacementsForReadiness(t *testing.T) *anchorReplacementAdapterFixture {
	t.Helper()
	f := newAnchorReplacementAdapterFixture(t)
	h := anchorDrainHistory(f.selection.Source)
	for !f.saved.Running() {
		e := f.open(t, f.saved)
		_, err := projectnetwork.ExecuteNextAnchorReplacement(context.Background(), h, f.saved, e, func(seq uint64, next projectnetwork.AnchorReplacement) error {
			if seq != f.saved.Sequence {
				t.Fatal("stale fixture commit")
			}
			f.saved = next.Clone()
			return nil
		}, time.Now)
		if err != nil {
			t.Fatal(err)
		}
	}
	return f
}
func TestProjectAnchorRecoveryReadinessAdapter(t *testing.T) {
	modes := []string{"pass", "caller-alias", "wrong-last-ship", "http-error", "first-restarted-before", "last-restarted-before", "first-restarted-after", "last-restarted-after", "anchor-process", "anchor-namespace", "publication", "engine-after-http", "old-container-reappeared", "changed-config", "urb-replaced", "unobserved-http", "arbitrary-endpoint", "stale-http", "caller-selection", "cancelled"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			f := completeReplacementsForReadiness(t)
			x := f.f.x
			h := anchorDrainHistory(f.selection.Source)
			s := f.saved.Clone()
			original := s.Clone()
			source := ProjectAnchorRecoveryReadinessSelection{Source: f.selection.Source, Replacement: s.Clone()}
			originalExecute := x.f.f.d.execute
			x.f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				namespaceRead := len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"})
				if len(a) < 2 || !(namespaceRead || a[0] == "image" && a[1] == "inspect" || a[0] == "network" && a[1] == "inspect" || a[0] == "container" && (a[1] == "inspect" || a[1] == "ls")) {
					t.Fatal("readiness attempted mutation", a)
				}
				return originalExecute(ctx, args, env)
			}
			raw, err := openProjectAnchorRecoveryReadiness(context.Background(), source, x.f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectAnchorRecoveryReadinessEngine)
			// Keep collector authority independent while mutating the caller-owned input.
			h = e.history.Clone()
			e.review.source.anchor.readBootID = func() (string, error) { return x.f.ns.BootID, nil }
			// Every fault starts from an independently successful observation.
			if _, err = e.InspectRecoveryReadiness(context.Background(), s); err != nil {
				t.Fatal("preflight", err)
			}
			e.inspectedAt = time.Time{}
			signals, removes, creates, starts := x.f.signals, len(f.f.removes), f.creates, f.starts
			sentinel := x.sentinel
			beforeBytes, err := os.ReadFile(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			beforeStat, err := os.Stat(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reads := 0
			restart := func(index int) {
				x.f.containers[s.Members[index].Container.ID].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
			e.review.source.anchor.base.q.d.hostIdentity = func(_ context.Context, endpoint string) (string, error) {
				reads++
				ship := e.endpoints[endpoint]
				if ship == "" {
					t.Fatal("unselected endpoint")
				}
				switch mode {
				case "wrong-last-ship":
					if reads == len(s.Members) {
						return "~bus", nil
					}
				case "http-error":
					return "", errors.New("private HTTP error")
				case "first-restarted-after":
					restart(0)
				case "last-restarted-after":
					restart(len(s.Members) - 1)
				case "anchor-process", "anchor-namespace":
					x.fault = mode
				case "publication":
					x.fault = "missing-publication"
				case "engine-after-http":
					x.f.f.engine.ID = "changed-engine"
				case "old-container-reappeared":
					v := projectMemberContainerFixture(t, f.contracts[0])
					v.ID = s.Retirement.Namespace.Drain.Review.Context.Members[0].Context.Member.Container.ID
					v.Name = "/foreign"
					v.HostConfig.NetworkMode = "none"
					x.f.containers[v.ID] = &v
				case "changed-config":
					x.f.containers[s.Members[0].Container.ID].HostConfig.Privileged = true
				case "urb-replaced":
					if reads == 1 {
						p := filepath.Join(x.f.paths[s.Members[0].Creation.InstanceID], ".urb")
						if err := os.Rename(p, p+"-retained"); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(p, 0700); err != nil {
							t.Fatal(err)
						}
						if strings.HasPrefix(sentinel, p+"/") {
							sentinel = p + "-retained" + strings.TrimPrefix(sentinel, p)
						}
					}
				case "cancelled":
					cancel()
				}
				return ship, nil
			}
			switch mode {
			case "first-restarted-before":
				restart(0)
			case "last-restarted-before":
				restart(len(s.Members) - 1)
			case "caller-alias":
				source.Replacement.Members[0].Creation.Storage.Path = "changed"
				source.Source.Launches[0].Storage.RuntimeID = "changed"
			}
			if mode == "unobserved-http" || mode == "arbitrary-endpoint" || mode == "stale-http" {
				endpoint := fmt.Sprintf("http://127.0.0.1:%d/~/host", h.Plan.Members[0].HTTPPort)
				if mode != "unobserved-http" {
					if _, err = e.InspectRecoveryReadiness(ctx, s); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "arbitrary-endpoint" {
					endpoint = "http://127.0.0.1:9/~/host"
				}
				if mode == "stale-http" {
					e.inspectedAt = time.Now().Add(-time.Minute)
				}
				if _, err = e.ReadHostIdentity(ctx, endpoint); err == nil || reads != 0 {
					t.Fatal("unbound HTTP allowed")
				}
			} else if mode == "caller-selection" {
				altered := s.Clone()
				altered.Members[0].Container.ID = strings.Repeat("e", 64)
				if _, err = e.InspectRecoveryReadiness(ctx, altered); err == nil {
					t.Fatal("changed selection accepted")
				}
			} else {
				result, err := projectnetwork.CollectAnchorRecoveryReadiness(ctx, h, s, e, time.Now)
				pass := mode == "pass" || mode == "caller-alias"
				if pass {
					if err != nil || result.Validate(h) != nil || reads != len(s.Members) {
						t.Fatal("complete readiness failed", err)
					}
				} else {
					if err == nil || !reflect.DeepEqual(result, projectnetwork.AnchorRecoveryReadiness{}) || strings.Contains(err.Error(), "private") {
						t.Fatal("failure accepted or leaked", err)
					}
					if strings.HasSuffix(mode, "before") {
						if reads != 0 {
							t.Fatal("changed process reached HTTP")
						}
					} else if reads == 0 {
						t.Fatal("fault did not reach HTTP")
					}
					if mode == "wrong-last-ship" && reads != len(s.Members) {
						t.Fatal("last identity not exercised")
					}
				}
			}
			if !reflect.DeepEqual(s, original) || x.f.signals != signals || len(f.f.removes) != removes || f.creates != creates || f.starts != starts {
				t.Fatal("readiness changed history or runtime")
			}
			afterBytes, err := os.ReadFile(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			afterStat, err := os.Stat(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			if string(afterBytes) != string(beforeBytes) || !os.SameFile(beforeStat, afterStat) {
				t.Fatal("lost initialized state")
			}
		})
	}
}
func TestProjectAnchorRecoveryReadinessRejectsIncompleteReplacement(t *testing.T) {
	f := newAnchorReplacementAdapterFixture(t)
	if _, err := openProjectAnchorRecoveryReadiness(context.Background(), ProjectAnchorRecoveryReadinessSelection{Source: f.selection.Source, Replacement: f.saved}, f.f.x.f.f.d); err == nil {
		t.Fatal("pending replacement accepted")
	}
}
