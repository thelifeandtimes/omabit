//go:build linux && amd64

package managed

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectAnchorRecoveryContinuationDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, true, true, true, true, true)
}

func TestProjectAnchorRecoveryDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, true, true, true, true)
}

func exerciseVMAnchorRecovery(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, receiver, sender, parent map[string]any, docker func(...string) json.RawMessage, registerCleanup func(func())) {
	t.Helper()
	result := map[string]any{"ok": false, "stage": "selection", "production_mutations": true}
	parent["anchor_recovery"] = result
	// The two old replacement cleanup callbacks retain this same result, including
	// partial retirement evidence when a subsequent adapter fails.
	cleanupReceipt := &vmRecoveryCleanupReceipt{}
	receiver["anchor_recovery_retirement"], sender["anchor_recovery_retirement"] = cleanupReceipt, cleanupReceipt
	history := currentStopHistory(t, r.s)
	h, _, _, err := projectRecoveryHistory(r.s.db, p.ID, true)
	if err != nil {
		t.Error(err)
		return
	}
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		return
	}
	req := projectAnchorRecoveryRequest{RequestID: "recover-vm-anchor", ExpectedRevision: p.Revision, ReviewDigest: digest, HistoryDigest: recoveryHistoryDigest(h)}
	b.vmAnchorRecoveryBackend = &vmAnchorRecoveryBackend{store: r.s, project: p.ID, loseSignal: b.continuation, loseCreate: b.continuation, loseStart: b.continuation}
	cleanups := []map[string]any{}
	b.vmAnchorRecoveryBackend.register = func(m projectnetwork.RecoveredMember, id string) {
		child := map[string]any{"instance_id": m.Creation.InstanceID, "container_id": id, "ok": false}
		cleanups = append(cleanups, child)
		registerCleanup(func() { vmStopRecoveredMember(t, r.s.runnerID, m, id, child, docker) })
	}
	if b.publicRecovery {
		cli, e := vmOpenPublicRecoveryCLI(ctx, r, p)
		if e != nil {
			t.Error(e)
			return
		}
		defer cli.close()
		b.vmAnchorRecoveryBackend.publicCLI = cli
		result["public_recovery"] = cli.proof
	}
	queue := func(replay bool) (protocol.Operation, bool, error) {
		if cli := b.vmAnchorRecoveryBackend.publicCLI; cli != nil {
			q := protocol.ProjectRecoveryActionRequest{RequestID: req.RequestID, ExpectedRevision: req.ExpectedRevision, ReviewDigest: req.ReviewDigest, HistoryDigest: req.HistoryDigest, ConfirmAction: true}
			op, e := cli.recover("start", "", q, replay)
			return op, replay, e
		}
		return r.queueProjectAnchorRecovery("qualification", p.ID, req)
	}
	op, hit, err := queue(false)
	if err != nil || hit {
		t.Error("recovery admission", err)
		return
	}
	result["operation_id"], result["request"] = op.ID, req
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Error("recovery worker deadline")
		return
	}
	if b.continuation {
		proof, continuationErr := vmExerciseRecoveryContinuation(ctx, r, b.vmAnchorRecoveryBackend, p, op.ID)
		result["continuation"] = proof
		if continuationErr != nil {
			t.Error(continuationErr)
		}
	}
	result["events"], result["cleanup"] = b.vmAnchorRecoveryBackend.events, cleanups
	job, err := loadProjectAnchorRecovery(r.s.db, op.ID)
	if err != nil {
		t.Error(err)
		return
	}
	result["journal"] = job
	cleanupReceipt.job = &job
	receipt, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		t.Error(err)
		return
	}
	result["operation"] = receipt
	if receipt.Status != "succeeded" || job.Journal.Phase != "complete" || job.Journal.Readiness == nil || job.Journal.Validate(job.Source) != nil {
		t.Error("coordinated recovery incomplete; exact journal and adapter events retained", receipt.Status, job.Journal.Phase)
		return
	}
	assertContextHistory(t, r.s, history)
	if t.Failed() {
		return
	}
	result["original_history_unchanged"] = true
	previousEvents := len(b.vmAnchorRecoveryBackend.events)
	replay, hit, err := queue(true)
	r.wg.Wait()
	if err != nil || !hit || !reflect.DeepEqual(replay, receipt) || len(b.vmAnchorRecoveryBackend.events) != previousEvents {
		t.Error("recovery replay changed outcome or dispatched work", err)
		return
	}
	result["replay_unchanged"] = true
	if cli := b.vmAnchorRecoveryBackend.publicCLI; cli != nil {
		if len(cli.calls) != 12 {
			t.Error("public recovery lost a submission or lookup", len(cli.calls))
			return
		}
		cli.proof["ok"] = true
	}
	if err = vmWriteApplicationSuccession(job); err != nil {
		t.Error(err)
		return
	}
	// Ordinary sender-cycle immutability was verified before the fault. Both
	// processes are expected to change here; now pin both recovered processes.
	before := map[string]json.RawMessage{}
	for _, m := range job.Journal.Readiness.Replacement.Members {
		before[m.Container.ID] = docker("container", "inspect", m.Container.ID)
	}
	b.checkLivePeer = func() error {
		for id, old := range before {
			if err := vmCheckUnchangedLivePeer(old, docker("container", "inspect", id)); err != nil {
				return err
			}
		}
		return nil
	}
	result["stage"] = "application-recovery"
	vmApplicationPhase(t, ctx, b, "after-anchor")
	if t.Failed() {
		return
	}
	if err = b.checkLivePeer(); err != nil {
		t.Error(err)
		return
	}
	result["successor_processes_unchanged"] = true
	if b.continuation {
		status, statusErr := vmProjectStatusCLI(ctx, r, p, op.ID)
		result["public_status"] = status
		if statusErr != nil {
			t.Error("public status CLI", statusErr)
			return
		}
	}
	result["stage"], result["ok"] = "complete", true
}

