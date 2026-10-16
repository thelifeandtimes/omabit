package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestLogsCLIWatchNDJSONAndCancellation(t *testing.T) {
	var requests atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-logs", Mode: "development-managed", AccessRole: "operator"})
			return
		}
		requests.Add(1)
		reply(t, w, protocol.RuntimeLogs{RunnerID: "runner-logs", InstanceID: "alpha", ConfigRevision: 2, Source: "none", ObservedAt: time.Now().UTC(), Lines: []string{}})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"--config-dir", dir, "enroll", "host", "--socket", socket}, io.Discard, &stderr); code != 0 {
		t.Fatal(code, stderr.String())
	}
	args := []string{"--config-dir", dir, "--runner", "host", "--expected-runner-id", "runner-logs", "logs", "alpha", "--revision", "2"}
	for _, extra := range [][]string{{"--watch", "--interval", "0s"}, {"--watch", "--interval", "999ms"}, {"--watch", "--interval", "bad"}} {
		if code := run(context.Background(), append(append([]string{}, args...), extra...), io.Discard, &stderr); code != 2 {
			t.Fatal("invalid interval accepted", code)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("invalid watch made remote requests")
	}
	var once bytes.Buffer
	if code := run(context.Background(), args, &once, &stderr); code != 0 {
		t.Fatal(code, stderr.String())
	}
	var tail protocol.RuntimeLogs
	if json.Unmarshal(once.Bytes(), &tail) != nil || tail.InstanceID != "alpha" || strings.Contains(once.String(), `"logs":`) {
		t.Fatal("one-shot output changed", once.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	done := make(chan int, 1)
	go func() {
		code := run(ctx, append(args, "--watch", "--interval", "1s"), writer, &stderr)
		writer.Close()
		done <- code
	}()
	scanner := bufio.NewScanner(reader)
	for i := 0; i < 2; i++ {
		if !scanner.Scan() {
			t.Fatal("missing immediate NDJSON result", scanner.Err())
		}
		var update collection.LogUpdate
		if json.Unmarshal(scanner.Bytes(), &update) != nil || update.Logs == nil || update.InstanceID != "alpha" || update.ConfigRevision != 2 {
			t.Fatal("not one complete bounded update per line", scanner.Text())
		}
	}
	cancel()
	if code := <-done; code != 0 {
		t.Fatal("signal cancellation failed", code, stderr.String())
	}
	if requests.Load() != 3 {
		t.Fatal("unexpected duplicate reads", requests.Load())
	}
}
