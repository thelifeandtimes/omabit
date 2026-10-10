package rootless

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Exact upstream bytes let tests exercise the production pins without network
// access, hash overrides, or an executable launcher in the source tree.
//
//go:embed assets/upstream/dockerd-rootless.sh assets/upstream/LICENSE assets/upstream/NOTICE
var upstreamFixtures embed.FS

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type readCloseFunc struct {
	read  func([]byte) (int, error)
	close func() error
}

func (b readCloseFunc) Read(p []byte) (int, error) { return b.read(p) }
func (b readCloseFunc) Close() error               { return b.close() }

func fixtureClient(t *testing.T, mutate func(*http.Request, *http.Response) (*http.Response, error)) *http.Client {
	t.Helper()
	client := newDownloadClient()
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		name, ok := map[string]string{
			launcherURL: "dockerd-rootless.sh", sourceURL + "LICENSE": "LICENSE", sourceURL + "NOTICE": "NOTICE",
		}[r.URL.String()]
		if !ok || r.Method != http.MethodGet || r.Header.Get("Accept-Encoding") != "identity" {
			t.Errorf("unexpected request: %s %s, headers %v", r.Method, r.URL, r.Header)
			return nil, errors.New("unexpected request; network is disabled")
		}
		data, err := upstreamFixtures.ReadFile("assets/upstream/" + name)
		if err != nil {
			t.Fatal(err)
		}
		resp := &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header), Request: r,
			ContentLength: int64(len(data)), Body: io.NopCloser(bytes.NewReader(data)),
		}
		if mutate != nil {
			return mutate(r, resp)
		}
		return resp, nil
	})
	return client
}

func requireNonRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("preparation intentionally refuses root; run these tests as a regular user")
	}
}

func assertEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entries in %s: got %v, want %v", dir, got, want)
	}
}

func TestDownloadPolicy(t *testing.T) {
	client := newDownloadClient()
	defer client.CloseIdleConnections()
	transport := client.Transport.(*http.Transport)
	if client.Timeout != 30*time.Second || transport.Proxy != nil || !transport.DisableCompression {
		t.Fatal("download client must have a 30-second timeout, no proxy, and no compression")
	}
	if transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS certificate verification is disabled")
	}
	if err := client.CheckRedirect(nil, nil); err != http.ErrUseLastResponse {
		t.Fatalf("redirect policy: %v", err)
	}
}

func TestUnitPinsPrivateSocketGroup(t *testing.T) {
	unit, err := renderedAsset(UnitName)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(unit), "--group=0 --host=unix://%t/"+SocketName) {
		t.Fatal("socket must use namespace GID 0, which maps to the dedicated account's host GID")
	}
}

