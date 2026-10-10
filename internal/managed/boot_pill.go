package managed

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (s *store) checkStagedPill(p *protocol.BootPill) error {
	if p == nil {
		return nil
	}
	if !protocol.ValidBootPill(p) {
		return bad("invalid selected pill descriptor")
	}
	f, err := bootpill.Open(bootpill.CachePath(s.directory, *p))
	if err != nil {
		return conflict("pill_unavailable", "upload the selected boot pill before registration")
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil || i.Size() != p.Bytes {
		return conflict("pill_unavailable", "selected pill metadata changed")
	}
	return nil
}

func (r *Server) stagePill(w http.ResponseWriter, req *http.Request, role string) {
	meta := protocol.BootPill{SHA256: req.Header.Get(protocol.PillHashHeader), Bytes: req.ContentLength}
	if role != "operator" || req.Method != http.MethodPost || req.Header.Get("Content-Type") != "application/octet-stream" || !protocol.ValidBootPill(&meta) {
		r.respondError(w, bad("pill upload requires an operator, declared size and SHA-256"))
		return
	}
	controller := http.NewResponseController(w)
	_ = controller.SetReadDeadline(time.Now().Add(10 * time.Minute))
	_ = controller.SetWriteDeadline(time.Now().Add(10 * time.Minute))
	dir := filepath.Join(r.s.directory, "boot-pills")
	if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		r.respondError(w, conflict("pill_staging_failed", "boot pill staging unavailable"))
		return
	}
	if err := bootpill.PrivateDirectory(dir); err != nil {
		r.respondError(w, conflict("pill_staging_failed", "unsafe boot pill storage"))
		return
	}
	f, err := os.Open(dir)
	if err != nil {
		r.respondError(w, conflict("pill_staging_failed", "boot pill storage unavailable"))
		return
	}
	entries, err := f.ReadDir(65)
	f.Close()
	if err != nil && !errors.Is(err, io.EOF) || len(entries) > 64 {
		r.respondError(w, conflict("pill_staging_failed", "boot pill cache limit reached"))
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, protocol.MaxBootPillBytes)
	defer req.Body.Close()
	path, _, err := bootpill.Stage(req.Context(), dir, req.Body, &meta)
	if err != nil {
		r.respondError(w, conflict("pill_staging_failed", "pill transfer did not match its declared content"))
		return
	}
	defer os.Remove(path)
	destination := bootpill.CachePath(r.s.directory, meta)
	if err := os.Link(path, destination); err != nil {
		if !errors.Is(err, os.ErrExist) || bootpill.Verify(req.Context(), destination, meta) != nil {
			r.respondError(w, conflict("pill_staging_failed", "could not publish or verify the staged pill"))
			return
		}
	}
	if d, e := os.Open(dir); e == nil {
		err = d.Sync()
		d.Close()
	} else {
		err = e
	}
	if err != nil {
		r.respondError(w, conflict("pill_staging_failed", "could not sync staged pill"))
		return
	}
	respond(w, 201, meta)
}
