package recording

import (
	"bytes"
	"encoding/json"
)

// GapEvent states lost or unverified coverage, never a process exit. A nil count
// means unknown, not zero. Counts must come from verified transport/storage
// evidence; absence of received bytes is not evidence that no bytes were emitted.
type GapEvent struct {
	Version           int     `json:"version"`
	Reason            string  `json:"reason"`
	KnownMissingBytes *uint64 `json:"known_missing_bytes,omitempty"`
}

func (g GapEvent) Validate() error {
	if g.Version != 1 {
		return ErrInvalid
	}
	switch g.Reason {
	case "transport-interrupted", "recorder-restarted", "host-disappeared", "disk-pressure", "write-failed", "retention", "corrupt-tail", "bootstrap-not-attached", "process-observation-unavailable", "start-failed-before-process":
		return nil
	}
	return ErrInvalid
}

// ExitEvent records an independently observed terminal process, not transport
// EOF, successful submission or a failed Start that never launched a process.
// Exact container equality with immutable session metadata is a storage/runtime
// admission responsibility. Wall time may reverse under clock adjustment.
type ExitEvent struct {
	Version            int    `json:"version"`
	ContainerID        string `json:"container_id"`
	StartedUnixNano    int64  `json:"started_unix_nano"`
	FinishedUnixNano   int64  `json:"finished_unix_nano"`
	Code               int    `json:"code"`
	OOMKilled          bool   `json:"oom_killed"`
	Dead               bool   `json:"dead"`
	DockerErrorPresent bool   `json:"docker_error_present"`
}

func (e ExitEvent) Validate() error {
	if e.Version != 1 || !lowerHex(e.ContainerID, 64) || e.StartedUnixNano <= 0 || e.FinishedUnixNano <= 0 || e.Code < 0 || e.Code > 255 {
		return ErrInvalid
	}
	return nil
}

func lowerHex(s string, length int) bool {
	if len(s) != length {
		return false
	}
	for _, c := range []byte(s) {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (g GapEvent) Bytes() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(g)
}
func (e ExitEvent) Bytes() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(e)
}

func ReadGap(data []byte) (GapEvent, error) {
	var g GapEvent
	if len(data) > MaxFrameBytes || json.Unmarshal(data, &g) != nil || g.Validate() != nil {
		return GapEvent{}, ErrInvalid
	}
	canonical, _ := g.Bytes()
	if !bytes.Equal(canonical, data) {
		return GapEvent{}, ErrInvalid
	}
	return g, nil
}
func ReadExit(data []byte) (ExitEvent, error) {
	var e ExitEvent
	if len(data) > MaxFrameBytes || json.Unmarshal(data, &e) != nil || e.Validate() != nil {
		return ExitEvent{}, ErrInvalid
	}
	canonical, _ := e.Bytes()
	if !bytes.Equal(canonical, data) {
		return ExitEvent{}, ErrInvalid
	}
	return e, nil
}

// Canonical comparison also refuses unknown/duplicate fields and alternate
// encodings. Consumers must not disagree about what a hashed lifecycle record
// means. Original stream frames are arbitrary bytes and remain untransformed.
func validateLifecycle(kind Kind, data []byte) error {
	switch kind {
	case Gap:
		_, err := ReadGap(data)
		return err
	case Exit:
		_, err := ReadExit(data)
		return err
	}
	return nil
}
