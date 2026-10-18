package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestLegacyHTTPConfigurationPreservesNativeControl(t *testing.T) {
	s := testStore(t)
	req := registration("fixture", "register", 18080)
	req.Config.NativeControl = true
	if _, _, err := s.register("actor", req); err != nil {
		t.Fatal(err)
	}
	r := &Server{s: s, uid: uint32(os.Geteuid())}
	for _, tc := range []struct {
		field    string
		revision uint64
		want     bool
	}{
		{"", 1, true}, {`,"native_control":false`, 2, false}, {`,"native_control":true`, 3, true},
	} {
		// Exact pre-settings wire shape for the first request, followed by
		// explicit disabling/enabling through the same authenticated endpoint.
		wire := []byte(fmt.Sprintf(`{"expected_revision":%d,"config":{"http_port":18081,"ames_port":34343,"loom":31%s}}`, tc.revision, tc.field))
		call := httptest.NewRequest("PUT", "/v1/instances/fixture/configuration", bytes.NewReader(wire))
		call.Header.Set("Content-Type", "application/json")
		call.Header.Set(protocol.Header, "1")
		call = call.WithContext(context.WithValue(call.Context(), peerKey{}, r.uid))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, call)
		var got protocol.Instance
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != 200 || got.Config.NativeControl != tc.want || got.ConfigRevision != tc.revision+1 {
			t.Fatal(w.Code, w.Body.String(), err)
		}
	}
}

func TestHandoffPreservesRuntimeSettingsAndRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a, b := privateDir(t), privateDir(t)
	r, source, _ := testEndpoint(t, a, &handoffBackend{dir: a})
	dest, target, _ := testEndpoint(t, b, &handoffBackend{dir: b})
	req := registration("fixture", "register", 18080)
	req.Config.Loom, req.Config.NativeControl = 34, true
	if _, err := source.CreateFake(ctx, req); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"start", "stop"} {
		op, err := source.Operate(ctx, req.ID, protocol.OperationRequest{Kind: kind, RequestID: kind, ExpectedRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		waitOperation(t, source, op.ID, "succeeded")
	}
	offer := protocol.PrepareHandoffRequest{DestinationRunner: dest.s.runnerID, DestinationInstance: "arrived", ExpectedRevision: 1, RequestID: "move"}
	// A legacy collector cannot fence nondefault settings even when the
	// destination happens to be new enough.
	if _, err := source.PrepareHandoff(ctx, req.ID, offer); err == nil {
		t.Fatal("legacy collector fenced source")
	}
	v, err := loadInstance(r.s.db, req.ID)
	if err != nil || v.Runtime.Handoff != nil || v.Public.ActiveOperationID != "" {
		t.Fatal("refusal changed source", err)
	}
	for _, id := range []string{"move", "retry"} {
		offer.RequestID = id
		h, err := client.MoveStopped(ctx, source, target, r.s.runnerID, req.ID, offer, 18081, 0)
		if err != nil || h.RuntimeSettings == nil || h.RuntimeSettings.Loom != 34 || !h.RuntimeSettings.NativeControl || !h.ValidCommitment(h.Commitment) {
			t.Fatal(h, err)
		}
	}
	v, err = loadInstance(dest.s.db, "arrived")
	if err != nil || v.Public.Config.Loom != 34 || !v.Public.Config.NativeControl || v.Runtime.Loom != 34 || !v.Runtime.NativeControl {
		t.Fatal("settings lost", v, err)
	}
}
