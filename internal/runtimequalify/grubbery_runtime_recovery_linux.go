//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

var errDeskProcess = errors.New("exact app runtime process/context unavailable or changed")

func deskProcessBootReader(q *qualifier) func() (string, error) {
	if q.d.readBootID != nil {
		return q.d.readBootID
	}
	return func() (string, error) {
		raw, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
		return strings.TrimSpace(string(raw)), err
	}
}

// The original constructors supply the full validated container and launch/
// network context, including every check in the ordinary process guard. Two
// complete snapshots, namespace reads and engine checks bracket observation;
// nesting another pair of the same process/network guards adds no new evidence.
// No source, application, credential or caller path is accepted.
func deskProcessReader(s *DeskRuntime, engineID, runtimeVersion string, boot func() (string, error), snapshot func(context.Context) (container, any, error)) func(context.Context) (grubbery.RuntimeProcess, error) {
	return func(ctx context.Context) (grubbery.RuntimeProcess, error) {
		refuse := func() (grubbery.RuntimeProcess, error) { return grubbery.RuntimeProcess{}, errDeskProcess }
		if s == nil || s.q == nil || s.check == nil || snapshot == nil || boot == nil || engineID == "" {
			return refuse()
		}
		q := s.q
		if q.verifyEngine(ctx) != nil || q.engineEvidence.ID != engineID {
			return refuse()
		}
		nsReader := &projectAnchorEngine{base: &projectDockerEngine{q: q}, readBootID: boot}
		ns, err := nsReader.namespaceFor(ctx, s.containerID)
		if err != nil {
			return refuse()
		}
		selected, contextBefore, err := snapshot(ctx)
		if err != nil || selected.ID != s.containerID || !deskRunning(selected) || selected.State.Pid < 2 || uint64(selected.State.Pid) > math.MaxUint32 {
			return refuse()
		}
		before, err := json.Marshal(contextBefore)
		if err != nil {
			return refuse()
		}
		afterNS, err := nsReader.namespaceFor(ctx, s.containerID)
		if err != nil || afterNS != ns {
			return refuse()
		}
		afterContainer, contextAfter, err := snapshot(ctx)
		if err != nil {
			return refuse()
		}
		after, err := json.Marshal(contextAfter)
		if err != nil || !bytes.Equal(before, after) || afterContainer.ID != selected.ID || q.verifyEngine(ctx) != nil || q.engineEvidence.ID != engineID {
			return refuse()
		}
		started, err := time.Parse(time.RFC3339Nano, selected.State.StartedAt)
		if err != nil || q.r.RepoDigest != candidate.Image && q.r.RepoDigest != strings.TrimPrefix(candidate.Image, "docker.io/") || runtimeVersion != candidate.VereVersion {
			return refuse()
		}
		private, err := json.Marshal(struct {
			Version   int
			Namespace projectnetwork.NamespaceIdentity
			Context   json.RawMessage
		}{1, ns, before})
		if err != nil {
			return refuse()
		}
		h := sha256.Sum256(append([]byte("omabit-desk-process-context-v1\x00"), private...))
		process := grubbery.RuntimeProcess{ContainerID: s.containerID, PID: uint32(selected.State.Pid), StartedAt: started.UTC(), HostBootID: ns.BootID, EngineID: engineID, ImageManifestDigest: candidate.ManifestDigest, RuntimeVersion: runtimeVersion, ContextSHA256: hex.EncodeToString(h[:])}
		if process.Validate() != nil {
			return refuse()
		}
		return process, nil
	}
}

// GrubberyProcess is a fresh trusted observation of this already selected exact
// process. It cannot select, start, stop, authorize or rebind a runtime.
func (s *DeskRuntime) GrubberyProcess(ctx context.Context) (grubbery.RuntimeProcess, error) {
	if s == nil || s.process == nil {
		return grubbery.RuntimeProcess{}, errDeskProcess
	}
	return s.process(ctx)
}

// GrubberyRecoveryView exposes original typed reads on a newly selected process
// solely for fresh review. Its GrubberyStep always refuses. The caller holds the
// common lifecycle guard and verifies the original actor/pier/framework first.
func (s *DeskRuntime) GrubberyRecoveryView(ctx context.Context, original grubbery.Binding, process grubbery.RuntimeProcess) (grubbery.NativeRuntime, error) {
	return s.grubberyRecovery(ctx, original, process, true)
}

// GrubberyRecovered uses a separately saved, validated immutable audit. The
// authenticated server must load its exact current head from the journal and
// recheck current actor/role/policy/fences for every later phase. A digest or
// caller-supplied receipt alone is not authority. Original requests stay intact.
func (s *DeskRuntime) GrubberyRecovered(ctx context.Context, original grubbery.Record, receipt grubbery.RuntimeRecoveryReceipt) (grubbery.NativeRuntime, error) {
	if receipt.ValidateAncestry(original) != nil {
		return nil, errDeskProcess
	}
	step, done, err := original.Current()
	if err != nil || done {
		return nil, errDeskProcess
	}
	state, err := original.State(step.ID)
	if err != nil || state != "pending" && state != "dispatched" && state != "acknowledged" && state != "uncertain" {
		return nil, errDeskProcess
	}
	return s.grubberyRecovery(ctx, original.Plan.Binding, receipt.Review.Process, false)
}

func (s *DeskRuntime) grubberyRecovery(ctx context.Context, original grubbery.Binding, process grubbery.RuntimeProcess, readOnly bool) (*DeskRuntime, error) {
	if s == nil || s.grubberyOriginal != nil || original.Validate() != nil || process.Validate() != nil || original.RuntimeID == process.ContainerID || s.containerID != process.ContainerID {
		return nil, errDeskProcess
	}
	current, err := s.GrubberyProcess(ctx)
	if err != nil || current != process {
		return nil, errDeskProcess
	}
	out := *s
	out.grubberyOriginal = &original
	out.grubberyReadOnly = readOnly
	out.check = func(ctx context.Context) error {
		current, err := s.GrubberyProcess(ctx)
		if err != nil || current != process {
			return errDeskProcess
		}
		return nil
	}
	if out.grubberyBinding(ctx, original) != nil {
		return nil, errDeskProcess
	}
	return &out, nil
}
