package hostinstall

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const activationJournalPath = "/etc/urbit-runner/management-activation.json"
const generationsPath = "/usr/local/lib/urbit-runner/generations"

type executableGeneration struct {
	SHA256    string `json:"sha256"`
	MinSchema int    `json:"min_schema"`
	Schema    int    `json:"schema"`
}

type activationJournal struct {
	Version  int                   `json:"version"`
	Owner    string                `json:"owner"`
	UID      uint32                `json:"uid"`
	RunnerID string                `json:"runner_id"`
	Token    string                `json:"token"`
	Stage    string                `json:"stage"`
	Previous *executableGeneration `json:"previous,omitempty"`
	Target   executableGeneration  `json:"target"`
	Offline  bool                  `json:"offline,omitempty"`
}

type serviceEvidence struct {
	PID        int
	Invocation string
	SHA256     string
}

func digestBytes(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func validDigest(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && hex.EncodeToString(b) == s
}
func (g executableGeneration) valid() bool {
	return validDigest(g.SHA256) && g.MinSchema >= 11 && g.Schema >= g.MinSchema
}
func (g executableGeneration) accepts(schema int) bool {
	return schema >= g.MinSchema && schema <= g.Schema
}
func generationPath(g executableGeneration) string {
	return generationsPath + "/" + g.SHA256 + "/urbit-runner"
}

// Query twice around process/RPC evidence. A foreground listener, an inactive
// unit, an exit-after-exec failure, or a restarted invocation is not readiness.
func (i *installer) serviceEvidence(ctx context.Context, owner string) (serviceEvidence, error) {
	if i.activationEvidence != nil {
		return i.activationEvidence(ctx, owner)
	}
	data, err := i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+owner+"@.host", "show", "urbit-runner.service", "--property=ActiveState,SubState,MainPID,InvocationID")
	if err != nil {
		return serviceEvidence{}, err
	}
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok || fields[k] != "" {
			return serviceEvidence{}, errors.New("ambiguous service invocation evidence")
		}
		fields[k] = v
	}
	pid, err := strconv.Atoi(fields["MainPID"])
	invocation, invErr := hex.DecodeString(fields["InvocationID"])
	if err != nil || pid <= 0 || invErr != nil || len(invocation) != 16 || fields["InvocationID"] == strings.Repeat("0", 32) || fields["ActiveState"] != "active" || fields["SubState"] != "running" {
		return serviceEvidence{}, errors.New("runner listener must belong to an active running service invocation; foreground listeners require explicit offline review")
	}
	f, err := os.Open(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return serviceEvidence{}, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, (128<<20)+1))
	if err != nil || n > 128<<20 {
		return serviceEvidence{}, errors.New("cannot verify service executable")
	}
	return serviceEvidence{pid, fields["InvocationID"], hex.EncodeToString(h.Sum(nil))}, nil
}

func (i *installer) managementRPC(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
	if i.updateRPC != nil {
		return i.updateRPC(ctx, socket, uid, pid, req)
	}
	c := client.NewLocalManagement(socket, uid, pid)
	defer c.Close()
	return c.ManagementUpdate(ctx, req)
}

func (i *installer) managementStopped(ctx context.Context, owner, socket string) error {
	if i.activationStopped != nil {
		return i.activationStopped(ctx, owner, socket)
	}
	data, err := i.command(ctx, "/usr/bin/systemctl", "--user", "--machine="+owner+"@.host", "show", "urbit-runner.service", "--property=ActiveState,MainPID")
	if err != nil {
		return err
	}
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		k, v, _ := strings.Cut(line, "=")
		fields[k] = v
	}
	if fields["MainPID"] != "0" || fields["ActiveState"] != "inactive" && fields["ActiveState"] != "failed" {
		return errors.New("service is not demonstrably stopped")
	}
	c, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "unix", socket)
	if err == nil {
		c.Close()
		return errors.New("management listener survived service stop; foreground listener is not authorized for activation")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("management listener absence is uncertain: %w", err)
	}
	return nil
}

func (i *installer) verifyActivationFence(ctx context.Context, socket string, j *activationJournal) error {
	proof, err := i.serviceEvidence(ctx, j.Owner)
	if err != nil {
		return i.managementStopped(ctx, j.Owner, socket)
	}
	if proof.SHA256 != j.Target.SHA256 && (j.Previous == nil || proof.SHA256 != j.Previous.SHA256) {
		return errors.New("unexpected service generation during activation recovery")
	}
	status, err := i.verifiedManagement(ctx, j.Owner, socket, j.UID, proof.SHA256, j.RunnerID, nil)
	if err != nil {
		return err
	}
	if !status.Quiesced || status.Token != j.Token {
		return errors.New("activation fence is not held; refusing to stop a potentially busy manager")
	}
	return nil
}

