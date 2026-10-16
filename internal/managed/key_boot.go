package managed

import (
	"crypto/sha256"
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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func (r *Server) serveCreateShip(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != http.MethodPost || role != "operator" || r.s.mode != DevelopmentMode {
		r.respondError(w, bad("key boot requires operator access on development-managed"))
		return
	}
	var body protocol.CreateShipRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	op, replayed, err := r.s.registerShip(actor, body)
	body.Key = ""
	if err != nil {
		r.respondError(w, err)
		return
	}
	status := 201
	if replayed {
		status = 200
	}
	respond(w, status, op)
}

func (s *store) registerShip(actor string, req protocol.CreateShipRequest) (protocol.Operation, bool, error) {
	req.Key = strings.TrimSpace(req.Key)
	req.Ship = "~" + strings.TrimPrefix(strings.TrimSpace(req.Ship), "~")
	if message := protocol.ShipNameProblem(req.Ship); message != "" {
		return protocol.Operation{}, false, bad(message)
	}
	class := protocol.ShipClass(req.Ship)
	name := strings.TrimPrefix(req.Ship, "~")
	if req.ID == "" {
		req.ID = name
	}
	if req.PierPath == "" {
		req.PierPath = "livenet/" + map[string]string{"galaxy": "galaxies", "star": "stars", "planet": "planets", "moon": "moons"}[class] + "/" + name + "/" + name
	}
	if req.Config.Loom == 0 {
		req.Config.Loom = 31
	}
	base := "livenet/" + map[string]string{"galaxy": "galaxies", "star": "stars", "planet": "planets", "moon": "moons"}[class] + "/" + name + "/"
	if req.ID != name || !pierstore.ValidPath(req.PierPath) || (pierstore.Relative(req.PierPath) != base+name && pierstore.Relative(req.PierPath) != base+"pier") {
		return protocol.Operation{}, false, bad("instance name and pier path are derived from the Urbit identity")
	}
	if req.Pill != nil && !protocol.ValidBootPill(req.Pill) {
		return protocol.Operation{}, false, bad("invalid selected pill descriptor")
	}
	if err := s.checkStorage(req.PierPath); err != nil {
		return protocol.Operation{}, false, err
	}
	if class == "" || class == "comet" || !req.ConfirmNetwork || !protocol.ValidBootKey(req.Key) || pierstore.Kind(req.PierPath) != class {
		return protocol.Operation{}, false, bad("require a valid owned ~ship, private 0w key, matching livenet directory and network acknowledgement")
	}
	registration := protocol.CreateFakeRequest{ID: req.ID, Label: req.Label, Config: protocol.Configuration{HTTPPort: 18080, AmesPort: 34343, Loom: req.Config.Loom}, RequestID: req.RequestID}
	if _, err := registrationFingerprint(&registration, nil); err != nil {
		return protocol.Operation{}, false, err
	}
	req.Label = registration.Label
	key := req.Key
	req.Key = ""
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	fingerprint, _ := json.Marshal(struct {
		Kind      string
		Request   protocol.CreateShipRequest
		KeyDigest string
	}{"register-ship", req, digest})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, true, err
	}
	revision, err := initialRevision(tx, req.ID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	req.Config.HTTPPort, err = s.choosePort(tx, req.ID, "tcp", req.Config.HTTPPort)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	req.Config.AmesPort, err = s.choosePort(tx, req.ID, "udp", req.Config.AmesPort)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if err := s.checkStagedPill(req.Pill); err != nil {
		return protocol.Operation{}, false, err
	}
	all, err := instances(tx)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	for _, v := range all {
		if v.Public.ID == req.ID {
			return protocol.Operation{}, false, conflict("instance_exists", "instance ID already registered")
		}
		if v.Public.Mode != "fake" && v.Public.Ship == req.Ship {
			return protocol.Operation{}, false, conflict("ship_already_enrolled", "livenet identity already registered")
		}
	}
	if err := reservePort(tx, req.ID, req.Config.HTTPPort); err != nil {
		return protocol.Operation{}, false, err
	}
	if err := reservePierPath(tx, req.ID, req.PierPath); err != nil {
		return protocol.Operation{}, false, err
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	v := storedInstance{Public: protocol.Instance{ID: req.ID, Label: req.Label, Config: req.Config, ConfigRevision: revision, Ship: req.Ship, Mode: class, PierPath: req.PierPath, DesiredState: "stopped", ObservedState: "uninitialized", Health: "unknown"}, Runtime: runtimequalify.ManagedState{ID: id, Port: req.Config.HTTPPort, AmesPort: req.Config.AmesPort, Boot: &runtimequalify.ManagedBoot{Pill: req.Pill}, NativeShip: req.Ship, NativeKeyDigest: digest, Comet: &runtimequalify.ManagedComet{Ship: req.Ship}, Storage: &pierstore.Location{Path: req.PierPath}}}
	v.Public.AmesPublished = true
	v.Public.Bootstrap = &protocol.Bootstrap{Kind: "bare", Pill: req.Pill}
	if req.Pill != nil {
		v.Public.Bootstrap.Kind = "recommended"
	}
	data, err := json.Marshal(v)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err := tx.Exec("INSERT INTO instances VALUES(?,?)", req.ID, data); err != nil {
		return protocol.Operation{}, false, err
	}
	opID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	now := time.Now().UTC()
	op := protocol.Operation{ID: "op-" + opID, RequestID: req.RequestID, InstanceID: req.ID, Kind: "register-ship", Actor: actor, Status: "succeeded", ConfigRevision: revision, CreatedAt: now, FinishedAt: &now}
	if err := insertOperation(tx, op, string(fingerprint)); err != nil {
		return op, false, err
	}
	// Only the private boot file contains key material. A failed DB commit may
	// leave an orphaned file, but cannot authorize boot or reuse its random ID.
	if err := stageBootKey(s.directory, id, key); err != nil {
		return op, false, conflict("key_staging_failed", "private boot-key staging failed; no registration committed")
	}
	key = ""
	return op, false, tx.Commit()
}

func stageBootKey(base, id, key string) error {
	dir := filepath.Join(base, "keys")
	if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	i, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	st, ok := i.Sys().(*syscall.Stat_t)
	if !ok || !i.IsDir() || i.Mode().Perm() != 0700 || st.Uid != uint32(os.Geteuid()) {
		return errors.New("unsafe key directory")
	}
	f, err := os.OpenFile(filepath.Join(dir, id+".key"), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(f, key)
	err = errors.Join(writeErr, f.Sync(), f.Close())
	if err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
