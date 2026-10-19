package hostcompat

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func qualificationFixture() (Report, QualificationRecord) {
	e := goodEngine("29.7.2")
	r := Report{Platform: "linux/amd64", Distribution: Distribution{ID: "arch", Provider: "pacman", Profile: "fixture-kernel-systemd-launcher-cgroup-profile"}, Engine: &e, InstalledEngineVersion: e.Version}
	record := QualificationRecord{ID: "isolated-test-record", EvidenceReference: "test fixture, not a release qualification", PolicyID: PolicyID, Platform: r.Platform, Distribution: r.Distribution, EngineVersion: e.Version, APIVersion: APIVersion, EngineAPIVersion: e.APIVersion, EngineMinAPIVersion: e.MinAPIVersion}
	for _, role := range requiredRoles {
		version := "1.2.3-1"
		if role.role == "engine" {
			version = "1:29.7.2-1"
		}
		p := Package{Role: role.role, Name: role.executable, Version: version, Provenance: "verified", Executables: []string{"/usr/bin/" + role.executable}}
		r.Packages = append(r.Packages, p)
		record.Packages = append(record.Packages, QualificationPackage{p.Role, p.Name, p.Version, p.Executables})
	}
	return r, record
}

func TestFutureCompatibleVersionQualifiesOnlyWithExactTrustedRecord(t *testing.T) {
	r, record := qualificationFixture()
	r.Engine.Version = "29.9.1+dfsg1"
	r.InstalledEngineVersion = "29.9.1"
	record.EngineVersion = r.Engine.Version
	for i := range r.Packages {
		if r.Packages[i].Role == "engine" {
			r.Packages[i].Version = "29.9.1+dfsg1-1"
			record.Packages[i].Version = r.Packages[i].Version
		}
	}
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, nil); got.Qualification != "unqualified" {
		t.Fatal("future version implicitly qualified")
	}
	b, _ := json.Marshal([]QualificationRecord{record})
	records, err := ReadQualificationRecords(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, records); got.Qualification != "qualified" {
		t.Fatalf("exact trusted future record rejected: %+v", got)
	}
	r.Engine.Version = "29.9.1+different"
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, records); got.Qualification != "unqualified" {
		t.Fatal("raw build evidence drift qualified")
	}
}

func TestExactQualificationRecordAndBoundedInput(t *testing.T) {
	r, record := qualificationFixture()
	b, _ := json.Marshal([]QualificationRecord{record})
	records, err := ReadQualificationRecords(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, records); got.Qualification != "qualified" || !got.CanMutate {
		t.Fatalf("exact record failed: %+v", got)
	}
	r.Distribution.IDLike = []string{}
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, records); got.Qualification != "qualified" {
		t.Fatal("equivalent empty IDLike forms differed")
	}
	for _, change := range []func(*Report){
		func(r *Report) { r.Distribution.Profile = "different-kernel-profile" },
		func(r *Report) { r.Distribution.ID = "new-distro" },
		func(r *Report) { r.Packages[0].Version = "1:29.7.2-2" },
		func(r *Report) { r.Engine.APIVersion = "1.55" },
		func(r *Report) { r.Engine.Version = "29.9.1"; r.InstalledEngineVersion = "29.9.1" },
		func(r *Report) { r.Packages[0].Provenance = "metadata-only" },
	} {
		r, _ := qualificationFixture()
		change(&r)
		if got := EvaluateWithQualification(r, r.Engine.DataRoot, records); got.Qualification != "unqualified" {
			t.Fatalf("drift qualified: %+v", got)
		}
	}
	if _, err := ReadQualificationRecords(strings.NewReader(strings.Repeat(" ", 256<<10) + "[]")); err == nil {
		t.Fatal("unbounded record accepted")
	}
	record.EngineVersion = "29.9.1"
	b, _ = json.Marshal([]QualificationRecord{record})
	if _, err := ReadQualificationRecords(bytes.NewReader(b)); err == nil {
		t.Fatal("inconsistent engine/package record accepted")
	}
	r, record = qualificationFixture()
	record.Packages = record.Packages[:1]
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, []QualificationRecord{record}); got.Qualification != "unqualified" {
		t.Fatal("partial stack qualified")
	}
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, nil); got.Qualification != "unqualified" {
		t.Fatal("bare baseline string qualified")
	}
	r, record = qualificationFixture()
	extra := QualificationPackage{Role: "firewall", Name: "iptables", Version: "1.8.11-1", Executables: []string{"/usr/bin/iptables"}}
	record.Packages = append(record.Packages, extra)
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, []QualificationRecord{record}); got.Qualification != "unqualified" {
		t.Fatal("missing recorded full-stack package was ignored")
	}
	r.Packages = append(r.Packages, Package{Role: extra.Role, Name: extra.Name, Version: extra.Version, Executables: extra.Executables, Provenance: "verified"})
	if got := EvaluateWithQualification(r, r.Engine.DataRoot, []QualificationRecord{record}); got.Qualification != "qualified" {
		t.Fatal("complete extra package tuple did not match")
	}
}
