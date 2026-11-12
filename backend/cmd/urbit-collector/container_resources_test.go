package main

import (
	"bytes"
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"testing"
)

func TestContainerResourcesCLI(t *testing.T) {
	for _, supported := range []bool{false, true} {
		calls := 0
		socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/hello" {
				features := []string{"runtime-settings"}
				if supported {
					features = append(features, protocol.ContainerResourcesFeature)
				}
				reply(t, w, protocol.Hello{ProtocolVersion: 1, Features: features})
				return
			}
			calls++
			var req protocol.ConfigureRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.Config.MemoryGiB != 16 || req.Config.CPUCores != 8 || !req.PreserveNativeControl {
				t.Error("limits or native preservation lost", req)
			}
			reply(t, w, protocol.Instance{ID: "alpha"})
		})
		var out, errs bytes.Buffer
		code := run(context.Background(), []string{"--socket", socket, "configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343", "--loom=31", "--memory-gib=16", "--cpus=8"}, &out, &errs)
		if supported && (code != 0 || calls != 1) || !supported && (code != 1 || calls != 0) {
			t.Fatal(supported, code, calls, errs.String())
		}
	}
	for _, flag := range []string{"--memory-gib", "--cpus"} {
		for _, value := range []string{"0", "17", "256", "-1", "1.5"} {
			var out, errs bytes.Buffer
			code := run(context.Background(), []string{"--socket", "/nonexistent.sock", "configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343", "--loom=31", flag + "=" + value}, &out, &errs)
			if code != 2 {
				t.Fatal("invalid limit reached transport", flag, value, code, errs.String())
			}
		}
	}
}
