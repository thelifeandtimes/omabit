package managed

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func handoffOperation(tx *sql.Tx, v *storedInstance, actor, kind, request string) (protocol.Operation, error) {
	fingerprint := "handoff:" + v.Runtime.Handoff.ID + ":" + kind
	if old, hit, err := replay(tx, actor, request, fingerprint); hit || err != nil {
		if err != nil {
			return old, err
		}
		return old, conflict("handoff_request_finished", "this attempt already has a recorded outcome; use a new request ID for an explicit retry")
	}
	key, err := randomID()
	if err != nil {
		return protocol.Operation{}, err
	}
	op := protocol.Operation{ID: "op-" + key, InstanceID: v.Public.ID, Actor: actor, Kind: kind, RequestID: request, ConfigRevision: v.Public.ConfigRevision, Status: "queued", CreatedAt: time.Now().UTC()}
	v.Public.ActiveOperationID = op.ID
	v.Public.ObservedState = "moving"
	v.Public.Health = "unknown"
	v.Public.DesiredState = "stopped"
	v.Public.Handoff = v.Runtime.Handoff
	v.Runtime.Handoff.Error = ""
	v.RuntimeRevision++
	if err = saveInstance(tx, *v); err != nil {
		return op, err
	}
	return op, insertOperation(tx, op, fingerprint)
}

// Synchronous handoff mutations keep their exact acceptance response beside the
// ordinary actor-scoped operation. Reservation RPC replay still reports current
// transfer progress for existing coordinators; the audit receipt never changes.
// Completed receipts are never worker inputs. Keeping this private extension out
// of Handoff preserves historical signature bytes and the public protocol; the
// schema20 fence prevents older writers losing it.
type handoffReceipt struct {
	Status  int              `json:"status"`
	Handoff protocol.Handoff `json:"handoff"`
}

type handoffReceiptOperation struct {
	protocol.Operation
	Receipt *handoffReceipt `json:"handoff_receipt,omitempty"`
}

func handoffFingerprint(kind, id string, request any) string {
	data, _ := json.Marshal([]any{kind, id, request})
	return fmt.Sprintf("handoff-receipt-v1:%x", sha256.Sum256(data))
}

func replayHandoff(tx *sql.Tx, actor, request, fingerprint string) (*handoffReceipt, error) {
	op, hit, err := replay(tx, actor, request, fingerprint)
	if err != nil || !hit {
		return nil, err
	}
	var data []byte
	if err = tx.QueryRow("SELECT body FROM operations WHERE id=?", op.ID).Scan(&data); err != nil {
		return nil, err
	}
	var stored handoffReceiptOperation
	if json.Unmarshal(data, &stored) != nil || stored.Receipt == nil || stored.Status != "succeeded" || (stored.Receipt.Status != 200 && stored.Receipt.Status != 201) {
		return nil, errors.New("invalid stored handoff receipt")
	}
	return stored.Receipt, nil
}

func recordHandoffReceipt(tx *sql.Tx, v storedInstance, actor, kind, request, fingerprint string, status int) error {
	key, err := randomID()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	op := protocol.Operation{ID: "op-" + key, InstanceID: v.Public.ID, Actor: actor, Kind: kind, RequestID: request, ConfigRevision: v.Public.ConfigRevision, Status: "succeeded", CreatedAt: now, FinishedAt: &now}
	if err = insertOperation(tx, op, fingerprint); err != nil {
		return err
	}
	data, err := json.Marshal(handoffReceiptOperation{Operation: op, Receipt: &handoffReceipt{Status: status, Handoff: *v.Runtime.Handoff}})
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE operations SET body=? WHERE id=?", data, op.ID)
	return err
}

