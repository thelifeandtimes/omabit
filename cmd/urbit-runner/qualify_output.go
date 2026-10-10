package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func writeRuntimePlan(w io.Writer, p map[string]any) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Profile: %s\nMode: %s\nAccount: %s (%s)\n", p["profile"], p["mode"], p["account"], p["home"])
	fmt.Fprintf(&b, "Engine: %s\nVere: %s\nImage: %s\n", p["engine_version"], p["vere_version"], p["image"])
	pill := p["pill"].(candidate.Artifact)
	fmt.Fprintf(&b, "Pill: %s\nPill SHA-256: %s (%d bytes)\n", pill.URL, pill.SHA256, pill.Bytes)
	fmt.Fprintf(&b, "Admission: %.0f GiB available RAM; %.0f GiB free disk\n", float64(p["minimum_available_ram_bytes"].(int64))/(1<<30), float64(p["minimum_available_disk_bytes"].(int64))/(1<<30))
	fmt.Fprintf(&b, "Container: %.0f GiB RAM; no additional swap; %v CPUs; %v PIDs; loom %v\n", float64(p["container_memory_bytes"].(int64))/(1<<30), p["cpus"], p["pids"], p["loom_exponent"])
	fmt.Fprintf(&b, "Network: %s\nControl socket: %s\n", p["network"], p["control_socket"])
	for _, key := range []string{"management", "utility", "host_probe", "measurements", "failure_policy"} {
		if value, ok := p[key]; ok {
			fmt.Fprintf(&b, "%s: %s\n", key, value)
		}
	}
	b.WriteString("\nSteps:\n")
	for i, step := range p["steps"].([]string) {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, step)
	}
	fmt.Fprintf(&b, "\nRemaining gates: %s\nProduction runtime qualified: no\n", strings.Join(p["remaining_gates"].([]string), "; "))
	_, err := io.WriteString(w, b.String())
	return err
}

