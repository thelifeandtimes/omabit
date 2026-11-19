package recording

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"time"
)

var ErrCapacity = errors.New("recording admission budget is exhausted or cannot be established; preserve saved jobs and inspect recording capacity")

type InstanceCommitment struct {
	InstanceID     string `json:"instance_id"`
	Jobs           int    `json:"jobs"`
	CommittedBytes uint64 `json:"committed_bytes"`
}

// Capacity charges every retained job's full immutable session limit, including
// closed and unclaimed jobs. It is an admission commitment, not measured disk
// usage or proof that missing output was never produced. Retirement is separate.
type Capacity struct {
	Version        int                  `json:"version"`
	RunnerID       string               `json:"runner_id"`
	Jobs           int                  `json:"jobs"`
	CommittedBytes uint64               `json:"committed_bytes"`
	Instances      []InstanceCommitment `json:"instances"`
	AvailableBytes uint64               `json:"available_bytes"`
	ReadAt         time.Time            `json:"read_at"`
	Basis          string               `json:"basis"`
}

func (c Capacity) Validate() error {
	if c.Version != 1 || !protocol.ValidID(c.RunnerID) || c.Jobs < 0 || c.Jobs > 4096 || c.Instances == nil || len(c.Instances) > 4096 || c.ReadAt.IsZero() {
		return ErrSelection
	}
	previous := ""
	jobs := 0
	var total uint64
	for _, i := range c.Instances {
		if !protocol.ValidID(i.InstanceID) || i.InstanceID <= previous || i.Jobs < 1 || i.Jobs > 4096 || i.CommittedBytes < uint64(i.Jobs)<<20 || i.CommittedBytes > uint64(i.Jobs)<<30 {
			return ErrSelection
		}
		previous = i.InstanceID
		jobs += i.Jobs
		total += i.CommittedBytes
	}
	if jobs != c.Jobs || total != c.CommittedBytes {
		return ErrSelection
	}
	return nil
}
func (c Capacity) Admit(p RetentionPolicy, instance string) error {
	if c.Validate() != nil || p.Validate() != nil || c.AvailableBytes < p.ReserveFreeBytes+uint64(p.MaxSessionBytes) || !protocol.ValidID(instance) || c.Jobs >= p.MaxJobs || c.CommittedBytes > p.HostMaxRawBytes || uint64(p.MaxSessionBytes) > p.HostMaxRawBytes-c.CommittedBytes {
		return ErrCapacity
	}
	limit, _ := p.ForInstance(instance)
	for _, i := range c.Instances {
		if i.InstanceID == instance && (i.CommittedBytes > limit || uint64(p.MaxSessionBytes) > limit-i.CommittedBytes) {
			return ErrCapacity
		}
	}
	return nil
}
