package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func clientStopReviewFixture(t *testing.T) protocol.ProjectStopReview {
	t.Helper()
	now := time.Now().UTC()
	r := protocol.ProjectStopReview{Version: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", ProjectID: "project", ProjectRevision: 1, ProjectGeneration: 1, InstanceID: "member", ContainerGeneration: 1, OriginalOperationID: "op-stop", OperationDigest: strings.Repeat("a", 64), LedgerDigest: strings.Repeat("b", 64), EvidenceBasis: "retained-stop-intent", Observation: protocol.ProjectStopObservation{ContainerID: strings.Repeat("c", 64), ContractDigest: strings.Repeat("d", 64), EngineID: "engine", AnchorID: strings.Repeat("e", 64), BridgeID: strings.Repeat("f", 64), BootID: "boot", NamespaceDevice: 5, NamespaceInode: 42, Storage: pierstore.Location{Path: "fakeships/member/pier", Identity: pierstore.Identity{Inode: 11, UrbInode: 12}}, StartedAt: now.Add(-time.Minute), FinishedAt: now.Add(-time.Second), ObservedAt: now}}
	r.OriginalRequestDigest = strings.Repeat("1", 64)
	r.ReviewDigest = r.Digest()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestProjectStopReconcileClientBindsReviewAndDoesNotRetry(t *testing.T) {
	for _, mode := range []string{"valid", "lookup", "unsupported", "observer", "review-actor", "review-target", "lost", "result-target", "result-original", "result-actor", "result-status"} {
		t.Run(mode, func(t *testing.T) {
			review := clientStopReviewFixture(t)
			req := protocol.ProjectStopReconcileRequest{RequestID: "original-reconciliation", Review: review, ConfirmAction: true}
			if mode == "review-actor" {
				req.Review.Actor = "different"
				req.Review.ReviewDigest = req.Review.Digest()
			}
			if mode == "review-target" {
				req.Review.RunnerID = "different"
				req.Review.ReviewDigest = req.Review.Digest()
			}
			var posts atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{protocol.ProjectStopRecoveryFeature, protocol.CreationReplayFeature}}
					if mode == "unsupported" {
						h.Features = nil
					}
					if mode == "observer" {
						h.AccessRole = "observer"
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				posts.Add(1)
				if q.Method != "POST" || q.URL.Path != protocol.ProjectStopReconcilePath("project") {
					t.Error("unselected endpoint")
				}
				if (q.Header.Get(protocol.CreationReplayHeader) == "true") != (mode == "lookup") {
					t.Error("lookup-only guard changed")
				}
				if mode == "lost" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
					} else {
						conn.Close()
					}
					return
				}
				now := time.Now().UTC()
				out := protocol.ProjectStopReconcileResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", OriginalOperationID: review.OriginalOperationID, ReviewDigest: review.ReviewDigest, VerifiedState: "stopped", Operation: protocol.Operation{ID: "op-resolution", ProjectID: "project", InstanceID: "member", Kind: "reconcile-project-ship-stop", Actor: "actor", ConfigRevision: 1, RequestID: req.RequestID, Status: "succeeded", CreatedAt: now, FinishedAt: &now}}
				switch mode {
				case "result-target":
					out.Operation.InstanceID = "other"
				case "result-original":
					out.OriginalOperationID = "other"
				case "result-actor":
					out.Operation.Actor = "other"
				case "result-status":
					out.Operation.Status = "queued"
				}
				json.NewEncoder(w).Encode(out)
			})
			ctx := context.Background()
			if mode == "lookup" {
				ctx = WithCreationReplayOnly(ctx)
			}
			out, err := c.ReconcileProjectStop(ctx, Identity{RunnerID: "runner", Mode: "simulation"}, "project", req)
			if mode == "valid" || mode == "lookup" {
				if err != nil || out.Operation.ID != "op-resolution" || posts.Load() != 1 {
					t.Fatal(out, err, posts.Load())
				}
				return
			}
			if err == nil || out.Operation.ID != "" {
				t.Fatal("invalid outcome accepted", mode, out)
			}
			preflight := mode == "unsupported" || mode == "observer" || strings.HasPrefix(mode, "review-")
			if preflight {
				if posts.Load() != 0 {
					t.Fatal("preflight reached an effect")
				}
			} else {
				var uncertain *OutcomeUncertainError
				if posts.Load() != 1 || !errors.As(err, &uncertain) {
					t.Fatal("lost outcome retried or misclassified", posts.Load(), err)
				}
			}
		})
	}
}
