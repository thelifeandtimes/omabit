//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func ordinaryRetirementAfterRecovery(t *testing.T) (*anchorReplacementAdapterFixture, projectnetwork.AnchorRecoveryHistory, projectnetwork.MemberRetirement) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	f := completeReplacementsForReadiness(t)
	h := anchorDrainHistory(f.selection.Source)
	raw, err := openProjectAnchorRecoveryReadiness(context.Background(), ProjectAnchorRecoveryReadinessSelection{Source: f.selection.Source, Replacement: f.saved}, f.f.x.f.f.d)
	must(err)
	e := raw.(*projectAnchorRecoveryReadinessEngine)
	e.review.source.anchor.readBootID = func() (string, error) { return f.f.x.f.ns.BootID, nil }
	e.review.source.anchor.base.q.d.hostIdentity = func(_ context.Context, endpoint string) (string, error) { return e.endpoints[endpoint], nil }
	ready, err := projectnetwork.CollectAnchorRecoveryReadiness(context.Background(), h, f.saved, e, time.Now)
	must(err)
	history, err := projectnetwork.AppendAnchorRecoveryEpoch(h, ready)
	must(err)
	instance := f.saved.Members[0].Creation.InstanceID
	c, err := projectnetwork.SelectRecoveredMemberContext(h, ready, nil, instance)
	must(err)
	now := time.Now().UTC()
	stop := projectnetwork.MemberStop{OperationID: "ordinary-stop", ReadinessOperationID: "ordinary-ready", LaunchOperationID: c.Member.OperationID, InstanceID: instance, Binding: c.Binding, Storage: c.Storage, Phase: "planned"}
	// JSON cloning keeps the independent test observations from aliasing readiness.
	clone := func(o projectnetwork.Observation) projectnetwork.Observation {
		b, _ := json.Marshal(o)
		var v projectnetwork.Observation
		must(json.Unmarshal(b, &v))
		return v
	}
	before := projectnetwork.MemberStopObservation{Runtime: clone(ready.After.Runtime), Storage: c.Storage, StartedAt: c.KnownStartedAt, CapturedAt: now}
	before.Runtime.ObservedAt = now
	stop, err = stop.Advance(h.Plan, "planned", "stop-dispatched", &before)
	must(err)
	exit := before
	exit.Runtime = clone(before.Runtime)
	exit.CapturedAt = time.Now().UTC()
	exit.Runtime.ObservedAt = exit.CapturedAt
	exit.FinishedAt = exit.CapturedAt
	for i := range exit.Runtime.Members {
		if exit.Runtime.Members[i].InstanceID == instance {
			exit.Runtime.Members[i].Running = false
			exit.Runtime.Members[i].Namespace = projectnetwork.NamespaceIdentity{}
		}
	}
	retired, err := projectnetwork.NewMemberRetirement(h.Plan, projectnetwork.MemberStopExitEvidence{OperationID: "ordinary-exit", Stop: stop, Observation: exit}, "ordinary-retire")
	must(err)
	retired, err = retired.Advance(h.Plan, "planned", "remove-dispatched", &exit, nil)
	must(err)
	retired, err = retired.Advance(h.Plan, "remove-dispatched", "removed", nil, nil)
	must(err)
	absence := projectnetwork.MemberAbsenceObservation{Runtime: clone(exit.Runtime), Storage: c.Storage, CapturedAt: time.Now().UTC()}
	absence.Runtime.ObservedAt = absence.CapturedAt
	absence.Runtime.Members = slices.DeleteFunc(absence.Runtime.Members, func(m projectnetwork.MemberObservation) bool { return m.InstanceID == instance })
	retired, err = retired.Advance(h.Plan, "removed", "absent", nil, &absence)
	must(err)
	return f, history, retired
}
func TestRecoveredResumeContractPreservesEarlierFormats(t *testing.T) {
	f, h, r := ordinaryRetirementAfterRecovery(t)
	instance := r.Exit.Stop.InstanceID
	original := h.Clone()
	c, err := newRecoveredResumeProjectMemberContract(f.selection.Source.Project, h, r, "ordinary-resume", f.f.x.f.f.d.home, filepath.Dir(f.f.x.f.paths[instance]), candidate.ManifestDigest)
	if err != nil {
		t.Fatal(err)
	}
	if c.postRecovery == nil || c.recovery != nil || c.postRecovery.Validate(h, r) != nil || c.pillPath != "" || c.selection.Mode != "resume" || c.selection.ContainerGeneration != f.saved.Members[0].Creation.Generation+1 || !reflect.DeepEqual(c.selection.Anchor, h.Launches[0].Anchor) {
		t.Fatal("wrong creation authority")
	}
	labels := c.labels()
	epoch := h.Epochs[0].Readiness.Replacement
	if labels[projectLabel+".namespace-operation"] != epoch.Retirement.Namespace.OperationID || labels[projectLabel+".recovery-operation"] != epoch.OperationID || labels[projectLabel+".retirement-operation"] != r.OperationID {
		t.Fatal("ordinary retirement/recovery conflated")
	}
	legacy, err := newProjectMemberContract(c.selection, c.home, c.mountSource, c.imageID)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(struct {
		Domain     string
		Selection  projectMemberSelection
		Arguments  []string
		PillSHA256 string
		PillBytes  int64
	}{"omabit-project-member-v1", legacy.selection, legacy.createArgs(), candidate.PillSHA256, candidate.PillBytes})
	if err != nil || legacy.digest != fmt.Sprintf("%x", sha256.Sum256(body)) || legacy.digest == c.digest {
		t.Fatal("ordinary v1 contract changed or new contract indistinct")
	}
	for _, key := range []string{".namespace-operation", ".recovery-operation", ".retirement-operation"} {
		if _, exists := legacy.labels()[projectLabel+key]; exists {
			t.Fatal("legacy labels changed")
		}
		v := projectMemberContainerFixture(t, c)
		v.ID = strings.Repeat("e", 64)
		if c.validateContainer(v, v.ID, false) != nil || legacy.validateContainer(v, v.ID, false) == nil {
			t.Fatal("new contract not enforced")
		}
		delete(v.Config.Labels, projectLabel+key)
		if c.validateContainer(v, v.ID, false) == nil {
			t.Fatal("missing history label accepted")
		}
	}
	*c.postRecovery.OriginalAnchor.Namespace = c.postRecovery.Namespace
	if !reflect.DeepEqual(h, original) {
		t.Fatal("contract aliases historical authority")
	}
}
func TestRecoveredResumeContractRejectsForeignProject(t *testing.T) {
	f, h, r := ordinaryRetirementAfterRecovery(t)
	instance := r.Exit.Stop.InstanceID
	for _, mode := range []string{"runner", "plan", "operation", "no-recovery", "mount"} {
		t.Run(mode, func(t *testing.T) {
			selected := f.selection.Source.Project
			selected.Plan.Members = slices.Clone(selected.Plan.Members)
			history := h.Clone()
			op := "ordinary-resume"
			mount := filepath.Dir(f.f.x.f.paths[instance])
			switch mode {
			case "runner":
				selected.RunnerID = "foreign"
			case "plan":
				selected.Plan.Members[0].HTTPPort++
			case "operation":
				op = h.Epochs[0].Readiness.Replacement.OperationID
			case "no-recovery":
				history.Epochs = nil
			case "mount":
				mount = "relative"
			}
			if _, err := newRecoveredResumeProjectMemberContract(selected, history, r, op, f.f.x.f.f.d.home, mount, candidate.ManifestDigest); err == nil {
				t.Fatal("foreign creation accepted")
			}
		})
	}
}
