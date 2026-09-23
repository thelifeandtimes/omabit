package hostsetup

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type fixtureFS struct {
	files  fstest.MapFS
	denied map[string]bool
	opened []string
}

func (f *fixtureFS) Open(path string) (fs.File, error) {
	f.opened = append(f.opened, path)
	if f.denied[path] {
		return nil, &fs.PathError{Op: "private-error", Path: "private-path", Err: fs.ErrPermission}
	}
	return f.files.Open(path)
}

func (f *fixtureFS) Stat(path string) (fs.FileInfo, error) {
	if f.denied[path] {
		return nil, &fs.PathError{Op: "private-error", Path: "private-path", Err: fs.ErrPermission}
	}
	return f.files.Stat(path)
}

func fixture() (probe, *fixtureFS) {
	f := &fixtureFS{files: fstest.MapFS{}, denied: map[string]bool{}}
	for path, text := range map[string]string{
		"sys/fs/cgroup/cgroup.controllers":                 "cpuset cpu io memory hugetlb pids\n",
		"proc/sys/kernel/unprivileged_userns_clone":        "1\n",
		"proc/sys/user/max_user_namespaces":                "32768\n",
		"etc/passwd":                                       "root:x:0:0:root:/root:/bin/bash\nurbit-runner:x:942:942:Private GECOS:/var/lib/urbit-runner:/usr/bin/nologin\ndesktop:x:1000:1000:Private Desktop:/home/desktop:/bin/bash\n",
		"etc/group":                                        "root:x:0:\nurbit-runner:x:942:\nwheel:x:998:desktop\ndocker:x:997:desktop\n",
		"etc/subuid":                                       "desktop:100000:65536\nurbit-runner:200000:65536\n",
		"etc/subgid":                                       "1000:100000:65536\n942:200000:65536\n",
		"var/lib/pacman/local/docker-29.7.2-1/desc":        "%NAME%\ndocker\n\n%VERSION%\n1:" + rootless.EngineVersion + "-1\n\n%DESC%\nDocker engine\n",
		"var/lib/systemd/linger/urbit-runner":              "",
		"etc/shadow":                                       "shadow-secret",
		"home/desktop/.docker/config.json":                 "docker-config-secret",
		"var/lib/urbit-runner/.docker/config.json":         "runner-config-secret",
		"etc/docker/daemon.json":                           "daemon-config-secret",
		"var/lib/pacman/local/docker-compose-9-1/desc":     "%NAME%\ndocker-compose\n\n%VERSION%\n9-1\n",
		"var/lib/pacman/local/docker-buildx-9-1/desc":      "intentionally-invalid",
		"var/lib/pacman/local/docker-rootless-9-1/desc":    "intentionally-invalid",
		"var/lib/pacman/local/docker-rootless-extras/desc": "intentionally-invalid",
	} {
		f.files[path] = &fstest.MapFile{Data: []byte(text), Mode: 0644}
	}
	f.files["run/systemd/system"] = &fstest.MapFile{Mode: fs.ModeDir | 0755}
	f.files["run/user/942/urbit-docker.sock"] = &fstest.MapFile{Mode: fs.ModeSocket | 0600}
	f.files["run/docker.sock"] = &fstest.MapFile{Mode: fs.ModeSocket | 0660}
	for _, name := range strings.Fields("docker dockerd rootlesskit slirp4netns newuidmap newgidmap systemctl loginctl runuser useradd getsubids iptables") {
		f.files["usr/bin/"+name] = &fstest.MapFile{Mode: 0755, Data: []byte("not an executable program")}
	}
	return probe{files: f, lstat: f.Stat, goos: "linux", goarch: "amd64"}, f
}

func check(t *testing.T, report Report, id, status string) Check {
	t.Helper()
	for _, c := range report.Checks {
		if c.ID == id {
			if c.Status != status {
				t.Fatalf("%s: got %s (%s), want %s", id, c.Status, c.Detail, status)
			}
			return c
		}
	}
	t.Fatalf("missing check %q", id)
	return Check{}
}

func TestInspectMetadataDoesNotQualifyRuntime(t *testing.T) {
	p, _ := fixture()
	r := p.inspect()
	if r.SchemaVersion != 1 || r.Deployment != deployment || r.RuntimeQualified {
		t.Fatalf("unsafe report metadata: %+v", r)
	}
	ids := []string{"platform", "systemd", "cgroup-v2", "userns-clone", "userns-limit", "binary-docker", "binary-dockerd", "binary-rootlesskit", "binary-slirp4netns", "binary-newuidmap", "binary-newgidmap", "binary-systemctl", "binary-loginctl", "binary-runuser", "binary-useradd", "binary-getsubids", "binary-iptables", "docker-version", "account", "account-groups", "subuid", "subgid", "linger", "account-socket", "rootful-socket", "docker-environment", "installation-verification", "runtime-verification"}
	var got []string
	for _, c := range r.Checks {
		got = append(got, c.ID)
		want := "pass"
		if c.ID == "rootful-socket" || c.ID == "docker-environment" {
			want = "info"
		}
		if c.ID == "installation-verification" || c.ID == "runtime-verification" {
			want = "unknown"
		}
		check(t, r, c.ID, want)
	}
	if !reflect.DeepEqual(got, ids) {
		t.Fatalf("check order changed: %v", got)
	}
}

