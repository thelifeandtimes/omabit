package checkplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

const MaxReportBytes = 1 << 20
const MaxReports = 32

var ErrReport = errors.New("named report is unavailable, changed or outside the supported format")

type ReportedPhase struct {
	Ordinal  int    `json:"ordinal"`
	Category string `json:"reported_category"`
	Passed   bool   `json:"reported_passed"`
}

// Summary deliberately contains no free-form logs/errors, host paths, network
// addresses or report-supplied command/source claims. Even a matching digest
// authenticates selected bytes only, not the truth of a caller-written report.
type Summary struct {
	Version                      int             `json:"version"`
	Name                         string          `json:"name"`
	PlanID                       string          `json:"plan_id"`
	SHA256                       string          `json:"sha256"`
	Bytes                        int64           `json:"bytes"`
	ReadAt                       time.Time       `json:"read_at"`
	Basis                        string          `json:"basis"`
	RunID                        string          `json:"run_id"`
	Profile                      string          `json:"profile"`
	ReportedCheckpointPassed     bool            `json:"reported_checkpoint_passed"`
	ReportedCleanupComplete      bool            `json:"reported_cleanup_complete"`
	ReportedLimitsObserved       bool            `json:"reported_limits_observed"`
	ReportedPierMetadataSeen     bool            `json:"reported_pier_metadata_seen"`
	ReportedPhases               []ReportedPhase `json:"reported_phases"`
	ReportedContainers           int             `json:"reported_containers"`
	ReportedPinsMatchCurrentPlan bool            `json:"reported_pins_match_current_plan"`
	CurrentPlanSHA256            string          `json:"current_plan_sha256"`
	Verification                 string          `json:"verification"`
	SourceProvenance             string          `json:"source_provenance"`
	ExecutedByThisService        bool            `json:"executed_by_this_service"`
	RuntimeQualified             bool            `json:"runtime_qualified"`
}

// Report is an immutable startup-selected snapshot. No operation can supply a
// path or change its bytes, name or digest. The file is closed before serving.
type Report struct{ summary Summary }

type ReportBinding struct {
	Name   string `json:"name"`
	PlanID string `json:"plan_id"`
	SHA256 string `json:"sha256"`
}

type ReportSelection struct {
	ReportBinding
	IncludePrivateData bool `json:"include_private_data"`
}

func (s ReportSelection) Validate() error {
	if !protocol.ValidID(s.Name) || !validHex(s.SHA256, 64) || !s.IncludePrivateData {
		return ErrSelection
	}
	_, err := Read(s.PlanID)
	return err
}

func (r *Report) Name() string {
	if r == nil {
		return ""
	}
	return r.summary.Name
}
func (r *Report) SHA256() string {
	if r == nil {
		return ""
	}
	return r.summary.SHA256
}
func (r *Report) PlanID() string {
	if r == nil {
		return ""
	}
	return r.summary.PlanID
}
func (r *Report) Summary() Summary {
	out := r.summary
	out.ReportedPhases = slices.Clone(out.ReportedPhases)
	return out
}

