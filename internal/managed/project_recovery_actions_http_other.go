//go:build !linux || !amd64

package managed

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
)

func (r *Server) projectRecoveryActionFeatures() []string { return nil }
func (r *Server) serveProjectRecoveryAction(w http.ResponseWriter, q *http.Request, actor, role, project, parent, kind string) {
	r.respondError(w, &fault{409, protocol.APIError{Code: "project_recovery_unsupported", Message: "recovery actions require a qualified Linux/AMD64 runner"}})
}
