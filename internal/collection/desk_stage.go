package collection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type DeskStagePlan struct {
	Version  int                  `json:"version"`
	RunnerID string               `json:"runner_id"`
	Mode     string               `json:"mode"`
	Intent   deskwork.StageIntent `json:"intent"`
}

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
	var out protocol.DeskStage
	if err := plan.Validate(); err != nil {
		return out, err
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return out, err
	}
	if err = entry.Identity().Check(plan.RunnerID, plan.Mode); err != nil {
		return out, err
	}
	m, err := deskwork.Snapshot(directory)
	if err != nil {
		return out, err
	}
	if err = plan.Intent.Check(m); err != nil {
		return out, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	reader, writer := io.Pipe()
	finished := make(chan struct{})
	go func() {
		err := deskwork.WriteBundle(ctx, directory, m, writer)
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
