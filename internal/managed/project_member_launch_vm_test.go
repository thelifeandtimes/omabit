//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// The common admission guard runs before all filesystem and engine effects.
func TestProjectMemberLaunchDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, false, false, false, false, false)
}

type vmMemberLaunchBackend struct {
	appState         bool
	livePeer         bool
	currentContext   bool
	currentLifecycle bool
	anchorReview     bool
	anchorRecovery   bool
	continuation     bool
	publicRecovery   bool
	memberCLI        *vmPublicRecoveryCLI
	*vmAnchorRecoveryBackend
	contextReadiness       *vmCurrentReadinessDriver
	contextReadinessErrors []string
	checkLivePeer          func() error
	appResult              map[string]any
	fakeBackend
	store                  *store
	wrapped                *vmMemberLaunchDriver
	register               func(projectnetwork.MemberLaunch, string)
	errors                 []string
	readiness              *vmMemberReadinessDriver
	readinessErrors        []string
	stop                   *vmMemberStopDriver
	stopErrors             []string
	loseStopReply          bool
	stopExit               *vmMemberStopExitDriver
	stopExitErrors         []string
	retirement             *vmMemberRetirementDriver
	retirementErrors       []string
	currentStop            *vmMemberStopDriver
	currentStopErrors      []string
	resumedReadiness       *vmMemberResumedReadinessDriver
	resumedReadinessErrors []string
	resume                 *vmMemberResumeDriver
	resumeErrors           []string
	resumeRegister         func(projectnetwork.MemberResume, string)
}

func (b *vmMemberLaunchBackend) OpenProjectMemberLaunch(ctx context.Context, selection runtimequalify.ProjectMemberLaunchSelection) (projectnetwork.MemberLaunch, projectnetwork.MemberLaunchDriver, error) {
	state, d, err := runtimequalify.OpenProjectMemberLaunch(ctx, selection)
	if err != nil {
		b.errors = append(b.errors, vmLaunchError(err))
		return state, nil, err
	}
	b.wrapped = &vmMemberLaunchDriver{store: b.store, driver: d, register: b.register}
	return state, b.wrapped, nil
}
func vmLaunchError(err error) string {
	s := err.Error()
	if len(s) > 512 {
		s = s[:512]
	}
	return s
}

type vmMemberLaunchDriver struct {
	store        *store
	driver       projectnetwork.MemberLaunchDriver
	register     func(projectnetwork.MemberLaunch, string)
	calls        []string
	observations []projectnetwork.Observation
	errors       []string
}

func (d *vmMemberLaunchDriver) check(s projectnetwork.MemberLaunch, phase string) error {
	j, err := loadProjectMemberLaunch(d.store.db, s.Storage.InstanceID)
	if err != nil {
		return err
	}
	if j.Launch == nil || s.Phase != phase || !reflect.DeepEqual(*j.Launch, s) {
		return errors.New("member effect preceded committed launch journal")
	}
	if err = validateProjectMemberLaunches(d.store.db); err != nil {
		return err
	}
	d.calls = append(d.calls, phase)
	return nil
}
func (d *vmMemberLaunchDriver) retain(err error) error {
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return err
}
func (d *vmMemberLaunchDriver) VerifyEngine(ctx context.Context, id string) error {
	return d.retain(d.driver.VerifyEngine(ctx, id))
}
func (d *vmMemberLaunchDriver) PrepareWorkspace(ctx context.Context, s projectnetwork.MemberLaunch) (pierstore.DirectoryIdentity, error) {
	if err := d.check(s, "workspace-dispatched"); err != nil {
		return pierstore.DirectoryIdentity{}, d.retain(err)
	}
	id, err := d.driver.PrepareWorkspace(ctx, s)
	return id, d.retain(err)
}
func (d *vmMemberLaunchDriver) Preflight(ctx context.Context, s projectnetwork.MemberLaunch) error {
	if err := d.check(s, s.Phase); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.Preflight(ctx, s))
}
func (d *vmMemberLaunchDriver) Inspect(ctx context.Context, s projectnetwork.MemberLaunch) (projectnetwork.Observation, error) {
	if err := d.check(s, s.Phase); err != nil {
		return projectnetwork.Observation{}, d.retain(err)
	}
	o, err := d.driver.Inspect(ctx, s)
	if err == nil {
		d.observations = append(d.observations, o)
	}
	return o, d.retain(err)
}
func (d *vmMemberLaunchDriver) Create(ctx context.Context, s projectnetwork.MemberLaunch) (string, error) {
	if err := d.check(s, "create-dispatched"); err != nil {
		return "", d.retain(err)
	}
	id, err := d.driver.Create(ctx, s)
	// Even a late client error must retain and clean only the exact returned ID.
	if id != "" {
		d.register(s, id)
	}
	return id, d.retain(err)
}
func (d *vmMemberLaunchDriver) Start(ctx context.Context, s projectnetwork.MemberLaunch) error {
	if err := d.check(s, "start-dispatched"); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.Start(ctx, s))
}

