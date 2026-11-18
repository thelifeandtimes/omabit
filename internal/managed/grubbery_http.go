package managed

import (
	"errors"
	"net/http"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Saved lookup deliberately runs outside the lifecycle mutex. Load validates a
// single SQLite snapshot, including original plan/event ancestry. It neither
// opens staging nor constructs a native runtime, even during recovery.
func (r *Server) serveGrubberyAppReceipt(w http.ResponseWriter, req *http.Request, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "app receipts require operator access"}})
		return
	}
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(req.URL.Path, protocol.GrubberyAppOperationsPath+"/")
	if !protocol.ValidDeskOperationID(id) {
		r.respondError(w, bad("invalid app operation ID"))
		return
	}
	record, err := (grubberyJournal{r.s}).Load(req.Context(), id)
	if errors.Is(err, grubbery.ErrMissing) {
		r.respondError(w, &fault{404, protocol.APIError{Code: "app_operation_not_found", Message: "app operation not found"}})
		return
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	out := protocol.GrubberyAppOperation{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Record: record}
	if err = out.Validate(); err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, out)
}
