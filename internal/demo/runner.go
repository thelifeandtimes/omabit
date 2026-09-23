// Package demo simulates a runner entirely in memory. It never opens a pier or
// invokes Docker, Vere, SSH, or any other subprocess.
package demo

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const maxOperations = 256

type requestRecord struct {
	instance string
	request  protocol.OperationRequest
	opID     string
}

type Runner struct {
	mu         sync.Mutex
	id         string
	actor      string
	delay      time.Duration
	instances  map[string]protocol.Instance
	operations []protocol.Operation
	requests   map[string]requestRecord
	incidents  []protocol.Incident
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	closing    bool
}

func New(delay time.Duration) *Runner {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runner{
		id: "demo-" + rand.Text(), actor: fmt.Sprintf("local-uid:%d", os.Geteuid()),
		delay: delay, ctx: ctx, cancel: cancel,
		instances: map[string]protocol.Instance{
			"alpha": {
				ID: "alpha", Label: "Simulated ship alpha",
				Config:         protocol.Configuration{HTTPPort: 8080, AmesPort: 34343, Loom: 31},
				ConfigRevision: 1, AppliedRevision: 1, DesiredState: "running", ObservedState: "failed",
			},
			"bravo": {
				ID: "bravo", Label: "Simulated ship bravo",
				Config:         protocol.Configuration{HTTPPort: 8081, AmesPort: 34344, Loom: 31},
				ConfigRevision: 1, AppliedRevision: 1, DesiredState: "stopped", ObservedState: "stopped",
			},
		},
		operations: make([]protocol.Operation, 0),
		requests:   make(map[string]requestRecord),
		incidents: []protocol.Incident{{
			ID: "demo-alpha-exit", InstanceID: "alpha", Code: "unexpected_exit",
			Message: "Simulated unexpected exit; no real ship is involved.",
			Active:  true, ObservedAt: time.Now().UTC(),
		}},
	}
}

func (r *Runner) Close() {
	r.mu.Lock()
	r.closing = true
	r.cancel()
	r.mu.Unlock()
	r.wg.Wait()
}

func (r *Runner) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
	w.Header().Set("Cache-Control", "no-store")
	// This socket API is not a browser API. The future gateway has a separate
	// authenticated web surface; do not let browser origins invoke this handler.
	if req.Header.Get("Origin") != "" {
		fail(w, http.StatusForbidden, "origin_not_allowed", "runner RPC does not accept browser origins", nil)
		return
	}
	if req.URL.Path == "/hello" && req.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, protocol.Hello{
			ProtocolVersion: protocol.Version, RunnerID: r.id, Mode: "simulation",
			Features: []string{"snapshot", "configuration-revisions", "idempotent-operations", "incidents"},
		})
		return
	}
	if req.Header.Get(protocol.Header) != strconv.Itoa(protocol.Version) {
		fail(w, http.StatusUpgradeRequired, "protocol_mismatch", "send Urbit-Runner-Protocol: 1", nil)
		return
	}
	if req.URL.Path == "/v1/snapshot" && req.Method == http.MethodGet {
		r.mu.Lock()
		snapshot := protocol.Snapshot{
			RunnerID: r.id, Mode: "simulation",
			Instances:  []protocol.Instance{r.instances["alpha"], r.instances["bravo"]},
			Operations: append([]protocol.Operation{}, r.operations...),
			Incidents:  append([]protocol.Incident{}, r.incidents...),
		}
		r.mu.Unlock()
		writeJSON(w, http.StatusOK, snapshot)
		return
	}
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "v1" || parts[1] != "instances" ||
		req.URL.Path != "/v1/instances/"+parts[2]+"/"+parts[3] {
		fail(w, http.StatusNotFound, "not_found", "unknown runner endpoint", nil)
		return
	}
	id := parts[2]
	switch {
	case parts[3] == "configuration" && req.Method == http.MethodPut:
		var body protocol.ConfigureRequest
		if decode(w, req, &body) {
			r.configure(w, id, body)
		}
	case parts[3] == "operations" && req.Method == http.MethodPost:
		var body protocol.OperationRequest
		if decode(w, req, &body) {
			r.operate(w, id, body)
		}
	default:
		fail(w, http.StatusNotFound, "not_found", "unknown runner endpoint or method", nil)
	}
}

