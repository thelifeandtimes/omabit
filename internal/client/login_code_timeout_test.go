package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestLoginCodeKeepsOrdinaryClientBudget(t *testing.T) {
	container := strings.Repeat("a", 64)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(200 * time.Millisecond):
		}
		w.Header().Set(protocol.Header, "1")
		if strings.HasSuffix(r.URL.Path, "/login-code") {
			json.NewEncoder(w).Encode(protocol.LoginCode{RunnerID: "runner-test", InstanceID: "test", ConfigRevision: 1, ContainerID: container, Code: "lidlut-tabwed-pillex-ridrup"})
		} else {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-test"})
		}
	})
	c.httpClient.Timeout = 50 * time.Millisecond
	request := protocol.LoginCodeRequest{ExpectedRevision: 1, ContainerID: container}
	if _, err := c.LoginCode(context.Background(), "runner-test", "test", request); err != nil {
		t.Fatal("code read retained ordinary deadline", err)
	}
	if c.httpClient.Timeout != 50*time.Millisecond {
		t.Fatal("code read changed the shared client timeout")
	}
	if _, err := c.Hello(context.Background()); err == nil {
		t.Fatal("ordinary request inherited secret-read timeout")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := c.LoginCode(ctx, "runner-test", "test", request)
	if err == nil || result.Code != "" {
		t.Fatal("canceled caller obtained a response")
	}
}
