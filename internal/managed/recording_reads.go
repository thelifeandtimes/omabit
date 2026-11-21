package managed

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type recordingReadBackend interface {
	ReadRecording(context.Context, recording.PageRequest) (recording.ReadWindow, error)
}

func recordingDescriptor(i recordingIntent) recording.SessionDescriptor {
	return recording.SessionDescriptor{JobID: i.JobID, JobSHA256: i.JobSHA256, RunnerID: i.RunnerID, Mode: i.Mode, InstanceID: i.InstanceID, RuntimeID: i.RuntimeID, ConfigurationRevision: i.ConfigurationRevision, ContainerID: i.ContainerID, OperationID: i.OperationID, RequestID: i.RequestID, CreatedAt: i.CreatedAt}
}

func (s *store) recordingCatalog(ctx context.Context, in recording.CatalogRequest) (recording.Catalog, error) {
	out := recording.Catalog{Version: 1, RunnerID: s.runnerID, InstanceID: in.InstanceID, Sessions: []recording.SessionDescriptor{}, ReadAt: time.Now().UTC(), Basis: "immutable saved launch intents; recording availability, completeness and current process health require separate evidence"}
	if in.Validate() != nil || in.RunnerID != s.runnerID {
		return out, recording.ErrSelection
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = validateRecordingHistory(tx); err != nil {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id FROM runtime_recording_intents ORDER BY ordinal")
	if err != nil {
		return out, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return out, err
		}
		ids = append(ids, id)
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return out, err
	}
	found := in.AfterJobID == ""
	for _, id := range ids {
		i, err := loadRecordingIntent(tx, id)
		if err != nil {
			return out, err
		}
		if i.InstanceID != in.InstanceID {
			continue
		}
		if !found {
			found = id == in.AfterJobID
			continue
		}
		if len(out.Sessions) == recording.MaxCatalogSessions {
			out.NextJobID = out.Sessions[len(out.Sessions)-1].JobID
			break
		}
		out.Sessions = append(out.Sessions, recordingDescriptor(i))
	}
	if !found || ctx.Err() != nil {
		return recording.Catalog{}, recording.ErrSelection
	}
	return out, nil
}

func (s *store) recordingReadSelection(ctx context.Context, in recording.PageRequest) (recordingIntent, error) {
	if in.Validate() != nil || in.RunnerID != s.runnerID {
		return recordingIntent{}, recording.ErrSelection
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return recordingIntent{}, err
	}
	defer tx.Rollback()
	if err = validateRecordingHistory(tx); err != nil {
		return recordingIntent{}, err
	}
	i, err := loadRecordingIntent(tx, in.JobID)
	if err != nil || i.RunnerID != in.RunnerID || i.InstanceID != in.InstanceID || i.JobSHA256 != in.JobSHA256 || ctx.Err() != nil {
		return recordingIntent{}, recording.ErrSelection
	}
	return i, nil
}

func (r *Server) serveRecordings(w http.ResponseWriter, req *http.Request, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "retained recordings require operator access"}})
		return
	}
	b, ok := r.b.(recordingReadBackend)
	if !ok {
		r.respondError(w, &fault{501, protocol.APIError{Code: "unsupported", Message: "retained recording reads unavailable"}})
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()
	if req.URL.Path == protocol.RecordingCatalogPath {
		var in recording.CatalogRequest
		if err := decode(w, req, &in); err != nil {
			r.respondError(w, err)
			return
		}
		if in.Validate() != nil || in.RunnerID != r.s.runnerID {
			r.respondError(w, bad("require an exact runner/instance and explicit private-data choice"))
			return
		}
		out, err := r.s.recordingCatalog(ctx, in)
		if err != nil {
			r.respondError(w, conflict("recording_history_unavailable", "saved recording history is unavailable or changed; preserve original evidence"))
			return
		}
		respond(w, 200, out)
		return
	}
	var in recording.PageRequest
	if err := decode(w, req, &in); err != nil {
		r.respondError(w, err)
		return
	}
	i, err := r.s.recordingReadSelection(ctx, in)
	if err != nil {
		r.respondError(w, conflict("recording_selection_changed", "require the exact saved job/target/digest and an explicit private representation"))
		return
	}
	out, err := b.ReadRecording(ctx, in)
	if err != nil {
		r.respondError(w, conflict("recording_unavailable", "retained recording or cursor is unavailable; preserve originals and inspect saved coverage"))
		return
	}
	after, err := r.s.recordingReadSelection(ctx, in)
	if err != nil || after != i || !reflect.DeepEqual(out.Session, recordingDescriptor(i)) || out.Representation != in.Representation || out.Version != 1 || out.CompleteExport || out.ReadAt.IsZero() {
		r.respondError(w, conflict("recording_selection_changed", "saved recording binding changed during read"))
		return
	}
	respond(w, 200, out)
}
