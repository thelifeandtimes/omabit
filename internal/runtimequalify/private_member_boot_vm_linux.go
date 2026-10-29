//go:build linux && amd64 && omabit_private_member_boot_vm && omabit_private_member_key_vm

package runtimequalify

// Qualification-only bridge. Ordinary builds have no entry point for this
// fixture's HTTP observations or one-shot Lens shutdown. The managed launch is
// performed separately through the normal internal runner queue.
import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

var errMemberBootVM = errors.New("managed member boot qualification refused")

type QualificationPrivateMemberBootResult struct {
	Version            int                              `json:"version"`
	Phase              string                           `json:"phase"`
	LaunchSHA256       string                           `json:"launch_sha256"`
	ContainerID        string                           `json:"container_id"`
	Ship               string                           `json:"ship"`
	PublicHTTPIdentity bool                             `json:"public_http_identity"`
	LensIdentity       bool                             `json:"lens_identity"`
	GatewayBefore      *privatechain.GatewayObservation `json:"gateway_before,omitempty"`
	GatewayAfter       *privatechain.GatewayObservation `json:"gateway_after,omitempty"`
	StopDispatched     bool                             `json:"stop_dispatched"`
	StopAcknowledged   bool                             `json:"stop_acknowledged"`
	ExitCode           int                              `json:"exit_code"`
	FinishedAt         time.Time                        `json:"finished_at"`
	StoppedMetadata    bool                             `json:"stopped_metadata"`
}

func memberBootVMGuard() error {
	if keyVMQualificationGuard() != nil || os.Getenv("OMABIT_PRIVATE_MEMBER_BOOT") != "first-v1" {
		return errMemberBootVM
	}
	return nil
}

