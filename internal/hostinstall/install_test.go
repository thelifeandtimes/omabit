package hostinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostsetup"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const testUID = 942
const testSocket = "/run/user/942/" + rootless.SocketName

type call struct {
	path string
	args []string
	env  []string
}

type fakeInfo struct {
	os.FileInfo
	uid, gid uint32
	mode     os.FileMode
}

func (f fakeInfo) Sys() any          { return &syscall.Stat_t{Uid: f.uid, Gid: f.gid} }
func (f fakeInfo) Mode() os.FileMode { return f.mode }
func (f fakeInfo) IsDir() bool       { return f.mode.IsDir() }

type fixture struct {
	i             installer
	t             *testing.T
	calls         []call
	created       bool
	files         map[string][]byte
	metadata      map[string]fakeInfo
	statuses      map[string]string
	info          []byte
	fail          func(call) error
	afterCreate   func()
	beforeCommand func(call)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		t: t, metadata: map[string]fakeInfo{}, statuses: map[string]string{},
		files: map[string][]byte{}, info: goodInfo(t),
	}
	f.i = installer{
		root: t.TempDir(), euid: 0, goos: "linux", goarch: "amd64",
		chown:    func(*os.File) error { return nil },
		groupIDs: func(*user.User) ([]string, error) { return []string{"942"}, nil },
	}
	for _, name := range helpers {
		f.put("/usr/bin/"+name, "not an executable: tests must never run this", 0755)
	}
	f.put("/proc/sys/kernel/unprivileged_userns_clone", "1", 0644)
	for _, path := range []string{"/var/lib", "/etc/systemd/user", "/run/user"} {
		f.mkdir(path, 0755)
	}
	for _, name := range []string{"dockerd-rootless.sh", "check-engine.sh", "daemon.json", "LICENSE", "NOTICE", rootless.UnitName} {
		f.files[name] = []byte("verified snapshot " + name)
	}
	f.i.verified = func(string) (map[string][]byte, error) {
		out := map[string][]byte{}
		for name, data := range f.files {
			out[name] = bytes.Clone(data)
		}
		return out, nil
	}
	f.i.lstat = func(path string) (os.FileInfo, error) {
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(f.i.root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
			t.Fatalf("filesystem operation escaped test root: %s", path)
		}
		meta, ok := f.metadata["/"+rel]
		if !ok {
			meta.mode = info.Mode()
		}
		meta.FileInfo = info
		return meta, nil
	}
	f.i.lookupUser = func(name string) (*user.User, error) {
		if name != rootless.AccountName {
			t.Fatalf("unexpected user lookup: %s", name)
		}
		if !f.created {
			return nil, user.UnknownUserError(name)
		}
		return &user.User{Username: name, Uid: "942", Gid: "942", HomeDir: rootless.HomeDir}, nil
	}
	f.i.lookupGroup = func(name string) (*user.Group, error) {
		if f.created && name == rootless.AccountName {
			return &user.Group{Name: name, Gid: "942"}, nil
		}
		return nil, user.UnknownGroupError(name)
	}
	f.i.inspect = func() hostsetup.Report {
		r := hostsetup.Report{}
		for _, id := range []string{"systemd", "cgroup-v2", "userns-clone", "userns-limit", "binary-iptables", "account", "account-groups", "subuid", "subgid", "docker-version", "installation-verification", "runtime-verification"} {
			status := "pass"
			if !f.created && slices.Contains([]string{"account", "account-groups", "subuid", "subgid"}, id) {
				status = "unknown"
			}
			if strings.Contains(id, "verification") || id == "docker-version" {
				status = "unknown"
			}
			if value, ok := f.statuses[id]; ok {
				status = value
			}
			r.Checks = append(r.Checks, hostsetup.Check{ID: id, Status: status})
		}
		return r
	}
	f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
		c := call{path, slices.Clone(args), slices.Clone(env)}
		f.calls = append(f.calls, c)
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > commandTimeout || ctx.Err() != nil {
			t.Fatalf("command missing bounded live context: %v, %v", deadline, ctx.Err())
		}
		if f.beforeCommand != nil {
			f.beforeCommand(c)
		}
		if f.fail != nil {
			if err := f.fail(c); err != nil {
				return nil, err
			}
		}
		switch {
		case path == "/usr/bin/dockerd":
			return []byte("Docker version 29.7.2, build reviewed\n"), nil
		case path == "/usr/bin/useradd" && slices.Equal(args, []string{"--help"}):
			return []byte("  --add-subids-for-system  allocate subordinate IDs\n"), nil
		case path == "/usr/bin/useradd":
			f.created = true
			f.mkdir(rootless.HomeDir, 0700)
			f.metadata[rootless.HomeDir] = fakeInfo{uid: testUID, gid: testUID, mode: os.ModeDir | 0700}
			f.put("/etc/passwd", "root:x:0:0::/root:/bin/sh\nurbit-runner:x:942:942::"+rootless.HomeDir+":/usr/bin/nologin\n", 0644)
			f.put("/etc/shadow", "urbit-runner:!:20000:0:99999:7:::\n", 0600)
			if f.afterCreate != nil {
				f.afterCreate()
			}
		case path == "/usr/bin/systemctl" && slices.Contains(args, "enable"):
			f.mkdir("/run/user/942", 0700)
			f.metadata["/run/user/942"] = fakeInfo{uid: testUID, gid: testUID, mode: os.ModeDir | 0700}
			// Socket evidence is synthetic; tests never bind or contact a socket.
			f.put(testSocket, "not a socket", 0600)
			f.metadata[testSocket] = fakeInfo{uid: testUID, gid: testUID, mode: os.ModeSocket | 0660}
		case path == "/usr/bin/runuser" && slices.Contains(args, "/usr/bin/docker"):
			return f.info, nil
		}
		return nil, nil
	}
	return f
}

