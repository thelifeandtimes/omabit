package managed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type creationReplayFixture struct {
	Source   string `json:"source_commit"`
	Schema   int    `json:"schema"`
	Runner   string `json:"runner_id"`
	Requests []struct {
		Path string          `json:"path"`
		Body json.RawMessage `json:"body"`
		CLI  []string        `json:"cli"`
	} `json:"requests"`
	Instances  []json.RawMessage `json:"instances"`
	Operations []struct {
		ID          string          `json:"id"`
		Actor       string          `json:"actor"`
		Request     string          `json:"request_id"`
		Fingerprint string          `json:"fingerprint"`
		Body        json.RawMessage `json:"body"`
	} `json:"operations"`
	Keys map[string]string `json:"keys"`
}

func compactCreationJSON(t *testing.T, data []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := json.Compact(&b, data); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func loadCreationReplayFixture(t *testing.T) (string, creationReplayFixture) {
	t.Helper()
	data, err := os.ReadFile("testdata/rc10_creation_replay.json")
	if err != nil {
		t.Fatal(err)
	}
	var f creationReplayFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	if f.Schema != 19 || f.Source != "f6981e6ce4b9bb1bddc1334ff23a3f1ae9e9794a" || len(f.Requests) != 3 || len(f.Instances) != 3 || len(f.Operations) != 3 {
		t.Fatal("invalid frozen RC.10 fixture")
	}
	dir := privateDir(t)
	path := filepath.Join(dir, "runner.sqlite")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Create the historical schema directly. Never create current instances and
	// relabel them, or derive historical fingerprints using the current encoder.
	_, err = db.Exec(`CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE instances (id TEXT PRIMARY KEY, body BLOB NOT NULL);
CREATE TABLE operations (id TEXT PRIMARY KEY, actor TEXT NOT NULL, request_id TEXT NOT NULL,
fingerprint TEXT NOT NULL, body BLOB NOT NULL, runtime BLOB, UNIQUE(actor, request_id));
CREATE TABLE maintenance_policies (instance_id TEXT PRIMARY KEY REFERENCES instances(id), body BLOB NOT NULL);
CREATE TABLE deleted_instances (id TEXT PRIMARY KEY, operation_id TEXT NOT NULL, revision INTEGER NOT NULL);
PRAGMA user_version=19;`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO meta VALUES ('runner_id',?)", f.Runner); err != nil {
		t.Fatal(err)
	}
	for _, body := range f.Instances {
		var v struct {
			Public struct {
				ID string `json:"id"`
			} `json:"public"`
		}
		if err := json.Unmarshal(body, &v); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("INSERT INTO instances VALUES (?,?)", v.Public.ID, compactCreationJSON(t, body)); err != nil {
			t.Fatal(err)
		}
	}
	for _, op := range f.Operations {
		if _, err := db.Exec("INSERT INTO operations(id,actor,request_id,fingerprint,body) VALUES (?,?,?,?,?)", op.ID, op.Actor, op.Request, op.Fingerprint, compactCreationJSON(t, op.Body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "keys"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, key := range f.Keys {
		if err := os.WriteFile(filepath.Join(dir, "keys", name), []byte(key), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir, f
}

type creationReplayBackend struct{ calls atomic.Int32 }

func (b *creationReplayBackend) Apply(context.Context, string, runtimequalify.ManagedState, func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	b.calls.Add(1)
	return runtimequalify.ManagedState{}, runtimequalify.ManagedObservation{}, errors.New("unexpected runtime dispatch")
}
func (b *creationReplayBackend) Observe(context.Context, runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	b.calls.Add(1)
	return runtimequalify.ManagedState{}, runtimequalify.ManagedObservation{}, errors.New("unexpected runtime observation")
}
func (*creationReplayBackend) PierSources(context.Context) ([]pierstore.Candidate, error) {
	return nil, nil
}

func creationReplayServer(t *testing.T, dir string) *Server {
	t.Helper()
	s, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.mode = DevelopmentMode
	s.portAvailable = func(string, uint16) bool { t.Error("replay probed/allocated a port"); return false }
	s.validateStorage = func(string) error {
		t.Error("replay consulted current storage")
		return errors.New("fixture storage offline")
	}
	b := &creationReplayBackend{}
	t.Cleanup(func() {
		if b.calls.Load() != 0 {
			t.Error("replay dispatched runtime work")
		}
		s.close()
	})
	return &Server{s: s, b: b, ctx: context.Background(), uid: 1000}
}

func creationReplayState(t *testing.T, s *store) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, table := range []string{"instances", "operations"} {
		rows, err := s.db.Query("SELECT id,body FROM " + table + " ORDER BY id")
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var id string
			var body []byte
			if err := rows.Scan(&id, &body); err != nil {
				t.Fatal(err)
			}
			result[table+"/"+id] = string(body)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
	}
	rows, err := s.db.Query("SELECT id,actor,request_id,fingerprint,runtime FROM operations ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, actor, request, fingerprint string
		var runtime []byte
		if err := rows.Scan(&id, &actor, &request, &fingerprint, &runtime); err != nil {
			t.Fatal(err)
		}
		result["receipt/"+id] = actor + "\x00" + request + "\x00" + fingerprint + "\x00" + string(runtime)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	for _, name := range []string{"keys", "boot-pills"} {
		root := filepath.Join(s.directory, name)
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if errors.Is(err, os.ErrNotExist) && path == root {
				return nil
			}
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(s.directory, path)
			if err != nil {
				return err
			}
			if entry.IsDir() {
				result["file/"+rel] = "directory"
				return nil
			}
			data, err := os.ReadFile(path)
			if err == nil {
				result["file/"+rel] = string(data)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func assertFrozenCreationRows(t *testing.T, state map[string]string, fixture creationReplayFixture) {
	t.Helper()
	for _, body := range fixture.Instances {
		var v struct {
			Public struct {
				ID string `json:"id"`
			} `json:"public"`
		}
		if err := json.Unmarshal(body, &v); err != nil {
			t.Fatal(err)
		}
		if state["instances/"+v.Public.ID] != string(compactCreationJSON(t, body)) {
			t.Fatal("upgrade rewrote historical instance", v.Public.ID)
		}
	}
	for _, op := range fixture.Operations {
		if state["operations/"+op.ID] != string(compactCreationJSON(t, op.Body)) || state["receipt/"+op.ID] != op.Actor+"\x00"+op.Request+"\x00"+op.Fingerprint+"\x00" {
			t.Fatal("upgrade rewrote historical receipt", op.ID)
		}
	}
	for name, key := range fixture.Keys {
		if state["file/keys/"+name] != key {
			t.Fatal("upgrade changed synthetic boot key")
		}
	}
}

func creationReplayCall(t *testing.T, r *Server, path string, body []byte, header []string, grant string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set(protocol.Header, "1")
	req.Header.Set("Content-Type", "application/json")
	if header != nil {
		req.Header[http.CanonicalHeaderKey(protocol.CreationReplayHeader)] = header
	}
	if grant != "" {
		req.Header.Set(access.Header, grant)
	}
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(0)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func creationReplayGrant(t *testing.T, dir string) (string, string) {
	t.Helper()
	line, grant := grantLine(t, "operator")
	path := filepath.Join(dir, "fixture-grants")
	if err := os.WriteFile(path, []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	return path, grant
}

func TestRC11CreationReplaySchema19HTTPGuards(t *testing.T) {
	dir, fixture := loadCreationReplayFixture(t)
	r := creationReplayServer(t, dir)
	keys, otherActor := creationReplayGrant(t, dir)
	r.keys = keys
	before := creationReplayState(t, r.s)
	assertFrozenCreationRows(t, before, fixture)
	for _, item := range fixture.Requests {
		t.Run(item.Path, func(t *testing.T) {
			var original map[string]any
			if err := json.Unmarshal(item.Body, &original); err != nil {
				t.Fatal(err)
			}
			var expected []byte
			for _, op := range fixture.Operations {
				if op.Request == original["request_id"] {
					expected = compactCreationJSON(t, op.Body)
				}
			}
			for _, header := range [][]string{nil, {"true"}} {
				w := creationReplayCall(t, r, item.Path, item.Body, header, "")
				if w.Code != 200 || !bytes.Equal(compactCreationJSON(t, w.Body.Bytes()), expected) {
					t.Fatal("historical replay changed", w.Code, w.Body.String())
				}
			}
			for _, header := range [][]string{{""}, {"false"}, {"TRUE"}, {"1"}, {"true,false"}, {"true", "true"}, {"true", "false"}, {}} {
				w := creationReplayCall(t, r, item.Path, item.Body, header, "")
				if w.Code != 400 {
					t.Fatal("malformed replay header accepted", header, w.Code, w.Body.String())
				}
			}
			w := creationReplayCall(t, r, item.Path, item.Body, []string{"true"}, otherActor)
			if w.Code != 409 || !strings.Contains(w.Body.String(), "replay_not_found") {
				t.Fatal("receipt crossed actors", w.Code, w.Body.String())
			}
			for _, change := range []string{"request_id", "label", "key", "pill"} {
				if change == "key" && item.Path != "/v1/ships" {
					continue
				}
				var body map[string]any
				json.Unmarshal(item.Body, &body)
				if change == "pill" {
					pill := map[string]any{"sha256": strings.Repeat("a", 64), "bytes": 12}
					if item.Path == "/v1/ships" {
						body["pill"] = pill
					} else {
						body["bootstrap"] = map[string]any{"kind": "recommended", "pill": pill}
					}
				} else if change == "key" {
					body["key"] = "0wdifferent.synthetic.key"
				} else {
					body[change] = "changed"
				}
				data, _ := json.Marshal(body)
				w := creationReplayCall(t, r, item.Path, data, []string{"true"}, "")
				code := "request_conflict"
				if change == "request_id" {
					code = "replay_not_found"
				}
				if w.Code != 409 || !strings.Contains(w.Body.String(), code) {
					t.Fatal("changed input not refused", change, w.Code, w.Body.String())
				}
			}
			// Even an otherwise selectable fresh loom31 request is lookup-only:
			// no ports, staging checks or synthetic keys may be created.
			original["request_id"] = "new-default"
			original["config"].(map[string]any)["loom"] = float64(31)
			data, _ := json.Marshal(original)
			w = creationReplayCall(t, r, item.Path, data, []string{"true"}, "")
			if w.Code != 409 || !strings.Contains(w.Body.String(), "replay_not_found") {
				t.Fatal("guard permitted new work", w.Code, w.Body.String())
			}
		})
	}
	if after := creationReplayState(t, r.s); !reflect.DeepEqual(before, after) {
		t.Fatal("replay/refusal changed stored instances, receipts, keys or pills")
	}
	if r.failed.Load() {
		t.Fatal("request-local refusal latched store failure")
	}
}

func TestRC11CreationReplaySchema19UpgradeAndCollectorCLI(t *testing.T) {
	dir, fixture := loadCreationReplayFixture(t)
	bin := filepath.Join(t.TempDir(), "collector")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	// This fixture tests CLI replay, including from an exported source archive.
	// Release provenance is checked separately by the artifact packaging tests.
	build := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-o", bin, "../../backend/cmd/urbit-collector")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("collector build: %v %s", err, out)
	}
	profile := filepath.Join(t.TempDir(), "collector-config")
	keyFile := filepath.Join(t.TempDir(), "synthetic.key")
	if err := os.WriteFile(keyFile, []byte("0wfixture.synthetic.only"), 0600); err != nil {
		t.Fatal(err)
	}
	pillFile := filepath.Join(t.TempDir(), "synthetic.pill")
	if err := os.WriteFile(pillFile, []byte("synthetic replay-only pill"), 0600); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		r := creationReplayServer(t, dir)
		var version int
		if err := r.s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != StateSchemaVersion {
			t.Fatal("historical store did not upgrade", version, err)
		}
		if r.s.runnerID != fixture.Runner {
			t.Fatal("stable runner identity changed")
		}
		before := creationReplayState(t, r.s)
		assertFrozenCreationRows(t, before, fixture)
		socket := filepath.Join(privateDir(t), "rpc.sock")
		listener, err := localrpc.Listen(socket)
		if err != nil {
			t.Fatal(err)
		}
		httpServer := &http.Server{Handler: r, ReadHeaderTimeout: time.Second}
		// Attribute the isolated local test peer to the frozen receipt's UID0,
		// preserving the actual historical operation/fingerprint bytes on any host.
		httpServer.ConnContext = func(ctx context.Context, _ net.Conn) context.Context {
			return context.WithValue(ctx, peerKey{}, uint32(0))
		}
		var creates atomic.Int32
		httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPost {
				creates.Add(1)
				if req.Header.Get(protocol.CreationReplayHeader) != "true" {
					t.Error("collector omitted replay-only guard")
				}
			}
			r.ServeHTTP(w, req)
		})
		done := make(chan error, 1)
		go func() { done <- httpServer.Serve(listener) }()
		t.Cleanup(func() { httpServer.Close() })
		invoke := func(args ...string) ([]byte, error) {
			cmd := exec.CommandContext(ctx, bin, append([]string{"--config-dir", profile}, args...)...)
			return cmd.CombinedOutput()
		}
		name := fmt.Sprintf("fixture%d", pass)
		// Each pass uses its own profile entry while preserving the runner pin.
		if pass != 0 {
			profile = filepath.Join(t.TempDir(), "collector-config")
		}
		if out, err := invoke("enroll", name, "--socket", socket); err != nil {
			t.Fatalf("enroll: %v %s", err, out)
		}
		prefix := []string{"--runner", name, "--expected-runner-id", fixture.Runner}
		if out, err := invoke(append(prefix, "hello")...); err != nil {
			t.Fatal(err, string(out))
		} else {
			var hello protocol.Hello
			if err := json.Unmarshal(out, &hello); err != nil || !slices.Contains(hello.Features, protocol.CreationReplayFeature) {
				t.Fatal("capability not advertised", err)
			}
		}
		for _, item := range fixture.Requests {
			args := append([]string(nil), item.CLI...)
			if item.Path == "/v1/ships" {
				args = append(args, "--key-file", keyFile)
			}
			out, err := invoke(append(prefix, args...)...)
			if err != nil {
				t.Fatalf("historical CLI retry %s: %v %s", item.Path, err, out)
			}
			var result protocol.Operation
			if err := json.Unmarshal(out, &result); err != nil {
				t.Fatal(err)
			}
			matched := false
			for _, op := range fixture.Operations {
				if op.Request == result.RequestID {
					matched = true
					if !bytes.Equal(compactCreationJSON(t, out), compactCreationJSON(t, op.Body)) {
						t.Fatal("collector changed historical receipt")
					}
				}
			}
			if !matched {
				t.Fatal("collector invented receipt")
			}
			for i, arg := range args {
				if strings.HasPrefix(arg, "--request-id=") {
					args[i] = "--request-id=new-small"
				}
			}
			out, err = invoke(append(prefix, args...)...)
			if err == nil || !bytes.Contains(out, []byte("replay_not_found")) {
				t.Fatal("fresh small-loom CLI request admitted", err, string(out))
			}
			// A selected custom pill still contributes its original hash/size to
			// the request, but must not create a cache entry before refusal.
			out, err = invoke(append(prefix, append(args, "--pill", pillFile)...)...)
			if err == nil || !bytes.Contains(out, []byte("replay_not_found")) {
				t.Fatal("replay-only pill selection did not reach guarded lookup", err, string(out))
			}
		}
		if creates.Load() != 9 {
			t.Fatal("unexpected creation dispatch count", creates.Load())
		}
		if after := creationReplayState(t, r.s); !reflect.DeepEqual(before, after) {
			t.Fatal("CLI retry/refusal changed persistent state")
		}
		if err := httpServer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			t.Fatal(err)
		}
		if err := r.s.close(); err != nil {
			t.Fatal(err)
		}
	}
}
