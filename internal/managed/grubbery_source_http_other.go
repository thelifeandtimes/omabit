//go:build !linux || !amd64

package managed

import "net/http"

func (r *Server) grubberyAppSourceAvailable() bool { return false }
func (r *Server) serveGrubberyAppSource(w http.ResponseWriter, req *http.Request, actor, role string) {
	w.WriteHeader(http.StatusNotImplemented)
}
