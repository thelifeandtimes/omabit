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
	"reflect"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const (
	maxPriorRuns        = 256
	maxPriorReportBytes = 128 << 10
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

func (q *qualifier) priorReportData(id string) ([]byte, error) {
	if !runIDPattern.MatchString(id) {
		return nil, errors.New("invalid prior run identity")
	}
	dir := filepath.Join(q.base, id)
	path := filepath.Join(dir, "report.json")
	if err := q.owned(dir, os.ModeDir, true); err != nil {
		return nil, err
	}
	if err := q.owned(path, 0, true); err != nil {
		return nil, err
	}
	n, err := q.d.stat(path)
	if err != nil || n.links != 1 {
		return nil, errors.New("unsafe report link metadata")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("private report unavailable")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxPriorReportBytes {
		return nil, errors.New("report must be a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxPriorReportBytes+1))
	if err != nil || len(data) > maxPriorReportBytes {
		return nil, errors.New("report read failed or byte limit exceeded")
	}
	return data, nil
}

func (q *qualifier) priorReport(id string) error {
	dir := filepath.Join(q.base, id)
	path := filepath.Join(dir, "report.json")
	data, err := q.priorReportData(id)
	if err != nil {
		return err
	}
	// Pointers distinguish absent/null evidence from a recorded false outcome.
	var wire struct {
		Profile          string `json:"profile"`
		RunID            string `json:"run_id"`
		Directory        string `json:"directory"`
		ReportPath       string `json:"report_path"`
		CleanupComplete  *bool  `json:"cleanup_complete"`
		RuntimeQualified *bool  `json:"runtime_qualified"`
		Comet            *struct {
			CometRecord
			Uncertain *bool `json:"uncertain"`
		} `json:"comet"`
		CometUtilities []*struct {
			CometUtilityRecord
			Uncertain       *bool  `json:"uncertain"`
			BeforePierBytes *int64 `json:"before_pier_bytes"`
		} `json:"comet_utilities"`
		CometFinalEpochs *epochs.Inventory `json:"comet_final_epochs"`
		PierImport       *struct {
			PierImportRecord
			Uncertain *bool `json:"uncertain"`
		} `json:"pier_import"`
		Meld *struct {
			MeldRecord
			Uncertain       *bool  `json:"uncertain"`
			BeforePierBytes *int64 `json:"before_pier_bytes"`
		} `json:"meld"`
		EpochLifecycle *struct {
			Initial *epochs.Inventory `json:"initial"`
			Rolls   *[]struct {
				EpochRollRecord
				Uncertain *bool `json:"uncertain"`
			} `json:"rolls"`
			Final   *epochs.Inventory `json:"final"`
			Preview *epochs.Preview   `json:"preview"`
		} `json:"epoch_lifecycle"`
		Prune *struct {
			PruneRecord
			Uncertain *bool `json:"uncertain"`
		} `json:"prune"`
		Recovery *struct {
			EpochRecoveryRecord
			Uncertain *bool `json:"uncertain"`
		} `json:"epoch_recovery"`
		PrefixPrunes []*struct {
			PrefixPruneRecord
			Uncertain *bool `json:"uncertain"`
			Recovery  *struct {
				EpochRecoveryRecord
				Uncertain *bool `json:"uncertain"`
			} `json:"recovery"`
		} `json:"prefix_prunes"`
		Containers *[]struct {
			Name             string `json:"name"`
			ID               string `json:"id"`
			Kind             string `json:"kind"`
			SnapshotMinutes  uint32 `json:"snapshot_minutes"`
			Removed          *bool  `json:"removed"`
			CreateDispatched *bool  `json:"create_dispatched"`
			CreateUncertain  *bool  `json:"create_uncertain"`
			HostHTTPVerified *bool  `json:"host_http_verified"`
		} `json:"containers"`
		Network *struct {
			Name             string `json:"name"`
			ID               string `json:"id"`
			Removed          *bool  `json:"removed"`
			CreateDispatched *bool  `json:"create_dispatched"`
			CreateUncertain  *bool  `json:"create_uncertain"`
		} `json:"network"`
	}
	if json.Unmarshal(data, &wire) != nil || wire.RunID != id || wire.Directory != dir || wire.ReportPath != path || wire.CleanupComplete == nil || wire.RuntimeQualified == nil || *wire.RuntimeQualified {
		return errors.New("malformed report identity or outcome metadata")
	}
	maxContainers := 5
	if wire.Profile == ProfileCometMaintenance {
		maxContainers = 7
	}
	if wire.Profile == ProfileCometEpochs {
		maxContainers = 9
	}
	if wire.Profile == ProfileCometPrefix {
		maxContainers = 17
	}
	if wire.Profile == ProfileEpochs {
		maxContainers = 8
	}
	if wire.Profile == ProfilePrune {
		maxContainers = 11
	}
	if wire.Profile == ProfilePrefix {
		maxContainers = 16
	}
	if wire.Containers == nil || len(*wire.Containers) > maxContainers {
		return errors.New("malformed container intent metadata")
	}
	if wire.Profile != ProfileLifecycle && wire.Profile != ProfileHTTP && wire.Profile != ProfileMeld && wire.Profile != ProfileImport && !cometProfile(wire.Profile) && !isEpochProfile(wire.Profile) {
		return errors.New("unknown prior qualification profile")
	}
	if !networkProfile(wire.Profile) && wire.Network != nil {
		return errors.New("isolated report contains a network intent")
	}
	if c := wire.Comet; c != nil {
		count := 3
		if wire.Profile == ProfileCometMaintenance {
			count = 7
		}
		if cometEpochProfile(wire.Profile) {
			count = 9
			if wire.Profile == ProfileCometPrefix {
				count = 17
				if len(wire.PrefixPrunes) != 2 || wire.CometFinalEpochs == nil {
					return errors.New("comet prefix outcome lacks both rounds or final inventory")
				}
			}
			if wire.EpochLifecycle == nil || wire.EpochLifecycle.Final == nil || wire.EpochLifecycle.Preview == nil || wire.EpochLifecycle.Rolls == nil || len(*wire.EpochLifecycle.Rolls) != epochRollCount(wire.Profile) {
				return errors.New("comet epoch qualification lacks its complete roll history and preview")
			}
		}
		if !cometProfile(wire.Profile) || c.Uncertain == nil || *c.Uncertain || c.StartedAt.IsZero() || c.FinishedAt == nil || c.FinishedAt.Before(c.StartedAt) || !cometName.MatchString(c.Ship) || !c.BootCompleted || !c.ResumeCompleted || len(*wire.Containers) != count || wire.Network == nil {
			return errors.New("comet provisioning outcome is incomplete or uncertain; manual review required")
		}
		if wire.Profile == ProfileCometMaintenance && len(wire.CometUtilities) != 2 {
			return errors.New("comet maintenance lacks both utility outcomes")
		}
	}
	if wire.CometUtilities != nil {
		if wire.Profile != ProfileCometMaintenance || wire.Comet == nil || len(wire.CometUtilities) != 2 {
			return errors.New("invalid comet utility history")
		}
		for i, m := range wire.CometUtilities {
			want := []string{"comet-pack", "comet-meld"}[i]
			if m == nil || m.Kind != want || m.Uncertain == nil || *m.Uncertain || !m.RestartVerified || m.StartedAt.IsZero() || m.FinishedAt == nil || m.FinishedAt.Before(m.StartedAt) || m.BeforePierBytes == nil || *m.BeforePierBytes < 0 || m.AfterPierBytes == nil || *m.AfterPierBytes < 0 || m.MemAvailableBytes < 5<<30 || m.DiskAvailableBytes < 8<<30 {
				return errors.New("comet utility outcome is incomplete or uncertain")
			}
		}
	}
	if p := wire.PierImport; p != nil {
		if len(*wire.Containers) != 4 {
			return errors.New("completed pier import requires its four container outcomes")
		}
		if wire.Profile != ProfileImport || p.Uncertain == nil || *p.Uncertain || p.StartedAt.IsZero() || p.FinishedAt == nil || p.FinishedAt.Before(p.StartedAt) || !p.SourceMoved || !p.CopyVerified || !p.FirstResume || !p.ReplacementResume || !p.SourceUnchanged || p.CopiedFiles < 1 || p.CopiedFiles > 8192 || p.CopiedBytes < 0 || p.CopiedBytes > protocol.MaxPierBytes || !epochs.ValidDigest(p.ContentDigest) {
			return errors.New("pier import outcome is missing, malformed or uncertain; manual review required")
		}
	}
	if wire.Meld != nil {
		m := wire.Meld
		if wire.Profile != ProfileMeld || m.Uncertain == nil || *m.Uncertain || m.StartedAt.IsZero() || m.FinishedAt == nil || m.FinishedAt.Before(m.StartedAt) || m.BeforePierBytes == nil || *m.BeforePierBytes < 0 || m.AfterPierBytes == nil || *m.AfterPierBytes < 0 || m.MemAvailableBytes < 5<<30 || m.DiskAvailableBytes < 8<<30 {
			return errors.New("meld outcome is missing, malformed or uncertain; pier requires manual review")
		}
	}
	if e := wire.EpochLifecycle; e != nil {
		if cometEpochProfile(wire.Profile) && wire.Comet == nil {
			return errors.New("comet epoch history lacks its pinned identity outcome")
		}
		if !isEpochProfile(wire.Profile) || validEpochSample(e.Initial, id, 1) != nil || e.Initial.Entries[0].Name != "0i0" || e.Rolls == nil || len(*e.Rolls) > epochRollCount(wire.Profile) {
			return errors.New("invalid epoch qualification baseline or roll history")
		}
		previous := e.Initial
		for i, r := range *e.Rolls {
			if cometEpochProfile(wire.Profile) && r.PostRestart == nil {
				return errors.New("comet roll lacks its identity-preserving restart inventory")
			}
			if r.Kind != fmt.Sprintf("roll-%d", i+1) || r.Uncertain == nil || *r.Uncertain || r.StartedAt.IsZero() || r.FinishedAt == nil || r.FinishedAt.Before(r.StartedAt) || r.MemAvailableBytes < 5<<30 || r.DiskAvailableBytes < 8<<30 || validEpochSample(r.Before, id, i+1) != nil || validEpochSample(r.After, id, i+2) != nil || !sameEpochNames(previous, r.Before) || !epochPrefix(r.Before, r.After) {
				return errors.New("roll outcome is missing, malformed or uncertain; pier requires manual review")
			}
			if r.PostRestart != nil && (validEpochSample(r.PostRestart, id, i+2) != nil || !sameEpochNames(r.After, r.PostRestart)) {
				return errors.New("invalid post-roll restart inventory")
			}
			previous = r.PostRestart
		}
		if e.Final != nil || e.Preview != nil {
			count := epochRollCount(wire.Profile)
			if len(*e.Rolls) != count || validEpochSample(e.Final, id, count+1) != nil || !sameEpochNames(previous, e.Final) || e.Preview == nil {
				return errors.New("invalid final epoch qualification inventory")
			}
			preview, err := epochs.Plan(e.Final, e.Final.Entries[count-3].Name, e.Final.Digest, e.Final.ObservedAt)
			if err != nil || !reflect.DeepEqual(preview, *e.Preview) {
				return errors.New("invalid epoch prefix preview evidence")
			}
		}
	}
	if p := wire.Prune; p != nil {
		if wire.Profile != ProfilePrune || wire.EpochLifecycle == nil || p.Uncertain == nil || *p.Uncertain || validPruneRecord(&p.PruneRecord, &EpochLifecycleRecord{Final: wire.EpochLifecycle.Final}, id) != nil {
			return errors.New("prune outcome is missing, malformed or uncertain; manual review required")
		}
	}
	if r := wire.Recovery; r != nil {
		if wire.Profile != ProfilePrune || wire.Prune == nil || r.Uncertain == nil || *r.Uncertain || validRecoveryRecord(&r.EpochRecoveryRecord, &wire.Prune.PruneRecord, id) != nil {
			return errors.New("checkpoint recovery is missing, malformed or uncertain; manual review required")
		}
	}
	if wire.PrefixPrunes != nil {
		if !prefixProfile(wire.Profile) || len(wire.PrefixPrunes) > 2 || wire.EpochLifecycle == nil || wire.EpochLifecycle.Final == nil {
			return errors.New("invalid prefix qualification history")
		}
		previous := wire.EpochLifecycle.Final
		for i, p := range wire.PrefixPrunes {
			if wire.Profile == ProfileCometPrefix && (p == nil || p.PostRestart == nil || p.Recovery == nil || p.Recovery.PostRestart == nil) {
				return errors.New("comet prefix history lacks ordinary/recovered restart evidence")
			}
			if p == nil || p.Uncertain == nil || *p.Uncertain || validPrefixRecord(&p.PrefixPruneRecord, previous, id, i+1) != nil {
				return errors.New("prefix deletion is missing, malformed or uncertain; manual review required")
			}
			previous = nil
			if r := p.Recovery; r != nil {
				if r.Uncertain == nil || *r.Uncertain || validLatestRecovery(&r.EpochRecoveryRecord, p.PostRestart, id) != nil {
					return errors.New("prefix recovery is missing, malformed or uncertain; manual review required")
				}
				previous = r.PostRestart
			}
		}
	}
	if inv := wire.CometFinalEpochs; inv != nil {
		if wire.Profile != ProfileCometPrefix || wire.Comet == nil || len(wire.PrefixPrunes) != 2 || wire.PrefixPrunes[1] == nil || wire.PrefixPrunes[1].Recovery == nil || validEpochSample(inv, id, 2) != nil || !sameEpochNames(wire.PrefixPrunes[1].Recovery.PostRestart, inv) {
			return errors.New("invalid final retained comet epoch inventory")
		}
	}
	if wire.Network != nil {
		n := wire.Network
		if n.Name != "urbit-qualification-"+id+"-http" || n.Removed == nil || !*n.Removed ||
			n.CreateDispatched == nil || n.CreateUncertain == nil || *n.CreateUncertain ||
			(*n.CreateDispatched && !containerIDPattern.MatchString(n.ID)) || (!*n.CreateDispatched && n.ID != "") {
			return errors.New("malformed or unresolved network intent")
		}
	}
	seen := make(map[string]bool)
	for _, record := range *wire.Containers {
		if record.SnapshotMinutes != 0 && record.SnapshotMinutes != DefaultSnapshotMinutes {
			return errors.New("invalid recorded snapshot interval")
		}
		switch record.Kind {
		case "probe":
		case "boot":
			if cometProfile(wire.Profile) || wire.Profile != ProfileLifecycle && wire.Profile != ProfileMeld && wire.Profile != ProfileImport && !isEpochProfile(wire.Profile) {
				return errors.New("boot container does not match isolated profile")
			}
		case "restart", "pack", "post-pack":
			if wire.Profile != ProfileLifecycle {
				return errors.New("container kind does not match the HTTP profile")
			}
		case "meld", "post-meld":
			if wire.Profile != ProfileMeld || wire.Meld == nil || record.CreateDispatched == nil || !*record.CreateDispatched {
				return errors.New("meld container lacks resolved maintenance evidence")
			}
		case "import-resume", "import-replacement":
			if wire.Profile != ProfileImport || wire.PierImport == nil || record.CreateDispatched == nil || !*record.CreateDispatched {
				return errors.New("import resume lacks completed transfer evidence")
			}
		case "roll-1", "roll-2", "roll-3", "roll-4", "epoch-restart-1", "epoch-restart-2", "epoch-restart-3", "epoch-restart-4":
			i := int(record.Kind[len(record.Kind)-1] - '1')
			if !isEpochProfile(wire.Profile) || cometProfile(wire.Profile) || wire.EpochLifecycle == nil || i >= len(*wire.EpochLifecycle.Rolls) || i >= epochRollCount(wire.Profile) {
				return errors.New("epoch container lacks resolved roll evidence")
			}
			if isRollKind(record.Kind) && (record.CreateDispatched == nil || !*record.CreateDispatched) {
				return errors.New("roll was not dispatched")
			}
		case "post-prune":
			if wire.Profile != ProfilePrune || wire.Prune == nil {
				return errors.New("post-prune container lacks deletion evidence")
			}
		case "epoch-recovery", "post-recovery":
			if wire.Profile != ProfilePrune || wire.Recovery == nil || record.CreateDispatched == nil || !*record.CreateDispatched {
				return errors.New("recovery container lacks resolved replay evidence")
			}
		case "prefix-restart-1", "prefix-restart-2", "prefix-recovery-1", "prefix-recovery-2", "prefix-recovered-1", "prefix-recovered-2":
			i := int(record.Kind[len(record.Kind)-1] - '1')
			if wire.Profile != ProfilePrefix || i >= len(wire.PrefixPrunes) || wire.PrefixPrunes[i] == nil {
				return errors.New("prefix container lacks resolved deletion evidence")
			}
			if record.Kind != fmt.Sprintf("prefix-restart-%d", i+1) && wire.PrefixPrunes[i].Recovery == nil {
				return errors.New("prefix replay container lacks resolved recovery evidence")
			}
			if isPrefixRecovery(record.Kind) && (record.CreateDispatched == nil || !*record.CreateDispatched) {
				return errors.New("prefix replay was not dispatched")
			}
		case "http-boot", "http-restart":
			if wire.Profile != ProfileHTTP || wire.Network == nil || !*wire.Network.CreateDispatched {
				return errors.New("HTTP container lacks a resolved network intent")
			}
		case "comet-boot", "comet-resume", "comet-post-pack", "comet-post-meld", "comet-epoch-restart-1", "comet-epoch-restart-2", "comet-epoch-restart-3", "comet-epoch-restart-4", "comet-prefix-restart-1", "comet-prefix-restart-2", "comet-prefix-recovered-1", "comet-prefix-recovered-2":
			if !cometProfile(wire.Profile) || isCometRestart(record.Kind) && wire.Profile != ProfileCometMaintenance || isCometEpochRestart(record.Kind) && !cometEpochContainer(wire.Profile, record.Kind) || isCometPrefixRuntime(record.Kind) && wire.Profile != ProfileCometPrefix || wire.Comet == nil || wire.Network == nil || wire.Network.CreateDispatched == nil || !*wire.Network.CreateDispatched || record.HostHTTPVerified == nil || !*record.HostHTTPVerified || record.CreateDispatched == nil || !*record.CreateDispatched {
				return errors.New("comet runtime lacks completed identity and network evidence")
			}
		case "comet-roll-1", "comet-roll-2", "comet-roll-3", "comet-roll-4":
			i := int(record.Kind[len(record.Kind)-1] - '1')
			if !cometEpochContainer(wire.Profile, record.Kind) || wire.EpochLifecycle == nil || wire.EpochLifecycle.Rolls == nil || i >= len(*wire.EpochLifecycle.Rolls) || record.CreateDispatched == nil || !*record.CreateDispatched {
				return errors.New("comet roll lacks completed epoch evidence")
			}
		case "comet-prefix-recovery-1", "comet-prefix-recovery-2":
			i := int(record.Kind[len(record.Kind)-1] - '1')
			if wire.Profile != ProfileCometPrefix || i >= len(wire.PrefixPrunes) || wire.PrefixPrunes[i] == nil || wire.PrefixPrunes[i].Recovery == nil || record.CreateDispatched == nil || !*record.CreateDispatched {
				return errors.New("comet recovery lacks resolved replay evidence")
			}
		case "comet-pack", "comet-meld":
			if wire.Profile != ProfileCometMaintenance || len(wire.CometUtilities) != 2 || record.CreateDispatched == nil || !*record.CreateDispatched {
				return errors.New("comet utility lacks completed maintenance evidence")
			}
		default:
			return errors.New("invalid recorded container kind")
		}
		if record.Name != "urbit-qualification-"+id+"-"+record.Kind || seen[record.Kind] || record.Removed == nil || !*record.Removed {
			return errors.New("malformed or unresolved container intent")
		}
		seen[record.Kind] = true
		if record.CreateDispatched == nil || record.CreateUncertain == nil || *record.CreateUncertain || (*record.CreateDispatched && !containerIDPattern.MatchString(record.ID)) || (!*record.CreateDispatched && record.ID != "") {
			return errors.New("malformed or uncertain create outcome")
		}
	}
	if wire.Meld != nil && !seen["meld"] {
		return errors.New("meld evidence lacks a recorded utility container")
	}
	if len(wire.CometUtilities) > 0 {
		for _, kind := range []string{"comet-pack", "comet-post-pack", "comet-meld", "comet-post-meld"} {
			if !seen[kind] {
				return errors.New("comet maintenance lacks its utility/restart containers")
			}
		}
	}
	if e := wire.EpochLifecycle; e != nil {
		for i, r := range *e.Rolls {
			if !seen[epochContainerKind(wire.Profile, r.Kind)] || (r.PostRestart != nil && !seen[epochContainerKind(wire.Profile, fmt.Sprintf("epoch-restart-%d", i+1))]) {
				return errors.New("roll evidence lacks matching owned container records")
			}
		}
	}
	if !*wire.CleanupComplete {
		return errors.New("cleanup is unresolved")
	}
	if wire.Prune != nil && wire.Prune.PostRestart != nil && !seen["post-prune"] {
		return errors.New("prune restart evidence lacks a container")
	}
	if wire.Recovery != nil && (!seen["epoch-recovery"] || (wire.Recovery.PostRestart != nil && !seen["post-recovery"])) {
		return errors.New("recovery evidence lacks matching containers")
	}
	for i, p := range wire.PrefixPrunes {
		if (p.PostRestart != nil && !seen[epochContainerKind(wire.Profile, fmt.Sprintf("prefix-restart-%d", i+1))]) || (p.Recovery != nil && (!seen[epochContainerKind(wire.Profile, fmt.Sprintf("prefix-recovery-%d", i+1))] || (p.Recovery.PostRestart != nil && !seen[epochContainerKind(wire.Profile, fmt.Sprintf("prefix-recovered-%d", i+1))]))) {
			return errors.New("prefix evidence lacks matching containers")
		}
	}
	return nil
}
