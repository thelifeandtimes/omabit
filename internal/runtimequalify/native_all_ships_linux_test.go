//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestNativeAllShipsLegacyCometLaunchNotRelabeled(t *testing.T) {
	f, s, p := managedCometFixture(t)
	s, _, err := applyManaged(context.Background(), "start", s, p, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for n := range s.Runtime.Containers {
		s.Runtime.Containers[n].Loom = 0
		s.Runtime.Containers[n].NativeControl = false
	}
	s.Loom, s.NativeControl = 34, true
	if _, obs, err := observeManaged(context.Background(), s, f.d); err != nil || obs.Health != "healthy" {
		t.Fatal("desired opt-in relabeled legacy launch", obs, err)
	}
	s, _, err = applyManaged(context.Background(), "stop", s, p, f.d)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err = applyManaged(context.Background(), "start", s, p, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range s.Runtime.Containers {
		if isCometKind(c.Kind) && (!c.NativeControl || c.Loom != 34) {
			t.Fatal("replacement lost new settings")
		}
	}
}

func TestNativeAllShipsLaunchContract(t *testing.T) {
	for _, ship := range []string{"", "~zod", "~marzod", "~sampel-palnet", "~dibpen-fogweg-dozzod-sarlev"} {
		for _, native := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", ship, native), func(t *testing.T) {
				f, s, p := managedCometFixture(t)
				s.Loom, s.NativeControl = 34, native
				if ship != "" {
					s.NativeShip, s.Comet.Ship, s.AmesPort, s.Boot = ship, ship, 34567, &ManagedBoot{}
					tier := map[string]string{"galaxy": "galaxies", "star": "stars", "planet": "planets", "moon": "moons"}[protocol.ShipClass(ship)]
					if tier == "" {
						t.Fatal("invalid fixture identity", ship)
					}
					s.Storage = &pierstore.Location{Path: "livenet/" + tier + "/fixture/pier"}
					base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/keys")
					if err := os.Mkdir(base, 0700); err != nil {
						t.Fatal(err)
					}
					key := "0w" + strings.Repeat("a", 128) // synthetic, never a real key
					s.NativeKeyDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
					if err := os.WriteFile(filepath.Join(base, s.ID+".key"), []byte(key), 0600); err != nil {
						t.Fatal(err)
					}
					f.d.hostIdentity = func(context.Context, string) (string, error) { return ship, nil }
				}
				q, err := managedSession(&s, p, f.d)
				if err != nil {
					t.Fatal("opt-in admission", err)
				}
				for _, kind := range []string{"comet-boot", "comet-resume", "comet-verify"} {
					want := []string{"-t", "--snap-time", "60"}

					if !native {
						want = append(want, "--no-conn")
					}
					want = append(want, "--no-dock", "--loom", "34", "-b", "0.0.0.0", "--http-port", "8080")
					port := "34343"
					if ship != "" {
						port = "34567"
					}
					want = append(want, "--ames-port", port)
					if kind == "comet-verify" {
						want = append(want, "-L")
					}
					if kind == "comet-boot" {
						if ship == "" {
							want = append(want, "-B", "/artifacts/boot.pill", "-c")
						} else {
							want = append(want, "-w", ship, "-k", "/artifacts/boot.key", "-c")
						}
					}
					want = append(want, "/urbit/pier")
					entry := "/bin/urbit"
					if native {
						entry = "/bin/sh"
						want = append([]string{"-c", `umask 077; exec /bin/urbit "$@"`, "urbit-native-control"}, want...)
					}
					gotEntry, got := q.command(kind)
					if gotEntry != entry || !slices.Equal(got, want) {
						t.Fatalf("%s command %s %q; want %s %q", kind, gotEntry, got, entry, want)
					}
				}
				save := func(v ManagedState) error {
					for _, c := range v.Runtime.Containers {
						if isCometKind(c.Kind) && c.NativeControl != native {
							t.Fatal("launch intent lost native setting", c)
						}
					}
					return p(v)
				}
				for _, action := range []string{"start", "stop", "start"} {
					s, _, err = applyManaged(context.Background(), action, s, save, f.d)
					if err != nil {
						t.Fatal(action, err)
					}
				}
				if _, obs, err := observeManaged(context.Background(), s, f.d); err != nil || obs.Health != "healthy" {
					t.Fatal("observe", obs, err)
				}
				q, err = managedSession(&s, p, f.d)
				if err != nil {
					t.Fatal(err)
				}
				for _, kind := range []string{"comet-pack", "comet-meld", "comet-roll", "probe"} {
					entry, args := q.command(kind)
					if entry == "/bin/sh" || slices.Contains(args, "urbit-native-control") {
						t.Fatal("native leaked into utility", kind)
					}
				}
			})
		}
	}
}
