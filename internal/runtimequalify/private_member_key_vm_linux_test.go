//go:build linux && amd64 && omabit_private_member_key_vm

package runtimequalify

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func TestKeyVMBridgeRefusesHostBeforeRuntimeAccess(t *testing.T) {
	for _, armed := range []bool{false, true} {
		if armed {
			t.Setenv("OMABIT_PRIVATE_MEMBER_KEY", "proof-v1")
			t.Setenv("OMABIT_PRIVATE_MANAGED_GATEWAY", "guest-v1")
			t.Setenv("OMABIT_PRIVATE_MANAGED_STAGE", "verify")
			t.Setenv("HOME", "/home/lab")
			t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
		}
		if _, err := QualifyPrivateMemberKeys(context.Background(), "/home/lab/.local/share/urbit-runner/managed", privatechain.Plan{}, privatechain.Workspace{}, privatechain.Gateway{}); !errors.Is(err, errKeyVMQualification) {
			t.Fatal("host guard did not precede selection/runtime", err)
		}
	}
}
func TestKeyVMNegativeProgramsRequireExplicitNegativeVerdict(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node needed for embedded negative program qualification")
	}
	c, _ := privateKeyProofFixture(t)
	for _, control := range []string{"wrong-public-key", "wrong-life"} {
		t.Run(control, func(t *testing.T) {
			args, err := keyVMNegativeArgs(c, control)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(args[8], "!=='%.n'") != 1 || strings.Contains(args[8], "!=='%.y'") || strings.Count(args[8], "expression(ship,value.key,qualificationNegativePoint(point))") != 1 {
				t.Fatal("negative verdict not exact")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, node, "--eval", args[8], args[9])
			cmd.Env = []string{"HOME=" + t.TempDir(), "LANG=C.UTF-8"}
			out, err := cmd.CombinedOutput()
			if err == nil || ctx.Err() != nil || string(out) != "private_member_key_proof_refused\n" {
				t.Fatal("embedded negative failed guard", err, string(out))
			}
		})
	}
	for _, control := range []string{"", "valid", "other"} {
		if _, err := keyVMNegativeArgs(c, control); err == nil {
			t.Fatal("unselected control accepted")
		}
	}
}
