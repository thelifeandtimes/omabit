package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type AppPolicyRollout struct {
	Name             string                    `json:"name"`
	RunnerID         string                    `json:"runner_id"`
	ExpectedRevision *uint64                   `json:"expected_revision"`
	Applied          bool                      `json:"applied"`
	Error            string                    `json:"error,omitempty"`
	Acknowledged     *AppPolicyAcknowledgement `json:"acknowledged,omitempty"`
}
type AppPolicyAcknowledgement struct {
	Revision uint64          `json:"revision"`
	Global   apppolicy.Gates `json:"global"`
}
type AppPolicyGlobal struct {
	Version  int                `json:"version"`
	Revision uint64             `json:"revision"`
	Gates    apppolicy.Gates    `json:"gates"`
	Runners  []AppPolicyRollout `json:"runners"`
}
type AppPolicyRunner struct {
	Name         string                    `json:"name"`
	RunnerID     string                    `json:"runner_id"`
	CanEdit      bool                      `json:"can_edit"`
	Status       *protocol.AppPolicyStatus `json:"status,omitempty"`
	Error        string                    `json:"error,omitempty"`
	Rollout      string                    `json:"rollout"`
	Acknowledged *AppPolicyAcknowledgement `json:"acknowledged,omitempty"`
}
type AppDevelopmentStatus struct {
	Version int               `json:"version"`
	Global  AppPolicyGlobal   `json:"global"`
	Runners []AppPolicyRunner `json:"runners"`
}

func (r *Registry) readAppPolicyGlobal() (AppPolicyGlobal, error) {
	out := AppPolicyGlobal{Version: 2, Gates: apppolicy.Blocked(), Runners: []AppPolicyRollout{}}
	f, err := privateFile(filepath.Join(r.dir, "app-development.json"), os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxRegistryBytes+1))
	if err != nil || len(b) > maxRegistryBytes {
		return out, errors.New("cannot read bounded application policy rollout")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	// Decode into zero values so omitted legacy fields cannot inherit a default
	// that hides an incomplete v2 record.
	out = AppPolicyGlobal{}
	if d.Decode(&out) != nil || d.Decode(new(any)) != io.EOF {
		return out, errors.New("invalid application policy rollout; state preserved")
	}
	if out.Version == 1 {
		if out.Gates.CreateDesk != "" {
			return out, errors.New("unexpected legacy desk-creation permission")
		}
		out.Version = 2
		out.Gates.CreateDesk = apppolicy.Block
		for i := range out.Runners {
			if a := out.Runners[i].Acknowledged; a != nil {
				if a.Global.CreateDesk != "" {
					return out, errors.New("unexpected legacy desk-creation acknowledgment")
				}
				a.Global.CreateDesk = apppolicy.Block
			}
		}
	}
	if out.Version != 2 || out.Revision > apppolicy.MaxRevision || out.Gates.Validate(true) != nil || out.Runners == nil || len(out.Runners) > maxConnections {
		return out, errors.New("invalid application policy rollout; state preserved")
	}
	seen := map[string]bool{}
	for _, v := range out.Runners {
		if !protocol.ValidID(v.Name) || !protocol.ValidID(v.RunnerID) || seen[v.RunnerID] || v.ExpectedRevision != nil && *v.ExpectedRevision >= apppolicy.MaxRevision {
			return out, errors.New("invalid application policy rollout runner")
		}
		seen[v.RunnerID] = true
		if v.Acknowledged != nil && (v.Acknowledged.Revision > apppolicy.MaxRevision || v.Acknowledged.Global.Validate(true) != nil) {
			return out, errors.New("invalid acknowledged application policy")
		}
	}
	return out, nil
}

// AppDevelopment performs read-only observations. Refresh never dispatches a
// pending write, and newly enrolled runners are not silently opted into a grant.
func (r *Registry) AppDevelopment(ctx context.Context) (AppDevelopmentStatus, error) {
	out := AppDevelopmentStatus{Version: 1, Runners: []AppPolicyRunner{}}
	var entries []Connection
	err := r.locked(ctx, func() error {
		var err error
		out.Global, err = r.readAppPolicyGlobal()
		if err != nil {
			return err
		}
		state, err := r.read()
		entries = state.Connections
		return err
	})
	if err != nil {
		return out, err
	}
	out.Runners = make([]AppPolicyRunner, len(entries))
	var wg sync.WaitGroup
	slots := make(chan struct{}, 4)
	for i, entry := range entries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			row := AppPolicyRunner{Name: entry.Name, RunnerID: entry.RunnerID, Rollout: "not enrolled in global rollout"}
			defer func() { out.Runners[i] = row }()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				row.Error = "policy read cancelled"
				return
			}
			c := entry.Client(true)
			defer c.Close()
			readctx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			h, err := c.Hello(readctx)
			if err == nil {
				var status protocol.AppPolicyStatus
				status, err = c.AppPolicy(readctx, entry.Identity())
				if err == nil {
					row.Status = &status
					row.CanEdit = h.AccessRole == "operator"
				}
			}
			if err != nil {
				row.Error = err.Error()
			}
			for _, pending := range out.Global.Runners {
				if pending.RunnerID == entry.RunnerID {
					row.Acknowledged = pending.Acknowledged
					row.Rollout = "pending"
					if row.Status != nil && row.Status.Policy.Global == out.Global.Gates {
						row.Rollout = "applied"
					} else if pending.Applied {
						row.Rollout = "previously applied; current state unverified or changed"
					}
					if pending.Error != "" && row.Rollout != "applied" {
						row.Rollout += " — " + pending.Error
					}
				}
			}
		}()
	}
	wg.Wait()
	return out, nil
}

