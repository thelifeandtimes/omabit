package managed

import (
	"context"
	"net/http"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type runtimeProfilesBackend interface {
	VerifyRuntimeProfile(context.Context, string) error
	InspectExternalPierWithProfile(context.Context, string, string) (pierstore.ExternalSource, error)
	InspectPierIdentityWithProfile(context.Context, string, string) (protocol.PierIdentity, error)
}

func (realBackend) VerifyRuntimeProfile(ctx context.Context, id string) error {
	return runtimequalify.VerifyRuntimeProfile(ctx, id)
}
func (realBackend) InspectExternalPierWithProfile(ctx context.Context, path, id string) (pierstore.ExternalSource, error) {
	return runtimequalify.InspectExternalPierWithProfile(ctx, path, id)
}
func (realBackend) InspectPierIdentityWithProfile(ctx context.Context, path, id string) (protocol.PierIdentity, error) {
	return runtimequalify.InspectPierIdentityWithProfile(ctx, path, id)
}

func (r *Server) verifyRuntimeProfile(ctx context.Context, id string) error {
	if err := validateRuntimeProfile(id); err != nil {
		return err
	}
	if id == "" {
		return nil
	}
	b, ok := r.b.(runtimeProfilesBackend)
	if !ok {
		return conflict("runtime_profiles_unavailable", "update the runner for runtime-profiles-v1")
	}
	if err := b.VerifyRuntimeProfile(ctx, id); err != nil {
		return conflict("runtime_artifact_unavailable", err.Error())
	}
	return nil
}
func (r *Server) inspectExternalForProfile(ctx context.Context, b storageActionsBackend, path, id string) (pierstore.ExternalSource, error) {
	if err := r.verifyRuntimeProfile(ctx, id); err != nil {
		return pierstore.ExternalSource{}, err
	}
	if id == "" {
		return b.InspectExternalPier(ctx, path)
	}
	return r.b.(runtimeProfilesBackend).InspectExternalPierWithProfile(ctx, path, id)
}
func (r *Server) inspectIdentityForProfile(ctx context.Context, path, id string) (protocol.PierIdentity, error) {
	if err := r.verifyRuntimeProfile(ctx, id); err != nil {
		return protocol.PierIdentity{}, err
	}
	if id == "" {
		return r.b.(guidedImportBackend).InspectPierIdentity(ctx, path)
	}
	return r.b.(runtimeProfilesBackend).InspectPierIdentityWithProfile(ctx, path, id)
}
func (r *Server) serveRuntimeProfile(w http.ResponseWriter, req *http.Request, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "runtime artifact preflight requires operator access"}})
		return
	}
	if req.Method != http.MethodGet {
		r.respondError(w, bad("runtime artifact preflight requires GET"))
		return
	}
	id := strings.TrimPrefix(req.URL.Path, "/v1/runtime-profiles/")
	p, err := candidate.Resolve(id)
	if err != nil || id == "" {
		r.respondError(w, bad("select an immutable runtime profile ID"))
		return
	}
	if err = r.verifyRuntimeProfile(req.Context(), id); err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, protocol.RuntimeProfilePreflight{RunnerID: r.s.runnerID, Profile: p, Available: true})
}
