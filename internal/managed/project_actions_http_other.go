//go:build !linux || !amd64

package managed

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
)

func (r *Server) projectActionFeatures() []string { return nil }
func (r *Server) serveProjectAction(w http.ResponseWriter, q *http.Request, actor, role, id string) {
	r.respondError(w, &fault{409, protocol.APIError{Code: "project_action_unsupported", Message: "project runtime actions require a qualified Linux/AMD64 runner"}})
}
