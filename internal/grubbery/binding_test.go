package grubbery_test

import (
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func binding() grubbery.Binding {
	return grubbery.Binding{RunnerID: "runner", Mode: "development-managed-fake", Actor: "local-uid:1000", Target: apppolicy.Target{InstanceID: "member", Desk: "grubbery", ProjectID: "project", Creator: "owner", Generation: 1}, RuntimeID: strings.Repeat("a", 64), ConfigRevision: 1, PolicyRevision: 2, Continuity: strings.Repeat("b", 64), FrameworkReceiptID: strings.Repeat("c", 64), FrameworkBaseline: deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v123", Publication: "private"}, AppID: "state"}
}

func TestNamespaceSurvivesProcessAndPolicyChangesButNeverIdentityChanges(t *testing.T) {
	b := binding()
	n, err := b.Namespace()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(n.DeskRoot, "/apps/shell/desks/omabit-") || !strings.HasSuffix(n.DeskRoot, ".desk") || n.CodeRoot != n.DeskRoot+"/desk/code" || n.DataRoot != n.DeskRoot+"/desk/data" || strings.HasPrefix(n.SourceRoot, n.DeskRoot) {
		t.Fatal("code/release/state separation changed", n)
	}
	for _, change := range []string{"actor", "runtime", "config", "policy", "framework-receipt", "framework-revision"} {
		changed := b
		switch change {
		case "actor":
			changed.Actor = "ssh-key:new-owner"
		case "runtime":
			changed.RuntimeID = strings.Repeat("d", 64)
		case "config":
			changed.ConfigRevision++
		case "policy":
			changed.PolicyRevision++
		case "framework-receipt":
			changed.FrameworkReceiptID = strings.Repeat("e", 64)
		case "framework-revision":
			changed.FrameworkBaseline.Revision++
		}
		got, err := changed.Namespace()
		if err != nil || got != n {
			t.Fatal("ordinary update renamed app state", change, got, err)
		}
	}
	for _, change := range []string{"runner", "mode", "pier", "instance", "project", "creator", "generation", "app"} {
		changed := b
		switch change {
		case "runner":
			changed.RunnerID = "other"
		case "mode":
			changed.Mode = "simulation"
		case "pier":
			changed.Continuity = strings.Repeat("f", 64)
		case "instance":
			changed.Target.InstanceID = "other"
		case "project":
			changed.Target.ProjectID = "other"
		case "creator":
			changed.Target.Creator = "other"
		case "generation":
			changed.Target.Generation++
		case "app":
			changed.AppID = "other"
		}
		got, err := changed.Namespace()
		if err != nil || got == n {
			t.Fatal("different identity shared namespace", change, got, err)
		}
	}
	a, err := b.ReleaseRoot(strings.Repeat("1", 64))
	if err != nil || a != n.SourceRoot+"/"+strings.Repeat("1", 64)+"/code" {
		t.Fatal(a, err)
	}
	z, err := b.ReleaseRoot(strings.Repeat("2", 64))
	if err != nil || z == a {
		t.Fatal("original requests shared staging root")
	}
}

func TestBindingRefusesUnresolvedGenerationAndUnsafeAddressInputs(t *testing.T) {
	for _, change := range []string{"unresolved", "framework", "actor", "runner", "app", "pier", "baseline"} {
		b := binding()
		switch change {
		case "unresolved":
			b.Target.Generation = 0
			b.Target.Creator = ""
		case "framework":
			b.Target.Desk = "other"
		case "actor":
			b.Actor = "owner\nforged"
		case "runner":
			b.RunnerID = "../other"
		case "app":
			b.AppID = "../state"
		case "pier":
			b.Continuity = ""
		case "baseline":
			b.FrameworkBaseline = deskwork.Observation{}
		}
		if b.Validate() == nil {
			t.Fatal("invalid binding accepted", change)
		}
	}
	b := binding()
	for _, id := range []string{"", "../request", strings.Repeat("A", 64), strings.Repeat("1", 63)} {
		if _, err := b.ReleaseRoot(id); err == nil {
			t.Fatal("invalid original request ID accepted")
		}
	}
}
