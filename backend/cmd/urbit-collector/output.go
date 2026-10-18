package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type output struct {
	w    io.Writer
	json *json.Encoder
	text bool
}

func newOutput(w io.Writer, format string, stream bool) *output {
	e := json.NewEncoder(w)
	if !stream {
		e.SetIndent("", "  ")
	}
	return &output{w: w, json: e, text: format == "text"}
}

// Remote labels and messages must not introduce terminal control sequences.
func cell(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, s)
}

func loomLabel(exponent uint8) string {
	if !protocol.ValidRecordedLoom(exponent) {
		return "unknown"
	}
	if exponent < 30 {
		return fmt.Sprintf("%d (%d MiB)", exponent, uint64(1)<<(exponent-20))
	}
	return fmt.Sprintf("%d (%d GiB)", exponent, uint64(1)<<(exponent-30))
}

func controlLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func instances(w io.Writer, values []protocol.Instance) {
	fmt.Fprintln(w, "INSTANCE\tDESIRED\tOBSERVED\tHEALTH\tREV / APPLIED\tENDPOINT")
	for _, v := range values {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d / %d\t%s\n", cell(v.ID), cell(v.DesiredState), cell(v.ObservedState), cell(v.Health), v.ConfigRevision, v.AppliedRevision, cell(v.Endpoint))
		fmt.Fprintf(w, "  Requested loom: %s address space\n  Requested native control: %s\n", loomLabel(v.Config.Loom), controlLabel(v.Config.NativeControl))
		if d := v.RuntimeStatus; d != nil {
			if d.LaunchLoom != nil {
				fmt.Fprintf(w, "  Recorded launch loom: %s address space\n", loomLabel(*d.LaunchLoom))
			}
			if d.LaunchNativeControl != nil {
				fmt.Fprintf(w, "  Recorded launch native control: %s\n", controlLabel(*d.LaunchNativeControl))
			}
			if d.NativeControlSocket != "" {
				fmt.Fprintf(w, "  Host-local socket path: %s (path only, reachability not verified)\n", cell(d.NativeControlSocket))
			}
		}
		fmt.Fprintln(w, "  Loom is address space, not resident memory or the container memory limit.")
		if v.RecoveryRequired {
			fmt.Fprintln(w, "  Recovery required")
		}
		if v.RollNeedsRuntime {
			fmt.Fprintln(w, "  A new roll requires a runtime start and clean stop first")
		}
		if v.ActiveOperationID != "" {
			fmt.Fprintln(w, "  Active operation:", cell(v.ActiveOperationID))
		}
		if e := v.Epochs; e != nil {
			fmt.Fprintf(w, "  Epoch inventory: %d entries; sampled %s; source=%s\n", len(e.Entries), e.ObservedAt.UTC().Format(time.RFC3339Nano), cell(e.Source))
			fmt.Fprintf(w, "  Inventory digest: %s\n", cell(e.Digest))
			for _, entry := range e.Entries {
				fmt.Fprintf(w, "    %s: logical=%d bytes; allocated=%d bytes; files=%d; protected=%t; shared-files=%t\n", cell(entry.Name), entry.LogicalBytes, entry.AllocatedBytes, entry.FileCount, entry.Protected, entry.SharedFiles)
			}
		}
		if r := v.Resources; r != nil {
			fmt.Fprintf(w, "  Container resources: %s; source=%s\n", cell(r.Status), cell(r.Source))
			if r.ObservedAt != nil {
				fmt.Fprintf(w, "  Sample: %s; container=%s\n", r.ObservedAt.UTC().Format(time.RFC3339Nano), cell(r.ContainerID))
			}
			if (r.Status == "fresh" || r.Status == "stale") && r.MemoryCurrentBytes != nil && r.MemoryLimitBytes != nil && r.PIDsCurrent != nil {
				fmt.Fprintf(w, "  Cgroup RAM (includes cache): %.2f / %.2f MiB; tasks: %d\n",
					float64(*r.MemoryCurrentBytes)/(1<<20), float64(*r.MemoryLimitBytes)/(1<<20), *r.PIDsCurrent)
				if r.CPUUsagePercent != nil && r.CPULimitCores != nil {
					fmt.Fprintf(w, "  Container CPU: %.2f%% (100%% = one CPU; limit %.0f%%)\n", *r.CPUUsagePercent, *r.CPULimitCores*100)
				}
			}
		}
		if m := v.Maintenance; m != nil {
			if len(m.EpochsBefore) > 0 {
				fmt.Fprintf(w, "  Roll epochs before: %s\n", cell(strings.Join(m.EpochsBefore, ", ")))
				if len(m.EpochsAfter) > 0 {
					fmt.Fprintf(w, "  Observed epochs after: %s\n", cell(strings.Join(m.EpochsAfter, ", ")))
				}
				if m.FinishedAt != nil && m.CreatedEpoch != "" {
					fmt.Fprintf(w, "  Created epoch: %s\n", cell(m.CreatedEpoch))
				}
			}
			after := "unknown"
			if m.FinishedAt != nil {
				after = fmt.Sprint(m.AfterPierBytes)
			}
			fmt.Fprintf(w, "  Maintenance: %s; logical pier bytes before=%d after=%s; completed=%t\n", cell(m.Kind), m.BeforePierBytes, after, m.FinishedAt != nil)
			fmt.Fprintf(w, "  Started: %s\n", m.StartedAt.UTC().Format(time.RFC3339Nano))
			if m.FinishedAt != nil {
				fmt.Fprintf(w, "  Finished: %s (%.2f seconds)\n", m.FinishedAt.UTC().Format(time.RFC3339Nano), m.FinishedAt.Sub(m.StartedAt).Seconds())
			}
			if m.MemAvailableBytes != 0 && m.DiskAvailableBytes != 0 {
				fmt.Fprintf(w, "  Admission: %.2f GiB available RAM; %.2f GiB free disk\n", float64(m.MemAvailableBytes)/(1<<30), float64(m.DiskAvailableBytes)/(1<<30))
			}
		}
		if p := v.Pruning; p != nil {
			fmt.Fprintf(w, "  Epoch pruning: cutoff=%s; removed-files=%d/%d; selected-epochs-removed=%t; uncertain=%t; completed=%t\n", cell(p.Cutoff), p.RemovedFiles, p.ExpectedFiles, p.EpochRemoved, p.Uncertain, p.FinishedAt != nil)
			fmt.Fprintf(w, "  Pruning inventory digest: %s\n", cell(p.InventoryDigest))
			if len(p.SelectedEpochs) > 0 {
				fmt.Fprintf(w, "  Selected epochs: %s\n", cell(strings.Join(p.SelectedEpochs, ", ")))
			}
			if len(p.RemovedEpochs) > 0 {
				fmt.Fprintf(w, "  Removed epochs: %s\n", cell(strings.Join(p.RemovedEpochs, ", ")))
			}
			if len(p.RetainedEpochs) > 0 {
				fmt.Fprintf(w, "  Observed retained epochs: %s\n", cell(strings.Join(p.RetainedEpochs, ", ")))
			}
		}
	}
}