func (f *fixture) mkdir(path string, mode os.FileMode) {
	f.t.Helper()
	if err := os.MkdirAll(f.i.path(path), mode); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Chmod(f.i.path(path), mode); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) put(path, text string, mode os.FileMode) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Dir(f.i.path(path)), 0755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(f.i.path(path), []byte(text), mode); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Chmod(f.i.path(path), mode); err != nil {
		f.t.Fatal(err)
	}
}

func goodInfo(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"ServerVersion": rootless.EngineVersion, "SecurityOptions": []string{"name=seccomp,profile=builtin", "name=rootless"},
		"CgroupDriver": "systemd", "CgroupVersion": "2", "DockerRootDir": rootless.HomeDir + "/.local/share/urbit-runner/docker",
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func tree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += string(data)
		}
		out[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestFirstInstallSnapshotAndCommands(t *testing.T) {
	f := newFixture(t)
	t.Setenv("HTTP_PROXY", "untrusted-proxy")
	t.Setenv("DOCKER_HOST", "unix:///run/docker.sock")
	t.Setenv("DOCKER_CONTEXT", "rootful")
	t.Setenv("LD_PRELOAD", "/untrusted.so")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "untrusted-bus")
	f.beforeCommand = func(c call) {
		// The bundle may disappear or change after verification; installation
		// must keep using the returned snapshot, not reopen its source paths.
		for name := range f.files {
			f.files[name] = []byte("unverified replacement")
		}
	}
	r, err := f.i.install(context.Background(), "/bundle-never-opened-by-installer")
	if err != nil {
		t.Fatal(err)
	}
	want := Result{AccountCreated: true, Account: rootless.AccountName, UID: testUID, GID: testUID,
		Service: rootless.UnitName, Socket: testSocket, RootlessVerified: true}
	if r != want {
		t.Fatalf("result = %+v, want %+v", r, want)
	}
	for _, name := range []string{"dockerd-rootless.sh", "check-engine.sh", "daemon.json", "LICENSE", "NOTICE", rootless.UnitName} {
		path := rootless.InstallDir + "/" + name
		mode := os.FileMode(0644)
		if strings.HasSuffix(name, ".sh") {
			mode = 0755
		}
		if name == rootless.UnitName {
			path = unitPath
		}
		data, err := os.ReadFile(f.i.path(path))
		if err != nil || string(data) != "verified snapshot "+name {
			t.Fatalf("installed unverified/wrong bytes in %s: %q, %v", path, data, err)
		}
		info, err := f.i.lstat(f.i.path(path))
		uid, gid, ok := ownership(info)
		if err != nil || !ok || uid != 0 || gid != 0 || info.Mode().Perm() != mode {
			t.Fatalf("wrong installed metadata for %s: %v, %v", path, info, err)
		}
	}
	if _, err := os.Lstat(f.i.path(rootless.InstallDir + "/manifest.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manifest should not be installed: %v", err)
	}
	wantCommands := [][]string{
		{"/usr/bin/dockerd", "--version"},
		{"/usr/bin/useradd", "--help"},
		{"/usr/bin/useradd", "--system", "--user-group", "--create-home", "--home-dir", rootless.HomeDir, "--shell", "/usr/bin/nologin", "--groups", "", "--add-subids-for-system", "--key", "HOME_MODE=0700", "--key", "SUB_UID_COUNT=65536", "--key", "SUB_GID_COUNT=65536", rootless.AccountName},
		{"/usr/bin/loginctl", "enable-linger", rootless.AccountName},
		{"/usr/bin/systemctl", "start", "user@942.service"},
		{"/usr/bin/systemctl", "--user", "--machine=urbit-runner@.host", "daemon-reload"},
		{"/usr/bin/systemctl", "--user", "--machine=urbit-runner@.host", "enable", "--now", rootless.UnitName},
		{"/usr/bin/runuser", "-u", rootless.AccountName, "--", "/usr/bin/env", "-i", "HOME=" + rootless.HomeDir, "PATH=/usr/bin:/bin", "/usr/bin/docker", "--host", "unix://" + testSocket, "info", "--format", "{{json .}}"},
	}
	wantEnv := []string{"PATH=/usr/bin:/bin", "HOME=/root", "LANG=C", "LC_ALL=C", "SYSTEMD_PAGER=", "SYSTEMD_PAGERSECURE=1", "SYSTEMD_COLORS=0"}
	var got [][]string
	for _, c := range f.calls {
		got = append(got, append([]string{c.path}, c.args...))
		if !slices.Equal(c.env, wantEnv) {
			t.Fatalf("unsanitized command environment: %v", c.env)
		}
	}
	if !reflect.DeepEqual(got, wantCommands) {
		t.Fatalf("commands = %q, want %q", got, wantCommands)
	}
	encoded, err := json.Marshal(r)
	if err != nil || !bytes.Contains(encoded, []byte(`"runtime_qualified":false`)) || !bytes.Contains(encoded, []byte(`"rootless_verified":true`)) {
		t.Fatalf("incorrect qualification JSON: %s, %v", encoded, err)
	}
}