func TestVerifiedFilesDoNotTrustManifest(t *testing.T) {
	requireNonRoot(t)
	dir := filepath.Join(t.TempDir(), "bundle")
	if _, err := prepare(context.Background(), dir, fixtureClient(t, nil)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"files":[{"name":"../../etc/passwd","sha256":"forged"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := VerifiedFiles(dir)
	if err != nil || len(files) != 6 {
		t.Fatalf("verification: files=%d err=%v", len(files), err)
	}
	for _, name := range []string{"dockerd-rootless.sh", UnitName, "check-engine.sh", "daemon.json", "LICENSE", "NOTICE"} {
		original := files[name]
		if err := os.WriteFile(filepath.Join(dir, name), []byte("tampered"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := VerifiedFiles(dir); err == nil {
			t.Fatalf("accepted tampered %s", name)
		}
		if err := os.WriteFile(filepath.Join(dir, name), original, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Installer gets a snapshot, not a path to reopen after authorization.
	originalConfig := bytes.Clone(files["daemon.json"])
	if err := os.WriteFile(filepath.Join(dir, "daemon.json"), []byte("changed after verification"), 0600); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(files["daemon.json"], originalConfig) {
		t.Fatalf("verified bytes changed: %q", files["daemon.json"])
	}
}

func TestVerifiedFilesRejectSymlinksAndOversize(t *testing.T) {
	requireNonRoot(t)
	dir := filepath.Join(t.TempDir(), "bundle")
	if _, err := prepare(context.Background(), dir, fixtureClient(t, nil)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "daemon.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("LICENSE", path); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifiedFiles(dir); err == nil {
		t.Fatal("accepted symlink artifact")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, maxDownloadBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifiedFiles(dir); err == nil {
		t.Fatal("accepted oversized artifact")
	}
	for _, path := range []string{"relative", dir + "/../bundle", filepath.Join(dir, "missing")} {
		if _, err := VerifiedFiles(path); err == nil {
			t.Fatalf("accepted invalid path %q", path)
		}
	}
}

func TestPrepareBundle(t *testing.T) {
	requireNonRoot(t)
	for _, mode := range []os.FileMode{0700, 0755} {
		t.Run(fmt.Sprintf("parent-%o", mode), func(t *testing.T) {
			parent := t.TempDir()
			if err := os.Chmod(parent, mode); err != nil {
				t.Fatal(err)
			}
			untouched := t.TempDir()
			// No Docker client, service manager, or helper is needed or invoked.
			for _, key := range []string{"HOME", "DOCKER_CONFIG", "XDG_CONFIG_HOME", "XDG_RUNTIME_DIR", "PATH"} {
				t.Setenv(key, untouched)
			}
			t.Setenv("DOCKER_HOST", "unix:///do-not-touch/docker.sock")
			t.Setenv("DOCKER_CONTEXT", "do-not-touch")
			var requests []string
			client := fixtureClient(t, func(r *http.Request, resp *http.Response) (*http.Response, error) {
				requests = append(requests, r.URL.String())
				return resp, nil
			})
			output := filepath.Join(parent, "bundle")
			manifest, err := prepare(context.Background(), output, client)
			if err != nil {
				t.Fatal(err)
			}
			assertEntries(t, untouched)
			assertEntries(t, parent, "bundle")
			if want := []string{launcherURL, sourceURL + "LICENSE", sourceURL + "NOTICE"}; !reflect.DeepEqual(requests, want) {
				t.Fatalf("downloads: %v", requests)
			}
			if manifest.SchemaVersion != 1 || manifest.EngineVersion != EngineVersion || manifest.SourceCommit != SourceCommit ||
				manifest.LauncherURL != launcherURL || manifest.LauncherSHA256 != LauncherSHA256 ||
				manifest.Account != AccountName || manifest.Home != HomeDir || manifest.InstallDir != InstallDir ||
				manifest.Unit != UnitName || manifest.UnitPath != "/etc/systemd/user/"+UnitName ||
				!strings.Contains(manifest.TrustNote, "compiled-in pins/templates, not this manifest") {
				t.Fatalf("unexpected manifest: %+v", manifest)
			}
			data, err := os.ReadFile(filepath.Join(output, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var decoded Manifest
			if err := json.Unmarshal(data, &decoded); err != nil || !reflect.DeepEqual(decoded, manifest) {
				t.Fatalf("manifest roundtrip: %v, %+v", err, decoded)
			}
			assertEntries(t, output, "LICENSE", "NOTICE", "check-engine.sh", "daemon.json", "dockerd-rootless.sh", "manifest.json", UnitName)
			if len(manifest.Files) != 6 {
				t.Fatalf("manifest has %d artifacts", len(manifest.Files))
			}
			seen := make(map[string]bool)
			for _, file := range manifest.Files {
				if seen[file.Name] || file.Name != filepath.Base(file.Name) || file.Name == "manifest.json" {
					t.Fatalf("invalid inventory entry: %+v", file)
				}
				seen[file.Name] = true
				data, err := os.ReadFile(filepath.Join(output, file.Name))
				if err != nil || file.SHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) || file.Mode != "0644" {
					t.Fatalf("artifact %s: %v, %+v", file.Name, err, file)
				}
				if file.SourceURL != "" {
					fixture, err := upstreamFixtures.ReadFile("assets/upstream/" + file.Name)
					if err != nil || !bytes.Equal(data, fixture) {
						t.Fatalf("modified upstream file %s: %v", file.Name, err)
					}
				} else if bytes.Contains(data, []byte("@")) {
					t.Fatalf("unexpanded placeholder in %s", file.Name)
				}
			}
			for _, name := range []string{"LICENSE", "NOTICE", "check-engine.sh", "daemon.json", "dockerd-rootless.sh", "manifest.json", UnitName} {
				info, err := os.Lstat(filepath.Join(output, name))
				if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0644 {
					t.Fatalf("artifact mode %s: %v, %v", name, info, err)
				}
			}
			for path, want := range map[string]os.FileMode{parent: mode, output: 0700} {
				info, err := os.Stat(path)
				if err != nil || info.Mode().Perm() != want {
					t.Fatalf("directory mode %s: %v, %v", path, info, err)
				}
			}
		})
	}
}

func TestPrepareDownloadFailuresCleanUp(t *testing.T) {
	requireNonRoot(t)
	for _, failure := range []string{
		"launcher-hash", "license-hash", "notice-hash", "404", "500", "redirect",
		"oversize-header", "oversize-body", "compressed", "read-error", "timeout", "body-timeout", "cancel",
	} {
		t.Run(failure, func(t *testing.T) {
			parent := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			requests := 0
			closed := false
			client := fixtureClient(t, func(r *http.Request, resp *http.Response) (*http.Response, error) {
				requests++
				reader := resp.Body
				resp.Body = readCloseFunc{read: reader.Read, close: func() error { closed = true; return reader.Close() }}
				switch failure {
				case "launcher-hash", "license-hash", "notice-hash":
					target := map[string]string{"launcher-hash": launcherURL, "license-hash": sourceURL + "LICENSE", "notice-hash": sourceURL + "NOTICE"}[failure]
					if r.URL.String() == target {
						resp.Body = io.NopCloser(strings.NewReader("not the pinned content"))
					}
				case "404", "500":
					resp.StatusCode = map[string]int{"404": 404, "500": 500}[failure]
				case "redirect":
					resp.StatusCode = http.StatusFound
					resp.Header.Set("Location", "https://example.invalid/not-allowed")
				case "oversize-header":
					resp.ContentLength = maxDownloadBytes + 1
				case "oversize-body":
					resp.ContentLength = -1
					resp.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", maxDownloadBytes+1)))
				case "compressed":
					resp.Header.Set("Content-Encoding", "gzip")
				case "read-error":
					resp.Body = readCloseFunc{read: func([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }, close: resp.Body.Close}
				case "timeout":
					<-r.Context().Done()
					return nil, r.Context().Err()
				case "body-timeout":
					resp.Body = readCloseFunc{read: func([]byte) (int, error) {
						<-r.Context().Done()
						return 0, r.Context().Err()
					}, close: resp.Body.Close}
				case "cancel":
					cancel()
				}
				return resp, nil
			})
			if strings.Contains(failure, "timeout") {
				client.Timeout = 10 * time.Millisecond
			}
			manifest, err := prepare(ctx, filepath.Join(parent, "bundle"), client)
			if err == nil || !reflect.DeepEqual(manifest, Manifest{}) {
				t.Fatalf("expected failure and empty manifest: %+v, %v", manifest, err)
			}
			want := ""
			switch {
			case strings.HasSuffix(failure, "hash"):
				want = "SHA-256 mismatch"
			case strings.HasPrefix(failure, "oversize"):
				want = "exceeds 1 MiB"
			case failure == "404" || failure == "500" || failure == "redirect":
				want = "expected HTTP 200"
			case failure == "compressed":
				want = "content encoding"
			case failure == "read-error":
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Fatalf("read failure: %v", err)
				}
			case failure == "cancel":
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation: %v", err)
				}
			case strings.Contains(failure, "timeout"):
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("timeout: %v", err)
				}
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("wanted %q: %v", want, err)
			}
			wantRequests := 1
			if failure == "license-hash" {
				wantRequests = 2
			} else if failure == "notice-hash" {
				wantRequests = 3
			}
			if requests != wantRequests {
				t.Fatalf("made %d requests, want %d", requests, wantRequests)
			}
			if failure != "timeout" && failure != "launcher-hash" && failure != "oversize-body" && !closed {
				t.Fatal("response body was not closed")
			}
			assertEntries(t, parent)
		})
	}
}

func TestPrepareRefusesUnsafeOutputs(t *testing.T) {
	requireNonRoot(t)
	for _, kind := range []string{"relative", "unclean", "missing-parent", "file", "directory", "symlink", "dangling-symlink", "parent-symlink", "ancestor-symlink", "world-writable", "group-writable", "foreign-parent"} {
		t.Run(kind, func(t *testing.T) {
			parent := t.TempDir()
			output := filepath.Join(parent, "bundle")
			var err error
			switch kind {
			case "relative":
				output = "relative-bundle"
			case "unclean":
				output = parent + "/child/../bundle"
			case "missing-parent":
				output = filepath.Join(parent, "absent", "bundle")
			case "file":
				err = os.WriteFile(output, []byte("untouched"), 0600)
			case "directory":
				err = os.Mkdir(output, 0700)
			case "symlink", "dangling-symlink", "parent-symlink", "ancestor-symlink":
				target := parent
				if kind == "dangling-symlink" {
					target = filepath.Join(parent, "absent")
				}
				err = os.Symlink(target, output)
				if kind == "parent-symlink" {
					output = filepath.Join(output, "child")
				} else if kind == "ancestor-symlink" {
					if err := os.Mkdir(filepath.Join(parent, "child"), 0700); err != nil {
						t.Fatal(err)
					}
					output = filepath.Join(output, "child", "output")
				}
			case "world-writable":
				err = os.Chmod(parent, 0702)
			case "group-writable":
				err = os.Chmod(parent, 0720)
			case "foreign-parent":
				output = "/omabit-rootless-must-not-be-created"
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(output)
			existed := err == nil
			client := fixtureClient(t, func(*http.Request, *http.Response) (*http.Response, error) {
				t.Error("invalid output attempted a download")
				return nil, errors.New("network forbidden")
			})
			if _, err := prepare(context.Background(), output, client); err == nil {
				t.Fatal("unsafe output accepted")
			}
			after, err := os.Lstat(output)
			if existed && (err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode()) {
				t.Fatalf("existing output changed: %v", err)
			}
			if !existed && !os.IsNotExist(err) {
				t.Fatalf("output unexpectedly created: %v", err)
			}
			if kind == "file" {
				data, err := os.ReadFile(output)
				if err != nil || string(data) != "untouched" {
					t.Fatal("existing file content changed")
				}
			}
		})
	}
}

func TestPrepareCanceledBeforeWork(t *testing.T) {
	requireNonRoot(t)
	parent := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Prepare(ctx, filepath.Join(parent, "bundle")); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled preparation: %v", err)
	}
	assertEntries(t, parent)
}

func TestPrepareRootRefused(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("root refusal is tested only when already running as root; never elevate tests")
	}
	parent := t.TempDir()
	if _, err := Prepare(context.Background(), filepath.Join(parent, "bundle")); err == nil || !strings.Contains(err.Error(), "must not run as root") {
		t.Fatalf("root preparation: %v", err)
	}
	assertEntries(t, parent)
}

func TestPrepareExclusiveReservation(t *testing.T) {
	requireNonRoot(t)
	parent := t.TempDir()
	output := filepath.Join(parent, "bundle")
	client := fixtureClient(t, func(r *http.Request, resp *http.Response) (*http.Response, error) {
		if r.URL.String() == launcherURL {
			assertEntries(t, output)
			other := fixtureClient(t, func(*http.Request, *http.Response) (*http.Response, error) {
				t.Error("competing preparation made a download")
				return nil, errors.New("network forbidden")
			})
			if _, err := prepare(context.Background(), output, other); !errors.Is(err, os.ErrExist) {
				t.Fatalf("competing preparation did not lose mkdir reservation: %v", err)
			}
		}
		return resp, nil
	})
	if _, err := prepare(context.Background(), output, client); err != nil {
		t.Fatal(err)
	}
	if _, err := prepare(context.Background(), output, client); !errors.Is(err, os.ErrExist) {
		t.Fatalf("repeated preparation overwrote output: %v", err)
	}
	assertEntries(t, parent, "bundle")
}

func TestPreparePreservesChangedReservation(t *testing.T) {
	requireNonRoot(t)
	for _, change := range []string{"nonempty", "replacement-directory", "replacement-symlink"} {
		t.Run(change, func(t *testing.T) {
			parent := t.TempDir()
			output := filepath.Join(parent, "bundle")
			var preserved os.FileInfo
			client := fixtureClient(t, func(r *http.Request, resp *http.Response) (*http.Response, error) {
				if r.URL.String() == launcherURL {
					if change == "nonempty" {
						if err := os.WriteFile(filepath.Join(output, "keep"), []byte("untouched"), 0600); err != nil {
							t.Fatal(err)
						}
					} else {
						if err := os.Rename(output, filepath.Join(parent, "moved")); err != nil {
							t.Fatal(err)
						}
						var err error
						if change == "replacement-directory" {
							err = os.Mkdir(output, 0700)
						} else {
							err = os.Symlink(filepath.Join(parent, "moved"), output)
						}
						if err != nil {
							t.Fatal(err)
						}
					}
					var err error
					preserved, err = os.Lstat(output)
					if err != nil {
						t.Fatal(err)
					}
				}
				return resp, nil
			})
			if _, err := prepare(context.Background(), output, client); err == nil {
				t.Fatal("published over changed reservation")
			}
			info, err := os.Lstat(output)
			if err != nil || !os.SameFile(preserved, info) {
				t.Fatalf("removed someone else's output: %v", err)
			}
			if change == "nonempty" {
				assertEntries(t, parent, "bundle")
				data, err := os.ReadFile(filepath.Join(output, "keep"))
				if err != nil || string(data) != "untouched" {
					t.Fatal("removed someone else's contents")
				}
			} else {
				assertEntries(t, parent, "bundle", "moved")
			}
		})
	}
}

func TestRenderedAssetsAndShellSyntax(t *testing.T) {
	requireNonRoot(t)
	output := filepath.Join(t.TempDir(), "bundle")
	if _, err := prepare(context.Background(), output, fixtureClient(t, nil)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"check-engine.sh", "dockerd-rootless.sh"} {
		// Parsing only. Never run either script, not even against fake binaries.
		cmd := exec.Command("/bin/sh", "-n", filepath.Join(output, name))
		if data, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("shell syntax %s: %v\n%s", name, err, data)
		}
	}
	data, err := os.ReadFile(filepath.Join(output, UnitName))
	if err != nil {
		t.Fatal(err)
	}
	unit := string(data)
	for _, line := range []string{
		"ConditionUser=" + AccountName, "Requires=dbus.socket", "After=dbus.socket",
		"Type=notify", "NotifyAccess=all", "Environment=PATH=/usr/bin:/bin", "Environment=HOME=" + HomeDir,
		"Environment=DOCKERD=/usr/bin/dockerd", "Environment=DOCKERD_ROOTLESS_ROOTLESSKIT_NET=slirp4netns",
		"Environment=DOCKERD_ROOTLESS_ROOTLESSKIT_PORT_DRIVER=builtin", "Environment=DOCKERD_ROOTLESS_ROOTLESSKIT_STATE_DIR=%t/urbit-rootlesskit",
		"ExecStartPre=" + InstallDir + "/check-engine.sh",
		"ExecStart=" + InstallDir + "/dockerd-rootless.sh --group=0 --host=unix://%t/" + SocketName + " --data-root=%h/.local/share/urbit-runner/docker --exec-root=%t/urbit-docker --pidfile=%t/urbit-docker.pid --exec-opt native.cgroupdriver=systemd --config-file " + InstallDir + "/daemon.json",
		"Delegate=yes", "KillMode=mixed", "Restart=on-failure", "RestartSec=2", "TimeoutStartSec=120", "TimeoutStopSec=360",
		"TasksMax=infinity", "LimitNOFILE=infinity", "LimitNPROC=infinity", "LimitCORE=0",
	} {
		if !strings.Contains("\n"+unit, "\n"+line+"\n") {
			t.Errorf("missing unit directive %q", line)
		}
	}
	for _, forbidden := range []string{"\nNoNewPrivileges=", "tcp://", "Requires=docker.service", "After=docker.service", "docker.socket", "dockerd-rootless-setuptool"} {
		if strings.Contains(unit, forbidden) {
			t.Errorf("unsafe unit content %q", forbidden)
		}
	}
	data, err = os.ReadFile(filepath.Join(output, "check-engine.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"\"$1\"", "id -u", "id -un", "'" + AccountName + "'", "'" + HomeDir + "'", "stat -Lc %u /usr/bin/dockerd", "/usr/bin/dockerd --version", "'Docker version " + EngineVersion + ", build '?*)", LauncherSHA256, InstallDir + "/dockerd-rootless.sh", "sha256sum --check --status -"} {
		if !strings.Contains(string(data), required) {
			t.Errorf("missing preflight check %q", required)
		}
	}
	data, err = os.ReadFile(filepath.Join(output, "daemon.json"))
	var config map[string]int
	if err != nil || json.Unmarshal(data, &config) != nil || len(config) != 1 || config["shutdown-timeout"] != 310 {
		t.Fatalf("daemon config: %q, %v", data, err)
	}
}
