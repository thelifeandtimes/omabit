//go:build linux && amd64 && omabit_private_gateway_vm && omabit_private_member_key_vm

package managed

import (
	"context"
	"os"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestPrivateMemberKeyProofVM(t *testing.T) {
	mode, err := privateVMGuard()
	if err != nil || mode != "verify" || os.Getenv("OMABIT_PRIVATE_MEMBER_KEY") != "proof-v1" {
		t.Fatal("private member key fixture refused outside its fresh no-NIC QEMU guest")
	}
	host, err := runtimequalify.OpenManagedHost()
	privateVMCheck(t, err, "key-proof-host-lock")
	defer host.Close()
	if host.Directory != privateVMRoot {
		t.Fatal("member key fixture root differs")
	}
	privateVMWrite(t, "qualification-private-member-key.attempt.json", map[string]any{"one_shot": true, "ship_boots": 0})
	r, err := newServerMode(privateVMRoot, realBackend{}, 10*time.Second, true)
	privateVMCheck(t, err, "key-proof-manager")
	defer func() { privateVMCheck(t, r.Close(), "key-proof-close-manager") }()
	prior := privateVMReadProof(t, "verify")
	p, err := loadPrivateProject(r.s.db, prior.ProjectID)
	privateVMCheck(t, err, "key-proof-project")
	w, err := readyPrivateWorkspace(r.s.db, prior.ProjectID)
	privateVMCheck(t, err, "key-proof-workspace")
	g, receipt, err := readyPrivateGateway(r.s.db, prior.ProjectID)
	privateVMCheck(t, err, "key-proof-gateway")
	if privateVMHash(t, p) != prior.ProjectSHA || privateVMHash(t, w) != prior.WorkspaceSHA || privateVMHash(t, g) != prior.ReadySHA || privateVMHash(t, receipt) != prior.RecoverySHA {
		t.Fatal("key proof source history changed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	result, err := runtimequalify.QualifyPrivateMemberKeys(ctx, privateVMRoot, p.Plan, w, g)
	privateVMCheck(t, err, "key-proof-runtime")
	if len(result.Cases) != 6 || result.ShipBoots != 0 {
		t.Fatal("key proof case inventory differs")
	}
	privateVMWrite(t, "qualification-private-member-key.json", result)
	after, afterReceipt, err := readyPrivateGateway(r.s.db, prior.ProjectID)
	privateVMCheck(t, err, "key-proof-retained-history")
	if privateVMHash(t, after) != prior.ReadySHA || privateVMHash(t, afterReceipt) != prior.RecoverySHA {
		t.Fatal("key proof rewrote history")
	}
	t.Log("private member key proof passed: two valid keys, four explicit negative verdicts, unchanged gateway and retained history")
}
