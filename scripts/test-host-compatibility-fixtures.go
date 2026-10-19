//go:build ignore

// Pure wire fixtures for JS/Qt tests. No host inspection, sockets, packages or
// services are accessed. Policy decisions and JSON tags come from shipped Go.
package main

import (
	"encoding/json"
	"os"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func main() {
	now := time.Now().UTC()
	fixtures := make(map[string]collection.Observation)
	for _, name := range []string{"compatible", "baseline", "metadata", "pending", "restarted", "unsupported", "uncertain", "unverified", "rootful", "cgroup", "data-root", "api", "stale", "older"} {
		const dataRoot = "/fixture/private/engine"
		r := hostcompat.Report{Platform: "linux/amd64", ObservedAt: now,
			Distribution:           hostcompat.Distribution{ID: "debian", VersionID: "13", Provider: "deb", Profile: "unqualified"},
			InstalledEngineVersion: "29.9.1", Guidance: []string{"Review Debian native package evidence."},
			Engine: &hostcompat.Engine{Version: "29.9.1", APIVersion: "1.54", MinAPIVersion: "1.44", Rootless: true, CgroupDriver: "systemd", CgroupVersion: "2", DataRoot: dataRoot},
		}
		for _, role := range []string{"engine", "cli", "containerd", "oci-runtime", "rootlesskit", "network", "uidmap", "gidmap"} {
			r.Packages = append(r.Packages, hostcompat.Package{Role: role, Name: "fixture-" + role, Version: "5:29.9.1-1~debian.13", UpstreamVersion: "29.9.1", Provenance: "verified", Executables: []string{"/usr/bin/fixture-" + role}})
		}
		switch name {
		case "baseline":
			r.InstalledEngineVersion, r.Engine.Version = "29.7.2", "29.7.2"
		case "metadata":
			r.Engine = nil
		case "pending":
			r.Engine.Version = "29.7.2"
		case "unsupported":
			r.InstalledEngineVersion, r.Engine.Version = "30.0.0", "30.0.0"
		case "uncertain":
			r.InstalledEngineVersion, r.Engine.Version = "unknown", "unknown"
		case "unverified":
			r.Packages[0].Provenance = "metadata-only"
		case "rootful":
			r.Engine.Rootless = false
		case "cgroup":
			r.Engine.CgroupVersion = "1"
		case "data-root":
			r.Engine.DataRoot = "/fixture/replaced/engine"
		case "api":
			r.Engine.MinAPIVersion = "1.54"
		case "stale":
			r.Engine.Rootless = false
			r.ObservedAt = now.Add(-6 * time.Minute)
		}
		r = hostcompat.Evaluate(r, dataRoot)
		bound := protocol.HostCompatibilityReport{SchemaVersion: protocol.HostCompatibilitySchema, ProtocolVersion: protocol.Version,
			RunnerID: "runner-fixture", Mode: "development-managed", Features: []string{protocol.HostCompatibilityFeature},
			Report: protocol.PublicHostCompatibility(r), ExpiresAt: r.ObservedAt.Add(5 * time.Second)}
		metadataAt := r.ObservedAt.Add(-10 * time.Second)
		bound.MetadataObservedAt = &metadataAt
		if !protocol.ValidHostCompatibility(bound) {
			panic("invalid wire fixture: " + name)
		}
		snapshot := &protocol.Snapshot{RunnerID: bound.RunnerID, Mode: bound.Mode, Operations: []protocol.Operation{}, Incidents: []protocol.Incident{},
			Instances: []protocol.Instance{{ID: "alpha", ConfigRevision: 1, AppliedRevision: 1, Config: protocol.Configuration{HTTPPort: 18080, AmesPort: 34343, Loom: 31},
				DesiredState: "running", ObservedState: "running", Health: "healthy", ObservedAt: &now}}}
		v := collection.Observation{Name: "fixture", RunnerID: bound.RunnerID, Mode: bound.Mode, Status: "available", AccessRole: "operator", Features: bound.Features,
			CheckedAt: now, Snapshot: snapshot, HostCompatibility: &bound, HostCompatibilityStatus: "available"}
		if name == "stale" {
			v.HostCompatibilityStatus = "stale"
		}
		if name == "older" {
			v.Features, v.HostCompatibility, v.HostCompatibilityStatus = nil, nil, "unsupported"
		}
		fixtures[name] = v
	}
	if err := json.NewEncoder(os.Stdout).Encode(fixtures); err != nil {
		panic(err)
	}
}
