package managed

import (
	"context"
	"net/http"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type recordingExportBackend interface {
	RecordingExportManifest(context.Context, recording.ExportRequest) (recording.ExportManifestResult, error)
	RecordingExportPart(context.Context, recording.ExportPartRequest) (recording.ExportPart, error)
}

func (r *Server) serveRecordingExport(w http.ResponseWriter, req *http.Request, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "original recording exports require operator access"}})
		return
	}
	b, ok := r.b.(recordingExportBackend)
	if !ok {
		r.respondError(w, &fault{501, protocol.APIError{Code: "unsupported", Message: "recording export adapter unavailable"}})
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), protocol.RecordingExportTimeout)
	defer cancel()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.RecordingExportTimeout + time.Second))
	var selection recording.ExportRequest
	var part recording.ExportPartRequest
	if req.URL.Path == protocol.RecordingExportManifestPath {
		if err := decode(w, req, &selection); err != nil {
			r.respondError(w, err)
			return
		}
	} else if req.URL.Path == protocol.RecordingExportPartPath {
		if err := decode(w, req, &part); err != nil {
			r.respondError(w, err)
			return
		}
		selection = part.ExportRequest
		if part.Validate() != nil {
			r.respondError(w, bad("require an exact saved original export chunk"))
			return
		}
	} else {
		r.respondError(w, bad("unsupported recording export path"))
		return
	}
	if selection.Validate() != nil {
		r.respondError(w, bad("require an exact job/digest and explicit private original export choice"))
		return
	}
	i, err := r.s.recordingReadSelection(ctx, selection.PageSelection())
	if err != nil {
		r.respondError(w, conflict("recording_selection_changed", "exact original saved recording intent required"))
		return
	}
	var out any
	if req.URL.Path == protocol.RecordingExportManifestPath {
		value, err := b.RecordingExportManifest(ctx, selection)
		if err != nil || value.Validate(selection) != nil || !reflect.DeepEqual(value.Session, recordingDescriptor(i)) {
			r.respondError(w, conflict("recording_export_unavailable", "closed original recording export is unavailable or changed; preserve its saved evidence"))
			return
		}
		out = value
	} else {
		value, err := b.RecordingExportPart(ctx, part)
		if err != nil || value.Validate() != nil || value.Request != part {
			r.respondError(w, conflict("recording_export_part_changed", "exact saved chunk is unavailable or changed; do not substitute another session or manifest"))
			return
		}
		out = value
	}
	after, err := r.s.recordingReadSelection(ctx, selection.PageSelection())
	if err != nil || after != i {
		r.respondError(w, conflict("recording_selection_changed", "saved recording binding changed during export read"))
		return
	}
	respond(w, 200, out)
}