func TestValidationNeverMutates(t *testing.T) {
	cases := []struct {
		name string
		edit func(*fixture)
	}{
		{"nonroot", func(f *fixture) { f.i.euid = 1000 }},
		{"platform", func(f *fixture) { f.i.goos = "darwin" }},
		{"architecture", func(f *fixture) { f.i.goarch = "arm64" }},
		{"bundle", func(f *fixture) {
			f.i.verified = func(string) (map[string][]byte, error) { return nil, errors.New("bad pinned hash") }
		}},
		{"incomplete-snapshot", func(f *fixture) { delete(f.files, "NOTICE") }},
		{"NSS-account", func(f *fixture) { f.created = true }},
		{"NSS-group", func(f *fixture) {
			f.i.lookupGroup = func(name string) (*user.Group, error) { return &user.Group{Name: name, Gid: "942"}, nil }
		}},
		{"NSS-failure", func(f *fixture) {
			f.i.lookupUser = func(string) (*user.User, error) { return nil, errors.New("NSS unavailable") }
		}},
		{"existing-home", func(f *fixture) { f.mkdir(rootless.HomeDir, 0700) }},
		{"existing-install", func(f *fixture) { f.mkdir(rootless.InstallDir, 0755) }},
		{"existing-service", func(f *fixture) { f.put(unitPath, "preserve unit", 0644) }},
		{"existing-override", func(f *fixture) { f.mkdir(unitPath+".d", 0755) }},
		{"existing-linger", func(f *fixture) { f.put("/var/lib/systemd/linger/"+rootless.AccountName, "", 0644) }},
		{"dangling-home", func(f *fixture) {
			if err := os.Symlink("/absent", f.i.path(rootless.HomeDir)); err != nil {
				t.Fatal(err)
			}
		}},
		{"writable-parent", func(f *fixture) { f.mkdir("/usr/local/lib", 0775) }},
		{"unowned-parent", func(f *fixture) { f.metadata["/var/lib"] = fakeInfo{uid: 1000, mode: os.ModeDir | 0755} }},
		{"unowned-runtime-parent", func(f *fixture) { f.metadata["/run/user"] = fakeInfo{uid: 1000, mode: os.ModeDir | 0755} }},
		{"symlink-parent", func(f *fixture) {
			f.mkdir("/usr/local", 0755)
			if err := os.Symlink(f.i.path("/var/lib"), f.i.path("/usr/local/lib")); err != nil {
				t.Fatal(err)
			}
		}},
		{"missing-helper", func(f *fixture) {
			if err := os.Remove(f.i.path("/usr/bin/getsubids")); err != nil {
				t.Fatal(err)
			}
		}},
		{"writable-helper", func(f *fixture) { f.put("/usr/bin/newuidmap", "preserve caps", 0775) }},
		{"unowned-helper", func(f *fixture) { f.metadata["/usr/bin/dockerd"] = fakeInfo{uid: 1000, mode: 0755} }},
		{"nonexecutable-helper", func(f *fixture) { f.put("/usr/bin/docker", "", 0644) }},
		{"symlink-helper", func(f *fixture) {
			path := f.i.path("/usr/bin/docker")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(f.i.path("/usr/bin/dockerd"), path); err != nil {
				t.Fatal(err)
			}
		}},
		{"cgroup", func(f *fixture) { f.statuses["cgroup-v2"] = "blocked" }},
		{"userns", func(f *fixture) { f.statuses["userns-limit"] = "unknown" }},
		{"malformed-clone", func(f *fixture) { f.statuses["userns-clone"] = "unknown" }},
		{"disabled-clone", func(f *fixture) { f.statuses["userns-clone"] = "blocked" }},
		{"engine-version", func(f *fixture) {
			run := f.i.run
			f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
				if path == "/usr/bin/dockerd" {
					return []byte("Docker version 29.7.20, build wrong"), nil
				}
				return run(ctx, path, args, env)
			}
		}},
		{"useradd-support", func(f *fixture) {
			run := f.i.run
			f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
				if path == "/usr/bin/useradd" {
					return []byte("--add-subids-for-system-fake"), nil
				}
				return run(ctx, path, args, env)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.edit(f)
			before := tree(t, f.i.root)
			r, err := f.i.install(context.Background(), "/bundle")
			if err == nil || r != (Result{}) || f.i.mutated {
				t.Fatalf("validation result = %+v, err = %v, mutation = %v", r, err, f.i.mutated)
			}
			if !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("validation changed filesystem")
			}
			for _, c := range f.calls {
				if !(c.path == "/usr/bin/dockerd" && slices.Equal(c.args, []string{"--version"})) && !(c.path == "/usr/bin/useradd" && slices.Equal(c.args, []string{"--help"})) {
					t.Fatalf("validation attempted mutation: %+v", c)
				}
			}
		})
	}
}

