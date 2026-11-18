package grubbery

import "errors"

// Origin points to the original observed marker effect. Later app/source updates
// cannot replace that ownership claim merely by presenting the same marker hash.
type Origin struct {
	RequestID   string `json:"request_id"`
	StepID      string `json:"step_id"`
	EventDigest string `json:"event_digest"`
}

type Ownership struct {
	Version          int       `json:"version"`
	Namespace        Namespace `json:"namespace"`
	OwnerSHA256      string    `json:"owner_sha256"`
	SourceOrigin     *Origin   `json:"source_origin,omitempty"`
	DeskOrigin       *Origin   `json:"desk_origin,omitempty"`
	Epoch            uint64    `json:"epoch"`
	ReleaseRoot      string    `json:"release_root,omitempty"`
	SourceDigest     string    `json:"source_digest,omitempty"`
	AppliedRequestID string    `json:"applied_request_id,omitempty"`
	PendingRequestID string    `json:"pending_request_id,omitempty"`
}

// DeriveOwnership consumes immutable operation headers in trusted database
// creation order, not wall-clock order. It is a saved-history projection, never
// native ownership evidence. Admission must also inspect the actual namespace.
// No current/ownership row can silently replace the original marker receipts.
func DeriveOwnership(records []Record) (Ownership, error) {
	if len(records) == 0 {
		return Ownership{}, ErrMissing
	}
	var out Ownership
	seen := map[string]bool{}
	for i, r := range records {
		if r.Validate() != nil || seen[r.Plan.RequestID] || out.PendingRequestID != "" {
			return Ownership{}, ErrHistory
		}
		seen[r.Plan.RequestID] = true
		n, err := r.Plan.Binding.Namespace()
		if err != nil {
			return Ownership{}, ErrHistory
		}
		marker, err := r.Plan.Binding.OwnerMarker()
		if err != nil {
			return Ownership{}, ErrHistory
		}
		ownerHash := hash(marker)
		if i == 0 {
			out = Ownership{Version: 1, Namespace: n, OwnerSHA256: ownerHash}
		} else if n != out.Namespace || ownerHash != out.OwnerSHA256 {
			return Ownership{}, errors.New("app history changed stable namespace identity")
		}
		before := r.Plan.Before
		sourceState, deskState := "absent", "absent"
		sourceHash, deskHash := "", ""
		if out.SourceOrigin != nil {
			sourceState = "owned"
			sourceHash = out.OwnerSHA256
		}
		if out.DeskOrigin != nil {
			deskState = "owned"
			deskHash = out.OwnerSHA256
		}
		if before.SourceState != sourceState || before.SourceOwnerSHA256 != sourceHash || before.DeskState != deskState || before.DeskOwnerSHA256 != deskHash || before.Epoch != out.Epoch || before.ReleaseRoot != out.ReleaseRoot {
			return Ownership{}, errors.New("app operation is missing its exact ownership/version predecessor")
		}
		steps := map[string]Step{}
		for _, step := range r.Plan.Steps {
			steps[step.ID] = step
		}
		for _, event := range r.Events {
			if event.State != "observed" {
				continue
			}
			step := steps[event.StepID]
			if step.Kind != "make-file" {
				continue
			}
			var destination **Origin
			switch step.Path {
			case n.SourceRoot + "/omabit-owner.json":
				destination = &out.SourceOrigin
			case n.DataRoot + "/omabit-owner.json":
				destination = &out.DeskOrigin
			default:
				continue
			}
			if *destination != nil || step.PayloadSHA256 != out.OwnerSHA256 {
				return Ownership{}, errors.New("app marker origin was replaced")
			}
			*destination = &Origin{RequestID: r.Plan.RequestID, StepID: step.ID, EventDigest: event.Digest}
		}
		_, done, err := r.Current()
		if err != nil {
			return Ownership{}, ErrHistory
		}
		if done {
			if out.SourceOrigin == nil || out.DeskOrigin == nil || r.Plan.Epoch != out.Epoch+1 {
				return Ownership{}, errors.New("app completion has no observed ownership/version ancestry")
			}
			out.Epoch = r.Plan.Epoch
			out.ReleaseRoot, _ = r.Plan.Binding.ReleaseRoot(r.Plan.RequestID)
			out.SourceDigest = r.Plan.Source.Manifest.Digest
			out.AppliedRequestID = r.Plan.RequestID
		} else {
			out.PendingRequestID = r.Plan.RequestID
		}
	}
	return out, nil
}
