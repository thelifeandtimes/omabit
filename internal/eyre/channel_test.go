package eyre

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestChannelReplayAcknowledgementAndCleanup(t *testing.T) {
	var mu sync.Mutex
	var puts []string
	gets := 0
	c, _ := fakeEyre(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == "PUT" {
			var commands []map[string]any
			if err := json.NewDecoder(r.Body).Decode(&commands); err != nil {
				t.Error(err)
			}
			puts = append(puts, commands[0]["action"].(string))
			w.WriteHeader(204)
			return
		}
		gets++
		w.Header().Set("Content-Type", "text/event-stream")
		if gets == 1 {
			fmt.Fprint(w, "id: 1\ndata: {\"id\":1,\"response\":\"subscribe\",\"ok\":\"ok\"}\n\nid: 2\ndata: {\"id\":1,\"response\":\"diff\",\"json\":{\"first\":true}}\n\nid: 3\ndata: {\"truncated\":")
		} else {
			if r.Header.Get("Last-Event-ID") != "2" {
				t.Errorf("reconnect cursor: %q", r.Header.Get("Last-Event-ID"))
			}
			fmt.Fprint(w, "id: 2\ndata: {\"id\":1,\"response\":\"diff\",\"json\":{\"duplicate\":true}}\n\nid: 3\ndata: {\"id\":1,\"response\":\"diff\",\"json\":{\"second\":true}}\n\n")
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Login(ctx, []byte(fakeCode)); err != nil {
		t.Fatal(err)
	}
	var ids []uint64
	err := c.Watch(ctx, "hood", "/kiln/vats", 2, func(event Event) error {
		if event.Kind == "event" {
			ids = append(ids, event.EventID)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(ids) != "[1 2 3]" {
		t.Fatalf("replayed or lost event: %v", ids)
	}
	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(puts) != "[subscribe ack ack ack delete]" {
		t.Fatalf("channel writes: %v", puts)
	}
}

func TestLostChannelAndConsumerFailure(t *testing.T) {
	for _, kind := range []string{"lost", "consumer", "nack", "wrong-id"} {
		t.Run(kind, func(t *testing.T) {
			var writes []string
			var mu sync.Mutex
			c, _ := fakeEyre(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "PUT" {
					var commands []map[string]any
					json.NewDecoder(r.Body).Decode(&commands)
					mu.Lock()
					writes = append(writes, commands[0]["action"].(string))
					mu.Unlock()
					w.WriteHeader(204)
					return
				}
				if kind == "lost" {
					w.WriteHeader(404)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				data := `{"id":1,"response":"diff","json":{}}`
				if kind == "nack" {
					data = `{"id":1,"response":"subscribe","err":["private-state"]}`
				}
				if kind == "wrong-id" {
					data = `{"id":2,"response":"diff","json":{}}`
				}
				fmt.Fprintf(w, "id: 1\ndata: %s\n\n", data)
			})
			ctx := context.Background()
			if err := c.Login(ctx, []byte(fakeCode)); err != nil {
				t.Fatal(err)
			}
			consumerErr := errors.New("output closed")
			err := c.Watch(ctx, "hood", "/kiln/vats", 1, func(event Event) error {
				if kind == "consumer" && event.Kind == "event" {
					return consumerErr
				}
				return nil
			})
			want := map[string]error{"lost": ErrReconcile, "consumer": consumerErr, "nack": ErrSubscription, "wrong-id": ErrProtocol}[kind]
			if !errors.Is(err, want) || strings.Contains(err.Error(), "private-state") {
				t.Fatalf("unexpected stream result: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if fmt.Sprint(writes) != "[subscribe delete]" {
				t.Fatalf("acknowledged undelivered event: %v", writes)
			}
		})
	}
}

func TestSSEBoundsAndMultiline(t *testing.T) {
	zeroDelivered := false
	_ = readEvents(strings.NewReader("id: 0\ndata: {}\n\n"), func(id uint64, data []byte) error { zeroDelivered = id == 0; return nil })
	if !zeroDelivered {
		t.Fatal("Eyre's initial event ID zero was discarded")
	}
	input := ":heartbeat\r\nid: 12\r\ndata: {\r\ndata: \"test\": true}\r\n\r\nid: 13\ndata: partial"
	seen := 0
	_ = readEvents(strings.NewReader(input), func(id uint64, data []byte) error {
		seen++
		if id != 12 || !json.Valid(data) {
			t.Fatalf("SSE frame: %d %s", id, data)
		}
		return nil
	})
	if seen != 1 {
		t.Fatal("incomplete frame delivered")
	}
	if err := readEvents(strings.NewReader("data: "+strings.Repeat("a", maxBody+1)), func(uint64, []byte) error { return nil }); err == nil {
		t.Fatal("oversized SSE line accepted")
	}
}

func TestReplayCursorZero(t *testing.T) {
	var mu sync.Mutex
	gets := 0
	c, _ := fakeEyre(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == "PUT" {
			w.WriteHeader(204)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		gets++
		if gets == 1 {
			fmt.Fprint(w, "id: 0\ndata: {\"id\":1,\"response\":\"subscribe\",\"ok\":\"ok\"}\n\n")
			return
		}
		if r.Header.Get("Last-Event-ID") != "0" {
			t.Errorf("initial zero cursor was lost: %q", r.Header.Get("Last-Event-ID"))
		}
		fmt.Fprint(w, "id: 0\ndata: {\"id\":1,\"response\":\"subscribe\",\"ok\":\"ok\"}\n\nid: 1\ndata: {\"id\":1,\"response\":\"diff\",\"json\":{}}\n\n")
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Login(ctx, []byte(fakeCode)); err != nil {
		t.Fatal(err)
	}
	var delivered []uint64
	if err := c.Watch(ctx, "hood", "/kiln/updates", 1, func(event Event) error {
		if event.Kind == "event" {
			delivered = append(delivered, event.EventID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(delivered) != "[0 1]" {
		t.Fatalf("zero cursor replay: %v", delivered)
	}
}
