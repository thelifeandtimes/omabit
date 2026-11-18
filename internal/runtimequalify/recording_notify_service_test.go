//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

var recordingNotifySmokeSource string

// Explicit inert unit primitive only. Never opens a manager, Docker, job store,
// recording session, user's process or pier. Ordinary source tests always skip.
func TestRecordingNotificationDisposableService(t *testing.T) {
	root := os.Getenv("OMABIT_RECORDING_NOTIFY_SMOKE_ROOT")
	if root == "" {
		t.Skip("explicit disposable notification service only")
	}
	if !regexp.MustCompile(`^/tmp/omabit-recording-notify-service-[0-9]{2}$`).MatchString(root) || os.Geteuid() == 0 || !recordingJobHex(recordingNotifySmokeSource, 40) || !recordingJobHex(os.Getenv("INVOCATION_ID"), 32) {
		t.Fatal("refused outside fresh inert supervised service")
	}
	info, err := os.Lstat(root)
	if err != nil || !recordingJobDirectory(info) {
		t.Fatal("unsafe private smoke root", err)
	}
	var scenario struct {
		Source string `json:"source"`
		Kind   string `json:"kind"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "scenario.json"))
	if err != nil || json.Unmarshal(raw, &scenario) != nil || scenario.Source != recordingNotifySmokeSource || scenario.Kind != "inert-recording-notify-v1" {
		t.Fatal("wrong frozen notification case", err)
	}
	worker, err := observeRecordingWorkerIdentity(&qualifier{d: defaults()})
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(filepath.Join(root, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write(data); err != nil {
			t.Fatal(err)
		}
		if err = file.Sync(); err != nil {
			t.Fatal(err)
		}
		if err = file.Close(); err != nil {
			t.Fatal(err)
		}
		directory, err := os.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		if err = directory.Sync(); err != nil {
			t.Fatal(err)
		}
		directory.Close()
	}
	write("before-notify.json", worker)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	if err = NotifyRecordingSupervisor(ctx); err != nil {
		t.Fatal(err)
	}
	write("after-notify.json", worker)
	for {
		if marker, e := os.Lstat(filepath.Join(root, "launcher-returned")); e == nil {
			if !recordingJobFileInfo(marker, 0) || marker.Size() != 0 {
				t.Fatal("unsafe launcher marker")
			}
			break
		} else if !os.IsNotExist(e) {
			t.Fatal(e)
		}
		select {
		case <-ctx.Done():
			t.Fatal("launcher completion marker unavailable")
		case <-time.After(50 * time.Millisecond):
		}
	}
	write("continued-after-launcher.json", worker)
}