func TestUnsupportedPlatformAndMissingPrerequisites(t *testing.T) {
	for _, platform := range [][2]string{{"linux", "arm64"}, {"darwin", "amd64"}, {"windows", "amd64"}} {
		t.Run(strings.Join(platform[:], "/"), func(t *testing.T) {
			p, _ := fixture()
			p.goos, p.goarch = platform[0], platform[1]
			r := p.inspect()
			check(t, r, "platform", "blocked")
			check(t, r, "runtime-verification", "unknown")
		})
	}
	p, f := fixture()
	f.files = fstest.MapFS{}
	r := p.inspect()
	if len(r.Checks) != 28 {
		t.Fatalf("inspection failed fast: %d checks", len(r.Checks))
	}
	for _, c := range r.Checks {
		if strings.HasPrefix(c.ID, "binary-") {
			check(t, r, c.ID, "missing")
		}
	}
	for _, id := range []string{"systemd", "cgroup-v2", "userns-limit", "account", "account-groups", "subuid", "subgid", "linger"} {
		check(t, r, id, "missing")
	}
	for _, id := range []string{"docker-version", "userns-clone", "account-socket"} {
		check(t, r, id, "unknown")
	}
	check(t, r, "rootful-socket", "info")
}

func TestDedicatedAccount(t *testing.T) {
	for _, tc := range []struct {
		name, passwd, group, id, status string
	}{
		{"missing-not-desktop", "desktop:x:1000:1000::/var/lib/urbit-runner:/bin/bash\n", "", "account", "missing"},
		{"wrong-home", "urbit-runner:x:942:942::/home/desktop:/bin/bash\n", "", "account", "blocked"},
		{"root-uid", "urbit-runner:x:0:942::/var/lib/urbit-runner:/bin/bash\n", "", "account", "blocked"},
		{"root-gid", "urbit-runner:x:942:0::/var/lib/urbit-runner:/bin/bash\n", "", "account", "blocked"},
		{"uid-overflow", "urbit-runner:x:4294967296:942::/var/lib/urbit-runner:/bin/bash\n", "", "account", "unknown"},
		{"invalid-uid", "urbit-runner:x:-1:942::/var/lib/urbit-runner:/bin/bash\n", "", "account", "unknown"},
		{"malformed", "urbit-runner:x:942\n", "", "account", "unknown"},
		{"duplicate", strings.Repeat("urbit-runner:x:942:942::/var/lib/urbit-runner:/bin/bash\n", 2), "", "account", "unknown"},
		{"shared-uid-before-target", "desktop:x:942:1000::/home/desktop:/bin/bash\nurbit-runner:x:942:942::/var/lib/urbit-runner:/usr/bin/nologin\n", "", "account", "blocked"},
		{"shared-uid-after-target", "urbit-runner:x:942:942::/var/lib/urbit-runner:/usr/bin/nologin\ndesktop:x:942:1000::/home/desktop:/bin/bash\n", "", "account", "blocked"},
		{"malformed-unrelated", "private-unrelated-entry\n", "", "account", "unknown"},
		{"wheel-secondary", "", "wheel:x:998:desktop,urbit-runner\n", "account-groups", "blocked"},
		{"docker-secondary", "", "docker:x:997:urbit-runner,desktop\n", "account-groups", "blocked"},
		{"wheel-primary", "", "wheel:x:942:\n", "account-groups", "blocked"},
		{"docker-primary", "", "docker:x:000942:\n", "account-groups", "blocked"},
		{"docker-secondary-alias-after", "", "docker:x:997:\nprivate-alias:x:997:urbit-runner\n", "account-groups", "blocked"},
		{"docker-secondary-alias-before", "", "private-alias:x:997:urbit-runner\ndocker:x:997:\n", "account-groups", "blocked"},
		{"wheel-secondary-alias-after", "", "wheel:x:998:\nprivate-alias:x:998:desktop,urbit-runner\n", "account-groups", "blocked"},
		{"wheel-secondary-alias-before", "", "private-alias:x:998:urbit-runner\nwheel:x:998:\n", "account-groups", "blocked"},
		{"docker-primary-alias", "", "private-alias:x:942:\ndocker:x:942:\n", "account-groups", "blocked"},
		{"wheel-primary-alias", "", "wheel:x:942:\nprivate-alias:x:942:\n", "account-groups", "blocked"},
		{"alias-numeric-gid-leading-zero", "", "docker:x:997:\nprivate-alias:x:000997:urbit-runner\n", "account-groups", "blocked"},
		{"different-gid-not-an-alias", "", "docker:x:997:\nprivate-alias:x:900:urbit-runner\n", "account-groups", "pass"},
		{"alias-only-desktop-membership", "", "docker:x:997:\nprivate-alias:x:997:desktop\n", "account-groups", "pass"},
		{"malformed-group", "", "wheel:x:998\n", "account-groups", "unknown"},
		{"invalid-group-gid", "", "docker:x:invalid:\n", "account-groups", "unknown"},
		{"desktop-membership-only", "", "wheel:x:998:desktop\ndocker:x:997:desktop\n", "account-groups", "pass"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, f := fixture()
			if tc.passwd != "" {
				f.files["etc/passwd"].Data = []byte(tc.passwd)
			}
			if tc.group != "" {
				f.files["etc/group"].Data = []byte(tc.group)
			}
			r := p.inspect()
			check(t, r, tc.id, tc.status)
			if tc.name == "missing-not-desktop" || tc.name == "root-uid" {
				check(t, r, "account-socket", "unknown")
			}
			if tc.name == "missing-not-desktop" {
				check(t, r, "account-groups", "unknown")
				check(t, r, "subuid", "unknown")
				check(t, r, "subgid", "unknown")
			}
		})
	}
}

