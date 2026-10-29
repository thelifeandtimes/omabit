//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"
)

const projectMountFormat = `{"id":{{json .Id}},"mounts":{{json .Mounts}}}`

type projectContainerMounts struct {
	ID     string  `json:"id"`
	Mounts []mount `json:"mounts"`
}

// mountInventory includes stopped/created containers: either can later start
// with its saved bindings. It reads no application environment or credentials.
func (e *projectAnchorEngine) mountInventory(ctx context.Context) ([]projectContainerMounts, error) {
	ids, err := e.listIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errUntrustedContainer
	}
	args := append([]string{"container", "inspect", "--format", projectMountFormat}, ids...)
	q := e.base.q
	out, err := q.docker(ctx, q.d.times.command, args...)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(out))
	decoder.DisallowUnknownFields()
	var values []projectContainerMounts
	for range ids {
		var row struct {
			ID     string          `json:"id"`
			Mounts json.RawMessage `json:"mounts"`
		}
		if decoder.Decode(&row) != nil || len(row.Mounts) == 0 {
			return nil, errUntrustedContainer
		}
		v := projectContainerMounts{ID: row.ID}
		if json.Unmarshal(row.Mounts, &v.Mounts) != nil || len(v.Mounts) > 128 {
			return nil, errUntrustedContainer
		}
		values = append(values, v)
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, errUntrustedContainer
	}
	slices.SortFunc(values, func(a, b projectContainerMounts) int { return strings.Compare(a.ID, b.ID) })
	for i, id := range ids {
		if values[i].ID != id {
			return nil, errUntrustedContainer
		}
	}
	after, err := e.listIDs(ctx)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(ids, after) {
		return nil, errors.New("container mount inventory changed during observation")
	}
	return values, nil
}

type projectPathObject struct {
	path string
	info os.FileInfo
}

func projectPathObjects(ctx context.Context, path string) ([]projectPathObject, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\r\n") {
		return nil, errors.New("unresolved container mount source")
	}
	var objects []projectPathObject
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, errors.New("container mount source identity unavailable")
		}
		objects = append(objects, projectPathObject{path, info})
		if path == "/" {
			return objects, nil
		}
		path = filepath.Dir(path)
	}
}
func projectPathsOverlap(a, b []projectPathObject) bool {
	if len(a) == 0 || len(b) == 0 {
		return true
	}
	for _, p := range a {
		if os.SameFile(p.info, b[0].info) {
			return true
		}
	}
	for _, p := range b {
		if os.SameFile(p.info, a[0].info) {
			return true
		}
	}
	return false
}
func sameProjectPathObjects(a, b []projectPathObject) bool {
	if len(a) != len(b) {
		return false
	}
	for i, p := range a {
		if p.path != b[i].path || p.info.Mode() != b[i].info.Mode() || !os.SameFile(p.info, b[i].info) {
			return false
		}
	}
	return true
}