func (r *Server) prepareHandoff(w http.ResponseWriter, req *http.Request, actor, role, id string) {
	if role != "operator" {
		r.respondError(w, bad("host moves require operator access"))
		return
	}
	var body protocol.PrepareHandoffRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if !protocol.ValidID(body.RequestID) || !protocol.ValidID(body.DestinationRunner) || !protocol.ValidPierName(body.DestinationInstance) || body.DestinationRunner == r.s.runnerID {
		r.respondError(w, bad("choose another pinned runner and destination instance"))
		return
	}
	tx, err := r.s.db.Begin()
	if err != nil {
		r.respondError(w, err)
		return
	}
	defer tx.Rollback()
	v, err := loadInstance(tx, id)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if v.Public.ConfigRevision != body.ExpectedRevision {
		r.respondError(w, conflict("revision_conflict", "source configuration changed"))
		return
	}
	settings := protocol.RuntimeSettings{Loom: v.Public.Config.Loom, NativeControl: v.Public.Config.NativeControl}
	if settings.Nondefault() && !body.DestinationRuntimeSettings {
		r.respondError(w, conflict("runtime_settings_required", "update the collector and destination for runtime-settings-handoff before moving this ship"))
		return
	}
	if h := v.Runtime.Handoff; h != nil && h.Direction == "out" {
		if settings.Nondefault() && h.RuntimeSettings == nil {
			r.respondError(w, conflict("handoff_settings_missing", "historical handoff lacks the selected runtime settings; retain the source fence for review"))
			return
		}
		if h.DestinationRunner != body.DestinationRunner || h.DestinationInstance != body.DestinationInstance {
			r.respondError(w, conflict("handoff_fenced", "source is already fenced for another move"))
			return
		}
		if h.State != "preparing" || v.Public.ActiveOperationID != "" || h.Error == "" {
			respond(w, 200, h)
			return
		}
		op, e := handoffOperation(tx, &v, actor, "prepare-handoff", body.RequestID)
		if e == nil {
			e = tx.Commit()
		}
		if e != nil {
			r.respondError(w, e)
			return
		}
		r.launch(op)
		respond(w, 202, h)
		return
	}
	if v.Public.ActiveOperationID != "" || v.Public.DesiredState != "stopped" || v.Public.ObservedState != "stopped" || v.Public.RecoveryRequired || !v.Runtime.Initialized || v.Runtime.UncertainMaintenance() || v.Runtime.UncertainProvisioning() {
		r.respondError(w, conflict("invalid_state", "source must be initialized and cleanly stopped before handoff"))
		return
	}
	for _, c := range v.Runtime.Runtime.Containers {
		if !c.Removed || c.CreateUncertain {
			r.respondError(w, conflict("invalid_state", "remove stopped runtime containers before handoff"))
			return
		}
	}
	key, err := randomID()
	if err != nil {
		r.respondError(w, err)
		return
	}
	v.Runtime.Handoff = &protocol.Handoff{ID: key, Direction: "out", State: "preparing", SourceRunner: r.s.runnerID, SourceInstance: id, DestinationRunner: body.DestinationRunner, DestinationInstance: body.DestinationInstance, Ship: v.Public.Ship, Mode: v.Public.Mode, AmesPublished: v.Public.AmesPublished, CreatedAt: time.Now().UTC(), Message: "Source fenced; preparing stopped transfer."}
	v.Runtime.Handoff.RollNeedsRuntime = v.Runtime.RollNeedsRuntime
	if settings.Nondefault() {
		v.Runtime.Handoff.RuntimeSettings = &settings
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		r.respondError(w, err)
		return
	}
	v.Runtime.Handoff.VerifyKey = fmt.Sprintf("%x", public)
	v.Runtime.HandoffSigningKey = private
	op, err := handoffOperation(tx, &v, actor, "prepare-handoff", body.RequestID)
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	r.launch(op)
	respond(w, 202, v.Runtime.Handoff)
}

