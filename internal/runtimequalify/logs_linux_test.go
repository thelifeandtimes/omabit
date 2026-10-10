//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"testing"
)

func TestRuntimeLogFramesAndSecretRedaction(t *testing.T) {
	var buf bytes.Buffer
	frame := func(stream byte, text string) {
		h := make([]byte, 8)
		h[0] = stream
		binary.BigEndian.PutUint32(h[4:], uint32(len(text)))
		buf.Write(h)
		buf.WriteString(text)
	}
	frame(1, "2026-10-02T15:00:00Z boot: starting\n")
	frame(2, "http: fail 504\n0w"+strings.Repeat("a", 130)+"\n~lidlut-tabwed-pillex-ridrup\nAuthorization: private-fixture\n\x1b[31mkiln: ready\x1b[0m\n")
	text, err := decodeDockerLogs(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	lines, _ := sanitizeLogs(text)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "http: fail 504") || !strings.Contains(joined, "kiln: ready") || strings.Contains(joined, "0w") || strings.Contains(joined, "lidlut") || strings.Contains(joined, "private-fixture") || strings.Contains(joined, "\x1b") {
		t.Fatal("logs lost stderr or exposed secrets/control bytes")
	}
	bad := append([]byte(nil), buf.Bytes()...)
	bad[0] = 9
	if _, err = decodeDockerLogs(bytes.NewReader(bad)); err == nil {
		t.Fatal("invalid stream accepted")
	}
	if _, err = decodeDockerLogs(bytes.NewReader(buf.Bytes()[:buf.Len()-1])); err == nil {
		t.Fatal("truncated frame accepted")
	}
	lines, truncated := sanitizeLogs(strings.Repeat("boot: step\n", 300))
	if !truncated || len(lines) != 200 {
		t.Fatal("unbounded lines")
	}
}

func TestManagedLogsAuthenticateRetainAndReadStoppedContainers(t *testing.T) {
	f, state, persist := managedFixture(t)
	f.d.logs = func(context.Context, string, string) (string, error) {
		return "boot: ready\n0w" + strings.Repeat("a", 100), nil
	}
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := readManagedLogs(context.Background(), state, f.d)
	if err != nil || logs.Source != "docker" || strings.Contains(strings.Join(logs.Lines, ""), "0w") {
		t.Fatal(logs, err)
	}
	f.editInspect = func(_ string, doc map[string]any) {
		mapAt(mapAt(doc, "Config"), "Labels")["org.urbit-runner.managed-fake.run"] = "replacement"
	}
	if _, err = readManagedLogs(context.Background(), state, f.d); err == nil {
		t.Fatal("foreign runtime logs read")
	}
	f.editInspect = nil
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := readManagedLogs(context.Background(), state, f.d)
	if err != nil || retained.Source != "retained" || retained.ContainerID != logs.ContainerID {
		t.Fatal(retained, err)
	}
}
