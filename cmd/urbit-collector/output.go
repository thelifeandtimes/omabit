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

func instances(w io.Writer, values []protocol.Instance) {
	fmt.Fprintln(w, "INSTANCE\tDESIRED\tOBSERVED\tHEALTH\tREV / APPLIED\tENDPOINT")
	for _, v := range values {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d / %d\t%s\n", cell(v.ID), cell(v.DesiredState), cell(v.ObservedState), cell(v.Health), v.ConfigRevision, v.AppliedRevision, cell(v.Endpoint))
		if v.RecoveryRequired {
			fmt.Fprintln(w, "  Recovery required")
		}
		if v.ActiveOperationID != "" {
			fmt.Fprintln(w, "  Active operation:", cell(v.ActiveOperationID))
		}
		if m := v.Maintenance; m != nil {
			fmt.Fprintf(w, "  Maintenance: %s; logical pier bytes before=%d after=%d; completed=%t\n", cell(m.Kind), m.BeforePierBytes, m.AfterPierBytes, m.FinishedAt != nil)
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
	case protocol.Instance:
		instances(w, []protocol.Instance{v})
	case protocol.Operation:
		fmt.Fprintf(w, "Operation: %s\nInstance: %s\nKind: %s\nStatus: %s\nActor: %s\nRequest: %s\n", cell(v.ID), cell(v.InstanceID), cell(v.Kind), cell(v.Status), cell(v.Actor), cell(v.RequestID))
	case collection.Connection:
		fmt.Fprintf(w, "Connection: %s\nRunner: %s\nMode: %s\nAddress: %s\nEnrolled: %s\n", cell(v.Name), cell(v.RunnerID), cell(v.Mode), cell(v.Address()), v.EnrolledAt.UTC().Format(time.RFC3339))
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
