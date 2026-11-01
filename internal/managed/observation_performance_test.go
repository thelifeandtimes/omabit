package managed

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Journal-only fixtures: no engine, pier directory, process or network probe.
func benchmarkObservationStore(b *testing.B, count, history int, rolls bool) *store {
	b.Helper()
	dir := b.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		b.Fatal(err)
	}
	s, err := openStore(dir)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { s.close() })
	tx, err := s.db.Begin()
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	for n := 0; n < count; n++ {
		v := storedInstance{Public: protocol.Instance{ID: fmt.Sprintf("fixture-%03d", n), Ship: "~zod", Mode: "fake", ConfigRevision: 1, DesiredState: "stopped", ObservedState: "stopped", ObservedAt: &now}, Runtime: runtimequalify.ManagedState{ID: fmt.Sprintf("%032x", n+1)}}
		body, e := json.Marshal(v)
		if e != nil {
			b.Fatal(e)
		}
		if _, e = tx.Exec("INSERT INTO instances VALUES (?,?)", v.Public.ID, body); e != nil {
			b.Fatal(e)
		}
	}
	for n := 0; n < history; n++ {
		op := protocol.Operation{ID: fmt.Sprintf("op-%08d", n), Actor: "fixture", RequestID: fmt.Sprintf("request-%08d", n), InstanceID: fmt.Sprintf("fixture-%03d", n%max(1, count)), Kind: "start", Status: "succeeded", ConfigRevision: 1, CreatedAt: now, FinishedAt: &now}
		if rolls {
			op.Kind = "roll"
		}
		if err := insertOperation(tx, op, op.RequestID); err != nil {
			b.Fatal(err)
		}
		if rolls {
			data, err := json.Marshal(runtimequalify.ManagedState{ID: fmt.Sprintf("%032x", n%max(1, count)+1), Maintenance: &runtimequalify.ManagedMaintenance{Kind: "roll", CreatedEpoch: fmt.Sprintf("0i%d", n+1), StartedAt: now, FinishedAt: &now}})
			if err != nil {
				b.Fatal(err)
			}
			if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", data, op.ID); err != nil {
				b.Fatal(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	return s
}

func BenchmarkObservationSnapshot(b *testing.B) {
	for _, sample := range []struct{ ships, history int }{{0, 0}, {1, 0}, {32, 0}, {1, 2048}, {32, 2048}, {128, 2048}} {
		b.Run(fmt.Sprintf("ships-%d/history-%d", sample.ships, sample.history), func(b *testing.B) {
			s := benchmarkObservationStore(b, sample.ships, sample.history, false)
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				v, err := s.snapshot()
				if err != nil || len(v.Instances) != sample.ships {
					b.Fatal(err, len(v.Instances))
				}
			}
		})
	}
}

func BenchmarkObservationRollHistory(b *testing.B) {
	for _, sample := range []struct{ ships, history int }{{1, 2048}, {32, 2048}, {128, 2048}, {32, 16384}} {
		b.Run(fmt.Sprintf("ships-%d/history-%d", sample.ships, sample.history), func(b *testing.B) {
			s := benchmarkObservationStore(b, sample.ships, sample.history, true)
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				v, err := s.snapshot()
				if err != nil || len(v.Instances) != sample.ships {
					b.Fatal(err, len(v.Instances))
				}
			}
		})
	}
}

// Retained logs model larger private operation records. The reader must not
// carry these payloads through the history-ranking temporary table.
func BenchmarkObservationRetainedLogs(b *testing.B) {
	s := benchmarkObservationStore(b, 32, 8192, true)
	logs, err := json.Marshal(protocol.RuntimeLogs{Source: "fixture", Lines: []string{strings.Repeat("x", 8192)}})
	if err != nil {
		b.Fatal(err)
	}
	if _, err = s.db.Exec("UPDATE operations SET runtime=json_set(runtime,'$.runtime.retained_logs',json(?))", string(logs)); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		v, err := s.snapshot()
		if err != nil || len(v.Instances) != 32 {
			b.Fatal(err, len(v.Instances))
		}
	}
}
