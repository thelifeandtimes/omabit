package managed

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type slowCodeBackend struct {
	fakeBackend
	calls atomic.Int32
	ended chan struct{}
}

func (b *slowCodeBackend) LoginCode(ctx context.Context, _ runtimequalify.ManagedState, _ string) (string, error) {
	b.calls.Add(1)
	defer func() { b.ended <- struct{}{} }()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5500 * time.Millisecond):
		return "lidlut-tabwed-pillex-ridrup", nil
	}
}

// A real local HTTP connection has both ordinary five-second server and client
// limits. A six-second fake backend proves that the complete secret-read path
// has its own bound, not just the direct Go function used by unit fixtures.
func TestLoginCodeSlowVerifiedReadAndCallerCancellation(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.observe(v, v.Runtime, runtimequalify.ManagedObservation{State: "running", Health: "healthy", Resources: resourceSample(time.Now().UTC())}); err != nil {
		t.Fatal(err)
	}
	b := &slowCodeBackend{ended: make(chan struct{}, 2)}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid())}
	socket := filepath.Join(privateDir(t), "code.sock")
	listener, err := localrpc.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: r, ConnContext: PeerContext, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	defer func() {
		server.Close()
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			t.Error(err)
		}
	}()
	c := client.New(socket)
	defer c.Close()
	request := protocol.LoginCodeRequest{ExpectedRevision: 1, ContainerID: strings.Repeat("a", 64)}
	code, err := c.LoginCode(context.Background(), s.runnerID, "alpha", request)
	if err != nil || code.Code != "lidlut-tabwed-pillex-ridrup" {
		t.Fatal("verified read did not survive ordinary transport limits", err)
	}
	<-b.ended
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	code, err = c.LoginCode(ctx, s.runnerID, "alpha", request)
	if err == nil || code.Code != "" || time.Since(start) > time.Second {
		t.Fatal("caller deadline not honored")
	}
	select {
	case <-b.ended:
	case <-time.After(time.Second):
		t.Fatal("backend did not observe caller cancellation")
	}
	r.mu.Lock()
	r.mu.Unlock() // Let the canceled access finish its redacted audit.
	if b.calls.Load() != 2 {
		t.Fatal("request replayed", b.calls.Load())
	}
}
