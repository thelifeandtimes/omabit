//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const (
	maxPriorRuns        = 256
	maxPriorReportBytes = 64 << 10
)

// Called only while holding the account lock. Read just the bounded base listing
// and each run's private report; never enumerate or stat retained data/piers.
// This is a gate, not an old-job recovery service. Missing, unsafe or unresolved
// metadata requires manual resolution even when the daemon inventory is empty.
func (q *qualifier) priorReports(ctx context.Context) error {
	base, err := os.Open(q.base)
	if err != nil {
		return errors.New("cannot read qualification metadata directory")
	}
	entries, readErr := base.ReadDir(maxPriorRuns + 2) // runs, .lock, overflow
	closeErr := base.Close()
	if (readErr != nil && !errors.Is(readErr, io.EOF)) || closeErr != nil || len(entries) > maxPriorRuns+1 {
		return errors.New("qualification metadata unavailable or entry limit exceeded; manual review required")
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Name() == ".lock" {
			continue
		}
		if !runIDPattern.MatchString(entry.Name()) {
			return errors.New("unexpected qualification base entry; manual review required")
		}
		if err := q.priorReport(entry.Name()); err != nil {
			return fmt.Errorf("prior run %s requires manual resolution: %w", entry.Name(), err)
		}
	}
	return nil
}

func (q *qualifier) priorReport(id string) error {
	dir := filepath.Join(q.base, id)
	path := filepath.Join(dir, "report.json")
	if err := q.owned(dir, os.ModeDir, true); err != nil {
		return err
	}
	if err := q.owned(path, 0, true); err != nil {
		return err
	}
	n, err := q.d.stat(path)
	if err != nil || n.links != 1 {
		return errors.New("unsafe report link metadata")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return errors.New("private report unavailable")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxPriorReportBytes {
		return errors.New("report must be a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxPriorReportBytes+1))
	if err != nil || len(data) > maxPriorReportBytes {
		return errors.New("report read failed or byte limit exceeded")
	}
	// Pointers distinguish absent/null evidence from a recorded false outcome.
	// Completed legacy reports are accepted only with full container IDs; their
	// old Removed=true/ID-empty outcome cannot prove an uncertain create absent.
	var wire struct {
		RunID            string `json:"run_id"`
		Directory        string `json:"directory"`
		ReportPath       string `json:"report_path"`
		CleanupComplete  *bool  `json:"cleanup_complete"`
		RuntimeQualified *bool  `json:"runtime_qualified"`
		Containers       *[]struct {
			Name             string `json:"name"`
			ID               string `json:"id"`
			Kind             string `json:"kind"`
			Removed          *bool  `json:"removed"`
			CreateDispatched *bool  `json:"create_dispatched"`
			CreateUncertain  *bool  `json:"create_uncertain"`
		} `json:"containers"`
	}
	if json.Unmarshal(data, &wire) != nil || wire.RunID != id || wire.Directory != dir || wire.ReportPath != path || wire.CleanupComplete == nil || wire.RuntimeQualified == nil || *wire.RuntimeQualified {
		return errors.New("malformed report identity or outcome metadata")
	}
	if wire.Containers == nil || len(*wire.Containers) > 5 {
		return errors.New("malformed container intent metadata")
	}
	seen := make(map[string]bool)
	for _, record := range *wire.Containers {
		switch record.Kind {
		case "probe", "boot", "restart", "pack", "post-pack":
		default:
			return errors.New("invalid recorded container kind")
		}
		if record.Name != "urbit-qualification-"+id+"-"+record.Kind || seen[record.Kind] || record.Removed == nil || !*record.Removed {
			return errors.New("malformed or unresolved container intent")
		}
		seen[record.Kind] = true
		if record.CreateDispatched == nil && record.CreateUncertain == nil {
			if !containerIDPattern.MatchString(record.ID) {
				return errors.New("legacy create outcome is uncertain")
			}
			continue
		}
		if record.CreateDispatched == nil || record.CreateUncertain == nil || *record.CreateUncertain || (*record.CreateDispatched && !containerIDPattern.MatchString(record.ID)) || (!*record.CreateDispatched && record.ID != "") {
			return errors.New("malformed or uncertain create outcome")
		}
	}
	if !*wire.CleanupComplete {
		return errors.New("cleanup is unresolved")
	}
	return nil
}