// The caller holds the management fence and saves each receipt exclusively and
// durably. No failed or uncertain shutdown is retried. The returned receipt is
// fixture evidence, not a managed Stop record or application-readiness grant.
func QualifyPrivateMemberBoot(ctx context.Context, stateRoot string, p privatechain.Plan, w privatechain.Workspace, s privatechain.MemberLaunch, save func(QualificationPrivateMemberBootResult) error) (QualificationPrivateMemberBootResult, error) {
	result := QualificationPrivateMemberBootResult{}
	bad := func() (QualificationPrivateMemberBootResult, error) { return result, errMemberBootVM }
	if memberBootVMGuard() != nil || stateRoot != "/home/lab/.local/share/urbit-runner/managed" || save == nil || s.Validate(p, w) != nil || s.Version != 3 || s.Phase != "running" || len(s.Peers) != 0 || s.Running == nil || s.Running.Birth == nil {
		return bad()
	}
	q, err := privateGatewaySession(ctx, stateRoot, defaults())
	if err != nil {
		return bad()
	}
	client, err := openPrivateGatewayClient(ctx, q, s.Gateway, false)
	if err != nil {
		return bad()
	}
	defer client.Close()
	if q.inspectImage(ctx) != nil {
		return bad()
	}
	c, err := newPrivateMemberContract(privateMemberSelection{Plan: p, Workspace: w, Gateway: s.Gateway, InstanceID: s.InstanceID, RuntimeID: s.RuntimeID, OperationID: s.OperationID, ContainerGeneration: 1, Mode: "boot"}, stateRoot, q.r.ImageID)
	if err != nil || c.digest != s.ContractDigest {
		return bad()
	}
	pier, err := privatechain.OpenMemberPierVerifier(ctx, filepath.Join(stateRoot, "private-projects"), p, w, s.InstanceID, s.Running.Birth.Pier)
	if err != nil {
		return bad()
	}
	defer pier.Close()
	pill, err := openPrivateMemberPill(ctx, c, s.Artifact)
	if err != nil {
		return bad()
	}
	defer pill.Close()
	key, err := privatechain.OpenMemberKeyVerifier(ctx, filepath.Join(stateRoot, "private-projects"), p, w, s.Gateway, s.InstanceID, s.Key.KeySHA256)
	if err != nil {
		return bad()
	}
	defer key.Close()
	var member privatechain.Member
	for _, m := range p.Members {
		if m.InstanceID == s.InstanceID {
			member = m
		}
	}
	inputRoot, err := privateGatewayInputPath(stateRoot, s.Gateway.Selection.Inputs.ManifestSHA256)
	if err != nil {
		return bad()
	}
	verify := func() error {
		if pier.Verify(ctx) != nil || pill.Verify(ctx) != nil || key.Verify(ctx) != nil {
			return errMemberBootVM
		}
		got, e := observePrivateMemberProcess(ctx, q, c, s.ContainerID, true, time.Now)
		if e != nil || !got.StartedAt.Equal(s.Running.Birth.After.StartedAt) || got.Namespace != s.Running.Birth.After.Namespace {
			return errMemberBootVM
		}
		return nil
	}
	gateway := func(state string) (privatechain.GatewayObservation, error) {
		members := []privateGatewayMember{{Contract: c, ContainerID: s.ContainerID, State: state, StartedAt: s.Running.Birth.After.StartedAt}}
		piers := []privatechain.GatewayMemberPier{{InstanceID: s.InstanceID, Identity: s.Running.Birth.Pier}}
		o, e := openPrivateMemberGatewayObserver(ctx, q, stateRoot, inputRoot, p, w, s.Gateway, members, piers)
		if e != nil {
			return privatechain.GatewayObservation{}, e
		}
		defer o.Close()
		got, e := o.ObserveGateway(ctx, s.Gateway)
		if e != nil {
			return got, e
		}
		if e = checkPrivateMemberArtifactMounts(ctx, o.runtime.mountInventory, c, members); e != nil {
			return got, e
		}
		want := s.Running.Gateway.Clone()
		copy := got.Clone()
		want.ObservedAt = time.Time{}
		copy.ObservedAt = time.Time{}
		if !reflect.DeepEqual(want, copy) {
			return got, errMemberBootVM
		}
		return got, nil
	}
	result = QualificationPrivateMemberBootResult{Version: 1, Phase: "observing", LaunchSHA256: keyVMHash(s), ContainerID: s.ContainerID, Ship: member.Ship}
	deadline := time.NewTimer(15 * time.Minute)
	defer deadline.Stop()
	for {
		if verify() != nil {
			return bad()
		}
		identity, e := readHostIdentity(ctx, "http://127.0.0.1:"+strconv.Itoa(int(member.HTTPPort))+"/~/host")
		if e == nil {
			if identity != member.Ship {
				return bad()
			}
			break
		}
		if !errors.Is(e, errHostNotReady) {
			return bad()
		}
		select {
		case <-ctx.Done():
			return bad()
		case <-deadline.C:
			return bad()
		case <-time.After(3 * time.Second):
		}
	}
	result.PublicHTTPIdentity = true
	ports, err := q.docker(ctx, 5*time.Second, "exec", s.ContainerID, "/bin/cat", "/urbit/"+strings.TrimPrefix(member.Ship, "~")+"/.http.ports")
	if err != nil {
		return bad()
	}
	admin, err := privateHTTPPort(ports, member.InternalHTTPPort)
	if err != nil {
		return bad()
	}
	lens := func(formula, app string) (string, error) {
		var sink any = map[string]any{"stdout": nil}
		if app != "" {
			sink = map[string]any{"app": app}
		}
		payload, _ := json.Marshal(map[string]any{"source": map[string]string{"dojo": formula}, "sink": sink})
		out, e := q.docker(ctx, 25*time.Second, "exec", s.ContainerID, "/usr/bin/curl", "--disable", "--noproxy", "*", "--proxy", "", "--silent", "--show-error", "--fail", "--max-time", "20", "--max-filesize", "1024", "--header", "Content-Type: application/json", "--data-binary", string(payload), "http://127.0.0.1:"+strconv.Itoa(int(admin)))
		var value string
		if e != nil || len(out) > 1024 || json.Unmarshal([]byte(out), &value) != nil {
			return "", errMemberBootVM
		}
		return strings.TrimSpace(value), nil
	}
	identity, err := lens("our", "")
	if err != nil || identity != member.Ship || verify() != nil {
		return bad()
	}
	result.LensIdentity = true
	before, err := gateway("running")
	if err != nil {
		return bad()
	}
	result.GatewayBefore = &before
	result.Phase = "ready"
	if save(result) != nil {
		return bad()
	}
	// The public and administrative endpoints are identity checked again directly
	// before dispatch. Only this fixed exit formula is allowed to mutate the ship.
	current, err := q.docker(ctx, 5*time.Second, "exec", s.ContainerID, "/bin/cat", "/urbit/"+strings.TrimPrefix(member.Ship, "~")+"/.http.ports")
	if err != nil || current != ports || verify() != nil {
		return bad()
	}
	identity, err = lens("our", "")
	if err != nil || identity != member.Ship {
		return bad()
	}
	identity, err = readHostIdentity(ctx, "http://127.0.0.1:"+strconv.Itoa(int(member.HTTPPort))+"/~/host")
	if err != nil || identity != member.Ship {
		return bad()
	}
	result.Phase = "stop-dispatched"
	result.StopDispatched = true
	if save(result) != nil {
		return bad()
	}
	ack, err := lens("+hood/exit", "hood")
	if err != nil || ack != ">=" {
		return bad()
	}
	result.Phase = "stop-acknowledged"
	result.StopAcknowledged = true
	if save(result) != nil {
		return bad()
	}
	exit, err := q.docker(ctx, 90*time.Second, "container", "wait", s.ContainerID)
	if err != nil || strings.TrimSpace(exit) != "0" {
		return bad()
	}
	raw, err := q.docker(ctx, 10*time.Second, "container", "inspect", s.ContainerID)
	if err != nil {
		return bad()
	}
	var rows []container
	if json.Unmarshal([]byte(raw), &rows) != nil || len(rows) != 1 || c.validateExitedContainer(rows[0], s.ContainerID, s.Running.Birth.After.StartedAt) != nil || *rows[0].State.OOMKilled || *rows[0].State.ExitCode != 0 {
		return bad()
	}
	result.FinishedAt, err = time.Parse(time.RFC3339Nano, rows[0].State.FinishedAt)
	if err != nil {
		return bad()
	}
	if pier.VerifyStoppedState(ctx) != nil || pill.Verify(ctx) != nil || key.Verify(ctx) != nil {
		return bad()
	}
	result.StoppedMetadata = true
	after, err := gateway("exited")
	if err != nil {
		return bad()
	}
	result.GatewayAfter = &after
	result.Phase = "stopped"
	if save(result) != nil {
		return bad()
	}
	return result, nil
}