func (r *Server) receiveHandoff(w http.ResponseWriter, req *http.Request, actor, role string) {
	if role != "operator" {
		r.respondError(w, bad("host moves require operator access"))
		return
	}
	var body protocol.ReceiveHandoffRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if !protocol.ValidID(body.RequestID) {
		r.respondError(w, bad("valid handoff request ID required"))
		return
	}
	tx, err := r.s.db.Begin()
	if err != nil {
		r.respondError(w, err)
		return
	}
	defer tx.Rollback()
	fingerprint := handoffFingerprint("reserve-handoff", "", body)
	if receipt, err := replayHandoff(tx, actor, body.RequestID, fingerprint); err != nil {
		r.respondError(w, err)
		return
	} else if receipt != nil {
		// Reservation is also the historical continuation/status endpoint. An
		// older collector trusts its phase and would retransmit an already
		// received archive if we returned the frozen 'reserved' audit response.
		// Resolve only the exact accepted transfer, never a reused name or ID.
		v, err := loadHandoff(tx, receipt.Handoff.ID)
		if err != nil {
			r.respondError(w, err)
			return
		}
		h := v.Runtime.Handoff
		if v.Public.ID != receipt.Handoff.DestinationInstance || h.Direction != "in" || !h.SameTransfer(receipt.Handoff) {
			r.respondError(w, conflict("handoff_conflict", "accepted handoff no longer matches the destination transfer"))
			return
		}
		respond(w, 200, h)
		return
	}
	h := body.Offer
	settings := protocol.RuntimeSettings{Loom: protocol.DefaultLoom}
	if h.RuntimeSettings != nil {
		settings = *h.RuntimeSettings
		if !protocol.ValidRecordedLoom(settings.Loom) {
			r.respondError(w, bad("invalid handoff runtime settings"))
			return
		}
	}
	if !fullHandoffID(h.ID) || !protocol.ValidID(body.RequestID) || h.DestinationRunner != r.s.runnerID || h.SourceRunner == r.s.runnerID || !protocol.ValidID(h.SourceRunner) || !protocol.ValidID(h.SourceInstance) || !protocol.ValidPierName(h.DestinationInstance) || !protocol.ValidFakeShip(h.Ship) || !validHandoffBytes(h) {
		r.respondError(w, bad("invalid pinned handoff offer"))
		return
	}
	if !fullContainerID.MatchString(h.VerifyKey) || h.Commitment != "" || h.Mode == "fake" && h.AmesPublished || h.Mode != "fake" && (h.Mode != protocol.ShipClass(h.Ship) || r.s.mode != DevelopmentMode) {
		r.respondError(w, bad("handoff ship classification mismatch"))
		return
	}
	all, err := instances(tx)
	if err != nil {
		r.respondError(w, err)
		return
	}
	revision, err := initialRevision(tx, h.DestinationInstance)
	if err != nil {
		r.respondError(w, err)
		return
	}
	for _, v := range all {
		if old := v.Runtime.Handoff; old != nil && old.ID == h.ID {
			if !old.SameTransfer(h) || body.HTTPPort != 0 && body.HTTPPort != v.Public.Config.HTTPPort || h.AmesPublished && body.AmesPort != 0 && body.AmesPort != v.Public.Config.AmesPort {
				r.respondError(w, conflict("handoff_conflict", "handoff offer changed"))
				return
			}
			// Legacy transfers have no reservation receipt. Record this actor's
			// new request without rewriting the transfer or its signed fields.
			if err = recordHandoffReceipt(tx, v, actor, "reserve-handoff", body.RequestID, fingerprint, 200); err == nil {
				err = tx.Commit()
			}
			if err != nil {
				r.respondError(w, err)
				return
			}
			respond(w, 200, old)
			return
		}
		if v.Public.ID == h.DestinationInstance || v.Public.Mode != "fake" && v.Public.Ship == h.Ship {
			r.respondError(w, conflict("instance_exists", "destination already has this instance or live identity"))
			return
		}
	}
	if h.Direction != "out" || h.State != "ready" {
		r.respondError(w, bad("source has not prepared a stopped transfer"))
		return
	}
	httpPort, err := r.s.choosePort(tx, h.DestinationInstance, "tcp", body.HTTPPort)
	if err != nil {
		r.respondError(w, err)
		return
	}
	ames := uint16(34343)
	if h.AmesPublished {
		ames, err = r.s.choosePort(tx, h.DestinationInstance, "udp", body.AmesPort)
		if err != nil {
			r.respondError(w, err)
			return
		}
	}
	tier := map[string]string{"galaxy": "galaxies", "star": "stars", "planet": "planets", "moon": "moons", "comet": "comets"}[h.Mode]
	leaf := strings.TrimPrefix(h.Ship, "~")
	path := "fakeships/" + h.DestinationInstance + "/" + leaf
	if h.Mode != "fake" {
		path = "livenet/" + tier + "/" + h.DestinationInstance + "/" + leaf
	}
	if err = reservePierPath(tx, h.DestinationInstance, path); err != nil {
		r.respondError(w, err)
		return
	}
	key, err := randomID()
	if err != nil {
		r.respondError(w, err)
		return
	}
	h.Direction = "in"
	h.State = "reserved"
	h.Message = "Destination reserved; waiting for stopped pier data."
	v := storedInstance{Public: protocol.Instance{ID: h.DestinationInstance, Label: h.DestinationInstance, Ship: h.Ship, Mode: h.Mode, AmesPublished: h.AmesPublished, PierPath: path, PierName: leaf, Config: protocol.Configuration{HTTPPort: httpPort, AmesPort: ames, Loom: settings.Loom, NativeControl: settings.NativeControl}, ConfigRevision: revision, DesiredState: "stopped", ObservedState: "moving", Health: "unknown", Handoff: &h}, Runtime: runtimequalify.ManagedState{ID: key, Port: httpPort, AmesPort: ames, Loom: settings.Loom, NativeControl: settings.NativeControl, Storage: &pierstore.Location{Path: path}, Handoff: &h}}
	if h.Mode == "fake" {
		v.Runtime.Fake = &runtimequalify.ManagedFake{Ship: h.Ship}
	} else {
		v.Runtime.Comet = &runtimequalify.ManagedComet{Ship: h.Ship}
		if h.Mode != "comet" {
			v.Runtime.NativeShip = h.Ship
		}
		if h.AmesPublished {
			v.Runtime.Boot = &runtimequalify.ManagedBoot{}
			v.Runtime.NativeShip = h.Ship
		}
	}
	v.Runtime.RollNeedsRuntime = h.RollNeedsRuntime
	data, _ := json.Marshal(v)
	if _, err = tx.Exec("INSERT INTO instances VALUES(?,?)", v.Public.ID, data); err == nil {
		err = recordHandoffReceipt(tx, v, actor, "reserve-handoff", body.RequestID, fingerprint, 201)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 201, h)
}