func TestContextValidation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, ctx := range []context.Context{nil, ctx} {
		f := newFixture(t)
		f.i.verified = func(string) (map[string][]byte, error) { t.Fatal("verified after invalid context"); return nil, nil }
		r, err := f.i.install(ctx, "/bundle")
		if err == nil || r != (Result{}) || len(f.calls) != 0 {
			t.Fatalf("invalid context accepted: %+v, %v", r, err)
		}
	}
	f := newFixture(t)
	ctx, cancel = context.WithCancel(context.Background())
	f.beforeCommand = func(c call) {
		if slices.Contains(c.args, "--help") {
			cancel()
		}
	}
	r, err := f.i.install(ctx, "/bundle")
	if !errors.Is(err, context.Canceled) || r.Partial || f.i.mutated {
		t.Fatalf("cancellation mutated: %+v, %v", r, err)
	}
}

func TestRealBundleVerifierRejectsMalformedInputWithoutCommands(t *testing.T) {
	f := newFixture(t)
	f.i.verified = rootless.VerifiedFiles
	f.mkdir("/bundle", 0700)
	f.put("/bundle/manifest.json", `{"files":[{"name":"evil","sha256":"trusted-by-user-only"}]}`, 0644)
	for _, name := range []string{"dockerd-rootless.sh", "check-engine.sh", "daemon.json", "LICENSE", "NOTICE", rootless.UnitName} {
		f.put("/bundle/"+name, "unreviewed code", 0644)
	}
	before := tree(t, f.i.root)
	r, err := f.i.install(context.Background(), f.i.path("/bundle"))
	if err == nil || !strings.Contains(err.Error(), "verify bundle") || r != (Result{}) || len(f.calls) != 0 || !reflect.DeepEqual(before, tree(t, f.i.root)) {
		t.Fatalf("malformed bundle reached host operations: %+v, %v, %v", r, err, f.calls)
	}
}