// rejectConflictingProjectMounts is a negative preflight, not launch authority.
// It rejects saved mount overlaps and unresolved sources using current filesystem
// objects, including symlink aliases and ancestor binds. It cannot establish the
// identity of a running container's historical kernel mount after its host path
// was moved/replaced. Lifecycle ownership and pier locks remain separate gates.
// Unknown volume/plugin sources are deliberately unresolved; do not infer safety
// from Docker's local-looking volume mountpoint. Supporting those requires a
// separately qualified volume-driver/source contract.
// The only exception is this dispatch's exact never-started member container,
// independently validated before and after collection. No names/labels are adopted.
func (e *projectAnchorEngine) rejectConflictingProjectMounts(ctx context.Context, c projectMemberContract, createdID string) error {
	fail := func() error {
		return errors.New("project member storage has conflicting or unresolved container mounts")
	}
	if !reflect.DeepEqual(c.selection.Project, e.base.contract.Selection) || !reflect.DeepEqual(c.selection.Anchor.Prepared, e.prepared) || c.home != e.base.q.d.home || c.imageID != e.base.contract.ImageID || createdID != "" && !containerIDPattern.MatchString(createdID) {
		return fail()
	}
	if err := c.checkStorageBeforeCreate(ctx); err != nil {
		return err
	}
	if err := e.VerifyEngine(ctx, e.prepared.EngineID); err != nil {
		return err
	}
	at := time.Now()
	inspectOwn := func() error {
		if createdID == "" {
			return nil
		}
		q := e.base.q
		raw, err := q.docker(ctx, q.d.times.command, "container", "inspect", createdID)
		if err != nil {
			return err
		}
		var rows []container
		if json.Unmarshal([]byte(raw), &rows) != nil || len(rows) != 1 {
			return errUntrustedContainer
		}
		return c.validateContainer(rows[0], createdID, false)
	}
	if err := inspectOwn(); err != nil {
		return err
	}
	inventory, err := e.mountInventory(ctx)
	if err != nil {
		return err
	}
	target, err := projectPathObjects(ctx, c.mountSource)
	if err != nil {
		return err
	}
	sources := map[string][]projectPathObject{}
	foundOwn := createdID == ""
	for _, v := range inventory {
		if v.ID == createdID {
			foundOwn = true
			want := c.mounts()
			seen := map[string]bool{}
			if len(v.Mounts) != len(want) {
				return fail()
			}
			for _, m := range v.Mounts {
				if seen[m.Destination] || !slices.Contains(want, m) {
					return fail()
				}
				seen[m.Destination] = true
			}
			continue
		}
		for _, m := range v.Mounts {
			switch m.Type {
			case "tmpfs":
				if m.Source != "" {
					return fail()
				}
				continue
			case "bind":
			default:
				return fail()
			}
			if _, ok := sources[m.Source]; ok {
				continue
			}
			if e.base.q.consumedBootKeyMount(m) {
				continue
			}
			objects, err := projectPathObjects(ctx, m.Source)
			if err != nil {
				return err
			}
			if projectPathsOverlap(target, objects) {
				return fail()
			}
			sources[m.Source] = objects
		}
	}
	if !foundOwn {
		return fail()
	}
	// Re-read both metadata and filesystem objects; a renamed/replaced source
	// during this observation is uncertainty, even if both paths look unrelated.
	after, err := e.mountInventory(ctx)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(inventory, after) {
		return fail()
	}
	again, err := projectPathObjects(ctx, c.mountSource)
	if err != nil {
		return err
	}
	if !sameProjectPathObjects(target, again) {
		return fail()
	}
	for source, before := range sources {
		after, err := projectPathObjects(ctx, source)
		if err != nil {
			return err
		}
		if !sameProjectPathObjects(before, after) {
			return fail()
		}
	}
	if err = inspectOwn(); err != nil {
		return err
	}
	if err = c.checkStorageBeforeCreate(ctx); err != nil {
		return err
	}
	if err = e.VerifyEngine(ctx, e.prepared.EngineID); err != nil {
		return err
	}
	if time.Since(at) > 30*time.Second {
		return fail()
	}
	return ctx.Err()
}

// Successful keyed boot removes its one-shot host key. Docker retains the
// read-only file mount until that container exits. It cannot be a pier writer.
// Only our private keys directory and exact key-file contract get this exception.
func (q *qualifier) consumedBootKeyMount(m mount) bool {
	if m.Type != "bind" || m.RW || m.Destination != "/artifacts/boot.key" || m.Propagation != "rprivate" || filepath.Clean(m.Source) != m.Source || filepath.Dir(m.Source) != filepath.Join(q.base, "keys") {
		return false
	}
	name := strings.TrimSuffix(filepath.Base(m.Source), ".key")
	if !strings.HasSuffix(m.Source, ".key") || !projectRuntimeID.MatchString(name) || q.owned(filepath.Dir(m.Source), os.ModeDir, true) != nil {
		return false
	}
	_, err := os.Lstat(m.Source)
	return errors.Is(err, os.ErrNotExist)
}
