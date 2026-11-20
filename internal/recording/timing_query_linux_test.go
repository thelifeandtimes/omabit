//go:build linux

package recording

import (
	"context"
	"strings"
	"testing"
)

func TestTimingQueryUsesStableSourceSequenceAndOriginalContext(t *testing.T) {
	s, _, in, result := timingIndexFixture(t, "pass")
	query := TimingQuery{Kind: Stdout, Limit: 2, ContextBefore: 1}
	first, err := QueryTiming(t.Context(), in, result, query, s.ExportPart)
	if err != nil || len(first.Entries) != 2 || first.Entries[0].Sequence != 1 || first.Entries[1].Sequence != 3 || !first.HasMore || first.NextAfterSequence != 3 || first.ContextFrom != nil || first.Index.RuntimeComplete {
		t.Fatal("first timing query wrong", err, first)
	}
	query.AfterSequence = first.NextAfterSequence
	query.ExpectedSourceSHA256 = first.Index.SourceSHA256
	next, err := QueryTiming(t.Context(), in, result, query, s.ExportPart)
	if err != nil || len(next.Entries) != 2 || next.Entries[0].Sequence != 5 || next.ContextFrom == nil || next.ContextFrom.Sequence != 3 || next.ContextFrom.Digest != first.Entries[1].Digest {
		t.Fatal("sequence/context continuation wrong", err, next)
	}
	page, err := s.ReadPage(t.Context(), result.Manifest.Metadata.SessionID, next.ContextFrom, 3, 3*MaxFrameBytes)
	if err != nil || len(page.Frames) != 3 || page.Frames[0].Sequence != 4 || page.Frames[1].Sequence != 5 || !strings.Contains(string(page.Frames[1].Data), "private-payload-marker") {
		t.Fatal("context cursor did not locate original bytes", err)
	}
	query.FromUnixNano = result.Manifest.Metadata.AttachedAt.UnixNano() - 8
	query.ThroughUnixNano = result.Manifest.Metadata.AttachedAt.UnixNano() - 4
	query.AfterSequence = 0
	query.Limit = 10
	rangeResult, err := QueryTiming(t.Context(), in, result, query, s.ExportPart)
	if err != nil || len(rangeResult.Entries) != 3 || rangeResult.Entries[0].Sequence != 5 || rangeResult.Entries[2].Sequence != 9 || rangeResult.HasMore {
		t.Fatal("reverse wall clock filter skipped frames", err, rangeResult)
	}
}

func TestTimingQueryWithholdsUnverifiedOrChangedSources(t *testing.T) {
	s, _, in, result := timingIndexFixture(t, "corrupt")
	q := TimingQuery{Limit: 1, ContextBefore: 32}
	out, err := QueryTiming(t.Context(), in, result, q, s.ExportPart)
	if err != nil || out.Index.Coverage.Tail != "corrupt" || !out.Index.VerifiedPrefixComplete {
		t.Fatal("damaged retained prefix disappeared", err)
	}
	q.ExpectedSourceSHA256 = strings.Repeat("f", 64)
	if got, err := QueryTiming(t.Context(), in, result, q, s.ExportPart); err == nil || len(got.Entries) != 0 {
		t.Fatal("stale source returned matches")
	}
	q.ExpectedSourceSHA256 = ""
	q.AfterSequence = 1
	calls := 0
	fetch := func(ctx context.Context, in ExportPartRequest) (ExportPart, error) {
		calls++
		return s.ExportPart(ctx, in)
	}
	if _, err := QueryTiming(t.Context(), in, result, q, fetch); err == nil || calls != 0 {
		t.Fatal("unbound continuation read source")
	}
	q.AfterSequence = 0
	fetch = func(ctx context.Context, in ExportPartRequest) (ExportPart, error) {
		part, err := s.ExportPart(ctx, in)
		if err == nil {
			part.Data[0] ^= 1
		}
		return part, err
	}
	if got, err := QueryTiming(t.Context(), in, result, q, fetch); err == nil || len(got.Entries) != 0 {
		t.Fatal("invalid data leaked early matches")
	}
}
