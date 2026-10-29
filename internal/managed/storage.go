package managed

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type storageBackend interface {
	PierSources(context.Context) ([]pierstore.Candidate, error)
}

func (realBackend) PierSources(ctx context.Context) ([]pierstore.Candidate, error) {
	return runtimequalify.ListPierSources(ctx)
}

func reservePierPath(q querier, id, path string) error {
	claims, err := projectStorageLocations(q)
	if err != nil {
		return err
	}
	for member, location := range claims {
		if pierstore.Overlap(pierstore.MountPath(path), pierstore.MountPath(location.Path)) {
			return conflict("pier_path_conflict", "pier path is reserved by project member "+member)
		}
	}
	return reserveStandalonePierPath(q, id, path)
}

func reserveStandalonePierPath(q querier, id, path string) error {
	path = pierstore.MountPath(path)
	all, err := instances(q)
	if err != nil {
		return err
	}
	for _, v := range all {
		if v.Public.ID == id {
			continue
		}
		paths := []string{v.Runtime.MoveTarget}
		if v.Runtime.Storage != nil {
			paths = append(paths, v.Runtime.Storage.Path)
		}
		for _, other := range paths {
			if other != "" {
				other = pierstore.MountPath(other)
			}
			if other != "" && (other == path || strings.HasPrefix(path, other+"/") || strings.HasPrefix(other, path+"/")) {
				return conflict("pier_path_conflict", "pier path is reserved by "+v.Public.ID)
			}
		}
	}
	return nil
}

func (r *Server) pierCatalogue(ctx context.Context) *pierstore.Catalogue {
	b, ok := r.b.(storageBackend)
	if !ok {
		return nil
	}
	c := &pierstore.Catalogue{RunnerID: r.s.runnerID, Root: rootless.HomeDir + "/piers", ObservedAt: time.Now().UTC(), Candidates: []pierstore.Candidate{}}
	if b, ok := r.b.(storageActionsBackend); ok {
		var err error
		c.Roots, err = b.StorageRoots()
		if err != nil {
			c.Error = "Storage location registry needs review."
			return c
		}
		if len(c.Roots) > 0 {
			c.Root = c.Roots[0].Path
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	sources, err := b.PierSources(ctx)
	if err != nil {
		c.Error = "Pier discovery could not complete. Check storage ownership, permissions and directory limits."
		return c
	}
	all, err := instances(r.s.db)
	if err != nil {
		c.Error = "Enrollment inventory unavailable."
		return c
	}
	for i := range sources {
		for _, v := range all {
			if s := v.Runtime.Storage; s != nil {
				if s.Identity.Inode != 0 && s.Identity.SameObject(sources[i].Identity) && sources[i].Fingerprint != "" {
					sources[i].EnrolledBy = v.Public.ID
					sources[i].Moved = s.Path != sources[i].Path || s.Uncertain || !s.MountIdentity.SameObject(sources[i].MountIdentity)
					break
				}
				if s.Path == sources[i].Path {
					sources[i].EnrolledBy = v.Public.ID
					if s.Identity.Inode != 0 && !s.Identity.SameObject(sources[i].Identity) {
						sources[i].Problem = "A different directory replaced this registered pier. Review it before management can resume."
					}
				}
			}
		}
	}
	projects, err := projectStorageLocations(r.s.db)
	if err != nil {
		c.Error = "Project storage inventory needs review."
		return c
	}
	for i := range sources {
		for id, loc := range projects {
			if loc.Path == sources[i].Path {
				sources[i].EnrolledBy = id
				if !loc.MountIdentity.SameObject(sources[i].MountIdentity) || loc.Identity.Inode != 0 && !loc.Identity.SameObject(sources[i].Identity) {
					sources[i].Problem = "Project pier identity changed; review its storage."
				}
			}
		}
	}
	c.Candidates = sources
	return c
}

func (r *Server) servePiers(w http.ResponseWriter, req *http.Request, actor, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "pier enrollment requires operator access"}})
		return
	}
	if req.Method != http.MethodPost {
		r.respondError(w, bad("use snapshot discovery and POST to enroll a selected pier"))
		return
	}
	var body protocol.EnrollPierRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if !protocol.ValidID(body.ID) || !protocol.ValidID(body.RequestID) || !protocol.ValidFakeShip(body.Ship) || !pierstore.ValidPath(body.Path) || !epochs.ValidDigest(body.Fingerprint) || !body.ConfirmStopped {
		r.respondError(w, bad("enrollment requires a convention path, fresh fingerprint, expected ~ship and stopped-source confirmation"))
		return
	}
	if kind := pierstore.Kind(body.Path); kind != "fake" && kind != protocol.ShipClass(body.Ship) {
		r.respondError(w, bad("livenet directory classification must match the expected ship identity"))
		return
	}
	fingerprint, _ := json.Marshal(struct {
		Kind    string
		Request protocol.EnrollPierRequest
	}{"enroll-pier", body})
	if op, hit, err := replay(r.s.db, actor, body.RequestID, string(fingerprint)); hit || err != nil {
		if err != nil {
			r.respondError(w, err)
		} else {
			respond(w, 200, op)
		}
		return
	}
	catalogue := r.pierCatalogue(req.Context())
	if catalogue == nil || catalogue.Error != "" {
		r.respondError(w, conflict("pier_unavailable", "pier discovery unavailable"))
		return
	}
	for _, source := range catalogue.Candidates {
		if source.Path != body.Path {
			continue
		}
		if source.Fingerprint != body.Fingerprint || source.Busy || source.Problem != "" {
			r.respondError(w, conflict("pier_changed_or_busy", "pier changed, has a Vere lock, or needs storage review"))
			return
		}
		if source.EnrolledBy != "" && !body.Relink {
			r.respondError(w, conflict("pier_already_enrolled", "reconnect the existing registration instead of enrolling the same pier twice"))
			return
		}
		op, err := r.s.enrollPier(actor, body, source, string(fingerprint))
		if err != nil {
			r.respondError(w, err)
			return
		}
		if !body.Relink {
			r.launch(op)
			respond(w, 202, op)
		} else {
			respond(w, 200, op)
		}
		return
	}
	r.respondError(w, conflict("pier_not_found", "pier is no longer in the discovery tree"))
}

