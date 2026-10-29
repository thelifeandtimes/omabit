//go:build linux && amd64

package managed

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// Own token; the common guard admits only fresh no-NIC qualification guests.
func TestProjectLivePeerDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, false, false, false)
}

type vmLivePeerInspection struct {
	ID           string `json:"Id"`
	Name         string
	Config       map[string]any
	HostConfig   map[string]any
	RestartCount int
	State        struct {
		Running, OOMKilled bool
		Pid, ExitCode      int
		StartedAt, Status  string
	}
}

func vmCheckUnchangedLivePeer(original, current json.RawMessage) error {
	var a, b []vmLivePeerInspection
	if json.Unmarshal(original, &a) != nil || json.Unmarshal(current, &b) != nil || len(a) != 1 || len(b) != 1 {
		return errors.New("running peer inspection invalid")
	}
	x := a[0]
	if !x.State.Running || x.State.OOMKilled || x.State.Pid <= 0 || x.State.ExitCode != 0 || x.State.Status != "running" || x.State.StartedAt == "" || x.State.StartedAt == "0001-01-01T00:00:00Z" || !reflect.DeepEqual(x, b[0]) {
		return errors.New("running peer identity or process changed")
	}
	return nil
}

func vmLivePeerObserver(t *testing.T, members []map[string]any, docker func(...string) json.RawMessage) func() error {
	t.Helper()
	var peer map[string]any
	for _, member := range members {
		if member["instance_id"] == "vm-nec" {
			peer = member
		}
	}
	if peer == nil {
		t.Fatal("running peer missing")
	}
	job, ok := peer["journal"].(projectMemberLaunchJob)
	if !ok || job.Launch == nil || job.Launch.Container.ID == "" {
		t.Fatal("running peer launch missing")
	}
	original, ok := peer["container_inspection"].(json.RawMessage)
	if !ok {
		t.Fatal("running peer initial inspection missing")
	}
	var rows []vmLivePeerInspection
	if json.Unmarshal(original, &rows) != nil || len(rows) != 1 || rows[0].ID != job.Launch.Container.ID || rows[0].HostConfig["NetworkMode"] != "container:"+job.Launch.Anchor.Prepared.Anchor.ID {
		t.Fatal("running peer initial resource binding invalid")
	}
	peer["live_peer_observations"] = []json.RawMessage{}
	// Final shutdown is exact-ID fixture cleanup. This does not claim that the
	// original peer's stale readiness can authorize a Stop after another member
	// changed generation; that is a separate current-context contract.
	peer["fixture_cleanup_only"] = true
	return func() error {
		current := docker("container", "inspect", job.Launch.Container.ID)
		if err := vmCheckUnchangedLivePeer(original, current); err != nil {
			return err
		}
		peer["live_peer_observations"] = append(peer["live_peer_observations"].([]json.RawMessage), append(json.RawMessage(nil), current...))
		return nil
	}
}

func TestVMLivePeerRejectsProcessOrResourceReplacement(t *testing.T) {
	original := json.RawMessage(`[{"Id":"same-container","Name":"/fresh-peer","Config":{"Labels":{"owned":"yes"}},"HostConfig":{"NetworkMode":"container:anchor"},"RestartCount":0,"State":{"Running":true,"OOMKilled":false,"Pid":123,"ExitCode":0,"StartedAt":"2026-10-23T12:00:00Z","Status":"running"}}]`)
	if err := vmCheckUnchangedLivePeer(original, original); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*vmLivePeerInspection){
		"container": func(v *vmLivePeerInspection) { v.ID = "replacement" },
		"pid":       func(v *vmLivePeerInspection) { v.State.Pid++ },
		"start":     func(v *vmLivePeerInspection) { v.State.StartedAt = "2026-10-23T13:00:00Z" },
		"exit":      func(v *vmLivePeerInspection) { v.State.Running = false; v.State.Status = "exited" },
		"oom":       func(v *vmLivePeerInspection) { v.State.OOMKilled = true },
		"restart":   func(v *vmLivePeerInspection) { v.RestartCount++ },
		"owner":     func(v *vmLivePeerInspection) { v.Config["Labels"] = map[string]any{} },
		"namespace": func(v *vmLivePeerInspection) { v.HostConfig["NetworkMode"] = "container:other" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var rows []vmLivePeerInspection
			if err := json.Unmarshal(original, &rows); err != nil {
				t.Fatal(err)
			}
			mutate(&rows[0])
			changed, err := json.Marshal(rows)
			if err != nil {
				t.Fatal(err)
			}
			if vmCheckUnchangedLivePeer(original, changed) == nil {
				t.Fatal("changed peer accepted")
			}
		})
	}
	if vmCheckUnchangedLivePeer(original, json.RawMessage(`[]`)) == nil {
		t.Fatal("absent peer accepted")
	}
}
