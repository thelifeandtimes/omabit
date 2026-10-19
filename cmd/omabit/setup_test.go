package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/onboarding"
)

func TestSetupTextSeparatesInstalledRunningAndQualification(t *testing.T) {
	s := onboarding.Status{Ready: true, Checks: []hostsetup.Check{{ID: "docker-version", Status: "unknown", Detail: "Package evidence changed."}}, Compatibility: &hostcompat.Report{
		PolicyID: "docker-rootless-v1", Platform: "linux/amd64",
		Distribution:           hostcompat.Distribution{ID: "debian", VersionID: "13", Provider: "deb", Profile: "unqualified"},
		InstalledEngineVersion: "29.9.1", Engine: &hostcompat.Engine{Version: "29.7.2", APIVersion: "1.54", MinAPIVersion: "1.44"},
		Packages:      []hostcompat.Package{{Role: "engine", Name: "docker-ce", Version: "5:29.9.1-1~debian.13", UpstreamVersion: "29.9.1", Provenance: "metadata-only"}},
		Compatibility: "compatible", Qualification: "unqualified", Provenance: "unverified", RestartPending: true, CanObserve: true, CanStop: true,
		Checks:   []hostcompat.Check{{ID: "policy.restart", Status: "pending", Detail: "Coordinate an explicit restart."}},
		Guidance: []string{"Use Debian native package guidance."}, ObservedAt: time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC),
	}}
	var out bytes.Buffer
	writeSetupStatus(&out, s)
	for _, want := range []string{"Local runner ready: true", "Package evidence changed.", "provider: deb", "Installed engine: 29.9.1", "Running engine: 29.7.2", "Qualification: unqualified", "Provenance: unverified", "Restart pending: true", "Observe: true · Stop: true · Mutate: false", "5:29.9.1-1~debian.13", "provenance: metadata-only", "Coordinate an explicit restart.", "Use Debian native package guidance.", "2026-10-18T09:00:00Z"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in %s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "Recovery") || strings.Contains(out.String(), "Omarchy") {
		t.Fatal("host diagnostic was recast as pier recovery or a different distro", out.String())
	}
}

func TestSetupTextMissingReportHasUpgradeHint(t *testing.T) {
	var out bytes.Buffer
	writeSetupStatus(&out, onboarding.Status{Ready: true})
	if !strings.Contains(out.String(), "Local runner ready: true") || !strings.Contains(out.String(), "Update the local Omabit and runner helpers") {
		t.Fatal(out.String())
	}
}

func TestSetupTextMetadataDoesNotClaimLiveControlDenial(t *testing.T) {
	var out bytes.Buffer
	writeSetupStatus(&out, onboarding.Status{Ready: true, Compatibility: &hostcompat.Report{Compatibility: "unknown", Qualification: "unqualified"}})
	if !strings.Contains(out.String(), "Metadata only: live control permissions have not been evaluated.") || strings.Contains(out.String(), "Stop: false") {
		t.Fatal("unresolved metadata was presented as a live safety verdict", out.String())
	}
}