func TestOptionalSysctlAndMissingParents(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(f.i.path("/proc/sys/kernel/unprivileged_userns_clone")); err != nil {
		t.Fatal(err)
	}
	f.statuses["userns-clone"] = "unknown"
	if err := os.RemoveAll(f.i.path("/var/lib")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.i.install(context.Background(), "/bundle"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/var/lib", "/usr/local", "/usr/local/lib", "/usr/local/lib/urbit-runner", "/usr/local/lib/urbit-runner/rootless", rootless.InstallDir} {
		info, err := f.i.lstat(f.i.path(path))
		uid, gid, ok := ownership(info)
		if err != nil || !ok || uid != 0 || gid != 0 || info.Mode().Perm() != 0755 {
			t.Fatalf("unsafe created directory %s: %v, %v", path, info, err)
		}
	}
}

func TestCommandFailurePreservesPartialInstall(t *testing.T) {
	for _, stage := range []string{"useradd", "linger", "manager", "reload", "enable", "info"} {
		t.Run(stage, func(t *testing.T) {
			f := newFixture(t)
			failure := errors.New("injected failure")
			f.fail = func(c call) error {
				match := stage == "useradd" && c.path == "/usr/bin/useradd" && !slices.Contains(c.args, "--help") ||
					stage == "linger" && c.path == "/usr/bin/loginctl" ||
					stage == "manager" && slices.Contains(c.args, "start") ||
					stage == "reload" && slices.Contains(c.args, "daemon-reload") ||
					stage == "enable" && slices.Contains(c.args, "enable") ||
					stage == "info" && slices.Contains(c.args, "info")
				if match {
					return failure
				}
				return nil
			}
			r, err := f.i.install(context.Background(), "/bundle")
			if !errors.Is(err, failure) || !r.Partial || r.RootlessVerified || r.RuntimeQualified || !strings.Contains(err.Error(), "resources preserved") {
				t.Fatalf("wrong partial result: %+v, %v", r, err)
			}
			if r.AccountCreated != (stage != "useradd") || (r.Service != "") != (stage == "info") {
				t.Fatalf("result claims uncompleted steps: %+v", r)
			}
			if data, err := os.ReadFile(f.i.path(unitPath)); err != nil || string(data) != "verified snapshot "+rootless.UnitName {
				t.Fatalf("unit not preserved: %s, %v", data, err)
			}
			if stage != "useradd" {
				if _, err := os.Lstat(f.i.path(rootless.HomeDir)); err != nil {
					t.Fatal("home not preserved", err)
				}
			}
			before := tree(t, f.i.root)
			// A separate invocation has no in-memory history and still refuses.
			i := f.i
			i.mutated = false
			if _, err := i.install(context.Background(), "/bundle"); err == nil || !reflect.DeepEqual(before, tree(t, f.i.root)) {
				t.Fatal("rerun took over partial installation")
			}
		})
	}
}

func TestPostCreationAccountChecksStopBeforeServices(t *testing.T) {
	cases := []struct {
		name string
		edit func(*fixture)
	}{
		{"root-uid", func(f *fixture) {
			lookup := f.i.lookupUser
			f.i.lookupUser = func(name string) (*user.User, error) {
				u, err := lookup(name)
				if err == nil {
					u.Uid = "0"
				}
				return u, err
			}
		}},
		{"private-group", func(f *fixture) {
			f.i.lookupGroup = func(name string) (*user.Group, error) { return &user.Group{Name: name, Gid: "943"}, nil }
		}},
		{"NSS-admin-membership", func(f *fixture) {
			f.i.groupIDs = func(*user.User) ([]string, error) { return []string{"942", "27"}, nil }
		}},
		{"NSS-admin-alias", func(f *fixture) {
			lookup := f.i.lookupGroup
			f.i.lookupGroup = func(name string) (*user.Group, error) {
				if name == "sudo" {
					return &user.Group{Name: name, Gid: "942"}, nil
				}
				return lookup(name)
			}
		}},
		{"wrong-home-owner", func(f *fixture) { f.metadata[rootless.HomeDir] = fakeInfo{uid: 0, mode: os.ModeDir | 0700} }},
		{"public-home", func(f *fixture) {
			f.metadata[rootless.HomeDir] = fakeInfo{uid: testUID, gid: testUID, mode: os.ModeDir | 0755}
		}},
		{"ssh-skeleton", func(f *fixture) { f.put(rootless.HomeDir+"/.ssh/authorized_keys", "must not permit key login", 0600) }},
		{"unit-skeleton", func(f *fixture) {
			f.put(rootless.HomeDir+"/.config/systemd/user/"+rootless.UnitName, "wrong unit", 0644)
		}},
		{"unlocked-password", func(f *fixture) { f.put("/etc/shadow", "urbit-runner:secret-hash:20000:0:99999:7:::\n", 0600) }},
		{"login-shell", func(f *fixture) { f.put("/etc/passwd", "urbit-runner:x:942:942::"+rootless.HomeDir+":/bin/sh\n", 0644) }},
		{"subuid-overlap", func(f *fixture) { f.statuses["subuid"] = "blocked" }},
		{"subgid-host-id", func(f *fixture) { f.statuses["subgid"] = "blocked" }},
		{"subuid-missing", func(f *fixture) { f.statuses["subuid"] = "missing" }},
		{"groups-unknown", func(f *fixture) { f.statuses["account-groups"] = "unknown" }},
		{"account-unknown", func(f *fixture) { f.statuses["account"] = "unknown" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.afterCreate = func() { tc.edit(f) }
			r, err := f.i.install(context.Background(), "/bundle")
			if err == nil || !r.AccountCreated || !r.Partial || r.RootlessVerified || r.RuntimeQualified {
				t.Fatalf("unsafe account accepted: %+v, %v", r, err)
			}
			if strings.Contains(err.Error(), "secret-hash") {
				t.Fatal("error disclosed shadow contents")
			}
			for _, c := range f.calls {
				if c.path == "/usr/bin/systemctl" || c.path == "/usr/bin/loginctl" {
					t.Fatalf("started services with unsafe account: %+v", c)
				}
			}
		})
	}
}

func TestRootfulAccessCheckedWithoutConnecting(t *testing.T) {
	for _, accessible := range []bool{false, true} {
		t.Run(fmt.Sprint(accessible), func(t *testing.T) {
			f := newFixture(t)
			f.put("/run/docker.sock", "rootful preserved", 0660)
			f.metadata["/run/docker.sock"] = fakeInfo{uid: 0, gid: 999, mode: os.ModeSocket | 0660}
			f.fail = func(c call) error {
				if accessible && slices.Contains(c.args, "/usr/bin/test") {
					return errors.New("exit status 1")
				}
				return nil
			}
			r, err := f.i.install(context.Background(), "/bundle")
			if accessible && (err == nil || !r.Partial || r.RootlessVerified || r.Service != "") {
				t.Fatalf("rootful access accepted: %+v, %v", r, err)
			}
			if !accessible && err != nil {
				t.Fatal(err)
			}
			checks := 0
			for _, c := range f.calls {
				if slices.Contains(c.args, "/run/docker.sock") {
					checks++
					want := []string{"-u", rootless.AccountName, "--", "/usr/bin/env", "-i", "HOME=" + rootless.HomeDir, "PATH=/usr/bin:/bin", "/usr/bin/test", "!", "-w", "/run/docker.sock"}
					if c.path != "/usr/bin/runuser" || !slices.Equal(c.args, want) {
						t.Fatalf("unexpected rootful interaction: %+v", c)
					}
				}
			}
			if checks != 1 {
				t.Fatalf("rootful access checks = %d", checks)
			}
			if data, err := os.ReadFile(f.i.path("/run/docker.sock")); err != nil || string(data) != "rootful preserved" {
				t.Fatal("changed rootful socket")
			}
		})
	}
}

func TestUnsafeSocketNeverContactsEngine(t *testing.T) {
	for _, tc := range []struct {
		path string
		meta fakeInfo
	}{
		{testSocket, fakeInfo{uid: 0, gid: testUID, mode: os.ModeSocket | 0660}},
		{testSocket, fakeInfo{uid: testUID, gid: 166502, mode: os.ModeSocket | 0660}},
		{testSocket, fakeInfo{uid: testUID, gid: testUID, mode: os.ModeSocket | 0666}},
		{testSocket, fakeInfo{uid: testUID, gid: testUID, mode: os.ModeSymlink | 0777}},
		{testSocket, fakeInfo{uid: testUID, gid: testUID, mode: 0600}},
		{"/run/user/942", fakeInfo{uid: 0, gid: testUID, mode: os.ModeDir | 0700}},
		{"/run/user/942", fakeInfo{uid: testUID, gid: testUID, mode: os.ModeDir | 0755}},
		{"/run/user/942", fakeInfo{uid: testUID, gid: testUID, mode: os.ModeSymlink | 0700}},
		{"/run/user", fakeInfo{uid: 1000, mode: os.ModeDir | 0755}},
	} {
		t.Run(fmt.Sprintf("%s-%d-%s", tc.path, tc.meta.uid, tc.meta.mode), func(t *testing.T) {
			f := newFixture(t)
			run := f.i.run
			f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
				out, err := run(ctx, path, args, env)
				if slices.Contains(args, "enable") {
					f.metadata[tc.path] = tc.meta
				}
				return out, err
			}
			r, err := f.i.install(context.Background(), "/bundle")
			if err == nil || !r.Partial || r.RootlessVerified || r.Socket != "" {
				t.Fatalf("unsafe socket accepted: %+v, %v", r, err)
			}
			for _, c := range f.calls {
				if slices.Contains(c.args, "info") {
					t.Fatal("contacted unsafe engine socket")
				}
			}
		})
	}
}

