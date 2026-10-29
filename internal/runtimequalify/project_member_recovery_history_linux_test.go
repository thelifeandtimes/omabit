//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestProjectRecoveryHistoryReader(t *testing.T) {
	for _, mode := range []string{"pass", "caller-alias", "wrong-label", "restarted-member", "retired-reappeared", "invalid-history"} {
		t.Run(mode, func(t *testing.T) {
			f, h, _ := ordinaryRetirementAfterRecovery(t)
			h = h.Clone()
			s := ProjectAnchorRecoveryReviewSelection{Project: f.selection.Source.Project, Epochs: h.Epochs, Launches: h.Launches, Resumes: h.Resumes, Originals: h.Originals, Current: h.Current}
			raw, err := openProjectAnchorRecoveryReview(context.Background(), s, f.f.x.f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectAnchorRecoveryReviewEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.f.x.f.ns.BootID, nil }
			c, err := projectnetwork.SelectAnchorRecoveryHistoryContext(h)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = e.InspectAnchorRecovery(context.Background(), c); err != nil {
				t.Fatal("healthy successor preflight", err)
			}
			if _, err = projectnetwork.CollectAnchorRecoveryHistoryReview(context.Background(), h, e, time.Now); err == nil {
				t.Fatal("healthy successor became recovery")
			}
			signals, removes, creates, starts := f.f.x.f.signals, len(f.f.removes), f.creates, f.starts
			before := h.Clone()
			history := h.Clone()
			// Observe a second anchor namespace, retaining the already recovered member
			// namespaces. No fake driver mutation command is allowed after this setup.
			f.f.x.fault = "anchor-namespace"
			original := f.f.x.f.f.d.execute
			anchorID := c.Binding.Anchor.ID
			started := time.Now().UTC().Format(time.RFC3339Nano)
			f.f.x.f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				ns := len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"})
				if len(a) < 2 || !(ns || a[0] == "image" && a[1] == "inspect" || a[0] == "network" && a[1] == "inspect" || a[0] == "container" && (a[1] == "inspect" || a[1] == "ls")) {
					t.Fatal("reader attempted mutation", a)
				}
				out, err := original(ctx, args, env)
				if err != nil {
					return out, err
				}
				if a[0] == "container" && a[1] == "inspect" && a[2] != "--format" {
					var values []container
					if err = json.Unmarshal([]byte(out), &values); err != nil {
						t.Fatal(err)
					}
					for i := range values {
						if values[i].ID == anchorID {
							values[i].State.FinishedAt = values[i].State.StartedAt
							values[i].State.StartedAt = started
						}
					}
					b, _ := json.Marshal(values)
					return string(b), nil
				}
				return out, nil
			}
			raw, err = openProjectAnchorRecoveryReview(context.Background(), s, f.f.x.f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e = raw.(*projectAnchorRecoveryReviewEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.f.x.f.ns.BootID, nil }
			switch mode {
			case "caller-alias":
				s.Epochs[0].Readiness.Replacement.Members[0].Creation.Storage.Path = "changed"
			case "wrong-label":
				delete(f.f.x.f.containers[c.Members[0].Context.Member.Container.ID].Config.Labels, projectLabel+".namespace-operation")
			case "restarted-member":
				f.f.x.f.containers[c.Members[0].Context.Member.Container.ID].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
			case "retired-reappeared":
				v := projectMemberContainerFixture(t, f.contracts[0])
				v.ID = h.Launches[0].Container.ID
				v.HostConfig.NetworkMode = "none"
				v.Name = "/foreign"
				f.f.x.f.containers[v.ID] = &v
			case "invalid-history":
				s.Epochs[0].Readiness.Members[0].Ship = "~bus"
				if _, err = openProjectAnchorRecoveryReview(context.Background(), s, f.f.x.f.f.d); err == nil {
					t.Fatal("invalid boundary opened")
				}
				return
			}
			review, err := projectnetwork.CollectAnchorRecoveryHistoryReview(context.Background(), history, e, time.Now)
			if mode == "pass" || mode == "caller-alias" {
				if err != nil || review.Assessment.State != "namespace-changed" || review.Context.Binding.Namespace != history.Epochs[0].Readiness.Binding.Namespace {
					t.Fatal("second review failed", err)
				}
			} else if err == nil {
				t.Fatal("changed history accepted")
			}
			if !reflect.DeepEqual(history, before) || f.f.x.f.signals != signals || len(f.f.removes) != removes || f.creates != creates || f.starts != starts {
				t.Fatal("reader changed history or dispatched effects")
			}
		})
	}
}

