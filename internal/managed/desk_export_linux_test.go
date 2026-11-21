//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type exportRuntimeFixture struct {
	*commitRuntimeFixture
	scenario string
	exports  int
}

func (f *exportRuntimeFixture) ExportSourceFiles(_ context.Context, desk string, revision uint64, paths []string) ([]deskwork.ExportedFile, error) {
	f.exports++
	if desk != "example" || revision != 1 || !reflect.DeepEqual(paths, []string{"hello.txt"}) {
		return nil, errors.New("wrong export selection")
	}
	if f.scenario == "error" {
		return nil, errors.New("private arbitrary failure must not escape")
	}
	if f.scenario == "clay" {
		f.observation.Revision++
	}
	if f.scenario == "pier" {
		f.continuity = "changed"
	}
	data := []byte{0, 255, 10}
	digest := sha256.Sum256(data)
	file := deskwork.ExportedFile{Path: paths[0], ContentBase64: base64.StdEncoding.EncodeToString(data), SHA256: hex.EncodeToString(digest[:])}
	if f.scenario == "path" {
		file.Path = "unselected.txt"
	}
	if f.scenario == "bytes" {
		file.ContentBase64 = base64.StdEncoding.EncodeToString([]byte("changed"))
	}
	return []deskwork.ExportedFile{file}, nil
}

type exportBackendFixture struct {
	*commitBackendFixture
	export *exportRuntimeFixture
}

func (b *exportBackendFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	return b.export, nil
}

func TestDeskSourceExportBracketsBindingsAndNeverMutates(t *testing.T) {
	for _, scenario := range []string{"ok", "stale", "runtime", "clay", "pier", "path", "bytes", "error", "invalid", "observer", "method"} {
		t.Run(scenario, func(t *testing.T) {
			r, f, _ := commitServerFixture(t)
			backend := r.b.(*commitBackendFixture)
			export := &exportRuntimeFixture{commitRuntimeFixture: f, scenario: scenario}
			r.b = &exportBackendFixture{backend, export}
			policy, err := loadAppPolicy(r.s.db)
			g02Check(t, err)
			q := protocol.DeskSourceExportRequest{Selection: deskSelection(deskIntentFixture()), Baseline: f.observation, Paths: []string{"hello.txt"}}
			want := 200
			switch scenario {
			case "stale":
				q.Baseline.Revision++
				want = 409
			case "runtime":
				q.Selection.RuntimeID = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
				want = 409
			case "clay", "pier", "path", "bytes", "error":
				want = 409
			case "invalid":
				q.Paths = []string{"../secret.txt"}
				want = 400
			case "observer":
				want = 403
			case "method":
				want = 405
			}
			var beforeOps int
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM desk_operations").Scan(&beforeOps))
			var got *httptest.ResponseRecorder
			if scenario == "observer" || scenario == "method" {
				body, _ := json.Marshal(q)
				got = httptest.NewRecorder()
				role, method := "operator", "POST"
				if scenario == "observer" {
					role = "observer"
				}
				if scenario == "method" {
					method = "GET"
				}
				r.serveDeskSourceExport(got, httptest.NewRequest(method, protocol.DeskSourceExportPath, bytes.NewReader(body)), role)
			} else {
				got = deskHTTPRequest(t, r, "POST", protocol.DeskSourceExportPath, q)
			}
			if got.Code != want {
				t.Fatal(got.Code, got.Body.String())
			}
			if want == 200 {
				var out protocol.DeskSourceExport
				g02Check(t, json.Unmarshal(got.Body.Bytes(), &out))
				g02Check(t, out.Validate())
				raw, e := out.Files[0].Decode()
				if e != nil || !reflect.DeepEqual(out.Request, q) || !bytes.Equal(raw, []byte{0, 255, 10}) {
					t.Fatal("selected original bytes changed")
				}
			}
			if bytes.Contains(got.Body.Bytes(), []byte("private arbitrary")) {
				t.Fatal("unsanitized backend error escaped")
			}
			if scenario == "stale" || scenario == "runtime" || scenario == "invalid" || scenario == "observer" || scenario == "method" {
				if export.exports != 0 {
					t.Fatal("refused selection read source")
				}
			}
			var afterOps int
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM desk_operations").Scan(&afterOps))
			afterPolicy, err := loadAppPolicy(r.s.db)
			g02Check(t, err)
			if beforeOps != afterOps || f.calls != 0 || !reflect.DeepEqual(policy, afterPolicy) {
				t.Fatal("source read mutated journal/runtime/policy")
			}
		})
	}
}
