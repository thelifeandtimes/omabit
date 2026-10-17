package main

import (
	"bytes"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRuntimeSettingsTextOutput(t *testing.T) {
	loom, native := uint8(31), false
	instance := protocol.Instance{ID: "dev", Config: protocol.Configuration{Loom: 34, NativeControl: true},
		RuntimeStatus: &protocol.RuntimeStatus{LaunchLoom: &loom, LaunchNativeControl: &native, NativeControlSocket: "/private/pier/.urb/conn.sock\x1b"}}
	var out bytes.Buffer
	if err := newOutput(&out, "text", false).Encode(instance); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Requested loom: 34 (16 GiB)", "Recorded launch loom: 31 (2 GiB)", "Requested native control: enabled", "Recorded launch native control: disabled", "Host-local socket path: /private/pier/.urb/conn.sock", "not resident memory", "path only"} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("text output omits %q: %s", text, out.String())
		}
	}
	if strings.ContainsRune(out.String(), '\x1b') {
		t.Fatal("socket path injected terminal control")
	}
}

func TestRuntimeSettingsLoomLabels(t *testing.T) {
	for _, tc := range []struct {
		exponent uint8
		label    string
	}{{20, "20 (1 MiB)"}, {29, "29 (512 MiB)"}, {30, "30 (1 GiB)"}, {31, "31 (2 GiB)"}, {34, "34 (16 GiB)"}, {0, "unknown"}, {35, "unknown"}} {
		var out bytes.Buffer
		if err := newOutput(&out, "text", false).Encode(protocol.Instance{Config: protocol.Configuration{Loom: tc.exponent}}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Requested loom: "+tc.label) {
			t.Errorf("exponent %d: %s", tc.exponent, out.String())
		}
		if strings.Contains(out.String(), "Recorded launch loom:") {
			t.Fatal("missing runtime state became a claimed launch")
		}
	}
}
