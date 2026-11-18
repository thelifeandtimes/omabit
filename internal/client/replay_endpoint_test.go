package client

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestLookupOnlyRefusesUnguardedEffectsBeforeDispatch(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("unsupported lookup reached a server")
	})
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		var out map[string]any
		err := c.do(WithCreationReplayOnly(context.Background()), method, "/v1/instances/selected/operations", map[string]string{"request_id": "do-not-dispatch"}, &out, true)
		var uncertain *OutcomeUncertainError
		if err == nil || errors.As(err, &uncertain) || calls.Load() != 0 {
			t.Fatal("lookup guard was dropped or dispatch misreported", method, err, calls.Load())
		}
	}
}
