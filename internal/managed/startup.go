package managed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

const startupActor = "system:startup"

var bootIDPattern = regexp.MustCompile(`^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$`)

// NewDevelopmentService enables the explicitly requested startup policy. Boot ID
// comes from the host, never an RPC request, and deduplicates attempts on restart.
func NewDevelopmentService(dir, keys string, resume bool) (*Server, error) {
	boot := ""
	if resume {
		data, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
		if err != nil {
			return nil, err
		}
		boot = strings.TrimSpace(string(data))
		if !bootIDPattern.MatchString(boot) {
			return nil, errors.New("invalid host boot identity")
		}
	}
	r, err := newServerWithStartup(dir, realBackend{}, 10*time.Second, true, boot)
	if err == nil {
		r.keys = keys
	}
	return r, err
}

func startupEligible(v storedInstance) bool {
	return v.Public.DesiredState == "running" && v.Public.ActiveOperationID == "" &&
		v.Public.ConfigRevision == v.Public.AppliedRevision && v.RuntimeConfigRevision == v.Public.ConfigRevision &&
		runtimequalify.StartupResumeCandidate(v.Runtime)
}

// Capture eligibility after reconciling interrupted jobs, before polling changes
// observations. A normal retained stopped container will then report recovery
// required until this specifically authorized cleanup/resume has completed.
func (s *store) startupCandidates() ([]storedInstance, error) {
	all, err := instances(s.db)
	if err != nil {
		return nil, err
	}
	var candidates []storedInstance
	for _, v := range all {
		if !v.Public.RecoveryRequired && startupEligible(v) {
			candidates = append(candidates, v)
		}
	}
	return candidates, nil
}

func (s *store) claimStartup(before storedInstance, state runtimequalify.ManagedState, observation runtimequalify.ManagedObservation, boot string) (protocol.Operation, bool, error) {
	if !bootIDPattern.MatchString(boot) || !runtimequalify.StartupResumeReady(state, observation) || state.ID != before.Runtime.ID {
		return protocol.Operation{}, false, errors.New("invalid startup resume evidence")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	fingerprint := "startup-resume:" + state.ID + ":" + boot
	digest := sha256.Sum256([]byte(fingerprint))
	request := "startup-" + hex.EncodeToString(digest[:16])
	if op, found, err := replay(tx, startupActor, request, fingerprint); found || err != nil {
		return op, false, err
	}
	v, err := loadInstance(tx, before.Public.ID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if v.RuntimeRevision != before.RuntimeRevision || v.Public.ConfigRevision != before.Public.ConfigRevision || !startupEligible(v) {
		return protocol.Operation{}, false, nil
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, RequestID: request, InstanceID: v.Public.ID, Kind: "startup-resume", Actor: startupActor, Status: "queued", ConfigRevision: v.Public.ConfigRevision, CreatedAt: time.Now().UTC()}
	v.Runtime, v.RuntimeRevision = state, v.RuntimeRevision+1
	v.Public.ActiveOperationID = op.ID
	// Fresh clean-exit evidence resolves the retained-container cleanup warning;
	// independent provisioning/maintenance uncertainty was rejected above.
	v.Public.RecoveryRequired = false
	v.Public.Health = "unknown"
	v.Public.Resources, v.Public.Epochs, v.Runtime.Epochs = nil, nil, nil
	if err := insertOperation(tx, op, fingerprint); err != nil {
		return op, false, err
	}
	if err := saveInstance(tx, v); err != nil {
		return op, false, err
	}
	return op, true, tx.Commit()
}

func (r *Server) resumeStartup(candidates []storedInstance, interval time.Duration) {
	defer r.wg.Done()
	pending := append([]storedInstance(nil), candidates...)
	for len(pending) > 0 && r.ctx.Err() == nil {
		next := pending[:0]
		for _, initial := range pending {
			if r.ctx.Err() != nil {
				return
			}
			v, err := loadInstance(r.s.db, initial.Public.ID)
			if err != nil {
				r.failed.Store(true)
				return
			}
			if v.Runtime.ID != initial.Runtime.ID || v.Public.ConfigRevision != initial.Public.ConfigRevision || !startupEligible(v) {
				continue
			}
			ctx, cancel := context.WithTimeout(r.ctx, 20*time.Second)
			state, observation, err := r.b.Observe(ctx, v.Runtime)
			cancel()
			if r.ctx.Err() != nil {
				return
			}
			if err != nil || observation.State == "unknown" {
				next = append(next, initial) // retry observations, never a mutation
				continue
			}
			if !runtimequalify.StartupResumeReady(state, observation) {
				continue
			}
			r.mu.Lock()
			if r.closing || r.failed.Load() {
				r.mu.Unlock()
				return
			}
			op, queued, err := r.s.claimStartup(v, state, observation, r.bootID)
			if err != nil {
				r.failed.Store(true)
			} else if queued {
				r.launch(op)
			}
			r.mu.Unlock()
			if err != nil {
				return
			}
			if !queued && op.ID == "" {
				next = append(next, initial)
			} // observation lost a revision race
		}
		pending = next
		if len(pending) == 0 {
			return
		}
		timer := time.NewTimer(interval)
		select {
		case <-r.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
