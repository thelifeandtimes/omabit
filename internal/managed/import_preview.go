package managed

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type guidedImportBackend interface {
	InspectPierIdentity(context.Context, string) (protocol.PierIdentity, error)
	ImportDestinationAvailable(string, string) (bool, error)
}

func (realBackend) InspectPierIdentity(ctx context.Context, path string) (protocol.PierIdentity, error) {
	return runtimequalify.InspectPierIdentity(ctx, path)
}
func (realBackend) ImportDestinationAvailable(path, source string) (bool, error) {
	return runtimequalify.ImportDestinationAvailable(path, source)
}

func (r *Server) previewImport(ctx context.Context, b storageActionsBackend, source pierstore.ExternalSource, rootID string) (protocol.PierImportPreview, error) {
	var out protocol.PierImportPreview
	g, ok := r.b.(guidedImportBackend)
	if !ok {
		return out, bad("update the runner for automatic pier identification")
	}
	roots, err := b.StorageRoots()
	if err != nil {
		return out, err
	}
	var root string
	for _, location := range roots {
		if location.ID == rootID {
			root = location.Path
		}
	}
	if root == "" {
		return out, bad("choose a configured storage location")
	}
	identity, err := g.InspectPierIdentity(ctx, source.Path)
	if err != nil {
		return out, conflict("identity_unavailable", err.Error())
	}
	if !protocol.ValidFakeShip(identity.Ship) || (identity.Network != "livenet" && identity.Network != "fakenet") || (identity.Network == "fakenet" && identity.Kind != "fake") || (identity.Network == "livenet" && identity.Kind != protocol.ShipClass(identity.Ship)) {
		return out, bad("pier identity/network metadata is unsupported")
	}
	// Bind the identification to the same filesystem review, not a name guess.
	again, err := b.InspectExternalPier(ctx, source.Path)
	if err != nil || again != source {
		return out, conflict("source_changed", "source changed during identification; inspect again")
	}
	all, err := instances(r.s.db)
	if err != nil {
		return out, err
	}
	names := map[string]bool{}
	for _, v := range all {
		names[v.Public.ID] = true
		if identity.Network == "livenet" && v.Public.Mode != "fake" && v.Public.Ship == identity.Ship {
			return out, conflict("ship_already_enrolled", "this live ship is already enrolled")
		}
	}
	leaf := strings.TrimPrefix(identity.Ship, "~")
	prefix := "fakeships"
	if identity.Network == "livenet" {
		prefix = "livenet/" + map[string]string{"galaxy": "galaxies", "star": "stars", "planet": "planets", "moon": "moons", "comet": "comets"}[identity.Kind]
	}
	base := leaf
	// Retain an existing instance name/group when the source is already laid out
	// in the selected root, while using the detected @p as the new pier leaf.
	if rel, err := filepath.Rel(root, source.Path); err == nil && pierstore.ValidPath(rel) && pierstore.Kind(rel) == identity.Kind {
		base = filepath.Base(filepath.Dir(rel))
		prefix = filepath.Dir(filepath.Dir(rel))
	}
	for n := 0; n < 1024; n++ {
		name := base
		if n > 0 {
			suffix := fmt.Sprintf("-%d", n+1)
			name = base[:min(len(base), 64-len(suffix))] + suffix
		}
		if !protocol.ValidID(name) || names[name] {
			continue
		}
		destination := pierstore.Qualify(rootID, filepath.Join(prefix, name, leaf))
		if err := reservePierPath(r.s.db, name, destination); err != nil {
			var fault *fault
			if errors.As(err, &fault) {
				continue
			}
			return out, err
		}
		available, err := g.ImportDestinationAvailable(destination, source.Path)
		if err != nil {
			return out, err
		}
		if !available {
			continue
		}
		port, err := r.s.suggestHTTPPort(ctx)
		if err != nil {
			return out, err
		}
		return protocol.PierImportPreview{RunnerID: r.s.runnerID, Source: source, PierIdentity: identity, StorageRoot: rootID, InstanceID: name, Destination: destination, HTTPPort: port, InPlace: source.Path == filepath.Join(root, pierstore.Relative(destination))}, nil
	}
	return out, conflict("destination_unavailable", "no unused instance directory is available")
}