func TestAbsentVersusDeniedEvidence(t *testing.T) {
	for _, tc := range []struct{ path, id, absent string }{
		{"etc/passwd", "account", "missing"},
		{"etc/group", "account-groups", "missing"},
		{"etc/subuid", "subuid", "missing"},
		{"etc/subgid", "subgid", "missing"},
		{"run/systemd/system", "systemd", "missing"},
		{"sys/fs/cgroup/cgroup.controllers", "cgroup-v2", "missing"},
		{"proc/sys/kernel/unprivileged_userns_clone", "userns-clone", "unknown"},
		{"proc/sys/user/max_user_namespaces", "userns-limit", "missing"},
		{"usr/bin/docker", "binary-docker", "missing"},
		{"var/lib/systemd/linger/urbit-runner", "linger", "missing"},
		{"run/user/942/urbit-docker.sock", "account-socket", "missing"},
		{"var/lib/pacman/local/docker-29.7.2-1/desc", "docker-version", "missing"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			p, f := fixture()
			f.denied[tc.path] = true
			check(t, p.inspect(), tc.id, "unknown")
			delete(f.denied, tc.path)
			delete(f.files, tc.path)
			check(t, p.inspect(), tc.id, tc.absent)
		})
	}
	for _, state := range []string{"present", "absent", "denied"} {
		t.Run("rootful-"+state, func(t *testing.T) {
			p, f := fixture()
			if state == "absent" {
				delete(f.files, "run/docker.sock")
			} else if state == "denied" {
				f.denied["run/docker.sock"] = true
			}
			c := check(t, p.inspect(), "rootful-socket", "info")
			if state == "denied" && !strings.Contains(c.Detail, "unknown") {
				t.Fatal("unreadable rootful socket presented as absent")
			}
		})
	}
}

func TestInvalidMetadataAndKernelControls(t *testing.T) {
	for _, tc := range []struct {
		path, id, text, status string
		mode                   fs.FileMode
	}{
		{"usr/bin/docker", "binary-docker", "", "blocked", 0644},
		{"usr/bin/dockerd", "binary-dockerd", "", "blocked", fs.ModeDir | 0755},
		{"usr/bin/newuidmap", "binary-newuidmap", "", "blocked", fs.ModeNamedPipe | 0755},
		{"run/systemd/system", "systemd", "", "blocked", 0644},
		{"run/user/942/urbit-docker.sock", "account-socket", "", "blocked", 0600},
		{"run/user/942/urbit-docker.sock", "account-socket", "", "blocked", fs.ModeSymlink | 0777},
		{"var/lib/systemd/linger/urbit-runner", "linger", "", "blocked", fs.ModeDir | 0755},
		{"etc/subuid", "subuid", "", "unknown", fs.ModeNamedPipe | 0600},
		{"etc/subgid", "subgid", "", "unknown", fs.ModeDevice | 0600},
		{"etc/passwd", "account", strings.Repeat("x", maxRead+1), "unknown", 0644},
		{"proc/sys/kernel/unprivileged_userns_clone", "userns-clone", "0\n", "blocked", 0644},
		{"proc/sys/kernel/unprivileged_userns_clone", "userns-clone", "2\n", "unknown", 0644},
		{"proc/sys/user/max_user_namespaces", "userns-limit", "0\n", "blocked", 0644},
		{"proc/sys/user/max_user_namespaces", "userns-limit", "-1\n", "unknown", 0644},
		{"proc/sys/user/max_user_namespaces", "userns-limit", "secret\n", "unknown", 0644},
		{"sys/fs/cgroup/cgroup.controllers", "cgroup-v2", "cpu memory\n", "blocked", 0644},
	} {
		t.Run(tc.id+"/"+tc.status, func(t *testing.T) {
			p, f := fixture()
			f.files[tc.path] = &fstest.MapFile{Data: []byte(tc.text), Mode: tc.mode}
			check(t, p.inspect(), tc.id, tc.status)
			if tc.mode&fs.ModeType != 0 && slices.Contains(f.opened, tc.path) {
				t.Fatal("special file was opened")
			}
		})
	}
}

