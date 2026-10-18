package managed

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRC11OfflineForwardRetryReconcilesActualPostReleaseStart(t *testing.T) {
	for _, released := range []bool{false, true} {
		t.Run(map[bool]string{false: "fence-held", true: "release-ack-lost-with-queued-start"}[released], func(t *testing.T) {
			dir := privateDir(t)
			r, c, shutdown := testEndpoint(t, dir, &fakeBackend{})
			if _, err := c.CreateFake(context.Background(), registration("alpha", "register", 18080)); err != nil {
				t.Fatal(err)
			}
			token, runner := strings.Repeat("a", 64), r.s.runnerID
			if w := rc11Fence(r, "acquire", token); w.Code != 200 {
				t.Fatal(w.Body.String())
			}
			if released {
				if w := rc11Fence(r, "release", token); w.Code != 200 {
					t.Fatal(w.Body.String())
				}
			}
			// Real API acceptance and worker registration, not a relabeled receipt.
			// Occupy the host gate so Close occurs strictly before beginOperation.
			r.workGate <- struct{}{}
			op, err := c.Operate(context.Background(), "alpha", protocol.OperationRequest{Kind: "start", RequestID: "after-release", ExpectedRevision: 1})
			if released && err != nil || !released && err == nil {
				t.Fatal(op, err)
			}
			shutdown()
			if err := os.WriteFile(filepath.Join(dir, ".lock"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			lock, err := LockOfflineState(dir, uint32(os.Geteuid()))
			if err != nil {
				t.Fatal(err)
			}
			db, err := lock.openDB(false)
			if err != nil {
				lock.Close()
				t.Fatal(err)
			}
			var before []byte
			if released {
				if err := db.QueryRow("SELECT body FROM operations WHERE id=?", op.ID).Scan(&before); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := lock.Inspect(context.Background()); released && err == nil {
				t.Fatal("initial idle prerequisite was weakened")
			}
			for _, bad := range []struct {
				runner, token string
				schema        int
			}{
				{"runner-" + strings.Repeat("b", 32), token, StateSchemaVersion},
				{runner, strings.Repeat("b", 64), StateSchemaVersion},
				{runner, token, 19}, {runner, token, StateSchemaVersion + 1},
			} {
				if _, err := lock.InspectContinuation(context.Background(), bad.runner, bad.token, bad.schema); err == nil {
					t.Fatal("accepted unbound continuation", bad)
				}
			}
			state, err := lock.InspectContinuation(context.Background(), runner, token, StateSchemaVersion)
			if err != nil || state.Quiesced == released || state.Token != token {
				t.Fatal(state, err)
			}
			if released {
				var after []byte
				if err := db.QueryRow("SELECT body FROM operations WHERE id=?", op.ID).Scan(&after); err != nil || string(before) != string(after) {
					t.Fatal("continuation rewrote accepted receipt", err)
				}
				var queued protocol.Operation
				if json.Unmarshal(after, &queued) != nil || queued.Status != "queued" {
					t.Fatal(string(after))
				}
			}
			db.Close()
			if err := lock.Close(); err != nil {
				t.Fatal(err)
			}
			// Restart the new manager normally. Its existing startup reconciliation
			// owns interrupted work; neither offline inspection nor release does it.
			next, err := newServer(dir, &fakeBackend{}, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			defer next.Close()
			if released {
				saved, err := loadOperation(next.s.db, op.ID)
				if err != nil || saved.Status != "interrupted" || saved.RequestID != op.RequestID || saved.Actor != op.Actor {
					t.Fatal(saved, err)
				}
			} else {
				w := rc11Request(next, context.Background(), "POST", "/v1/instances/alpha/operations", protocol.OperationRequest{Kind: "start", RequestID: "still-held", ExpectedRevision: 1})
				if w.Code != 409 || !strings.Contains(w.Body.String(), "management_quiesced") {
					t.Fatal(w.Code, w.Body.String())
				}
			}
			if w := rc11Fence(next, "release", strings.Repeat("b", 64)); w.Code != 409 {
				t.Fatal("released another token", w.Body.String())
			}
			for range 2 {
				if w := rc11Fence(next, "release", token); w.Code != 200 {
					t.Fatal(w.Body.String())
				}
			}
			w := rc11Request(next, context.Background(), "POST", "/v1/instances/alpha/operations", protocol.OperationRequest{Kind: "start", RequestID: "forward-retry", ExpectedRevision: 1})
			if w.Code != 202 {
				t.Fatal(w.Code, w.Body.String())
			}
			var retry protocol.Operation
			if err := json.Unmarshal(w.Body.Bytes(), &retry); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(2 * time.Second)
			for {
				saved, err := loadOperation(next.s.db, retry.ID)
				if err != nil {
					t.Fatal(err)
				}
				if saved.Status == "succeeded" {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("forward retry did not complete", saved)
				}
				time.Sleep(time.Millisecond)
			}
		})
	}
}
