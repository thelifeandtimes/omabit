package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type backend interface {
	Apply(context.Context, string, runtimequalify.ManagedState, func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error)
	Observe(context.Context, runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error)
}

type realBackend struct{}

func (realBackend) Apply(ctx context.Context, action string, state runtimequalify.ManagedState, persist func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	return runtimequalify.ApplyManaged(ctx, action, state, persist)
}
func (realBackend) Observe(ctx context.Context, state runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	return runtimequalify.ObserveManaged(ctx, state)
}

// Server serves only the private local management API. Its host/state lock must
// outlive it. Close stops management workers, not the independently running ships.
type Server struct {
	s        *store
	b        backend
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	closing  bool
	workGate chan struct{}
	failed   atomic.Bool
	uid      uint32
	keys     string
	bootID   string
}

func New(dir string) (*Server, error) { return newServer(dir, realBackend{}, 10*time.Second) }

// NewWithAccess enables authentication-derived SSH grants on the private socket.
func NewWithAccess(dir, keys string) (*Server, error) {
	r, err := New(dir)
	if err == nil {
		r.keys = keys
	}
	return r, err
}

func newServer(dir string, b backend, interval time.Duration) (*Server, error) {
	return newServerMode(dir, b, interval, false)
}

func NewDevelopmentWithAccess(dir, keys string) (*Server, error) {
	r, err := newServerMode(dir, realBackend{}, 10*time.Second, true)
	if err == nil {
		r.keys = keys
	}
	return r, err
}

func newServerMode(dir string, b backend, interval time.Duration, comets bool) (*Server, error) {
	return newServerWithStartup(dir, b, interval, comets, "")
}