func exerciseVMMemberLaunch(t *testing.T, ctx context.Context, s *store, p storedProject, docker func(...string) json.RawMessage, registerCleanup func(func()), applicationState, livePeer, currentContext, currentLifecycle, anchorReview, anchorRecovery, continuation, publicRecovery, publicMembers bool) []map[string]any {
	t.Helper()
	results := []map[string]any{}
	// Explicit fixture dependency staging; the production launch driver never
	// downloads an artifact or writes the shared cache implicitly.
	meta := protocol.BootPill{SHA256: candidate.PillSHA256, Bytes: candidate.PillBytes}
	cache := bootpill.CachePath("/home/lab/.local/share/urbit-runner/managed", meta)
	if err := os.Mkdir(filepath.Dir(cache), 0700); err != nil {
		t.Error(err)
		return results
	}
	f, err := bootpill.Open("/opt/omabit-lab/runtime/boot.pill")
	if err != nil {
		t.Error(err)
		return results
	}
	err = bootpill.Publish(ctx, f, meta, cache)
	f.Close()
	if err != nil {
		t.Error(err)
		return results
	}
	child, cancel := context.WithCancel(ctx)
	b := &vmMemberLaunchBackend{store: s, appState: applicationState, livePeer: livePeer, currentContext: currentContext, currentLifecycle: currentLifecycle, anchorReview: anchorReview, anchorRecovery: anchorRecovery, continuation: continuation, publicRecovery: publicRecovery}
	r := &Server{s: s, b: b, ctx: child, cancel: cancel, workGate: make(chan struct{}, 1), uid: uint32(os.Geteuid())}
	defer func() { cancel(); r.wg.Wait() }()
	if publicMembers {
		var openErr error
		b.memberCLI, openErr = vmOpenPublicProjectCLI(ctx, r, p.ID, "offline-project-public-members-v1", "members")
		if openErr != nil {
			t.Error(openErr)
			return results
		}
		defer b.memberCLI.close()
	}
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		return results
	}
	// Match the already qualified fake-network boot order. The journal still
	// records exact peers; plan display ordering does not determine boot order.
	members := slices.Clone(p.Members)
	slices.SortStableFunc(members, func(a, b projectMember) int {
		if a.Ship == b.Ship {
			return 0
		}
		if a.Ship == "~zod" {
			return -1
		}
		if b.Ship == "~zod" {
			return 1
		}
		return 0
	})
	for _, member := range members {
		result := map[string]any{"ok": false, "stage": "queue", "instance_id": member.InstanceID, "pill": meta}
		results = append(results, result)
		if b.memberCLI != nil {
			result["public_members"] = b.memberCLI.proof
		}
		b.wrapped = nil
		b.register = func(state projectnetwork.MemberLaunch, id string) {
			registerCleanup(func() { vmStopLaunchedMember(t, s.runnerID, state, id, result, docker) })
		}
		req := projectMemberLaunchRequest{RequestID: "launch-" + member.InstanceID, ExpectedRevision: p.Revision, ReviewDigest: digest}
		op, hit, err := vmQueueMemberAction(r, b, p.ID, member.InstanceID, req, false)
		if err != nil || hit {
			t.Error(op, hit, err)
			return results
		}
		result["operation_id"] = op.ID
		done := make(chan struct{})
		go func() { r.wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-ctx.Done():
			t.Error("member launch worker deadline")
			return results
		}
		result["setup_errors"] = append([]string(nil), b.errors...)
		if b.wrapped != nil {
			result["calls"] = b.wrapped.calls
			result["observations"] = b.wrapped.observations
			result["adapter_errors"] = b.wrapped.errors
		}
		state, err := loadProjectMemberLaunch(s.db, member.InstanceID)
		if err != nil {
			t.Error(err)
			return results
		}
		result["journal"] = state
		receipt, err := loadOperation(s.db, op.ID)
		if err != nil {
			t.Error(err)
			return results
		}
		result["operation"] = receipt
		if state.Launch != nil && state.Launch.Container.ID != "" {
			result["container_inspection"] = docker("container", "inspect", state.Launch.Container.ID)
		}
		if receipt.Status != "succeeded" || state.Launch == nil || state.Launch.Phase != "running" || b.wrapped == nil || len(b.wrapped.observations) != 4 || len(b.wrapped.errors) != 0 {
			t.Error("member launch did not complete", receipt.Status)
			return results
		}
		if !reflect.DeepEqual(b.wrapped.calls, []string{"planned", "workspace-dispatched", "workspace-created", "workspace-created", "create-dispatched", "created", "created", "start-dispatched", "started"}) {
			t.Error("unexpected member effect sequence")
			return results
		}
		repeated, hit, err := vmQueueMemberAction(r, b, p.ID, member.InstanceID, req, true)
		if err != nil || !hit || !reflect.DeepEqual(repeated, receipt) {
			t.Error("member replay changed receipt", err)
			return results
		}
		req.RequestID = "another-launch-" + member.InstanceID
		if _, _, err = vmQueueMemberAction(r, b, p.ID, member.InstanceID, req, false); err == nil {
			t.Error("second member launch redispatched")
			return results
		}
		if r.failed.Load() {
			t.Error("member worker failed runner")
			return results
		}
		result["repeat_dispatch_refused"] = true
		result["stage"] = "complete"
		result["ok"] = true
		// Match the established cold-boot fixture: wait for the exact identity
		// before starting its peer. This does not create a durable ready receipt.
		warmup, cancelWarmup := context.WithTimeout(ctx, 900*time.Second)
		err = vmWaitInitialIdentity(warmup, member, result)
		cancelWarmup()
		if err != nil {
			result["ok"] = false
			t.Error(err)
			return results
		}
	}
	exerciseVMMemberReadiness(t, ctx, r, b, p, results)
	if livePeer && !t.Failed() {
		b.checkLivePeer = vmLivePeerObserver(t, results, docker)
	}
	if applicationState && !t.Failed() {
		vmApplicationSetup(t, ctx, b, p, results, docker)
	}
	if !t.Failed() {
		exerciseVMMemberStop(t, ctx, r, b, p, results)
	}
	if !t.Failed() {
		exerciseVMMemberRetirement(t, ctx, r, b, p, results, docker)
	}
	if !t.Failed() {
		exerciseVMMemberResume(t, ctx, r, b, p, results, docker, registerCleanup)
	}
	if !t.Failed() {
		exerciseVMRepeatedMemberCycle(t, ctx, r, b, p, results, docker, registerCleanup)
	}
	if currentLifecycle && !t.Failed() {
		exerciseVMContextPeerCycle(t, ctx, r, b, p, results, docker, registerCleanup)
	}
	if b.memberCLI != nil && !t.Failed() {
		b.memberCLI.proof["ok"] = true
	}
	return results
}

