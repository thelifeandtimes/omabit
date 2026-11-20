package recording

import (
	"context"
	"encoding/json"
	"time"
)

type TimingQuery struct {
	ExpectedSourceSHA256 string `json:"expected_source_sha256,omitempty"`
	AfterSequence        uint64 `json:"after_sequence"`
	FromUnixNano         int64  `json:"from_unix_nano"`
	ThroughUnixNano      int64  `json:"through_unix_nano"`
	Kind                 Kind   `json:"kind"`
	Limit                uint16 `json:"limit"`
	ContextBefore        uint8  `json:"context_before"`
}

func (q TimingQuery) Validate() error {
	if q.ExpectedSourceSHA256 != "" && !APIHex(q.ExpectedSourceSHA256, 64) || q.AfterSequence > 0 && q.ExpectedSourceSHA256 == "" || q.FromUnixNano < 0 || q.ThroughUnixNano < 0 || q.ThroughUnixNano > 0 && q.FromUnixNano > q.ThroughUnixNano || q.Kind > Exit || q.Limit < 1 || q.Limit > 256 || q.ContextBefore > 32 {
		return ErrSelection
	}
	return nil
}

type TimingQueryResult struct {
	Version           int                `json:"version"`
	Source            TimingSource       `json:"source"`
	Index             TimingIndexSummary `json:"index"`
	Query             TimingQuery        `json:"query"`
	Entries           []TimingEntry      `json:"entries"`
	HasMore           bool               `json:"has_more"`
	NextAfterSequence uint64             `json:"next_after_sequence,omitempty"`
	// nil means start of the retained original. This is a page selector, never
	// new file/action authority. Reading original or redacted context still needs
	// the exact operator-private job/session selection and explicit representation.
	ContextFrom *Cursor   `json:"context_from,omitempty"`
	ReadAt      time.Time `json:"read_at"`
	Basis       string    `json:"basis"`
}

type timingQuerySink struct {
	query          TimingQuery
	result         TimingQueryResult
	previous       []TimingEntry
	started, ended bool
}

// The producer supplies one bounded complete JSONL record per Write. This sink
// is private to that producer, not a parser for an arbitrary untrusted file.
func (s *timingQuerySink) Write(p []byte) (int, error) {
	var row TimingIndexRecord
	if json.Unmarshal(p, &row) != nil || s.ended {
		return 0, ErrCorrupt
	}
	switch row.Type {
	case "header":
		if s.started || row.Source == nil || s.query.ExpectedSourceSHA256 != "" && row.SourceSHA256 != s.query.ExpectedSourceSHA256 {
			return 0, ErrSelection
		}
		s.started = true
		s.result.Source = *row.Source
	case "frame":
		if !s.started || row.Entry == nil {
			return 0, ErrCorrupt
		}
		e := *row.Entry
		match := e.Sequence > s.query.AfterSequence && (s.query.Kind == 0 || s.query.Kind == e.Kind) && (s.query.FromUnixNano == 0 || e.UnixNano >= s.query.FromUnixNano) && (s.query.ThroughUnixNano == 0 || e.UnixNano <= s.query.ThroughUnixNano)
		if match {
			if len(s.result.Entries) == int(s.query.Limit) {
				s.result.HasMore = true
			} else {
				if len(s.result.Entries) == 0 && e.Sequence > uint64(s.query.ContextBefore)+1 {
					wanted := e.Sequence - uint64(s.query.ContextBefore) - 1
					found := false
					for _, prior := range s.previous {
						if prior.Sequence == wanted {
							s.result.ContextFrom = &Cursor{Version: 1, SessionID: s.result.Source.SessionID, MetadataSHA256: s.result.Source.MetadataSHA256, Sequence: prior.Sequence, Digest: prior.Digest}
							found = true
							break
						}
					}
					if !found {
						return 0, ErrCorrupt
					}
				}
				s.result.Entries = append(s.result.Entries, e)
			}
		}
		s.previous = append(s.previous, e)
		if len(s.previous) > 33 {
			copy(s.previous, s.previous[1:])
			s.previous = s.previous[:33]
		}
	case "end":
		if !s.started || row.End == nil || !row.End.AllSourceFilesVerified || !row.End.VerifiedPrefixComplete || row.End.RuntimeComplete {
			return 0, ErrCorrupt
		}
		s.ended = true
	default:
		return 0, ErrCorrupt
	}
	return len(p), nil
}

// QueryTiming reuses the same bounded verified index producer. It returns no
// matches until all selected file hashes, codec prefix and final coverage agree.
// Wall time may reverse, so sequence remains the ordering/continuation key.
func QueryTiming(ctx context.Context, in ExportRequest, result ExportManifestResult, query TimingQuery, fetch func(context.Context, ExportPartRequest) (ExportPart, error)) (TimingQueryResult, error) {
	var zero TimingQueryResult
	if query.Validate() != nil {
		return zero, ErrSelection
	}
	sink := &timingQuerySink{query: query, result: TimingQueryResult{Version: 1, Query: query, Entries: []TimingEntry{}, ReadAt: result.Manifest.ReadAt, Basis: "operator-private exact closed original source; complete verified prefix timing scan only, not payload search, runtime health or action authority; time bounds inclusive, rows ordered by sequence; original context needs separate explicit private read"}}
	summary, err := WriteTimingIndex(ctx, in, result, fetch, sink)
	if err != nil || !sink.ended {
		if err == nil {
			err = ErrCorrupt
		}
		return zero, err
	}
	sink.result.Index = summary
	if sink.result.HasMore {
		sink.result.NextAfterSequence = sink.result.Entries[len(sink.result.Entries)-1].Sequence
	}
	return sink.result, nil
}
