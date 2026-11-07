//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func TestDeskFrontendReadsUseBoundRuntimeAndStrictDecoding(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
	if err = os.WriteFile(filepath.Join(managedPier(f, state), ".http.ports"), []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
		t.Fatal(err)
	}
	response, calls := "", 0
	f.d.executeInput = func(_ context.Context, args, _ []string, input []byte) (string, error) {
		calls++
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "http://127.0.0.1:12322") || strings.Contains(joined, "index.html") || strings.Contains(joined, "PRIVATE") {
			t.Fatal("wrong transport or input in argv")
		}
		var req struct {
			Source map[string]json.RawMessage `json:"source"`
			Sink   map[string]json.RawMessage `json:"sink"`
		}
		if json.Unmarshal(input, &req) != nil || len(req.Sink) != 1 || string(req.Sink["stdout"]) != "null" {
			t.Fatal("not a read-only typed call")
		}
		if string(req.Source["dojo"]) == `"our"` {
			return `"~zod\n"`, nil
		}
		b, _ := json.Marshal(response)
		return string(b), nil
	}
	runtime, err := openManagedDesk(context.Background(), state, id, f.d)
	if err != nil {
		t.Fatal(err)
	}
	atom := func(value any) string {
		b, _ := json.Marshal(value)
		for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
			b[i], b[j] = b[j], b[i]
		}
		return "0x" + hex.EncodeToString(b)
	}
	ref := deskwork.FrontendReference{Mode: "glob-ames", Base: "other-base", Ship: "~zod", GlobHash: "0v1", DocketHash: "0v2"}
	want := deskwork.FrontendState{SourceRevision: 3, DocketAvailable: true, Stage: "glob", Declared: ref, Current: ref}
	response = atom(want)
	got, err := runtime.Frontend(context.Background(), "example")
	if err != nil || got != want || calls != 3 {
		t.Fatal("frontend observation changed", err, calls)
	}
	want.SourceRevision = 0
	response = atom(want)
	if _, err = runtime.Frontend(context.Background(), "example"); err == nil {
		t.Fatal("invalid observation accepted")
	}
	files := []deskwork.SourceFile{{Path: "index.html", Bytes: []byte("PRIVATE fixture frontend")}}
	h := sha256.Sum256(files[0].Bytes)
	source, err := deskwork.NewManifest([]deskwork.File{{Path: "index.html", Bytes: int64(len(files[0].Bytes)), SHA256: hex.EncodeToString(h[:])}})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := deskwork.NewFrontendManifest(source)
	if err != nil {
		t.Fatal(err)
	}
	response = atom("0v1")
	hash, err := runtime.FrontendHash(context.Background(), artifact, files)
	if err != nil || hash != "0v1" {
		t.Fatal("artifact hash unavailable", err)
	}
	response = atom("PRIVATE invalid hash")
	if _, err = runtime.FrontendHash(context.Background(), artifact, files); err == nil || strings.Contains(err.Error(), "PRIVATE") {
		t.Fatal("unsafe hash response", err)
	}
	before := calls
	files[0].Bytes = []byte("different")
	if _, err = runtime.FrontendHash(context.Background(), artifact, files); err == nil || calls != before {
		t.Fatal("changed artifact reached runtime")
	}
}
