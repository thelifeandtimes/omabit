//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func replacementStopProof(t *testing.T, p projectnetwork.Plan, s projectnetwork.MemberResume) projectnetwork.ResumedMemberReadiness {
	t.Helper()
	b, err := projectnetwork.ResumedRuntimeBinding(p, s)
	if err != nil {
		t.Fatal(err)
	}
	var copy projectnetwork.MemberResume
	data, _ := json.Marshal(s)
	if err = json.Unmarshal(data, &copy); err != nil {
		t.Fatal(err)
	}
	o := *copy.After
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	var ship, endpoint string
	for _, m := range p.Members {
		if m.InstanceID == s.Retirement.Exit.Stop.InstanceID {
			ship = m.Ship
			endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
		}
	}
	r := projectnetwork.ResumedMemberReadiness{InstanceID: s.Retirement.Exit.Stop.InstanceID, ResumeOperationID: s.OperationID, ContainerGeneration: s.PreviousGeneration + 1, Binding: b, Storage: s.After.Storage, Ship: ship, Endpoint: endpoint, Before: o, After: o, CapturedAt: o.CapturedAt}
	if err = r.Validate(p, s); err != nil {
		t.Fatal(err)
	}
	return r
}
func resumedStopAdapterFixture(t *testing.T, stoppedPeer bool, generation int) (*memberStopFixture, ProjectResumedMemberStopSelection, string) {
	t.Helper()
	f, input, sentinel := resumedReadinessAdapterFixture(t, stoppedPeer)
	p := input.Resume.Stop.Project.Plan
	s := ProjectResumedMemberStopSelection{Project: input.Resume.Stop.Project, Launches: input.Resume.Stop.Launches, Resumes: []projectnetwork.MemberResume{input.Completed}, Readiness: replacementStopProof(t, p, input.Completed), StoppedPeers: input.Resume.Stop.StoppedPeers}
	var err error
	s.Stop, err = projectnetwork.NewResumedMemberStop(p, s.Launches, s.Resumes, s.Readiness, "ready-generation2", "stop-generation2")
	if err != nil {
		t.Fatal(err)
	}
	if generation == 2 {
		return f, s, sentinel
	}
	// Construct a second complete history using an actual inert Stop driver and
	// pure retirement/replacement transitions. No real Docker process is invoked.
	raw, err := openResumedProjectMemberStop(context.Background(), s, f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	e := raw.(*projectMemberStopEngine)
	e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
	stop, err := projectnetwork.ExecuteMemberStop(context.Background(), p, s.Stop, e, func(string, projectnetwork.MemberStop) error { return nil }, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	r, err := projectnetwork.NewMemberRetirement(p, projectnetwork.MemberStopExitEvidence{OperationID: "exit-generation2", Stop: stop, Observation: *stop.After}, "retire-generation2")
	must(err)
	r, err = r.Advance(p, "planned", "remove-dispatched", stop.After, nil)
	must(err)
	r, err = r.Advance(p, "remove-dispatched", "removed", nil, nil)
	must(err)
	absent := projectnetwork.MemberAbsenceObservation{Runtime: stop.After.Runtime, Storage: stop.Storage, CapturedAt: time.Now().UTC()}
	absent.Runtime.Members = slices.Clone(absent.Runtime.Members)
	absent.Runtime.Members = slices.DeleteFunc(absent.Runtime.Members, func(m projectnetwork.MemberObservation) bool { return m.InstanceID == stop.InstanceID })
	absent.Runtime.ObservedAt = absent.CapturedAt
	r, err = r.Advance(p, "removed", "absent", nil, &absent)
	must(err)
	delete(f.containers, e.target.id)
	selected := e.target.contract.selection
	selected.ContainerGeneration = 3
	selected.OperationID = "resume-generation3"
	c, err := newProjectMemberContract(selected, f.f.d.home, e.target.contract.mountSource, e.target.contract.imageID)
	must(err)
	next, err := projectnetwork.NewMemberResume(p, r, 2, selected.OperationID, c.digest)
	must(err)
	next, err = next.Advance(p, "planned", "create-dispatched", projectnetwork.MemberResumeResult{Before: &absent})
	must(err)
	id := strings.Repeat("9", 64)
	next, err = next.Advance(p, "create-dispatched", "created", projectnetwork.MemberResumeResult{ContainerID: id})
	must(err)
	v := projectMemberContainerFixture(t, c)
	v.ID = id
	f.containers[id] = &v
	known := slices.Clone(e.source.members)
	for i, m := range known {
		if m.id == e.target.id {
			known[i] = projectObservedMember{contract: c, id: id}
		} else if exit, ok := e.exits[m.contract.selection.InstanceID]; ok {
			known[i].processStartedAt = exit.started
			known[i].allowExited = true
		}
	}
	snap, err := e.source.anchor.inspectProjectMemberSnapshot(context.Background(), known)
	must(err)
	o := projectnetwork.MemberResumeObservation{Runtime: snap.Observation, Storage: stop.Storage, CapturedAt: time.Now().UTC()}
	next, err = next.Advance(p, "created", "start-dispatched", projectnetwork.MemberResumeResult{Observation: &o})
	must(err)
	next, err = next.Advance(p, "start-dispatched", "started", projectnetwork.MemberResumeResult{})
	must(err)
	*v.State.Running = true
	v.State.Status = "running"
	v.State.Pid = 100
	started := time.Now().UTC()
	v.State.StartedAt = started.Format(time.RFC3339Nano)
	snap, err = e.source.anchor.inspectProjectMemberSnapshot(context.Background(), known)
	must(err)
	o = projectnetwork.MemberResumeObservation{Runtime: snap.Observation, Storage: stop.Storage, StartedAt: started, CapturedAt: time.Now().UTC()}
	next, err = next.Advance(p, "started", "running", projectnetwork.MemberResumeResult{Observation: &o})
	must(err)
	s.Resumes = []projectnetwork.MemberResume{next, input.Completed} // reverse input order is intentional
	s.Readiness = replacementStopProof(t, p, next)
	s.Stop, err = projectnetwork.NewResumedMemberStop(p, s.Launches, s.Resumes, s.Readiness, "ready-generation3", "stop-generation3")
	must(err)
	f.signals = 0
	return f, s, sentinel
}
func TestResumedProjectMemberStopAdapter(t *testing.T) {
	for _, generation := range []int{2, 3} {
		for _, mode := range []string{"pass", "stopped-peer", "live-lock", "lost-ack", "uncommitted", "target-restarted", "stopped-peer-changed", "peer-unrecorded", "unknown-container", "pier-replaced", "namespace-changed", "caller-alias"} {
			t.Run(fmt.Sprintf("generation%d/%s", generation, mode), func(t *testing.T) {
				f, input, sentinel := resumedStopAdapterFixture(t, strings.HasPrefix(mode, "stopped-peer"), generation)
				originalStat, err := os.Stat(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				originalBytes, err := os.ReadFile(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := openResumedProjectMemberStop(context.Background(), input, f.f.d)
				if err != nil {
					t.Fatal(err)
				}
				e := raw.(*projectMemberStopEngine)
				e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				if e.target.contract.selection.ContainerGeneration != uint64(generation) || e.target.contract.pillPath != "" {
					t.Fatal("wrong current contract")
				}
				oldSignals := f.signals
				saved := input.Stop
				signal := f.signal
				f.signal = func(ctx context.Context, id string) (string, error) {
					if saved.Phase != "stop-dispatched" || id != e.target.id {
						t.Fatal("uncommitted or obsolete target signalled")
					}
					out, err := signal(ctx, id)
					if mode == "lost-ack" {
						return "private response", errors.New("private output")
					}
					return out, err
				}
				if mode == "live-lock" {
					if err = os.WriteFile(filepath.Join(f.paths[input.Stop.InstanceID], ".vere.lock"), []byte("99"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "peer-unrecorded" {
					for id := range f.containers {
						if id != e.target.id {
							f.exit(id, 0, false)
						}
					}
				}
				if mode == "stopped-peer-changed" {
					for id, v := range f.containers {
						if id != e.target.id {
							*v.State.ExitCode = 1
						}
					}
				}
				if mode == "unknown-container" {
					v := *f.containers[e.target.id]
					v.ID = strings.Repeat("a", 64)
					v.Name = "/unowned"
					f.containers[v.ID] = &v
				}
				initial := input.Stop
				if mode == "caller-alias" {
					input.Stop.Binding.Members[0].InstanceID = "changed"
					input.Launches[0].Storage.RuntimeID = "changed"
					initial = e.initial
				}
				commit := func(phase string, next projectnetwork.MemberStop) error {
					if phase != saved.Phase {
						t.Fatal("stale persistence")
					}
					if mode == "uncommitted" {
						return errors.New("uncommitted")
					}
					saved = next
					if next.Phase == "stop-dispatched" {
						switch mode {
						case "target-restarted":
							f.containers[e.target.id].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
						case "namespace-changed":
							e.source.anchor.readBootID = func() (string, error) { return "22345678-1234-1234-1234-123456789abc", nil }
						case "pier-replaced":
							path := f.paths[initial.InstanceID]
							if err = os.Rename(filepath.Join(path, ".urb"), filepath.Join(path, ".urb-retained")); err != nil {
								t.Fatal(err)
							}
							if err = os.Mkdir(filepath.Join(path, ".urb"), 0700); err != nil {
								t.Fatal(err)
							}
						}
					}
					return nil
				}
				result, err := projectnetwork.ExecuteMemberStop(context.Background(), e.source.anchor.base.contract.Selection.Plan, initial, e, commit, time.Now)
				pass := slices.Contains([]string{"pass", "stopped-peer", "live-lock", "caller-alias"}, mode)
				if (err == nil) != pass || strings.Contains(fmt.Sprint(err), "private") {
					t.Fatal("unexpected result", result.Phase, err)
				}
				wantSignals := 0
				if pass || mode == "lost-ack" {
					wantSignals = 1
				}
				if f.signals-oldSignals != wantSignals || !reflect.DeepEqual(result, saved) {
					t.Fatal("wrong effects or durable state", f.signals-oldSignals, wantSignals)
				}
				if pass && !result.Clean() {
					t.Fatal("clean exit not retained")
				}
				if mode == "lost-ack" && result.Phase != "stop-dispatched" {
					t.Fatal("invented acknowledgement")
				}
				if e.SignalStop(context.Background(), result) == nil || f.signals-oldSignals != wantSignals {
					t.Fatal("signal replayed")
				}
				if mode != "pier-replaced" {
					after, err := os.Stat(sentinel)
					if err != nil || !os.SameFile(after, originalStat) {
						t.Fatal("state object changed", err)
					}
					data, err := os.ReadFile(sentinel)
					if err != nil || string(data) != string(originalBytes) {
						t.Fatal("state bytes changed", err)
					}
				}
			})
		}
	}
}
func TestResumedProjectMemberStopRefusesUnboundHistory(t *testing.T) {
	for _, mode := range []string{"missing-predecessor", "old-readiness", "wrong-generation", "changed-contract", "duplicate-peer", "wrong-peer-creator", "unfinished-peer", "foreign-peer-binding", "changed-intent"} {
		t.Run(mode, func(t *testing.T) {
			f, s, _ := resumedStopAdapterFixture(t, true, 3)
			switch mode {
			case "missing-predecessor":
				s.Resumes = s.Resumes[:1]
			case "old-readiness":
				s.Readiness = replacementStopProof(t, s.Project.Plan, s.Resumes[1])
			case "wrong-generation":
				s.Resumes[0].PreviousGeneration++
			case "changed-contract":
				f.containers[s.Resumes[0].Container.ID].Config.Labels[projectLabel+".container-generation"] = "2"
			case "duplicate-peer":
				s.StoppedPeers = append(s.StoppedPeers, s.StoppedPeers[0])
			case "wrong-peer-creator":
				s.StoppedPeers[0].LaunchOperationID = "foreign"
			case "unfinished-peer":
				s.StoppedPeers[0].Phase = "stop-signalled"
				s.StoppedPeers[0].After = nil
			case "foreign-peer-binding":
				s.StoppedPeers[0].Binding.Members[0].Container.ID = strings.Repeat("b", 64)
			case "changed-intent":
				s.Stop.Storage.Identity.Inode++
			}
			raw, err := openResumedProjectMemberStop(context.Background(), s, f.f.d)
			if mode == "changed-contract" && err == nil {
				e := raw.(*projectMemberStopEngine)
				e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
				_, err = e.Inspect(context.Background(), s.Stop)
			}
			if err == nil || f.signals != 0 {
				t.Fatal("unbound selection accepted or signalled", err)
			}
		})
	}
}
