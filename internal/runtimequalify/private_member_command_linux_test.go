//go:build linux && amd64

package runtimequalify

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

// Generated from the independent runtime-qualified Python PublishedShip fixture.
// Provenance: {"docker_publication.py": "397cb708acf1821e8e6a37c13947993fca0ba3b6036f081e523e65626179c94f", "docker_runtime.py": "f5fe78cf2ae3b64151827f038a0a4db571cb42b54179b161c3222a8c67650ea4"}
const privateMemberQualifiedCommands = `{"~nec": {"boot": ["-t", "-L", "--no-conn", "--no-dock", "--loom", "31", "--snap-time", "60", "-b", "0.0.0.0", "--http-port", "18082", "-p", "31338", "-H", "galaxies.omabit.invalid", "-w", "nec", "-k", "/run/omabit-boot/key", "-e", "http://127.0.0.1:25432", "-B", "/run/omabit-boot/boot.pill", "-c", "/urbit/nec"], "resume": ["-t", "-L", "--no-conn", "--no-dock", "--loom", "31", "--snap-time", "60", "-b", "0.0.0.0", "--http-port", "18082", "-p", "31338", "/urbit/nec"]}, "~zod": {"boot": ["-t", "-L", "--no-conn", "--no-dock", "--loom", "31", "--snap-time", "60", "-b", "0.0.0.0", "--http-port", "18081", "-p", "31337", "-H", "galaxies.omabit.invalid", "-w", "zod", "-k", "/run/omabit-boot/key", "-e", "http://127.0.0.1:25432", "-B", "/run/omabit-boot/boot.pill", "-c", "/urbit/zod"], "resume": ["-t", "-L", "--no-conn", "--no-dock", "--loom", "31", "--snap-time", "60", "-b", "0.0.0.0", "--http-port", "18081", "-p", "31337", "/urbit/zod"]}}`

func privateMemberCommandPlan(t *testing.T) privatechain.Plan {
	t.Helper()
	raw, err := os.ReadFile("../privatechain/testdata/qualified-foundation.json")
	if err != nil {
		t.Fatal(err)
	}
	var f privatechain.Foundation
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	p, err := privatechain.Build(privatechain.Request{Label: "Private command qualification", Foundation: f, ConfirmSharedAdministration: true, Members: []privatechain.MemberRequest{{InstanceID: "zod-instance", Ship: "~zod", HTTPPort: 18381}, {InstanceID: "nec-instance", Ship: "~nec", HTTPPort: 18382}}}, privatechain.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPrivateMemberCommandsMatchQualifiedRuntime(t *testing.T) {
	p := privateMemberCommandPlan(t)
	var expected map[string]map[string][]string
	if err := json.Unmarshal([]byte(privateMemberQualifiedCommands), &expected); err != nil {
		t.Fatal(err)
	}
	for _, m := range p.Members {
		for _, mode := range []string{"boot", "resume"} {
			t.Run(m.Ship+"/"+mode, func(t *testing.T) {
				entry, args, err := privateMemberCommand(p, m.InstanceID, mode)
				if err != nil || entry != "/bin/urbit" || !reflect.DeepEqual(args, expected[m.Ship][mode]) {
					t.Fatal("command differs from independently qualified fixture", entry, args, err)
				}
				if mode == "resume" {
					for _, flag := range []string{"-F", "-w", "-H", "-e", "-k", "-B", "-c"} {
						if slices.Contains(args, flag) {
							t.Fatal("reopen retained a first-boot input", flag)
						}
					}
				}
				args[0] = "mutated"
				_, again, err := privateMemberCommand(p, m.InstanceID, mode)
				if err != nil || !reflect.DeepEqual(again, expected[m.Ship][mode]) {
					t.Fatal("returned command aliases another invocation")
				}
			})
		}
	}
}

func TestPrivateMemberCommandRefusesUnreviewedSelection(t *testing.T) {
	mutations := map[string]func(*privatechain.Plan){
		"mode":                  func(p *privatechain.Plan) { p.Mode = "fake-galaxies-v1" },
		"foundation":            func(p *privatechain.Plan) { p.Foundation.ChainID = 1 },
		"gateway":               func(p *privatechain.Plan) { p.GatewayPort++ },
		"public-bind":           func(p *privatechain.Plan) { p.BindAddress = "0.0.0.0" },
		"rpc-publication":       func(p *privatechain.Plan) { p.PublishRPC = true },
		"ames-publication":      func(p *privatechain.Plan) { p.PublishAmes = true },
		"shared-administration": func(p *privatechain.Plan) { p.SharedAdministration = false },
		"shutdown":              func(p *privatechain.Plan) { p.FirstStop = "sigterm" },
		"http-port":             func(p *privatechain.Plan) { p.Members[0].InternalHTTPPort++ },
		"ames-port":             func(p *privatechain.Plan) { p.Members[0].InternalAmesPort++ },
		"fresh-key":             func(p *privatechain.Plan) { p.Members[0].FreshKeyRequired = false },
		"ship-path":             func(p *privatechain.Plan) { p.Members[0].Ship = "../zod" },
		"duplicate-member":      func(p *privatechain.Plan) { p.Members[1] = p.Members[0] },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			p := privateMemberCommandPlan(t)
			mutate(&p)
			for _, mode := range []string{"boot", "resume"} {
				entry, args, err := privateMemberCommand(p, "zod-instance", mode)
				if err == nil || entry != "" || args != nil {
					t.Fatal("unreviewed plan returned an invocation")
				}
			}
		})
	}
	p := privateMemberCommandPlan(t)
	for _, selection := range [][2]string{{"missing", "boot"}, {"../zod-instance", "resume"}, {"zod-instance", ""}, {"zod-instance", "restart"}, {"zod-instance", "reset"}} {
		entry, args, err := privateMemberCommand(p, selection[0], selection[1])
		if err == nil || entry != "" || args != nil {
			t.Fatal("unknown selection returned an invocation", selection)
		}
	}
}
