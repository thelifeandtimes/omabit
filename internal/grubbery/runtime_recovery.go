package grubbery

import (
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

const RuntimeRecoveryReviewMaxAge = 5 * time.Minute
const MaxRuntimeRecoveries = 64

// RuntimeProcess is trusted adapter evidence, never caller authority. Context
// binds verified launch/network/namespace facts not otherwise represented here.
// The factory must independently observe them and fence this exact process on
// every later call. No command, environment, cookie, path or secret is retained.
type RuntimeProcess struct {
	ContainerID         string    `json:"container_id"`
	PID                 uint32    `json:"pid"`
	StartedAt           time.Time `json:"started_at"`
	HostBootID          string    `json:"host_boot_id"`
	EngineID            string    `json:"engine_id"`
	ImageManifestDigest string    `json:"image_manifest_digest"`
	RuntimeVersion      string    `json:"runtime_version"`
	ContextSHA256       string    `json:"context_sha256"`
}

func recoveryDigest(s string) bool { return digest.MatchString(s) && s != strings.Repeat("0", 64) }
func recoveryBootID(id string) bool {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	boot := strings.ReplaceAll(id, "-", "")
	if len(boot) != 32 || boot == strings.Repeat("0", 32) {
		return false
	}
	for _, c := range boot {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func (p RuntimeProcess) Validate() error {
	if !recoveryDigest(p.ContainerID) || p.PID < 2 || !utcTime(p.StartedAt) || !p.StartedAt.After(time.Unix(0, 0)) || !recoveryBootID(p.HostBootID) || !hostcompat.ValidEngineIdentity(p.EngineID) || !strings.HasPrefix(p.ImageManifestDigest, "sha256:") || !recoveryDigest(strings.TrimPrefix(p.ImageManifestDigest, "sha256:")) || !recoveryDigest(p.ContextSHA256) {
		return ErrConflict
	}
	if len(p.RuntimeVersion) == 0 || len(p.RuntimeVersion) > 64 {
		return ErrConflict
	}
	for _, c := range p.RuntimeVersion {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_' || c == '+') {
			return ErrConflict
		}
	}
	return nil
}

// RuntimeRecoveryReview names an exact saved prefix and a newly verified
// replacement process. It changes no original plan/binding/event, resolves no
// phase, grants no Commit permission and performs no runtime action. Its digest
// is a content binding, NOT a signature or authorization token. A server must
// authenticate the actor, derive this review from actual state and reobserve
// every fact at apply. Store/runtime/API adapters are deliberately separate.
type RuntimeRecoveryReview struct {
	Version                int            `json:"version"`
	OriginalRequestID      string         `json:"original_request_id"`
	PlanDigest             string         `json:"plan_digest"`
	HeaderDigest           string         `json:"header_digest"`
	OriginalRevision       uint64         `json:"original_revision"`
	OriginalEventDigest    string         `json:"original_event_digest"`
	StepID                 string         `json:"step_id"`
	State                  string         `json:"state"`
	OriginalBinding        Binding        `json:"original_binding"`
	PreviousRecoveryDigest string         `json:"previous_recovery_digest,omitempty"`
	PreviousContainerID    string         `json:"previous_container_id"`
	Process                RuntimeProcess `json:"process"`
	EvidenceBasis          string         `json:"evidence_basis"`
	NativeEvidenceDigest   string         `json:"native_evidence_digest"`
	AppliedPhaseProof      *Proof         `json:"applied_phase_proof,omitempty"`
	ObservedAt             time.Time      `json:"observed_at"`
	Digest                 string         `json:"digest"`
}

func (r RuntimeRecoveryReview) ContentDigest() string {
	r.Digest = ""
	return hash([]byte("grubbery-runtime-review-v1\x00" + textJSON(r)))
}
func lastRecordDigest(r Record) string {
	if len(r.Events) == 0 {
		return r.HeaderDigest
	}
	return r.Events[len(r.Events)-1].Digest
}

// ValidateAncestry accepts later app progress only when this exact reviewed
// prefix remains intact. Fresh apply uses MatchesCurrent instead, so later
// progress can never silently refresh an old review into new authority.
func (r RuntimeRecoveryReview) ValidateAncestry(original Record) error {
	if r.Version != 1 || original.Validate() != nil || original.Version != 2 || r.OriginalRequestID != original.Plan.RequestID || r.PlanDigest != original.Plan.Digest || r.HeaderDigest != original.HeaderDigest || r.OriginalBinding != original.Plan.Binding || r.OriginalRevision > original.Revision || !recoveryDigest(r.OriginalEventDigest) || r.Digest != r.ContentDigest() {
		return ErrHistory
	}
	if r.Process.Validate() != nil || !recoveryDigest(r.PreviousContainerID) || r.Process.ContainerID == r.PreviousContainerID || !utcTime(r.ObservedAt) || r.ObservedAt.Before(r.Process.StartedAt) || !recoveryDigest(r.NativeEvidenceDigest) {
		return ErrConflict
	}
	if r.PreviousRecoveryDigest == "" {
		if r.PreviousContainerID != original.Plan.Binding.RuntimeID {
			return ErrConflict
		}
	} else if !recoveryDigest(r.PreviousRecoveryDigest) {
		return ErrConflict
	}
	prefix := original
	prefix.Events = original.Events[:r.OriginalRevision]
	prefix.Revision = r.OriginalRevision
	if prefix.Validate() != nil || lastRecordDigest(prefix) != r.OriginalEventDigest {
		return ErrHistory
	}
	step, done, err := prefix.Current()
	if err != nil || done || step.ID != r.StepID {
		return ErrPhase
	}
	state, err := prefix.State(step.ID)
	if err != nil || state != r.State {
		return ErrPhase
	}
	switch state {
	case "pending":
		if r.EvidenceBasis != "verified-pending-prefix" || r.AppliedPhaseProof != nil {
			return ErrObservation
		}
	case "dispatched", "acknowledged", "uncertain":
		if r.EvidenceBasis != "native-phase-observed" || r.AppliedPhaseProof == nil || !r.AppliedPhaseProof.matches(original.Plan, r.StepID) || r.AppliedPhaseProof.NativeEvidenceDigest != r.NativeEvidenceDigest {
			return ErrObservation
		}
	default:
		return ErrPhase
	}
	return nil
}
func (r RuntimeRecoveryReview) MatchesCurrent(original Record, previousRecoveryDigest string, now time.Time) error {
	if err := r.ValidateAncestry(original); err != nil {
		return err
	}
	if original.Revision != r.OriginalRevision || r.PreviousRecoveryDigest != previousRecoveryDigest {
		return ErrCAS
	}
	if !utcTime(now) || now.Before(r.ObservedAt) || now.Sub(r.ObservedAt) > RuntimeRecoveryReviewMaxAge {
		return ErrAdmission
	}
	return nil
}

// EffectiveBinding changes only the explicitly reviewed container selection.
// This helper does not validate live facts or make it legal to rewrite a Plan.
// The journal and native request payloads must keep their ORIGINAL bindings/IDs;
// a separately authorized runtime adapter fences the effective process.
func (r RuntimeRecoveryReview) EffectiveBinding(original Record) (Binding, error) {
	if err := r.ValidateAncestry(original); err != nil {
		return Binding{}, err
	}
	binding := r.OriginalBinding
	binding.RuntimeID = r.Process.ContainerID
	return binding, nil
}

type RuntimeRecoveryReceipt struct {
	Version   int                   `json:"version"`
	RequestID string                `json:"request_id"`
	Review    RuntimeRecoveryReview `json:"review"`
	AppliedAt time.Time             `json:"applied_at"`
	Digest    string                `json:"digest"`
}

func (r RuntimeRecoveryReceipt) ContentDigest() string {
	r.Digest = ""
	return hash([]byte("grubbery-runtime-receipt-v1\x00" + textJSON(r)))
}
func (r RuntimeRecoveryReceipt) ValidateAncestry(original Record) error {
	if r.Version != 1 || !recoveryDigest(r.RequestID) || r.RequestID == original.Plan.RequestID || r.Digest != r.ContentDigest() || !utcTime(r.AppliedAt) || r.AppliedAt.Before(r.Review.ObservedAt) || r.AppliedAt.Sub(r.Review.ObservedAt) > RuntimeRecoveryReviewMaxAge {
		return ErrHistory
	}
	return r.Review.ValidateAncestry(original)
}

// MatchesFreshReview requires a separately derived server-side observation of
// all reviewed facts. It ignores only the newer observation time after both age
// and exact-prefix checks. Supplying two caller-created values establishes no
// authority; the authenticated adapter must produce fresh itself.
func (r RuntimeRecoveryReview) MatchesFreshReview(original Record, previousRecoveryDigest string, fresh RuntimeRecoveryReview, now time.Time) error {
	if err := r.MatchesCurrent(original, previousRecoveryDigest, now); err != nil {
		return err
	}
	if err := fresh.MatchesCurrent(original, previousRecoveryDigest, now); err != nil {
		return err
	}
	if fresh.ObservedAt.Before(r.ObservedAt) {
		return ErrAdmission
	}
	fresh.ObservedAt = r.ObservedAt
	if fresh.ContentDigest() != r.Digest {
		return ErrConflict
	}
	return nil
}

// ValidatePredecessor checks the immutable audit link. The transactional store
// must separately select/compare the actual latest receipt and enforce the
// recovery-count bound; a caller-selected predecessor cannot move that head.
func (r RuntimeRecoveryReview) ValidatePredecessor(original Record, previous *RuntimeRecoveryReceipt) error {
	if err := r.ValidateAncestry(original); err != nil {
		return err
	}
	if previous == nil {
		if r.PreviousRecoveryDigest != "" || r.PreviousContainerID != original.Plan.Binding.RuntimeID {
			return ErrCAS
		}
		return nil
	}
	if previous.ValidateAncestry(original) != nil || r.PreviousRecoveryDigest != previous.Digest || r.PreviousContainerID != previous.Review.Process.ContainerID || r.OriginalRevision < previous.Review.OriginalRevision {
		return ErrCAS
	}
	return nil
}