func (i *installer) verifiedManagement(ctx context.Context, owner, socket string, uid uint32, digest, runner string, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
	var empty protocol.ManagementUpdateStatus
	before, err := i.serviceEvidence(ctx, owner)
	if err != nil {
		return empty, err
	}
	if before.PID <= 0 || before.Invocation == "" || before.SHA256 != digest {
		return empty, errors.New("service executable differs from the verified generation")
	}
	status, err := i.managementRPC(ctx, socket, uid, before.PID, req)
	if err != nil {
		return empty, fmt.Errorf("local quiesce protocol unavailable or refused; for historical schema19, finish work, stop only urbit-runner.service, then rerun install-host with --offline-upgrade (docs/management-offline-upgrade.md): %w", err)
	}
	after, err := i.serviceEvidence(ctx, owner)
	if err != nil || before != after {
		return empty, errors.New("service invocation changed or exited during verification")
	}
	if status.PID != before.PID || status.ExecutableSHA256 != digest || status.Mode != "development-managed" || !strings.HasPrefix(status.RunnerID, "runner-") || runner != "" && status.RunnerID != runner || status.MinStateSchema < 11 || status.StateSchema < status.MinStateSchema {
		return empty, errors.New("runner process/build/identity/schema evidence mismatch")
	}
	return status, nil
}

func (i *installer) saveActivation(ctx context.Context, j *activationJournal, stage string) error {
	j.Stage = stage
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return i.installFile(ctx, activationJournalPath, append(data, '\n'), 0600, true)
}

func (i *installer) retainGeneration(ctx context.Context, binary []byte, g executableGeneration) error {
	if !g.valid() || digestBytes(binary) != g.SHA256 {
		return errors.New("invalid executable generation")
	}
	if err := i.installFile(ctx, generationPath(g), binary, 0755, false); err != nil {
		return err
	}
	actual, err := i.readInstalled(ctx, generationPath(g), 128<<20)
	if err != nil || digestBytes(actual) != g.SHA256 {
		return errors.New("staged executable verification failed")
	}
	data, _ := json.Marshal(g)
	return i.installFile(ctx, filepath.Dir(generationPath(g))+"/contract.json", append(data, '\n'), 0644, false)
}

func (i *installer) retainedBytes(ctx context.Context, g executableGeneration) ([]byte, error) {
	data, err := i.readInstalled(ctx, generationPath(g), 128<<20)
	if err != nil {
		return nil, err
	}
	if !g.valid() || digestBytes(data) != g.SHA256 {
		return nil, errors.New("retained generation digest mismatch; preserve journal for review")
	}
	contract, err := i.readInstalled(ctx, filepath.Dir(generationPath(g))+"/contract.json", 8192)
	var actual executableGeneration
	if err != nil || json.Unmarshal(contract, &actual) != nil || actual != g {
		return nil, errors.New("retained compatibility contract mismatch")
	}
	return data, nil
}