// SaveAppPolicyGlobal records intent before any network write. Expected runner
// revisions come from the user's reviewed Setup snapshot, including nil for an
// unavailable runner. A later retry cannot invent that missing review.
func (r *Registry) SaveAppPolicyGlobal(ctx context.Context, revision uint64, gates apppolicy.Gates, targets []AppPolicyRollout) (AppPolicyGlobal, error) {
	var out AppPolicyGlobal
	err := r.locked(ctx, func() error {
		current, err := r.readAppPolicyGlobal()
		if err != nil {
			return err
		}
		if current.Revision != revision || revision >= apppolicy.MaxRevision {
			return errors.New("global policy changed; refresh Setup")
		}
		if err = gates.Validate(true); err != nil {
			return err
		}
		registry, err := r.read()
		if err != nil {
			return err
		}
		if len(targets) != len(registry.Connections) {
			return errors.New("enrollment changed; refresh Setup")
		}
		seen := map[string]bool{}
		for i, t := range targets {
			found := false
			for _, c := range registry.Connections {
				if c.Name == t.Name && c.RunnerID == t.RunnerID {
					found = true
				}
			}
			if !found || seen[t.RunnerID] || t.ExpectedRevision != nil && *t.ExpectedRevision >= apppolicy.MaxRevision {
				return errors.New("invalid or changed rollout target")
			}
			seen[t.RunnerID] = true
			targets[i].Applied = false
			targets[i].Error = ""
			targets[i].Acknowledged = nil
			for _, previous := range current.Runners {
				if previous.RunnerID == t.RunnerID {
					targets[i].Acknowledged = previous.Acknowledged
				}
			}
		}
		if targets == nil {
			targets = []AppPolicyRollout{}
		}
		out = AppPolicyGlobal{Version: 2, Revision: revision + 1, Gates: gates, Runners: targets}
		return r.writeFile("app-development.json", out)
	})
	return out, err
}
func (r *Registry) ApplyAppPolicyGlobal(ctx context.Context, revision uint64) error {
	// Lock each dispatch and its receipt together. A newer local global edit
	// cannot overtake an older in-flight dispatch; another collector is fenced by
	// the runner's CAS. Lost responses retain the reviewed revision for retry.
	var targets []AppPolicyRollout
	if err := r.locked(ctx, func() error {
		g, err := r.readAppPolicyGlobal()
		if err != nil {
			return err
		}
		if g.Revision != revision {
			return errors.New("global policy changed; refresh Setup")
		}
		targets = g.Runners
		return nil
	}); err != nil {
		return err
	}
	for _, target := range targets {
		if err := r.locked(ctx, func() error {
			g, err := r.readAppPolicyGlobal()
			if err != nil {
				return err
			}
			if g.Revision != revision {
				return errors.New("global policy changed; refresh Setup")
			}
			for i, t := range g.Runners {
				if t.RunnerID != target.RunnerID {
					continue
				}
				if t.Applied {
					return nil
				}
				var acknowledged *AppPolicyAcknowledgement
				apply := func() error {
					registry, err := r.read()
					if err != nil {
						return err
					}
					var selected *Connection
					for _, c := range registry.Connections {
						if c.Name == t.Name && c.RunnerID == t.RunnerID {
							copy := c
							selected = &copy
						}
					}
					if selected == nil {
						return errors.New("enrollment changed; review and save global settings again")
					}
					c := selected.Client(true)
					defer c.Close()
					status, err := c.AppPolicy(ctx, selected.Identity())
					if err != nil {
						return err
					}
					if status.Policy.Global == g.Gates {
						acknowledged = &AppPolicyAcknowledgement{Revision: status.Policy.Revision, Global: status.Policy.Global}
						return nil
					}
					if t.ExpectedRevision == nil {
						return errors.New("runner was unavailable during review; refresh and save global settings again")
					}
					var saved apppolicy.Policy
					saved, err = c.SetAppPolicy(ctx, selected.Identity(), apppolicy.Update{ExpectedRevision: *t.ExpectedRevision, Scope: "global", Gates: g.Gates})
					if err == nil {
						acknowledged = &AppPolicyAcknowledgement{Revision: saved.Revision, Global: saved.Global}
					}
					return err
				}()
				g.Runners[i].Applied = apply == nil
				if acknowledged != nil {
					g.Runners[i].Acknowledged = acknowledged
				}
				g.Runners[i].Error = ""
				if apply != nil {
					g.Runners[i].Error = apply.Error()
				}
				return r.writeFile("app-development.json", g)
			}
			return errors.New("global rollout target changed")
		}); err != nil {
			return err
		}
	}
	return nil
}
func (r *Registry) SaveAppPolicyScope(ctx context.Context, name, runner string, u apppolicy.Update) error {
	if u.Scope == "global" {
		return errors.New("use the global rollout authoring path")
	}
	return r.locked(ctx, func() error {
		registry, err := r.read()
		if err != nil {
			return err
		}
		for _, entry := range registry.Connections {
			if entry.Name == name && entry.RunnerID == runner {
				c := entry.Client(true)
				defer c.Close()
				_, err = c.SetAppPolicy(ctx, entry.Identity(), u)
				return err
			}
		}
		return errors.New("runner enrollment changed; refresh Setup")
	})
}
