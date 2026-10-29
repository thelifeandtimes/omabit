package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type storageActionsBackend interface {
	StorageRoots() ([]pierstore.StorageRoot, error)
	RegisterStorageRoot(string, string) (pierstore.StorageRoot, error)
	InspectExternalPier(context.Context, string) (pierstore.ExternalSource, error)
}

func initialRevision(q querier, id string) (uint64, error) {
	reserved, err := projectReservations(q)
	if err != nil {
		return 0, err
	}
	if reserved.Instances[id] {
		return 0, conflict("instance_reserved", "instance ID is reserved by a project member")
	}
	var previous uint64
	err = q.QueryRow("SELECT revision FROM deleted_instances WHERE id=?", id).Scan(&previous)
	if errors.Is(err, sql.ErrNoRows) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	if previous >= math.MaxInt64 {
		return 0, conflict("revision_limit", "instance revision exhausted")
	}
	return previous + 1, nil
}

func (s *store) checkStorage(path string) error {
	if s.validateStorage != nil {
		return s.validateStorage(path)
	}
	return nil
}

func (r *Server) externalPathAvailable(b storageActionsBackend, path string) error {
	roots, err := b.StorageRoots()
	if err != nil {
		return conflict("storage_unavailable", err.Error())
	}
	all, err := instances(r.s.db)
	if err != nil {
		return err
	}
	claims, err := projectMemberStorageClaims(r.s.db)
	if err != nil {
		return err
	}
	for _, member := range claims {
		rootID, rel := pierstore.Split(member.Location.Path)
		if rootID == "" {
			rootID = "default"
		}
		for _, root := range roots {
			if root.ID == rootID && pierstore.Overlap(path, filepath.Join(root.Path, pierstore.MountPath(rel))) {
				return conflict("source_reserved", "source overlaps reserved project member storage")
			}
		}
	}
	for _, v := range all {
		paths := []string{v.Runtime.MoveTarget}
		if v.Runtime.Storage != nil {
			paths = append(paths, v.Runtime.Storage.Path)
		}
		for _, p := range paths {
			if p == "" {
				continue
			}
			id, rel := pierstore.Split(p)
			if id == "" {
				id = "default"
			}
			for _, root := range roots {
				if root.ID == id && pierstore.Overlap(path, filepath.Join(root.Path, pierstore.MountPath(rel))) {
					return conflict("source_reserved", "source overlaps a registered pier; use that instance's storage actions")
				}
			}
		}
		if v.Runtime.ExternalImport != nil && v.Runtime.ExternalImport.Stage != "complete" && (pierstore.Overlap(v.Runtime.ExternalImport.Source.Path, path) || v.Runtime.ExternalImport.HeldPath != "" && pierstore.Overlap(v.Runtime.ExternalImport.HeldPath, path)) {
			return conflict("source_reserved", "source is reserved by an existing import")
		}
	}
	return nil
}

func (realBackend) StorageRoots() ([]pierstore.StorageRoot, error) {
	return runtimequalify.StorageRoots()
}
func (realBackend) RegisterStorageRoot(id, path string) (pierstore.StorageRoot, error) {
	return runtimequalify.RegisterStorageRoot(id, path)
}
func (realBackend) InspectExternalPier(ctx context.Context, path string) (pierstore.ExternalSource, error) {
	return runtimequalify.InspectExternalPier(ctx, path)
}

