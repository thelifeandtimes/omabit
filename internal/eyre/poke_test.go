package eyre

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
)

func TestPokeRequiresGallAckAndNeverReplays(t *testing.T) {
	for _, mode := range []string{"ok", "nack", "missing"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			var actions []string
			c, _ := fakeEyre(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Method == "PUT" {
					var messages []map[string]any
					json.NewDecoder(r.Body).Decode(&messages)
					actions = append(actions, messages[0]["action"].(string))
					w.WriteHeader(204)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				if mode == "missing" {
					return
				}
				value := `"ok":"ok"`
				if mode == "nack" {
					value = `"err":["private application detail"]`
				}
				fmt.Fprintf(w, "id: 0\ndata: {\"id\":1,\"response\":\"poke\",%s}\n\n", value)
			})
			if err := c.Login(context.Background(), []byte(fakeCode)); err != nil {
				t.Fatal(err)
			}
			result, err := c.Poke(context.Background(), "settings", "settings-event", json.RawMessage(`{}`))
			if mode == "ok" {
				if err != nil || !result.Acknowledged || !result.CleanupConfirmed {
					t.Fatalf("ack: %+v %v", result, err)
				}
			} else {
				want := ErrUncertain
				if mode == "nack" {
					want = ErrNack
				}
				if !errors.Is(err, want) || result.Acknowledged {
					t.Fatalf("false acknowledgement: %+v %v", result, err)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			pokes := 0
			for _, action := range actions {
				if action == "poke" {
					pokes++
				}
			}
			if pokes != 1 || actions[len(actions)-1] != "delete" {
				t.Fatalf("replay/cleanup: %v", actions)
			}
		})
	}
}
