package grubbery

import (
	"context"
	"reflect"
)

// RuntimeRecoveryEvidence is native read-only evidence, not a process selection
// or authority. The server must independently bind it to fresh process facts,
// authenticate the original actor and CAS its exact original prefix/audit head.
type RuntimeRecoveryEvidence struct {
	EvidenceBasis        string
	NativeEvidenceDigest string
	AppliedPhaseProof    *Proof
}

func (b *NativeBackend) recoveryHistory(ctx context.Context, original Record) ([]Record, error) {
	records, err := b.history(ctx, original.Plan.Binding)
	if err != nil {
		return nil, ErrHistory
	}
	found := false
	for _, record := range records {
		if record.Validate() != nil || CheckNativeSource(record.Plan.Source) != nil {
			return nil, ErrHistory
		}
		if record.Plan.Version != 2 {
			return nil, ErrLegacyPlan
		}
		if record.Plan.RequestID == original.Plan.RequestID {
			if found || !reflect.DeepEqual(record, original) {
				return nil, ErrCAS
			}
			found = true
		}
	}
	if !found {
		return nil, ErrMissing
	}
	own, err := DeriveOwnership(records)
	if err != nil || own.PendingRequestID != original.Plan.RequestID {
		return nil, ErrHistory
	}
	return records, nil
}

// ObserveRuntimeRecovery checks the complete retained native prefix. An unknown
// original effect must already be observable; this never dispatches or resolves
// its phase. Prior observed compile phases are checked again inside the same
// scoped capture before cleanup, including after a process/compile-cache restart.
// No pending compile or other pending original phase is submitted by review.
func (b *NativeBackend) ObserveRuntimeRecovery(ctx context.Context, original Record) (RuntimeRecoveryEvidence, error) {
	var out RuntimeRecoveryEvidence
	if original.Validate() != nil {
		return out, ErrHistory
	}
	if original.Version != 2 || original.Plan.Version != 2 {
		return out, ErrLegacyPlan
	}
	step, done, err := original.Current()
	if err != nil || done {
		return out, ErrPhase
	}
	state, err := original.State(step.ID)
	if err != nil || state != "pending" && state != "dispatched" && state != "acknowledged" && state != "uncertain" {
		return out, ErrPhase
	}
	if _, err = b.guard(ctx, original.Plan.Binding, false); err != nil {
		return out, ErrAdmission
	}
	records, err := b.recoveryHistory(ctx, original)
	if err != nil {
		return out, err
	}
	candidate := ""
	if state != "pending" {
		candidate = step.ID
	}
	projected, err := projection(original.Plan.Binding, records, candidate)
	if err != nil {
		return out, err
	}
	checks := func(c *nativeCapture) error {
		for _, record := range records {
			release, err := original.Plan.Binding.ReleaseRoot(record.Plan.RequestID)
			if err != nil {
				return err
			}
			versionAllowed := false
			for _, phase := range record.Plan.Steps {
				if phase.Path == release+"/version.json" {
					phaseState, err := record.State(phase.ID)
					if err != nil {
						return ErrHistory
					}
					versionAllowed = phaseState == "observed" || phase.ID == candidate
				}
			}
			for _, phase := range record.Plan.Steps {
				if phase.Kind != "observe-code" {
					continue
				}
				phaseState, err := record.State(phase.ID)
				if err != nil {
					return ErrHistory
				}
				if phaseState != "observed" && phase.ID != candidate {
					continue
				}
				// The historical release path/IDs come only from validated saved
				// plans. Seal the read to this exact current operation's binding.
				read, err := newRead(original.Plan.Binding, "recovery-compiled-prefix", record.Plan.Digest, phase.ID, "build"+phase.Path, "all=1")
				if err != nil {
					return err
				}
				response, err := c.read(ctx, read)
				if err != nil {
					clear(response.Body)
					return err
				}
				report, parseErr := ParseBuildReport(response.Body, phase.Path)
				clear(response.Body)
				if response.Status != 200 || parseErr != nil || report.CheckStaged(record.Plan.Source, versionAllowed) != nil {
					return ErrObservation
				}
			}
		}
		return nil
	}
	captured, err := b.captureWithChecks(ctx, original.Plan.Binding, projected, checks)
	if err != nil {
		return out, err
	}
	if _, err = b.guard(ctx, original.Plan.Binding, false); err != nil {
		return out, ErrAdmission
	}
	fresh, err := b.recoveryHistory(ctx, original)
	if err != nil {
		return out, err
	}
	if !reflect.DeepEqual(fresh, records) {
		return out, ErrCAS
	}
	out.EvidenceBasis, out.NativeEvidenceDigest = "verified-pending-prefix", captured.digest()
	if candidate != "" {
		out.EvidenceBasis = "native-phase-observed"
		out.AppliedPhaseProof = &Proof{1, original.Plan.Digest, step.ID, bindingDigest(original.Plan), out.NativeEvidenceDigest}
	}
	return out, nil
}