// OpenReport is a startup-only local-owner selection, never a tool argument.
// It reads one exact owner-private ordinary file; it runs no command/test.
func OpenReport(name, path, expectedSHA256 string) (*Report, error) {
	if !protocol.ValidID(name) || !validHex(expectedSHA256, 64) {
		return nil, ErrSelection
	}
	raw, err := readReportFile(path)
	if err != nil {
		return nil, ErrReport
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != expectedSHA256 {
		return nil, ErrReport
	}
	return parseReport(name, expectedSHA256, raw, time.Now().UTC())
}

func validHex(value string, size int) bool {
	if len(value) != size {
		return false
	}
	for _, v := range value {
		if !(v >= '0' && v <= '9' || v >= 'a' && v <= 'f') {
			return false
		}
	}
	return true
}

func parseReport(name, digest string, raw []byte, readAt time.Time) (*Report, error) {
	if len(raw) == 0 || len(raw) > MaxReportBytes || uniqueJSON(raw) != nil {
		return nil, ErrReport
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return nil, ErrReport
	}
	for _, key := range []string{"profile", "run_id", "checkpoint_passed", "runtime_qualified", "cleanup_complete", "phases", "containers"} {
		value, ok := fields[key]
		if !ok || (key != "phases" && key != "containers" && bytes.Equal(bytes.TrimSpace(value), []byte("null"))) {
			return nil, ErrReport
		}
	}
	var saved runtimequalify.Result
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&saved) != nil || saved.RuntimeQualified || !validHex(saved.RunID, 32) || len(saved.Phases) > 256 || len(saved.Containers) > 64 {
		return nil, ErrReport
	}
	var id string
	switch saved.Profile {
	case runtimequalify.ProfileLifecycle:
		id = Lifecycle
	case runtimequalify.ProfileHTTP:
		id = HTTP
	default:
		return nil, ErrReport
	}
	plan, err := Read(id)
	if err != nil {
		return nil, ErrReport
	}
	if saved.CheckpointPassed && (!saved.CleanupComplete || len(saved.Phases) == 0 || saved.Failure != "") {
		return nil, ErrReport
	}
	out := Summary{Version: 1, Name: name, PlanID: id, SHA256: digest, Bytes: int64(len(raw)), ReadAt: readAt, Basis: "immutable startup-selected historical file snapshot; no fresh runtime observation or execution", RunID: saved.RunID, Profile: saved.Profile, ReportedCheckpointPassed: saved.CheckpointPassed, ReportedCleanupComplete: saved.CleanupComplete, ReportedLimitsObserved: saved.LimitsObserved, ReportedPierMetadataSeen: saved.PierMetadataSeen, ReportedContainers: len(saved.Containers), CurrentPlanSHA256: plan.SHA256, Verification: "selected-content-digest-and-bounded-format-only; reported outcomes are not independently attested", SourceProvenance: "unavailable in this legacy report; repository snapshot and execution environment are not established", ReportedPhases: []ReportedPhase{}}
	for i, phase := range saved.Phases {
		if saved.CheckpointPassed && !phase.Passed {
			return nil, ErrReport
		}
		out.ReportedPhases = append(out.ReportedPhases, ReportedPhase{Ordinal: i + 1, Category: phaseCategory(id, phase.Name), Passed: phase.Passed})
	}
	repoMatches := saved.RepoDigest == plan.Runtime.Image || saved.RepoDigest == strings.TrimPrefix(plan.Runtime.Image, "docker.io/")
	imageMatches := saved.ImageID == plan.Runtime.ConfigDigest || saved.ImageID == plan.Runtime.ManifestDigest
	out.ReportedPinsMatchCurrentPlan = saved.Image == plan.Runtime.Image && repoMatches && imageMatches && saved.EngineVersion == plan.Runtime.EngineVersion && saved.Platform == plan.Runtime.Platform && saved.VereVersion == plan.Runtime.VereVersion && saved.PillURL == plan.Runtime.Pill.URL && saved.PillSHA256 == plan.Runtime.Pill.SHA256 && saved.PillBytes == plan.Runtime.Pill.Bytes
	return &Report{summary: out}, nil
}

// Only fixed categories from the two supported checkpoints can enter a summary.
// Unknown names remain visible as other, without copying arbitrary report text.
func phaseCategory(plan, name string) string {
	switch name {
	case "engine", "image", "pill", "probe", "cleanup":
		return name
	case "boot", "restart", "pack", "post-pack":
		if plan == Lifecycle {
			return name
		}
	case "network", "host-http-boot", "host-http-restart":
		if plan == HTTP {
			return name
		}
	}
	return "other"
}

// Reject duplicate/ambiguous fields and excessive nesting before typed decode.
// This is bounded JSON parsing of the existing report, not a second schema.
func uniqueJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	remaining := 131072
	var value func(int) error
	value = func(depth int) error {
		remaining--
		if remaining < 0 || depth > 32 {
			return ErrReport
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return ErrReport
				}
				seen[name] = true
				if err = value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err = value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return ErrReport
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if delimiter == '{' && end != json.Delim('}') || delimiter == '[' && end != json.Delim(']') {
			return ErrReport
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return ErrReport
	}
	return nil
}
