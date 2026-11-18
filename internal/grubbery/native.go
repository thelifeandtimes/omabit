package grubbery

import (
	"context"
	"errors"
	"path"
	"reflect"
	"sort"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

// NativeResponse is private transport data. Only explicitly selected bounded
// evidence, never compiler text, app data or credentials, belongs in a reply.
type NativeResponse struct {
	SessionCleanup string `json:"session_cleanup,omitempty"`
	Attempted      bool   `json:"attempted"`
	Status         int    `json:"status"`
	ContentType    string `json:"content_type"`
	Body           []byte `json:"-"`
	BodySHA256     string `json:"body_sha256"`
}

type NativeReader interface {
	GrubberyRead(context.Context, ReadRequest) (NativeResponse, error)
}

type NativeRuntime interface {
	NativeReader
	GrubberyStep(context.Context, Plan, string, []deskwork.SourceFile) (NativeResponse, error)
}

const NativeReadScopeActive = "scope-active"

// NativeReadScopeRuntime optionally amortizes owner authentication over one
// read-only capture. The callback keeps the original sequential/conditional
// checks: it cannot dispatch a write through its reader. Provisional responses
// are not evidence until this method returns confirmed cleanup. A runtime must
// fence retained readers after the callback and bind every request to the exact
// original target/process. This interface is trusted runner code, never RPC data.
type NativeReadScopeRuntime interface {
	WithGrubberyReads(context.Context, Binding, func(NativeReader) error) (string, error)
}

// These adapters are assembled inside the authenticated runner. History must
// return the complete validated namespace history in durable creation order.
// Guard checks the selected actor/role, runner, process/pier, framework receipt,
// configuration and resources. For writes it also checks fresh app policy.
// The runner holds its lifecycle/policy mutex across an entire Execution call.
type NativeHistory func(context.Context, Binding) ([]Record, error)
type NativeGuard func(context.Context, Binding, bool) (uint64, error)
type NativeSource func(context.Context, string) ([]deskwork.SourceFile, error)

type NativeBackend struct {
	runtime NativeRuntime
	history NativeHistory
	guard   NativeGuard
	source  NativeSource
}

func NewNativeBackend(runtime NativeRuntime, history NativeHistory, guard NativeGuard, source NativeSource) (*NativeBackend, error) {
	if runtime == nil || history == nil || guard == nil || source == nil {
		return nil, ErrAdmission
	}
	return &NativeBackend{runtime, history, guard, source}, nil
}

// Execution binds the coordinator and its exact private stage loader together.
// The runner supplies its durable store; no tool body can replace these adapters.
func (b *NativeBackend) Execution(store Store) Execution {
	return Execution{Store: store, Backend: b, LoadSource: b.source}
}

// Pinned +force-foundational-marks copies these exact framework sources. They
// are checked independently of the application's source and compile report.
var nativeFoundations = []deskwork.File{
	{Path: "mar/hoon.hoon", Bytes: 1098, SHA256: "fcec4e0a96c28b75dd8284b4b7e824c911442a706424b2ccf0b05cca7e857afe"},
	{Path: "mar/kelvin.hoon", Bytes: 689, SHA256: "706b3c9e305ef54653e6c4e4d7fc4561b59ebf9827715bd48351b1bdfa1c23ba"},
	{Path: "mar/mime.hoon", Bytes: 439, SHA256: "3ca9949786a5876bf8de05c9913cd0b6b1e78509c8e86912dcc0d766c1075863"},
	{Path: "mar/tang.hoon", Bytes: 824, SHA256: "a6dec16f1c704df54e6f3015f5cf0c5a792e126d67ae290f8540dfda68ff5749"},
}

type nativeFile struct {
	mark, sha, plan, step string
	bytes                 int64
}
type nativeNode struct {
	neck  string
	dirs  map[string]*nativeNode
	files map[string]nativeFile
	// Pinned foundational grubs are readable through the code namespace but
	// are not entries in its materialized tree (native layout qualification).
	foundations map[string]nativeFile
}

func plainNode() *nativeNode {
	return &nativeNode{dirs: map[string]*nativeNode{}, files: map[string]nativeFile{}}
}
func (n *nativeNode) directory(relative string) *nativeNode {
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." {
			continue
		}
		if n.dirs[part] == nil {
			n.dirs[part] = plainNode()
		}
		n = n.dirs[part]
	}
	return n
}
func checkNode(want *nativeNode, got NativeDirectory) error {
	if want == nil || want.neck == "" && got.Neck != nil || want.neck != "" && (got.Neck == nil || *got.Neck != want.neck) || len(want.files) != len(got.Files) || len(want.dirs) != len(got.Directories) {
		return ErrObservation
	}
	for name, f := range want.files {
		if got.Files[name] != f.mark {
			return ErrObservation
		}
	}
	for name, child := range want.dirs {
		actual, ok := got.Directories[name]
		if !ok || checkNode(child, actual) != nil {
			return ErrObservation
		}
	}
	return nil
}

