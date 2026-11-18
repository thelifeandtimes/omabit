package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRecordingWorkerCLIExactPrivateJobAndFiniteErrors(t *testing.T) {
	id, digest := strings.Repeat("a", 32), strings.Repeat("b", 64)
	for _, scenario := range []string{"pass", "failure", "missing", "duplicate", "root", "uppercase", "path", "trailing", "help"} {
		t.Run(scenario, func(t *testing.T) {
			args := []string{"--job", id, "--sha256", digest}
			want, calls := 2, 0
			switch scenario {
			case "pass":
				want = 0
			case "failure":
				want = 1
			case "missing":
				args = nil
			case "duplicate":
				args = append(args, "--job", id)
			case "root":
				args = append(args, "--root", "/private/other")
			case "uppercase":
				args[1] = strings.ToUpper(id)
			case "path":
				args[1] = "/tmp/private-job"
			case "trailing":
				args = append(args, "extra")
			case "help":
				args, want = []string{"--help"}, 0
			}
			var out bytes.Buffer
			code := runRecordingWorker(t.Context(), args, &out, &out, func(ctx context.Context, gotID, gotDigest string) error {
				calls++
				if ctx != t.Context() || gotID != id || gotDigest != digest {
					t.Fatal("entry rebound original request")
				}
				if scenario == "failure" {
					return errors.New("PRIVATE-JOB-CREDENTIAL")
				}
				return nil
			})
			if code != want || strings.Contains(out.String(), "PRIVATE-JOB-CREDENTIAL") || calls > 1 || calls == 1 && scenario != "pass" && scenario != "failure" {
				t.Fatal("unexpected execution/output", code, calls, out.String())
			}
		})
	}
}

func TestRecordingWorkerCLIUnsupervisedRefusal(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	t.Setenv("INVOCATION_ID", "")
	var out bytes.Buffer
	if code := run(t.Context(), []string{"recording-worker", "--job", strings.Repeat("a", 32), "--sha256", strings.Repeat("b", 64)}, &out, &out); code != 1 {
		t.Fatal("ordinary invocation entered recording", code, out.String())
	}
	if strings.Contains(out.String(), "/home/") || strings.Contains(out.String(), "/srv/") {
		t.Fatal("private root leaked")
	}
}
