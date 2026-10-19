//go:build linux

package hostcompat

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestAPTSandboxIdentityCredentialsAndScope(t *testing.T) {
	m := fixtureFS(map[string]string{"etc/passwd": "root:x:0:0::/root:/bin/sh\n_apt:x:42:65534::/nonexistent:/usr/sbin/nologin\n"})
	uid, gid, err := aptDownloadIdentity(m)
	if err != nil || uid != 42 || gid != 65534 {
		t.Fatal("standard existing account unresolved", err)
	}
	p := &downloadPolicy{stage: "var/lib/omabit/hostcompat/.fetch-test", aptUID: uid, aptGID: gid, aptReady: true}
	c := aptCredential(p)
	if c.Uid == 0 || c.Uid != uid || c.Gid != gid || len(c.Groups) != 0 || c.Groups == nil || c.NoSetGroups {
		t.Fatal("APT subprocess would retain root/supplementary credentials")
	}
	addFixture(m, "sample", []byte("download"), 0600)
	m["sample"].Sys = &syscall.Stat_t{Uid: uid, Gid: gid}
	info, _ := fs.Stat(m, "sample")
	if !p.allows(p.stage+"/apt/archives/pkg.deb", info) || p.allows(p.stage+"/apt/apt.conf", info) || p.allows(p.stage+"/pacman/archives/pkg", info) || p.allows(p.stage+"/apt-evidence/pkg", info) {
		t.Fatal("download UID permission escaped designated subtrees")
	}
	m["sample"].Sys = &syscall.Stat_t{Uid: uid + 1, Gid: gid}
	info, _ = fs.Stat(m, "sample")
	if p.allows(p.stage+"/apt/archives/pkg", info) {
		t.Fatal("arbitrary untrusted owner accepted")
	}
	for _, passwd := range []string{"root:x:0:0::/root:/bin/sh\n", "_apt:x:0:65534::/:/bin/false\n", "_apt:x:42:0::/:/bin/false\n", "_apt:x:42:65534::/:/bin/false\n_apt:x:43:65534::/:/bin/false\n"} {
		m["etc/passwd"].Data = []byte(passwd)
		if _, _, err := aptDownloadIdentity(m); err != errAPTSandboxUnavailable {
			t.Fatal("unsafe or missing standard helper accepted")
		}
	}
	config := aptPrivateConfig(p.stage+"/apt", "amd64")
	if strings.Contains(config, `APT::Sandbox::User "root"`) || !strings.Contains(config, `APT::Sandbox::User "_apt"`) {
		t.Fatal("native APT sandbox disabled")
	}
}

// Temp-root ownership is injected only for protected paths; the acquisition
// subtree retains the real test UID and real kernel inode/ctime information.
type acquisitionFS struct {
	fs.FS
	p *downloadPolicy
}

func (f acquisitionFS) Stat(name string) (fs.FileInfo, error) {
	i, e := fs.Stat(f.FS, name)
	if e != nil {
		return nil, e
	}
	if !f.p.aptArea(name) {
		return rootInfo{i}, nil
	}
	return i, nil
}
func (f acquisitionFS) Lstat(name string) (fs.FileInfo, error) {
	i, e := fs.Lstat(f.FS, name)
	if e != nil {
		return nil, e
	}
	if !f.p.aptArea(name) {
		return rootInfo{i}, nil
	}
	return i, nil
}
func (f acquisitionFS) ReadLink(name string) (string, error) { return fs.ReadLink(f.FS, name) }
func (f acquisitionFS) Open(name string) (fs.File, error) {
	file, e := f.FS.Open(name)
	if e != nil {
		return nil, e
	}
	if !f.p.aptArea(name) {
		return &racingFile{File: file}, nil
	}
	return file, nil
}

type acquisitionRoot struct {
	*os.Root
	p      *downloadPolicy
	mutate func(string)
}

func (r acquisitionRoot) FS() fs.FS { return acquisitionFS{r.Root.FS(), r.p} }
func (r acquisitionRoot) Lstat(name string) (fs.FileInfo, error) {
	if r.mutate != nil {
		r.mutate(name)
	}
	return fs.Lstat(r.FS(), name)
}

func TestDownloadPromotionCopiesStableBytesAndRejectsAliasesRaces(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("unprivileged acquisition fixture uses the ordinary test UID")
	}
	dir := t.TempDir()
	stage := "var/lib/omabit/hostcompat/.fetch-promote"
	for _, name := range []string{stage + "/apt/archives", stage + "/apt/lists", stage + "/apt-evidence/archives"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	p := &downloadPolicy{stage: stage, aptUID: uint32(os.Geteuid()), aptGID: uint32(os.Getegid()), aptReady: true}
	r := acquisitionRoot{Root: root, p: p}
	source := stage + "/apt/archives/pkg.deb"
	destination := stage + "/apt-evidence/archives/pkg.deb"
	if err := root.WriteFile(source, []byte("untrusted downloaded bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readTrusted(r.FS(), source, maxArtifact); err == nil {
		t.Fatal("untrusted download was treated as root-verified evidence in place")
	}
	if err := promoteDownload(context.Background(), r, p, source, destination, maxArtifact, r.FS()); err != nil {
		t.Fatal("stable copy failed", err)
	}
	data, err := readTrusted(r.FS(), destination, maxArtifact)
	if err != nil || string(data) != "untrusted downloaded bytes" {
		t.Fatal("sealed bytes differ")
	}
	if err := root.WriteFile(source, []byte("changed after promotion"), 0600); err != nil {
		t.Fatal(err)
	}
	data, _ = readTrusted(r.FS(), destination, maxArtifact)
	if string(data) != "untrusted downloaded bytes" {
		t.Fatal("sealing retained the downloader's mutable object")
	}
	root.Remove(destination)
	root.Remove(source)
	if err := root.Symlink("../../../../../../outside", source); err != nil {
		t.Fatal(err)
	}
	if err := promoteDownload(context.Background(), r, p, source, destination, maxArtifact, r.FS()); err == nil {
		t.Fatal("download symlink followed")
	}
	root.Remove(source)
	if err := root.WriteFile(source, []byte("race original"), 0600); err != nil {
		t.Fatal(err)
	}
	changed := false
	r.mutate = func(name string) {
		if name == source && !changed {
			changed = true
			if err := root.WriteFile(source, []byte("race modified"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := promoteDownload(context.Background(), r, p, source, destination, maxArtifact, r.FS()); err == nil {
		t.Fatal("mutation during promotion accepted")
	}
	if _, err := root.Lstat(destination); !os.IsNotExist(err) {
		t.Fatal("failed promotion published data")
	}
	r.mutate = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := promoteDownload(ctx, r, p, source, destination, maxArtifact, r.FS()); err == nil {
		t.Fatal("canceled acquisition published")
	}
	if stageWithinBoundsPolicy(r, stage, p) != nil {
		t.Fatal("expected acquisition UID rejected by bounded staging monitor")
	}
	if stageWithinBounds(r, stage) == nil {
		t.Fatal("blanket untrusted ownership accepted without an acquisition policy")
	}
}
