//go:build linux && amd64

package runtimequalify

// Adapted from wundrellama's PR #2: preserve offline verification, publication
// and containment checks, including RC.8's ship-named pier leaves.
import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func TestOwnedEnrollmentPublishesAmesAfterIdentityVerification(t *testing.T) {
	for _, tc := range []struct {
		name, ship, tier string
		mismatch         bool
	}{
		{"moon", "~dibpen-fogweg-dozzod-sarlev", "moons", false},
		{"planet", "~sampel-palnet", "planets", false},
		{"comet", fixtureComet, "comets", false},
		{"identity-mismatch", "~sampel-palnet", "planets", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, state, persist := managedCometFixture(t)
			state.Boot = &ManagedBoot{}
			state.AmesPort = 34567
			state.NativeShip = tc.ship
			state.Comet.Ship = tc.ship
			rel := "livenet/" + tc.tier + "/imported/" + strings.TrimPrefix(tc.ship, "~")
			p, err := pierstore.Parents(f.d.home, rel)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(filepath.Join(p, ".urb/chk"), 0700); err != nil {
				t.Fatal(err)
			}
			writePierEpochPath(t, p, "0i0", false)
			identity, err := pierstore.Identify(p)
			if err != nil {
				t.Fatal(err)
			}
			mount, err := pierstore.IdentifyMount(filepath.Dir(p))
			if err != nil {
				t.Fatal(err)
			}
			state.Storage = &pierstore.Location{Path: rel, Identity: identity, MountIdentity: mount}
			state.Adoption = &ManagedAdoption{Fingerprint: pierstore.Fingerprint(identity), Uncertain: true}
			state, obs, err := applyManaged(context.Background(), "enroll-pier", state, persist, f.d)
			if err != nil || obs.State != "stopped" || state.MaintenanceQualified() {
				t.Fatal(obs, err)
			}
			if _, _, err = applyManaged(context.Background(), "pack", state, persist, f.d); err == nil {
				t.Fatal("maintenance preceded identity verification")
			}
			f.d.hostIdentity = func(context.Context, string) (string, error) {
				if tc.mismatch {
					return "~marzod", nil
				}
				return tc.ship, nil
			}
			state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
			if tc.mismatch {
				if err == nil || state.Adoption.Verified {
					t.Fatal("mismatched identity accepted")
				}
				for _, call := range f.calls {
					joined := " " + strings.Join(call, " ") + " "
					if strings.Contains(joined, "/udp") || strings.Contains(joined, "comet-resume") {
						t.Fatal("networked resume before identity verification", call)
					}
				}
				return
			}
			if err != nil || obs.Health != "healthy" || !state.Adoption.Verified {
				t.Fatal(obs, err)
			}
			verified, resumed := false, false
			for _, call := range f.calls {
				if len(call) < 2 || call[0] != "container" || call[1] != "create" {
					continue
				}
				joined := " " + strings.Join(call, " ") + " "
				if strings.Contains(joined, " -c ") || strings.Contains(joined, " -w ") || strings.Contains(joined, " -k ") {
					t.Fatal("import used creation/key flags")
				}
				if strings.Contains(joined, "comet-verify") {
					verified = strings.Contains(joined, " -L ")
					if strings.Contains(joined, "/udp") {
						t.Fatal("verification published UDP")
					}
				}
				if strings.Contains(joined, "comet-resume") {
					resumed = !strings.Contains(joined, " -L ") && verified
					if !strings.Contains(joined, " --ames-port 34567 ") || !strings.Contains(joined, "0.0.0.0:34567:34567/udp") {
						t.Fatal("missing Ames mapping", call)
					}
				}
			}
			if !verified || !resumed || f.downloads != 0 {
				t.Fatal("handover ordering or bootstrap download", verified, resumed, f.downloads)
			}
			if _, obs, err = observeManaged(context.Background(), state, f.d); err != nil || obs.Health != "healthy" {
				t.Fatal(obs, err)
			}
			f.editInspect = func(_ string, doc map[string]any) {
				mapAt(mapAt(doc, "HostConfig"), "PortBindings")["34567/udp"] = []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "34568"}}
			}
			if _, _, err = observeManaged(context.Background(), state, f.d); err == nil {
				t.Fatal("wrong Ames mapping accepted")
			}
		})
	}
}