func (r *Runner) configure(w http.ResponseWriter, id string, req protocol.ConfigureRequest) {
	if req.ExpectedRevision == 0 || req.Config.HTTPPort == 0 || req.Config.AmesPort == 0 ||
		req.Config.Loom < 31 || req.Config.Loom > 33 {
		fail(w, http.StatusBadRequest, "invalid_configuration", "require a positive revision, nonzero ports, and demo loom 31..33", nil)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	instance, exists := r.instances[id]
	if !exists {
		fail(w, http.StatusNotFound, "instance_not_found", "unknown simulated instance", nil)
		return
	}
	if instance.ConfigRevision != req.ExpectedRevision {
		fail(w, http.StatusConflict, "revision_conflict", "configuration changed; inspect a fresh snapshot before editing", &instance.ConfigRevision)
		return
	}
	if instance.ActiveOperationID != "" {
		fail(w, http.StatusConflict, "instance_busy", "wait for the active operation before changing configuration", nil)
		return
	}
	for otherID, other := range r.instances {
		if otherID != id && (other.Config.HTTPPort == req.Config.HTTPPort || other.Config.AmesPort == req.Config.AmesPort) {
			fail(w, http.StatusConflict, "port_conflict", "port is reserved by another simulated instance for the same transport", nil)
			return
		}
	}
	if instance.Config != req.Config {
		instance.Config = req.Config
		instance.ConfigRevision++
		r.instances[id] = instance
	}
	writeJSON(w, http.StatusOK, instance)
}

func (r *Runner) operate(w http.ResponseWriter, id string, req protocol.OperationRequest) {
	if !validRequestID(req.RequestID) || req.ExpectedRevision == 0 || (req.Kind != "start" && req.Kind != "stop") {
		fail(w, http.StatusBadRequest, "invalid_operation", "require request_id (1..64 ASCII letters/digits/_/-), positive revision, and start or stop", nil)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if previous, exists := r.requests[req.RequestID]; exists {
		if previous.instance != id || previous.request != req {
			fail(w, http.StatusConflict, "request_conflict", "request_id already identifies different input; do not reuse it", nil)
			return
		}
		for _, op := range r.operations {
			if op.ID == previous.opID {
				writeJSON(w, http.StatusOK, op)
				return
			}
		}
	}
	instance, exists := r.instances[id]
	if !exists {
		fail(w, http.StatusNotFound, "instance_not_found", "unknown simulated instance", nil)
		return
	}
	if r.closing {
		fail(w, http.StatusServiceUnavailable, "runner_stopping", "runner is stopping", nil)
		return
	}
	if instance.ConfigRevision != req.ExpectedRevision {
		fail(w, http.StatusConflict, "revision_conflict", "configuration changed; inspect a fresh snapshot before acting", &instance.ConfigRevision)
		return
	}
	if instance.ActiveOperationID != "" {
		fail(w, http.StatusConflict, "instance_busy", "an incompatible operation is already running", nil)
		return
	}
	if (req.Kind == "start" && instance.ObservedState == "running") ||
		(req.Kind == "stop" && instance.ObservedState == "stopped") {
		fail(w, http.StatusConflict, "invalid_state", "instance is already in the requested state", nil)
		return
	}
	if len(r.operations) >= maxOperations {
		fail(w, http.StatusServiceUnavailable, "demo_capacity", "demo history is full; restart the simulator to reset it", nil)
		return
	}
	op := protocol.Operation{
		ID: "op-" + rand.Text(), RequestID: req.RequestID, InstanceID: id,
		Kind: req.Kind, Actor: r.actor, Status: "running",
		ConfigRevision: instance.ConfigRevision, CreatedAt: time.Now().UTC(),
	}
	instance.ActiveOperationID = op.ID
	if req.Kind == "start" {
		instance.DesiredState, instance.ObservedState = "running", "starting"
	} else {
		instance.DesiredState, instance.ObservedState = "stopped", "stopping"
	}
	r.instances[id] = instance
	r.operations = append(r.operations, op)
	r.requests[req.RequestID] = requestRecord{instance: id, request: req, opID: op.ID}
	r.wg.Add(1)
	go r.finish(len(r.operations) - 1)
	writeJSON(w, http.StatusAccepted, op)
}

func (r *Runner) finish(index int) {
	defer r.wg.Done()
	timer := time.NewTimer(r.delay)
	defer timer.Stop()
	select {
	case <-r.ctx.Done():
		return
	case <-timer.C:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	op := &r.operations[index]
	instance := r.instances[op.InstanceID]
	instance.ObservedState = instance.DesiredState
	instance.ActiveOperationID = ""
	if op.Kind == "start" {
		instance.AppliedRevision = op.ConfigRevision
		for i := range r.incidents {
			if r.incidents[i].InstanceID == instance.ID {
				r.incidents[i].Active = false
			}
		}
	}
	r.instances[instance.ID] = instance
	finished := time.Now().UTC()
	op.Status, op.FinishedAt = "succeeded", &finished
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for i := range id {
		c := id[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func decode(w http.ResponseWriter, req *http.Request, value any) bool {
	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		fail(w, http.StatusUnsupportedMediaType, "invalid_content_type", "require application/json", nil)
		return false
	}
	req.Body = http.MaxBytesReader(w, req.Body, 4096)
	defer req.Body.Close()
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		fail(w, http.StatusBadRequest, "invalid_json", "request must match the documented JSON schema (maximum 4096 bytes)", nil)
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		fail(w, http.StatusBadRequest, "invalid_json", "require exactly one JSON value", nil)
		return false
	}
	return true
}

func fail(w http.ResponseWriter, status int, code, message string, revision *uint64) {
	writeJSON(w, status, protocol.ErrorResponse{Error: protocol.APIError{
		Code: code, Message: message, CurrentRevision: revision,
	}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