func vmWaitInitialIdentity(ctx context.Context, member projectMember, result map[string]any) error {
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	started := time.Now()
	attempts, transportErrors := 0, 0
	result["stage"] = "fixture-identity-warmup"
	defer func() { result["fixture_identity_seconds"] = time.Since(started).Seconds() }()
	for {
		attempts++
		result["fixture_identity_attempts"] = attempts
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/~/host", member.HTTPPort), nil)
		if err != nil {
			return errors.New("fixture identity request invalid")
		}
		response, err := client.Do(request)
		if err == nil {
			result["fixture_identity_last_http_status"] = response.StatusCode
			body, readErr := io.ReadAll(io.LimitReader(response.Body, 129))
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				if readErr != nil || len(body) > 128 || strings.TrimSpace(string(body)) != member.Ship {
					return errors.New("fixture identity endpoint returned an unexpected identity")
				}
				result["fixture_identity_http_status"] = 200
				result["fixture_identity_ship"] = member.Ship
				result["stage"] = "complete"
				return nil
			}
		} else {
			transportErrors++
			result["fixture_identity_transport_errors"] = transportErrors
		}
		select {
		case <-ctx.Done():
			return errors.New("fixture identity listener did not become ready within its cold-boot budget")
		case <-time.After(time.Second):
		}
	}
}