func newServerWithStartup(dir string, b backend, interval time.Duration, comets bool, bootID string) (*Server, error) {
	s, err := openStore(dir)
	if err != nil {
		return nil, err
	}
	if ports, ok := b.(interface{ PortAvailable(string, uint16) bool }); ok {
		s.portAvailable = ports.PortAvailable
	}
	if storage, ok := b.(storageActionsBackend); ok {
		s.validateStorage = func(path string) error {
			id, _ := pierstore.Split(path)
			if id == "" {
				return nil
			}
			roots, err := storage.StorageRoots()
			if err != nil {
				return conflict("storage_unavailable", err.Error())
			}
			for _, root := range roots {
				if root.ID == id {
					return nil
				}
			}
			return bad("unknown storage location; add it in Storage locations before selecting it")
		}
	}
	if comets {
		s.mode = DevelopmentMode
	} else {
		all, err := instances(s.db)
		if err != nil {
			s.close()
			return nil, err
		}
		for _, v := range all {
			if v.Runtime.Comet != nil {
				s.close()
				return nil, errors.New("registered comets require serve-development; fake-only mode refuses this state")
			}
		}
	}
	if err := s.reconcileInterrupted(); err != nil {
		s.close()
		return nil, err
	}
	var resume []storedInstance
	if bootID != "" {
		if !bootIDPattern.MatchString(bootID) {
			s.close()
			return nil, errors.New("invalid host boot identity")
		}
		s.bootID = bootID
		resume, err = s.startupCandidates()
		if err != nil {
			s.close()
			return nil, err
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Server{s: s, b: b, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1), uid: uint32(os.Geteuid()), bootID: bootID}
	if len(resume) > 0 {
		r.wg.Add(1)
		go r.resumeStartup(resume, interval)
	}
	r.wg.Add(1)
	go r.poll(interval)
	r.wg.Add(1)
	go r.scheduleLoop()
	return r, nil
}

func (r *Server) Close() error {
	r.mu.Lock()
	r.closing = true
	r.cancel()
	r.mu.Unlock()
	r.wg.Wait()
	return r.s.close()
}

func (r *Server) poll(interval time.Duration) {
	defer r.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if r.ctx.Err() != nil {
			return
		}
		all, err := instances(r.s.db)
		if err != nil {
			r.failed.Store(true)
			return
		}
		var batch sync.WaitGroup
		for _, v := range all {
			if v.Public.ActiveOperationID != "" {
				continue
			}
			batch.Add(1)
			go func(v storedInstance) {
				defer batch.Done()
				ctx, cancel := context.WithTimeout(r.ctx, 20*time.Second)
				defer cancel()
				state, observation, err := r.b.Observe(ctx, v.Runtime)
				if r.ctx.Err() != nil {
					return
				}
				if err != nil {
					state = v.Runtime
					observation = runtimequalify.ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
				}
				if err := r.s.observe(v, state, observation); err != nil {
					r.failed.Store(true)
				}
			}(v)
		}
		batch.Wait()
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Server) launch(op protocol.Operation) {
	// Caller holds r.mu across journal acceptance and worker registration.
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if strings.HasSuffix(op.Kind, "-handoff") || op.Kind == "delete-pier" || op.Kind == "import-pier" || op.Kind == "start" || op.Kind == "startup-resume" || op.Kind == "move-pier" || op.Kind == "enroll-pier" || op.Kind == "pack" || op.Kind == "meld" || op.Kind == "roll" || op.Kind == "prune-epochs" || op.Kind == "import-fake" || routineKind(op.Kind) {
			select {
			case r.workGate <- struct{}{}:
				defer func() { <-r.workGate }()
			case <-r.ctx.Done():
				return // restart marks the durably queued job interrupted
			}
		}
		actual, instance, err := r.s.beginOperation(op.ID)
		if err != nil {
			var admission *fault
			if errors.As(err, &admission) && admission.problem.Code == "host_recovery_required" {
				// This is a failed runtime admission, not a broken state store.
				// Finish the queued request without dispatching it; cleanup jobs
				// must remain usable and replay must retain this outcome.
				if err := r.s.finish(actual, instance.Runtime, runtimequalify.ManagedObservation{}, err, false); err != nil {
					r.failed.Store(true)
				}
				return
			}
			r.failed.Store(true)
			return
		}
		if routineKind(actual.Kind) && !r.routineAuthorized(actual.Actor) {
			if err := r.s.finish(actual, instance.Runtime, runtimequalify.ManagedObservation{}, errors.New("routine owner no longer has operator access"), false); err != nil {
				r.failed.Store(true)
			}
			return
		}
		state, observation, err := r.b.Apply(r.ctx, actual.Kind, instance.Runtime, func(state runtimequalify.ManagedState) error {
			if err := r.s.runtimeUpdate(actual.InstanceID, actual.ID, state); err != nil {
				r.failed.Store(true)
				return errors.New("resource intent journal failed; operation stopped")
			}
			return nil
		})
		if err := r.s.finish(actual, state, observation, err, r.ctx.Err() != nil); err != nil {
			r.failed.Store(true)
		}
	}()
}

type peerKey struct{}

func peerActor(ctx context.Context) (uint32, bool) {
	uid, ok := ctx.Value(peerKey{}).(uint32)
	return uid, ok
}

