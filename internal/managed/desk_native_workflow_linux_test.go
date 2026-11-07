//go:build linux && amd64

package managed

// Explicit disposable native qualification, never part of ordinary test runs.
// The managed lifecycle backend is a fixture; the kernel, typed desk requests,
// collector/Unix transport, policy store, staging and operation journal are real.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type nativeDeskFixture struct {
	PID              int    `json:"pid"`
	Start            string `json:"start"`
	Desk             string `json:"desk"`
	VereSHA          string `json:"vere_sha256"`
	ShipName         string `json:"ship_name,omitempty"`
	LensPort         int    `json:"lens_port,omitempty"`
	HTTPPort         int    `json:"http_port,omitempty"`
	ledger           string
	client           *http.Client
	creates, commits atomic.Int32
	effects          atomic.Int32
	loseReply        atomic.Bool
}

func nativeDeskGuard(t *testing.T) *nativeDeskFixture {
	t.Helper()
	if _, err := os.Stat("/lab/workflow-runtime.json"); os.IsNotExist(err) {
		t.Skip("explicit private native workflow capsule required")
	}
	var guard struct {
		Namespaces map[string]string `json:"namespaces"`
		Unit       string            `json:"unit"`
		Workflow   bool              `json:"workflow"`
	}
	data, err := os.ReadFile("/guard.json")
	g02Check(t, err)
	g02Check(t, json.Unmarshal(data, &guard))
	if !guard.Workflow || !strings.HasPrefix(guard.Unit, "omabit-desk-probe-") {
		t.Fatal("invalid capsule")
	}
	for _, key := range []string{"net", "pid", "mnt", "user"} {
		got, err := os.Readlink("/proc/self/ns/" + key)
		g02Check(t, err)
		if guard.Namespaces[key] == "" || got == guard.Namespaces[key] {
			t.Fatal("host namespace refused", key)
		}
	}
	for _, path := range []string{"/home", "/run/user", "/var/run/docker.sock"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("host access refused", path)
		}
	}
	links, err := net.Interfaces()
	g02Check(t, err)
	if len(links) != 1 || links[0].Name != "lo" {
		t.Fatal("external network refused")
	}
	data, err = os.ReadFile("/lab/admitted")
	g02Check(t, err)
	if string(data) != guard.Unit {
		t.Fatal("scope admission absent")
	}
	f := new(nativeDeskFixture)
	data, err = os.ReadFile("/lab/workflow-runtime.json")
	g02Check(t, err)
	g02Check(t, json.Unmarshal(data, f))
	if !strings.HasPrefix(f.Desk, "omabit-test-") || !deskwork.ValidSourceName(f.Desk) || f.VereSHA != "b5c54439ae5b9f0ff94d75319e67e04f04369776fa5e2180a5ab1a5ebd4c2ff0" {
		t.Fatal("unexpected fixture identity")
	}
	f.client = &http.Client{Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") }}
	g02Check(t, f.check())
	return f
}
func (f *nativeDeskFixture) ship() string {
	if f.ShipName != "" {
		return f.ShipName
	}
	return "~zod"
}
func (f *nativeDeskFixture) lensPort() int {
	if f.LensPort != 0 {
		return f.LensPort
	}
	return 12321
}
func (f *nativeDeskFixture) pier() string { return "/lab/" + strings.TrimPrefix(f.ship(), "~") }
func (f *nativeDeskFixture) check() error {
	if f.PID < 2 {
		return errors.New("owned child absent")
	}
	path := "/proc/" + strconv.Itoa(f.PID)
	stat, err := os.ReadFile(path + "/stat")
	if err != nil {
		return err
	}
	fields := strings.Fields(string(stat)[strings.LastIndex(string(stat), ")")+2:])
	if len(fields) < 20 || fields[19] != f.Start {
		return errors.New("runtime process changed")
	}
	exe, err := os.Readlink(path + "/exe")
	if err != nil || exe != "/artifacts/vere" {
		return errors.New("unexpected owned executable")
	}
	file, err := os.Open(path + "/exe")
	if err != nil {
		return err
	}
	defer file.Close()
	h := sha256.New()
	if _, err = io.Copy(h, file); err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != f.VereSHA {
		return errors.New("runtime artifact changed")
	}
	return nil
}
func (f *nativeDeskFixture) post(ctx context.Context, body []byte) (string, error) {
	if err := f.check(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://127.0.0.1:%d/", f.lensPort()), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 || len(data) > 1<<20 {
		data, _ := json.MarshalIndent(map[string]any{"status": response.StatusCode, "body": string(data)}, "", "  ")
		_ = os.WriteFile("/lab/last-failed-lens-transport.json", data, 0600)
		return "", errors.New("invalid Lens response")
	}
	var value string
	if err = json.Unmarshal(data, &value); err != nil {
		return "", err
	}
	return value, f.check()
}
func (f *nativeDeskFixture) call(ctx context.Context, q deskwork.LensRequest, err error) (string, error) {
	if err != nil {
		return "", err
	}
	body, err := q.Body()
	if err != nil {
		return "", err
	}
	identity := []byte(`{"source":{"dojo":"our"},"sink":{"stdout":null}}`)
	before, err := f.post(ctx, identity)
	if err != nil || strings.TrimSpace(before) != f.ship() {
		return "", fmt.Errorf("fixture identity before request: %q %w", before, err)
	}
	out, err := f.post(ctx, body)
	if err != nil {
		return "", err
	}
	after, err := f.post(ctx, identity)
	if err != nil || strings.TrimSpace(after) != f.ship() {
		return "", errors.New("fixture identity after request")
	}
	return out, nil
}
func (f *nativeDeskFixture) effect(ctx context.Context, q deskwork.LensRequest, err error) error {
	if err != nil {
		return err
	}
	id := f.effects.Add(1)
	record := func(state string) error { return f.recordEffect(id, state) }
	if err = record("dispatched"); err != nil {
		return err
	}
	out, err := f.call(ctx, q, nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != ">=" {
		body, _ := q.Body()
		data, _ := json.MarshalIndent(map[string]any{"request": json.RawMessage(body), "response": out}, "", "  ")
		_ = os.WriteFile("/lab/last-failed-desk-effect.json", data, 0600)
		return fmt.Errorf("effect unacknowledged: %s", out)
	}
	return record("acknowledged")
}
func (f *nativeDeskFixture) decode(ctx context.Context, q deskwork.LensRequest, err error, out any) error {
	reply, err := f.call(ctx, q, err)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(reply)
	if err = deskwork.DecodeJSONOutput(body, out); err != nil {
		request, _ := q.Body()
		data, _ := json.MarshalIndent(map[string]any{"request": json.RawMessage(request), "response": reply, "error": err.Error()}, "", "  ")
		_ = os.WriteFile("/lab/last-failed-desk-read.json", data, 0600)
		return fmt.Errorf("%w: %s", err, reply)
	}
	return nil
}
func (f *nativeDeskFixture) Application(ctx context.Context, desk string) (deskwork.AppState, error) {
	var out deskwork.AppState
	q, err := deskwork.AppInventory(f.ship(), desk)
	err = f.decode(ctx, q, err, &out)
	if err == nil {
		err = out.Normalize()
	}
	return out, err
}
func (f *nativeDeskFixture) Observe(ctx context.Context, desk string) (deskwork.Observation, error) {
	var out deskwork.Observation
	q, err := deskwork.Inventory(f.ship(), desk)
	err = f.decode(ctx, q, err, &out)
	if err == nil {
		err = out.Validate()
	}
	return out, err
}
func (f *nativeDeskFixture) Create(ctx context.Context, desk string) error {
	f.creates.Add(1)
	q, err := deskwork.Create(f.ship(), desk)
	return f.effect(ctx, q, err)
}
func (f *nativeDeskFixture) ListDesks(ctx context.Context) ([]string, error) {
	var names []string
	q, err := deskwork.ListDesks(f.ship())
	if err = f.decode(ctx, q, err, &names); err != nil {
		return nil, err
	}
	return deskwork.NormalizeDeskNames(names)
}
func (f *nativeDeskFixture) SourceIdentity(context.Context) (string, error) {
	if err := f.check(); err != nil {
		return "", err
	}
	i, err := pierstore.Identify(f.pier())
	if err != nil {
		return "", err
	}
	if i.Birth != "" && i.UrbBirth != "" {
		i.Device = 0
		i.UrbDevice = 0
	}
	return pierstore.Fingerprint(i), nil
}
func (f *nativeDeskFixture) canonical(ctx context.Context, q deskwork.LensRequest, err error) (deskwork.Manifest, error) {
	var files []deskwork.CanonicalFile
	if err = f.decode(ctx, q, err, &files); err != nil {
		return deskwork.Manifest{}, err
	}
	return deskwork.CanonicalManifest(files)
}
func (f *nativeDeskFixture) ClayFiles(ctx context.Context, desk string, rev uint64) (deskwork.Manifest, error) {
	q, err := deskwork.ClayFiles(f.ship(), desk, rev)
	return f.canonical(ctx, q, err)
}
func (f *nativeDeskFixture) NormalizeFiles(ctx context.Context, desk string, rev uint64, files []deskwork.SourceFile) (deskwork.Manifest, error) {
	q, err := deskwork.NormalizeFiles(f.ship(), desk, rev, files)
	return f.canonical(ctx, q, err)
}
func (f *nativeDeskFixture) AutocommitActive(ctx context.Context) (bool, error) {
	q, err := deskwork.AutocommitActive(f.ship())
	reply, err := f.call(ctx, q, err)
	if err != nil {
		return false, err
	}
	switch strings.TrimSpace(reply) {
	case "%.y":
		return true, nil
	case "%.n":
		return false, nil
	}
	return false, errors.New("autocommit unsupported")
}
func (f *nativeDeskFixture) InspectSource(ctx context.Context, desk, source string, base *deskwork.Manifest) (deskwork.SourceStatus, error) {
	if !deskwork.ValidSourceName(source) {
		return deskwork.SourceStatus{}, errors.New("invalid source")
	}
	return deskwork.InspectSource(ctx, f, filepath.Join(f.pier(), source), desk, base)
}
func (f *nativeDeskFixture) Release(ctx context.Context, desk, action string, baseline deskwork.Observation) error {
	var q deskwork.LensRequest
	var err error
	switch action {
	case "activate-local":
		q, err = deskwork.ActivateLocal(f.ship(), desk, baseline)
	case "publish-source":
		q, err = deskwork.PublishSourceAt(f.ship(), desk, baseline)
	case "publish-treaty":
		q, err = deskwork.PublishTreatyAt(f.ship(), desk, baseline)
	default:
		return errors.New("unsupported release action")
	}
	return f.effect(ctx, q, err)
}
func (f *nativeDeskFixture) CommitFiles(ctx context.Context, desk string, baseline deskwork.Observation, files []deskwork.DeltaFile) error {
	f.commits.Add(1)
	q, err := deskwork.CommitFiles(f.ship(), desk, baseline, files)
	err = f.effect(ctx, q, err)
	if err == nil && f.loseReply.Swap(false) {
		return errors.New("qualification: discard real acknowledgement")
	}
	return err
}

type nativeDeskBackend struct {
	*deskBackendFixture
	runtime *nativeDeskFixture
}

func (b *nativeDeskBackend) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	return b.runtime, b.runtime.check()
}