// Called while the shared setup lock is held. The current authorized installer
// executable supplies the target schema contract; Install* copies /proc/self/exe.
// No candidate is executed as root to discover its metadata.
func (i *installer) prepareActivation(ctx context.Context, owner, socket string, uid uint32, binary []byte, probe hostProbe) (*activationJournal, error) {
	target := executableGeneration{SHA256: digestBytes(binary), MinSchema: 11, Schema: managed.StateSchemaVersion}
	data, err := i.readInstalled(ctx, activationJournalPath, 16384)
	if err == nil {
		var j activationJournal
		if json.Unmarshal(data, &j) != nil || j.Version != 1 || j.Owner != owner || j.UID != uid || !validDigest(j.Token) || !j.Target.valid() || j.Previous != nil && (!j.Previous.valid() || j.RunnerID == "") {
			return nil, errors.New("activation journal is invalid or belongs to another owner; preserve and review it")
		}
		if j.Offline && j.Stage != "complete" {
			return nil, errors.New("resume this activation with install-host --offline-upgrade using the same retained installer generation")
		}
		switch j.Stage {
		case "prepared", "quiesced", "stopping", "starting", "verified", "rollback", "rollback-verified":
			if j.Target != target {
				return nil, errors.New("unfinished activation requires the same installer generation; rerun it using the retained generation, without resetting state")
			}
			if _, err := i.retainedBytes(ctx, j.Target); err != nil {
				return nil, err
			}
			if j.Previous != nil {
				if _, err := i.retainedBytes(ctx, *j.Previous); err != nil {
					return nil, err
				}
			}
			return &j, nil
		case "complete":
			if j.Target == target {
				return &j, nil
			}
		case "rolled-back": // A later rerun may retry the target with a fresh token.
			// Restore availability of the completed rollback first if that
			// service exited. An already live rollback keeps the normal retry
			// behavior below; it is never stopped merely to repair availability.
			if j.Target == target {
				if _, err := i.serviceEvidence(ctx, owner); err != nil {
					if err := i.managementStopped(ctx, owner, socket); err != nil {
						return nil, err
					}
					return &j, nil
				}
			}
		default:
			return nil, errors.New("unknown activation journal stage; preserve for review")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	j := &activationJournal{Version: 1, Owner: owner, UID: uid, Target: target}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	j.Token = hex.EncodeToString(token)
	previous, oldErr := i.readInstalled(ctx, access.ShellPath, 128<<20)
	_, socketErr := i.lstat(i.path(socket))
	if oldErr == nil {
		if socketErr != nil {
			return nil, errors.New("installed runner has no verifiable managed listener; use install-host --offline-upgrade for stopped historical schema19 (docs/management-offline-upgrade.md)")
		}
		h, busy, err := probe(ctx, socket)
		if err != nil || busy {
			return nil, errors.New("existing management is unavailable or busy; finish accepted work before updating")
		}
		status, err := i.verifiedManagement(ctx, owner, socket, uid, digestBytes(previous), h.RunnerID, nil)
		if err != nil {
			return nil, err
		}
		if status.Quiesced {
			return nil, errors.New("runner has a durable activation fence without the matching journal; preserve state and recover that journal")
		}
		if !target.accepts(status.StateSchema) {
			return nil, errors.New("candidate cannot read current state schema; healthy management was preserved")
		}
		j.RunnerID = status.RunnerID
		j.Previous = &executableGeneration{SHA256: digestBytes(previous), MinSchema: status.MinStateSchema, Schema: status.StateSchema}
		if err := i.retainGeneration(ctx, previous, *j.Previous); err != nil {
			return nil, err
		}
	} else if !errors.Is(oldErr, os.ErrNotExist) {
		return nil, oldErr
	} else if !errors.Is(socketErr, os.ErrNotExist) {
		return nil, errors.New("unmanaged or unexplained listener exists; explicit offline review required")
	}
	if err := i.retainGeneration(ctx, binary, target); err != nil {
		return nil, err
	}
	if err := i.saveActivation(ctx, j, "prepared"); err != nil {
		return nil, err
	}
	return j, nil
}

func (i *installer) quiesceActivation(ctx context.Context, socket string, j *activationJournal, manager func(...string) error) error {
	if j.Stage != "prepared" {
		return nil
	}
	if j.Previous != nil {
		// "prepared" records intent, not proof that acquire committed. Restore
		// only the unchanged previous generation if it exited, then obtain an
		// actual acquire response with this same token. Normal startup may admit
		// work when the original acquire never committed; the RPC must refuse
		// that busy state without stopping the previous manager.
		if err := i.ensureGenerationRunning(ctx, socket, j, *j.Previous, manager); err != nil {
			return err
		}
		if _, err := i.awaitGeneration(ctx, socket, j, *j.Previous); err != nil {
			return err
		}
		status, err := i.verifiedManagement(ctx, j.Owner, socket, j.UID, j.Previous.SHA256, j.RunnerID, &protocol.ManagementUpdateRequest{Action: "acquire", RunnerID: j.RunnerID, Token: j.Token})
		if err != nil {
			return err
		}
		if !status.Quiesced || status.Token != j.Token || status.StateSchema != j.Previous.Schema || status.MinStateSchema != j.Previous.MinSchema || !j.Target.accepts(status.StateSchema) {
			return errors.New("runner did not establish the expected compatible quiescence fence")
		}
	}
	return i.saveActivation(ctx, j, "quiesced")
}

func (i *installer) awaitGeneration(ctx context.Context, socket string, j *activationJournal, g executableGeneration) (protocol.ManagementUpdateStatus, error) {
	wait := i.activationTimeout
	if wait == 0 {
		wait = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	var last error
	for {
		status, err := i.verifiedManagement(ctx, j.Owner, socket, j.UID, g.SHA256, j.RunnerID, nil)
		if err == nil {
			if status.StateSchema != g.Schema || status.MinStateSchema != g.MinSchema {
				return status, errors.New("management generation schema contract changed")
			}
			return status, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return protocol.ManagementUpdateStatus{}, fmt.Errorf("activation not verified: %w", last)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (i *installer) awaitActivation(ctx context.Context, socket string, j *activationJournal, g executableGeneration) (protocol.ManagementUpdateStatus, error) {
	status, err := i.awaitGeneration(ctx, socket, j, g)
	if err != nil {
		return status, err
	}
	releaseReplay := j.Stage == "verified" || j.Stage == "rollback-verified" || j.Stage == "complete" || j.Stage == "rolled-back"
	if (j.Previous != nil || releaseReplay) && (status.Token != j.Token || !status.Quiesced && !releaseReplay) {
		return status, errors.New("activation lost its durable fence")
	}
	return status, nil
}

func (i *installer) finishActivation(ctx context.Context, socket string, j *activationJournal, g executableGeneration, final string) (protocol.ManagementUpdateStatus, error) {
	status, err := i.awaitActivation(ctx, socket, j, g)
	if err != nil {
		return status, err
	}
	j.RunnerID = status.RunnerID
	if j.Stage == "complete" || j.Stage == "rolled-back" {
		if status.Quiesced {
			return status, errors.New("completed activation has a new fence; recover its owning journal")
		}
		return status, nil
	}
	// Fresh installation has no previous manager to fence. Acquire now so the
	// same release/replay contract applies (there are no automatic registrations).
	if j.Previous == nil && !status.Quiesced && status.Token != j.Token {
		status, err = i.verifiedManagement(ctx, j.Owner, socket, j.UID, g.SHA256, j.RunnerID, &protocol.ManagementUpdateRequest{Action: "acquire", RunnerID: j.RunnerID, Token: j.Token})
		if err != nil {
			return status, err
		}
	}
	// Persist verification before release: a lost release acknowledgement can be
	// replayed against this same token without stopping/canceling newly accepted work.
	stage := "verified"
	if final == "rolled-back" {
		stage = "rollback-verified"
	}
	if err := i.saveActivation(ctx, j, stage); err != nil {
		return status, err
	}
	status, err = i.verifiedManagement(ctx, j.Owner, socket, j.UID, g.SHA256, j.RunnerID, &protocol.ManagementUpdateRequest{Action: "release", RunnerID: j.RunnerID, Token: j.Token, ExpectedSHA256: g.SHA256})
	if err != nil {
		return status, err
	}
	if status.Quiesced || status.Token != j.Token {
		return status, errors.New("activation release was not verified")
	}
	return status, i.saveActivation(ctx, j, final)
}

// A verified generation can exit before or after release. Recover availability
// with start, never stop/restart: a live released manager may have accepted work.
// Failure to prove inactivity/listener absence is a refusal, not permission to
// take over a foreground process. Every successful start still needs the usual
// invocation/peer/build/identity/schema/token checks before release or readiness.
func (i *installer) resumeVerifiedActivation(ctx context.Context, socket string, j *activationJournal, g executableGeneration, final string, manager func(...string) error) (protocol.ManagementUpdateStatus, error) {
	if err := i.ensureGenerationRunning(ctx, socket, j, g, manager); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	return i.finishActivation(ctx, socket, j, g, final)
}

// Availability recovery proves the intended retained AND installed generation.
// It does not infer admission state from a journal stage or start acknowledgement.
func (i *installer) ensureGenerationRunning(ctx context.Context, socket string, j *activationJournal, g executableGeneration, manager func(...string) error) error {
	if _, err := i.retainedBytes(ctx, g); err != nil {
		return err
	}
	installed, err := i.readInstalled(ctx, access.ShellPath, 128<<20)
	if err != nil || digestBytes(installed) != g.SHA256 {
		return errors.New("management recovery requires the expected installed executable; preserving unexpected generation")
	}
	if _, err := i.serviceEvidence(ctx, j.Owner); err != nil {
		if err := i.managementStopped(ctx, j.Owner, socket); err != nil {
			return err
		}
		// Recheck the installed generation at the dispatch boundary. The shared
		// installer lock excludes another supported installer replacing it.
		installed, err = i.readInstalled(ctx, access.ShellPath, 128<<20)
		if err != nil || digestBytes(installed) != g.SHA256 {
			return errors.New("installed generation changed before management recovery")
		}
		i.mutated = true
		if err := acknowledgedManagementStart(ctx, manager("start", "urbit-runner.service")); err != nil {
			return err
		}
	}
	return nil
}

func (i *installer) activateManagement(ctx context.Context, socket string, j *activationJournal, manager func(...string) error) (protocol.ManagementUpdateStatus, error) {
	if j.Stage == "complete" || j.Stage == "verified" {
		return i.resumeVerifiedActivation(ctx, socket, j, j.Target, "complete", manager)
	}
	if j.Stage == "starting" {
		status, err := i.finishActivation(ctx, socket, j, j.Target, "complete")
		if err == nil || j.Stage == "verified" || j.Stage == "complete" || ctx.Err() != nil {
			return status, err
		}
	}
	if j.Stage == "rollback-verified" || j.Stage == "rolled-back" {
		if j.Previous == nil || !j.Previous.accepts(j.Target.Schema) {
			return protocol.ManagementUpdateStatus{}, errors.New("rollback generation missing or schema-incompatible")
		}
		return i.resumeVerifiedActivation(ctx, socket, j, *j.Previous, "rolled-back", manager)
	}
	if j.Stage == "rollback" {
		return i.rollbackManagement(ctx, socket, j, manager)
	}
	if err := i.quiesceActivation(ctx, socket, j, manager); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := i.verifyActivationFence(ctx, socket, j); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := i.saveActivation(ctx, j, "stopping"); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := manager("stop", "urbit-runner.service"); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := i.managementStopped(ctx, j.Owner, socket); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	data, err := i.retainedBytes(ctx, j.Target)
	if err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := i.installFile(ctx, access.ShellPath, data, 0755, true); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := i.saveActivation(ctx, j, "starting"); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	err = acknowledgedManagementStart(ctx, manager("enable", "--now", "urbit-runner.service"))
	if err == nil {
		status, finishErr := i.finishActivation(ctx, socket, j, j.Target, "complete")
		if finishErr == nil {
			return status, nil
		}
		err = finishErr
	}
	// A verified/released activation may already be accepting work. Never stop
	// it merely because the release acknowledgement or journal sync was lost.
	if j.Stage == "verified" || j.Stage == "complete" {
		return protocol.ManagementUpdateStatus{}, err
	}
	if j.Previous == nil || !j.Previous.accepts(j.Target.Schema) {
		return protocol.ManagementUpdateStatus{}, fmt.Errorf("activation failed; automatic rollback is schema-incompatible or unavailable; retain fence and generations, rerun the same installer to resume: %w", err)
	}
	if ctx.Err() != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if e := i.saveActivation(ctx, j, "rollback"); e != nil {
		return protocol.ManagementUpdateStatus{}, errors.Join(err, e)
	}
	_, rollbackErr := i.rollbackManagement(ctx, socket, j, manager)
	if rollbackErr != nil {
		return protocol.ManagementUpdateStatus{}, errors.Join(err, fmt.Errorf("rollback incomplete; rerun retained installer: %w", rollbackErr))
	}
	return protocol.ManagementUpdateStatus{}, fmt.Errorf("activation failed; verified compatible previous generation restored and admission released: %w", err)
}

func (i *installer) rollbackManagement(ctx context.Context, socket string, j *activationJournal, manager func(...string) error) (protocol.ManagementUpdateStatus, error) {
	if j.Previous == nil || !j.Previous.accepts(j.Target.Schema) {
		return protocol.ManagementUpdateStatus{}, errors.New("rollback refused: previous generation cannot read possibly migrated schema; no marker edits or resets")
	}
	if err := i.verifyActivationFence(ctx, socket, j); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := manager("stop", "urbit-runner.service"); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := i.managementStopped(ctx, j.Owner, socket); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	data, err := i.retainedBytes(ctx, *j.Previous)
	if err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := i.installFile(ctx, access.ShellPath, data, 0755, true); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	if err := acknowledgedManagementStart(ctx, manager("enable", "--now", "urbit-runner.service")); err != nil {
		return protocol.ManagementUpdateStatus{}, err
	}
	return i.finishActivation(ctx, socket, j, *j.Previous, "rolled-back")
}
