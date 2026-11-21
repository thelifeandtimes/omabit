//go:build !linux || !amd64

package managed

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
)

func (r *Server) projectAllocationFeatures() []string { return nil }
func (r *Server) serveProjectAllocation(w http.ResponseWriter, req *http.Request, role string) {
	r.respondError(w, &fault{501, protocol.APIError{Code: "unsupported", Message: "project allocation adapter unavailable on this platform"}})
}