func TestNativeDeskWorkflow(t *testing.T) {
	f := nativeDeskGuard(t)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	g02Check(t, os.Mkdir("/lab/runner", 0700))
	s, err := testOpenStore("/lab/runner")
	g02Check(t, err)
	defer s.close()
	_, _, err = s.register("fixture", registration("alpha", "native-workflow", 18081))
	g02Check(t, err)
	instance, err := loadInstance(s.db, "alpha")
	g02Check(t, err)
	instance.Public.ObservedState = "running"
	instance.Public.Resources = &protocol.ResourceMetrics{ContainerID: strings.Repeat("b", 64)}
	saveRecoveryFixture(t, s, instance)
	r := &Server{s: s, b: &nativeDeskBackend{&deskBackendFixture{}, f}, ctx: ctx, cancel: cancel, uid: uint32(os.Geteuid())}
	defer r.Close()
	socket := "/lab/runner.sock"
	listener, err := net.Listen("unix", socket)
	g02Check(t, err)
	g02Check(t, os.Chmod(socket, 0600))
	server := &http.Server{Handler: r, ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
		return context.WithValue(ctx, peerKey{}, uint32(os.Geteuid()))
	}}
	go server.Serve(listener)
	defer server.Close()
	registry, err := collection.Open("/lab/collector")
	g02Check(t, err)
	entry, err := registry.Enroll(ctx, "native", socket)
	g02Check(t, err)
	proof := map[string]any{"boundary": "real native kernel, API/store/collector; fixture lifecycle backend"}
	save := func(name string, value any) {
		proof[name] = value
		data, err := json.MarshalIndent(proof, "", "  ")
		g02Check(t, err)
		g02Check(t, os.WriteFile("/lab/workflow-proof.json", data, 0600))
		t.Log(name)
	}
	wait := func(id, want string) deskwork.Receipt {
		t.Helper()
		r.wg.Wait()
		v, err := registry.DeskOperation(ctx, "native", entry.RunnerID, id)
		g02Check(t, err)
		if got := v.Receipt.Steps[len(v.Receipt.Steps)-1].State; got != want {
			save("unexpected-operation-"+id, v.Receipt)
			t.Fatalf("operation %s: want %s got %s: %+v", id, want, got, v.Receipt)
		}
		return v.Receipt
	}
	create := func(want string) {
		t.Helper()
		plan, err := registry.PlanDeskCreation(ctx, "native", entry.RunnerID, "alpha", f.Desk)
		g02Check(t, err)
		_, err = registry.CreateDesk(ctx, "native", entry.RunnerID, plan)
		g02Check(t, err)
		save("creation-"+want, wait(plan.Intent.ID, want))
	}
	create("blocked")
	if f.creates.Load() != 0 {
		t.Fatal("default gate dispatched")
	}
	target := apppolicy.Target{InstanceID: "alpha", Desk: f.Desk}
	policy, err := s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "ship", Target: target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Inherit, Publish: apppolicy.Inherit}})
	g02Check(t, err)
	create("observed")
	if f.creates.Load() != 1 {
		t.Fatal("creation count")
	}
	inventory, err := registry.DiscoverDesks(ctx, "native", entry.RunnerID, "alpha")
	g02Check(t, err)
	if !slices.Contains(inventory.Desks, f.Desk) {
		t.Fatal("new desk missing")
	}
	save("discovery", inventory)
	q, err := deskwork.Mount("~zod", f.Desk, f.Desk, 0)
	g02Check(t, f.effect(ctx, q, err))
	source := filepath.Join("/lab/zod", f.Desk)
	for end := time.Now().Add(20 * time.Second); ; {
		if _, err := os.Stat(filepath.Join(source, "sys.kelvin")); err == nil {
			break
		}
		if time.Now().After(end) {
			t.Fatal("mount missing")
		}
		time.Sleep(100 * time.Millisecond)
	}
	status := func(name, want string) {
		t.Helper()
		v, err := registry.InspectDeskSource(ctx, "native", entry.RunnerID, "alpha", f.Desk)
		g02Check(t, err)
		if v.State != want || v.Autocommit != "inactive" {
			t.Fatalf("%s: %+v", name, v)
		}
		save(name, v)
	}
	status("initial-source", "synchronized")
	// Explicitly select one existing scaffold file; the rest must stay unowned.
	adoptionDir := "/lab/adoption-source"
	g02Check(t, os.Mkdir(adoptionDir, 0700))
	kelvin, err := os.ReadFile(filepath.Join(source, "sys.kelvin"))
	g02Check(t, err)
	g02Check(t, os.WriteFile(filepath.Join(adoptionDir, "sys.kelvin"), kelvin, 0600))
	adoptionStage, err := registry.PlanDeskStage(ctx, "native", entry.RunnerID, "alpha", f.Desk, adoptionDir)
	g02Check(t, err)
	_, err = registry.StageDesk(ctx, "native", entry.RunnerID, adoptionStage, adoptionDir)
	g02Check(t, err)
	adopt := func(want string) collection.DeskAdoptionPlan {
		t.Helper()
		plan, err := registry.PlanDeskAdoption(ctx, "native", entry.RunnerID, adoptionStage.Intent.ID)
		g02Check(t, err)
		if plan.Review.NewCount != 1 || plan.Review.RefreshCount != 0 || plan.Review.ConflictCount != 0 || !slices.Equal(plan.Review.Paths, []string{"sys.kelvin"}) {
			t.Fatal("adoption selection changed", plan)
		}
		before, err := f.Observe(ctx, f.Desk)
		g02Check(t, err)
		result, err := registry.AdoptDeskSource(ctx, "native", entry.RunnerID, plan)
		g02Check(t, err)
		after, err := f.Observe(ctx, f.Desk)
		g02Check(t, err)
		if result.Receipt.Steps[len(result.Receipt.Steps)-1].State != want || before != after || f.commits.Load() != 0 {
			t.Fatal("adoption gate or unexpected native commit", result)
		}
		save("source-adoption-"+want, result)
		save("source-adoption-"+want+"-review", plan)
		return plan
	}
	adopt("blocked")
	write := func(name, value string) {
		t.Helper()
		g02Check(t, os.WriteFile(filepath.Join(source, name), []byte(value), 0600))
	}
	write("hello.txt", "first source\n")
	write("remove.txt", "temporary\n")
	status("uncommitted-source", "unix-changes")
	stage := func() collection.DeskStagePlan {
		t.Helper()
		plan, err := registry.PlanDeskStage(ctx, "native", entry.RunnerID, "alpha", f.Desk, source)
		g02Check(t, err)
		receipt, err := registry.StageDesk(ctx, "native", entry.RunnerID, plan, source)
		g02Check(t, err)
		save("stage-"+plan.Intent.ID, receipt)
		return plan
	}
	first := stage()
	commit := func(staged collection.DeskStagePlan, name, want string) collection.DeskCommitPlan {
		t.Helper()
		plan, err := registry.PlanDeskCommit(ctx, "native", entry.RunnerID, staged.Intent.ID)
		g02Check(t, err)
		save(name+"-review", plan)
		_, err = registry.CommitDesk(ctx, "native", entry.RunnerID, plan)
		g02Check(t, err)
		save(name, wait(plan.Intent.Intent.ID, want))
		return plan
	}
	commit(first, "default-commit", "blocked")
	if f.commits.Load() != 0 {
		t.Fatal("closed commit gate dispatched")
	}
	policy, err = s.putAppPolicy("fixture-user", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}})
	g02Check(t, err)
	adopted := adopt("observed")
	_, err = registry.DiscardDeskStage(ctx, "native", entry.RunnerID, adoptionStage.Intent.ID, adoptionStage.Intent.SourceDigest)
	g02Check(t, err)
	_, err = registry.AdoptDeskSource(ctx, "native", entry.RunnerID, adopted)
	g02Check(t, err)
	firstCommit := commit(first, "first-commit", "observed")
	if firstCommit.Review.ChangeCount != 2 || firstCommit.Review.PreservedCount < 4 {
		t.Fatal("scaffold preservation", firstCommit.Review)
	}
	status("committed-source", "synchronized")
	write("hello.txt", "second source\n")
	write("added.txt", "new source\n")
	g02Check(t, os.Remove(filepath.Join(source, "remove.txt")))
	status("second-uncommitted-source", "unix-changes")
	second := stage()
	f.loseReply.Store(true)
	secondCommit := commit(second, "lost-acknowledgement", "uncertain")
	if secondCommit.Review.ChangeCount != 3 {
		t.Fatal("update/add/delete delta missing")
	}
	if _, err = registry.DiscardDeskStage(ctx, "native", entry.RunnerID, second.Intent.ID, second.Intent.SourceDigest); err == nil {
		t.Fatal("uncertain source removed")
	}
	policy, err = s.putAppPolicy("fixture-user", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Block, Publish: apppolicy.Block}})
	g02Check(t, err)
	_, err = registry.ReconcileDeskOperation(ctx, "native", entry.RunnerID, secondCommit.Intent.Intent.ID)
	g02Check(t, err)
	save("reconciled-with-gate-closed", wait(secondCommit.Intent.Intent.ID, "observed"))
	if f.commits.Load() != 2 {
		t.Fatal("reconciliation repeated commit")
	}
	status("reconciled-source", "synchronized")
	_, err = registry.CommitDesk(ctx, "native", entry.RunnerID, secondCommit)
	g02Check(t, err)
	if f.commits.Load() != 2 {
		t.Fatal("same ID replay repeated commit")
	}
	_, err = registry.DiscardDeskStage(ctx, "native", entry.RunnerID, second.Intent.ID, second.Intent.SourceDigest)
	g02Check(t, err)
	setPublication := func(method string) {
		t.Helper()
		setting, err := s.deskSetting(target)
		g02Check(t, err)
		setting.Desk.Publication = method
		_, err = s.putDeskSetting("fixture-user", deskwork.SettingUpdate{Target: target, ExpectedRevision: setting.Revision, Desk: setting.Desk})
		g02Check(t, err)
	}
	release := func(action, name, want string) collection.DeskReleasePlan {
		t.Helper()
		plan, err := registry.PlanDeskRelease(ctx, "native", entry.RunnerID, "alpha", f.Desk, action, []deskwork.ReleaseCheck{})
		g02Check(t, err)
		save(name+"-plan", plan)
		_, err = registry.ReleaseDesk(ctx, "native", entry.RunnerID, plan)
		g02Check(t, err)
		save(name, wait(plan.Intent.Intent.ID, want))
		return plan
	}
	setPublication("source")
	effectsBefore := f.effects.Load()
	release("publish-source", "source-publication-default-blocked", "blocked")
	if f.effects.Load() != effectsBefore {
		t.Fatal("closed publication gate dispatched")
	}
	policy, err = s.putAppPolicy("fixture-user", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Block, Publish: apppolicy.Allow}})
	g02Check(t, err)
	publishedPlan := release("publish-source", "source-publication-observed", "observed")
	_, err = registry.ReleaseDesk(ctx, "native", entry.RunnerID, publishedPlan)
	g02Check(t, err)
	if f.effects.Load() != effectsBefore+1 {
		t.Fatal("publication replay repeated effect")
	}
	published, err := f.Observe(ctx, f.Desk)
	g02Check(t, err)
	if published.Publication != "published" {
		t.Fatal("source publication not observed")
	}
	save("native-source-publication", published)
	write("hello.txt", "public update\n")
	third := stage()
	policy, err = s.putAppPolicy("fixture-user", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}})
	g02Check(t, err)
	commit(third, "published-commit-blocked", "blocked")
	if f.commits.Load() != 2 {
		t.Fatal("published desk bypassed publish gate")
	}
	_, err = s.putAppPolicy("fixture-user", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Allow}})
	g02Check(t, err)
	commit(third, "published-commit-allowed", "observed")
	if f.commits.Load() != 3 {
		t.Fatal("published commit count")
	}
	status("published-source-synchronized", "synchronized")
	appStatus := func(name string) deskwork.AppState {
		t.Helper()
		inspected, err := registry.InspectDeskApplication(ctx, "native", entry.RunnerID, "alpha", f.Desk)
		g02Check(t, err)
		save(name, inspected)
		return inspected.Observation.Application
	}
	appStatus("application-before-activation")
	if _, err := os.Stat("/artifacts/release-source"); err == nil {
		// An empty file must normalize, commit, and compare as zero octets. The
		// kernel's base64 parser requires nonempty input, so it needs an explicit
		// zero-length representation in every typed source path.
		write("empty.txt", "")
		g02Check(t, filepath.WalkDir("/artifacts/release-source", func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, e := filepath.Rel("/artifacts/release-source", path)
			if e != nil {
				return e
			}
			if rel == "." {
				return nil
			}
			target := filepath.Join(source, rel)
			if d.IsDir() {
				return os.MkdirAll(target, 0700)
			}
			if d.Type() != 0 {
				return errors.New("unsafe release fixture")
			}
			data, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			return os.WriteFile(target, data, 0600)
		}))
		releaseStage := stage()
		releaseCommit := commit(releaseStage, "application-source-commit", "observed")
		_ = releaseCommit
		before, err := f.Observe(ctx, f.Desk)
		g02Check(t, err)
		emptyCheck, err := deskwork.MarkEquivalent(f.ship(), f.Desk, before.Revision, "empty.txt", []byte{})
		matches, err := f.call(ctx, emptyCheck, err)
		g02Check(t, err)
		if strings.TrimSpace(matches) != "%.y" {
			t.Fatal("empty source file differs after commit")
		}
		save("empty-source-normalized-committed-and-matched", true)
		release("activate-local", "local-activation-observed", "observed")
		var active deskwork.AppState
		for end := time.Now().Add(30 * time.Second); ; {
			active = appStatus("application-activation")
			if active.Zest == "live" && active.SourceRevision == before.Revision && slices.ContainsFunc(active.Agents, func(a deskwork.AppAgent) bool { return a.Name == "omabit-state" && a.Live }) && active.FrontendState == "site" {
				break
			}
			if time.Now().After(end) {
				t.Fatal("application activation not observed", active)
			}
			time.Sleep(200 * time.Millisecond)
		}
		if active.DocketHash == "" || active.DocketHash != active.FrontendDocketHash || !slices.Contains(active.ExpectedAgents, "omabit-state") {
			t.Fatal("docket/agent verification missing", active)
		}
		setPublication("treaty")
		release("publish-treaty", "treaty-publication-observed", "observed")
		for end := time.Now().Add(20 * time.Second); ; {
			published := appStatus("application-treaty")
			if published.TreatyListed && published.TreatyHash == before.ClayHash {
				break
			}
			if time.Now().After(end) {
				t.Fatal("Treaty publication not observed", published)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	if _, err := os.Stat("/artifacts/release-source"); err == nil {
		docket, err := os.ReadFile(filepath.Join(source, "desk.docket-0"))
		g02Check(t, err)
		globDocket := strings.Replace(string(docket), "site+/apps/omabit-fixture", "base+'"+f.Desk+"'\n   glob-ames+[~zod 0v0]", 1)
		if globDocket == string(docket) {
			t.Fatal("fixture docket marker absent")
		}
		write("desk.docket-0", globDocket)
		commit(stage(), "glob-metadata-commit", "observed")
		for end := time.Now().Add(20 * time.Second); ; {
			s := appStatus("glob-before-upload")
			if s.FrontendMode == "glob-ames" && s.DocketHash == s.FrontendDocketHash {
				break
			}
			if time.Now().After(end) {
				t.Fatal("glob metadata not realized", s)
			}
			time.Sleep(200 * time.Millisecond)
		}
		before, err := f.Observe(ctx, f.Desk)
		g02Check(t, err)
		content := []byte("<!doctype html><title>Omabit native fixture</title><p>Exact frozen frontend</p>\n")
		g02Check(t, f.uploadGlob(ctx, content))
		var uploaded deskwork.Observation
		for end := time.Now().Add(20 * time.Second); ; {
			s := appStatus("glob-after-upload")
			uploaded, err = f.Observe(ctx, f.Desk)
			g02Check(t, err)
			if s.FrontendState == "glob" && s.DocketHash == s.FrontendDocketHash && uploaded.Revision > before.Revision {
				break
			}
			if time.Now().After(end) {
				t.Fatal("glob upload not observed", s, uploaded)
			}
			time.Sleep(200 * time.Millisecond)
		}
		save("glob-upload-committed-docket", uploaded)
		// Upload edits Clay's docket itself; frozen ownership must not overwrite it.
		write("app/omabit-broken.hoon", "this-is-deliberately-not-an-agent\n")
		write("desk.bill", "~[%omabit-state %omabit-broken]\n")
		staged := stage()
		review, err := registry.PlanDeskCommit(ctx, "native", entry.RunnerID, staged.Intent.ID)
		g02Check(t, err)
		save("glob-source-ownership-conflict", review)
		if review.Review.ConflictCount == 0 {
			t.Fatal("external docket commit was overwritten")
		}

	}
	save("ok", true)
}
