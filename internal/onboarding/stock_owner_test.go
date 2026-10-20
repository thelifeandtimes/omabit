package onboarding

import (
	"context"
	"os"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

func TestStockOwnerGrantsReachTheExistingAuthorizationAndProgressFlow(t *testing.T) {
	d, dir, config, runner, calls := setupFixture(t)
	p := ownerhost.Profile{Version: 1, Name: "desktop", UID: 1000, GID: 1000, Home: "/home/desktop", Piers: ownerhost.DefaultPiers}
	passwd := "root:x:0:0::/root:/bin/sh\ndesktop:x:1000:1000::/home/desktop:/bin/sh\n"
	uids, gids := hostsetup.LocalIDs(passwd, "root:x:0:\ndesktop:x:1000:\n")
	nss := "passwd: files systemd\ngroup: files [SUCCESS=merge] systemd\n"
	d.inspect = func() hostsetup.Report {
		r := hostsetup.Report{Compatibility: &hostcompat.Report{Compatibility: "unknown", Provenance: "unverified"}, Checks: []hostsetup.Check{
			{ID: "platform", Status: "pass"}, {ID: "docker-version", Status: "pass"}, {ID: "owner-account", Status: "pass"}, {ID: "account", Status: "missing"},
		}}
		for kind, ids := range map[string][]uint64{"subuid": uids, "subgid": gids} {
			status, detail := hostsetup.ValidateOwnerSubIDAuthority(nss, nil, os.ErrNotExist, hostsetup.LocalOwnerBinding(passwd, p), p.Name, "desktop:100000:65536\n", "1000", kind, ids)
			r.Checks = append(r.Checks, hostsetup.Check{ID: kind, Status: status, Detail: detail})
		}
		return r
	}
	before, err := inspect(context.Background(), config, dir, d)
	if err != nil || !before.CanSetup || before.Ready || before.Compatibility.Engine != nil {
		t.Fatal("fresh metadata did not permit setup", before, err)
	}
	var steps []string
	authorize := d.elevate
	d.elevate = func(ctx context.Context, exe, bundle, public string, emit func(Event)) (hostinstall.HostResult, error) {
		if len(steps) == 0 || steps[len(steps)-1] != "authorize" {
			t.Fatal("host installation preceded its authorization progress event", steps)
		}
		return authorize(ctx, exe, bundle, public, emit)
	}
	after, err := setup(context.Background(), config, dir, runner, d, func(event Event) { steps = append(steps, event.Step) })
	if err != nil || !after.Ready || *calls != 1 {
		t.Fatal("stock owner did not complete setup", after, err)
	}
	if want := []string{"check", "key", "download", "authorize", "connect", "ready"}; !reflect.DeepEqual(steps, want) {
		t.Fatal("setup checklist lost its existing progression", steps)
	}
}
