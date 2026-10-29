//go:build linux && amd64

package runtimequalify

import (
	"reflect"
	"slices"
	"strconv"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestProjectMemberCommandsKeepDistinctPortsAndResumeExistingPiers(t *testing.T) {
	plan, err := projectnetwork.Build(projectnetwork.Request{Label: "Fresh peers", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "zod-instance", Ship: "~zod", HTTPPort: 18281}, {InstanceID: "nec-instance", Ship: "~nec", HTTPPort: 18282}}}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	value := func(args []string, key string) string {
		t.Helper()
		i := slices.Index(args, key)
		if i < 0 || i+1 >= len(args) {
			t.Fatalf("missing command value %s", key)
		}
		return args[i+1]
	}
	ports := map[string]bool{}
	for _, m := range plan.Members {
		entry, boot, err := projectMemberCommand(plan, m.InstanceID, "boot")
		if err != nil || entry != "/bin/urbit" {
			t.Fatal(entry, err)
		}
		_, resume, err := projectMemberCommand(plan, m.InstanceID, "resume")
		if err != nil {
			t.Fatal(err)
		}
		if value(boot, "--http-port") != strconv.Itoa(int(m.InternalHTTPPort)) || value(resume, "--http-port") != value(boot, "--http-port") || ports[value(boot, "--http-port")] {
			t.Fatal("shared or changed member endpoint", boot, resume)
		}
		ports[value(boot, "--http-port")] = true
		if value(boot, "--http-port") == strconv.Itoa(int(m.HTTPPort)) || value(boot, "-b") != "0.0.0.0" {
			t.Fatal("host and project endpoint contracts confused", boot)
		}
		if value(boot, "-F") != m.Ship[1:] || !slices.Contains(boot, "-c") || value(boot, "-B") != "/artifacts/boot.pill" {
			t.Fatal("fresh boot lost identity or pill", boot)
		}
		for _, flag := range []string{"-c", "-F", "-B", "--ames-port"} {
			if slices.Contains(resume, flag) {
				t.Fatal("resume requests boot or fixed Ames behavior", resume)
			}
		}
		if boot[len(boot)-1] != "/urbit/"+m.Ship[1:] || resume[len(resume)-1] != boot[len(boot)-1] {
			t.Fatal("member pier leaf changed", boot, resume)
		}
		for _, flag := range []string{"-L", "--no-conn", "--no-dock"} {
			if !slices.Contains(boot, flag) || !slices.Contains(resume, flag) {
				t.Fatal("fake isolation/native control changed", flag)
			}
		}
		// Returned command mutation cannot rewrite another call or the reviewed plan.
		copy := slices.Clone(resume)
		resume[0] = "changed"
		_, again, err := projectMemberCommand(plan, m.InstanceID, "resume")
		if err != nil || !reflect.DeepEqual(copy, again) {
			t.Fatal("commands share mutable storage")
		}
	}
}
func TestProjectMemberCommandRefusesUnreviewedOrUnknownSelections(t *testing.T) {
	c, _, _ := resourceContractFixture(t)
	for _, test := range []struct {
		id, mode string
		alter    bool
	}{{"missing", "boot", false}, {"sender", "", false}, {"sender", "cleanup", false}, {"sender", "resume", true}} {
		plan := c.Selection.Plan
		plan.Members = slices.Clone(plan.Members)
		if test.alter {
			plan.Members[0].InternalHTTPPort++
		}
		entry, args, err := projectMemberCommand(plan, test.id, test.mode)
		if err == nil || entry != "" || args != nil {
			t.Fatal("unreviewed invocation returned", test, entry, args, err)
		}
	}
}
