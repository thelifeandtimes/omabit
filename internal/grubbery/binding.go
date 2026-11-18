package grubbery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

type Binding struct {
	RunnerID           string               `json:"runner_id"`
	Mode               string               `json:"mode"`
	Actor              string               `json:"actor"`
	Target             apppolicy.Target     `json:"target"`
	RuntimeID          string               `json:"runtime_id"`
	ConfigRevision     uint64               `json:"config_revision"`
	PolicyRevision     uint64               `json:"policy_revision"`
	Continuity         string               `json:"continuity"`
	FrameworkReceiptID string               `json:"framework_receipt_id"`
	FrameworkBaseline  deskwork.Observation `json:"framework_baseline"`
	AppID              string               `json:"app_id"`
}

func (b Binding) Validate() error {
	if !identifier(b.RunnerID) || !identifier(b.Mode) || b.Actor == "" || len(b.Actor) > 256 || !utf8.ValidString(b.Actor) || b.Target.Validate() != nil || b.Target.InstanceID == "" || b.Target.Desk != FrameworkDesk || b.Target.ProjectID != "" && b.Target.Generation == 0 || !digest.MatchString(b.RuntimeID) || !digest.MatchString(b.Continuity) || !digest.MatchString(b.FrameworkReceiptID) || b.ConfigRevision == 0 || b.ConfigRevision > apppolicy.MaxRevision || b.PolicyRevision > apppolicy.MaxRevision || b.FrameworkBaseline.Validate() != nil || !b.FrameworkBaseline.Exists || !term.MatchString(b.AppID) {
		return errors.New("invalid exact Grubbery adapter binding")
	}
	for _, r := range b.Actor {
		if r < 32 || r == 127 {
			return errors.New("invalid adapter actor")
		}
	}
	return nil
}
func identifier(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

type Namespace struct {
	SourceRoot string `json:"source_root"`
	DeskRoot   string `json:"desk_root"`
	CodeRoot   string `json:"code_root"`
	DataRoot   string `json:"data_root"`
}

// Namespace derives a private candidate address. It is not an ownership claim:
// the runtime must refuse an existing unowned root and recheck saved ownership.
// Actor, process, configuration and policy changes do not rename saved app state.
// Runner, pier, project creator/generation, member, framework desk and app do.
func (b Binding) Namespace() (Namespace, error) {
	if err := b.Validate(); err != nil {
		return Namespace{}, err
	}
	stable := struct {
		RunnerID   string           `json:"runner_id"`
		Mode       string           `json:"mode"`
		Target     apppolicy.Target `json:"target"`
		Continuity string           `json:"continuity"`
		AppID      string           `json:"app_id"`
	}{b.RunnerID, b.Mode, b.Target, b.Continuity, b.AppID}
	data, _ := json.Marshal(stable)
	sum := sha256.Sum256(data)
	key := hex.EncodeToString(sum[:16])
	root := "/apps/shell/desks/omabit-" + key + ".desk"
	return Namespace{SourceRoot: "/omabit-source-" + key, DeskRoot: root, CodeRoot: root + "/desk/code", DataRoot: root + "/desk/data"}, nil
}

// ReleaseRoot names immutable staging space for one original request, separate
// from the live desk mirror and its saved state. Existing content still needs
// explicit observation; deriving this name never authorizes reuse or overwrite.
func (b Binding) ReleaseRoot(requestID string) (string, error) {
	n, err := b.Namespace()
	if err != nil || !digest.MatchString(requestID) {
		return "", errors.New("invalid original app release ID")
	}
	return n.SourceRoot + "/" + requestID + "/code", nil
}
