//go:build linux && amd64 && omabit_private_member_key_vm

package runtimequalify

// Qualification-only bridge; absent from ordinary runner builds. It performs
// observations and ephemeral Vere evaluation in the fresh guest's exact gateway,
// never a member launch, key mutation or chain write. The caller holds the host
// management fence and retains the private receipt for stopped-disk inspection.
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

var errKeyVMQualification = errors.New("private member key fixture refused outside its fresh no-NIC QEMU guest")

func keyVMQualificationGuard() error {
	if os.Getenv("OMABIT_PRIVATE_MEMBER_KEY") != "proof-v1" || os.Getenv("OMABIT_PRIVATE_MANAGED_GATEWAY") != "guest-v1" || os.Getenv("OMABIT_PRIVATE_MANAGED_STAGE") != "verify" || os.Getuid() != 1000 || os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/lab" || os.Getenv("XDG_RUNTIME_DIR") != "/run/user/1000" {
		return errKeyVMQualification
	}
	for p, want := range map[string]string{"/opt/omabit-lab/guest-marker": "offline-import-guest-v1", "/sys/class/dmi/id/sys_vendor": "QEMU", "/proc/1/comm": "systemd"} {
		raw, err := os.ReadFile(p)
		if err != nil || strings.TrimSpace(string(raw)) != want {
			return errKeyVMQualification
		}
	}
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil || len(entries) != 1 || entries[0].Name() != "lo" {
		return errKeyVMQualification
	}
	return nil
}

type QualificationPrivateMemberKeyCase struct {
	InstanceID      string          `json:"instance_id"`
	Ship            string          `json:"ship"`
	Control         string          `json:"control"`
	Proof           json.RawMessage `json:"proof"`
	BeforeSHA256    string          `json:"before_sha256"`
	AfterSHA256     string          `json:"after_sha256"`
	HostKeyVerified bool            `json:"host_key_verified"`
	Seconds         float64         `json:"seconds"`
}
type QualificationPrivateMemberKeyResult struct {
	Version         int                                 `json:"version"`
	PlanDigest      string                              `json:"plan_digest"`
	GatewaySHA256   string                              `json:"gateway_sha256"`
	WorkspaceSHA256 string                              `json:"workspace_sha256"`
	Cases           []QualificationPrivateMemberKeyCase `json:"cases"`
	ShipBoots       int                                 `json:"ship_boots"`
}

