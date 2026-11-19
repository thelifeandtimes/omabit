package recording

import (
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
