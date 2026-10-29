//go:build linux && amd64 && omabit_private_gateway_vm && omabit_private_member_key_vm && omabit_private_member_boot_vm

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type privateMemberVMProof struct {
	Version         int    `json:"version"`
	ProjectID       string `json:"project_id"`
	InstanceID      string `json:"instance_id"`
	OperationID     string `json:"operation_id"`
	OperationSHA256 string `json:"operation_sha256"`
	LaunchSHA256    string `json:"launch_sha256"`
	ManagerReopened bool   `json:"manager_reopened"`
	ReplayPreserved bool   `json:"replay_preserved"`
}

func TestPrivateMemberBootVM(t *testing.T) {
	mode, err := privateVMGuard()
	stage := os.Getenv("OMABIT_PRIVATE_MEMBER_BOOT_STAGE")
	if err != nil || mode != "verify" || os.Getenv("OMABIT_PRIVATE_MEMBER_KEY") != "proof-v1" || os.Getenv("OMABIT_PRIVATE_MEMBER_BOOT") != "first-v1" || (stage != "launch" && stage != "reopen") {
		t.Fatal("private member boot fixture refused outside its fresh no-NIC QEMU guest")
	}
	host, err := runtimequalify.OpenManagedHost()
	privateVMCheck(t, err, "member-host-lock")
	defer host.Close()
	if host.Directory != privateVMRoot {
		t.Fatal("member fixture root differs")
	}
	privateVMWrite(t, "qualification-private-member-"+stage+".attempt.json", map[string]any{"one_shot": true, "stage": stage})
	r, err := newServerMode(privateVMRoot, realBackend{}, 10*time.Second, true)
	privateVMCheck(t, err, "member-manager")
	defer func() { privateVMCheck(t, r.Close(), "member-close-manager") }()
	prior := privateVMReadProof(t, "verify")
	p, err := loadPrivateProject(r.s.db, prior.ProjectID)
	privateVMCheck(t, err, "member-project")
	w, err := readyPrivateWorkspace(r.s.db, prior.ProjectID)
	privateVMCheck(t, err, "member-workspace")
	g, receipt, err := readyPrivateGateway(r.s.db, prior.ProjectID)
	privateVMCheck(t, err, "member-gateway")
	if privateVMHash(t, p) != prior.ProjectSHA || privateVMHash(t, w) != prior.WorkspaceSHA || privateVMHash(t, g) != prior.ReadySHA || privateVMHash(t, receipt) != prior.RecoverySHA {
		t.Fatal("member launch source history changed")
	}
	const instance = "managed-vm-zod"
	req := privateMemberLaunchRequest{RequestID: "managed-vm-member-first", ExpectedRevision: 1, ReviewDigest: w.PlanDigest, GatewayEvidenceOperationID: receipt.ID, GatewayDigest: privateGatewaySourceDigest(g)}
	if stage == "launch" {
		// Explicitly stage the immutable candidate pill from the sealed appliance.
		// No download, installation change, existing cache replacement or retry.
		cache := filepath.Join(privateVMRoot, "boot-pills")
		if err = os.Mkdir(cache, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			privateVMCheck(t, err, "member-pill-directory")
		}
		privateVMCheck(t, bootpill.PrivateDirectory(cache), "member-pill-directory")
		pill := protocol.BootPill{SHA256: candidate.PillSHA256, Bytes: candidate.PillBytes}
		input, err := bootpill.Open("/opt/omabit-lab/runtime/boot.pill")
		privateVMCheck(t, err, "member-pill-source")
		err = bootpill.Publish(r.ctx, input, pill, bootpill.CachePath(privateVMRoot, pill))
		privateVMCheck(t, errors.Join(err, input.Close()), "member-pill-stage")
		op, hit, err := r.queuePrivateMemberLaunch(privateVMActor, prior.ProjectID, instance, req)
		privateVMCheck(t, err, "member-launch-queue")
		if hit {
			t.Fatal("fresh member launch replayed")
		}
		op = privateVMWait(t, r, op.ID)
		j, err := loadPrivateMemberLaunch(r.s.db, instance)
		privateVMCheck(t, err, "member-launch-record")
		if op.Status != "succeeded" || j.Version != 3 || j.Phase != "running" || j.Running == nil || j.Running.Birth == nil {
			t.Fatalf("member launch incomplete: operation=%s phase=%s", op.Status, j.Phase)
		}
		privateVMCheck(t, j.Validate(p.Plan, w), "member-launch-validation")
		proof := privateMemberVMProof{Version: 1, ProjectID: prior.ProjectID, InstanceID: instance, OperationID: op.ID, OperationSHA256: privateVMHash(t, op), LaunchSHA256: privateVMHash(t, j)}
		privateVMWrite(t, "qualification-private-member-launch.json", proof)
	} else {
		raw, err := os.ReadFile(filepath.Join(privateVMRoot, "qualification-private-member-launch.json"))
		privateVMCheck(t, err, "member-launch-receipt")
		var proof privateMemberVMProof
		if len(raw) > 16<<10 {
			t.Fatal("member launch receipt bound")
		}
		privateVMCheck(t, json.Unmarshal(raw, &proof), "member-launch-receipt")
		j, err := loadPrivateMemberLaunch(r.s.db, instance)
		privateVMCheck(t, err, "member-retained-launch")
		op, err := loadOperation(r.s.db, proof.OperationID)
		privateVMCheck(t, err, "member-retained-operation")
		if proof.Version != 1 || proof.ProjectID != prior.ProjectID || proof.InstanceID != instance || proof.LaunchSHA256 != privateVMHash(t, j) || proof.OperationSHA256 != privateVMHash(t, op) || op.Status != "succeeded" {
			t.Fatal("manager reopen changed member launch")
		}
		for _, only := range []bool{false, true} {
			req.ReplayOnly = only
			again, hit, err := r.queuePrivateMemberLaunch(privateVMActor, prior.ProjectID, instance, req)
			privateVMCheck(t, err, "member-replay")
			if !hit || !reflect.DeepEqual(again, op) {
				t.Fatal("member replay changed receipt")
			}
		}
		wrong, hit, err := r.queuePrivateMemberLaunch(privateVMObserver, prior.ProjectID, instance, req)
		var fault *fault
		if err == nil || !errors.As(err, &fault) || fault.problem.Code != "replay_not_found" || hit || wrong.ID != "" {
			t.Fatal("member replay crossed actors")
		}
		proof.ManagerReopened, proof.ReplayPreserved = true, true
		privateVMWrite(t, "qualification-private-member-reopen.json", proof)
		ctx, cancel := context.WithTimeout(r.ctx, 20*time.Minute)
		defer cancel()
		// The host lock and common runner work gate cover this fixture observation
		// and its explicit shutdown. It supplies no public managed Stop API.
		r.workGate <- struct{}{}
		defer func() { <-r.workGate }()
		result, err := runtimequalify.QualifyPrivateMemberBoot(ctx, privateVMRoot, p.Plan, w, j, func(v runtimequalify.QualificationPrivateMemberBootResult) error {
			privateVMWrite(t, "qualification-private-member-boot-"+v.Phase+".json", v)
			return nil
		})
		privateVMCheck(t, err, "member-http-lens-shutdown")
		if result.Phase != "stopped" || !result.StopAcknowledged || !result.StoppedMetadata {
			t.Fatal("member fixture shutdown incomplete")
		}
		after, err := loadPrivateMemberLaunch(r.s.db, instance)
		privateVMCheck(t, err, "member-unchanged-launch")
		if privateVMHash(t, after) != proof.LaunchSHA256 {
			t.Fatal("fixture rewrote managed launch history")
		}
	}
	after, afterReceipt, err := readyPrivateGateway(r.s.db, prior.ProjectID)
	privateVMCheck(t, err, "member-retained-gateway")
	if privateVMHash(t, after) != prior.ReadySHA || privateVMHash(t, afterReceipt) != prior.RecoverySHA {
		t.Fatal("member fixture rewrote gateway history")
	}
	t.Log("private managed member stage passed: " + stage)
}
