package deskwork

import (
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

// StageIntent freezes a transfer into the runner's private cache, never a pier.
// The ID is retained before upload so a lost reply can be reconciled read-only.
type StageIntent struct {
	Version          int              `json:"version"`
	ID               string           `json:"id"`
	Target           apppolicy.Target `json:"target"`
	SettingsRevision uint64           `json:"settings_revision"`
	SourceDigest     string           `json:"source_digest"`
	Files            int              `json:"files"`
	Bytes            int64            `json:"bytes"`
}

func (i StageIntent) Validate() error {
	if i.Version != 1 || !digestPattern.MatchString(i.ID) || i.Target.Validate() != nil || i.Target.InstanceID == "" || i.Target.Desk == "" || i.Target.ProjectID != "" && i.Target.Generation == 0 || i.SettingsRevision > apppolicy.MaxRevision || !digestPattern.MatchString(i.SourceDigest) || i.Files < 0 || i.Files > MaxFiles || i.Bytes < 0 || i.Bytes > MaxTreeBytes {
		return errors.New("invalid source staging intent")
	}
	return nil
}
func (i StageIntent) Check(m Manifest) error {
	if i.Validate() != nil || m.Validate() != nil || m.Digest != i.SourceDigest || len(m.Files) != i.Files {
		return errors.New("staged source differs from transfer intent")
	}
	var bytes int64
	for _, f := range m.Files {
		bytes += f.Bytes
	}
	if bytes != i.Bytes {
		return errors.New("staged source size differs from transfer intent")
	}
	body, _ := json.Marshal(m)
	if len(body) > MaxManifestBytes {
		return errors.New("source manifest exceeds transfer bound")
	}
	return nil
}

type StageReceipt struct {
	Version   int         `json:"version"`
	Intent    StageIntent `json:"intent"`
	Actor     string      `json:"actor"`
	CreatedAt time.Time   `json:"created_at"`
}

func (r StageReceipt) Validate() error {
	if r.Version != 1 || r.Intent.Validate() != nil || r.Actor == "" || !settingText(r.Actor, 128, false) || r.CreatedAt.IsZero() {
		return errors.New("invalid source stage receipt")
	}
	return nil
}

type CachedStage struct {
	Receipt  StageReceipt `json:"receipt"`
	Manifest Manifest     `json:"manifest"`
}

func (s CachedStage) Validate() error {
	if s.Receipt.Validate() != nil || s.Receipt.Intent.Check(s.Manifest) != nil {
		return errors.New("invalid staged source metadata")
	}
	body, _ := json.Marshal(s.Manifest)
	if len(body) > MaxManifestBytes {
		return errors.New("staged manifest exceeds bound")
	}
	return nil
}
