package grubbery

import "errors"

// Native reads are constructed from a checked binding/plan. They carry no URL,
// credentials, caller Hoon or arbitrary namespace selector from a tool request.
type ReadPurpose string

const (
	ReadSourceRoot      ReadPurpose = "source-root"
	ReadSourceOwner     ReadPurpose = "source-owner"
	ReadDeskRoot        ReadPurpose = "desk-root"
	ReadDeskOwner       ReadPurpose = "desk-owner"
	ReadSourceConfig    ReadPurpose = "source-config"
	ReadFollowedVersion ReadPurpose = "followed-version"
)

type ReadRequest struct {
	binding            Binding
	purpose            string
	planDigest, stepID string
	request            Request
	seal               string
}
type ReadDescriptor struct {
	Purpose       string `json:"purpose"`
	BindingDigest string `json:"binding_digest"`
	PlanDigest    string `json:"plan_digest,omitempty"`
	StepID        string `json:"step_id,omitempty"`
}

func (r ReadRequest) digest() string {
	return hash([]byte(textJSON(struct {
		Binding                     Binding
		Purpose, PlanDigest, StepID string
		Request                     Request
	}{r.binding, r.purpose, r.planDigest, r.stepID, r.request})))
}
func newRead(b Binding, purpose, plan, step, path, query string) (ReadRequest, error) {
	if b.Validate() != nil {
		return ReadRequest{}, ErrConflict
	}
	r := ReadRequest{binding: b, purpose: purpose, planDigest: plan, stepID: step, request: Request{Method: "GET", Path: "/grubbery/api/" + path, Query: query}}
	r.seal = r.digest()
	return r, nil
}
func (r ReadRequest) Request() (Request, error) {
	if r.binding.Validate() != nil || r.purpose == "" || r.seal == "" || r.seal != r.digest() || r.request.Method != "GET" || len(r.request.Body) != 0 {
		return Request{}, errors.New("invalid bound native read")
	}
	return r.request, nil
}
func (r ReadRequest) Descriptor() (ReadDescriptor, error) {
	if _, err := r.Request(); err != nil {
		return ReadDescriptor{}, err
	}
	return ReadDescriptor{Purpose: r.purpose, BindingDigest: hash([]byte(textJSON(r.binding))), PlanDigest: r.planDigest, StepID: r.stepID}, nil
}
func (b Binding) Read(purpose ReadPurpose) (ReadRequest, error) {
	n, err := b.Namespace()
	if err != nil {
		return ReadRequest{}, err
	}
	path, query := "", ""
	switch purpose {
	case ReadSourceRoot:
		path = "peek" + n.SourceRoot + "/"
		query = "as=tree"
	case ReadSourceOwner:
		path = "peek" + n.SourceRoot + "/omabit-owner.json"
	case ReadDeskRoot:
		path = "peek" + n.DeskRoot + "/"
		query = "as=tree"
	case ReadDeskOwner:
		path = "peek" + n.DataRoot + "/omabit-owner.json"
	case ReadSourceConfig:
		path = "peek" + n.DeskRoot + "/source.json"
	case ReadFollowedVersion:
		path = "peek" + n.DeskRoot + "/version.json"
	default:
		return ReadRequest{}, errors.New("unsupported native app read")
	}
	return newRead(b, string(purpose), "", "", path, query)
}

// AfterReads needs only immutable saved metadata, including legacy plan1. It
// neither loads source bytes nor constructs a write. A successful read is not an
// ownership/after proof until the trusted adapter checks exact native content.
func (p Plan) AfterReads(stepID string) ([]ReadRequest, error) {
	if p.ValidateMetadata() != nil {
		return nil, ErrHistory
	}
	var selected *Step
	for _, s := range p.Steps {
		if s.ID == stepID {
			copy := s
			selected = &copy
			break
		}
	}
	if selected == nil {
		return nil, ErrPhase
	}
	s := *selected
	paths := []struct{ name, path, query string }{}
	switch s.Kind {
	case "make-directory":
		paths = append(paths, struct{ name, path, query string }{"created-directory", "peek" + s.Path + "/", "as=tree"})
	case "make-file":
		paths = append(paths, struct{ name, path, query string }{"created-file", "peek" + s.Path, ""})
	case "observe-code":
		paths = append(paths, struct{ name, path, query string }{"compiled-code", "build" + s.Path, "all=1"})
	case "poke-source":
		n, _ := p.Binding.Namespace()
		paths = append(paths, struct{ name, path, query string }{"source-config", "peek" + n.DeskRoot + "/source.json", ""}, struct{ name, path, query string }{"followed-version", "peek" + n.DeskRoot + "/version.json", ""})
	case "observe-followed-source":
		paths = append(paths, struct{ name, path, query string }{"followed-version", "peek" + s.Path, ""})
	default:
		return nil, ErrPhase
	}
	out := []ReadRequest{}
	for _, v := range paths {
		r, err := newRead(p.Binding, v.name, p.Digest, stepID, v.path, v.query)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// StagedReads addresses only the original plan's bounded source inventory. It
// excludes historical releases, arbitrary paths, state, credentials and mounts.
func (p Plan) StagedReads() ([]ReadRequest, error) {
	if p.ValidateMetadata() != nil {
		return nil, ErrHistory
	}
	root, err := p.Binding.ReleaseRoot(p.RequestID)
	if err != nil {
		return nil, err
	}
	out := []ReadRequest{}
	for _, f := range p.Source.Manifest.Files {
		r, err := newRead(p.Binding, "staged-source:"+f.Path, p.Digest, "", "peek"+root+"/"+f.Path, "")
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (r ReadRequest) Binding() (Binding, error) {
	if _, err := r.Request(); err != nil {
		return Binding{}, err
	}
	return r.binding, nil
}
