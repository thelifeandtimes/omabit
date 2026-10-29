package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type dependencyTestBackend struct {
	fakeBackend
	calls int
	fail  bool
}

func (b *dependencyTestBackend) PrepareProjectDependencies(context.Context) error {
	b.calls++
	if b.fail {
		return errors.New("download failed")
	}
	return nil
}
func TestProjectDependenciesConsentAndAdmission(t *testing.T) {
	for _, mode := range []string{"ok", "consent", "observer", "quiesced", "cancelled", "backend", "replay"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			b := &dependencyTestBackend{fail: mode == "backend"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := &Server{s: s, b: b, ctx: ctx, workGate: make(chan struct{}, 1), uid: uint32(os.Geteuid())}
			if mode == "quiesced" {
				r.updateFence.Held = true
			}
			if mode == "cancelled" {
				cancel()
			}
			body, _ := json.Marshal(protocol.ProjectDependenciesRequest{ConfirmDownload: mode != "consent"})
			q := httptest.NewRequest("POST", protocol.ProjectDependenciesPath, bytes.NewReader(body))
			q.Header.Set("Content-Type", "application/json")
			role := "operator"
			if mode == "observer" {
				role = "observer"
			}
			if mode == "replay" {
				q.Header.Set(protocol.CreationReplayHeader, "true")
			}
			w := httptest.NewRecorder()
			r.serveProjectDependencies(w, q, role)
			if (w.Code == 200) != (mode == "ok") {
				t.Fatal(w.Code, w.Body.String())
			}
			want := 0
			if mode == "ok" || mode == "backend" {
				want = 1
			}
			if b.calls != want || r.updateBulk != 0 || len(r.workGate) != 0 {
				t.Fatal(b.calls, r.updateBulk)
			}
		})
	}
}

type blockingDependenciesBackend struct {
	fakeBackend
	opened  chan struct{}
	release chan struct{}
}

func (b *blockingDependenciesBackend) PrepareProjectDependencies(ctx context.Context) error {
	close(b.opened)
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func TestProjectDependenciesHoldResourceAndUpdateFences(t *testing.T) {
	r, _, _, _ := projectQueueFixture(t)
	b := &blockingDependenciesBackend{opened: make(chan struct{}), release: make(chan struct{})}
	r.b = b
	r.workGate <- struct{}{}
	q := httptest.NewRequest("POST", protocol.ProjectDependenciesPath, bytes.NewBufferString(`{"confirm_download":true}`))
	q.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { defer close(done); r.serveProjectDependencies(w, q, "operator") }()
	deadline := time.Now().Add(2 * time.Second)
	for {
		r.mu.Lock()
		admitted := r.updateBulk > 0
		r.mu.Unlock()
		if admitted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("download admission not reached")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-b.opened:
		t.Fatal("download bypassed resource gate")
	default:
	}
	if result := rc11Fence(r, "acquire", strings.Repeat("a", 64)); result.Code != 409 {
		t.Fatal("update admitted during queued download", result.Code)
	}
	<-r.workGate
	select {
	case <-b.opened:
	case <-time.After(2 * time.Second):
		t.Fatal("download did not enter resource gate")
	}
	if result := rc11Fence(r, "acquire", strings.Repeat("a", 64)); result.Code != 409 {
		t.Fatal("update admitted during active download", result.Code)
	}
	close(b.release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("download did not finish")
	}
	if w.Code != 200 || len(r.workGate) != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
	if result := rc11Fence(r, "acquire", strings.Repeat("a", 64)); result.Code != 200 {
		t.Fatal("update fence leaked", result.Code, result.Body.String())
	}
}