type nativeProjection struct {
	source          *nativeNode
	desk, marker    bool
	config, version []byte
	owner           []byte
	before          Before
	mirror          *nativeNode
}

// projection admits only original observed writes plus, for an exact dispatched
// after-read, the selected original candidate effect. A refused/pending write
// never becomes an ownership claim because matching bytes appear later.
func projection(b Binding, records []Record, candidate string) (nativeProjection, error) {
	var out nativeProjection
	n, err := b.Namespace()
	if err != nil {
		return out, err
	}
	out.owner, _ = b.OwnerMarker()
	out.config, out.version = []byte("null"), []byte("null")
	out.before = Before{SourceState: "absent", DeskState: "absent"}
	if len(records) == 0 {
		return out, nil
	}
	own, err := DeriveOwnership(records)
	if err != nil || own.Namespace != n || own.OwnerSHA256 != hash(out.owner) {
		return out, ErrHistory
	}
	if own.SourceOrigin != nil {
		out.before.SourceState, out.before.SourceOwnerSHA256 = "owned", own.OwnerSHA256
	}
	if own.DeskOrigin != nil {
		out.before.DeskState, out.before.DeskOwnerSHA256 = "owned", own.OwnerSHA256
	}
	out.before.Epoch, out.before.ReleaseRoot = own.Epoch, own.ReleaseRoot
	for _, r := range records {
		states := map[string]string{}
		for _, e := range r.Events {
			states[e.StepID] = e.State
		}
		for _, s := range r.Plan.Steps {
			state := states[s.ID]
			include := state == "observed"
			if s.ID == candidate {
				if state != "dispatched" && state != "acknowledged" && state != "uncertain" {
					return out, ErrPhase
				}
				include = true
			}
			if !include {
				continue
			}
			if strings.HasPrefix(s.Path, n.SourceRoot+"/") || s.Path == n.SourceRoot {
				if s.Kind != "make-file" && s.Kind != "make-directory" {
					continue
				}
				if out.source == nil {
					out.source = plainNode()
				}
				rel := strings.TrimPrefix(strings.TrimPrefix(s.Path, n.SourceRoot), "/")
				if s.Kind == "make-directory" {
					dir := out.source.directory(rel)
					if s.ControlJSON == `{"neck":"/code"}` {
						dir.neck = "code"
						dir.foundations = map[string]nativeFile{}
						for _, f := range nativeFoundations {
							dir.foundations[f.Path] = nativeFile{"hoon", f.SHA256, r.Plan.Digest, s.ID, f.Bytes}
						}
					}
				} else {
					mark, size := "mime", int64(len(s.ControlJSON))
					if s.SourcePath != "" {
						if codePath(s.SourcePath) {
							mark = "hoon"
						}
						size = r.Plan.Source.Manifest.Index()[s.SourcePath].Bytes
					}
					out.source.directory(path.Dir(rel)).files[path.Base(rel)] = nativeFile{mark, s.PayloadSHA256, r.Plan.Digest, s.ID, size}
				}
			}
			if s.Path == n.DeskRoot && s.Kind == "make-directory" {
				out.desk = true
			}
			if s.Path == n.DataRoot+"/omabit-owner.json" && s.Kind == "make-file" {
				out.marker = true
			}
			if s.Kind == "poke-source" {
				out.config = []byte(s.ControlJSON)
				for _, version := range r.Plan.Steps {
					release, _ := b.ReleaseRoot(r.Plan.RequestID)
					if version.Kind == "make-file" && version.Path == release+"/version.json" {
						out.version = []byte(version.ControlJSON)
					}
				}
				release, _ := b.ReleaseRoot(r.Plan.RequestID)
				out.mirror = out.source.directory(strings.TrimPrefix(release, n.SourceRoot+"/"))
			}
		}
	}
	return out, nil
}

