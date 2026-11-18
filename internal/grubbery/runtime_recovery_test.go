package grubbery_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func recoveryReview(t *testing.T, state string) (grubbery.RuntimeRecoveryReview, grubbery.Record) {
	t.Helper()
	e, c, _, j, b, _ := execution(t)
	record, err := j.Load(t.Context(), c.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if state == "uncertain" {
		b.hide.Store(true)
		record, err = e.Advance(t.Context(), c)
		if err != nil {
			t.Fatal(err)
		}
	}
	step, _, err := record.Current()
	if err != nil {
		t.Fatal(err)
	}
	eventDigest := record.HeaderDigest
	if len(record.Events) > 0 {
		eventDigest = record.Events[len(record.Events)-1].Digest
	}
	at := record.CreatedAt.Add(time.Hour)
	review := grubbery.RuntimeRecoveryReview{Version: 1, OriginalRequestID: record.Plan.RequestID, PlanDigest: record.Plan.Digest, HeaderDigest: record.HeaderDigest, OriginalRevision: record.Revision, OriginalEventDigest: eventDigest, StepID: step.ID, State: state, OriginalBinding: record.Plan.Binding, PreviousContainerID: record.Plan.Binding.RuntimeID, Process: grubbery.RuntimeProcess{ContainerID: strings.Repeat("9", 64), PID: 42, StartedAt: at.Add(-time.Minute), HostBootID: "11111111-2222-3333-4444-555555555555", EngineID: "daemon-fixture:one", ImageManifestDigest: "sha256:" + strings.Repeat("c", 64), RuntimeVersion: "4.6-5a7614f", ContextSHA256: strings.Repeat("e", 64)}, EvidenceBasis: "verified-pending-prefix", NativeEvidenceDigest: strings.Repeat("f", 64), ObservedAt: at}
	if state == "uncertain" {
		review.EvidenceBasis = "native-phase-observed"
		review.AppliedPhaseProof = &grubbery.Proof{Version: 1, PlanDigest: record.Plan.Digest, StepID: step.ID, BindingDigest: hashValue(record.Plan.Binding), NativeEvidenceDigest: review.NativeEvidenceDigest}
	}
	review.Digest = review.ContentDigest()
	if err = review.ValidateAncestry(record); err != nil {
		t.Fatal(err)
	}
	return review, record
}
func copyRecovery(r grubbery.RuntimeRecoveryReview) grubbery.RuntimeRecoveryReview {
	raw, _ := json.Marshal(r)
	var out grubbery.RuntimeRecoveryReview
	json.Unmarshal(raw, &out)
	return out
}

func TestRuntimeRecoveryModelPreservesOriginalPlanEventsAndExactNativePayloads(t *testing.T) {
	for _, state := range []string{"pending", "uncertain"} {
		t.Run(state, func(t *testing.T) {
			review, original := recoveryReview(t, state)
			before, _ := json.Marshal(original)
			binding, err := review.EffectiveBinding(original)
			if err != nil {
				t.Fatal(err)
			}
			expected := original.Plan.Binding
			expected.RuntimeID = review.Process.ContainerID
			if binding != expected {
				t.Fatal("recovery changed more than the selected runtime")
			}
			originalNamespace, _ := original.Plan.Binding.Namespace()
			effectiveNamespace, _ := binding.Namespace()
			if originalNamespace != effectiveNamespace {
				t.Fatal("recovery renamed app state")
			}
			after, _ := json.Marshal(original)
			if string(before) != string(after) || original.Validate() != nil {
				t.Fatal("recovery rewrote original plan/events")
			}
			fresh := copyRecovery(review)
			fresh.ObservedAt = fresh.ObservedAt.Add(time.Second)
			fresh.Digest = fresh.ContentDigest()
			if err = review.MatchesFreshReview(original, "", fresh, fresh.ObservedAt); err != nil {
				t.Fatal("identical independently refreshed facts refused", err)
			}
		})
	}
}
func TestRuntimeRecoveryModelRejectsStalePrefixButKeepsHistoricalAudit(t *testing.T) {
	review, original := recoveryReview(t, "pending")
	receipt := grubbery.RuntimeRecoveryReceipt{Version: 1, RequestID: strings.Repeat("8", 64), Review: review, AppliedAt: review.ObservedAt.Add(time.Second)}
	receipt.Digest = receipt.ContentDigest()
	if err := receipt.ValidateAncestry(original); err != nil {
		t.Fatal(err)
	}
	step := original.Plan.Steps[0]
	advanced, err := original.WithEvent(grubbery.Event{StepID: step.ID, State: "dispatched", At: review.ObservedAt.Add(2 * time.Second), Review: &grubbery.Review{Version: 1, PlanDigest: original.Plan.Digest, StepID: step.ID, BindingDigest: hashValue(original.Plan.Binding), PolicyRevision: original.Plan.Binding.PolicyRevision, NativeEvidenceDigest: strings.Repeat("7", 64)}})
	if err != nil {
		t.Fatal(err)
	}
	if err = review.MatchesCurrent(advanced, "", review.ObservedAt.Add(3*time.Second)); !errors.Is(err, grubbery.ErrCAS) {
		t.Fatal("old review silently became authority for later progress", err)
	}
	if err = receipt.ValidateAncestry(advanced); err != nil {
		t.Fatal("later legitimate progress invalidated immutable historical receipt", err)
	}
	if err = review.MatchesCurrent(original, strings.Repeat("6", 64), review.ObservedAt.Add(time.Second)); !errors.Is(err, grubbery.ErrCAS) {
		t.Fatal("changed recovery head accepted", err)
	}
	for _, now := range []time.Time{review.ObservedAt.Add(-time.Nanosecond), review.ObservedAt.Add(grubbery.RuntimeRecoveryReviewMaxAge + time.Nanosecond)} {
		if err = review.MatchesCurrent(original, "", now); !errors.Is(err, grubbery.ErrAdmission) {
			t.Fatal("review age ignored", err)
		}
	}
}
func TestRuntimeRecoveryFreshComparisonRejectsChangedProcessAndNativeEvidence(t *testing.T) {
	review, original := recoveryReview(t, "uncertain")
	changes := map[string]func(*grubbery.RuntimeRecoveryReview){
		"pid": func(r *grubbery.RuntimeRecoveryReview) { r.Process.PID++ },
		"started": func(r *grubbery.RuntimeRecoveryReview) {
			r.Process.StartedAt = r.Process.StartedAt.Add(time.Nanosecond)
		},
		"container": func(r *grubbery.RuntimeRecoveryReview) { r.Process.ContainerID = strings.Repeat("7", 64) },
		"host":      func(r *grubbery.RuntimeRecoveryReview) { r.Process.HostBootID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" },
		"engine":    func(r *grubbery.RuntimeRecoveryReview) { r.Process.EngineID = "different:daemon" },
		"image": func(r *grubbery.RuntimeRecoveryReview) {
			r.Process.ImageManifestDigest = "sha256:" + strings.Repeat("6", 64)
		},
		"runtime":          func(r *grubbery.RuntimeRecoveryReview) { r.Process.RuntimeVersion = "4.7" },
		"namespace-launch": func(r *grubbery.RuntimeRecoveryReview) { r.Process.ContextSHA256 = strings.Repeat("5", 64) },
		"native-proof": func(r *grubbery.RuntimeRecoveryReview) {
			r.NativeEvidenceDigest = strings.Repeat("4", 64)
			r.AppliedPhaseProof.NativeEvidenceDigest = r.NativeEvidenceDigest
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			fresh := copyRecovery(review)
			change(&fresh)
			fresh.ObservedAt = fresh.ObservedAt.Add(time.Second)
			fresh.Digest = fresh.ContentDigest()
			if err := review.MatchesFreshReview(original, "", fresh, fresh.ObservedAt); !errors.Is(err, grubbery.ErrConflict) {
				t.Fatal("changed fresh native facts matched reviewed authority", err)
			}
		})
	}
}
func TestRuntimeRecoveryModelCannotResealChangedImmutableIdentityOrProof(t *testing.T) {
	review, original := recoveryReview(t, "uncertain")
	for name, change := range map[string]func(*grubbery.RuntimeRecoveryReview){
		"actor":      func(r *grubbery.RuntimeRecoveryReview) { r.OriginalBinding.Actor = "new-key" },
		"pier":       func(r *grubbery.RuntimeRecoveryReview) { r.OriginalBinding.Continuity = strings.Repeat("2", 64) },
		"config":     func(r *grubbery.RuntimeRecoveryReview) { r.OriginalBinding.ConfigRevision++ },
		"generation": func(r *grubbery.RuntimeRecoveryReview) { r.OriginalBinding.Target.Generation++ },
		"framework": func(r *grubbery.RuntimeRecoveryReview) {
			r.OriginalBinding.FrameworkReceiptID = strings.Repeat("3", 64)
		},
		"step":            func(r *grubbery.RuntimeRecoveryReview) { r.StepID = original.Plan.Steps[1].ID },
		"omitted-proof":   func(r *grubbery.RuntimeRecoveryReview) { r.AppliedPhaseProof = nil },
		"wrong-proof":     func(r *grubbery.RuntimeRecoveryReview) { r.AppliedPhaseProof.PlanDigest = strings.Repeat("4", 64) },
		"unknown-outcome": func(r *grubbery.RuntimeRecoveryReview) { r.EvidenceBasis = "unknown" },
		"same-container":  func(r *grubbery.RuntimeRecoveryReview) { r.Process.ContainerID = r.PreviousContainerID },
	} {
		t.Run(name, func(t *testing.T) {
			changed := copyRecovery(review)
			change(&changed)
			changed.Digest = changed.ContentDigest()
			if err := changed.ValidateAncestry(original); err == nil {
				t.Fatal("changed immutable identity/insufficient evidence admitted")
			}
		})
	}
	copy := copyRecord(original)
	copy.Events[0].Digest = strings.Repeat("1", 64)
	if err := review.ValidateAncestry(copy); err == nil {
		t.Fatal("changed original chain accepted")
	}
	if !reflect.DeepEqual(review, copyRecovery(review)) {
		t.Fatal("review encoding lost exact bindings")
	}
}

func TestRuntimeRecoveryAuditLinkCannotNameAnotherContainerOrHead(t *testing.T) {
	first, original := recoveryReview(t, "pending")
	prior := grubbery.RuntimeRecoveryReceipt{Version: 1, RequestID: strings.Repeat("8", 64), Review: first, AppliedAt: first.ObservedAt.Add(time.Second)}
	prior.Digest = prior.ContentDigest()
	next := copyRecovery(first)
	next.PreviousRecoveryDigest = prior.Digest
	next.PreviousContainerID = prior.Review.Process.ContainerID
	next.Process.ContainerID = strings.Repeat("7", 64)
	next.Process.PID++
	next.Process.StartedAt = next.Process.StartedAt.Add(time.Hour)
	next.ObservedAt = next.ObservedAt.Add(time.Hour)
	next.Digest = next.ContentDigest()
	if err := next.ValidatePredecessor(original, &prior); err != nil {
		t.Fatal("exact prior audit refused", err)
	}
	if err := next.ValidatePredecessor(original, nil); !errors.Is(err, grubbery.ErrCAS) {
		t.Fatal("existing recovery head silently discarded", err)
	}
	if err := first.ValidatePredecessor(original, &prior); !errors.Is(err, grubbery.ErrCAS) {
		t.Fatal("old review applied over a new recovery head", err)
	}
	wrong := copyRecovery(next)
	wrong.PreviousContainerID = original.Plan.Binding.RuntimeID
	wrong.Digest = wrong.ContentDigest()
	if err := wrong.ValidatePredecessor(original, &prior); !errors.Is(err, grubbery.ErrCAS) {
		t.Fatal("audit link skipped the last effective process", err)
	}
	prior.RequestID = original.Plan.RequestID
	prior.Digest = prior.ContentDigest()
	if err := next.ValidatePredecessor(original, &prior); !errors.Is(err, grubbery.ErrCAS) {
		t.Fatal("invalid predecessor receipt accepted", err)
	}
}
func TestRuntimeRecoveryRejectsIncompleteProcessAndReceiptMetadata(t *testing.T) {
	review, original := recoveryReview(t, "pending")
	for _, change := range []func(*grubbery.RuntimeProcess){
		func(p *grubbery.RuntimeProcess) { p.PID = 1 }, func(p *grubbery.RuntimeProcess) { p.ContainerID = strings.Repeat("0", 64) }, func(p *grubbery.RuntimeProcess) { p.StartedAt = time.Time{} }, func(p *grubbery.RuntimeProcess) { p.HostBootID = strings.Repeat("x", 36) }, func(p *grubbery.RuntimeProcess) { p.HostBootID = "00000000-0000-0000-0000-000000000000" }, func(p *grubbery.RuntimeProcess) { p.EngineID = "/run/docker.sock" }, func(p *grubbery.RuntimeProcess) { p.ImageManifestDigest = "mutable:tag" }, func(p *grubbery.RuntimeProcess) { p.RuntimeVersion = "version\ncredential" }, func(p *grubbery.RuntimeProcess) { p.ContextSHA256 = "" },
	} {
		process := review.Process
		change(&process)
		if process.Validate() == nil {
			t.Fatal("incomplete/unbounded process metadata admitted")
		}
	}
	receipt := grubbery.RuntimeRecoveryReceipt{Version: 1, RequestID: strings.Repeat("8", 64), Review: review, AppliedAt: review.ObservedAt.Add(grubbery.RuntimeRecoveryReviewMaxAge + time.Nanosecond)}
	receipt.Digest = receipt.ContentDigest()
	if err := receipt.ValidateAncestry(original); err == nil {
		t.Fatal("receipt claims an expired review was applied")
	}
	receipt.AppliedAt = review.ObservedAt.Add(time.Second)
	receipt.RequestID = original.Plan.RequestID
	receipt.Digest = receipt.ContentDigest()
	if err := receipt.ValidateAncestry(original); err == nil {
		t.Fatal("recovery reused original mutation request ID")
	}
}
