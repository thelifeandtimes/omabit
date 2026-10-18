package managed

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// Independent of installer mocks: use the real HTTP admission handler, SQLite
// fence, server reconstruction and startup scheduler with an inert backend.
func TestRC11PreparedAcquireReplayUsesActualFenceAndStartupAdmission(t *testing.T) {
	for _, outcome := range []string{"held", "released", "not-acquired"} {
		t.Run(outcome, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			seedHealthyBoot(t, s)
			ctx, cancel := context.WithCancel(context.Background())
			previous := &Server{s: s, ctx: ctx, cancel: cancel, uid: uint32(os.Geteuid()), executableSHA256: strings.Repeat("f", 64)}
			token := strings.Repeat("a", 64)
			if outcome != "not-acquired" {
				if w := rc11Fence(previous, "acquire", token); w.Code != 200 {
					t.Fatal(w.Body.String())
				}
			}
			if outcome == "released" {
				if w := rc11Fence(previous, "release", token); w.Code != 200 {
					t.Fatal(w.Body.String())
				}
			}
			// Discard any response and shut down the previous management process.
			// The activation journal cannot tell which of these outcomes occurred.
			if err := previous.Close(); err != nil {
				t.Fatal(err)
			}
			b := &startupBackend{exitCode: 255, entered: make(chan struct{}, 1), release: make(chan struct{})}
			r, err := newServerWithStartup(dir, b, time.Millisecond, true, nextBootID)
			if err != nil {
				t.Fatal(err)
			}
			drained := false
			defer func() {
				if !drained {
					close(b.release)
				}
				r.Close()
			}()
			if outcome == "held" {
				time.Sleep(20 * time.Millisecond)
				b.mu.Lock()
				calls := b.calls
				b.mu.Unlock()
				if calls != 0 {
					t.Fatal("held fence admitted startup", calls)
				}
				if w := rc11Fence(r, "acquire", token); w.Code != 200 {
					t.Fatal("same-token durable acquire did not replay", w.Body.String())
				}
				fence, err := loadManagementFence(r.s)
				if err != nil || !fence.Held || fence.Token != token {
					t.Fatal(fence, err)
				}
				if w := rc11Fence(r, "release", token); w.Code != 200 {
					t.Fatal(w.Body.String())
				}
			}
			select {
			case <-b.entered:
			case <-time.After(time.Second):
				t.Fatal("normal startup did not admit its eligible job")
			}
			if outcome != "held" {
				w := rc11Fence(r, "acquire", token)
				code := "update_busy"
				if outcome == "released" {
					code = "update_token_released"
				}
				if w.Code != 409 || !strings.Contains(w.Body.String(), code) {
					t.Fatal(w.Code, w.Body.String())
				}
				if r.ctx.Err() != nil {
					t.Fatal("acquire refusal canceled management/startup")
				}
				b.mu.Lock()
				active := b.active
				b.mu.Unlock()
				if active != 1 {
					t.Fatal("acquire refusal drained/canceled the active job", active)
				}
				fence, err := loadManagementFence(r.s)
				if err != nil || fence.Held || outcome == "not-acquired" && fence.Token != "" || outcome == "released" && fence.Token != token {
					t.Fatal(fence, err)
				}
			}
			close(b.release)
			drained = true
			startupOperations(t, r, 1, "succeeded")
			if outcome == "not-acquired" {
				deadline := time.Now().Add(time.Second)
				for {
					w := rc11Fence(r, "acquire", token)
					if w.Code == 200 {
						break
					}
					if !strings.Contains(w.Body.String(), "update_busy") || time.Now().After(deadline) {
						t.Fatal(w.Body.String())
					}
					time.Sleep(time.Millisecond)
				}
			} else {
				if w := rc11Fence(r, "acquire", token); w.Code != 409 || !strings.Contains(w.Body.String(), "update_token_released") {
					t.Fatal("reused an actually released token", w.Code, w.Body.String())
				}
			}
		})
	}
}