func keyVMHash(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func keyVMObservationHash(o privatechain.GatewayObservation) string {
	o.ObservedAt = time.Time{}
	return keyVMHash(o)
}

// Negative controls preserve both keyfiles and the chain. Only the public point
// supplied to veri:dawn changes, and the evaluator must explicitly return %.n.
// The ordinary final chain observation still sees the real unchanged point.
func keyVMNegativeArgs(c privateMemberContract, control string) ([]string, error) {
	if control != "wrong-public-key" && control != "wrong-life" {
		return nil, errPrivateMemberKeyProof
	}
	args, err := privateMemberKeyProofArgs(c)
	if err != nil {
		return nil, err
	}
	program := args[8]
	if strings.Count(program, "expression(ship,value.key,point)") != 1 || strings.Count(program, "!=='%.y'") != 1 {
		return nil, errPrivateMemberKeyProof
	}
	program = strings.Replace(program, "expression(ship,value.key,point)", "expression(ship,value.key,qualificationNegativePoint(point))", 1)
	program = strings.Replace(program, "!=='%.y'", "!=='%.n'", 1)
	mutation := "p.result.network.keys.auth='0x'+'0'.repeat(64);"
	if control == "wrong-life" {
		mutation = "p.result.network.keys.life='2';"
	}
	args[8] = "function qualificationNegativePoint(value){const p=structuredClone(value);" + mutation + "return p;}\n" + program
	return args, nil
}
func qualifyKeyVMNegative(ctx context.Context, q *qualifier, c privateMemberContract, control string) (privateMemberKeyProofObservation, error) {
	args, err := keyVMNegativeArgs(c, control)
	if err != nil {
		return privateMemberKeyProofObservation{}, err
	}
	out, err := q.docker(ctx, 90*time.Second, args...)
	if err != nil {
		return privateMemberKeyProofObservation{}, errPrivateMemberKeyProof
	}
	return parsePrivateMemberKeyProof(c, []byte(out))
}

func QualifyPrivateMemberKeys(ctx context.Context, stateRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) (QualificationPrivateMemberKeyResult, error) {
	bad := func() (QualificationPrivateMemberKeyResult, error) {
		return QualificationPrivateMemberKeyResult{}, errPrivateMemberKeyProof
	}
	if err := keyVMQualificationGuard(); err != nil {
		return QualificationPrivateMemberKeyResult{}, err
	}
	if stateRoot != "/home/lab/.local/share/urbit-runner/managed" || g.Validate(p, w) != nil || g.Phase != "ready" {
		return bad()
	}
	source := g.Clone()
	source.Phase = "started"
	source.Observation = nil
	raw, err := OpenPrivateGatewayObserver(ctx, stateRoot, p, w, source, "qualification-member-key-observer")
	if err != nil {
		return bad()
	}
	defer raw.Close()
	session, ok := raw.(*privateSessionObserver)
	if !ok {
		return bad()
	}
	observer, ok := session.GatewayObserver.(*privateGatewayObserver)
	if !ok {
		return bad()
	}
	result := QualificationPrivateMemberKeyResult{Version: 1, PlanDigest: w.PlanDigest, GatewaySHA256: keyVMHash(g), WorkspaceSHA256: keyVMHash(w)}
	for _, m := range p.Members {
		var accepted string
		for _, control := range []string{"valid", "wrong-public-key", "wrong-life"} {
			start := time.Now()
			before, err := observer.ObserveGateway(ctx, source)
			if err != nil || keyVMObservationHash(before) != keyVMObservationHash(*g.Observation) {
				return bad()
			}
			current := g.Clone()
			current.Observation = &before
			runtime := sha256.Sum256([]byte("qualification-member-key/" + m.InstanceID + "/" + control))
			c, err := newPrivateMemberContract(privateMemberSelection{Plan: p, Workspace: w, Gateway: current, InstanceID: m.InstanceID, RuntimeID: hex.EncodeToString(runtime[:16]), OperationID: "qualification-key-" + m.InstanceID + "-" + control, ContainerGeneration: 1, Mode: "boot"}, stateRoot, before.ImageID)
			if err != nil {
				return bad()
			}
			var proof privateMemberKeyProofObservation
			if control == "valid" {
				proof, err = observer.runtime.q.privateMemberKeyProof(ctx, c)
			} else {
				proof, err = qualifyKeyVMNegative(ctx, observer.runtime.q, c, control)
			}
			if err != nil {
				return bad()
			}
			if control == "valid" {
				accepted = proof.KeySHA256
			} else if accepted != proof.KeySHA256 {
				return bad()
			}
			held, err := privatechain.OpenMemberKeyVerifier(ctx, filepath.Join(stateRoot, "private-projects"), p, w, current, m.InstanceID, proof.KeySHA256)
			if err != nil {
				return bad()
			}
			err = held.Verify(ctx)
			closeErr := held.Close()
			if err != nil || closeErr != nil {
				return bad()
			}
			after, err := observer.ObserveGateway(ctx, source)
			if err != nil || keyVMObservationHash(before) != keyVMObservationHash(after) {
				return bad()
			}
			encoded, err := json.Marshal(proof)
			if err != nil {
				return bad()
			}
			result.Cases = append(result.Cases, QualificationPrivateMemberKeyCase{InstanceID: m.InstanceID, Ship: m.Ship, Control: control, Proof: encoded, BeforeSHA256: keyVMObservationHash(before), AfterSHA256: keyVMObservationHash(after), HostKeyVerified: true, Seconds: time.Since(start).Seconds()})
		}
	}
	if len(result.Cases) != 6 || ctx.Err() != nil {
		return bad()
	}
	return result, nil
}