func TestInspectionIsRedactedAndLeavesFixtureUnchanged(t *testing.T) {
	p, f := fixture()
	t.Setenv("DOCKER_HOST", "tcp://private-host:secret-token@private.example")
	t.Setenv("DOCKER_CONTEXT", "secret-context-name")
	_, p.dockerHost = os.LookupEnv("DOCKER_HOST")
	_, p.dockerContext = os.LookupEnv("DOCKER_CONTEXT")
	f.files[strings.TrimPrefix(rootless.InstallDir, "/")+"/dockerd-rootless.sh"] = &fstest.MapFile{Mode: 0777}
	f.denied["etc/group"] = true
	before, err := json.Marshal(f.files)
	if err != nil {
		t.Fatal(err)
	}
	r := p.inspect()
	check(t, r, "installation-verification", "unknown")
	if c := check(t, r, "docker-environment", "info"); !strings.Contains(c.Detail, "DOCKER_HOST present: true; DOCKER_CONTEXT present: true") {
		t.Fatal(c.Detail)
	}
	output, err := json.Marshal(struct {
		Report Report
		Plan   SetupPlan
	}{r, Plan(r)})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"Private GECOS", "Private Desktop", "/home/desktop", "100000", "200000", "942", "secret-token", "private.example", "secret-context-name", "shadow-secret", "config-secret", "private-error", "private-path"} {
		if bytes.Contains(output, []byte(secret)) {
			t.Fatalf("private fixture data leaked: %q", secret)
		}
	}
	for _, path := range f.opened {
		if strings.Contains(path, "shadow") || strings.Contains(path, ".docker") || strings.Contains(path, "daemon.json") || strings.Contains(path, "docker-compose") || strings.Contains(path, "docker-buildx") || strings.Contains(path, "docker-rootless") || strings.HasPrefix(path, "usr/bin/") || strings.HasSuffix(path, ".sock") {
			t.Fatalf("forbidden open: %s", path)
		}
	}
	after, _ := json.Marshal(f.files)
	if !bytes.Equal(before, after) {
		t.Fatal("fixture was modified")
	}
}

// A proc-like regular file can advertise size zero and still stream indefinitely.
type streamingFile struct {
	fs.FileInfo
	read int
}

func (f *streamingFile) Read(b []byte) (int, error) { f.read += len(b); return len(b), nil }
func (f *streamingFile) Close() error               { return nil }
func (f *streamingFile) Stat() (fs.FileInfo, error) { return f.FileInfo, nil }

type streamingFS struct{ file *streamingFile }

func (f streamingFS) Open(string) (fs.File, error)     { return f.file, nil }
func (f streamingFS) Stat(string) (fs.FileInfo, error) { return f.file.FileInfo, nil }

func TestBoundedRead(t *testing.T) {
	m := fstest.MapFS{"control": &fstest.MapFile{Mode: 0644}}
	info, err := m.Stat("control")
	if err != nil {
		t.Fatal(err)
	}
	f := &streamingFile{FileInfo: info}
	p := probe{files: streamingFS{f}}
	if _, err := p.read("control"); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("unbounded streaming input accepted: %v", err)
	}
	if f.read != maxRead+1 {
		t.Fatalf("read %d bytes, want bounded sentinel %d", f.read, maxRead+1)
	}
	m["control"].Data = bytes.Repeat([]byte("x"), maxRead)
	p.files = m
	if text, err := p.read("control"); err != nil || len(text) != maxRead {
		t.Fatalf("exact limit rejected: %d, %v", len(text), err)
	}
}