func fullHandoffID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
func validHandoffBytes(h protocol.Handoff) bool {
	return fullContainerID.MatchString(h.SHA256) && fullContainerID.MatchString(h.TreeDigest) && h.Bytes > 0 && h.Bytes <= protocol.MaxHandoffBytes
}
func (s *store) handoff(id string) (storedInstance, error) {
	return loadHandoff(s.db, id)
}

func loadHandoff(q querier, id string) (storedInstance, error) {
	all, err := instances(q)
	if err != nil {
		return storedInstance{}, err
	}
	for _, v := range all {
		if v.Runtime.Handoff != nil && v.Runtime.Handoff.ID == id {
			return v, nil
		}
	}
	return storedInstance{}, &fault{404, protocol.APIError{Code: "handoff_not_found", Message: "handoff not found"}}
}

func (r *Server) handoffControl(w http.ResponseWriter, req *http.Request, actor, role, id, verb string) {
	if role != "operator" {
		r.respondError(w, bad("host moves require operator access"))
		return
	}
	if req.Method == "POST" && verb == "activate" {
		r.activateHandoff(w, req, actor, id)
		return
	}
	v, err := r.s.handoff(id)
	if err != nil {
		r.respondError(w, err)
		return
	}
	h := v.Runtime.Handoff
	if req.Method == "GET" && verb == "" {
		respond(w, 200, h)
		return
	}
	var body protocol.FinishHandoffRequest
	if err = decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if req.Method != "POST" || !protocol.ValidID(body.RequestID) || body.SHA256 != h.SHA256 || body.ExpectedRevision != v.Public.ConfigRevision {
		r.respondError(w, bad("handoff commitment must match its revision and digest"))
		return
	}
	if verb == "commit" && h.Direction == "out" || verb == "retry" && h.Direction == "in" && h.State == "receiving" {
		if h.State == "moved" || v.Public.ActiveOperationID != "" {
			respond(w, 200, h)
			return
		}
		if h.State != "ready" && !(verb == "retry" && h.Error != "") {
			r.respondError(w, conflict("invalid_state", "source transfer is not ready"))
			return
		}
		tx, err := r.s.db.Begin()
		if err != nil {
			r.respondError(w, err)
			return
		}
		defer tx.Rollback()
		kind := "commit-handoff"
		if verb == "retry" {
			kind = "receive-handoff"
		}
		op, err := handoffOperation(tx, &v, actor, kind, body.RequestID)
		if err == nil {
			err = tx.Commit()
		}
		if err != nil {
			r.respondError(w, err)
			return
		}
		r.launch(op)
		respond(w, 202, h)
		return
	}
	r.respondError(w, bad("unsupported handoff action"))
}

