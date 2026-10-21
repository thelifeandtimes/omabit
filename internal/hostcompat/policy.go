// Package hostcompat reports native host evidence independently of installation,
// orchestration and the application's persisted runtime history.
package hostcompat

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const Feature = "host-compatibility-v1"
const PolicyID = "docker-rootless-v1"
const APIVersion = "1.47"

const minEngineMajor = 27
const minEngineMinor = 2
const maxEngineMajorExclusive = 30
const maxVersionComponentDigits = 6
const maxEngineVersionBytes = 256

type Check struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type Distribution struct {
	ID        string   `json:"id"`
	VersionID string   `json:"version_id"`
	Provider  string   `json:"provider"`
	Profile   string   `json:"profile"`
	IDLike    []string `json:"id_like"`
}

type Package struct {
	Role            string   `json:"role"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	UpstreamVersion string   `json:"upstream_version"`
	Provenance      string   `json:"provenance"`
	Detail          string   `json:"detail"`
	Executables     []string `json:"executables"`
}

type Engine struct {
	Version       string `json:"version"`
	APIVersion    string `json:"api_version"`
	MinAPIVersion string `json:"min_api_version"`
	Rootless      bool   `json:"rootless"`
	CgroupDriver  string `json:"cgroup_driver"`
	CgroupVersion string `json:"cgroup_version"`
	DataRoot      string `json:"data_root"`
}

type Report struct {
	SchemaVersion          int          `json:"schema_version"`
	PolicyID               string       `json:"policy_id"`
	Platform               string       `json:"platform"`
	Distribution           Distribution `json:"distribution"`
	Packages               []Package    `json:"packages"`
	InstalledEngineVersion string       `json:"installed_engine_version"`
	Engine                 *Engine      `json:"engine"`
	Compatibility          string       `json:"compatibility"`
	Qualification          string       `json:"qualification"`
	Provenance             string       `json:"provenance"`
	RestartPending         bool         `json:"restart_pending"`
	CanObserve             bool         `json:"can_observe"`
	CanStop                bool         `json:"can_stop"`
	CanMutate              bool         `json:"can_mutate"`
	Checks                 []Check      `json:"checks"`
	Guidance               []string     `json:"guidance"`
	ObservedAt             time.Time    `json:"observed_at"`
}

func numericVersion(s string, count int) ([]int, bool) {
	if len(s) > 40 {
		return nil, false
	}
	parts := strings.Split(s, ".")
	if len(parts) != count {
		return nil, false
	}
	n := make([]int, count)
	for i, p := range parts {
		if p == "" || len(p) > maxVersionComponentDigits || (len(p) > 1 && p[0] == '0') {
			return nil, false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil, false
			}
		}
		n[i], _ = strconv.Atoi(p)
	}
	return n, true
}

// EngineVersionStatus describes contract support, never live qualification.
func EngineVersionStatus(version string) string {
	_, v, ok := stableEngineVersion(version)
	if !ok {
		return "unknown"
	}
	if v[0] < minEngineMajor || v[0] >= maxEngineMajorExclusive || (v[0] == minEngineMajor && v[1] < minEngineMinor) {
		return "unsupported"
	}
	return "compatible"
}

// EnginePackageVersionStatus applies the same policy to a native repository
// candidate before provisioning installs it. Metadata is not provenance evidence.
func EnginePackageVersionStatus(version string) string {
	return EngineVersionStatus(upstreamVersion(version))
}

// ValidateRecordedEngineVersion validates history without applying today's new
// work policy. Callers must not rewrite a historical version to the baseline.
func ValidateRecordedEngineVersion(version string) bool {
	_, v, ok := stableEngineVersion(version)
	return ok && v[0] > 0
}

func stableEngineVersion(version string) (string, []int, bool) {
	if len(version) > maxEngineVersionBytes {
		return "", nil, false
	}
	core, build, hasBuild := strings.Cut(version, "+")
	if hasBuild {
		for _, part := range strings.Split(build, ".") {
			if part == "" {
				return "", nil, false
			}
			for _, c := range part {
				if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
					return "", nil, false
				}
			}
		}
	}
	v, ok := numericVersion(core, 3)
	return core, v, ok
}

// EngineVersionsEquivalent compares stable SemVer precedence, ignoring build
// metadata, while callers preserve both raw versions for reporting/history.
func EngineVersionsEquivalent(a, b string) bool {
	x, _, xok := stableEngineVersion(a)
	y, _, yok := stableEngineVersion(b)
	return xok && yok && x == y
}

func supportsAPI(min, max string) bool {
	a, aok := numericVersion(min, 2)
	b, bok := numericVersion(max, 2)
	return aok && bok && a[0] == 1 && b[0] == 1 && a[1] <= 47 && b[1] >= 47 && a[1] <= b[1]
}

// CheckEngine checks daemon-reported invariants. The caller must establish socket
// identity (ProbeEngine does so); this pure check cannot prove process ownership.
func CheckEngine(engine Engine, expectedDataRoot string, controlOnly bool) error {
	if !supportsAPI(engine.MinAPIVersion, engine.APIVersion) {
		return errors.New("engine does not support required API 1.47")
	}
	if !engine.Rootless {
		return errors.New("engine is not rootless")
	}
	if engine.CgroupDriver != "systemd" || engine.CgroupVersion != "2" {
		return errors.New("engine requires systemd cgroup v2")
	}
	if expectedDataRoot == "" || !filepath.IsAbs(expectedDataRoot) || filepath.Clean(expectedDataRoot) != expectedDataRoot || engine.DataRoot != expectedDataRoot {
		return errors.New("engine data root does not match expected root")
	}
	if !controlOnly && EngineVersionStatus(engine.Version) != "compatible" {
		return errors.New("engine version is outside supported stable families")
	}
	return nil
}

// Evaluate does not probe, install, restart, or qualify a new distribution. Engine
// must come from an identity-verified probe. Package provenance gates new work,
// while safe observation/Stop remain possible after a package update.
func Evaluate(report Report, expectedDataRoot string) Report {
	report.SchemaVersion, report.PolicyID = 1, PolicyID
	report.Compatibility, report.Qualification = "unknown", "unqualified"
	report.CanObserve, report.CanStop, report.CanMutate, report.RestartPending = false, false, false, false
	// Replace derived checks so repeated evaluation is idempotent.
	checks := make([]Check, 0, len(report.Checks)+4)
	for _, c := range report.Checks {
		if !strings.HasPrefix(c.ID, "policy.") {
			checks = append(checks, c)
		}
	}
	report.Checks = checks
	report.Provenance = "unknown"
	if len(report.Packages) > 0 {
		report.Provenance = "verified"
		for _, p := range report.Packages {
			if p.Provenance != "verified" {
				report.Provenance = "unverified"
				break
			}
		}
	}
	if report.Engine == nil {
		report.Checks = append(report.Checks, Check{"policy.engine", "unknown", "Running engine has not been identity-verified."})
		return report
	}
	e := *report.Engine
	report.RestartPending = report.InstalledEngineVersion != "" && !EngineVersionsEquivalent(e.Version, report.InstalledEngineVersion)
	if !strings.HasPrefix(report.Platform, "linux/") {
		report.Compatibility = "unsupported"
		return report
	}
	if err := CheckEngine(e, expectedDataRoot, true); err != nil {
		report.Compatibility = "incompatible"
		report.Checks = append(report.Checks, Check{"policy.engine", "failed", err.Error()})
		return report
	}
	report.CanObserve, report.CanStop = true, true
	report.Compatibility = EngineVersionStatus(e.Version)
	report.CanMutate = report.Compatibility == "compatible" && report.Provenance == "verified" && EngineVersionStatus(report.InstalledEngineVersion) == "compatible" && completeStack(report.Packages)
	report.Checks = append(report.Checks, Check{"policy.engine", report.Compatibility, "API 1.47, rootless, systemd cgroup v2 and expected data root checked."})
	if report.InstalledEngineVersion == "" {
		report.Checks = append(report.Checks, Check{"policy.installed-engine", "unknown", "Installed stable engine version is unresolved; new work requires compatible installed and running evidence."})
	} else if status := EngineVersionStatus(report.InstalledEngineVersion); status != "compatible" {
		report.Checks = append(report.Checks, Check{"policy.installed-engine", status, "Installed engine is outside the supported stable families; new work is blocked independently of safe running-engine control."})
	}
	if report.RestartPending {
		report.Checks = append(report.Checks, Check{"policy.restart", "pending", "Installed and running engine versions differ; coordinate an explicit restart."})
	}
	detail := "No matching full-stack and distribution-profile qualification record is supplied."
	if e.Version == "29.7.2" {
		detail = "Engine matches the recorded 29.7.2 baseline; this alone does not qualify the stack or distribution."
	}
	report.Checks = append(report.Checks, Check{"policy.qualification", "unqualified", detail})
	return report
}

func completeStack(packages []Package) bool {
	for _, role := range requiredRoles {
		found := false
		for _, p := range packages {
			if p.Role == role.role && p.Provenance == "verified" && len(p.Executables) > 0 {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