func (r *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
	w.Header().Set("Cache-Control", "no-store")
	uid, ok := peerActor(req.Context())
	if !ok || (uid != r.uid && uid != 0) {
		r.respondError(w, &fault{403, protocol.APIError{Code: "unauthorized_peer", Message: "local management requires an authenticated Unix peer"}})
		return
	}
	if req.Header.Get("Origin") != "" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "origin_not_allowed", Message: "runner RPC does not accept browser origins"}})
		return
	}
	actor := fmt.Sprintf("local-uid:%d", uid)
	role := "operator"
	if values, present := req.Header[http.CanonicalHeaderKey(access.Header)]; present {
		if len(values) != 1 || r.keys == "" {
			r.respondError(w, &fault{403, protocol.APIError{Code: "ssh_access_denied", Message: "SSH grants are not available"}})
			return
		}
		grant, err := access.Load(r.keys, values[0])
		if err != nil || !grant.Allows(req.Method, req.URL.Path) {
			r.respondError(w, &fault{403, protocol.APIError{Code: "ssh_access_denied", Message: "SSH grant does not permit this request"}})
			return
		}
		actor = "ssh-key:" + grant.Fingerprint
		role = grant.Role
	}
	if req.Method == http.MethodGet && req.URL.Path == "/hello" {
		features := []string{"snapshot", "create-fake", "configuration-revisions", "durable-operations", "local-peer-credentials", "offline-pack", "offline-meld", "offline-roll", "container-resources", "container-cpu", "epoch-inventory", "epoch-prune-genesis", "epoch-prune-prefix"}
		features = append(features, "maintenance-schedules")
		features = append(features, "fake-ship-options")
		if p, err := ownerhost.Current(); err == nil && p != nil {
			features = append(features, "owner-user-runtime", "human-pier-storage")
		}
		if _, ok := r.b.(storageActionsBackend); ok {
			features = append(features, "storage-locations", "import-pier", "delete-pier")
		}
		if _, ok := r.b.(storageBackend); ok {
			features = append(features, "bootstrap-selection")
			features = append(features, "pier-storage")
			features = append(features, "host-handoff")
			if r.s.mode == DevelopmentMode {
				features = append(features, "key-boot", "key-boot-options", "livenet-import")
			}
		}
		if r.bootID != "" {
			features = append(features, "startup-resume", "unclean-reboot-resume")
		}
		if r.s.mode == DevelopmentMode {
			features = append(features, "create-comet", "comet-compaction", "comet-epochs", "comet-prune-prefix")
		}
		if _, ok := r.b.(codeBackend); ok {
			features = append(features, "login-code")
		}
		if _, ok := r.b.(logsBackend); ok {
			features = append(features, "runtime-logs")
		}
		if _, ok := r.b.(importBackend); ok {
			features = append(features, "import-qualified-fake")
		}
		if r.keys != "" {
			features = append(features, "ssh-key-grants")
		}
		respond(w, 200, protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: r.s.runnerID, Mode: r.s.mode, Features: features, Actor: actor, AccessRole: role})
		return
	}
	if req.Header.Get(protocol.Header) != strconv.Itoa(protocol.Version) {
		r.respondError(w, &fault{426, protocol.APIError{Code: "protocol_mismatch", Message: "send Urbit-Runner-Protocol: 1"}})
		return
	}
	if req.URL.Path == protocol.PillUploadPath {
		if r.ctx.Err() != nil || r.failed.Load() {
			r.respondError(w, conflict("runner_unavailable", "runner is unavailable"))
			return
		}
		r.stagePill(w, req, role)
		return
	}
	if req.Method == http.MethodGet && req.URL.Path == "/v1/snapshot" {
		snapshot, err := r.s.snapshot()
		if err != nil {
			r.respondError(w, err)
			return
		}
		if r.failed.Load() {
			snapshot.Incidents = append(snapshot.Incidents, protocol.Incident{ID: "state-store", Code: "state_store_error", Message: "state journal failed; mutations are disabled until the store is reviewed", Active: true, ObservedAt: time.Now().UTC()})
		}
		snapshot.Piers = r.pierCatalogue(req.Context())
		respond(w, 200, snapshot)
		return
	}
	if strings.HasPrefix(req.URL.Path, "/v1/handoffs") || strings.HasSuffix(req.URL.Path, "/handoff") {
		if _, ok := r.b.(storageBackend); !ok {
			r.respondError(w, bad("host handoff is unavailable on this runner"))
			return
		}
	}
	if protocol.HandoffDataPath(req.URL.Path) {
		r.handoffData(w, req, actor, role, strings.Split(req.URL.Path, "/")[3])
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() {
		r.respondError(w, &fault{503, protocol.APIError{Code: "runner_unavailable", Message: "runner is stopping or state storage needs review"}})
		return
	}
	if req.Method == "POST" && req.URL.Path == "/v1/handoffs" {
		r.receiveHandoff(w, req, actor, role)
		return
	}
	if strings.HasPrefix(req.URL.Path, "/v1/handoffs/") {
		parts := strings.Split(req.URL.Path, "/")
		if (len(parts) == 4 || len(parts) == 5) && fullHandoffID(parts[3]) {
			verb := ""
			if len(parts) == 5 {
				verb = parts[4]
			}
			r.handoffControl(w, req, actor, role, parts[3], verb)
			return
		}
	}
	if req.URL.Path == "/v1/imports" {
		r.serveImports(w, req, actor, role)
		return
	}
	if req.URL.Path == "/v1/piers" {
		r.servePiers(w, req, actor, role)
		return
	}
	if req.URL.Path == "/v1/storage-roots" || req.URL.Path == "/v1/pier-import-preview" || req.URL.Path == "/v1/pier-import" {
		r.serveStorageActions(w, req, actor, role)
		return
	}
	if req.URL.Path == "/v1/ships" {
		r.serveCreateShip(w, req, actor, role)
		return
	}
	if req.Method == http.MethodPost && req.URL.Path == "/v1/comets" {
		var body protocol.CreateCometRequest
		if err := decode(w, req, &body); err != nil {
			r.respondError(w, err)
			return
		}
		if !body.ConfirmNetwork {
			r.respondError(w, bad("comet registration requires explicit networked-identity acknowledgement"))
			return
		}
		op, replayed, err := r.s.registerKind(actor, body.CreateFakeRequest, nil, true)
		if err != nil {
			r.respondError(w, err)
			return
		}
		status := 201
		if replayed {
			status = 200
		}
		respond(w, status, op)
		return
	}
	if req.Method == http.MethodPost && req.URL.Path == "/v1/instances" {
		var body protocol.CreateFakeRequest
		if err := decode(w, req, &body); err != nil {
			r.respondError(w, err)
			return
		}
		op, replay, err := r.s.register(actor, body)
		if err != nil {
			r.respondError(w, err)
			return
		}
		status := 201
		if replay {
			status = 200
		}
		respond(w, status, op)
		return
	}
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 5 || parts[0] != "" || parts[1] != "v1" || parts[2] != "instances" || !protocol.ValidID(parts[3]) {
		r.respondError(w, &fault{404, protocol.APIError{Code: "not_found", Message: "unknown runner endpoint"}})
		return
	}
	switch {
	case req.Method == "POST" && parts[4] == "handoff":
		r.prepareHandoff(w, req, actor, role, parts[3])
	case req.Method == http.MethodPost && parts[4] == "logs":
		r.serveLogs(w, req, role, parts[3])
	case parts[4] == "routine":
		r.serveRoutine(w, req, actor, role, parts[3])
	case req.Method == http.MethodPost && parts[4] == "login-code":
		r.loginCode(w, req, actor, role, parts[3])
	case req.Method == http.MethodPut && parts[4] == "configuration":
		var body protocol.ConfigureRequest
		if err := decode(w, req, &body); err != nil {
			r.respondError(w, err)
			return
		}
		instance, err := r.s.configure(actor, parts[3], body)
		if err != nil {
			r.respondError(w, err)
			return
		}
		respond(w, 200, instance)
	case req.Method == http.MethodPost && parts[4] == "operations":
		var body protocol.OperationRequest
		if err := decode(w, req, &body); err != nil {
			r.respondError(w, err)
			return
		}
		op, replay, err := r.s.accept(actor, parts[3], body)
		if err != nil {
			r.respondError(w, err)
			return
		}
		status := 200
		if !replay {
			r.launch(op)
			status = 202
		}
		respond(w, status, op)
	default:
		r.respondError(w, &fault{404, protocol.APIError{Code: "not_found", Message: "unknown runner endpoint or method"}})
	}
}

func decode(w http.ResponseWriter, req *http.Request, value any) error {
	media, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return &fault{415, protocol.APIError{Code: "invalid_content_type", Message: "require application/json"}}
	}
	req.Body = http.MaxBytesReader(w, req.Body, 4096)
	defer req.Body.Close()
	d := json.NewDecoder(req.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return bad("request must match the JSON schema and 4096-byte limit")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return bad("require exactly one JSON value")
	}
	return nil
}

func (r *Server) respondError(w http.ResponseWriter, err error) {
	var f *fault
	if errors.As(err, &f) {
		respond(w, f.status, protocol.ErrorResponse{Error: f.problem})
		return
	}
	r.failed.Store(true)
	respond(w, 500, protocol.ErrorResponse{Error: protocol.APIError{Code: "state_store_error", Message: "state journal error; mutations are disabled until reviewed"}})
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
