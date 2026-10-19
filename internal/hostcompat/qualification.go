package hostcompat

import (
	"io"
	"strings"
)

// QualificationRecord is release-maintainer evidence, not host-supplied claims.
// Populate records only from reviewed, recorded full-stack qualification runs.
// The exact profile identifies the observed running-daemon executable and
// kernel/systemd/rootless-launcher/configuration stack attested by the caller.
// Installed package versions alone cannot establish that running profile (in
// particular across same-upstream-version native package revision updates).
type QualificationRecord struct {
	ID                  string                 `json:"id"`
	EvidenceReference   string                 `json:"evidence_reference"`
	PolicyID            string                 `json:"policy_id"`
	Platform            string                 `json:"platform"`
	Distribution        Distribution           `json:"distribution"`
	EngineVersion       string                 `json:"engine_version"`
	APIVersion          string                 `json:"api_version"`
	EngineAPIVersion    string                 `json:"engine_api_version"`
	EngineMinAPIVersion string                 `json:"engine_min_api_version"`
	Packages            []QualificationPackage `json:"packages"`
}

type QualificationPackage struct {
	Role        string   `json:"role"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Executables []string `json:"executables"`
}

// ReadQualificationRecords bounds and validates packaged record input. It does
// not authenticate the input: the caller must load reviewed embedded data or a
// trusted root-owned release file, never a remote host's assertions.
func ReadQualificationRecords(r io.Reader) ([]QualificationRecord, error) {
	b, err := io.ReadAll(io.LimitReader(r, (256<<10)+1))
	if err != nil || len(b) > 256<<10 {
		return nil, errEvidence
	}
	var records []QualificationRecord
	if decodeStrict(b, &records) != nil || len(records) > 32 {
		return nil, errEvidence
	}
	ids := map[string]bool{}
	for _, record := range records {
		if !validQualification(record) || ids[record.ID] {
			return nil, errEvidence
		}
		ids[record.ID] = true
	}
	return records, nil
}

func validQualification(r QualificationRecord) bool {
	if !token(r.ID) || r.PolicyID != PolicyID || EngineVersionStatus(r.EngineVersion) != "compatible" || r.APIVersion != APIVersion || !supportsAPI(r.EngineMinAPIVersion, r.EngineAPIVersion) || !strings.HasPrefix(r.Platform, "linux/") || !token(strings.TrimPrefix(r.Platform, "linux/")) {
		return false
	}
	if !token(r.Distribution.ID) || (r.Distribution.VersionID != "" && !token(r.Distribution.VersionID)) || (r.Distribution.Provider != "pacman" && r.Distribution.Provider != "deb") || !token(r.Distribution.Profile) || r.Distribution.Profile == "unqualified" || r.Distribution.Profile == "unknown" {
		return false
	}
	if len(r.EvidenceReference) == 0 || len(r.EvidenceReference) > 512 || strings.ContainsAny(r.EvidenceReference, "\r\n\x00") || len(r.Distribution.IDLike) > 8 || len(r.Packages) < len(requiredRoles) || len(r.Packages) > 32 {
		return false
	}
	roles := map[string]bool{}
	for _, id := range r.Distribution.IDLike {
		if !token(id) {
			return false
		}
	}
	for _, p := range r.Packages {
		if !token(p.Role) || roles[p.Role] || !token(p.Name) || !token(p.Version) || len(p.Executables) == 0 || len(p.Executables) > 8 {
			return false
		}
		roles[p.Role] = true
		if p.Role == "engine" && !EngineVersionsEquivalent(upstreamVersion(p.Version), r.EngineVersion) {
			return false
		}
		for _, name := range p.Executables {
			if len(name) > 4096 || !strings.HasPrefix(name, "/usr/") || strings.Contains(name, "..") || strings.ContainsAny(name, "\r\n\x00") {
				return false
			}
		}
	}
	for _, role := range requiredRoles {
		if !roles[role.role] {
			return false
		}
	}
	return true
}

// EvaluateWithQualification applies compatibility first, then an exact recorded
// stack/profile/API/package tuple. Drift remains compatible but unqualified.
// No static baseline is fabricated here: release maintainers supply records only
// when the original run includes the complete tuple and profile evidence.
func EvaluateWithQualification(report Report, expectedDataRoot string, records []QualificationRecord) Report {
	r := Evaluate(report, expectedDataRoot)
	if !r.CanMutate || r.RestartPending || r.Engine == nil || len(records) > 32 {
		return r
	}
	for _, record := range records {
		if !validQualification(record) || r.Platform != record.Platform || r.Engine.Version != record.EngineVersion || r.Engine.APIVersion != record.EngineAPIVersion || r.Engine.MinAPIVersion != record.EngineMinAPIVersion {
			continue
		}
		a, b := r.Distribution, record.Distribution
		if a.ID != b.ID || a.VersionID != b.VersionID || a.Provider != b.Provider || a.Profile != b.Profile || !sameStrings(a.IDLike, b.IDLike) || len(r.Packages) != len(record.Packages) {
			continue
		}
		matched := true
		seen := map[string]bool{}
		for _, p := range r.Packages {
			if seen[p.Role] {
				matched = false
				break
			}
			seen[p.Role] = true
			found := false
			for _, q := range record.Packages {
				if p.Role == q.Role && p.Name == q.Name && p.Version == q.Version && sameStrings(p.Executables, q.Executables) {
					found = true
				}
			}
			if !found {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		r.Qualification = "qualified"
		for i := range r.Checks {
			if r.Checks[i].ID == "policy.qualification" {
				r.Checks[i] = Check{"policy.qualification", "qualified", "Exact recorded stack, distribution profile, package tuple and API matched."}
			}
		}
		return r
	}
	return r
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
