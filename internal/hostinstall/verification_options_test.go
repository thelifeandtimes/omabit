package hostinstall

import (
	"context"
	"errors"
	"os"
	"os/user"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func TestAuthorizedHostSetupFetchFailurePrecedesCommandsAndServiceChanges(t *testing.T) {
	for _, deployment := range []string{"owner", "dedicated"} {
		for _, fail := range []string{"download-error", "unverified-result", "unsupported-result"} {
			t.Run(deployment+"/"+fail, func(t *testing.T) {
				f, p, probe := ownerFixture(t)
				f.i.inspectCompatibility = func() hostcompat.Report {
					return hostcompat.Report{InstalledEngineVersion: "29.9.1", Provenance: "metadata-only"}
				}
				fetches := 0
				f.i.verifyCompatibility = func(context.Context) (hostcompat.Report, error) {
					t.Fatal("authorized setup used offline-only verifier")
					return hostcompat.Report{}, nil
				}
				f.i.verifyCompatibilityWithOptions = func(ctx context.Context, options hostcompat.VerificationOptions) (hostcompat.Report, error) {
					fetches++
					if ctx.Err() != nil || !options.FetchMissing || f.i.mutated || len(f.calls) != 0 {
						t.Fatal("download authorization/order changed")
					}
					if fail == "download-error" {
						return hostcompat.Report{}, errors.New("native download failed authentication")
					}
					if fail == "unsupported-result" {
						return hostcompat.Report{InstalledEngineVersion: "30.0.0", Provenance: "verified"}, nil
					}
					return hostcompat.Report{InstalledEngineVersion: "29.9.1", Provenance: "metadata-only"}, nil
				}
				before := tree(t, f.i.root)
				var err error
				if deployment == "owner" {
					_, err = f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
				} else {
					_, err = f.i.installHost(context.Background(), "/bundle", hostPublic(1), []byte("runner"), probe, nil)
				}
				if err == nil || fetches != 1 || f.i.mutated || len(f.calls) != 0 || !reflect.DeepEqual(before, tree(t, f.i.root)) {
					t.Fatal("unverified download crossed host execution boundary", err)
				}
			})
		}
	}
}

func TestRepairVerificationOptionsAreExplicitAndKeepGuardOnlyContract(t *testing.T) {
	for _, mode := range []string{"default-offline", "failed-offline", "fetch", "failed-fetch", "unauthorized-fetch"} {
		t.Run(mode, func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			service, err := rootless.ServiceFiles()
			if err != nil {
				t.Fatal(err)
			}
			for name, data := range service {
				f.files[name] = data
			}
			launcher, err := os.ReadFile("../rootless/assets/upstream/dockerd-rootless.sh")
			if err != nil {
				t.Fatal(err)
			}
			f.files["dockerd-rootless.sh"] = launcher
			if _, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil); err != nil {
				t.Fatal(err)
			}
			f.i.lookupUser = func(name string) (*user.User, error) {
				if name != p.Name {
					t.Fatal(name)
				}
				return &user.User{Username: p.Name, Uid: "942", Gid: "942", HomeDir: p.Home}, nil
			}
			f.calls, f.i.mutated = nil, false
			f.i.inspectCompatibility = func() hostcompat.Report {
				return hostcompat.Report{InstalledEngineVersion: "29.9.1", Provenance: "metadata-only"}
			}
			offline, fetched := 0, 0
			verified := hostcompat.Report{InstalledEngineVersion: "29.9.1", Provenance: "verified"}
			f.i.verifyCompatibility = func(context.Context) (hostcompat.Report, error) {
				offline++
				if mode == "failed-offline" {
					return hostcompat.Report{}, errors.New("offline evidence unavailable")
				}
				return verified, nil
			}
			f.i.verifyCompatibilityWithOptions = func(ctx context.Context, options hostcompat.VerificationOptions) (hostcompat.Report, error) {
				fetched++
				if !options.FetchMissing || ctx.Err() != nil || f.i.mutated || len(f.calls) != 0 {
					t.Fatal("fetch authorization/order changed")
				}
				if mode == "failed-fetch" {
					return hostcompat.Report{}, errors.New("missing native authentication evidence")
				}
				return verified, nil
			}
			if mode == "unauthorized-fetch" {
				f.i.euid = 1000
			}
			before := tree(t, f.i.root)
			var result GuardRepairResult
			if mode == "default-offline" || mode == "failed-offline" {
				result, err = f.i.repairEngineGuard(context.Background())
			} else {
				result, err = f.i.repairEngineGuardWithOptions(context.Background(), VerifyOptions{FetchMissing: true})
			}
			failed := mode == "failed-offline" || mode == "failed-fetch" || mode == "unauthorized-fetch"
			if (err != nil) != failed || result.Partial || result.Updated || result.Reloaded {
				t.Fatal(result, err)
			}
			wantOffline, wantFetch := 0, 1
			if mode == "default-offline" || mode == "failed-offline" {
				wantOffline, wantFetch = 1, 0
			}
			if mode == "unauthorized-fetch" {
				wantFetch = 0
			}
			if offline != wantOffline || fetched != wantFetch {
				t.Fatal("wrong verification path", offline, fetched)
			}
			if !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("verification options changed registered host state")
			}
			for _, call := range f.calls {
				if failed || call.path != "/usr/bin/systemctl" || len(call.args) == 0 || call.args[0] != "show" {
					t.Fatal("repair performed service or engine mutation", call)
				}
			}
		})
	}
}
