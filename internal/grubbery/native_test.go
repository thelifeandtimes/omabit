package grubbery_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

// An inert model of the pinned file/directory HTTP contract, not a source of
// ownership or compiler health. State and replies are deliberately independent
// of the production projection/decoder. No ship, filesystem or HTTP effect.
type nativeModel struct {
	t                  *testing.T
	dirs               map[string]string
	files              map[string][]byte
	marks              map[string]string
	writes             map[string]int
	reads              int
	lost, reject, hide bool
	cleanup            string
	foundations        map[string][]byte
}

func newNativeModel(t *testing.T) *nativeModel {
	t.Helper()
	f := &nativeModel{t: t, dirs: map[string]string{}, files: map[string][]byte{}, marks: map[string]string{}, writes: map[string]int{}, cleanup: "confirmed", foundations: map[string][]byte{}}
	framework, err := appdev.ReadFramework(appdev.GrubberyFramework)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range foundationFixture {
		data, err := os.ReadFile("testdata/foundations/" + path.Base(name))
		if err != nil {
			t.Fatal(err)
		}
		pin := framework.Source.Index()["gub/"+name]
		if hashRaw(data) != pin.SHA256 || int64(len(data)) != pin.Bytes {
			t.Fatal("upstream foundational fixture changed")
		}
		f.foundations[name] = data
	}
	return f
}
func hashRaw(raw []byte) string { s := sha256.Sum256(raw); return hex.EncodeToString(s[:]) }
func (f *nativeModel) mkdir(name, neck string) {
	for parent := path.Dir(name); parent != "/" && parent != "."; parent = path.Dir(parent) {
		if _, ok := f.dirs[parent]; !ok {
			f.dirs[parent] = ""
		}
	}
	f.dirs[name] = neck
	if neck == "code" {
		f.dirs[name+"/mar"] = ""
		for p, data := range f.foundations {
			f.put(name+"/"+p, "hoon", data)
		}
	}
	if neck == "desk" {
		f.mkdir(name+"/desk", "")
		f.mkdir(name+"/desk/code", "code")
		f.mkdir(name+"/desk/data", "")
		f.put(name+"/source.json", "json", []byte("null"))
		f.put(name+"/version.json", "json", []byte("null"))
	}
}
func (f *nativeModel) put(name, mark string, raw []byte) {
	if _, ok := f.dirs[path.Dir(name)]; !ok {
		f.mkdir(path.Dir(name), "")
	}
	f.files[name] = append([]byte{}, raw...)
	f.marks[name] = mark
}
func (f *nativeModel) tree(root string) map[string]any {
	var neck any
	if f.dirs[root] != "" {
		neck = f.dirs[root]
	}
	files, dirs := map[string]string{}, map[string]any{}
	for p, mark := range f.marks {
		if path.Dir(p) == root {
			files[path.Base(p)] = mark
		}
	}
	for p := range f.dirs {
		if path.Dir(p) == root {
			dirs[path.Base(p)] = f.tree(p)
		}
	}
	return map[string]any{"neck": neck, "files": files, "dirs": dirs}
}
func (f *nativeModel) reply(status int, raw []byte) grubbery.NativeResponse {
	return grubbery.NativeResponse{Attempted: true, SessionCleanup: f.cleanup, Status: status, Body: append([]byte{}, raw...), BodySHA256: hashRaw(raw)}
}
func (f *nativeModel) GrubberyRead(_ context.Context, r grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	f.reads++
	q, err := r.Request()
	if err != nil {
		return grubbery.NativeResponse{}, err
	}
	if f.hide {
		return f.reply(404, []byte("Not found")), nil
	}
	p := strings.TrimPrefix(q.Path, "/grubbery/api/peek")
	if q.Query == "as=tree" {
		p = strings.TrimSuffix(p, "/")
		if _, ok := f.dirs[p]; ok {
			raw, _ := json.Marshal(f.tree(p))
			return f.reply(200, raw), nil
		}
	}
	if strings.HasPrefix(q.Path, "/grubbery/api/build") {
		root := strings.TrimPrefix(q.Path, "/grubbery/api/build")
		rows := []any{}
		keys := []string{}
		for p := range f.files {
			if strings.HasPrefix(p, root+"/") {
				keys = append(keys, p)
			}
		}
		sort.Strings(keys)
		for _, p := range keys {
			status := "mime"
			if f.marks[p] == "hoon" {
				status = "vase"
			}
			rows = append(rows, map[string]any{"source": strings.TrimPrefix(p, root), "status": status, "error": nil})
		}
		raw, _ := json.Marshal(map[string]any{"namespace": root, "total": len(rows), "failed": 0, "files": rows})
		return f.reply(200, raw), nil
	}
	if raw, ok := f.files[p]; ok {
		return f.reply(200, raw), nil
	}
	return f.reply(404, []byte("Not found")), nil
}
func (f *nativeModel) GrubberyStep(_ context.Context, p grubbery.Plan, id string, files []deskwork.SourceFile) (grubbery.NativeResponse, error) {
	q, err := p.Request(id, files)
	if err != nil {
		return grubbery.NativeResponse{}, err
	}
	if q.Method == "GET" { // Native reads share the same independent decoder model.
		reads, err := p.AfterReads(id)
		if err != nil {
			return grubbery.NativeResponse{}, err
		}
		return f.GrubberyRead(context.Background(), reads[0])
	}
	f.writes[id]++
	if f.reject {
		return f.reply(409, []byte("Already exists")), grubbery.ErrNativeRejected
	}
	if q.Method == "PUT" {
		name := strings.TrimPrefix(q.Path, "/grubbery/api/make")
		if strings.HasSuffix(name, "/") {
			name = strings.TrimSuffix(name, "/")
			if _, ok := f.dirs[name]; ok {
				return f.reply(409, []byte("Already exists")), grubbery.ErrNativeRejected
			}
			var body struct {
				Neck string `json:"neck"`
			}
			json.Unmarshal(q.Body, &body)
			f.mkdir(name, strings.TrimPrefix(body.Neck, "/"))
		} else {
			if _, ok := f.files[name]; ok {
				return f.reply(409, []byte("Already exists")), grubbery.ErrNativeRejected
			}
			mark := "mime"
			if q.Query == "blot=%2Fhoon" {
				mark = "hoon"
			}
			f.put(name, mark, q.Body)
		}
	} else {
		name := strings.TrimPrefix(q.Path, "/grubbery/api/poke")
		var config struct {
			Code string `json:"code"`
		}
		if json.Unmarshal(q.Body, &config) != nil || config.Code == "" {
			f.t.Fatal("untyped source poke")
		}
		f.put(name, "json", q.Body)
		desk := strings.TrimSuffix(name, "/source.json")
		mirror := desk + "/desk/code"
		for p := range f.files {
			if strings.HasPrefix(p, mirror+"/") {
				delete(f.files, p)
				delete(f.marks, p)
			}
		}
		for p := range f.dirs {
			if strings.HasPrefix(p, mirror+"/") {
				delete(f.dirs, p)
			}
		}
		for p, neck := range f.dirs {
			if strings.HasPrefix(p, config.Code+"/") {
				f.dirs[mirror+strings.TrimPrefix(p, config.Code)] = neck
			}
		}
		for p, data := range f.files {
			if strings.HasPrefix(p, config.Code+"/") {
				f.put(mirror+strings.TrimPrefix(p, config.Code), f.marks[p], data)
			}
		}
		f.put(desk+"/version.json", "json", f.files[config.Code+"/version.json"])
	}
	if f.lost {
		f.lost = false
		return grubbery.NativeResponse{}, errors.New("lost native reply")
	}
	if q.Method == "PUT" {
		return f.reply(201, []byte("Created")), nil
	}
	return f.reply(200, []byte("OK")), nil
}

