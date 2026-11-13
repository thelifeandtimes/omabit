//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Server) checkStageSelection(intent deskwork.StageIntent, actor string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || !r.routineAuthorized(actor) {
		return conflict("runner_unavailable", "source staging authority is unavailable")
	}
	setting, err := r.s.deskSetting(intent.Target)
	if err != nil {
		return err
	}
	if setting.Revision != intent.SettingsRevision || setting.Desk == nil || setting.Desk.Workflow != "mounted" {
		return conflict("desk_settings_changed", "source workflow changed; inspect it again")
	}
	return nil
}
func (r *Server) serveDeskStage(w http.ResponseWriter, req *http.Request, actor, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "source staging requires operator access"}})
		return
	}
	upload := req.URL.Path == protocol.DeskStagesPath
	id := strings.TrimPrefix(req.URL.Path, protocol.DeskStagesPath+"/")
	if upload && req.Method != "POST" || !upload && req.Method != "GET" && req.Method != "DELETE" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !upload && !protocol.ValidDeskOperationID(id) {
		r.respondError(w, bad("invalid source stage ID"))
		return
	}
	if req.Method == "DELETE" {
		r.discardDeskStage(w, req, actor, id)
		return
	}
	controller := http.NewResponseController(w)
	_ = controller.SetReadDeadline(time.Now().Add(protocol.DeskStageTimeout))
	_ = controller.SetWriteDeadline(time.Now().Add(protocol.DeskStageTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskStageTimeout)
	defer cancel()
	stop := context.AfterFunc(r.ctx, cancel)
	defer stop()
	originalBody := req.Body
	interrupt := context.AfterFunc(ctx, func() { _ = controller.SetReadDeadline(time.Now()); _ = originalBody.Close() })
	defer interrupt()
	var stage deskwork.CachedStage
	var err error
	if upload {
		if !r.deskStageMu.TryLock() {
			r.respondError(w, conflict("source_stage_busy", "another source upload is active; reconcile its stage before retrying"))
			return
		}
		defer r.deskStageMu.Unlock()
		raw := req.Header.Get(protocol.DeskStageHeader)
		if len(raw) > 8192 {
			r.respondError(w, bad("source descriptor exceeds bound"))
			return
		}
		body, e := base64.RawURLEncoding.Strict().DecodeString(raw)
		if e != nil || len(raw) > 8192 || req.Header.Get("Content-Type") != "application/octet-stream" || req.ContentLength > deskwork.MaxBundleBytes {
			r.respondError(w, bad("invalid source upload descriptor"))
			return
		}
		var intent deskwork.StageIntent
		d := json.NewDecoder(bytes.NewReader(body))
		d.DisallowUnknownFields()
		if d.Decode(&intent) != nil || d.Decode(new(any)) != io.EOF || intent.Validate() != nil {
			r.respondError(w, bad("invalid source transfer intent"))
			return
		}
		if err = r.checkStageSelection(intent, actor); err != nil {
			r.respondError(w, err)
			return
		}
		root, e := deskwork.PrepareStageCache(r.s.directory)
		if e != nil {
			r.respondError(w, conflict("source_stage_unavailable", "private source cache unavailable"))
			return
		}
		req.Body = http.MaxBytesReader(w, req.Body, deskwork.MaxBundleBytes)
		stage, err = deskwork.CacheStage(ctx, root, intent, actor, req.Body)
		if err == nil {
			err = r.checkStageSelection(intent, actor)
		}
	} else {
		stage, _, err = deskwork.ReadStage(ctx, filepath.Join(r.s.directory, "desk-stages-v1"), id, actor)
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.respondError(w, &fault{404, protocol.APIError{Code: "source_stage_not_found", Message: "no verified source stage found for this ID"}})
			return
		}
		r.respondError(w, conflict("source_stage_unverified", "source stage could not be verified; inspect the original stage ID and cache state before retrying"))
		return
	}
	status := 200
	if upload {
		status = 201
	}
	respond(w, status, protocol.DeskStage{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Receipt: stage.Receipt, ReadAt: time.Now().UTC()})
}

func (r *Server) discardDeskStage(w http.ResponseWriter, req *http.Request, actor, id string) {
	var input protocol.DeskStageDiscardRequest
	if err := decode(w, req, &input); err != nil {
		r.respondError(w, err)
		return
	}
	if !protocol.ValidDeskOperationID(input.SourceDigest) {
		r.respondError(w, bad("require the expected source digest"))
		return
	}
	if !r.deskStageMu.TryLock() {
		r.respondError(w, conflict("source_stage_busy", "a source transfer is active"))
		return
	}
	defer r.deskStageMu.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || !r.routineAuthorized(actor) {
		r.respondError(w, conflict("runner_unavailable", "source staging authority is unavailable"))
		return
	}
	// Hold the mutation lock through deletion. Conservatively retain all stages
	// for a target while any desk effect is unresolved, including another actor's.
	guard := func(stage deskwork.StageReceipt) error {
		ids, err := deskOperationIDs(r.s.db)
		if err != nil {
			return err
		}
		for _, operationID := range ids {
			op, err := loadDeskOperation(r.s.db, operationID)
			if err != nil {
				return err
			}
			state := op.Steps[len(op.Steps)-1].State
			if op.Intent.Target == stage.Intent.Target && (state == "planned" || state == "dispatched" || state == "acknowledged" || state == "uncertain") {
				return conflict("desk_outcome_unresolved", "reconcile the desk operation before discarding its source")
			}
		}
		return nil
	}
	if err := deskwork.DiscardStage(filepath.Join(r.s.directory, "desk-stages-v1"), id, actor, input.SourceDigest, guard); err != nil {
		r.respondError(w, conflict("source_discard_refused", "source cache retained; check the stage owner, digest and unresolved desk operations"))
		return
	}
	respond(w, 200, protocol.DeskStageDiscard{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, ID: id, SourceDigest: input.SourceDigest, Discarded: true})
}
