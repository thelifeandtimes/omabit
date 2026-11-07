package deskwork

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
)

// Ownership pairs raw source with the canonical Clay files verified after the
// last deployment. It includes only paths this source binding actually owns.
// Shared identical files and unspecified ship-generated files are never adopted.
type Ownership struct {
	Version int      `json:"version"`
	Raw     Manifest `json:"raw"`
	Clay    Manifest `json:"clay"`
}

func EmptyOwnership() Ownership { m, _ := NewManifest([]File{}); return Ownership{1, m, m} }
func (o Ownership) Validate() error {
	if o.Version != 1 || o.Raw.Validate() != nil || !canonicalManifest(o.Clay) || len(o.Raw.Files) != len(o.Clay.Files) {
		return errors.New("invalid source ownership")
	}
	for i, f := range o.Raw.Files {
		if f.Path != o.Clay.Files[i].Path {
			return errors.New("source ownership path mismatch")
		}
	}
	return nil
}

// DeploymentPlan contains only reviewed hashes and paths, never file bytes.
// Known contains source normalized with the baseline desk's mark converters.
// It may omit new marks; unknown conversion is never treated as equality.
// Before in Changes is a canonical Clay file; After is an exact raw source file.
type DeploymentPlan struct {
	Version   int       `json:"version"`
	Previous  Ownership `json:"previous"`
	Current   Manifest  `json:"current"`
	Source    Manifest  `json:"source"`
	Known     Manifest  `json:"known"`
	Changes   []Change  `json:"changes"`
	Preserved []string  `json:"preserved"`
	Conflicts []string  `json:"conflicts"`
	Managed   []string  `json:"managed"`
	Digest    string    `json:"digest"`
}

func deploymentDigest(p DeploymentPlan) string {
	p.Digest = ""
	data, _ := json.Marshal(p)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// PlanDeployment protects unspecified Clay files and refuses changed/deleted
// owned files or colliding unowned files. Equality established through the
// selected desk's mark conversion permits shared files without adopting them.
func PlanDeployment(previous Ownership, current, source, known Manifest) (DeploymentPlan, error) {
	var out DeploymentPlan
	if previous.Validate() != nil || !canonicalManifest(current) || source.Validate() != nil || !canonicalManifest(known) {
		return out, errors.New("invalid deployment inventory")
	}
	now, next, old, prior, normalized := current.Index(), source.Index(), previous.Raw.Index(), previous.Clay.Index(), known.Index()
	for name := range normalized {
		if _, ok := next[name]; !ok {
			return out, errors.New("normalized file is outside staged source")
		}
	}
	out = DeploymentPlan{Version: 1, Previous: previous, Current: current, Source: source, Known: known, Changes: []Change{}, Preserved: []string{}, Conflicts: []string{}, Managed: []string{}}
	names := map[string]bool{}
	for name := range old {
		names[name] = true
	}
	for name := range next {
		names[name] = true
	}
	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)
	var changedBytes int64
	for _, name := range ordered {
		before, exists := now[name]
		after, wanted := next[name]
		oldRaw, owned := old[name]
		normalizedAfter, knownAfter := normalized[name]
		if wanted && exists && knownAfter && normalizedAfter == before {
			if owned {
				out.Managed = append(out.Managed, name)
			} else {
				out.Preserved = append(out.Preserved, name)
			}
			continue
		}
		if !wanted && !exists {
			continue
		}
		if owned && (!exists || before != prior[name]) || !owned && exists {
			out.Conflicts = append(out.Conflicts, name)
			continue
		}
		if wanted && owned && after == oldRaw && !knownAfter {
			out.Managed = append(out.Managed, name)
			continue
		}
		collision := false
		if wanted {
			for n := range now {
				if n != name && (strings.HasPrefix(n, name+"/") || strings.HasPrefix(name, n+"/")) {
					collision = true
					break
				}
			}
		}
		if collision {
			out.Conflicts = append(out.Conflicts, name)
			continue
		}
		c := Change{Path: name}
		if exists {
			v := before
			c.Before = &v
		}
		if wanted {
			v := after
			c.After = &v
			changedBytes += after.Bytes
			out.Managed = append(out.Managed, name)
		}
		switch {
		case !wanted:
			c.Kind = "delete"
		case exists:
			c.Kind = "replace"
		default:
			c.Kind = "add"
		}
		out.Changes = append(out.Changes, c)
	}
	if changedBytes > MaxFileBytes {
		return DeploymentPlan{}, errors.New("single Clay commit exceeds 32 MiB of changed source")
	}
	if len(out.Changes) > MaxFiles {
		return DeploymentPlan{}, errors.New("single Clay commit exceeds file limit")
	}
	// List all untouched unowned files for review, including generated content
	// absent from the supplied source. A collision is not reported as preserved.
	preserve := map[string]bool{}
	for _, n := range out.Preserved {
		preserve[n] = true
	}
	conflicts := map[string]bool{}
	for _, n := range out.Conflicts {
		conflicts[n] = true
	}
	for n := range now {
		if _, owned := old[n]; !owned && !conflicts[n] {
			preserve[n] = true
		}
	}
	out.Preserved = out.Preserved[:0]
	for n := range preserve {
		out.Preserved = append(out.Preserved, n)
	}
	sort.Strings(out.Preserved)
	if len(out.Conflicts) == 0 {
		result := current.Index()
		for _, c := range out.Changes {
			if c.Kind == "delete" {
				delete(result, c.Path)
			} else {
				v := *c.After
				v.Bytes = 0
				result[c.Path] = v
			}
		}
		files := make([]File, 0, len(result))
		for _, f := range result {
			files = append(files, f)
		}
		if _, err := NewManifest(files); err != nil {
			return DeploymentPlan{}, errors.New("resulting Clay tree exceeds supported bounds")
		}
	}
	out.Digest = deploymentDigest(out)
	return out, nil
}
func (p DeploymentPlan) Validate() error {
	if p.Version != 1 || !digestPattern.MatchString(p.Digest) {
		return errors.New("invalid deployment plan")
	}
	expected, err := PlanDeployment(p.Previous, p.Current, p.Source, p.Known)
	if err != nil {
		return err
	}
	a, _ := json.Marshal(p)
	b, _ := json.Marshal(expected)
	if !bytes.Equal(a, b) {
		return errors.New("deployment plan changed")
	}
	return nil
}

