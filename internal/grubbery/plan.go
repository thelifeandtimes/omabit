package grubbery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"sort"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

// Before is populated by trusted native observation plus saved ownership, never
// by accepting caller claims. Hashes alone do not establish ownership. The
// admission layer must independently reobserve this exact native baseline.
type Before struct {
	SnapshotDigest    string `json:"snapshot_digest"`
	SourceState       string `json:"source_state"` // absent or owned
	SourceOwnerSHA256 string `json:"source_owner_sha256,omitempty"`
	DeskState         string `json:"desk_state"` // absent or owned
	DeskOwnerSHA256   string `json:"desk_owner_sha256,omitempty"`
	Epoch             uint64 `json:"epoch"`
	ReleaseRoot       string `json:"release_root,omitempty"`
}

type Step struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Path          string `json:"path"`
	SourcePath    string `json:"source_path,omitempty"`
	PayloadSHA256 string `json:"payload_sha256,omitempty"`
	ObserveSHA256 string `json:"observe_sha256,omitempty"`
	ControlJSON   string `json:"control_json,omitempty"`
}

type Plan struct {
	Version   int     `json:"version"`
	Adapter   string  `json:"adapter"`
	Binding   Binding `json:"binding"`
	StageID   string  `json:"stage_id"`
	RequestID string  `json:"request_id"`
	Source    Source  `json:"source"`
	Before    Before  `json:"before"`
	Epoch     uint64  `json:"epoch"`
	Steps     []Step  `json:"steps"`
	Digest    string  `json:"digest"`
}

func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func textJSON(v any) string   { b, _ := json.Marshal(v); return string(b) }

func ownerJSON(b Binding) string {
	// Intentionally match the stable namespace identity rather than actor or
	// container. Audit ownership and current authority remain runner records.
	return textJSON(struct {
		Version    int              `json:"version"`
		Adapter    string           `json:"adapter"`
		RunnerID   string           `json:"runner_id"`
		Mode       string           `json:"mode"`
		Target     apppolicy.Target `json:"target"`
		Continuity string           `json:"continuity"`
		AppID      string           `json:"app_id"`
	}{1, HTTPContract, b.RunnerID, b.Mode, b.Target, b.Continuity, b.AppID})
}

func (b Binding) OwnerMarker() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return []byte(ownerJSON(b)), nil
}

func NewPlan(binding Binding, stageID, requestID string, manifest deskwork.Manifest, files []deskwork.SourceFile, before Before) (Plan, error) {
	source, err := ReviewSource(manifest, files)
	if err != nil {
		return Plan{}, err
	}
	return reviewedPlan(binding, stageID, requestID, source, before)
}

func reviewedPlan(binding Binding, stageID, requestID string, source Source, before Before) (Plan, error) {
	var out Plan
	n, err := binding.Namespace()
	if err != nil || !digest.MatchString(stageID) || !digest.MatchString(requestID) || !digest.MatchString(before.SnapshotDigest) || before.Epoch > 1<<53-2 {
		return out, errors.New("invalid app deployment plan binding")
	}
	if source.ValidateMetadata() != nil {
		return out, errors.New("invalid attested app source metadata")
	}
	owner := ownerJSON(binding)
	ownerHash := hash([]byte(owner))
	for _, state := range []struct{ State, SHA string }{{before.SourceState, before.SourceOwnerSHA256}, {before.DeskState, before.DeskOwnerSHA256}} {
		if state.State != "absent" && state.State != "owned" || state.State == "absent" && state.SHA != "" || state.State == "owned" && state.SHA != ownerHash {
			return out, errors.New("app namespace is unowned or its marker changed")
		}
	}
	if before.SourceState == "absent" && (before.DeskState != "absent" || before.Epoch != 0 || before.ReleaseRoot != "") || before.DeskState == "absent" && (before.Epoch != 0 || before.ReleaseRoot != "") || before.Epoch == 0 && before.ReleaseRoot != "" {
		return out, errors.New("inconsistent saved app deployment baseline")
	}
	if before.Epoch != 0 {
		prefix := n.SourceRoot + "/"
		suffix := "/code"
		if !strings.HasPrefix(before.ReleaseRoot, prefix) || !strings.HasSuffix(before.ReleaseRoot, suffix) || !digest.MatchString(strings.TrimSuffix(strings.TrimPrefix(before.ReleaseRoot, prefix), suffix)) {
			return out, errors.New("current app source is outside its immutable release space")
		}
	}
	release, err := binding.ReleaseRoot(requestID)
	if err != nil {
		return out, err
	}
	if release == before.ReleaseRoot {
		return out, errors.New("original app request already names current source; read its saved operation")
	}
	out = Plan{Version: 1, Adapter: HTTPContract, Binding: binding, StageID: stageID, RequestID: requestID, Source: source, Before: before, Epoch: before.Epoch + 1, Steps: []Step{}}
	add := func(kind, p, source, body, expected string) {
		step := Step{Kind: kind, Path: p, SourcePath: source, ControlJSON: body}
		if body != "" {
			step.PayloadSHA256 = hash([]byte(body))
		} else if kind == "make-file" {
			step.PayloadSHA256 = expected
		} else {
			step.ObserveSHA256 = expected
		}
		out.Steps = append(out.Steps, step)
	}
	if before.SourceState == "absent" {
		add("make-directory", n.SourceRoot, "", "{}", "")
		add("make-file", n.SourceRoot+"/omabit-owner.json", "", owner, "")
	}
	add("make-directory", path.Dir(release), "", "{}", "")
	add("make-directory", release, "", `{"neck":"/code"}`, "")
	directories := map[string]bool{}
	for _, file := range source.Manifest.Files {
		for dir := path.Dir(file.Path); dir != "."; dir = path.Dir(dir) {
			directories[dir] = true
		}
	}
	if len(directories) > 128 {
		return Plan{}, errors.New("app release has too many source directories")
	}
	dirs := []string{}
	for dir := range directories {
		dirs = append(dirs, dir)
	}
	sort.Slice(dirs, func(i, j int) bool {
		a, b := strings.Count(dirs[i], "/"), strings.Count(dirs[j], "/")
		if a != b {
			return a < b
		}
		return dirs[i] < dirs[j]
	})
	for _, dir := range dirs {
		add("make-directory", release+"/"+dir, "", "{}", "")
	}
	for _, file := range source.Manifest.Files {
		add("make-file", release+"/"+file.Path, file.Path, "", file.SHA256)
	}
	add("observe-code", release, "", "", "")
	version := textJSON(struct {
		Epoch        uint64 `json:"epoch"`
		SourceDigest string `json:"source_digest"`
	}{out.Epoch, source.Manifest.Digest})
	add("make-file", release+"/version.json", "", version, "")
	if before.DeskState == "absent" {
		add("make-directory", n.DeskRoot, "", `{"neck":"/desk"}`, "")
		add("make-file", n.DataRoot+"/omabit-owner.json", "", owner, "")
	}
	// Poke the framework's source fiber. Overwriting its JSON as a mime would
	// bypass that contract. Approval/reload/sand/alias/cull are not plan effects.
	add("poke-source", n.DeskRoot+"/source.json", "", textJSON(struct {
		Code string `json:"code"`
	}{release}), "")
	add("observe-followed-source", n.DeskRoot+"/version.json", "", "", hash([]byte(version)))
	contextHash := hash([]byte(textJSON(struct {
		Binding                          Binding
		StageID, RequestID, SourceDigest string
		Before                           Before
		Epoch                            uint64
	}{binding, stageID, requestID, source.Manifest.Digest, before, out.Epoch})))
	for i := range out.Steps {
		out.Steps[i].ID = hash([]byte(textJSON(struct {
			Context string
			Index   int
			Step    Step
		}{contextHash, i, out.Steps[i]})))
	}
	out.Digest = hash([]byte(textJSON(out)))
	return out, nil
}

