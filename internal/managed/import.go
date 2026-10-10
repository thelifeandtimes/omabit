package managed

import (
	"context"
	"net/http"
	"regexp"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type importBackend interface {
	ImportSources(context.Context) ([]protocol.ImportSource, error)
}

func (realBackend) ImportSources(ctx context.Context) ([]protocol.ImportSource, error) {
	return runtimequalify.ListImportSources(ctx)
}

var importRunID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func importRequest(req protocol.ImportFakeRequest) (protocol.CreateFakeRequest, *protocol.ImportSource, error) {
	base := protocol.CreateFakeRequest{ID: req.ID, Label: req.Label, Config: req.Config, RequestID: req.RequestID}
	if !req.ConfirmStopped || !importRunID.MatchString(req.SourceRunID) || !epochs.ValidDigest(req.ExpectedDigest) {
		return base, nil, bad("import requires a qualified source ID, its expected digest and explicit stopped-source confirmation")
	}
	return base, &protocol.ImportSource{RunID: req.SourceRunID, ContentDigest: req.ExpectedDigest}, nil
}

func updateImportReport(v *storedInstance) {
	if p := v.Runtime.Import; p != nil {
		v.Public.Import = &protocol.ImportReport{SourceRunID: p.Source.RunID, ContentDigest: p.Source.ContentDigest, CopiedFiles: p.CopiedFiles, CopiedBytes: p.CopiedBytes, ExpectedFiles: p.Source.Files, ExpectedBytes: p.Source.Bytes, StartedAt: p.StartedAt, FinishedAt: p.FinishedAt, Uncertain: p.Uncertain || p.FinishedAt == nil}
	}
}

// Called under the normal mutation mutex. Exact replay precedes catalogue reads,
// so moving/removing a completed source cannot change historical operation replay.
func (r *Server) serveImports(w http.ResponseWriter, req *http.Request, actor, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "imports require operator access"}})
		return
	}
	b, ok := r.b.(importBackend)
	if !ok {
		r.respondError(w, &fault{501, protocol.APIError{Code: "unsupported", Message: "qualified imports unavailable"}})
		return
	}
	var body protocol.ImportFakeRequest
	var base protocol.CreateFakeRequest
	var ref *protocol.ImportSource
	if req.Method == http.MethodPost {
		if err := decode(w, req, &body); err != nil {
			r.respondError(w, err)
			return
		}
		var err error
		base, ref, err = importRequest(body)
		if err != nil {
			r.respondError(w, err)
			return
		}
		fingerprint, err := registrationFingerprint(&base, ref)
		if err != nil {
			r.respondError(w, err)
			return
		}
		if op, found, err := replay(r.s.db, actor, base.RequestID, fingerprint); found || err != nil {
			if err != nil {
				r.respondError(w, err)
			} else {
				respond(w, 200, op)
			}
			return
		}
	} else if req.Method != http.MethodGet {
		r.respondError(w, bad("unsupported import method"))
		return
	}
	sources, err := b.ImportSources(req.Context())
	if err != nil {
		r.respondError(w, conflict("source_unavailable", "qualified import catalogue is unavailable or needs review"))
		return
	}
	all, err := instances(r.s.db)
	if err != nil {
		r.respondError(w, err)
		return
	}
	for i := range sources {
		for _, v := range all {
			if v.Runtime.Import != nil && v.Runtime.Import.Source.RunID == sources[i].RunID {
				sources[i].ClaimedBy = v.Public.ID
			}
		}
	}
	if req.Method == http.MethodGet {
		if sources == nil {
			sources = []protocol.ImportSource{}
		}
		respond(w, 200, protocol.ImportCatalogue{RunnerID: r.s.runnerID, Sources: sources})
		return
	}
	for _, source := range sources {
		if source.RunID != ref.RunID {
			continue
		}
		if source.ContentDigest != ref.ContentDigest || source.ClaimedBy != "" || source.Ship != "~zod" {
			r.respondError(w, conflict("source_changed_or_claimed", "select an unchanged, unclaimed qualified fake source"))
			return
		}
		op, replayed, err := r.s.registerInstance(actor, base, &source)
		if err != nil {
			r.respondError(w, err)
			return
		}
		status := 200
		if !replayed {
			r.launch(op)
			status = 202
		}
		respond(w, status, op)
		return
	}
	r.respondError(w, conflict("source_not_found", "source is not a completed import-qualified fake"))
}
