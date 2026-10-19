package hostinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

const ownerEngineActivationPath = "/etc/urbit-runner/owner-engine-activation.json"

// Only a new installation can create prepared intent. Publish it durably before
// owner.json, then complete it after private-socket/API verification and before
// management setup. An existing profile without this journal never authorizes
// automatic engine startup: it may be an established, deliberately stopped host.
type ownerEngineActivation struct {
	Version int               `json:"version"`
	Owner   ownerhost.Profile `json:"owner"`
	Stage   string            `json:"stage"`
}

func (i *installer) loadOwnerEngineActivation(ctx context.Context, p ownerhost.Profile, ownerExists bool) (*ownerEngineActivation, error) {
	data, err := i.readInstalled(ctx, ownerEngineActivationPath, 8192)
	if errors.Is(err, os.ErrNotExist) {
		if ownerExists {
			return nil, nil
		}
		return &ownerEngineActivation{Version: 1, Owner: p, Stage: "prepared"}, nil
	}
	if err != nil {
		return nil, err
	}
	var state ownerEngineActivation
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var extra any
	if decoder.Decode(&state) != nil || decoder.Decode(&extra) != io.EOF || state.Version != 1 || state.Owner != p || (state.Stage != "prepared" && state.Stage != "complete") {
		return nil, errors.New("initial engine activation evidence is invalid or belongs to another owner; preserving it")
	}
	if !ownerExists && state.Stage == "complete" {
		return nil, errors.New("completed engine activation lost its owner profile; explicit review is required")
	}
	if state.Stage == "prepared" {
		for _, path := range []string{managerUnit, activationJournalPath} {
			if _, err := i.lstat(i.path(path)); !errors.Is(err, os.ErrNotExist) {
				return nil, errors.New("initial engine activation conflicts with established management state; preserving stopped intent")
			}
		}
	}
	return &state, nil
}

func (i *installer) saveOwnerEngineActivation(ctx context.Context, state ownerEngineActivation, complete bool) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if !complete {
		// Also repeats the directory publication barrier when recovering a lost
		// sync acknowledgement for either known state.
		return i.installFile(ctx, ownerEngineActivationPath, data, 0600, false)
	}
	old, err := i.readInstalled(ctx, ownerEngineActivationPath, 8192)
	if err != nil || state.Stage != "prepared" || !bytes.Equal(old, data) {
		return errors.New("initial engine activation evidence changed before completion")
	}
	state.Stage = "complete"
	data, err = json.Marshal(state)
	if err != nil {
		return err
	}
	return i.installFile(ctx, ownerEngineActivationPath, append(data, '\n'), 0600, true)
}