// ValidateMetadata verifies an already attested plan without requiring a source
// cache or contacting a runtime. Pending writes additionally require Validate.
func (p Plan) ValidateMetadata() error {
	want, err := reviewedPlan(p.Binding, p.StageID, p.RequestID, p.Source, p.Before)
	if err != nil || textJSON(want) != textJSON(p) {
		return errors.New("saved app plan metadata changed")
	}
	return nil
}

func (p Plan) Validate(files []deskwork.SourceFile) error {
	want, err := NewPlan(p.Binding, p.StageID, p.RequestID, p.Source.Manifest, files, p.Before)
	if err != nil || textJSON(want) != textJSON(p) {
		return errors.New("app deployment plan or staged source changed")
	}
	return nil
}

// Request is an internal native adapter input, not a public generic HTTP tool.
// A durable coordinator must admit one exact original Step.ID before constructing
// a write, persist dispatch before I/O, and never replay an uncertain write.
type Request struct {
	Method, Path, Query, ContentType string
	Body                             []byte
}

func (p Plan) Request(stepID string, files []deskwork.SourceFile) (Request, error) {
	var out Request
	if p.Validate(files) != nil {
		return out, errors.New("changed app deployment plan")
	}
	var selected *Step
	for i := range p.Steps {
		if p.Steps[i].ID == stepID {
			selected = &p.Steps[i]
			break
		}
	}
	if selected == nil {
		return out, errors.New("select one original app deployment step")
	}
	step := *selected
	query := url.Values{}
	if step.ControlJSON != "" {
		out.Body = []byte(step.ControlJSON)
		out.ContentType = "application/json"
	}
	switch step.Kind {
	case "make-directory":
		out.Method = "PUT"
		out.Path = "/grubbery/api/make" + step.Path + "/"
	case "make-file":
		out.Method = "PUT"
		out.Path = "/grubbery/api/make" + step.Path
		if step.SourcePath != "" {
			for _, file := range files {
				if file.Path == step.SourcePath {
					out.Body = append([]byte{}, file.Bytes...)
					break
				}
			}
			out.ContentType = sourceContentType(step.SourcePath)
			if strings.HasPrefix(step.SourcePath, "nex/") || strings.HasPrefix(step.SourcePath, "lib/") || strings.HasPrefix(step.SourcePath, "mar/") {
				query.Set("blot", "/hoon")
			}
		}
	case "observe-code":
		out.Method = "GET"
		out.Path = "/grubbery/api/build" + step.Path
		query.Set("all", "1")
	case "poke-source":
		out.Method = "POST"
		out.Path = "/grubbery/api/poke" + step.Path
		query.Set("blot", "/json")
	case "observe-followed-source":
		out.Method = "GET"
		out.Path = "/grubbery/api/peek" + step.Path
	default:
		return Request{}, errors.New("unsupported app deployment step")
	}
	out.Query = query.Encode()
	return out, nil
}

func sourceContentType(name string) string {
	switch path.Ext(name) {
	case ".hoon":
		return "text/x-hoon"
	case ".json":
		return "application/json"
	case ".html":
		return "text/html"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".txt", ".md":
		return "text/plain"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
