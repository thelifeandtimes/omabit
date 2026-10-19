//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piercopy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func importCatalogue(d dependencies) (*qualifier, error) {
	d.managed = false
	d.profile = ProfileImport
	q := &qualifier{d: d, base: filepath.Join(d.home, ".local/share/urbit-runner/qualification")}
	if err := q.accountBoundary(false); err != nil {
		return nil, err
	}
	for _, p := range q.storagePaths() {
		if _, err := q.d.stat(p); errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		if err := q.owned(p, os.ModeDir, false); err != nil {
			return nil, err
		}
	}
	return q, nil
}

func (q *qualifier) importReport(id string) (Result, error) {
	var r Result
	if !runIDPattern.MatchString(id) {
		return r, errors.New("invalid import source ID")
	}
	path := filepath.Join(q.base, id, "report.json")
	if err := q.owned(filepath.Dir(path), os.ModeDir, true); err != nil {
		return r, err
	}
	if err := q.owned(path, 0, true); err != nil {
		return r, err
	}
	n, err := q.d.stat(path)
	if err != nil || n.links != 1 {
		return r, errors.New("unsafe import report")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return r, errors.New("import report unavailable")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxPriorReportBytes+1))
	if err != nil || len(data) > maxPriorReportBytes || json.Unmarshal(data, &r) != nil || r.RunID != id || r.Directory != filepath.Dir(path) || r.ReportPath != path {
		return r, errors.New("invalid import report metadata")
	}
	return r, nil
}

func (q *qualifier) importSourceInfo(id string) (protocol.ImportSource, Result, error) {
	var s protocol.ImportSource
	r, err := q.importReport(id)
	if err != nil {
		return s, r, err
	}
	if r.Profile != ProfileImport || !r.CheckpointPassed || !r.CleanupComplete || !r.LimitsObserved || !r.PierMetadataSeen || r.RuntimeQualified || r.PierImport == nil || r.Image != candidate.Image || !acceptedImageID(r.ImageID) || (r.RepoDigest != candidate.Image && r.RepoDigest != strings.TrimPrefix(candidate.Image, "docker.io/")) || !hostcompat.ValidateRecordedEngineVersion(r.EngineVersion) || r.Platform != candidate.Platform || r.VereVersion != candidate.VereVersion || r.PillSHA256 != q.d.pillSHA || r.PillBytes != q.d.pillBytes || r.PillURL != candidate.PillURL || r.PillPath != filepath.Join(r.Directory, candidate.PillName) {
		return s, r, errors.New("source is not a completed pinned disposable-fake import qualification")
	}
	if err := q.priorReport(id); err != nil {
		return s, r, err
	}
	p := r.PierImport
	if p.Uncertain || p.FinishedAt == nil || !p.CopyVerified || !p.SourceUnchanged || !p.FirstResume || !p.ReplacementResume {
		return s, r, errors.New("source transfer evidence is incomplete")
	}
	s = protocol.ImportSource{RunID: id, ContentDigest: p.ContentDigest, Files: p.CopiedFiles, Bytes: p.CopiedBytes, Ship: "~zod", QualifiedAt: *p.FinishedAt}
	return s, r, nil
}

func ListImportSources(ctx context.Context) ([]protocol.ImportSource, error) {
	return listImportSources(ctx, defaults())
}
func listImportSources(ctx context.Context, d dependencies) ([]protocol.ImportSource, error) {
	out := []protocol.ImportSource{}
	q, err := importCatalogue(d)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	f, err := os.Open(q.base)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(maxPriorRuns + 2)
	if (err != nil && err != io.EOF) || len(entries) > maxPriorRuns+1 {
		return nil, errors.New("import catalogue limit exceeded")
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if e.Name() == ".lock" {
			continue
		}
		r, err := q.importReport(e.Name())
		if err != nil {
			return nil, err
		}
		if r.Profile != ProfileImport || !r.CheckpointPassed {
			continue
		}
		s, _, err := q.importSourceInfo(e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RunID < out[j].RunID })
	return out, nil
}