func (r *Server) activateHandoff(w http.ResponseWriter, req *http.Request, actor, id string) {
	var body protocol.FinishHandoffRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if !protocol.ValidID(body.RequestID) {
		r.respondError(w, bad("valid handoff request ID required"))
		return
	}
	tx, err := r.s.db.Begin()
	if err != nil {
		r.respondError(w, err)
		return
	}
	defer tx.Rollback()
	fingerprint := handoffFingerprint("activate-handoff", id, body)
	// Replay precedes live transfer/revision lookup: later configuration, moves,
	// or retirement must not change the result of an already accepted request.
	if receipt, err := replayHandoff(tx, actor, body.RequestID, fingerprint); err != nil {
		r.respondError(w, err)
		return
	} else if receipt != nil {
		respond(w, receipt.Status, receipt.Handoff)
		return
	}
	v, err := loadHandoff(tx, id)
	if err != nil {
		r.respondError(w, err)
		return
	}
	h := v.Runtime.Handoff
	if h.Direction != "in" || body.SHA256 != h.SHA256 || body.ExpectedRevision != v.Public.ConfigRevision {
		r.respondError(w, bad("handoff commitment must match its revision and digest"))
		return
	}
	if !h.ValidCommitment(body.Commitment) {
		r.respondError(w, conflict("handoff_uncommitted", "destination requires the source's matching signed commitment"))
		return
	}
	if h.State != "complete" {
		if h.State != "received" || !v.Runtime.Initialized || v.Public.ActiveOperationID != "" {
			r.respondError(w, conflict("invalid_state", "destination transfer has not been verified"))
			return
		}
		h.State = "complete"
		h.Commitment = body.Commitment
		h.Error = ""
		h.Message = "Stopped host move complete; explicit Start is now available on this runner."
		v.Public.ObservedState = "stopped"
		v.Public.RecoveryRequired = false
		v.Public.Handoff = h
		v.RuntimeRevision++
		if err = saveInstance(tx, v); err != nil {
			r.respondError(w, err)
			return
		}
	}
	if err = recordHandoffReceipt(tx, v, actor, "activate-handoff", body.RequestID, fingerprint, 200); err == nil {
		err = tx.Commit()
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, h)
}