func (r *Server) serveStorageActions(w http.ResponseWriter, req *http.Request, actor, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "storage actions require operator access"}})
		return
	}
	if req.Method != http.MethodPost {
		r.respondError(w, bad("storage actions require POST"))
		return
	}
	b, ok := r.b.(storageActionsBackend)
	if !ok {
		r.respondError(w, bad("update this runner for storage actions"))
		return
	}
	if req.URL.Path == "/v1/pier-import-preview" {
		var body protocol.ImportPierPreviewRequest
		if err := decode(w, req, &body); err != nil {
			r.respondError(w, err)
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
		defer cancel()
		if err := r.externalPathAvailable(b, body.Source); err != nil {
			r.respondError(w, err)
			return
		}
		source, err := b.InspectExternalPier(ctx, body.Source)
		if err != nil {
			r.respondError(w, conflict("source_unavailable", err.Error()))
			return
		}
		if body.StorageRoot != "" {
			preview, err := r.previewImport(ctx, b, source, body.StorageRoot)
			if err != nil {
				r.respondError(w, err)
			} else {
				respond(w, 200, preview)
			}
			return
		}
		respond(w, 200, source)
		return
	}
	if req.URL.Path == "/v1/storage-roots" {
		var body protocol.StorageRootRequest
		if err := decode(w, req, &body); err != nil {
			r.respondError(w, err)
			return
		}
		if !protocol.ValidID(body.RequestID) {
			r.respondError(w, bad("request identity required"))
			return
		}
		fingerprint, _ := json.Marshal(body)
		fp := "storage-root:" + string(fingerprint)
		if op, hit, err := replay(r.s.db, actor, body.RequestID, fp); hit || err != nil {
			if err != nil {
				r.respondError(w, err)
			} else {
				respond(w, 200, op)
			}
			return
		}
		if _, err := b.RegisterStorageRoot(body.ID, body.Path); err != nil {
			r.respondError(w, conflict("storage_unavailable", err.Error()))
			return
		}
		id, err := randomID()
		if err != nil {
			r.respondError(w, err)
			return
		}
		now := time.Now().UTC()
		op := protocol.Operation{ID: "op-" + id, RequestID: body.RequestID, Kind: "storage-root", Actor: actor, Status: "succeeded", CreatedAt: now, FinishedAt: &now}
		tx, err := r.s.db.Begin()
		if err != nil {
			r.respondError(w, err)
			return
		}
		defer tx.Rollback()
		if err = insertOperation(tx, op, fp); err == nil {
			err = tx.Commit()
		}
		if err != nil {
			r.respondError(w, err)
			return
		}
		respond(w, 200, op)
		return
	}
	var body protocol.ImportPierRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if !protocol.ValidID(body.ID) || !protocol.ValidID(body.RequestID) || !protocol.ValidFakeShip(body.Ship) || !pierstore.ValidPath(body.Path) || !epochs.ValidDigest(body.Fingerprint) || !body.ConfirmStopped || body.Relink {
		r.respondError(w, bad("import requires source review, destination, expected ship and explicit stopped/move confirmation"))
		return
	}
	kind := pierstore.Kind(body.Path)
	if kind != "fake" && kind != protocol.ShipClass(body.Ship) {
		r.respondError(w, bad("destination classification does not match ship"))
		return
	}
	data, _ := json.Marshal(body)
	fp := "import-pier:" + string(data)
	if op, hit, err := replay(r.s.db, actor, body.RequestID, fp); hit || err != nil {
		if err != nil {
			r.respondError(w, err)
		} else {
			respond(w, 200, op)
		}
		return
	}
	if err := r.externalPathAvailable(b, body.Source); err != nil {
		r.respondError(w, err)
		return
	}
	source, err := b.InspectExternalPier(req.Context(), body.Source)
	if err != nil {
		r.respondError(w, conflict("source_unavailable", err.Error()))
		return
	}
	if source.Fingerprint != body.Fingerprint {
		r.respondError(w, conflict("source_changed", "source changed since review"))
		return
	}
	if body.Identify {
		g, ok := r.b.(guidedImportBackend)
		if !ok {
			r.respondError(w, bad("automatic identification is unavailable"))
			return
		}
		identity, err := g.InspectPierIdentity(req.Context(), body.Source)
		if err != nil || identity.Ship != body.Ship || identity.Kind != kind {
			r.respondError(w, conflict("identity_changed", "pier identity or network differs from the import review"))
			return
		}
	}
	op, err := r.s.enrollPierSource(actor, body.EnrollPierRequest, pierstore.Candidate{Path: body.Path, Kind: kind, Identity: source.Identity}, fp, &source, body.Identify)
	if err != nil {
		r.respondError(w, err)
		return
	}
	r.launch(op)
	respond(w, 202, op)
}