func applyImportedFake(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if persist == nil || state.Import == nil || state.Import.StartedAt != nil || state.Prepared || state.Initialized {
		return state, unknown, errors.New("import requires its fresh durable registration")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	catalogue, err := importCatalogue(d)
	if err != nil {
		return state, unknown, err
	}
	source, report, err := catalogue.importSourceInfo(state.Import.Source.RunID)
	if err != nil {
		return state, unknown, err
	}
	want := state.Import.Source
	want.ClaimedBy = ""
	if source != want {
		return state, unknown, errors.New("selected import source metadata changed")
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	finish := func(err error) (ManagedState, ManagedObservation, error) {
		state.Runtime = q.r
		return state, unknown, err
	}
	if err := q.accountBoundary(false); err != nil {
		return finish(err)
	}
	for _, p := range q.storagePaths() {
		if err := q.owned(p, os.ModeDir, false); err != nil {
			return finish(err)
		}
	}
	catalogue.r = report
	catalogue.config = filepath.Join(report.Directory, "docker-config")
	if err := catalogue.verifyEngine(ctx); err != nil {
		return finish(err)
	}
	out, err := catalogue.docker(ctx, catalogue.d.times.command, "container", "ls", "--all", "--filter", "label="+label+"=true", "--filter", "label="+runLabel+"="+source.RunID, "--format", "{{.Names}}")
	if err != nil || strings.TrimSpace(out) != "" {
		return finish(errors.New("qualification source has unresolved container resources"))
	}
	sourcePath := filepath.Join(report.Directory, "source")
	if err := catalogue.owned(sourcePath, os.ModeDir, true); err != nil {
		return finish(err)
	}
	input, err := os.OpenRoot(sourcePath)
	if err != nil {
		return finish(err)
	}
	defer input.Close()
	before, err := piercopy.Measure(ctx, input)
	if err != nil || before.Digest != source.ContentDigest || before.Files != source.Files || before.Bytes != source.Bytes {
		return finish(errors.New("stopped import source content changed"))
	}
	_, disk, err := d.admission(q.base)
	if err != nil || disk < uint64(source.Bytes)+(8<<30) {
		return finish(errors.New("import requires source logical bytes plus 8 GiB free disk"))
	}
	dir := filepath.Join(q.base, state.ID)
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		return finish(errors.New("import destination already exists"))
	}
	now := time.Now().UTC()
	state.Import.StartedAt = &now
	state.Import.Uncertain = true
	state.Prepared = true
	q.r = Result{Profile: ProfileHTTP, RunID: state.ID, Directory: dir, Image: candidate.Image, ImageID: report.ImageID, RepoDigest: report.RepoDigest, Platform: candidate.Platform, EngineVersion: catalogue.engineEvidence.Version, VereVersion: candidate.VereVersion, PillURL: candidate.PillURL, PillSHA256: d.pillSHA, PillBytes: d.pillBytes, PillPath: filepath.Join(dir, candidate.PillName), Containers: []ContainerRecord{}}
	if err := q.save(); err != nil {
		return finish(err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return finish(err)
	}
	for _, p := range []string{q.config} {
		if err := os.Mkdir(p, 0700); err != nil {
			return finish(err)
		}
	}
	if _, err = pierstore.PrepareMount(d.home, state.Storage); err != nil {
		return finish(err)
	}
	if err = q.save(); err != nil {
		return finish(err)
	}
	if err = os.Mkdir(q.pierPath(), 0700); err != nil {
		return finish(err)
	}
	// The disposable source report measures its parent. Copy only the actual
	// source pier into the permanent target, verifying that subtree separately.
	pierInput, err := os.OpenRoot(filepath.Join(sourcePath, "fake-zod"))
	if err != nil {
		return finish(err)
	}
	defer pierInput.Close()
	pierBefore, err := piercopy.Measure(ctx, pierInput)
	if err != nil {
		return finish(err)
	}
	destination, err := os.OpenRoot(q.pierPath())
	if err != nil {
		return finish(err)
	}
	defer destination.Close()
	copied, err := piercopy.Copy(ctx, pierInput, destination, func(files int, bytes int64) error {
		state.Import.CopiedFiles = files
		state.Import.CopiedBytes = bytes
		return q.save()
	})
	if err != nil {
		return finish(err)
	}
	after, measureErr := piercopy.Measure(ctx, input)
	if copied != pierBefore || measureErr != nil || after != before {
		return finish(errors.New("import source changed between inspection and transfer"))
	}
	if err := q.pier(); err != nil {
		return finish(err)
	}
	for _, p := range []string{dir, q.base} {
		f, err := os.Open(p)
		if err != nil {
			return finish(err)
		}
		err = f.Sync()
		f.Close()
		if err != nil {
			return finish(err)
		}
	}
	completed := time.Now().UTC()
	state.Import.FinishedAt = &completed
	state.Import.Uncertain = false
	state.Initialized = true
	if err := q.save(); err != nil {
		state.Import.Uncertain = true
		state.Initialized = false
		return finish(err)
	}
	state.Runtime = q.r
	return state, ManagedObservation{State: "stopped", Health: "unknown"}, nil
}
