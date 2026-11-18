//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

// This fixed, fresh-namespace contract experiment is not a product HTTP API or
// the managed native adapter. Lifecycle and runner bindings remain synthetic.
// It submits each write once, retains dispatch/response hashes, and leaves a
// failed or uncertain outcome intact for the outer supervisor's clean shutdown.
func nativeGrubberyAppContract(t *testing.T, ctx context.Context, f *nativeDeskFixture, owner *eyre.Client, save func(string, any)) {
	nativeGrubberyAppContractGeneration(t, ctx, f, owner, save, false)
}
func nativeGrubberyAppContractV2(t *testing.T, ctx context.Context, f *nativeDeskFixture, owner *eyre.Client, save func(string, any)) {
	nativeGrubberyAppContractGeneration(t, ctx, f, owner, save, true)
}
func nativeGrubberyAppContractGeneration(t *testing.T, ctx context.Context, f *nativeDeskFixture, owner *eyre.Client, save func(string, any), correctedVersion bool) {
	t.Helper()
	save("app-contract-scope", "fresh kernel-only API/source/nexus/state experiment; no public app adapter, permits, routes, peers or installed lifecycle")
	encoded, err := owner.ExportSession()
	g02Check(t, err)
	defer clear(encoded)
	var cookie struct {
		Name  string `json:"cookie_name"`
		Value string `json:"cookie_value"`
	}
	g02Check(t, json.Unmarshal(encoded, &cookie))
	defer func() { cookie.Value = "" }()
	log, err := os.OpenFile("/lab/grubbery-app-contract-effects.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	g02Check(t, err)
	defer log.Close()
	write := func(value any) {
		t.Helper()
		raw, err := json.Marshal(value)
		g02Check(t, err)
		_, err = log.Write(append(raw, '\n'))
		g02Check(t, err)
		g02Check(t, log.Sync())
	}
	calls := 0
	call := func(name, method, path, contentType string, body []byte) (int, []byte) {
		t.Helper()
		g02Check(t, f.check())
		calls++
		write(map[string]any{"sequence": calls, "name": name, "state": "dispatched", "at": time.Now().UTC(), "method": method, "path": path, "bytes": len(body), "sha256": stopEvidenceDigest(body)})
		request, err := http.NewRequestWithContext(ctx, method, "http://127.0.0.1:18081"+path, bytes.NewReader(body))
		g02Check(t, err)
		request.GetBody = nil
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		request.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
		response, err := f.client.Do(request)
		if err != nil {
			save("uncertain-native-call", name)
			t.Fatal("native HTTP outcome unconfirmed; no retry", name)
		}
		raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
		response.Body.Close()
		g02Check(t, err)
		if len(raw) > 1<<20 {
			t.Fatal("bounded native response exceeded")
		}
		g02Check(t, f.check())
		write(map[string]any{"sequence": calls, "name": name, "state": "returned", "status": response.StatusCode, "bytes": len(raw), "sha256": stopEvidenceDigest(raw)})
		return response.StatusCode, raw
	}
	expect := func(name, method, path, contentType string, body []byte, status int) []byte {
		t.Helper()
		got, raw := call(name, method, path, contentType, body)
		if got != status {
			save("app-contract-refusal", map[string]any{"name": name, "status": got, "body": string(raw)})
			t.Fatal("native contract refused", name, got)
		}
		return raw
	}
	// Diagnose the independently reported tool namespace without relying on its
	// broken tools to describe themselves. Keep the report private and bounded.
	tools := expect("nested-tools-build", "GET", "/grubbery/api/build/apps/shell/tools/code?all=1", "", nil, 200)
	save("nested-tools-build-report", json.RawMessage(tools))
	nonce, err := randomID()
	g02Check(t, err)
	sourceRoot := "/omabit-contract-" + nonce
	deskRoot := "/apps/shell/desks/omabit-contract-" + nonce + ".desk"
	marker, _ := json.Marshal(map[string]any{"version": 1, "purpose": "fresh-native-contract-only", "original_request": nonce})
	expect("source-absent", "GET", "/grubbery/api/peek"+sourceRoot+"/", "", nil, 404)
	// Establish whether a single file make can atomically create its plain
	// parent and marker. This is a hypothesis until this exact case passes.
	expect("atomic-parent-marker", "PUT", "/grubbery/api/make"+sourceRoot+"/omabit-owner.json", "application/json", marker, 201)
	got := expect("source-marker-read", "GET", "/grubbery/api/peek"+sourceRoot+"/omabit-owner.json", "", nil, 200)
	if !bytes.Equal(got, marker) {
		t.Fatal("native marker bytes changed")
	}
	save("atomic-plain-parent-and-marker", true)
	candidate := func(label string) []byte {
		return []byte(fmt.Sprintf(`:: fixed native contract candidate; qualified only by exact source hashes
^-  nexus:nexus
|%%
++  on-load
  |=  =ball:tarball
  ^-  bole:tarball
  %%+  spin:loader  ball
  :~  (manifest:loader 0)
      [%%fall %%& [/ %%'state.json'] [[/ %%json] [%%s 'initial']]]
      [%%over %%& [/ %%'label.json'] [[/ %%json] [%%s '%s']]]
  ==
++  on-file
  |=  [=rail:tarball =blot:tarball]
  ^-  spool:fiber:nexus
  |=  =prod:fiber:nexus
  =/  m  (fiber:fiber:nexus ,~)
  ^-  process:fiber:nexus
  ?+  rail  stay:m
      [~ %%'state.json']
    ;<  ~  bind:m  (rise-wait:io prod "omabit state fixture restarted")
    |-
    ;<  =sage:tarball  bind:m  take-poke:io
    ?.  =([/ %%json] p.sage)  $
    ;<  ~  bind:m  (replace:io !<(json q.sage))
    $
  ==
--
`, label))
	}
	type release struct {
		Root     string
		Version  []byte
		Manifest deskwork.Manifest
		Files    []deskwork.SourceFile
	}
	prepare := func(number int) release {
		t.Helper()
		root := fmt.Sprintf("%s/release-%d/code", sourceRoot, number)
		expect(fmt.Sprintf("create-release-%d", number), "PUT", "/grubbery/api/make"+root+"/", "application/json", []byte(`{"neck":"/code"}`), 201)
		files := []deskwork.SourceFile{{Path: "bill.json", Bytes: []byte(`{"state":"/state"}`)}, {Path: "nex/state.hoon", Bytes: candidate(fmt.Sprintf("v%d", number))}}
		entries := []deskwork.File{}
		for _, file := range files {
			entries = append(entries, deskwork.File{Path: file.Path, SHA256: stopEvidenceDigest(file.Bytes), Bytes: int64(len(file.Bytes))})
			query, ct := "", "application/json"
			if strings.HasPrefix(file.Path, "nex/") {
				query = "?blot=%2Fhoon"
				ct = "text/x-hoon"
			}
			expect(fmt.Sprintf("release-%d-%s", number, file.Path), "PUT", "/grubbery/api/make"+root+"/"+file.Path+query, ct, file.Bytes, 201)
		}
		manifest, err := deskwork.NewManifest(entries)
		g02Check(t, err)
		_, err = grubbery.ReviewSource(manifest, files)
		g02Check(t, err)
		save(fmt.Sprintf("candidate-%d", number), map[string]any{"manifest": manifest, "files": files})
		build := expect(fmt.Sprintf("release-%d-build", number), "GET", "/grubbery/api/build"+root+"?all=1", "", nil, 200)
		save(fmt.Sprintf("candidate-%d-build", number), json.RawMessage(build))
		var report struct {
			Namespace string `json:"namespace"`
			Total     int    `json:"total"`
			Failed    int    `json:"failed"`
			Files     []struct {
				Source string `json:"source"`
				Status string `json:"status"`
			} `json:"files"`
		}
		g02Check(t, json.Unmarshal(build, &report))
		if report.Namespace != root || report.Total != len(report.Files) || report.Failed != 0 {
			t.Fatal("candidate compilation is not complete/clean")
		}
		live := false
		for _, file := range report.Files {
			if file.Source == "/nex/state.hoon" && file.Status == "vase" {
				live = true
			}
		}
		if !live {
			t.Fatal("native state nexus did not compile")
		}
		marker := map[string]any{"epoch": number, "source_digest": manifest.Digest}
		if correctedVersion {
			request := stopEvidenceDigest([]byte(nonce + ":" + fmt.Sprint(number)))
			marker["version"] = "omabit:" + request
			marker["request_id"] = request
		}
		version, _ := json.Marshal(marker)
		expect(fmt.Sprintf("release-%d-version", number), "PUT", "/grubbery/api/make"+root+"/version.json", "application/json", version, 201)
		return release{root, version, manifest, files}
	}
	first := prepare(1)
	expect("desk-absent", "GET", "/grubbery/api/peek"+deskRoot+"/", "", nil, 404)
	expect("create-closed-desk-wrapper", "PUT", "/grubbery/api/make"+deskRoot+"/", "application/json", []byte(`{"neck":"/desk"}`), 201)
	expect("desk-original-marker", "PUT", "/grubbery/api/make"+deskRoot+"/desk/data/omabit-owner.json", "application/json", marker, 201)
	follow := func(r release) {
		t.Helper()
		config, _ := json.Marshal(map[string]string{"code": r.Root})
		expect("follow-immutable-source", "POST", "/grubbery/api/poke"+deskRoot+"/source.json?blot=%2Fjson", "application/json", config, 200)
		for deadline := time.Now().Add(25 * time.Second); ; {
			status, raw := call("read-followed-version", "GET", "/grubbery/api/peek"+deskRoot+"/version.json", "", nil)
			if status == 200 {
				var expected, observed any
				g02Check(t, json.Unmarshal(r.Version, &expected))
				if json.Unmarshal(raw, &observed) == nil {
					a, _ := json.Marshal(expected)
					b, _ := json.Marshal(observed)
					if bytes.Equal(a, b) {
						break
					}
				}
			}
			if time.Now().After(deadline) {
				save("unconfirmed-followed-version", string(raw))
				t.Fatal("source acknowledgement did not prove followed version")
			}
			time.Sleep(time.Second)
		}
	}
	follow(first)
	statePath := "/grubbery/api/peek" + deskRoot + "/desk/data/state/state.json"
	initial := expect("initial-state", "GET", statePath, "", nil, 200)
	if string(initial) != `"initial"` {
		t.Fatal("initial app state differs")
	}
	expect("save-state", "POST", "/grubbery/api/poke"+deskRoot+"/desk/data/state/state.json?blot=%2Fjson", "application/json", []byte(`"saved-entry"`), 200)
	saved := expect("read-saved-state", "GET", statePath, "", nil, 200)
	if string(saved) != `"saved-entry"` {
		t.Fatal("state poke not observed")
	}
	expect("known-create-conflict", "PUT", "/grubbery/api/make"+sourceRoot+"/omabit-owner.json", "application/json", []byte(`{"foreign":true}`), 409)
	unchanged := expect("conflict-preserved-marker", "GET", "/grubbery/api/peek"+sourceRoot+"/omabit-owner.json", "", nil, 200)
	if !bytes.Equal(unchanged, marker) {
		t.Fatal("known rejection changed marker")
	}
	second := prepare(2)
	follow(second)
	saved = expect("state-after-source-update", "GET", statePath, "", nil, 200)
	if string(saved) != `"saved-entry"` {
		t.Fatal("code update reset saved state")
	}
	label := expect("new-code-label", "GET", "/grubbery/api/peek"+deskRoot+"/desk/data/state/label.json", "", nil, 200)
	if string(label) != `"v2"` {
		t.Fatal("new nexus code not observed")
	}
	save("app-contract-passed", map[string]any{"atomic_plain_parent_marker": true, "local_nexus_compiled": true, "desk_followed_both_versions": true, "saved_state_preserved_on_update": true, "new_code_label_observed": true, "known_409_unchanged_marker": true, "effect_calls": calls, "source_root": sourceRoot, "desk_root": deskRoot})
}