// ServeHTTP owns bulk-update admission for the entire exchange (including GET
// exports and PUTs waiting on workGate). Keep dispatch through that server gate;
// the local locks below protect transfer lookup and final publication only.
func (r *Server) handoffData(w http.ResponseWriter, req *http.Request, actor, role, id string) {
	if role != "operator" {
		r.respondError(w, bad("handoff data requires operator access"))
		return
	}
	r.mu.Lock()
	if r.closing || r.failed.Load() {
		r.mu.Unlock()
		r.respondError(w, conflict("runner_unavailable", "runner is unavailable"))
		return
	}
	v, err := r.s.handoff(id)
	r.mu.Unlock()
	if err != nil {
		r.respondError(w, err)
		return
	}
	h := v.Runtime.Handoff
	control := http.NewResponseController(w)
	_ = control.SetReadDeadline(time.Now().Add(30 * time.Minute))
	_ = control.SetWriteDeadline(time.Now().Add(30 * time.Minute))
	path := runtimequalify.HandoffArchive(r.s.directory, h)
	if req.Method == "GET" && h.Direction == "out" && h.State == "ready" && validHandoffBytes(*h) {
		f, err := runtimequalify.OpenHandoffArchive(path)
		if err != nil {
			r.respondError(w, conflict("handoff_unavailable", "stopped transfer artifact unavailable"))
			return
		}
		defer f.Close()
		i, err := f.Stat()
		if err != nil || i.Size() != h.Bytes {
			r.respondError(w, bad("transfer artifact size changed"))
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprint(h.Bytes))
		w.Header().Set(protocol.HandoffHashHeader, h.SHA256)
		_, _ = io.CopyN(w, f, h.Bytes)
		return
	}
	if req.Method != "PUT" || h.Direction != "in" || h.State != "reserved" || req.ContentLength != h.Bytes || req.Header.Get(protocol.HandoffHashHeader) != h.SHA256 || req.Header.Get("Content-Type") != "application/octet-stream" {
		r.respondError(w, bad("handoff data must match the reserved destination"))
		return
	}
	// Bulk disk work shares the host admission gate with export/extraction and
	// offline utilities. Reserve enough room for both spool and extracted contents.
	select {
	case r.workGate <- struct{}{}:
		defer func() { <-r.workGate }()
	case <-r.ctx.Done():
		r.respondError(w, conflict("runner_unavailable", "runner is stopping"))
		return
	case <-req.Context().Done():
		return
	}
	free := handoffDiskAvailable
	if b, ok := r.b.(interface{ HandoffDiskAvailable(string, uint64) bool }); ok {
		free = b.HandoffDiskAvailable
	}
	if !free(r.s.directory, 2*uint64(h.Bytes)+(8<<30)) {
		r.respondError(w, conflict("handoff_capacity", "destination needs two transfer sizes plus 8 GiB free disk"))
		return
	}
	dir := filepath.Dir(path)
	if err = os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		r.respondError(w, err)
		return
	}
	if err = bootpill.PrivateDirectory(dir); err != nil {
		r.respondError(w, err)
		return
	}
	f, err := os.CreateTemp(dir, ".incoming-")
	if err != nil {
		r.respondError(w, err)
		return
	}
	defer os.Remove(f.Name())
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, hash), http.MaxBytesReader(w, req.Body, protocol.MaxHandoffBytes))
	req.Body.Close()
	err = errors.Join(copyErr, f.Chmod(0400), f.Sync(), f.Close())
	if err != nil || n != h.Bytes || fmt.Sprintf("%x", hash.Sum(nil)) != h.SHA256 {
		r.respondError(w, bad("handoff data incomplete or digest mismatch"))
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() {
		r.respondError(w, conflict("runner_unavailable", "runner is unavailable"))
		return
	}
	v, err = r.s.handoff(id)
	if err != nil {
		r.respondError(w, err)
		return
	}
	h = v.Runtime.Handoff
	if h.State != "reserved" || v.Public.ActiveOperationID != "" {
		r.respondError(w, conflict("handoff_changed", "destination transfer advanced; read its current status"))
		return
	}
	if err = os.Link(f.Name(), path); err != nil {
		if !errors.Is(err, os.ErrExist) || !runtimequalify.VerifyHandoffArchive(path, h) {
			r.respondError(w, conflict("handoff_uncertain", "existing artifact does not match the reserved transfer"))
			return
		}
	}
	if err = os.Remove(f.Name()); err != nil {
		r.respondError(w, err)
		return
	}
	h.State = "receiving"
	if err = runtimequalify.SyncHandoffDirectory(dir); err != nil {
		r.respondError(w, err)
		return
	}
	h.Message = "Verifying stopped pier contents on destination."
	tx, err := r.s.db.Begin()
	if err != nil {
		r.respondError(w, err)
		return
	}
	defer tx.Rollback()
	op, err := handoffOperation(tx, &v, actor, "receive-handoff", "receive-"+id)
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	r.launch(op)
	respond(w, 202, h)
}

func handoffDiskAvailable(path string, needed uint64) bool {
	var space syscall.Statfs_t
	return syscall.Statfs(path, &space) == nil && space.Bsize > 0 && space.Bavail <= (^uint64(0))/uint64(space.Bsize) && space.Bavail*uint64(space.Bsize) >= needed
}