func materializeOrdinarySuccessorFixture(t *testing.T) (*anchorReplacementAdapterFixture, projectnetwork.AnchorRecoveryHistory, projectnetwork.MemberResume) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	f, h, retired := ordinaryRetirementAfterRecovery(t)
	instance := retired.Exit.Stop.InstanceID
	operation := "ordinary-after-recovery"
	c, err := newRecoveredResumeProjectMemberContract(f.selection.Source.Project, h, retired, operation, f.f.x.f.f.d.home, filepath.Dir(f.f.x.f.paths[instance]), f.contracts[0].imageID)
	must(err)
	s, err := projectnetwork.NewMemberResume(h.Plan, retired, c.postRecovery.PreviousGeneration, operation, c.digest)
	must(err)
	clone := func(o projectnetwork.Observation) projectnetwork.Observation {
		b, _ := json.Marshal(o)
		var v projectnetwork.Observation
		must(json.Unmarshal(b, &v))
		return v
	}
	absence := *retired.After
	absence.Runtime = clone(absence.Runtime)
	absence.CapturedAt = time.Now().UTC()
	absence.Runtime.ObservedAt = absence.CapturedAt
	s, err = s.Advance(h.Plan, "planned", "create-dispatched", projectnetwork.MemberResumeResult{Before: &absence})
	must(err)
	id := fmt.Sprintf("%064x", 500)
	s, err = s.Advance(h.Plan, "create-dispatched", "created", projectnetwork.MemberResumeResult{ContainerID: id})
	must(err)
	created := projectnetwork.MemberResumeObservation{Runtime: clone(absence.Runtime), Storage: c.selection.Storage, CapturedAt: time.Now().UTC()}
	created.Runtime.ObservedAt = created.CapturedAt
	created.Runtime.Members = append(created.Runtime.Members, projectnetwork.MemberObservation{InstanceID: instance, ResourceObservation: projectnetwork.ResourceObservation{Resource: s.Container, Present: true}, AnchorID: retired.Exit.Stop.Binding.Anchor.ID})
	slices.SortFunc(created.Runtime.Members, func(a, b projectnetwork.MemberObservation) int { return strings.Compare(a.InstanceID, b.InstanceID) })
	s, err = s.Advance(h.Plan, "created", "start-dispatched", projectnetwork.MemberResumeResult{Observation: &created})
	must(err)
	s, err = s.Advance(h.Plan, "start-dispatched", "started", projectnetwork.MemberResumeResult{})
	must(err)
	running := created
	running.Runtime = clone(created.Runtime)
	running.CapturedAt = time.Now().UTC()
	running.Runtime.ObservedAt = running.CapturedAt
	running.StartedAt = running.CapturedAt
	for i := range running.Runtime.Members {
		if running.Runtime.Members[i].InstanceID == instance {
			running.Runtime.Members[i].Running = true
			running.Runtime.Members[i].Namespace = retired.Exit.Stop.Binding.Namespace
		}
	}
	s, err = s.Advance(h.Plan, "started", "running", projectnetwork.MemberResumeResult{Observation: &running})
	must(err)
	// This fixture updates only its in-memory engine catalogue; it runs no commands.
	oldID := ""
	for _, m := range retired.Exit.Stop.Binding.Members {
		if m.InstanceID == instance {
			oldID = m.Container.ID
		}
	}
	delete(f.f.x.f.containers, oldID)
	v := projectMemberContainerFixture(t, c)
	v.ID = id
	v.State.Status = "running"
	*v.State.Running = true
	v.State.Pid = 250
	v.State.StartedAt = running.StartedAt.Format(time.RFC3339Nano)
	f.f.x.f.containers[id] = &v
	h.Resumes = []projectnetwork.MemberResume{s}
	if _, err = projectnetwork.ResolveAnchorRecoveryHistory(h); err != nil {
		t.Fatal("invalid ordinary successor history", err)
	}
	return f, h, s
}
func TestProjectRecoveryHistoryReaderAfterOrdinaryResume(t *testing.T) {
	for _, mode := range []string{"pass", "changed-recovery-label", "changed-retirement-label", "process-restarted"} {
		t.Run(mode, func(t *testing.T) {
			f, h, resume := materializeOrdinarySuccessorFixture(t)
			selection := ProjectAnchorRecoveryReviewSelection{Project: f.selection.Source.Project, Epochs: h.Epochs, Launches: h.Launches, Resumes: h.Resumes, Originals: h.Originals, Current: h.Current}
			raw, err := openProjectAnchorRecoveryReview(context.Background(), selection, f.f.x.f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectAnchorRecoveryReviewEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.f.x.f.ns.BootID, nil }
			c, err := projectnetwork.SelectAnchorRecoveryHistoryContext(h)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = e.InspectAnchorRecovery(context.Background(), c); err != nil {
				t.Fatal("ordinary successor preflight", err)
			}
			ordinary, coordinated := 0, 0
			for _, m := range e.source.members {
				if m.contract.postRecovery != nil {
					ordinary++
				}
				if m.contract.recovery != nil {
					coordinated++
				}
			}
			if ordinary != 1 || coordinated != 1 {
				t.Fatal("mixed creator contracts not reconstructed")
			}
			signals, removes, creates, starts := f.f.x.f.signals, len(f.f.removes), f.creates, f.starts
			// A second anchor namespace leaves both different creator types in the old one.
			f.f.x.fault = "anchor-namespace"
			raw, err = openProjectAnchorRecoveryReview(context.Background(), selection, f.f.x.f.f.d)
			if err != nil {
				t.Fatal(err)
			}
			e = raw.(*projectAnchorRecoveryReviewEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.f.x.f.ns.BootID, nil }
			v := f.f.x.f.containers[resume.Container.ID]
			switch mode {
			case "changed-recovery-label":
				v.Config.Labels[projectLabel+".recovery-operation"] = "foreign"
			case "changed-retirement-label":
				v.Config.Labels[projectLabel+".retirement-operation"] = "foreign"
			case "process-restarted":
				v.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
			review, err := projectnetwork.CollectAnchorRecoveryHistoryReview(context.Background(), h, e, time.Now)
			if mode == "pass" {
				if err != nil || review.Assessment.State != "namespace-changed" {
					t.Fatal("mixed-creator review failed", err)
				}
			} else if err == nil {
				t.Fatal("changed ordinary successor accepted")
			}
			if f.f.x.f.signals != signals || len(f.f.removes) != removes || f.creates != creates || f.starts != starts {
				t.Fatal("reader dispatched effects")
			}
		})
	}
}