func vmWriteApplicationSuccession(j projectAnchorRecoveryJob) error {
	if j.Journal.Readiness == nil || j.Journal.Phase != "complete" || j.Journal.Validate(j.Source) != nil {
		return errors.New("application succession requires complete persisted recovery")
	}
	raw, err := os.ReadFile(vmAppSpecPath)
	if err != nil || len(raw) > 65536 {
		return errors.New("original application spec unavailable")
	}
	var spec map[string]any
	if json.Unmarshal(raw, &spec) != nil {
		return errors.New("original spec invalid")
	}
	binding := j.Journal.Readiness.Binding
	previous := fmt.Sprintf("net:[%d]", j.Journal.Readiness.Replacement.Retirement.Namespace.Drain.Review.Context.Binding.Namespace.Inode)
	next := fmt.Sprintf("net:[%d]", binding.Namespace.Inode)
	if spec["namespace"] != previous || previous == next || spec["project_id"] != j.ProjectID || spec["anchor_id"] != binding.Anchor.ID {
		return errors.New("application succession source changed")
	}
	v := map[string]any{"format": 1, "spec_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "runner_id": spec["runner_id"], "project_id": j.ProjectID, "anchor_id": binding.Anchor.ID, "previous_namespace": previous, "namespace": next, "operation_id": j.OperationID}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile("/home/lab/project-app-succession.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return errors.Join(err, f.Sync(), f.Close())
}

func vmReopenAnchorRecovery(t *testing.T, s *store, members []map[string]any) {
	t.Helper()
	for _, member := range members {
		result, ok := member["anchor_recovery"].(map[string]any)
		if !ok {
			continue
		}
		old, ok := result["journal"].(projectAnchorRecoveryJob)
		if !ok || result["ok"] != true {
			t.Fatal("complete recovery receipt missing")
		}
		j, err := loadProjectAnchorRecovery(s.db, old.OperationID)
		if err != nil || !reflect.DeepEqual(j, old) {
			t.Fatal("reopen changed recovery journal", err)
		}
		op, err := loadOperation(s.db, old.OperationID)
		if err != nil || !reflect.DeepEqual(op, result["operation"]) {
			t.Fatal("reopen changed recovery operation", err)
		}
		req := result["request"].(projectAnchorRecoveryRequest)
		// Preserve the accepted public or private actor and perform lookup only.
		req.ReplayOnly = true
		replay, hit, err := s.acceptProjectAnchorRecovery(op.Actor, j.ProjectID, req)
		if err != nil || !hit || !reflect.DeepEqual(replay, op) {
			t.Fatal("reopen recovery replay changed", err)
		}
		if err = validateProjectAnchorRecoveries(s.db); err != nil {
			t.Fatal(err)
		}
		h, _, _, err := projectRecoveryHistory(s.db, j.ProjectID, false)
		if err != nil || len(h.Epochs) != 1 {
			t.Fatal("completed recovery history missing", err)
		}
		result["reopen_unchanged"] = true
		return
	}
	t.Fatal("recovery result not retained")
}

// The unexported pointer is fixture cleanup state, not duplicate JSON evidence.
// The single parent journal and explicit operation link retain its authority.
type vmRecoveryCleanupReceipt struct{ job *projectAnchorRecoveryJob }

// Return true only when this result explicitly belongs to coordinated cleanup.
func vmRecoveryRetirementCleanup(t *testing.T, id, creator string, storage pierstore.Location, result map[string]any, docker func(...string) json.RawMessage) bool {
	raw, ok := result["anchor_recovery_retirement"].(*vmRecoveryCleanupReceipt)
	if !ok {
		return false
	}
	j := raw.job
	if j == nil {
		return false
	}
	var retired *projectnetwork.AnchorRetirement
	switch j.Journal.Phase {
	case "retiring":
		retired = j.Journal.Retirement
	case "replacing":
		if j.Journal.Replacement != nil {
			retired = &j.Journal.Replacement.Retirement
		}
	case "complete":
		if j.Journal.Readiness != nil {
			retired = &j.Journal.Readiness.Replacement.Retirement
		}
	}
	if retired == nil {
		return false
	}
	if j.Journal.Validate(j.Source) != nil {
		t.Error("invalid retained recovery cleanup authority")
		return true
	}
	for i, m := range retired.Namespace.Drain.Review.Context.Members {
		old := m.Context.Member
		if old.Container.ID != id {
			continue
		}
		if old.OperationID != creator || old.Storage != storage || i >= len(retired.Members) || retired.Members[i].Phase != "absent" {
			return false
		}
		inventory := docker("container", "ls", "--all", "--quiet", "--no-trunc")
		if inventory == nil {
			t.Error("recovery cleanup inventory unavailable")
			return true
		}
		for _, found := range strings.Fields(string(inventory)) {
			if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(found) || found == id {
				t.Error("recovered old container still present")
				return true
			}
		}
		result["anchor_recovery_operation_id"] = j.OperationID
		result["retirement_cleanup_inventory"], result["retired_container_absent"], result["clean_member_stop"] = strings.Fields(string(inventory)), true, true
		return true
	}
	return false
}
func vmStopRecoveredMember(t *testing.T, runner string, m projectnetwork.RecoveredMember, id string, result map[string]any, docker func(...string) json.RawMessage) {
	t.Helper()
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(id) {
		t.Error("invalid recovered cleanup ID")
		return
	}
	c := m.Creation
	before := docker("container", "inspect", id)
	result["before_stop"] = before
	row, err := vmRecoveredCleanupInspection(runner, c, id, before)
	if err != nil {
		t.Error(err)
		return
	}
	if row.State.Running {
		docker("container", "stop", "--timeout", "300", id)
	}
	after := docker("container", "inspect", id)
	result["after_stop"] = after
	row, err = vmRecoveredCleanupInspection(runner, c, id, after)
	if err != nil || row.State.Running || row.State.OOMKilled || row.State.ExitCode != 0 {
		t.Error("recovered member cleanup failed")
		return
	}
	state, err := vmRetirementState(c.Storage)
	if err != nil {
		t.Error(err)
		return
	}
	result["state_after_stop"], result["ok"] = state, true
}

// The cleanup fixture must retain the same complete ownership contract as the
// recovered container. A successful recovery adds two operation bindings.
type vmRecoveredCleanupContainer struct {
	ID     string `json:"Id"`
	Name   string
	Config struct{ Labels map[string]string }
	State  struct {
		Running, OOMKilled bool
		ExitCode           int
		Status, StartedAt  string
	}
}

func vmRecoveredCleanupInspection(runner string, c projectnetwork.RecoveryMemberCreation, id string, raw json.RawMessage) (vmRecoveredCleanupContainer, error) {
	prefix := "org.urbit-runner.project"
	labels := map[string]string{prefix: "true", prefix + ".runner": runner, prefix + ".project": c.OriginalAnchor.Prepared.ProjectID, prefix + ".generation": strconv.FormatUint(c.OriginalAnchor.Prepared.Generation, 10), prefix + ".instance": c.InstanceID, prefix + ".runtime": c.RuntimeID, prefix + ".operation": c.OperationID, prefix + ".container-generation": strconv.FormatUint(c.Generation, 10), prefix + ".namespace-operation": c.NamespaceOperationID, prefix + ".retirement-operation": c.RetirementOperationID}
	var rows []vmRecoveredCleanupContainer
	if c.NamespaceOperationID == "" || c.RetirementOperationID == "" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(id) || json.Unmarshal(raw, &rows) != nil || len(rows) != 1 || rows[0].ID != id || rows[0].Name != "/urbit-project-member-"+c.RuntimeID+"-g"+strconv.FormatUint(c.Generation, 10) || !reflect.DeepEqual(rows[0].Config.Labels, labels) {
		return vmRecoveredCleanupContainer{}, errors.New("recovered cleanup ownership mismatch")
	}
	return rows[0], nil
}
