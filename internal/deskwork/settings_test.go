package deskwork

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func TestDevelopmentSettingsAreDescriptiveAndDefaultWithoutGrants(t *testing.T) {
	for _, target := range []apppolicy.Target{{InstanceID: "publisher"}, {InstanceID: "publisher", Desk: "example"}, {InstanceID: "member", Desk: "example", ProjectID: "network", Generation: 2, Creator: "replace-2"}} {
		v := DefaultSetting(target)
		if v.Validate() != nil || v.Revision != 0 || v.Actor != "" {
			t.Fatal(v)
		}
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), "allow") || strings.Contains(string(b), "policy") {
			t.Fatal("metadata includes permission")
		}
	}
	if (DefaultSetting(apppolicy.Target{InstanceID: "member", ProjectID: "network"})).Validate() == nil {
		t.Fatal("unresolved generation accepted")
	}
}
func TestDevelopmentSettingsRejectMalformedWorkflowAndHistory(t *testing.T) {
	base := DefaultSetting(apppolicy.Target{InstanceID: "publisher", Desk: "example"})
	for _, change := range []func(*Setting){
		func(s *Setting) { s.Desk.SourceName = "../base" }, func(s *Setting) { s.Desk.SourceName = ".urb" },
		func(s *Setting) { s.Desk.Workflow = "urgit" }, func(s *Setting) { s.Desk.Publication = "automatic" },
		func(s *Setting) { s.Desk.Instructions = strings.Repeat("x", 2049) }, func(s *Setting) { s.Desk.Checks = []string{""} },
		func(s *Setting) { s.Desk.Checks = nil }, func(s *Setting) { s.Ship = &ShipConfig{Purpose: "standard"} },
		func(s *Setting) { s.Revision = 1 }, func(s *Setting) { s.Actor = "unexpected" }, func(s *Setting) { s.Desk.Branch = "master\x00" },
	} {
		v := base
		config := *base.Desk
		v.Desk = &config
		change(&v)
		if v.Validate() == nil {
			t.Fatal("invalid setting accepted", v)
		}
	}
	base.Desk.Workflow = "urgit"
	base.Desk.SourceName = ""
	base.Desk.Repository = "example"
	base.Desk.Branch = "master"
	base.Desk.Checks = []string{"Native protected-branch checks"}
	base.Revision = 1
	base.Actor = "desktop"
	base.UpdatedAt = time.Now().UTC()
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
}
