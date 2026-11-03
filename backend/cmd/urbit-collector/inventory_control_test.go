package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestInventoryControlRejectsInvalidOrUnboundedInput(t *testing.T) {
	for _, input := range []string{`{}`, `{"interval":"1ms"}`, `{"interval":"-1s"}`, `{"interval":"no"}`, `{"interval":"2s","refresh":1}`, `{"interval":"2s","runner":"other"}`, `{"interval":"2s"} {}`, strings.Repeat("x", 2048)} {
		c, _ := collection.NewWatchCadence(time.Second)
		if err := readInventoryControls(strings.NewReader(input), c); err == nil {
			t.Errorf("accepted invalid schedule %q", input[:min(80, len(input))])
		}
	}
	c, _ := collection.NewWatchCadence(time.Second)
	if err := readInventoryControls(strings.NewReader("{\"interval\":\"30s\"}\n{\"interval\":\"2s\",\"refresh\":true}\n"), c); err != nil {
		t.Fatal(err)
	}
}

func TestInventoryControlChangesCadenceWithoutReplacingTransport(t *testing.T) {
	for _, ending := range []string{"eof", "cancel", "invalid"} {
		t.Run(ending, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "r.sock")
			ln, err := net.Listen("unix", path)
			if err != nil {
				t.Fatal(err)
			}
			var opened, closed atomic.Int32
			server := &http.Server{ConnState: func(_ net.Conn, s http.ConnState) {
				if s == http.StateNew {
					opened.Add(1)
				}
				if s == http.StateClosed {
					closed.Add(1)
				}
			}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if r.URL.Path == "/hello" {
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation"})
					return
				}
				json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "runner", Mode: "simulation"})
			})}
			go server.Serve(ln)
			defer server.Close()
			input, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			defer writer.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			observations := make(chan collection.Observation, 8)
			done := make(chan error, 1)
			go func() {
				done <- controlledInventory(ctx, []collection.Connection{{Name: "test", RunnerID: "runner", Mode: "simulation", Transport: "unix", Socket: path}}, time.Hour, input, func(o collection.Observation) error { observations <- o; return nil })
			}()
			next := func() {
				t.Helper()
				select {
				case o := <-observations:
					if o.Status != "available" {
						t.Fatal(o)
					}
				case <-time.After(time.Second):
					t.Fatal("watch did not refresh promptly")
				}
			}
			next()
			select {
			case <-observations:
				t.Fatal("idle watch polled again")
			case <-time.After(130 * time.Millisecond):
			}
			io.WriteString(writer, "{\"interval\":\"100ms\",\"refresh\":true}\n")
			next()
			next()
			io.WriteString(writer, "{\"interval\":\"1h\"}\n")
			time.Sleep(20 * time.Millisecond)
			select {
			case <-observations:
				t.Fatal("slowing watch triggered a poll")
			case <-time.After(130 * time.Millisecond):
			}
			switch ending {
			case "eof":
				writer.Close()
			case "cancel":
				cancel()
			case "invalid":
				io.WriteString(writer, "{\"interval\":\"bad\"}\n")
			}
			select {
			case err = <-done:
			case <-time.After(time.Second):
				t.Fatal("control owner/transport cleanup blocked")
			}
			if ending == "eof" && err != nil || ending == "cancel" && !errors.Is(err, context.Canceled) || ending == "invalid" && err == nil {
				t.Fatal(ending, err)
			}
			if opened.Load() != 1 {
				t.Fatal("visibility changes replaced transport", opened.Load())
			}
			deadline := time.Now().Add(time.Second)
			for closed.Load() != opened.Load() && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			if closed.Load() != opened.Load() {
				t.Fatal("transport retained after watch exit")
			}
		})
	}
}
