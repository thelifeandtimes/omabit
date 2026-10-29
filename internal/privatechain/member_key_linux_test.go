//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func memberKeyFixture(t *testing.T) (string, Plan, Workspace, Gateway, string, []byte) {
	t.Helper()
	directory, p, w, g, root := gatewayStateFixture(t)
	state, err := OpenGatewayStateVerifier(context.Background(), directory, p, w, g)
	if err != nil {
		t.Fatal(err)
	}
	o, err := state.Verify(context.Background())
	mustInput(t, err)
	mustInput(t, state.Close())
	observed := gatewayObservation(p, w, g)
	observed.Chain = o.Chain
	observed.Secrets = o.Secrets
	g.Phase = "ready"
	g.Observation = &observed
	mustInput(t, g.Validate(p, w))
	raw := []byte("0w0123456789abcdefghijk\n")
	for _, member := range p.Members {
		mustInput(t, os.WriteFile(filepath.Join(root, "secrets", strings.TrimPrefix(member.Ship, "~")+".key"), raw, 0600))
	}
	return directory, p, w, g, root, raw
}
func TestMemberKeyExactBytesAndReadOnlySelection(t *testing.T) {
	for _, index := range []int{0, 1} {
		t.Run(string(rune('0'+index)), func(t *testing.T) {
			directory, p, w, g, root, raw := memberKeyFixture(t)
			member := p.Members[index]
			peer := strings.TrimPrefix(p.Members[1-index].Ship, "~") + ".key"
			v, err := OpenMemberKeyVerifier(context.Background(), directory, p, w, g, member.InstanceID, inputSHA(raw))
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			before := workspaceFiles(t, directory)
			p.Members[index].Ship = "~bud"
			w.Root.Inode++
			g.Observation.Secrets.Inode++
			for range 2 {
				if err = v.Verify(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
				t.Fatal("reader changed state")
			}
			// The other member's file is outside this verifier's selection.
			mustInput(t, os.WriteFile(filepath.Join(root, "secrets", peer), []byte("other synthetic bytes"), 0600))
			if err = v.Verify(context.Background()); err != nil {
				t.Fatal("unselected key was inspected", err)
			}
			mustInput(t, v.Close())
			mustInput(t, v.Close())
			if v.Verify(context.Background()) == nil {
				t.Fatal("closed verifier accepted")
			}
		})
	}
}
func TestMemberKeyRefusesUnboundInputs(t *testing.T) {
	for _, kind := range []string{"member", "phase", "project", "secret-identity", "hash-empty", "hash-zero", "hash-other", "hash-uppercase"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w, g, _, raw := memberKeyFixture(t)
			member := p.Members[0].InstanceID
			digest := inputSHA(raw)
			switch kind {
			case "member":
				member = "unselected"
			case "phase":
				g.Phase = "started"
				g.Observation = nil
			case "project":
				g.Selection.Binding.ProjectID = "private-project-other"
			case "secret-identity":
				g.Observation.Secrets.Inode++
			case "hash-empty":
				digest = ""
			case "hash-zero":
				digest = strings.Repeat("0", 64)
			case "hash-other":
				digest = strings.Repeat("e", 64)
			case "hash-uppercase":
				digest = strings.ToUpper(digest)
			}
			if v, err := OpenMemberKeyVerifier(context.Background(), directory, p, w, g, member, digest); err == nil {
				v.Close()
				t.Fatal("unbound key accepted")
			}
		})
	}
}
func TestMemberKeyRefusesUnsafeFilesWithoutRepair(t *testing.T) {
	for _, kind := range []string{"missing", "empty", "large", "bad-format", "double-newline", "crlf", "symlink", "hardlink", "fifo", "directory", "public", "writable", "symlink-parent"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w, g, root, raw := memberKeyFixture(t)
			member := p.Members[0]
			key := filepath.Join(root, "secrets", strings.TrimPrefix(member.Ship, "~")+".key")
			expected := inputSHA(raw)
			switch kind {
			case "missing":
				mustInput(t, os.Remove(key))
			case "empty":
				mustInput(t, os.Truncate(key, 0))
			case "large":
				mustInput(t, os.Truncate(key, 4096))
			case "bad-format", "double-newline", "crlf":
				bad := []byte("private-injection synthetic payload")
				if kind == "double-newline" {
					bad = append(raw, '\n')
				}
				if kind == "crlf" {
					bad = append(append([]byte{}, raw[:len(raw)-1]...), '\r', '\n')
				}
				mustInput(t, os.WriteFile(key, bad, 0600))
				expected = inputSHA(bad)
			case "symlink":
				mustInput(t, os.Remove(key))
				mustInput(t, os.Symlink("nec.key", key))
			case "hardlink":
				mustInput(t, os.Link(key, filepath.Join(root, "home/alias")))
			case "fifo":
				mustInput(t, os.Remove(key))
				mustInput(t, unix.Mkfifo(key, 0600))
			case "directory":
				mustInput(t, os.Remove(key))
				mustInput(t, os.Mkdir(key, 0700))
			case "public":
				mustInput(t, os.Chmod(key, 0644))
			case "writable":
				mustInput(t, os.Chmod(key, 0620))
			case "symlink-parent":
				alias := filepath.Join(filepath.Dir(directory), "alias")
				mustInput(t, os.Mkdir(alias, 0700))
				link := filepath.Join(alias, "private-projects")
				mustInput(t, os.Symlink(directory, link))
				directory = link
			}
			v, err := OpenMemberKeyVerifier(context.Background(), directory, p, w, g, member.InstanceID, expected)
			if err == nil {
				v.Close()
				t.Fatal("unsafe key accepted")
			}
			if strings.Contains(err.Error(), "private-injection") || strings.Contains(err.Error(), string(raw)) || strings.Contains(err.Error(), key) || strings.Contains(err.Error(), expected) {
				t.Fatal("private data exported")
			}
			if kind != "missing" {
				if _, err = os.Lstat(key); err != nil {
					t.Fatal("refusal removed file")
				}
			}
		})
	}
}
func TestMemberKeyRejectsReplacementAndMutation(t *testing.T) {
	for _, kind := range []string{"parent", "root", "secrets", "key", "in-place", "metadata"} {
		t.Run(kind, func(t *testing.T) {
			directory, p, w, g, root, raw := memberKeyFixture(t)
			member := p.Members[0]
			key := filepath.Join(root, "secrets", strings.TrimPrefix(member.Ship, "~")+".key")
			v, err := OpenMemberKeyVerifier(context.Background(), directory, p, w, g, member.InstanceID, inputSHA(raw))
			if err != nil {
				t.Fatal(err)
			}
			defer v.Close()
			if kind == "in-place" {
				bad := append([]byte{}, raw...)
				bad[4] = 'z'
				mustInput(t, os.WriteFile(key, bad, 0600))
			} else if kind == "metadata" {
				mustInput(t, os.Chtimes(key, time.Unix(10, 0), time.Unix(10, 0)))
			} else {
				target := map[string]string{"parent": directory, "root": root, "secrets": filepath.Join(root, "secrets"), "key": key}[kind]
				retained := filepath.Join(filepath.Dir(directory), "retained-"+kind)
				mustInput(t, os.Rename(target, retained))
				if kind == "key" {
					mustInput(t, os.WriteFile(target, raw, 0600))
				} else {
					mustInput(t, os.Mkdir(target, 0700))
					entries, err := os.ReadDir(retained)
					mustInput(t, err)
					for _, entry := range entries {
						mustInput(t, os.Rename(filepath.Join(retained, entry.Name()), filepath.Join(target, entry.Name())))
					}
				}
			}
			before := workspaceFiles(t, directory)
			if v.Verify(context.Background()) == nil {
				t.Fatal("changed key storage accepted")
			}
			if !reflect.DeepEqual(before, workspaceFiles(t, directory)) {
				t.Fatal("refusal changed storage")
			}
		})
	}
}
func TestMemberKeyCancellationAndConcurrentClose(t *testing.T) {
	directory, p, w, g, _, raw := memberKeyFixture(t)
	member := p.Members[0].InstanceID
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenMemberKeyVerifier(ctx, directory, p, w, g, member, inputSHA(raw)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	v, err := OpenMemberKeyVerifier(context.Background(), directory, p, w, g, member, inputSHA(raw))
	if err != nil {
		t.Fatal(err)
	}
	if err = v.Verify(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() { defer group.Done(); _ = v.Verify(context.Background()); _ = v.Close() }()
	}
	group.Wait()
	if v.Verify(context.Background()) == nil {
		t.Fatal("closed reader accepted")
	}
}
