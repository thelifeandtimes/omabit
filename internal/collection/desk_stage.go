package collection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type DeskStagePlan struct {
	Version  int                  `json:"version"`
	RunnerID string               `json:"runner_id"`
	Mode     string               `json:"mode"`
	Intent   deskwork.StageIntent `json:"intent"`
}

// DeskStageAdmissionError establishes that no staging upload was dispatched.
// Failures after handing the stream to the transport retain its uncertainty.
type DeskStageAdmissionError struct{ Err error }

func (e *DeskStageAdmissionError) Error() string { return e.Err.Error() }
func (e *DeskStageAdmissionError) Unwrap() error { return e.Err }

func (p DeskStagePlan) Validate() error {
	if p.Version != 1 || !protocol.ValidID(p.RunnerID) || !protocol.ValidID(p.Mode) || p.Intent.Validate() != nil {
		return errors.New("invalid source staging plan")
	}
	return nil
}
func (r *Registry) PlanDeskStage(ctx context.Context, name, runner, instance, desk, directory string) (DeskStagePlan, error) {
	var out DeskStagePlan
	if desk == "" {
		return out, errors.New("select a source desk")
	}
	m, err := deskwork.Snapshot(directory)
	if err != nil {
		return out, err
	}
	return r.planDeskStage(ctx, name, runner, instance, desk, m)
}

// PlanDeskStageRoot is the confined source entry point for a service with
// explicitly approved directory handles. It shares the existing staging plan.
func (r *Registry) PlanDeskStageRoot(ctx context.Context, name, runner, instance, desk string, root *os.Root) (DeskStagePlan, error) {
	m, err := deskwork.SnapshotRoot(root)
	if err != nil {
		return DeskStagePlan{}, err
	}
	return r.planDeskStage(ctx, name, runner, instance, desk, m)
}
func (r *Registry) planDeskStage(ctx context.Context, name, runner, instance, desk string, m deskwork.Manifest) (DeskStagePlan, error) {
	var out DeskStagePlan
	if desk == "" {
		return out, errors.New("select a source desk")
	}
	setting, err := r.InspectDeskSetting(ctx, name, runner, instance, desk)
	if err != nil {
		return out, err
	}
	if setting.Setting.Desk.Workflow != "mounted" {
		return out, errors.New("follow Urgit's native source workflow")
	}
	var id [32]byte
	if _, err = rand.Read(id[:]); err != nil {
		return out, err
	}
	var size int64
	for _, f := range m.Files {
		size += f.Bytes
	}
	out = DeskStagePlan{Version: 1, RunnerID: setting.RunnerID, Mode: setting.Mode, Intent: deskwork.StageIntent{Version: 1, ID: hex.EncodeToString(id[:]), Target: setting.Setting.Target, SettingsRevision: setting.Setting.Revision, SourceDigest: m.Digest, Files: len(m.Files), Bytes: size}}
	if err = out.Intent.Check(m); err != nil {
		return DeskStagePlan{}, err
	}
	return out, out.Validate()
}
func (r *Registry) StageDesk(ctx context.Context, name, runner string, plan DeskStagePlan, directory string) (protocol.DeskStage, error) {
	if err := plan.Validate(); err != nil {
		return protocol.DeskStage{}, err
	}
	st, err := os.Lstat(directory)
	if err != nil {
		return protocol.DeskStage{}, err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return protocol.DeskStage{}, errors.New("source must be an ordinary directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return protocol.DeskStage{}, err
	}
	defer root.Close()
	return r.StageDeskRoot(ctx, name, runner, plan, root)
}

// StageDeskRoot transfers the unchanged plan from a pinned approved root. The
// runner remains authoritative for cache verification, policy and native effects.
func (r *Registry) StageDeskRoot(ctx context.Context, name, runner string, plan DeskStagePlan, root *os.Root) (protocol.DeskStage, error) {
	var out protocol.DeskStage
	if err := plan.Validate(); err != nil {
		return out, &DeskStageAdmissionError{Err: err}
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return out, &DeskStageAdmissionError{Err: err}
	}
	if err = entry.Identity().Check(plan.RunnerID, plan.Mode); err != nil {
		return out, &DeskStageAdmissionError{Err: err}
	}
	m, err := deskwork.SnapshotRoot(root)
	if err != nil {
		return out, &DeskStageAdmissionError{Err: err}
	}
	if err = plan.Intent.Check(m); err != nil {
		return out, &DeskStageAdmissionError{Err: err}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	reader, writer := io.Pipe()
	finished := make(chan struct{})
	go func() {
		err := deskwork.WriteBundleRoot(ctx, root, m, writer)
		_ = writer.CloseWithError(err)
		close(finished)
	}()
	c := entry.Client(true)
	defer c.Close()
	out, err = c.StageDesk(ctx, entry.Identity(), plan.Intent, reader)
	cancel()
	_ = reader.Close()
	<-finished
	if err != nil {
		return protocol.DeskStage{}, err
	}
	// A verified exact receipt can reconcile an earlier upload without reading
	// this stream. Only that server-side byte verification permits this case.
	return out, nil
}
func (r *Registry) DeskStage(ctx context.Context, name, runner, id string) (protocol.DeskStage, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.DeskStage{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.DeskStage(ctx, entry.Identity(), id)
}

func (r *Registry) DiscardDeskStage(ctx context.Context, name, runner, id, digest string) (protocol.DeskStageDiscard, error) {
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.DeskStageDiscard{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.DiscardDeskStage(ctx, entry.Identity(), id, digest)
}
