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

// The pending boot is saved before polling can replace the previous healthy
// observation with an exited-container warning. It survives a management restart
// before dispatch; once dispatched, the operation journal owns the attempt.
type startupState struct {
	HealthyBootID  string `json:"healthy_boot_id,omitempty"`
	ContainerID    string `json:"container_id"`
	ConfigRevision uint64 `json:"config_revision"`
	PendingBootID  string `json:"pending_boot_id,omitempty"`
	AllowUnclean   bool   `json:"allow_unclean,omitempty"`
}

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

func startupContainer(v storedInstance) string {
	for _, c := range v.Runtime.Runtime.Containers {
		if !c.Removed {
			return c.ID
		}
	}
	return ""
}

func startupMatches(v storedInstance) bool {
	return v.Startup != nil && v.Startup.ContainerID == startupContainer(v) && v.Startup.ConfigRevision == v.Public.ConfigRevision
}

func (s *store) recordHealthyBoot(v *storedInstance) {
	if s.bootID == "" || v.Public.ObservedState != "running" || v.Public.Health != "healthy" || v.Public.RecoveryRequired || !startupEligible(*v) {
		return
	}
	v.Startup = &startupState{HealthyBootID: s.bootID, ContainerID: startupContainer(*v), ConfigRevision: v.Public.ConfigRevision}
}

// Capture and persist eligibility after reconciling interrupted jobs, before
// polling. Legacy records retain clean-only startup until observed healthy by
// this version. A known failure without a pending boot is never re-authorized.
func (s *store) startupCandidates() ([]storedInstance, error) {
	if !bootIDPattern.MatchString(s.bootID) {
		return nil, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	all, err := instances(tx)
	if err != nil {
		return nil, err
	}
	var candidates []storedInstance
	for _, v := range all {
		if !startupEligible(v) {
			continue
		}
		pending := startupMatches(v) && bootIDPattern.MatchString(v.Startup.PendingBootID)
		if !pending && v.Public.RecoveryRequired {
			continue
		}
		if !startupMatches(v) {
			v.Startup = &startupState{ContainerID: startupContainer(v), ConfigRevision: v.Public.ConfigRevision}
		}
		if !pending || v.Startup.PendingBootID != s.bootID {
			v.Startup.AllowUnclean = bootIDPattern.MatchString(v.Startup.HealthyBootID) && v.Startup.HealthyBootID != s.bootID
		}
		v.Startup.PendingBootID = s.bootID
		v.RuntimeRevision++ // invalidate observations captured before authorization
		if err := saveInstance(tx, v); err != nil {
			return nil, err
		}
		candidates = append(candidates, v)
	}
	return candidates, tx.Commit()
}

func startupAuthorized(v storedInstance, boot string) bool {
	return startupEligible(v) && startupMatches(v) && v.Startup.PendingBootID == boot &&
		(!v.Startup.AllowUnclean || bootIDPattern.MatchString(v.Startup.HealthyBootID) && v.Startup.HealthyBootID != boot)
}

// A definitive non-candidate (including a same-boot failure) must not retain a
// pending authorization that could be carried into a later boot.
func (s *store) dismissStartup(before storedInstance) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	v, err := loadInstance(tx, before.Public.ID)
	if err != nil {
		var f *fault
		if errors.As(err, &f) && f.problem.Code == "instance_not_found" {
			return nil
		}
		return err
	}
	if v.Runtime.ID == before.Runtime.ID && startupAuthorized(v, s.bootID) && *v.Startup == *before.Startup {
		v.Startup = nil
		v.RuntimeRevision++
		if err := saveInstance(tx, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func authorizeStartupState(v storedInstance, state runtimequalify.ManagedState) runtimequalify.ManagedState {
	state.StartupResume = &runtimequalify.StartupResumeIntent{ContainerID: v.Startup.ContainerID, AllowUnclean: v.Startup.AllowUnclean}
	return state
}

func (s *store) claimStartup(before storedInstance, state runtimequalify.ManagedState, observation runtimequalify.ManagedObservation, boot string) (protocol.Operation, bool, error) {
	if !bootIDPattern.MatchString(boot) || !startupAuthorized(before, boot) || state.ID != before.Runtime.ID {
		return protocol.Operation{}, false, errors.New("invalid startup resume authorization")
	}
	state = authorizeStartupState(before, state)
	if !runtimequalify.StartupResumeReady(state, observation) {
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
		var f *fault
		if errors.As(err, &f) && f.problem.Code == "instance_not_found" {
			return protocol.Operation{}, false, nil
		}
		return protocol.Operation{}, false, err
	}
	if v.Runtime.ID != before.Runtime.ID || v.RuntimeRevision != before.RuntimeRevision || v.Public.ConfigRevision != before.Public.ConfigRevision || !startupAuthorized(v, boot) || *v.Startup != *before.Startup {
		return protocol.Operation{}, false, nil
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, RequestID: request, InstanceID: v.Public.ID, Kind: "startup-resume", Actor: startupActor, Status: "queued", ConfigRevision: v.Public.ConfigRevision, CreatedAt: time.Now().UTC()}
	if state.StartupResume.AllowUnclean {
		op.Message = "Resuming the existing pier after host reboot; Vere replays persisted events as needed."
	}
	v.Runtime, v.RuntimeRevision = state, v.RuntimeRevision+1
	v.Public.ActiveOperationID = op.ID
	v.Startup.PendingBootID = ""
	// The journaled attempt resolves only the retained-container warning;
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
				var f *fault
				if errors.As(err, &f) && f.problem.Code == "instance_not_found" {
					continue // deletion won the race
				}
				r.failed.Store(true)
				return
			}
			if v.Runtime.ID != initial.Runtime.ID || v.Public.ConfigRevision != initial.Public.ConfigRevision || !startupAuthorized(v, r.bootID) {
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
			if !runtimequalify.StartupResumeReady(authorizeStartupState(v, state), observation) {
				if err := r.s.dismissStartup(v); err != nil {
					r.failed.Store(true)
					return
				}
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