func TestInfoRejectsUnverifiedEngines(t *testing.T) {
	for field, values := range map[string][]any{
		"ServerVersion":   {"29.7.20", "29.7.2-dev", "", nil},
		"SecurityOptions": {[]string{"name=rootless-extra"}, []string{"rootless"}, []string{"name=rootful"}, nil},
		"CgroupDriver":    {"cgroupfs", "", nil}, "CgroupVersion": {"1", 2, nil},
		"DockerRootDir": {"/var/lib/docker", rootless.HomeDir + "/.local/share/docker", rootless.HomeDir + "/.local/share/urbit-runner/docker/", nil},
	} {
		for _, value := range values {
			t.Run(fmt.Sprintf("%s-%v", field, value), func(t *testing.T) {
				f := newFixture(t)
				var info map[string]any
				if err := json.Unmarshal(f.info, &info); err != nil {
					t.Fatal(err)
				}
				info[field] = value
				var err error
				f.info, err = json.Marshal(info)
				if err != nil {
					t.Fatal(err)
				}
				r, err := f.i.install(context.Background(), "/bundle")
				if err == nil || !r.Partial || r.RootlessVerified || r.RuntimeQualified || r.Socket != testSocket {
					t.Fatalf("unverified info accepted: %+v, %v", r, err)
				}
			})
		}
	}
	for _, data := range [][]byte{nil, []byte("null"), []byte("[]"), []byte("{}{}"), []byte("not JSON"), append(goodInfo(t), bytes.Repeat([]byte(" "), maxOutput)...)} {
		if err := verifyInfo(data); err == nil {
			t.Fatal("malformed or oversized info accepted")
		}
	}
}