func (b *NativeBackend) records(ctx context.Context, p Plan, step Step, state string) ([]Record, error) {
	if p.ValidateMetadata() != nil || CheckNativeSource(p.Source) != nil {
		return nil, ErrHistory
	}
	records, err := b.history(ctx, p.Binding)
	if err != nil {
		return nil, ErrHistory
	}
	found := false
	for _, r := range records {
		if r.Plan.RequestID != p.RequestID {
			continue
		}
		if found || !reflect.DeepEqual(r.Plan, p) {
			return nil, ErrHistory
		}
		found = true
		current, done, err := r.Current()
		actual, stateErr := r.State(step.ID)
		if err != nil || stateErr != nil || done || current != step || state != "after" && actual != state || state == "after" && actual != "dispatched" && actual != "acknowledged" && actual != "uncertain" {
			return nil, ErrPhase
		}
	}
	if !found {
		if state != "pending" || len(p.Steps) == 0 || p.Steps[0] != step {
			return nil, ErrMissing
		}
		r, err := NewRecord(p, fixedProjectionTime)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	if _, err := DeriveOwnership(records); err != nil {
		return nil, ErrHistory
	}
	return records, nil
}

// No timestamp in this virtual, unpersisted record enters snapshot evidence.
var fixedProjectionTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

type nativeEvidence struct {
	Purpose, Plan, Step, Path, Query string
	Status                           int
	SHA256                           string
}
type nativeCapture struct {
	backend  *NativeBackend
	binding  Binding
	reader   NativeReader
	cleanup  string
	evidence []nativeEvidence
	count    int
}

func (c *nativeCapture) read(ctx context.Context, r ReadRequest) (NativeResponse, error) {
	c.count++
	if c.count > 512 || ctx.Err() != nil || c.reader == nil {
		return NativeResponse{}, ErrObservation
	}
	response, err := c.reader.GrubberyRead(ctx, r)
	if err != nil || !response.Attempted || response.SessionCleanup != c.cleanup || response.Status != 200 && response.Status != 404 || len(response.Body) > 1<<20 || hash(response.Body) != response.BodySHA256 {
		clear(response.Body)
		return NativeResponse{}, ErrObservation
	}
	d, err := r.Descriptor()
	if err != nil {
		clear(response.Body)
		return NativeResponse{}, err
	}
	q, _ := r.Request()
	c.evidence = append(c.evidence, nativeEvidence{d.Purpose, d.PlanDigest, d.StepID, q.Path, q.Query, response.Status, response.BodySHA256})
	return response, nil
}
func (c *nativeCapture) bound(ctx context.Context, purpose ReadPurpose) (NativeResponse, error) {
	r, err := c.binding.Read(purpose)
	if err != nil {
		return NativeResponse{}, err
	}
	return c.read(ctx, r)
}
func (c *nativeCapture) json(ctx context.Context, purpose ReadPurpose, expected []byte) error {
	r, err := c.bound(ctx, purpose)
	if err != nil {
		return err
	}
	defer clear(r.Body)
	if expected == nil {
		if r.Status != 404 {
			return ErrObservation
		}
		return nil
	}
	if r.Status != 200 || !EqualNativeJSON(r.Body, expected) {
		return ErrObservation
	}
	return nil
}
func (c *nativeCapture) files(ctx context.Context, node *nativeNode, root string) error {
	names := []string{}
	all := map[string]nativeFile{}
	for name := range node.files {
		names = append(names, name)
		all[name] = node.files[name]
	}
	for name, f := range node.foundations {
		names = append(names, name)
		all[name] = f
	}
	sort.Strings(names)
	for _, name := range names {
		f := all[name]
		purpose := "immutable-source"
		if _, virtual := node.foundations[name]; virtual {
			purpose = "pinned-virtual-foundation"
		}
		r, err := newRead(c.binding, purpose, f.plan, f.step, "peek"+root+"/"+name, "")
		if err != nil {
			return err
		}
		response, err := c.read(ctx, r)
		clear(response.Body)
		if err != nil || response.Status != 200 || response.BodySHA256 != f.sha || int64(len(response.Body)) != f.bytes {
			return ErrObservation
		}
	}
	names = names[:0]
	for name := range node.dirs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := c.files(ctx, node.dirs[name], root+"/"+name); err != nil {
			return err
		}
	}
	return nil
}

