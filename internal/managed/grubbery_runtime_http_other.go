//go:build !linux || !amd64

package managed

import "net/http"

func (r *Server) grubberyRuntimeRecoveryAvailable() bool { return false }

func (r *Server) serveGrubberyRuntimeRecovery(w http.ResponseWriter, req *http.Request, actor, role string) {
	w.WriteHeader(http.StatusNotImplemented)
}
