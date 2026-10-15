package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type logsBackend interface {
	Logs(context.Context, runtimequalify.ManagedState) (protocol.RuntimeLogs, error)
}

func (realBackend) Logs(ctx context.Context, state runtimequalify.ManagedState) (protocol.RuntimeLogs, error) {
	return runtimequalify.ReadManagedLogs(ctx, state)
}

func (r *Server) serveLogs(w http.ResponseWriter, req *http.Request, role, id string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "runtime logs require operator access"}})
		return
	}
	var body protocol.LogsRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	v, err := loadInstance(r.s.db, id)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if body.ExpectedRevision == 0 || body.ExpectedRevision != v.Public.ConfigRevision {
		r.respondError(w, conflict("revision_conflict", "refresh the selected ship before reading logs"))
		return
	}
	b, ok := r.b.(logsBackend)
	if !ok {
		r.respondError(w, &fault{501, protocol.APIError{Code: "unsupported", Message: "runtime logs unavailable"}})
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 4*time.Second)
	defer cancel()
	logs, err := b.Logs(ctx, v.Runtime)
	if err != nil {
		r.respondError(w, &fault{502, protocol.APIError{Code: "logs_unavailable", Message: "Selected runtime logs are unavailable; the container may be changing. Refresh to retry."}})
		return
	}
	// Workers can persist replacement intents while a read is in flight. A tail
	// must not be attributed to a different runtime generation or configuration.
	after, err := loadInstance(r.s.db, id)
	if err != nil || after.RuntimeConfigRevision != v.RuntimeConfigRevision || after.Public.ConfigRevision != v.Public.ConfigRevision || after.Runtime.ID != v.Runtime.ID || !sameLogContainer(after.Runtime, logs.ContainerID) {
		r.respondError(w, conflict("runtime_changed", "runtime changed while reading logs; refresh the selection"))
		return
	}
	logs.RunnerID = r.s.runnerID
	logs.InstanceID = id
	logs.ConfigRevision = v.Public.ConfigRevision
	respond(w, 200, logs)
}

func sameLogContainer(s runtimequalify.ManagedState, id string) bool {
	for i := len(s.Runtime.Containers) - 1; i >= 0; i-- {
		c := s.Runtime.Containers[i]
		if c.Kind != "probe" && !c.Removed {
			return c.ID == id
		}
	}
	if s.Runtime.RetainedLogs != nil {
		return s.Runtime.RetainedLogs.ContainerID == id
	}
	return id == ""
}

func attachRuntimeStatus(tx *sql.Tx, v *storedInstance) error {
	s := v.Runtime
	d := &protocol.RuntimeStatus{Stage: s.Runtime.Stage, Initialized: s.Initialized, DefaultSnapshotMinutes: runtimequalify.DefaultSnapshotMinutes}
	if s.ExternalImport != nil {
		v.Public.ExternalImport = &protocol.ExternalImportStatus{Source: s.ExternalImport.Source.Path, HoldingPath: s.ExternalImport.HeldPath, Stage: s.ExternalImport.Stage, CopiedFiles: s.ExternalImport.CopiedFiles, CopiedBytes: s.ExternalImport.CopiedBytes}
		if s.ExternalImport.Stage != "complete" {
			d.Stage = "import-" + s.ExternalImport.Stage
		}
	}
	if s.Deletion != nil {
		d.Stage = "delete-" + s.Deletion.Stage
		v.Public.DeletePending = true
	}
	for i := len(s.Runtime.Containers) - 1; i >= 0; i-- {
		c := s.Runtime.Containers[i]
		if c.Kind == "probe" {
			continue
		}
		d.ContainerID = c.ID
		d.ContainerKind = c.Kind
		switch c.Kind {
		case "http-boot", "http-restart", "comet-boot", "comet-resume", "comet-verify":
			if c.ID != "" {
				minutes := c.SnapshotIntervalMinutes()
				d.SnapshotMinutes = &minutes
			}
		}
		if c.LastState != nil {
			st := c.LastState
			d.LaunchError = st.LaunchError
			d.Running = &st.Running
			d.ExitCode = &st.ExitCode
			d.OOMKilled = &st.OOMKilled
			d.ObservedAt = &st.ObservedAt
		}
		break
	}
	var body []byte
	err := tx.QueryRow(`SELECT body FROM operations WHERE json_extract(body,'$.instance_id')=? AND json_extract(body,'$.config_revision') > COALESCE((SELECT revision FROM deleted_instances WHERE id=?),0) AND json_extract(body,'$.kind') IN ('start','stop','startup-resume','pack','meld','roll','routine-pack','routine-meld','move-pier','enroll-pier','import-pier','delete-pier','prune-epochs') ORDER BY rowid DESC LIMIT 1`, v.Public.ID, v.Public.ID).Scan(&body)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		var op protocol.Operation
		if err = json.Unmarshal(body, &op); err != nil {
			return err
		}
		d.LastOperation = &op
	}
	v.Public.RuntimeStatus = d
	return nil
}
