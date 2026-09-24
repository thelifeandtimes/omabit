package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestIdentityPinOnEveryConnection(t *testing.T) {
	var replaced atomic.Bool
	var mutations atomic.Int32
	unpinned := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			id := "runner-original"
			if replaced.Load() {
				id = "runner-replacement"
			}
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: id, Mode: "simulation"})
			return
		}
		mutations.Add(1)
		w.Header().Set(protocol.Header, "1")
		json.NewEncoder(w).Encode(protocol.Operation{ID: "op-1"})
	})
	// Reuse the test listener address without introducing another listener helper.
	conn, err := unpinned.httpClient.Transport.(*http.Transport).DialContext(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	path := conn.RemoteAddr().String()
	conn.Close()
	c := NewPinned(path, Identity{RunnerID: "runner-original", Mode: "simulation"})
	defer c.Close()
	if _, err := c.Hello(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Operate(context.Background(), "alpha", protocol.OperationRequest{}); err != nil {
		t.Fatal(err)
	}
	c.httpClient.CloseIdleConnections()
	replaced.Store(true)
	_, err = c.Operate(context.Background(), "alpha", protocol.OperationRequest{})
	if !errors.Is(err, ErrIdentityMismatch) || mutations.Load() != 1 {
		t.Fatalf("replacement received a mutation: error=%v mutations=%d", err, mutations.Load())
	}
}