// DeploymentFiles loads only the exact reviewed changes. Callers must journal
// dispatch and enforce policy/runtime/revision bindings before sending the delta.
func DeploymentFiles(ctx context.Context, directory string, p DeploymentPlan) ([]DeltaFile, error) {
	if p.Validate() != nil || len(p.Conflicts) != 0 {
		return nil, errors.New("deployment has unresolved conflicts")
	}
	before, err := Snapshot(directory)
	if err != nil || before.Digest != p.Source.Digest {
		return nil, errors.New("staged deployment source changed")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	out := make([]DeltaFile, 0, len(p.Changes))
	for _, change := range p.Changes {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		f := DeltaFile{Path: change.Path, Delete: change.Kind == "delete"}
		if !f.Delete {
			f.Bytes, err = ReadSourceFile(root, *change.After)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, f)
	}
	after, err := Snapshot(directory)
	if err != nil || after.Digest != p.Source.Digest {
		return nil, errors.New("staged deployment source changed")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// VerifyDeployment checks the entire resulting Clay tree and every requested
// source file after conversion with the resulting revision's marks. Neither a
// new Clay revision nor an accepted HTTP response alone establishes completion.
// Compilation, activation and publication are independent subsequent stages.
func VerifyDeployment(p DeploymentPlan, result, normalizedSource Manifest) (Ownership, error) {
	if p.Validate() != nil || len(p.Conflicts) != 0 || !canonicalManifest(result) || !canonicalManifest(normalizedSource) || len(normalizedSource.Files) != len(p.Source.Files) {
		return Ownership{}, errors.New("unverified deployment result")
	}
	for i, f := range normalizedSource.Files {
		if f.Path != p.Source.Files[i].Path {
			return Ownership{}, errors.New("normalized deployment selection changed")
		}
	}
	expected := p.Current.Index()
	for _, change := range p.Changes {
		if change.Kind == "delete" {
			delete(expected, change.Path)
		}
	}
	for _, f := range normalizedSource.Files {
		expected[f.Path] = f
	}
	files := make([]File, 0, len(expected))
	for _, f := range expected {
		files = append(files, f)
	}
	want, err := NewManifest(files)
	if err != nil || want.Digest != result.Digest {
		return Ownership{}, errors.New("Clay differs from reviewed deployment or preserved files changed")
	}
	raw, clay := p.Source.Index(), result.Index()
	managedRaw, managedClay := []File{}, []File{}
	for _, name := range p.Managed {
		managedRaw = append(managedRaw, raw[name])
		managedClay = append(managedClay, clay[name])
	}
	a, err := NewManifest(managedRaw)
	if err != nil {
		return Ownership{}, err
	}
	b, err := NewManifest(managedClay)
	if err != nil {
		return Ownership{}, err
	}
	ownership := Ownership{1, a, b}
	return ownership, ownership.Validate()
}
