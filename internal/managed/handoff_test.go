package managed

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type handoffBackend struct {
	fakeBackend
	dir         string
	failReceive atomic.Bool
}

func TestHandoffSchemaUpgradePreservesPermanentRegistration(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("keep", "keep", 18080)); err != nil {
		t.Fatal(err)
	}
	before, err := loadInstance(s.db, "keep")
	if err != nil {
		t.Fatal(err)
	}
	runner := s.runnerID
	if _, err = s.db.Exec("PRAGMA user_version=11"); err != nil {
		t.Fatal(err)
	}
	if err = s.initialize(); err != nil {
		t.Fatal(err)
	}
	after, err := loadInstance(s.db, "keep")
	if err != nil || !reflect.DeepEqual(before, after) || s.runnerID != runner {
		t.Fatal("upgrade changed registration", err)
	}
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
		t.Fatal(version, err)
	}
	if _, err = s.db.Exec(fmt.Sprintf("PRAGMA user_version=%d", currentSchema+1)); err != nil {
		t.Fatal(err)
	}
	if err = s.initialize(); err == nil {
		t.Fatal("future fencing schema accepted")
	}
}

func (*handoffBackend) HandoffDiskAvailable(string, uint64) bool { return true }

func (*handoffBackend) PierSources(context.Context) ([]pierstore.Candidate, error) {
	return []pierstore.Candidate{}, nil
}
func (b *handoffBackend) Observe(ctx context.Context, s runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if s.Handoff.Fenced() {
		state := "moving"
		if s.Handoff.State == "moved" {
			state = "moved"
		}
		return s, runtimequalify.ManagedObservation{State: state, Health: "unknown"}, nil
	}
	return b.fakeBackend.Observe(ctx, s)
}
func (b *handoffBackend) Apply(ctx context.Context, kind string, s runtimequalify.ManagedState, persist func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if !strings.HasSuffix(kind, "-handoff") {
		return b.fakeBackend.Apply(ctx, kind, s, persist)
	}
	h := s.Handoff
	obs := runtimequalify.ManagedObservation{State: "moving", Health: "unknown"}
	switch kind {
	case "prepare-handoff":
		data := []byte(strings.Repeat("stopped pier fixture\x00", 1024))
		h.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		h.TreeDigest = h.SHA256
		h.Bytes = int64(len(data))
		h.State = "ready"
		path := runtimequalify.HandoffArchive(b.dir, h)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return s, obs, err
		}
		if err := os.WriteFile(path, data, 0400); err != nil {
			return s, obs, err
		}
	case "receive-handoff":
		if b.failReceive.Swap(false) {
			return s, obs, errors.New("injected interruption")
		}
		if !runtimequalify.VerifyHandoffArchive(runtimequalify.HandoffArchive(b.dir, h), h) {
			return s, obs, errors.New("bad stream")
		}
		s.Initialized = true
		s.Prepared = true
		h.State = "received"
	case "commit-handoff":
		h.State = "moved"
		h.Commitment = fmt.Sprintf("%x", ed25519.Sign(s.HandoffSigningKey, h.CommitmentBytes()))
		obs.State = "moved"
	}
	h.Error = ""
	return s, obs, persist(s)
}

func TestCrossHostHandoffResumesFailureAndKeepsSourceFencedAfterRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a, b := privateDir(t), privateDir(t)
	sourceBackend, destinationBackend := &handoffBackend{dir: a}, &handoffBackend{dir: b}
	r, source, closeSource := testEndpoint(t, a, sourceBackend)
	dest, target, _ := testEndpoint(t, b, destinationBackend)
	// Exercise both collector preflight and destination receipt above the former
	// registration cap, using real RPC/journal paths without starting more ships.
	for n := range 16 {
		id := fmt.Sprintf("existing-%d", n)
		if _, err := target.CreateFake(ctx, registration(id, id, uint16(18100+n))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := source.CreateFake(ctx, registration("ship", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"start", "stop"} {
		op, err := source.Operate(ctx, "ship", protocol.OperationRequest{Kind: kind, RequestID: kind, ExpectedRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		waitOperation(t, source, op.ID, "succeeded")
	}
	req := protocol.PrepareHandoffRequest{DestinationRunner: dest.s.runnerID, DestinationInstance: "arrived", ExpectedRevision: 1, RequestID: "attempt-one"}
	destinationBackend.failReceive.Store(true)
	h, err := client.MoveStopped(ctx, source, target, r.s.runnerID, "ship", req, 18081, 0)
	if err == nil || !strings.Contains(err.Error(), "injected interruption") {
		t.Fatal(h, err)
	}
	if _, err = source.Operate(ctx, "ship", protocol.OperationRequest{Kind: "start", RequestID: "forbidden", ExpectedRevision: 1}); err == nil {
		t.Fatal("source unfenced after failed relay")
	}
	if _, err = target.FinishHandoff(ctx, h.ID, "activate", protocol.FinishHandoffRequest{SHA256: h.SHA256, ExpectedRevision: 1, RequestID: "early", Commitment: strings.Repeat("0", 128)}); err == nil {
		t.Fatal("destination accepted forged source commitment")
	}
	req.RequestID = "explicit-continue"
	result, err := client.MoveStopped(ctx, source, target, r.s.runnerID, "ship", req, 18081, 0)
	if err != nil || result.State != "complete" || !result.ValidCommitment(result.Commitment) {
		t.Fatal(result, err)
	}
	s, err := target.Snapshot(ctx)
	if err != nil || s.Instances[0].DesiredState != "stopped" || s.Instances[0].ActiveOperationID != "" {
		t.Fatal(s, err)
	}
	// A lost success response replays by durable transfer identity, without a new copy.
	req.RequestID = "read-completed"
	if _, err = client.MoveStopped(ctx, source, target, r.s.runnerID, "ship", req, 18081, 0); err != nil {
		t.Fatal(err)
	}
	closeSource()
	r, source, _ = testEndpoint(t, a, sourceBackend)
	for _, kind := range []string{"start", "stop", "pack", "roll", "move-pier"} {
		if _, err = source.Operate(ctx, "ship", protocol.OperationRequest{Kind: kind, RequestID: "after-restart-" + kind, ExpectedRevision: 1, PierPath: "fakeships/elsewhere/pier"}); err == nil {
			t.Fatal("source fence bypass", kind)
		}
	}
	if _, err = source.Configure(ctx, "ship", protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18090, AmesPort: 34343, Loom: 31}}); err == nil {
		t.Fatal("configured moved source")
	}
	v, err := loadInstance(r.s.db, "ship")
	if err != nil || !v.Runtime.Handoff.Fenced() || startupEligible(v) {
		t.Fatal("lost restart fence", err)
	}
}