func (b *NativeBackend) capture(ctx context.Context, binding Binding, p nativeProjection) (*nativeCapture, error) {
	return b.captureWithChecks(ctx, binding, p, nil)
}

// Every conditional verification read must run inside the same callback as the
// capture. After that callback no reader can remain usable, and no proof can be
// returned before confirmed cleanup. Checks retain original read order/evidence.
func (b *NativeBackend) captureWithChecks(ctx context.Context, binding Binding, p nativeProjection, checks func(*nativeCapture) error) (*nativeCapture, error) {
	captureReader := func(reader NativeReader, cleanup string) (*nativeCapture, error) {
		capture, err := b.captureReader(ctx, binding, p, reader, cleanup)
		if err == nil && checks != nil {
			err = checks(capture)
		}
		if err != nil {
			return nil, err
		}
		return capture, nil
	}
	if runtime, ok := b.runtime.(NativeReadScopeRuntime); ok {
		var capture *nativeCapture
		var callbackErr error
		calls := 0
		cleanup, err := runtime.WithGrubberyReads(ctx, binding, func(reader NativeReader) error {
			calls++
			if calls != 1 || reader == nil {
				callbackErr = ErrObservation
				return callbackErr
			}
			capture, callbackErr = captureReader(reader, NativeReadScopeActive)
			return callbackErr
		})
		if err != nil || callbackErr != nil || cleanup != "confirmed" || capture == nil || calls != 1 {
			return nil, ErrObservation
		}
		capture.reader = nil
		return capture, nil
	}
	capture, err := captureReader(b.runtime, "confirmed")
	if capture != nil {
		capture.reader = nil
	}
	return capture, err
}

