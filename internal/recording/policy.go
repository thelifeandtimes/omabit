package recording

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type InstanceRetention struct {
	InstanceID    string `json:"instance_id"`
	MaxRawBytes   uint64 `json:"max_raw_bytes"`
	MaxAgeSeconds uint64 `json:"max_age_seconds"`
}

// RetentionPolicy controls new admissions/closed history. Existing workers retain
// their immutable job limits. AutoRetire requires separately qualified deletion;
// merely storing/reading a policy never changes files or native lifecycle.
type RetentionPolicy struct {
	Version             int                 `json:"version"`
	MaxJobs             int                 `json:"max_jobs"`
	HostMaxRawBytes     uint64              `json:"host_max_raw_bytes"`
	InstanceMaxRawBytes uint64              `json:"instance_max_raw_bytes"`
	MaxAgeSeconds       uint64              `json:"max_age_seconds"`
	MaxSessionBytes     int64               `json:"max_session_bytes"`
	MaxSegmentBytes     int64               `json:"max_segment_bytes"`
	ReserveFreeBytes    uint64              `json:"reserve_free_bytes"`
	AutoRetire          bool                `json:"auto_retire"`
	Instances           []InstanceRetention `json:"instances"`
}

func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{Version: 1, MaxJobs: 16, HostMaxRawBytes: 1 << 30, InstanceMaxRawBytes: 256 << 20, MaxAgeSeconds: 30 * 24 * 60 * 60, MaxSessionBytes: 64 << 20, MaxSegmentBytes: 8 << 20, ReserveFreeBytes: 2 << 30, Instances: []InstanceRetention{}}
}
func (p RetentionPolicy) Validate() error {
	if p.Version != 1 || p.MaxJobs < 1 || p.MaxJobs > 1024 || p.MaxSessionBytes < 1<<20 || p.MaxSessionBytes > 1<<30 || p.MaxSegmentBytes < 4096 || p.MaxSegmentBytes > 16<<20 || p.MaxSegmentBytes > p.MaxSessionBytes || p.HostMaxRawBytes < uint64(p.MaxSessionBytes) || p.HostMaxRawBytes > 1<<40 || p.InstanceMaxRawBytes < uint64(p.MaxSessionBytes) || p.InstanceMaxRawBytes > p.HostMaxRawBytes || p.ReserveFreeBytes < 512<<20 || p.ReserveFreeBytes > 1<<40 || p.MaxAgeSeconds > 10*365*24*60*60 || p.Instances == nil || len(p.Instances) > 128 {
		return ErrSelection
	}
	previous := ""
	for _, in := range p.Instances {
		if !protocol.ValidID(in.InstanceID) || in.InstanceID <= previous || in.MaxRawBytes < uint64(p.MaxSessionBytes) || in.MaxRawBytes > p.HostMaxRawBytes || in.MaxAgeSeconds > 10*365*24*60*60 {
			return ErrSelection
		}
		previous = in.InstanceID
	}
	return nil
}
func (p RetentionPolicy) ForInstance(id string) (uint64, uint64) {
	for _, in := range p.Instances {
		if in.InstanceID == id {
			return in.MaxRawBytes, in.MaxAgeSeconds
		}
	}
	return p.InstanceMaxRawBytes, p.MaxAgeSeconds
}

type IncidentPin struct {
	JobID     string    `json:"job_id"`
	JobSHA256 string    `json:"job_sha256"`
	Reason    string    `json:"reason"`
	PinnedAt  time.Time `json:"pinned_at"`
}

type ControlRequest struct {
	RunnerID         string           `json:"runner_id"`
	RequestID        string           `json:"request_id"`
	ExpectedRevision uint64           `json:"expected_revision"`
	ExpectedDigest   string           `json:"expected_digest"`
	Kind             string           `json:"kind"`
	Policy           *RetentionPolicy `json:"policy,omitempty"`
	JobID            string           `json:"job_id,omitempty"`
	JobSHA256        string           `json:"job_sha256,omitempty"`
	PinReason        string           `json:"pin_reason,omitempty"`
}

func (r ControlRequest) Validate() error {
	if !protocol.ValidID(r.RunnerID) || !protocol.ValidID(r.RequestID) || r.ExpectedRevision > 4095 || r.ExpectedRevision == 0 && r.ExpectedDigest != "" || r.ExpectedRevision > 0 && !APIHex(r.ExpectedDigest, 64) {
		return ErrSelection
	}
	switch r.Kind {
	case "configure":
		if r.Policy == nil || r.Policy.Validate() != nil || r.JobID != "" || r.JobSHA256 != "" || r.PinReason != "" {
			return ErrSelection
		}
	case "pin", "unpin":
		if r.Policy != nil || !APIHex(r.JobID, 32) || !APIHex(r.JobSHA256, 64) || r.Kind == "pin" && r.PinReason != "incident" && r.PinReason != "owner" || r.Kind == "unpin" && r.PinReason != "" {
			return ErrSelection
		}
	default:
		return ErrSelection
	}
	return nil
}

