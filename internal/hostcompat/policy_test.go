package hostcompat

import (
	"reflect"
	"testing"
)

func goodEngine(version string) Engine {
	return Engine{Version: version, APIVersion: "1.54", MinAPIVersion: "1.44", Rootless: true, CgroupDriver: "systemd", CgroupVersion: "2", DataRoot: "/engine/data"}
}

func TestVersionPolicyAndHistoricalVersions(t *testing.T) {
	for _, tc := range []struct {
		version, status string
		recorded        bool
	}{
		{"27.1.9", "unsupported", true}, {"27.2.0", "compatible", true}, {"28.5.1", "compatible", true},
		{"29.7.2", "compatible", true}, {"29.9.1", "compatible", true}, {"30.0.0", "unsupported", true},
		{"26.1.4", "unsupported", true}, {"29.9.1-rc1", "unknown", false}, {"29.9", "unknown", false},
		{"29.09.1", "unknown", false}, {"v29.9.1", "unknown", false}, {"29.9.1\n", "unknown", false},
		{"29.9.1+build", "compatible", true}, {"29.9.1+dfsg1", "compatible", true}, {"29.9.1+build.01-foo", "compatible", true},
		{"29.9.1-rc.1+build", "unknown", false}, {"29.9.1+", "unknown", false}, {"29.9.1+a+b", "unknown", false}, {"29.9.1+a..b", "unknown", false},
		{"999999999999.0.0", "unknown", false}, {"", "unknown", false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			if got := EngineVersionStatus(tc.version); got != tc.status {
				t.Fatalf("status %q, want %q", got, tc.status)
			}
			if got := ValidateRecordedEngineVersion(tc.version); got != tc.recorded {
				t.Fatalf("recorded=%v", got)
			}
		})
	}
}

func TestBuildMetadataDoesNotRewriteRawVersionOrCreateRestartPending(t *testing.T) {
	e := goodEngine("29.9.1+dfsg1")
	r := Report{Platform: "linux/amd64", Engine: &e, InstalledEngineVersion: "29.9.1"}
	for _, role := range requiredRoles {
		r.Packages = append(r.Packages, Package{Role: role.role, Provenance: "verified", Executables: []string{"/usr/bin/" + role.executable}})
	}
	r = Evaluate(r, e.DataRoot)
	if !r.CanMutate || r.RestartPending || r.Engine.Version != "29.9.1+dfsg1" || r.Qualification != "unqualified" {
		t.Fatalf("Debian build metadata policy: %+v", r)
	}
	if !EngineVersionsEquivalent("29.9.1+one", "29.9.1+two") || EngineVersionsEquivalent("29.9.1-rc1", "29.9.1") {
		t.Fatal("stable semantic comparison")
	}
}

func TestRequiredAPIFamilyAndControlSafety(t *testing.T) {
	for _, tc := range []struct {
		min, max string
		ok       bool
	}{
		{"1.44", "1.47", true}, {"1.47", "1.54", true}, {"1.48", "1.54", false},
		{"1.24", "1.46", false}, {"2.0", "2.1", false}, {"1.44", "2.47", false},
		{"", "1.54", false}, {"1.047", "1.54", false}, {"1.54", "1.47", false},
	} {
		e := goodEngine("30.0.0")
		e.MinAPIVersion, e.APIVersion = tc.min, tc.max
		if got := CheckEngine(e, e.DataRoot, true) == nil; got != tc.ok {
			t.Fatalf("%s..%s control=%v", tc.min, tc.max, got)
		}
		if CheckEngine(e, e.DataRoot, false) == nil {
			t.Fatal("unsupported family admitted new work")
		}
	}
	for _, mutate := range []func(*Engine){func(e *Engine) { e.Rootless = false }, func(e *Engine) { e.CgroupDriver = "cgroupfs" }, func(e *Engine) { e.CgroupVersion = "1" }, func(e *Engine) { e.DataRoot = "/other" }} {
		e := goodEngine("29.9.1")
		mutate(&e)
		if CheckEngine(e, "/engine/data", true) == nil {
			t.Fatal("unsafe control admitted")
		}
	}
}

func TestEvaluationSeparatesInstalledRunningProvenanceQualification(t *testing.T) {
	e := goodEngine("29.9.1")
	r := Report{Platform: "linux/amd64", Engine: &e, InstalledEngineVersion: e.Version, Distribution: Distribution{ID: "new-distro", Profile: "unknown"}}
	for _, role := range requiredRoles {
		r.Packages = append(r.Packages, Package{Role: role.role, Provenance: "verified", Executables: []string{"/usr/bin/" + role.executable}})
	}
	checked := Evaluate(r, e.DataRoot)
	if checked.Compatibility != "compatible" || checked.Qualification != "unqualified" || !checked.CanMutate || !checked.CanStop || !checked.CanObserve {
		t.Fatalf("unexpected report: %+v", checked)
	}
	if twice := Evaluate(checked, e.DataRoot); !reflect.DeepEqual(twice, checked) {
		t.Fatal("evaluation is not idempotent")
	}
	r.InstalledEngineVersion = ""
	if got := Evaluate(r, e.DataRoot); got.CanMutate || !got.CanStop {
		t.Fatal("unresolved installed version admitted work or blocked control")
	}
	r.InstalledEngineVersion = "29.10.0"
	checked = Evaluate(r, e.DataRoot)
	if !checked.RestartPending || !checked.CanMutate || !checked.CanStop || checked.Engine.Version != "29.9.1" {
		t.Fatalf("mixed installed/running: %+v", checked)
	}
	r.InstalledEngineVersion = "30.0.0"
	if got := Evaluate(r, e.DataRoot); got.CanMutate || !got.RestartPending || !got.CanStop {
		t.Fatal("unsupported installed family admitted new work or disabled control")
	}
	r.InstalledEngineVersion = e.Version
	r.Packages[0].Provenance = "metadata-only"
	checked = Evaluate(r, e.DataRoot)
	if checked.CanMutate || !checked.CanObserve || !checked.CanStop || checked.Provenance == "verified" {
		t.Fatal("metadata admitted work or disabled safe control")
	}
	e.Version = "29.7.2"
	r.Qualification = "qualified"
	if got := Evaluate(r, e.DataRoot); got.Qualification != "unqualified" {
		t.Fatal("baseline string qualified a new profile")
	}
	e.Version = "30.0.0"
	if got := Evaluate(r, e.DataRoot); got.CanMutate || !got.CanStop || got.Compatibility != "unsupported" {
		t.Fatal("unsupported family control policy")
	}
}