func writeRuntimeResult(w io.Writer, r runtimequalify.Result) error {
	if r.RunID == "" {
		_, err := io.WriteString(w, "Checkpoint: NOT STARTED\nNo qualification run was created; see the qualification error.\n")
		return err
	}
	var b strings.Builder
	status := "FAILED"
	if r.CheckpointPassed {
		status = "PASSED"
	}
	fmt.Fprintf(&b, "\nCheckpoint: %s\nProfile: %s\nRun: %s\nReport: %s\n", status, r.Profile, r.RunID, r.ReportPath)
	fmt.Fprintf(&b, "Cleanup complete: %t\nLive limits observed: %t\nPier metadata seen: %t\nProduction runtime qualified: %t\n", r.CleanupComplete, r.LimitsObserved, r.PierMetadataSeen, r.RuntimeQualified)
	for _, p := range r.Phases {
		status := "FAIL"
		if p.Passed {
			status = "PASS"
		}
		fmt.Fprintf(&b, "  %-4s %-18s %s\n", status, p.Name, p.Detail)
	}
	if m := r.Meld; m != nil {
		fmt.Fprintf(&b, "\nMeld outcome uncertain: %t\nLogical pier bytes before: %d\n", m.Uncertain, m.BeforePierBytes)
		if m.AfterPierBytes != nil {
			fmt.Fprintf(&b, "Logical pier bytes after:  %d\n", *m.AfterPierBytes)
		}
		fmt.Fprintf(&b, "Meld admission: %.2f GiB available RAM; %.2f GiB free disk\n", float64(m.MemAvailableBytes)/(1<<30), float64(m.DiskAvailableBytes)/(1<<30))
		fmt.Fprintf(&b, "Meld started: %s\n", m.StartedAt.Format(time.RFC3339Nano))
		if m.FinishedAt != nil {
			fmt.Fprintf(&b, "Meld finished: %s (%.2f seconds)\n", m.FinishedAt.Format(time.RFC3339Nano), m.FinishedAt.Sub(m.StartedAt).Seconds())
		}
	}
	if e := r.EpochLifecycle; e != nil {
		if e.Initial != nil {
			fmt.Fprintf(&b, "\nInitial epoch inventory: %d epoch(s); digest %s\n", len(e.Initial.Entries), e.Initial.Digest)
		}
		for _, roll := range e.Rolls {
			fmt.Fprintf(&b, "%s: uncertain=%t; started=%s\n", roll.Kind, roll.Uncertain, roll.StartedAt.Format(time.RFC3339Nano))
			fmt.Fprintf(&b, "  Admission: %.2f GiB available RAM; %.2f GiB free disk\n", float64(roll.MemAvailableBytes)/(1<<30), float64(roll.DiskAvailableBytes)/(1<<30))
			if roll.FinishedAt != nil {
				fmt.Fprintf(&b, "  Finished: %s (%.2f seconds)\n", roll.FinishedAt.Format(time.RFC3339Nano), roll.FinishedAt.Sub(roll.StartedAt).Seconds())
			}
			if roll.After != nil {
				fmt.Fprintf(&b, "  Epoch count after roll: %d\n", len(roll.After.Entries))
			}
			fmt.Fprintf(&b, "  Post-roll restart inventory verified: %t\n", roll.PostRestart != nil)
		}
		if e.Final != nil {
			label := "Final inventory"
			if r.Profile == runtimequalify.ProfilePrune || r.Profile == runtimequalify.ProfilePrefix || r.Profile == runtimequalify.ProfileCometPrefix {
				label = "Prepared inventory before deletion"
			}
			fmt.Fprintf(&b, "%s: %s\n", label, e.Final.Digest)
			for _, entry := range e.Final.Entries {
				fmt.Fprintf(&b, "  %s: logical=%d bytes; allocated=%d bytes; snapshot=%d bytes; protected=%t\n", entry.Name, entry.LogicalBytes, entry.AllocatedBytes, entry.SnapshotBytes, entry.Protected)
			}
		}
		if p := e.Preview; p != nil {
			fmt.Fprintf(&b, "Prefix preview cutoff: %s; selected=%d; retained=%d\n", p.Cutoff, len(p.Selected), len(p.Retained))
			if r.Profile == runtimequalify.ProfilePrune || r.Profile == runtimequalify.ProfilePrefix || r.Profile == runtimequalify.ProfileCometPrefix {
				fmt.Fprintf(&b, "Managed preview deletion supported: %t\n", p.DeletionSupported)
			} else {
				fmt.Fprintf(&b, "Deletion supported: %t\n", p.DeletionSupported)
			}
			for _, entry := range p.Selected {
				fmt.Fprintf(&b, "  Selected: %s\n", entry.Name)
			}
			for _, entry := range p.Retained {
				fmt.Fprintf(&b, "  Retained: %s\n", entry.Name)
			}
		}
	}
	if p := r.Prune; p != nil {
		fmt.Fprintf(&b, "\nPrefix deletion: cutoff=%s; uncertain=%t; epoch-removed=%t\n", p.Cutoff, p.Uncertain, p.EpochRemoved)
		fmt.Fprintf(&b, "  Files removed: %d / %d\n", len(p.RemovedFiles), len(p.Files))
		fmt.Fprintf(&b, "  Retained metadata verified: %t\n", p.FinishedAt != nil && !p.Uncertain)
		if p.After != nil {
			for _, e := range p.After.Entries {
				fmt.Fprintf(&b, "  Observed retained epoch: %s; digest=%s\n", e.Name, e.Digest)
			}
		}
		fmt.Fprintf(&b, "  Ordinary post-prune restart verified: %t\n", p.PostRestart != nil)
	}
	if r := r.Recovery; r != nil {
		fmt.Fprintf(&b, "Latest-epoch recovery: uncertain=%t; current-checkpoint-removed=%t\n", r.Uncertain, r.CheckpointRemoved)
		if r.FinishedAt != nil && !r.Uncertain {
			fmt.Fprintf(&b, "  Restored checkpoint: %d logical bytes\n", r.RecoveredCheckpointBytes)
		} else {
			b.WriteString("  Restored checkpoint: unverified\n")
		}
		fmt.Fprintf(&b, "  Post-recovery restart verified: %t\n", r.PostRestart != nil)
		if r.Before != nil && len(r.Before.Entries) > 0 {
			fmt.Fprintf(&b, "  Recovery target epoch: %s\n", r.Before.Entries[len(r.Before.Entries)-1].Name)
		}
	}
	for i, p := range r.PrefixPrunes {
		fmt.Fprintf(&b, "\nPrefix round %d: cutoff=%s; uncertain=%t; completed=%t\n", i+1, p.Cutoff, p.Uncertain, p.FinishedAt != nil)
		for _, target := range p.Targets {
			fmt.Fprintf(&b, "  Selected %s: files-removed=%d/%d; directory-removed=%t\n", target.Name, len(target.RemovedFiles), len(target.Files), target.Removed)
		}
		fmt.Fprintf(&b, "  Retained metadata verified: %t\n  Ordinary restart verified: %t\n", p.FinishedAt != nil && !p.Uncertain, p.PostRestart != nil)
		if p.After != nil {
			for _, e := range p.After.Entries {
				fmt.Fprintf(&b, "  Observed retained: %s; digest=%s\n", e.Name, e.Digest)
			}
		}
		if recovery := p.Recovery; recovery != nil {
			fmt.Fprintf(&b, "  Latest-epoch recovery: uncertain=%t; checkpoint-removed=%t; post-recovery-restart=%t\n", recovery.Uncertain, recovery.CheckpointRemoved, recovery.PostRestart != nil)
			if recovery.FinishedAt != nil && !recovery.Uncertain {
				fmt.Fprintf(&b, "  Restored checkpoint: %d logical bytes\n", recovery.RecoveredCheckpointBytes)
			}
		}
	}
	for _, c := range r.Containers {
		fmt.Fprintf(&b, "Container %s: removed=%t, create-uncertain=%t, private-http=%t\n", c.Name, c.Removed, c.CreateUncertain, c.PrivateHTTPReachable)
		if s := c.LastState; s != nil {
			fmt.Fprintf(&b, "  Last Docker state: observed=%s; running=%t; paused=%t; restarting=%t; dead=%t; oom-killed=%t; exit-code=%d\n", s.ObservedAt.Format(time.RFC3339Nano), s.Running, s.Paused, s.Restarting, s.Dead, s.OOMKilled, s.ExitCode)
		}
		if c.HostEndpoint != "" {
			fmt.Fprintf(&b, "  Host HTTP: %s (verified=%t)\n", c.HostEndpoint, c.HostHTTPVerified)
		}
	}
	if p := r.PierImport; p != nil {
		fmt.Fprintf(&b, "\nPier copy: uncertain=%t; original-reserved=%t\n  Copied files: %d\n  Logical bytes: %d\n  Content digest: %s\n  Copy verified: %t\n  First resume: %t\n  Replacement resume: %t\n  Original unchanged: %t\n", p.Uncertain, p.SourceMoved, p.CopiedFiles, p.CopiedBytes, p.ContentDigest, p.CopyVerified, p.FirstResume, p.ReplacementResume, p.SourceUnchanged)
	}
	if c := r.Comet; c != nil {
		fmt.Fprintf(&b, "\nComet provisioning: uncertain=%t\n  Public identity: %s\n  Initial boot completed: %t\n  Same-identity resume completed: %t\n", c.Uncertain, c.Ship, c.BootCompleted, c.ResumeCompleted)
		if c.FinishedAt != nil {
			fmt.Fprintf(&b, "  Completed: %s\n", c.FinishedAt.UTC().Format(time.RFC3339Nano))
		}
	}
	if inv := r.CometFinalEpochs; inv != nil {
		fmt.Fprintf(&b, "Final retained comet inventory: %s\n", inv.Digest)
		for _, e := range inv.Entries {
			fmt.Fprintf(&b, "  Retained: %s; snapshot=%d bytes; protected=%t\n", e.Name, e.SnapshotBytes, e.Protected)
		}
	}
	for _, m := range r.CometUtilities {
		fmt.Fprintf(&b, "\n%s: uncertain=%t; identity-preserving restart=%t\n  Logical bytes before: %d\n", m.Kind, m.Uncertain, m.RestartVerified, m.BeforePierBytes)
		if m.AfterPierBytes != nil {
			fmt.Fprintf(&b, "  Logical bytes after: %d\n", *m.AfterPierBytes)
		}
		fmt.Fprintf(&b, "  Admission: %.2f GiB RAM; %.2f GiB disk\n", float64(m.MemAvailableBytes)/(1<<30), float64(m.DiskAvailableBytes)/(1<<30))
	}
	if n := r.Network; n != nil {
		fmt.Fprintf(&b, "Network %s: removed=%t, create-uncertain=%t\n", n.Name, n.Removed, n.CreateUncertain)
	}
	if r.Failure != "" {
		fmt.Fprintf(&b, "Failure: %s\n", r.Failure)
	}
	_, err := io.WriteString(w, b.String())
	return err
}