func vmStopLaunchedMember(t *testing.T, runnerID string, state projectnetwork.MemberLaunch, id string, result map[string]any, docker func(...string) json.RawMessage) {
	t.Helper()
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(id) {
		t.Error("invalid returned member cleanup ID")
		return
	}
	if retired, ok := result["retirement"].(map[string]any); ok && retired["ok"] == true {
		j, valid := retired["journal"].(projectMemberRetirementJob)
		if !valid || j.ContainerID != id || j.InstanceID != state.Storage.InstanceID || j.Retirement.Phase != "absent" || j.Retirement.After == nil || !j.Retirement.Exit.Clean() {
			result["ok"] = false
			t.Error("retired member cleanup lacks exact retained evidence")
			return
		}
		raw := docker("container", "ls", "--all", "--quiet", "--no-trunc")
		if raw == nil {
			result["ok"] = false
			return
		}
		ids := strings.Fields(string(raw))
		for _, found := range ids {
			if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(found) || found == id {
				result["ok"] = false
				t.Error("retired member still present or incomplete inventory")
				return
			}
		}
		result["retirement_cleanup_inventory"] = ids
		result["retired_container_absent"] = true
		result["clean_member_stop"] = true // Historical clean exit, separately verified before removal.
		return
	}
	raw := docker("container", "inspect", id)
	result["final_inspection_before_stop"] = raw
	if result["fixture_cleanup_only"] == true {
		original, ok := result["container_inspection"].(json.RawMessage)
		if !ok || vmCheckUnchangedLivePeer(original, raw) != nil {
			result["ok"] = false
			t.Error("running peer changed before final fixture cleanup")
			return
		}
		result["live_peer_final_identity_unchanged"] = true
	}
	var values []struct {
		ID     string `json:"Id"`
		Name   string
		Config struct{ Labels map[string]string }
		State  struct {
			Running, OOMKilled bool
			ExitCode           int
		}
	}
	if json.Unmarshal(raw, &values) != nil || len(values) != 1 {
		result["ok"] = false
		t.Error("cannot inspect fresh member cleanup")
		return
	}
	v := values[0]
	label := "org.urbit-runner.project"
	want := map[string]string{label: "true", label + ".runner": runnerID, label + ".project": state.Storage.ProjectID, label + ".generation": "1", label + ".instance": state.Storage.InstanceID, label + ".runtime": state.Storage.RuntimeID, label + ".operation": state.OperationID, label + ".container-generation": "1"}
	if v.ID != id || v.Name != "/urbit-project-member-"+state.Storage.RuntimeID+"-g1" || !reflect.DeepEqual(v.Config.Labels, want) {
		result["ok"] = false
		t.Error("fresh member cleanup ownership mismatch")
		return
	}
	if v.State.Running {
		docker("container", "stop", "--timeout", "300", id)
	}
	raw = docker("container", "inspect", id)
	result["cleanup_inspection"] = raw
	if json.Unmarshal(raw, &values) != nil || len(values) != 1 || values[0].ID != id || values[0].State.Running || values[0].State.OOMKilled || values[0].State.ExitCode != 0 {
		result["ok"] = false
		t.Error("fresh member did not stop cleanly")
		return
	}
	result["clean_member_stop"] = true
}