func (s *store) enrollPier(actor string, req protocol.EnrollPierRequest, source pierstore.Candidate, fingerprint string) (protocol.Operation, error) {
	return s.enrollPierSource(actor, req, source, fingerprint, nil)
}

func (s *store) enrollPierSource(actor string, req protocol.EnrollPierRequest, source pierstore.Candidate, fingerprint string, external *pierstore.ExternalSource, identify ...bool) (protocol.Operation, error) {
	if err := s.checkStorage(req.Path); err != nil {
		return protocol.Operation{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, err
	}
	defer tx.Rollback()
	var v storedInstance
	if req.Relink {
		v, err = loadInstance(tx, req.ID)
		if err != nil {
			return protocol.Operation{}, err
		}
		if v.Public.ConfigRevision != req.ExpectedRevision || v.Public.ActiveOperationID != "" || v.Public.DesiredState != "stopped" || v.Runtime.Storage == nil || !v.Runtime.Storage.Identity.SameObject(source.Identity) || v.Public.Ship != req.Ship {
			return protocol.Operation{}, conflict("relink_conflict", "stop the same registered pier before reconnecting its moved directory")
		}
		if v.Public.Mode != source.Kind {
			return protocol.Operation{}, bad("keep a moved pier in the same fake/livenet classification")
		}
		review := v.Runtime
		location := *review.Storage
		location.Uncertain = false
		review.Storage = &location
		if review.UncertainMaintenance() || review.UncertainProvisioning() {
			return protocol.Operation{}, conflict("recovery_required", "independent provisioning or maintenance uncertainty requires review")
		}
		for _, c := range v.Runtime.Runtime.Containers {
			if !c.Removed || c.CreateUncertain {
				return protocol.Operation{}, conflict("invalid_state", "stop retained containers before reconnecting storage")
			}
		}
	} else {
		revision, err := initialRevision(tx, req.ID)
		if err != nil {
			return protocol.Operation{}, err
		}
		if source.Kind != "fake" && s.mode != DevelopmentMode {
			return protocol.Operation{}, conflict("comet_mode_required", "livenet imports require development-managed mode")
		}
		base := protocol.CreateFakeRequest{ID: req.ID, Label: req.Label, RequestID: req.RequestID, Config: protocol.Configuration{HTTPPort: req.HTTPPort, AmesPort: 34343, Loom: 31}}
		if _, err := registrationFingerprint(&base, nil); err != nil {
			return protocol.Operation{}, err
		}
		var count int
		if err := tx.QueryRow("SELECT count(*) FROM instances WHERE id=?", req.ID).Scan(&count); err != nil {
			return protocol.Operation{}, err
		}
		if count != 0 {
			return protocol.Operation{}, conflict("instance_exists", "instance ID is already registered")
		}
		if err := reservePort(tx, req.ID, req.HTTPPort); err != nil {
			return protocol.Operation{}, err
		}
		if err := s.availablePort("tcp", req.HTTPPort); err != nil {
			return protocol.Operation{}, err
		}
		id, err := randomID()
		if err != nil {
			return protocol.Operation{}, err
		}
		v = storedInstance{Public: protocol.Instance{ID: req.ID, Label: base.Label, Ship: req.Ship, Mode: source.Kind, Config: base.Config, ConfigRevision: revision, DesiredState: "stopped", ObservedState: "enrolling", Health: "unknown"}, Runtime: runtimequalify.ManagedState{ID: id, Port: req.HTTPPort, Adoption: &runtimequalify.ManagedAdoption{Fingerprint: req.Fingerprint, Uncertain: true}}}
		if source.Kind != "fake" {
			ames, err := s.choosePort(tx, req.ID, "udp", 0)
			if err != nil {
				return protocol.Operation{}, err
			}
			v.Public.Config.AmesPort, v.Runtime.AmesPort = ames, ames
			v.Public.AmesPublished = true
			// As with an incoming handoff, opt into host Ames publication.
			// Enrollment initializes an existing pier without a key/bootstrap;
			// comet-verify still disables Ames until its identity matches.
			v.Runtime.Boot = &runtimequalify.ManagedBoot{}
			v.Runtime.NativeShip = req.Ship
			v.Runtime.Comet = &runtimequalify.ManagedComet{Ship: req.Ship}
		} else {
			v.Runtime.Fake = &runtimequalify.ManagedFake{Ship: req.Ship}
		}
	}
	if err := reservePierPath(tx, req.ID, req.Path); err != nil {
		return protocol.Operation{}, err
	}
	all, err := instances(tx)
	if err != nil {
		return protocol.Operation{}, err
	}
	for _, other := range all {
		if other.Runtime.ExternalImport != nil && other.Runtime.ExternalImport.Stage != "complete" && other.Runtime.ExternalImport.Source.Identity.SameObject(source.Identity) && source.Identity.Inode != 0 {
			return protocol.Operation{}, conflict("source_reserved", "this source is reserved by an import")
		}
		if external != nil && other.Runtime.ExternalImport != nil && other.Runtime.ExternalImport.Stage != "complete" && pierstore.Overlap(other.Runtime.ExternalImport.Source.Path, external.Path) {
			return protocol.Operation{}, conflict("source_reserved", "source is reserved by another import")
		}
		if other.Public.ID != req.ID && other.Runtime.Storage != nil && other.Runtime.Storage.Identity.SameObject(source.Identity) && source.Identity.Inode != 0 {
			return protocol.Operation{}, conflict("pier_already_enrolled", "this filesystem pier already belongs to another registration")
		}
	}
	if source.Kind != "fake" {
		all, err := instances(tx)
		if err != nil {
			return protocol.Operation{}, err
		}
		for _, other := range all {
			if other.Public.ID != req.ID && other.Public.Mode != "fake" && other.Public.Ship == req.Ship {
				return protocol.Operation{}, conflict("ship_already_enrolled", "this livenet identity is already registered")
			}
		}
	}
	v.Runtime.Storage = &pierstore.Location{Path: req.Path, Identity: source.Identity, MountIdentity: source.MountIdentity}
	v.Runtime.MoveTarget = ""
	v.Runtime.Epochs = nil
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, err
	}
	now := time.Now().UTC()
	op := protocol.Operation{ID: "op-" + id, RequestID: req.RequestID, InstanceID: req.ID, Kind: "enroll-pier", Actor: actor, Status: "queued", ConfigRevision: v.Public.ConfigRevision, CreatedAt: now}
	if external != nil {
		op.Kind = "import-pier"
		v.Public.ObservedState = "importing"
		v.Runtime.ExternalImport = &runtimequalify.ManagedExternalImport{Source: *external, Stage: "reviewed"}
		v.Runtime.ExternalImport.Identify = len(identify) > 0 && identify[0]
		v.Runtime.ExternalImport.Events = []protocol.ImportEvent{{Stage: "reviewed", At: now}}
		v.Runtime.Storage.Identity = pierstore.Identity{}
		v.Runtime.Storage.MountIdentity = pierstore.DirectoryIdentity{}
	}
	v.Public.PierPath = req.Path
	v.Public.IdentityPending = v.Runtime.Adoption != nil && !v.Runtime.Adoption.Verified
	v.Public.ActiveOperationID = op.ID
	if req.Relink {
		op.Kind = "relink-pier"
		op.Status = "succeeded"
		op.FinishedAt = &now
		v.Public.ActiveOperationID = ""
		applyObservation(&v, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, now)
	}
	v.RuntimeRevision++
	if !req.Relink {
		data, err := json.Marshal(v)
		if err != nil {
			return op, err
		}
		if _, err := tx.Exec("INSERT INTO instances VALUES(?,?)", req.ID, data); err != nil {
			return op, err
		}
	} else if err := saveInstance(tx, v); err != nil {
		return op, err
	}
	if err := insertOperation(tx, op, fingerprint); err != nil {
		return op, err
	}
	return op, tx.Commit()
}