func TestExclusiveWriteAndPartialFilesystemFailure(t *testing.T) {
	f := newFixture(t)
	f.put(unitPath, "must survive", 0644)
	if err := f.i.write(context.Background(), unitPath, []byte("replacement"), 0644); err == nil || f.i.mutated {
		t.Fatal("exclusive write overwrote existing file")
	}
	if data, err := os.ReadFile(f.i.path(unitPath)); err != nil || string(data) != "must survive" {
		t.Fatal("existing file changed")
	}
	f = newFixture(t)
	f.i.chown = func(*os.File) error { return errors.New("injected ownership failure") }
	r, err := f.i.install(context.Background(), "/bundle")
	if err == nil || !r.Partial || r.AccountCreated {
		t.Fatalf("wrong filesystem failure result: %+v, %v", r, err)
	}
	if _, err := os.Lstat(f.i.path("/usr/local")); err != nil {
		t.Fatal("created directory was removed", err)
	}
}

func TestHelperRecheckedAndCapabilitiesUntouched(t *testing.T) {
	f := newFixture(t)
	var chowned []string
	f.i.chown = func(file *os.File) error { chowned = append(chowned, file.Name()); return nil }
	f.beforeCommand = func(c call) {
		if slices.Contains(c.args, "--help") {
			f.metadata["/usr/bin/useradd"] = fakeInfo{uid: 1000, mode: 0755}
		}
	}
	r, err := f.i.install(context.Background(), "/bundle")
	if err == nil || !r.Partial || r.AccountCreated {
		t.Fatalf("tampered helper used: %+v, %v", r, err)
	}
	for _, path := range chowned {
		if strings.Contains(path, "/usr/bin/") {
			t.Fatalf("changed packaged helper (could erase capabilities): %s", path)
		}
	}
	for _, c := range f.calls {
		if c.path == "/usr/bin/useradd" && !slices.Contains(c.args, "--help") {
			t.Fatal("executed tampered useradd")
		}
	}
	if _, err := f.i.command(context.Background(), "/bin/sh", "-c", "anything"); err == nil {
		t.Fatal("allowed a shell")
	}
}

