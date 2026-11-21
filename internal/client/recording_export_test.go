package client

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type exportRoundTrip func(*http.Request) (*http.Response, error)

func (f exportRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRecordingExportManifestHasScopedLongDeadlineAndNoRetry(t *testing.T) {
	calls := 0
	c := &Client{httpClient: &http.Client{Timeout: 5 * time.Second, Transport: exportRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) < protocol.RecordingExportTimeout || time.Until(deadline) > protocol.RecordingExportTimeout+3*time.Second {
			t.Error("manifest inherited ordinary five-second timeout")
		}
		return nil, errors.New("injected read disconnect")
	})}}
	in := recording.ExportRequest{RunnerID: "runner", InstanceID: "alpha", JobID: strings.Repeat("b", 32), JobSHA256: strings.Repeat("c", 64), Representation: "original", IncludePrivateData: true}
	if _, err := c.RecordingExportManifest(t.Context(), in); err == nil || calls != 1 || c.httpClient.Timeout != 5*time.Second {
		t.Fatal("read retried or changed shared client timeout", err, calls)
	}
	part := recording.ExportPartRequest{ExportRequest: in, SessionID: strings.Repeat("d", 32), MetadataSHA256: strings.Repeat("e", 64), FileName: "frames.obr", FileBytes: 1, ChunkSHA256: strings.Repeat("f", 64)}
	if _, err := c.RecordingExportPart(t.Context(), part); err == nil || calls != 2 || c.httpClient.Timeout != 5*time.Second {
		t.Fatal("part retried or changed ordinary client deadline", err, calls)
	}
	in.Representation = "redacted"
	if _, err := c.RecordingExportManifest(t.Context(), in); err == nil || calls != 2 {
		t.Fatal("redacted choice contacted original exporter")
	}
}
