//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// The first recovery uses the existing concrete fixtures. This independent
// in-memory engine then checks every command in the second recovery against its
// committed phase; no Docker daemon or real ship is used.
func TestProjectSecondRecoveryAdapters(t *testing.T) {
	for _, mode := range []string{"restarted", "stopped", "mixed-creators", "retired-before-signal", "retired-before-namespace", "retired-before-remove", "retired-before-create", "retired-before-start", "retired-during-readiness"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			var f *anchorReplacementAdapterFixture
			var h projectnetwork.AnchorRecoveryHistory
			if mode == "mixed-creators" {
				f, h, _ = materializeOrdinarySuccessorFixture(t)
			} else {
				f, h, _ = ordinaryRetirementAfterRecovery(t)
			}
			h = h.Clone()
			beforeHistory := h.Clone()
			x := f.f.x
			d := x.f.f.d
			selection := ProjectAnchorRecoveryReviewSelection{Project: f.selection.Source.Project, Epochs: h.Epochs, Launches: h.Launches, Resumes: h.Resumes, Originals: h.Originals, Current: h.Current}
			current, err := projectnetwork.ResolveAnchorRecoveryHistory(h)
			must(err)
			anchorID := current.Binding.Anchor.ID
			oldNS := current.Binding.Namespace
			newNS := oldNS
			newNS.Inode++
			anchorRunning := mode != "stopped"
			anchorStarted := time.Now().UTC()
			anchorFinished := anchorStarted.Add(-time.Millisecond)
			if !anchorRunning {
				anchorFinished = anchorStarted
				anchorStarted = anchorStarted.Add(-time.Second)
			}
			namespaces := map[string]projectnetwork.NamespaceIdentity{}
			for _, m := range current.Members {
				namespaces[m.Container.ID] = oldNS
			}
			var drain projectnetwork.AnchorDrain
			var ns projectnetwork.AnchorNamespace
			var retired projectnetwork.AnchorRetirement
			var replacement projectnetwork.AnchorReplacement
			var contracts []projectMemberContract
			signals, removes, creates, starts, anchorStarts := 0, 0, 0, 0, 0
			oldExecute := d.execute
			d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				switch {
				case len(a) == 5 && reflect.DeepEqual(a[:4], []string{"container", "kill", "--signal", "SIGTERM"}):
					found := false
					for i, m := range drain.Members {
						if m.Phase == "stop-dispatched" && drain.Review.Context.Members[i].Context.Member.Container.ID == a[4] {
							found = true
						}
					}
					if !found || x.f.containers[a[4]] == nil || !*x.f.containers[a[4]].State.Running {
						t.Fatal("uncommitted second drain", a)
					}
					signals++
					x.f.exit(a[4], 0, false)
					return a[4], nil
				case len(a) > 1 && a[0] == "container" && a[1] == "rm":
					if len(a) != 3 {
						t.Fatal("unexpected removal flags", a)
					}
					found := false
					for i, m := range retired.Members {
						if m.Phase == "remove-dispatched" && retired.Namespace.Drain.Review.Context.Members[i].Context.Member.Container.ID == a[2] {
							found = true
						}
					}
					if !found || x.f.containers[a[2]] == nil || *x.f.containers[a[2]].State.Running {
						t.Fatal("uncommitted second retirement", a)
					}
					removes++
					delete(x.f.containers, a[2])
					return a[2], nil
				case len(a) > 1 && a[0] == "container" && a[1] == "create":
					index := creates
					if index >= len(contracts) || replacement.Members[index].Phase != "create-dispatched" || !reflect.DeepEqual(a, contracts[index].createArgs()) {
						t.Fatal("uncommitted second Create", a)
					}
					id := fmt.Sprintf("%064x", 700+index)
					v := projectMemberContainerFixture(t, contracts[index])
					v.ID = id
					x.f.containers[id] = &v
					namespaces[id] = newNS
					creates++
					return id, nil
				case len(a) > 1 && a[0] == "container" && a[1] == "start":
					if len(a) != 3 {
						t.Fatal("unexpected Start flags", a)
					}
					if a[2] == anchorID {
						if anchorRunning || ns.Phase != "start-dispatched" || anchorStarts != 0 {
							t.Fatal("uncommitted second anchor Start")
						}
						anchorRunning = true
						anchorStarted = time.Now().UTC()
						anchorStarts++
						return anchorID, nil
					}
					index := starts
					if index >= len(contracts) || replacement.Members[index].Phase != "start-dispatched" || replacement.Members[index].Container.ID != a[2] {
						t.Fatal("uncommitted second member Start")
					}
					v := x.f.containers[a[2]]
					*v.State.Running = true
					v.State.Status = "running"
					v.State.Pid = 800 + index
					v.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					starts++
					return a[2], nil
				case len(a) == 8 && a[0] == "container" && a[1] == "exec":
					if !reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
						t.Fatal("unexpected exec", a)
					}
					n := namespaces[a[2]]
					if a[2] == anchorID {
						if !anchorRunning {
							t.Fatal("namespace read of stopped anchor")
						}
						n = newNS
					} else if x.f.containers[a[2]] == nil || !*x.f.containers[a[2]].State.Running {
						t.Fatal("namespace read of missing/stopped member")
					}
					return fmt.Sprintf("%d:%d", n.Device, n.Inode), nil
				}
				out, err := oldExecute(ctx, args, env)
				if err != nil {
					return out, err
				}
				if len(a) > 2 && a[0] == "container" && a[1] == "inspect" && a[2] != "--format" {
					var values []container
					must(json.Unmarshal([]byte(out), &values))
					for i := range values {
						v := &values[i]
						if v.ID != anchorID {
							continue
						}
						*v.State.Running = anchorRunning
						v.State.StartedAt = anchorStarted.Format(time.RFC3339Nano)
						v.State.FinishedAt = anchorFinished.Format(time.RFC3339Nano)
						if anchorRunning {
							v.State.Status = "running"
							v.State.Pid = 900
							*v.State.ExitCode = 0
							v.NetworkSettings.Ports = v.HostConfig.PortBindings
						} else {
							v.State.Status = "exited"
							v.State.Pid = 0
							*v.State.ExitCode = 143
							v.NetworkSettings.Ports = map[string]json.RawMessage{}
						}
					}
					b, _ := json.Marshal(values)
					return string(b), nil
				}
				if len(a) > 1 && a[0] == "network" && a[1] == "inspect" && !anchorRunning {
					var values []bridge
					must(json.Unmarshal([]byte(out), &values))
					values[0].Containers = map[string]json.RawMessage{}
					b, _ := json.Marshal(values)
					return string(b), nil
				}
				return out, nil
			}
			boot := func() (string, error) { return oldNS.BootID, nil }
			inject := func() {
				v := projectMemberContainerFixture(t, f.contracts[0])
				v.ID = h.Launches[0].Container.ID
				v.HostConfig.NetworkMode = "none"
				v.Mounts = nil
				v.Name = "/historical-foreign"
				x.f.containers[v.ID] = &v
			}
			bytesBefore, err := os.ReadFile(x.sentinel)
			must(err)
			statBefore, err := os.Stat(x.sentinel)
			must(err)
			defer func() {
				b, err := os.ReadFile(x.sentinel)
				must(err)
				st, err := os.Stat(x.sentinel)
				must(err)
				if !reflect.DeepEqual(b, bytesBefore) || !os.SameFile(st, statBefore) || !reflect.DeepEqual(h, beforeHistory) {
					t.Fatal("second recovery rewrote pier/history")
				}
			}()
			raw, err := openProjectAnchorRecoveryReview(ctx, selection, d)
			must(err)
			observer := raw.(*projectAnchorRecoveryReviewEngine)
			observer.source.anchor.readBootID = boot
			review, err := projectnetwork.CollectAnchorRecoveryHistoryReview(ctx, h, observer, time.Now)
			must(err)
			drain, err = projectnetwork.NewAnchorDrain(h, review, "second-drain")
			must(err)
			for !drain.Clean() {
				raw, err := openProjectAnchorDrain(ctx, ProjectAnchorDrainSelection{Source: selection, Drain: drain}, d)
				must(err)
				e := raw.(*projectAnchorDrainEngine)
				e.review.source.anchor.readBootID = boot
				got, err := projectnetwork.ExecuteNextAnchorDrain(ctx, h, drain, e, func(seq uint64, next projectnetwork.AnchorDrain) error {
					if seq != drain.Sequence {
						t.Fatal("stale drain commit")
					}
					drain = next.Clone()
					if mode == "retired-before-signal" && signals == 0 && next.Members[0].Phase == "stop-dispatched" {
						inject()
					}
					return nil
				}, time.Now)
				if mode == "retired-before-signal" {
					if err == nil || signals != 0 || !reflect.DeepEqual(got, drain) {
						t.Fatal("historical ID did not fence signal", err)
					}
					return
				}
				must(err)
				if !reflect.DeepEqual(got, drain) {
					t.Fatal("drain receipt lost")
				}
			}
			ns, err = projectnetwork.NewAnchorNamespace(h, drain, "second-namespace")
			must(err)
			rawNS, err := openProjectAnchorNamespace(ctx, ProjectAnchorNamespaceSelection{Source: selection, Namespace: ns}, d)
			must(err)
			eNS := rawNS.(*projectAnchorNamespaceEngine)
			eNS.observation.review.source.anchor.readBootID = boot
			_, err = projectnetwork.ExecuteAnchorNamespace(ctx, h, ns, eNS, func(seq uint64, next projectnetwork.AnchorNamespace) error {
				if seq != ns.Sequence {
					t.Fatal("stale namespace commit")
				}
				ns = next.Clone()
				if mode == "retired-before-namespace" {
					inject()
				}
				return nil
			}, time.Now)
			if mode == "retired-before-namespace" {
				if err == nil || anchorStarts != 0 || ns.Phase == "observed" {
					t.Fatal("historical ID did not fence namespace", err)
				}
				return
			}
			must(err)
			retired, err = projectnetwork.NewAnchorRetirement(h, ns, "second-retire")
			must(err)
			for !retired.Complete() {
				raw, err := openProjectAnchorRetirement(ctx, ProjectAnchorRetirementSelection{Source: selection, Retirement: retired}, d)
				must(err)
				e := raw.(*projectAnchorRetirementEngine)
				e.review.source.anchor.readBootID = boot
				_, err = projectnetwork.ExecuteNextAnchorRetirement(ctx, h, retired, e, func(seq uint64, next projectnetwork.AnchorRetirement) error {
					if seq != retired.Sequence {
						t.Fatal("stale retirement commit")
					}
					retired = next.Clone()
					if mode == "retired-before-remove" && removes == 0 && next.Members[0].Phase == "remove-dispatched" {
						inject()
					}
					return nil
				}, time.Now)
				if mode == "retired-before-remove" {
					if err == nil || removes != 0 {
						t.Fatal("historical ID did not fence removal", err)
					}
					return
				}
				must(err)
			}
			var members []projectnetwork.RecoveredMember
			for _, m := range retired.Members {
				c, err := newRecoveryProjectMemberContract(selection.Project, h, retired, m.InstanceID, "second-create-"+m.InstanceID, d.home, filepath.Dir(x.f.paths[m.InstanceID]), f.contracts[0].imageID)
				must(err)
				contracts = append(contracts, c)
				members = append(members, projectnetwork.RecoveredMember{Creation: c.recovery.Clone(), Container: projectnetwork.ResourceIdentity{ContractDigest: c.digest}, Phase: "planned"})
			}
			replacement, err = projectnetwork.NewAnchorReplacement(h, retired, "second-replace", members)
			must(err)
			for !replacement.Running() {
				raw, err := openProjectAnchorReplacement(ctx, ProjectAnchorReplacementSelection{Source: selection, Replacement: replacement}, d)
				must(err)
				e := raw.(*projectAnchorReplacementEngine)
				e.review.source.anchor.readBootID = boot
				_, err = projectnetwork.ExecuteNextAnchorReplacement(ctx, h, replacement, e, func(seq uint64, next projectnetwork.AnchorReplacement) error {
					if seq != replacement.Sequence {
						t.Fatal("stale replacement commit")
					}
					replacement = next.Clone()
					if mode == "retired-before-create" && next.Members[0].Phase == "create-dispatched" || mode == "retired-before-start" && next.Members[0].Phase == "start-dispatched" {
						inject()
					}
					return nil
				}, time.Now)
				if strings.HasPrefix(mode, "retired-before-c") || mode == "retired-before-start" {
					want := 0
					if mode == "retired-before-start" {
						want = 1
					}
					if err == nil || creates != want || starts != 0 {
						t.Fatal("historical ID did not fence replacement", err, creates, starts)
					}
					return
				}
				must(err)
			}
			rawReady, err := openProjectAnchorRecoveryReadiness(ctx, ProjectAnchorRecoveryReadinessSelection{Source: selection, Replacement: replacement}, d)
			must(err)
			eReady := rawReady.(*projectAnchorRecoveryReadinessEngine)
			eReady.review.source.anchor.readBootID = boot
			reads := 0
			eReady.review.source.anchor.base.q.d.hostIdentity = func(_ context.Context, endpoint string) (string, error) {
				reads++
				if mode == "retired-during-readiness" {
					inject()
				}
				ship := eReady.endpoints[endpoint]
				if ship == "" {
					t.Fatal("unselected identity endpoint")
				}
				return ship, nil
			}
			ready, err := projectnetwork.CollectAnchorRecoveryReadiness(ctx, h, replacement, eReady, time.Now)
			if mode == "retired-during-readiness" {
				if err == nil || reads == 0 {
					t.Fatal("historical ID did not fence readiness", err)
				}
				return
			}
			must(err)
			complete, err := projectnetwork.AppendAnchorRecoveryEpoch(h, ready)
			must(err)
			final, err := projectnetwork.ResolveAnchorRecoveryHistory(complete)
			must(err)
			wantStarts := 0
			if mode == "stopped" {
				wantStarts = 1
			}
			if len(complete.Epochs) != 2 || final.Binding.Namespace != newNS || signals != 2 || removes != 2 || creates != 2 || starts != 2 || anchorStarts != wantStarts || reads != 2 {
				t.Fatal("second recovery incomplete", signals, removes, creates, starts, anchorStarts, reads)
			}
			for i, m := range final.Members {
				if m.Generation != current.Members[i].Generation+1 || m.Storage != current.Members[i].Storage || m.OperationID == current.Members[i].OperationID {
					t.Fatal("second generation lost original pier/creator")
				}
			}
			// A new concrete reader must reconstruct the second boundary and reject a
			// healthy recovery attempt without replaying any lifecycle command.
			selection.Epochs = complete.Epochs
			selection.Resumes = complete.Resumes
			selection.Current = complete.Current
			raw, err = openProjectAnchorRecoveryReview(ctx, selection, d)
			must(err)
			observer = raw.(*projectAnchorRecoveryReviewEngine)
			observer.source.anchor.readBootID = boot
			selected, err := projectnetwork.SelectAnchorRecoveryHistoryContext(complete)
			must(err)
			_, err = observer.InspectAnchorRecovery(ctx, selected)
			must(err)
			if _, err = projectnetwork.CollectAnchorRecoveryHistoryReview(ctx, complete, observer, time.Now); err == nil {
				t.Fatal("healthy twice-recovered project admitted repair")
			}
		})
	}
}