func (b *NativeBackend) captureReader(ctx context.Context, binding Binding, p nativeProjection, reader NativeReader, cleanup string) (*nativeCapture, error) {
	c := &nativeCapture{backend: b, binding: binding, reader: reader, cleanup: cleanup}
	source, err := c.bound(ctx, ReadSourceRoot)
	if err != nil {
		return nil, err
	}
	defer clear(source.Body)
	if p.source == nil {
		if source.Status != 404 {
			return nil, ErrObservation
		}
	} else {
		tree, err := ParseDirectory(source.Body)
		if source.Status != 200 || err != nil || checkNode(p.source, tree) != nil {
			return nil, ErrObservation
		}
		n, _ := binding.Namespace()
		if err := c.files(ctx, p.source, n.SourceRoot); err != nil {
			return nil, err
		}
	}
	desk, err := c.bound(ctx, ReadDeskRoot)
	if err != nil {
		return nil, err
	}
	defer clear(desk.Body)
	if !p.desk {
		if desk.Status != 404 {
			return nil, ErrObservation
		}
		return c, nil
	}
	tree, err := ParseDirectory(desk.Body)
	if desk.Status != 200 || err != nil || tree.Neck == nil || *tree.Neck != "desk" {
		return nil, ErrObservation
	}
	world, ok := tree.Directories["desk"]
	code, codeOK := world.Directories["code"]
	data, dataOK := world.Directories["data"]
	if !ok || world.Neck != nil || !codeOK || !dataOK || code.Neck == nil || *code.Neck != "code" || data.Neck != nil {
		return nil, ErrObservation
	}
	marker := []byte(nil)
	if p.marker {
		marker = p.owner
	}
	if err = c.json(ctx, ReadDeskOwner, marker); err != nil {
		return nil, err
	}
	if err = c.json(ctx, ReadSourceConfig, p.config); err != nil {
		return nil, err
	}
	if err = c.json(ctx, ReadFollowedVersion, p.version); err != nil {
		return nil, err
	}
	if p.mirror != nil {
		if checkNode(p.mirror, code) != nil {
			return nil, ErrObservation
		}
		n, _ := binding.Namespace()
		if err := c.files(ctx, p.mirror, n.CodeRoot); err != nil {
			return nil, err
		}
	}
	return c, nil
}
func (c *nativeCapture) digest() string { return hash([]byte(textJSON(c.evidence))) }

// Inspect establishes an initial baseline from native state and immutable saved
// ownership. A namespace with an unfinished operation must use its exact handle.
func (b *NativeBackend) Inspect(ctx context.Context, binding Binding) (Before, error) {
	if binding.Validate() != nil {
		return Before{}, ErrConflict
	}
	if _, err := b.guard(ctx, binding, true); err != nil {
		return Before{}, ErrAdmission
	}
	records, err := b.history(ctx, binding)
	if err != nil {
		return Before{}, ErrHistory
	}
	if len(records) != 0 {
		own, err := DeriveOwnership(records)
		if err != nil {
			return Before{}, err
		}
		if own.PendingRequestID != "" {
			return Before{}, ErrNamespaceBusy
		}
	}
	p, err := projection(binding, records, "")
	if err != nil {
		return Before{}, err
	}
	c, err := b.capture(ctx, binding, p)
	if err != nil {
		return Before{}, err
	}
	if _, err = b.guard(ctx, binding, true); err != nil {
		return Before{}, ErrAdmission
	}
	p.before.SnapshotDigest = c.digest()
	return p.before, nil
}

func (b *NativeBackend) admit(ctx context.Context, p Plan, s Step, state string) (Review, error) {
	revision, err := b.guard(ctx, p.Binding, true)
	if err != nil {
		return Review{}, ErrAdmission
	}
	records, err := b.records(ctx, p, s, state)
	if err != nil {
		return Review{}, err
	}
	projected, err := projection(p.Binding, records, "")
	if err != nil {
		return Review{}, err
	}
	c, err := b.capture(ctx, p.Binding, projected)
	if err != nil {
		return Review{}, err
	}
	// Only before the first dispatch is this the immutable reviewed baseline.
	for _, r := range records {
		if r.Plan.RequestID == p.RequestID && r.Revision == 0 && c.digest() != p.Before.SnapshotDigest {
			return Review{}, ErrObservation
		}
	}
	if after, err := b.guard(ctx, p.Binding, true); err != nil || after != revision {
		return Review{}, ErrAdmission
	}
	return Review{1, p.Digest, s.ID, bindingDigest(p), revision, c.digest()}, nil
}
func (b *NativeBackend) Admit(ctx context.Context, p Plan, s Step) (Review, error) {
	return b.admit(ctx, p, s, "pending")
}