type nativeHarness struct {
	model               *nativeModel
	journal             *memoryJournal
	backend             *grubbery.NativeBackend
	files               []deskwork.SourceFile
	sourceCalls         int
	deny, sourceMissing bool
	revision            uint64
}

func nativeHarnessFor(t *testing.T, p grubbery.Plan, files []deskwork.SourceFile) *nativeHarness {
	t.Helper()
	h := &nativeHarness{model: newNativeModel(t), journal: newJournal(), files: files, revision: p.Binding.PolicyRevision}
	history := func(context.Context, grubbery.Binding) ([]grubbery.Record, error) {
		h.journal.mu.Lock()
		defer h.journal.mu.Unlock()
		ids := []string{}
		for id := range h.journal.records {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out := []grubbery.Record{}
		for _, id := range ids {
			out = append(out, copyRecord(h.journal.records[id]))
		}
		return out, nil
	}
	guard := func(context.Context, grubbery.Binding, bool) (uint64, error) {
		if h.deny {
			return 0, errors.New("role/policy/runtime/resource changed")
		}
		return h.revision, nil
	}
	load := func(context.Context, string) ([]deskwork.SourceFile, error) {
		h.sourceCalls++
		if h.sourceMissing {
			return nil, grubbery.ErrSource
		}
		return h.files, nil
	}
	var err error
	h.backend, err = grubbery.NewNativeBackend(h.model, history, guard, load)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
func (h *nativeHarness) execution() grubbery.Execution {
	return grubbery.Execution{Store: h.journal, Backend: h.backend, LoadSource: func(ctx context.Context, id string) ([]deskwork.SourceFile, error) {
		h.sourceCalls++
		if h.sourceMissing {
			return nil, grubbery.ErrSource
		}
		return h.files, nil
	}}
}
func callFor(p grubbery.Plan, id string) grubbery.Call {
	return grubbery.Call{RunnerID: p.Binding.RunnerID, Mode: p.Binding.Mode, Actor: p.Binding.Actor, Target: p.Binding.Target, AppID: p.Binding.AppID, RequestID: p.RequestID, PlanDigest: p.Digest, StepID: id}
}
func nativePlan(t *testing.T, h *nativeHarness, prior grubbery.Plan, request string) grubbery.Plan {
	t.Helper()
	before, err := h.backend.Inspect(t.Context(), prior.Binding)
	if err != nil {
		t.Fatal("native baseline", err)
	}
	p, err := grubbery.NewPlan(prior.Binding, prior.StageID, request, prior.Source.Manifest, h.files, before)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func nativeAdvance(t *testing.T, h *nativeHarness, p grubbery.Plan, step grubbery.Step) grubbery.Record {
	t.Helper()
	r, err := h.execution().Advance(t.Context(), callFor(p, step.ID))
	if err != nil {
		t.Fatal("advance", step.Kind, step.Path, err)
	}
	state, _ := r.State(step.ID)
	if state != "observed" {
		t.Fatal("unobserved phase", step.Kind, step.Path, state)
	}
	return r
}

func TestNativeCoordinatorCompleteUpdatesRetainOriginalOwnershipAndGeneratedDirectory(t *testing.T) {
	prior, files := plan(t)
	// A custom mark requires the framework-generated mar directory. It is not
	// one of the four protected foundational paths.
	m, files := source(t, `{"state":"/state"}`, deskwork.SourceFile{Path: "mar/custom.hoon", Bytes: []byte(":: inert custom mark\n")})
	var err error
	prior, err = grubbery.NewPlan(prior.Binding, prior.StageID, prior.RequestID, m, files, prior.Before)
	if err != nil {
		t.Fatal(err)
	}
	h := nativeHarnessFor(t, prior, files)
	p := nativePlan(t, h, prior, prior.RequestID)
	if _, err := h.execution().Begin(t.Context(), callFor(p, ""), p); err != nil {
		t.Fatal(err)
	}
	for _, step := range p.Steps {
		nativeAdvance(t, h, p, step)
	}
	root, _ := p.Binding.ReleaseRoot(p.RequestID)
	for _, step := range p.Steps {
		if step.Path == root+"/mar" && h.model.writes[step.ID] != 0 {
			t.Fatal("retried generated directory")
		}
	}
	first, _ := h.journal.Load(t.Context(), p.RequestID)
	own, err := grubbery.DeriveOwnership([]grubbery.Record{first})
	if err != nil || own.Epoch != 1 || own.SourceOrigin == nil || own.DeskOrigin == nil {
		t.Fatal("original ownership missing", err)
	}
	// A new process/policy is admitted for a new operation without renaming app
	// data or requiring the old source cache. Old native paths use current binding.
	prior = p
	prior.Binding.RuntimeID = strings.Repeat("9", 64)
	h.revision++
	p = nativePlan(t, h, prior, strings.Repeat("2", 64))
	if _, err = h.execution().Begin(t.Context(), callFor(p, ""), p); err != nil {
		t.Fatal(err)
	}
	for _, step := range p.Steps {
		nativeAdvance(t, h, p, step)
	}
	second, _ := h.journal.Load(t.Context(), p.RequestID)
	updated, err := grubbery.DeriveOwnership([]grubbery.Record{first, second})
	if err != nil || updated.Epoch != 2 || *updated.SourceOrigin != *own.SourceOrigin || *updated.DeskOrigin != *own.DeskOrigin {
		t.Fatal("update replaced original ownership", err)
	}
}

func TestNativeCoordinatorLostReplyReconcilesWithoutCacheOrRedispatch(t *testing.T) {
	prior, files := plan(t)
	h := nativeHarnessFor(t, prior, files)
	p := nativePlan(t, h, prior, prior.RequestID)
	_, err := h.execution().Begin(t.Context(), callFor(p, ""), p)
	if err != nil {
		t.Fatal(err)
	}
	step := p.Steps[0]
	h.model.lost = true
	r, err := h.execution().Advance(t.Context(), callFor(p, step.ID))
	if err != nil {
		t.Fatal(err)
	}
	state, _ := r.State(step.ID)
	if state != "uncertain" {
		t.Fatal(state)
	}
	original := copyRecord(r)
	h.sourceMissing = true
	calls, reads := h.sourceCalls, h.model.reads
	r, err = h.execution().Advance(t.Context(), callFor(p, step.ID))
	if err != nil || h.sourceCalls != calls || h.model.reads != reads {
		t.Fatal("repeat reentered native backend", err)
	}
	r, err = h.execution().Reconcile(t.Context(), callFor(p, step.ID))
	state, _ = r.State(step.ID)
	if err != nil || state != "observed" || h.sourceCalls != calls || h.model.writes[step.ID] != 1 || len(r.Events) != len(original.Events)+1 {
		t.Fatal("cache-free exact recovery", state, err)
	}
	for i := range original.Events {
		if r.Events[i].Digest != original.Events[i].Digest {
			t.Fatal("rewrote uncertainty")
		}
	}
}

func TestNativeCoordinatorRefusesUnownedAndChangedNativeEvidenceBeforeWrites(t *testing.T) {
	for _, kind := range []string{"unowned-source", "unowned-desk", "source-marker", "foreign-path", "foundational-bytes", "native-cleanup", "guard"} {
		t.Run(kind, func(t *testing.T) {
			prior, files := plan(t)
			h := nativeHarnessFor(t, prior, files)
			n, _ := prior.Binding.Namespace()
			if kind == "unowned-source" || kind == "unowned-desk" {
				if kind == "unowned-source" {
					marker, _ := prior.Binding.OwnerMarker()
					h.model.put(n.SourceRoot+"/omabit-owner.json", "mime", marker)
				} else {
					h.model.mkdir(n.DeskRoot, "desk")
				}
				if _, err := h.backend.Inspect(t.Context(), prior.Binding); err == nil {
					t.Fatal("adopted unowned native path")
				}
				return
			}
			p := nativePlan(t, h, prior, prior.RequestID)
			_, err := h.execution().Begin(t.Context(), callFor(p, ""), p)
			if err != nil {
				t.Fatal(err)
			}
			index := 1
			nativeAdvance(t, h, p, p.Steps[0])
			switch kind {
			case "source-marker":
				h.model.files[n.SourceRoot+"/omabit-owner.json"] = []byte(`{"foreign":true}`)
			case "foreign-path":
				h.model.put(n.SourceRoot+"/foreign.json", "mime", []byte("null"))
			case "foundational-bytes":
				for index < len(p.Steps) && p.Steps[index].Path != mustRelease(t, p) {
					nativeAdvance(t, h, p, p.Steps[index])
					index++
				}
				nativeAdvance(t, h, p, p.Steps[index])
				index++
				h.model.files[mustRelease(t, p)+"/mar/hoon.hoon"] = []byte("wrong framework bytes")
			case "native-cleanup":
				h.model.cleanup = "unconfirmed"
			case "guard":
				h.deny = true
			}
			before := len(h.model.writes)
			_, err = h.execution().Advance(t.Context(), callFor(p, p.Steps[index].ID))
			if err == nil || len(h.model.writes) != before {
				t.Fatal("changed evidence authorized a write", err)
			}
		})
	}
}
func mustRelease(t *testing.T, p grubbery.Plan) string {
	t.Helper()
	root, err := p.Binding.ReleaseRoot(p.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestNativeCoordinatorCompileAndVersionCannotHideWrongServedSource(t *testing.T) {
	prior, files := plan(t)
	h := nativeHarnessFor(t, prior, files)
	p := nativePlan(t, h, prior, prior.RequestID)
	_, err := h.execution().Begin(t.Context(), callFor(p, ""), p)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range p.Steps {
		nativeAdvance(t, h, p, step)
	}
	n, _ := p.Binding.Namespace()
	h.model.files[n.CodeRoot+"/nex/state.hoon"] = []byte("matching version with wrong code")
	if _, err = h.backend.Inspect(t.Context(), p.Binding); err == nil {
		t.Fatal("version plus clean compiler hid changed live code")
	}
}