func observation(w io.Writer, v collection.Observation) {
	fmt.Fprintf(w, "\n%s — %s (%s), checked %s\n", cell(v.Name), cell(v.Status), cell(v.RunnerID), v.CheckedAt.UTC().Format(time.RFC3339))
	if v.Error != nil {
		fmt.Fprintf(w, "  %s: %s\n", cell(v.Error.Code), cell(v.Error.Message))
	}
	if v.Snapshot != nil {
		instances(w, v.Snapshot.Instances)
		for _, incident := range v.Snapshot.Incidents {
			if incident.Active {
				fmt.Fprintf(w, "  Incident %s: %s\n", cell(incident.Code), cell(incident.Message))
			}
		}
	}
}

func (o *output) Encode(value any) error {
	if !o.text {
		return o.json.Encode(value)
	}
	// Build the table in memory so errors from the final writer are propagated.
	var text strings.Builder
	w := tabwriter.NewWriter(&text, 0, 4, 2, ' ', 0)
	switch v := value.(type) {
	case protocol.ErrorResponse:
		fmt.Fprintf(w, "Error %s: %s\n", cell(v.Error.Code), cell(v.Error.Message))
		if v.Error.CurrentRevision != nil {
			fmt.Fprintln(w, "Current revision:", *v.Error.CurrentRevision)
		}
	case protocol.Hello:
		fmt.Fprintf(w, "Runner: %s\nMode: %s\nProtocol: %d\nFeatures: %s\n", cell(v.RunnerID), cell(v.Mode), v.ProtocolVersion, cell(strings.Join(v.Features, ", ")))
		if v.Actor != "" {
			fmt.Fprintf(w, "Actor: %s\nAccess: %s\n", cell(v.Actor), cell(v.AccessRole))
		}
	case protocol.Snapshot:
		fmt.Fprintf(w, "Runner: %s (%s)\n", cell(v.RunnerID), cell(v.Mode))
		instances(w, v.Instances)
		fmt.Fprintln(w, "\nOPERATION\tINSTANCE\tKIND\tSTATUS\tACTOR\tREQUEST")
		for _, op := range v.Operations {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", cell(op.ID), cell(op.InstanceID), cell(op.Kind), cell(op.Status), cell(op.Actor), cell(op.RequestID))
		}
		for _, incident := range v.Incidents {
			fmt.Fprintf(w, "Incident %s (active=%t): %s\n", cell(incident.Code), incident.Active, cell(incident.Message))
		}
	case *pierstore.Catalogue:
		fmt.Fprintf(w, "Pier root: %s\n", cell(v.Root))
		for _, p := range v.Candidates {
			fmt.Fprintf(w, "%s\t%s\tregistered=%s\tmoved=%t\tbusy=%t\t%s\n", cell(p.Path), cell(p.Kind), cell(p.EnrolledBy), p.Moved, p.Busy, cell(p.Problem))
		}
		if v.Error != "" {
			fmt.Fprintln(w, cell(v.Error))
		}
	case protocol.Instance:
		instances(w, []protocol.Instance{v})
	case epochs.Preview:
		fmt.Fprintf(w, "Epoch prefix preview: %s on %s, revision %d\nInventory digest: %s\nCutoff: %s (inclusive)\n", cell(v.InstanceID), cell(v.RunnerID), v.ConfigRevision, cell(v.InventoryDigest), cell(v.Cutoff))
		for _, e := range v.Selected {
			fmt.Fprintf(w, "  Selected: %s\n", cell(e.Name))
		}
		for _, e := range v.Retained {
			fmt.Fprintf(w, "  Retained: %s (protected=%t)\n", cell(e.Name), e.Protected)
		}
		fmt.Fprintf(w, "Selected logical bytes: %d\nSelected allocated file bytes: %d\nShared files: %t\nDeletion supported: %t\nRequires fresh stopped-pier revalidation before deletion.\nAllocated file bytes are not a guarantee of disk space reclaimed.\n", v.SelectedLogicalBytes, v.SelectedAllocatedBytes, v.SharedFiles, v.DeletionSupported)
		if v.DeletionSupported {
			fmt.Fprintln(w, "The displayed oldest-prefix selection is supported. Explicit history-loss confirmation is required.")
		}
	case protocol.Operation:
		fmt.Fprintf(w, "Operation: %s\nInstance: %s\nKind: %s\nStatus: %s\nActor: %s\nRequest: %s\n", cell(v.ID), cell(v.InstanceID), cell(v.Kind), cell(v.Status), cell(v.Actor), cell(v.RequestID))
		fmt.Fprintf(w, "Revision: %d\nCreated: %s\n", v.ConfigRevision, v.CreatedAt.UTC().Format(time.RFC3339Nano))
		if v.FinishedAt != nil {
			fmt.Fprintf(w, "Finished: %s\n", v.FinishedAt.UTC().Format(time.RFC3339Nano))
		}
		if v.Message != "" {
			fmt.Fprintf(w, "Message: %s\n", cell(v.Message))
		}
	case collection.Connection:
		fmt.Fprintf(w, "Connection: %s\nRunner: %s\nMode: %s\nAddress: %s\nConnection revision: %d\nEnrolled: %s\n", cell(v.Name), cell(v.RunnerID), cell(v.Mode), cell(v.Address()), v.Revision, v.EnrolledAt.UTC().Format(time.RFC3339))
	case protocol.ImportCatalogue:
		fmt.Fprintf(w, "Runner: %s\nQualified fake import sources: %d\n", cell(v.RunnerID), len(v.Sources))
		for _, s := range v.Sources {
			fmt.Fprintf(w, "  Source: %s\n    Ship: %s; files: %d; logical bytes: %d\n    Digest: %s\n    Claimed by: %s\n", cell(s.RunID), cell(s.Ship), s.Files, s.Bytes, cell(s.ContentDigest), cell(s.ClaimedBy))
		}
	case []collection.Connection:
		fmt.Fprintln(w, "NAME\tTRANSPORT\tRUNNER\tMODE\tADDRESS")
		for _, c := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", cell(c.Name), cell(c.Transport), cell(c.RunnerID), cell(c.Mode), cell(c.Address()))
		}
	case collection.Observation:
		observation(w, v)
	case collection.Inventory:
		fmt.Fprintf(w, "Inventory complete: %t (%d connections)\n", v.Complete, len(v.Connections))
		for _, c := range v.Connections {
			observation(w, c)
		}
	case shipInfo:
		fmt.Fprintf(w, "Ship connection: %s\nShip: %s\nOrigin: %s\nCredentials: %s\nStatus: %s\n", cell(v.Name), cell(v.Ship), cell(v.Origin), cell(v.Credentials), cell(v.Status))
		if v.Message != "" {
			fmt.Fprintln(w, cell(v.Message))
		}
	case []shipInfo:
		fmt.Fprintln(w, "NAME\tSHIP\tSTATUS\tCREDENTIALS\tORIGIN")
		for _, s := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", cell(s.Name), cell(s.Ship), cell(s.Status), cell(s.Credentials), cell(s.Origin))
		}
	case shipData:
		fmt.Fprintf(w, "%s: %s %s\n", cell(v.Name), cell(v.App), cell(v.Path))
		data, err := json.MarshalIndent(v.Data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, safeJSONText(string(data)))
	case shipEvent:
		fmt.Fprintf(w, "%s: %s", cell(v.Name), cell(v.Kind))
		if v.Response != "" {
			fmt.Fprintf(w, " — %s, event %d", cell(v.Response), v.EventID)
		}
		fmt.Fprintln(w)
		if len(v.Data) > 0 {
			data, err := json.MarshalIndent(v.Data, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(w, safeJSONText(string(data)))
		}
	default:
		// Small confirmation objects retain readable, indented JSON.
		return o.json.Encode(value)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_, err := io.WriteString(o.w, text.String())
	return err
}

func safeJSONText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) || (unicode.IsControl(r) && r != '\n' && r != '\t') {
			return ' '
		}
		return r
	}, s)
}
