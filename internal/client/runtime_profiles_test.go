package client

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
)

func TestRuntimeProfileCapabilitiesRefuseBeforeMutation(t *testing.T) {
	for _, fault := range []string{"old-runner", "wrong-binary", "wrong-image", "unknown-profile"} {
		t.Run(fault, func(t *testing.T) {
			p, _ := runtimeprofile.Resolve(runtimeprofile.JSONFixProfile)
			h := protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: "selected", Mode: "development-managed", Features: []string{protocol.RuntimeProfilesFeature}, RuntimeProfiles: []runtimeprofile.Profile{p}}
			switch fault {
			case "old-runner":
				h.Features = nil
			case "wrong-binary":
				h.RuntimeProfiles[0].BinarySHA256 = "other"
			case "wrong-image":
				h.RuntimeProfiles[0].ConfigDigest = "other"
			}
			var mutations atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/hello" {
					json.NewEncoder(w).Encode(h)
					return
				}
				mutations.Add(1)
				http.Error(w, "unexpected mutation", 500)
			})
			id := p.ID
			if fault == "unknown-profile" {
				id = "latest"
			}
			req := protocol.CreateFakeRequest{ID: "new", Config: protocol.Configuration{RuntimeProfile: id}}
			if _, err := c.CreateFake(t.Context(), req); err == nil {
				t.Fatal("fake creation ignored capability")
			}
			if _, err := c.CreateComet(t.Context(), protocol.CreateCometRequest{CreateFakeRequest: req}); err == nil {
				t.Fatal("comet creation ignored capability")
			}
			if _, err := c.EnrollPier(t.Context(), protocol.EnrollPierRequest{RuntimeProfile: id}); err == nil {
				t.Fatal("enrollment ignored capability")
			}
			if _, err := c.PreviewPierImport(t.Context(), "/synthetic/pier", id); err == nil {
				t.Fatal("preview ignored capability")
			}
			if _, err := c.Configure(t.Context(), "existing", protocol.ConfigureRequest{Config: req.Config}); err == nil {
				t.Fatal("configuration ignored capability")
			}
			if mutations.Load() != 0 {
				t.Fatal("unsafe request dispatched")
			}
		})
	}
}

func TestRuntimeProfilePreflightBindsRunnerAndExactArtifact(t *testing.T) {
	for _, fault := range []string{"none", "other-runner", "other-profile", "unavailable"} {
		t.Run(fault, func(t *testing.T) {
			p, _ := runtimeprofile.Resolve(runtimeprofile.JSONFixProfile)
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/hello" {
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: "selected", Mode: "development-managed", Features: []string{protocol.RuntimeProfilesFeature}, RuntimeProfiles: []runtimeprofile.Profile{p}})
					return
				}
				if r.URL.Path != "/v1/runtime-profiles/"+p.ID || r.Method != http.MethodGet {
					t.Error("unexpected request", r.Method, r.URL.Path)
				}
				w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
				v := protocol.RuntimeProfilePreflight{RunnerID: "selected", Profile: p, Available: true}
				switch fault {
				case "other-runner":
					v.RunnerID = "other"
				case "other-profile":
					v.Profile.BinarySHA256 = "other"
				case "unavailable":
					v.Available = false
				}
				json.NewEncoder(w).Encode(v)
			})
			_, err := c.RuntimeProfilePreflight(t.Context(), p.ID)
			if (err == nil) != (fault == "none") {
				t.Fatal("preflight binding", fault, err)
			}
		})
	}
}

func TestRuntimeProfileHandoffRefusesBeforeSourceFence(t *testing.T) {
	for _, fault := range []string{"old-source", "old-destination", "source-catalogue", "destination-catalogue", "missing-image"} {
		t.Run(fault, func(t *testing.T) {
			p, _ := runtimeprofile.Resolve(runtimeprofile.JSONFixProfile)
			var mutations atomic.Int32
			endpoint := func(id string) *Client {
				return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
					if r.Method != http.MethodGet {
						mutations.Add(1)
						http.Error(w, "unexpected fence", 500)
						return
					}
					switch r.URL.Path {
					case "/hello":
						features := []string{"host-handoff", protocol.RuntimeSettingsHandoffFeature, protocol.ContainerResourcesFeature}
						if fault != "old-"+id {
							features = append(features, protocol.RuntimeProfilesFeature)
						}
						advertised := p
						if fault == id+"-catalogue" {
							advertised.BinarySHA256 = "changed"
						}
						json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: id, AccessRole: "operator", Features: features, RuntimeProfiles: []runtimeprofile.Profile{advertised}})
					case "/v1/snapshot":
						json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: id, Instances: []protocol.Instance{{ID: "fixture", ConfigRevision: 1, Config: protocol.Configuration{Loom: 34, RuntimeProfile: p.ID}}}})
					case "/v1/runtime-profiles/" + p.ID:
						json.NewEncoder(w).Encode(protocol.RuntimeProfilePreflight{RunnerID: id, Profile: p, Available: fault != "missing-image"})
					default:
						t.Error("unexpected path", r.URL.Path)
					}
				})
			}
			_, err := MoveStopped(t.Context(), endpoint("source"), endpoint("destination"), "source", "fixture", protocol.PrepareHandoffRequest{DestinationRunner: "destination", DestinationInstance: "arrived", ExpectedRevision: 1, RequestID: "move-pinned"}, 0, 0)
			if err == nil || mutations.Load() != 0 {
				t.Fatal("unavailable/mismatched destination fenced source", fault, err, mutations.Load())
			}
		})
	}
}

func TestLocalRuntimePreflightUsesSelectedRunnerCatalogue(t *testing.T) {
	p := runtimeprofile.Profile{Format: 1, Kind: "local", Platform: runtimeprofile.Platform, VereVersion: "local-64", Image: "omabit/vere@sha256:" + strings.Repeat("a", 64), ManifestDigest: "sha256:" + strings.Repeat("a", 64), ConfigDigest: "sha256:" + strings.Repeat("b", 64), BinarySHA256: strings.Repeat("c", 64), ArchiveSHA256: strings.Repeat("d", 64), ArchiveBytes: 1234, MaxLoom: 38}
	p.ID = runtimeprofile.LocalID(p)
	for _, tamper := range []bool{false, true} {
		t.Run(strconv.FormatBool(tamper), func(t *testing.T) {
			advertised := p
			if tamper {
				advertised.MaxLoom++
			}
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
				if r.URL.Path == "/hello" {
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: "selected", Features: []string{protocol.RuntimeProfilesFeature}, RuntimeProfiles: []runtimeprofile.Profile{advertised}})
					return
				}
				if r.Method != http.MethodGet || r.URL.Path != "/v1/runtime-profiles/"+p.ID {
					t.Error("unexpected request", r.Method, r.URL.Path)
				}
				json.NewEncoder(w).Encode(protocol.RuntimeProfilePreflight{RunnerID: "selected", Available: true, Profile: p})
			})
			got, err := c.RuntimeProfilePreflight(t.Context(), p.ID)
			if (err == nil) == tamper || !tamper && got.Profile != p {
				t.Fatal("local profile binding", got, err)
			}
		})
	}
}