func (b *NativeBackend) Dispatch(ctx context.Context, p Plan, s Step, q Request) (string, error) {
	files, err := b.source(ctx, p.StageID)
	if err != nil {
		return "", ErrSource
	}
	want, err := p.Request(s.ID, files)
	if err != nil {
		return "", err
	}
	defer clear(want.Body)
	if !reflect.DeepEqual(q, want) {
		return "", ErrConflict
	}
	_, err = b.admit(ctx, p, s, "dispatched")
	if err != nil {
		return "", err
	}
	response, err := b.runtime.GrubberyStep(ctx, p, s.ID, files)
	defer clear(response.Body)
	if !response.Attempted || response.SessionCleanup != "confirmed" || hash(response.Body) != response.BodySHA256 {
		return "", ErrObservation
	}
	if err != nil {
		if errors.Is(err, ErrNativeRejected) {
			return response.BodySHA256, ErrNativeRejected
		}
		return "", ErrObservation
	}
	return hash([]byte(textJSON(response))), nil
}

func (b *NativeBackend) ObserveAfter(ctx context.Context, p Plan, s Step) (*Proof, error) {
	if _, err := b.guard(ctx, p.Binding, false); err != nil {
		return nil, ErrAdmission
	}
	records, err := b.records(ctx, p, s, "after")
	if err != nil {
		return nil, err
	}
	projected, err := projection(p.Binding, records, s.ID)
	if err != nil {
		return nil, err
	}
	var checks func(*nativeCapture) error
	if s.Kind == "observe-code" {
		checks = func(c *nativeCapture) error {
			reads, err := p.AfterReads(s.ID)
			if err != nil || len(reads) != 1 {
				return ErrObservation
			}
			response, err := c.read(ctx, reads[0])
			if err != nil {
				return err
			}
			defer clear(response.Body)
			report, err := ParseBuildReport(response.Body, s.Path)
			if response.Status != 200 || err != nil || report.CheckStaged(p.Source, false) != nil {
				return ErrObservation
			}
			return nil
		}
	}
	c, err := b.captureWithChecks(ctx, p.Binding, projected, checks)
	if err != nil {
		return nil, err
	}
	if _, err := b.guard(ctx, p.Binding, false); err != nil {
		return nil, ErrAdmission
	}
	return &Proof{1, p.Digest, s.ID, bindingDigest(p), c.digest()}, nil
}

func (b *NativeBackend) ObserveAbandon(ctx context.Context, original Record) (*AbandonProof, error) {
	if original.Validate() != nil {
		return nil, ErrHistory
	}
	allowed, effects := original.abandonable()
	if !allowed || !effects {
		return nil, ErrAdmission
	}
	step, _, _ := original.Current()
	// The source-only prefix must match the exact currently retained record.
	records, err := b.history(ctx, original.Plan.Binding)
	if err != nil {
		return nil, ErrHistory
	}
	found := false
	for _, r := range records {
		if r.Plan.RequestID == original.Plan.RequestID {
			found = reflect.DeepEqual(r, original)
		}
	}
	if !found || step.ID == "" {
		return nil, ErrHistory
	}
	if _, err = b.guard(ctx, original.Plan.Binding, false); err != nil {
		return nil, ErrAdmission
	}
	projected, err := projection(original.Plan.Binding, records, "")
	if err != nil {
		return nil, err
	}
	c, err := b.capture(ctx, original.Plan.Binding, projected)
	if err != nil {
		return nil, err
	}
	if _, err = b.guard(ctx, original.Plan.Binding, false); err != nil {
		return nil, ErrAdmission
	}
	return &AbandonProof{1, original.Plan.Digest, bindingDigest(original.Plan), original.Revision, original.tailDigest(), original.Plan.Before.Epoch, original.Plan.Before.ReleaseRoot, c.digest()}, nil
}