type ControlEvent struct {
	Version   int            `json:"version"`
	ID        string         `json:"id"`
	RunnerID  string         `json:"runner_id"`
	Actor     string         `json:"actor"`
	Request   ControlRequest `json:"request"`
	CreatedAt time.Time      `json:"created_at"`
}

type ControlState struct {
	Version  int             `json:"version"`
	RunnerID string          `json:"runner_id"`
	Revision uint64          `json:"revision"`
	Digest   string          `json:"digest"`
	Policy   RetentionPolicy `json:"policy"`
	Pins     []IncidentPin   `json:"pins"`
	Basis    string          `json:"basis"`
}

type ControlReceipt struct {
	Event ControlEvent `json:"event"`
	State ControlState `json:"state"`
}

type ControlReadRequest struct {
	RunnerID           string `json:"runner_id"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (r ControlReadRequest) Validate() error {
	if !protocol.ValidID(r.RunnerID) || !r.IncludePrivateData {
		return ErrSelection
	}
	return nil
}

type ControlLookupRequest struct {
	RunnerID           string `json:"runner_id"`
	RequestID          string `json:"request_id"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (r ControlLookupRequest) Validate() error {
	if !protocol.ValidID(r.RunnerID) || !protocol.ValidID(r.RequestID) || !r.IncludePrivateData {
		return ErrSelection
	}
	return nil
}

type ControlStatus struct {
	Version        int          `json:"version"`
	State          ControlState `json:"state"`
	Capacity       *Capacity    `json:"capacity,omitempty"`
	CapacityStatus string       `json:"capacity_status"`
	PilotEnabled   bool         `json:"pilot_enabled"`
	ReadAt         time.Time    `json:"read_at"`
	Basis          string       `json:"basis"`
}
type ControlApplyResult struct {
	Receipt  ControlReceipt `json:"receipt"`
	Replayed bool           `json:"replayed"`
}
type ControlLookupResult struct {
	Version   int             `json:"version"`
	RunnerID  string          `json:"runner_id"`
	RequestID string          `json:"request_id"`
	Found     bool            `json:"found"`
	Receipt   *ControlReceipt `json:"receipt,omitempty"`
}

func (s ControlState) Validate() error {
	if s.Version != 1 || !protocol.ValidID(s.RunnerID) || s.Revision > 4096 || s.Revision == 0 && s.Digest != "" || s.Revision > 0 && !APIHex(s.Digest, 64) || s.Policy.Validate() != nil || s.Pins == nil || len(s.Pins) > 256 {
		return ErrSelection
	}
	last := ""
	for _, p := range s.Pins {
		if !APIHex(p.JobID, 32) || !APIHex(p.JobSHA256, 64) || p.JobID <= last || p.Reason != "incident" && p.Reason != "owner" || p.PinnedAt.IsZero() {
			return ErrSelection
		}
		last = p.JobID
	}
	return nil
}

// Validate binds a returned receipt to its exact request and predecessor. The
// full historical chain and authenticated actor remain runner responsibilities.
func (r ControlReceipt) Validate() error {
	e := r.Event
	if e.Version != 1 || !protocol.ValidID(e.ID) || e.Request.Validate() != nil || e.RunnerID != e.Request.RunnerID || e.Actor == "" || len(e.Actor) > 128 || e.CreatedAt.Before(time.Unix(0, 1)) || r.State.Validate() != nil || r.State.RunnerID != e.RunnerID || r.State.Revision != e.Request.ExpectedRevision+1 {
		return ErrSelection
	}
	for _, b := range e.Actor {
		if b < 33 || b > 126 {
			return ErrSelection
		}
	}
	switch e.Request.Kind {
	case "configure":
		if !reflect.DeepEqual(r.State.Policy, *e.Request.Policy) {
			return ErrSelection
		}
	case "pin":
		found := false
		for _, p := range r.State.Pins {
			if p.JobID == e.Request.JobID {
				found = p.JobSHA256 == e.Request.JobSHA256 && p.Reason == e.Request.PinReason && p.PinnedAt.Equal(e.CreatedAt)
			}
		}
		if !found {
			return ErrSelection
		}
	case "unpin":
		for _, p := range r.State.Pins {
			if p.JobID == e.Request.JobID {
				return ErrSelection
			}
		}
	}
	e.CreatedAt = e.CreatedAt.UTC()
	raw, err := json.Marshal(struct {
		Domain   string       `json:"domain"`
		Ordinal  uint64       `json:"ordinal"`
		Previous string       `json:"previous_digest"`
		Event    ControlEvent `json:"event"`
	}{"omabit-recording-control-v1", r.State.Revision, e.Request.ExpectedDigest, e})
	digest := sha256.Sum256(raw)
	if err != nil || hex.EncodeToString(digest[:]) != r.State.Digest {
		return ErrSelection
	}
	return nil
}
