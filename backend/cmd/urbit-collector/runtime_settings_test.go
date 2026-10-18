package main

import (
	"bytes"
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"strings"
	"testing"
)

func TestRuntimeSettingsCLI(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		loom   uint8
		native bool
	}{
		{"create", []string{"create-fake", "alpha", "--http-port=18080", "--request-id=one", "--loom=34", "--native-control"}, 34, true},
		{"default", []string{"create-fake", "alpha", "--http-port=18080", "--request-id=one"}, 31, false},
		{"configure", []string{"configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343", "--loom=34", "--native-control=true"}, 34, true},
		{"minimum", []string{"create-fake", "alpha", "--http-port=18080", "--request-id=one", "--loom=30"}, 30, false},
		{"disable", []string{"configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343", "--loom=30", "--native-control=false"}, 30, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/hello" {
					features := []string{"create-fake", "runtime-settings"}
					if tc.args[0] == "configure" {
						features = append(features, "native-control-all-ships")
					}
					reply(t, w, protocol.Hello{ProtocolVersion: 1, Features: features})
					return
				}
				calls++
				var body struct {
					Config protocol.Configuration `json:"config"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Config.Loom != tc.loom || body.Config.NativeControl != tc.native {
					t.Errorf("runtime settings lost in CLI request: %+v", body.Config)
				}
				if tc.args[0] == "configure" {
					reply(t, w, protocol.Instance{ID: "alpha"})
				} else {
					reply(t, w, protocol.Operation{ID: "op-one"})
				}
			})
			var out, errs bytes.Buffer
			if code := run(context.Background(), append([]string{"--socket", socket}, tc.args...), &out, &errs); code != 0 || calls != 1 {
				t.Fatal("settings request failed", code, calls, errs.String())
			}
		})
	}
}

func TestRuntimeSettingsCLICapabilityAndBounds(t *testing.T) {
	for _, value := range []string{"19", "35", "256", "999999", "-1", "1.5"} {
		for _, command := range [][]string{
			{"create-fake", "alpha", "--http-port=18080", "--request-id=one"},
			{"create-comet", "alpha", "--http-port=18080", "--request-id=one", "--confirm-network"},
			{"create-ship", "--ship=~sampel-palnet", "--key-file=/missing", "--request-id=one", "--confirm-network"},
			{"configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343"},
		} {
			var out, errs bytes.Buffer
			args := append([]string{"--socket", "/nonexistent.sock"}, command...)
			if code := run(context.Background(), append(args, "--loom="+value), &out, &errs); code != 2 {
				t.Fatal("invalid loom not rejected before connection", command[0], value, code, errs.String())
			}
		}
	}
	calls := 0
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, Features: []string{"create-fake"}})
			return
		}
		calls++
		reply(t, w, protocol.Operation{})
	})
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"--socket", socket, "create-fake", "alpha", "--http-port=18080", "--request-id=one", "--loom=34", "--native-control"}, &out, &errs); code != 1 || calls != 0 || !strings.Contains(errs.String(), "runtime-settings") {
		t.Fatal("unsupported runner not gated", code, calls, errs.String())
	}
}

func TestRuntimeSettingsCLIHelp(t *testing.T) {
	for _, cmd := range []string{"create-fake", "create-comet", "create-ship", "configure"} {
		var out, errs bytes.Buffer
		if code := run(context.Background(), []string{cmd, "--help"}, &out, &errs); code != 0 {
			t.Fatal(code, errs.String())
		}
		for _, text := range []string{"30..34", "1 GiB", "2 GiB", "16 GiB", "full ship-control", "disconnect", "4 GiB", "resident", "Primarily for development", "Leave disabled for routine use", "unnecessary and adds risk", "known crash bug", "Keep clients connected until every reply has been received", "Do not kill clients or use short timeouts"} {
			if !strings.Contains(out.String(), text) {
				t.Errorf("%s help missing %q", cmd, text)
			}
		}
	}
}
