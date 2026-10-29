package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type importRestoreBackend interface {
	PreviewImportRestore(context.Context, runtimequalify.ManagedState) (runtimequalify.ImportRestoreReview, error)
}

func (realBackend) PreviewImportRestore(ctx context.Context, s runtimequalify.ManagedState) (runtimequalify.ImportRestoreReview, error) {
	return runtimequalify.PreviewImportRestore(ctx, s)
}

func restoreResult(r runtimequalify.ImportRestoreReview) protocol.ImportRestoreResult {
	return protocol.ImportRestoreResult{Source: r.Source.Path, RestoreTo: r.OriginalPath, Destination: r.Destination, RetainAt: r.RetainedPath, ReviewDigest: r.Digest}
}

// The caller holds r.mu, serializing admission. No active job may own this
// instance; the worker rechecks all filesystem evidence after acquiring workGate.
func (r *Server) serveImportRestorePreview(w http.ResponseWriter, req *http.Request, role, id string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "source restoration review requires operator access"}})
		return
	}
	b, ok := r.b.(importRestoreBackend)
	if !ok {
		r.respondError(w, bad("update the runner for source restoration"))
		return
	}
	var body protocol.ImportRestorePreviewRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if body.ExpectedRevision == 0 || body.ExpectedRevision > math.MaxInt64 {
		r.respondError(w, bad("review requires a positive configuration revision"))
		return
	}
	v, err := loadInstance(r.s.db, id)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if v.Public.ConfigRevision != body.ExpectedRevision {
		r.respondError(w, &fault{409, protocol.APIError{Code: "revision_conflict", Message: "configuration changed; inspect a fresh snapshot", CurrentRevision: &v.Public.ConfigRevision}})
		return
	}
	if v.Public.ActiveOperationID != "" {
		r.respondError(w, conflict("instance_busy", "an operation is already active"))
		return
	}
	if v.Public.DesiredState != "stopped" || !v.Runtime.ImportRestoreEligible() {
		r.respondError(w, conflict("invalid_state", "review requires a stopped interrupted partial copy"))
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()
	review, err := b.PreviewImportRestore(ctx, v.Runtime)
	if err != nil {
		r.respondError(w, conflict("restoration_review_unavailable", err.Error()))
		return
	}
	out := protocol.ImportRestorePreview{ImportRestoreResult: restoreResult(review), RunnerID: r.s.runnerID, InstanceID: id, ConfigRevision: v.Public.ConfigRevision, SourceFiles: review.Source.Files, SourceBytes: review.Source.Bytes, Stage: "review"}
	if review.DestinationTree != nil {
		for _, entry := range review.DestinationTree.Entries {
			if !entry.Directory {
				out.DestinationFiles++
				out.DestinationBytes += entry.Size
			}
		}
	}
	if v.Runtime.ImportRestore != nil && v.Runtime.ImportRestore.Journal != nil {
		out.Continuation = true
		out.Stage = v.Runtime.ImportRestore.Journal.Stage
	}
	respond(w, 200, out)
}

// Ephemeral mount IDs are intentionally absent from persisted inventories.
// Compare the durable representation, not process-local inspection fields.
func sameRestoreRecord(a, b any) bool {
	x, xe := json.Marshal(a)
	y, ye := json.Marshal(b)
	return xe == nil && ye == nil && bytes.Equal(x, y)
}