func TestBoundedOutput(t *testing.T) {
	for _, size := range []int{maxOutput - 1, maxOutput, maxOutput + 1, maxOutput * 3} {
		var output boundedOutput
		_, err := io.Copy(&output, strings.NewReader(strings.Repeat("x", size)))
		if output.buffer.Len() > maxOutput || (err != nil) != (size > maxOutput) || output.exceeded != (size > maxOutput) {
			t.Fatalf("output limit failed: size=%d len=%d exceeded=%v err=%v", size, output.buffer.Len(), output.exceeded, err)
		}
	}
	f := newFixture(t)
	f.i.run = func(context.Context, string, []string, []string) ([]byte, error) {
		return bytes.Repeat([]byte("x"), maxOutput+1), nil
	}
	if _, err := f.i.command(context.Background(), "/usr/bin/dockerd", "--version"); err == nil {
		t.Fatal("accepted oversized runner output")
	}
}

func TestCancellationAfterMutationPreservesResources(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.beforeCommand = func(c call) {
		if c.path == "/usr/bin/loginctl" {
			cancel()
		}
	}
	r, err := f.i.install(ctx, "/bundle")
	if !errors.Is(err, context.Canceled) || !r.Partial || !r.AccountCreated || r.UID != testUID || r.Service != "" || r.RootlessVerified {
		t.Fatalf("wrong canceled partial result: %+v, %v", r, err)
	}
	for _, path := range []string{rootless.HomeDir, rootless.InstallDir, unitPath} {
		if _, err := os.Lstat(f.i.path(path)); err != nil {
			t.Fatalf("cancellation removed %s: %v", path, err)
		}
	}
	for _, c := range f.calls {
		if c.path == "/usr/bin/systemctl" || slices.Contains(c.args, "info") {
			t.Fatalf("installation continued after cancellation: %+v", c)
		}
	}
}
