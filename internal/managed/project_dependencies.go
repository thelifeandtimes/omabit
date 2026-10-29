package managed

import (
	"context"
	"log"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

type projectDependenciesBackend interface{ PrepareProjectDependencies(context.Context) error }

func (r *Server) serveProjectDependencies(w http.ResponseWriter, q *http.Request, role string) {
	if role != "operator" || q.Method != http.MethodPost {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "project downloads require an operator"}})
		return
	}
	var body protocol.ProjectDependenciesRequest
	if err := decode(w, q, &body); err != nil {
		r.respondError(w, err)
		return
	}
	replay, err := protocol.CreationReplayOnly(q.Header)
	if err != nil || replay || !body.ConfirmDownload {
		r.respondError(w, bad("explicit project download consent required; this cache step is not a replay lookup"))
		return
	}
	b, ok := r.b.(projectDependenciesBackend)
	if !ok {
		r.respondError(w, conflict("project_dependencies_unsupported", "runner needs project download support"))
		return
	}
	if err = r.beginUpdateBulk(); err != nil {
		r.respondError(w, err)
		return
	}
	defer r.endUpdateBulk()
	ctx, cancel := context.WithTimeout(q.Context(), 10*time.Minute)
	defer cancel()
	stop := context.AfterFunc(r.ctx, cancel)
	defer stop()
	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Now().Add(11 * time.Minute))
	select {
	case r.workGate <- struct{}{}:
		defer func() { <-r.workGate }()
	case <-ctx.Done():
		r.respondError(w, conflict("project_dependencies_cancelled", "project downloads cancelled"))
		return
	}
	if ctx.Err() != nil || r.ctx.Err() != nil || r.failed.Load() {
		r.respondError(w, conflict("runner_unavailable", "runner is unavailable"))
		return
	}
	if err = b.PrepareProjectDependencies(ctx); err != nil {
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		log.Printf("project-dependencies failed=%q", message)
		r.respondError(w, conflict("project_dependencies_failed", "could not verify pinned project image and pill; check internet access, disk space and runner logs before continuing"))
		return
	}
	respond(w, 200, protocol.ProjectDependenciesResult{SchemaVersion: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Image: candidate.Image, Pill: protocol.BootPill{SHA256: candidate.PillSHA256, Bytes: candidate.PillBytes}})
}
