//go:build linux

package hostcompat

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

func repositoryFixture(t *testing.T, p nativePackage, filename string, artifact, sig []byte) []byte {
	t.Helper()
	h := sha256.Sum256(artifact)
	desc := fmt.Sprintf("%%NAME%%\n%s\n\n%%VERSION%%\n%s\n\n%%ARCH%%\n%s\n\n%%FILENAME%%\n%s\n\n%%CSIZE%%\n%d\n\n%%SHA256SUM%%\n%x\n\n%%PGPSIG%%\n%s\n", p.name, p.version, p.arch, filename, len(artifact), h, base64.StdEncoding.EncodeToString(sig))
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Write(makeTar(t, map[string][]byte{p.name + "-" + p.version + "/desc": []byte(desc)}))
	gz.Close()
	return b.Bytes()
}

func fixtureFetcher(t *testing.T, m fstest.MapFS, v *verifier, run fetchRunner) *nativeFetcher {
	t.Helper()
	stage := "var/lib/omabit/hostcompat/.fetch-fixture"
	addFixture(m, stage, nil, fs.ModeDir|0700)
	f := &nativeFetcher{v: v, stage: stage, run: run, mkdir: func(name string) error {
		if !strings.HasPrefix(name, stage+"/") {
			return errEvidence
		}
		if m[name] == nil {
			addFixture(m, name, nil, fs.ModeDir|0700)
		}
		return nil
	}, write: func(name string, b []byte) error {
		if !strings.HasPrefix(name, stage+"/") {
			return errEvidence
		}
		addFixture(m, name, append([]byte{}, b...), 0600)
		return nil
	}}
	v.fetch = f.fetch
	return f
}

func TestEpochBearingPacmanCacheAndEmbeddedRepositorySignature(t *testing.T) {
	for _, epochFilename := range []bool{false, true} {
		t.Run(fmt.Sprint(epochFilename), func(t *testing.T) {
			m, home, old := authenticatedPacmanFixture(t)
			// Real local DB directory name includes the epoch, independently of
			// whether the repository chose an epoch-bearing archive filename.
			for name, file := range m {
				if strings.HasPrefix(name, "var/lib/pacman/local/docker-29.9.1-1") {
					next := strings.Replace(name, "docker-29.9.1-1", "docker-1:29.9.1-1", 1)
					m[next] = file
					delete(m, name)
				}
			}
			artifact := old
			if epochFilename {
				artifact = strings.Replace(old, "docker-29.9.1-1", "docker-1:29.9.1-1", 1)
				m[artifact] = m[old]
				delete(m, old)
			}
			sig := append([]byte{}, m[old+".sig"].Data...)
			delete(m, old+".sig")
			s := inspect(m, "linux", "amd64")
			owner := s.owners["engine"]
			if s.report.Packages[0].Version != "1:29.9.1-1" {
				t.Fatal("epoch directory/version metadata lost")
			}
			addFixture(m, "var/lib/pacman/sync/extra.db", repositoryFixture(t, owner, path.Base(artifact), m[artifact].Data, sig), 0644)
			v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
			if _, err := v.verifyPackage(context.Background(), "pacman", s.report.Packages[0], owner); err != nil {
				t.Fatalf("native PGPSIG authentication: %v", err)
			}
			m["var/lib/pacman/sync/extra.db"].Data = repositoryFixture(t, owner, path.Base(artifact), m[artifact].Data, []byte("forged repository signature assertion"))
			v = verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
			if _, err := v.verifyPackage(context.Background(), "pacman", s.report.Packages[0], owner); err == nil {
				t.Fatal("untrusted repository metadata was authentication")
			}
		})
	}
}

func TestActualArchDownloadOnlyBootstrapUsingNativeTools(t *testing.T) {
	for _, tool := range []string{"/usr/bin/pacman-conf", "/usr/bin/curl"} {
		if _, err := os.Stat(tool); err != nil {
			t.Skip("native Arch download tools unavailable")
		}
	}
	m, home, archiveName := authenticatedPacmanFixture(t)
	s := inspect(m, "linux", "amd64")
	owner := s.owners["engine"]
	artifact := append([]byte{}, m[archiveName].Data...)
	sig := append([]byte{}, m[archiveName+".sig"].Data...)
	filename := "docker-1:29.9.1-1-x86_64.pkg.tar.gz"
	database := repositoryFixture(t, owner, filename, artifact, sig)
	requests := []string{}
	var requestsMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestsMu.Lock()
		requests = append(requests, r.URL.Path)
		requestsMu.Unlock()
		switch path.Base(r.URL.Path) {
		case "extra.db":
			w.Write(database)
		case filename:
			w.Write(artifact)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	config := "[options]\nArchitecture = x86_64\nSigLevel = Required DatabaseOptional\n[extra]\nServer = " + server.URL + "/$repo/os/$arch\n"
	addFixture(m, "etc/pacman.conf", []byte(config), 0644)
	delete(m, archiveName)
	delete(m, archiveName+".sig")
	physical := t.TempDir()
	configPath := filepath.Join(physical, "pacman.conf")
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	var commands []fetchCommand
	run := func(ctx context.Context, c fetchCommand) ([]byte, error) {
		commands = append(commands, c)
		args := append([]string{}, c.Args...)
		output := ""
		physicalOutput := ""
		for i, arg := range args {
			if i > 0 && args[i-1] == "--config" {
				args[i] = configPath
			}
			if i > 0 && args[i-1] == "--output" {
				output = strings.TrimPrefix(arg, "/")
				physicalOutput = filepath.Join(physical, output)
				if err := os.MkdirAll(filepath.Dir(physicalOutput), 0700); err != nil {
					return nil, err
				}
				args[i] = physicalOutput
			}
		}
		cmd := exec.CommandContext(ctx, c.Tool, args...)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + physical, "LANG=C", "LC_ALL=C"}
		cmd.Dir = physical
		b, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		if output != "" {
			data, err := os.ReadFile(physicalOutput)
			if err != nil {
				return nil, err
			}
			addFixture(m, output, data, 0600)
		}
		return b, nil
	}
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	f := fixtureFetcher(t, m, &v, run)
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || len(set.Packages) != 2 {
		t.Fatalf("Arch bootstrap did not authenticate: %v %+v; calls %+v", err, set.Packages, commands)
	}
	for _, rec := range set.Packages {
		if !rec.Authorized {
			t.Fatal("native fetched bytes not authenticated")
		}
	}
	requestsMu.Lock()
	gotRequests := append([]string{}, requests...)
	requestsMu.Unlock()
	if len(gotRequests) != 2 {
		t.Fatalf("unexpected package/network requests: %v", gotRequests)
	}
	for _, command := range commands {
		if command.Tool == "/usr/bin/curl" {
			if command.Args[0] != "--disable" || !containsString(command.Args, "--max-filesize") || !containsString(command.Args, "--proto") {
				t.Fatal("download argv lost bounds")
			}
		}
		if command.Tool != "/usr/bin/curl" && command.Tool != "/usr/bin/pacman-conf" {
			t.Fatal("package-mutating native tool invoked")
		}
	}
	publishFixtureSet(t, m, set)
	for name := range m {
		if strings.HasPrefix(name, f.stage) {
			delete(m, name)
		}
	}
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance != "verified" {
		t.Fatal("fetched authentication depended on staging availability")
	}
}

func TestAPTFetchUsesPrivateNativeStateAndExactInstalledBuild(t *testing.T) {
	m, run, _, _ := dockerAPTFixture(t, true)
	original := map[string][]byte{}
	for name, file := range m {
		if strings.HasPrefix(name, "var/cache/apt/archives/") || strings.HasPrefix(name, "var/lib/apt/lists/") {
			original[name] = append([]byte{}, file.Data...)
			delete(m, name)
		}
	}
	s := inspect(m, "linux", "amd64")
	owner := s.owners["engine"]
	var commands []fetchCommand
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	f := fixtureFetcher(t, m, &v, func(ctx context.Context, c fetchCommand) ([]byte, error) {
		commands = append(commands, c)
		if c.Tool != "/usr/bin/apt-get" || c.APTConfig == "" || !strings.Contains(c.APTConfig, "/.fetch-fixture/apt/apt.conf") {
			return nil, errFetch
		}
		base := strings.TrimSuffix(strings.TrimPrefix(c.APTConfig, "/"), "/apt.conf")
		switch {
		case reflect.DeepEqual(c.Args, []string{"-qq", "update"}):
			for name, b := range original {
				if strings.HasPrefix(name, "var/lib/apt/lists/") {
					addFixture(m, base+"/lists/"+path.Base(name), b, 0644)
				}
			}
		case reflect.DeepEqual(c.Args, []string{"-qq", "download", "--", owner.name + ":" + owner.arch + "=" + owner.version}):
			for name, b := range original {
				if strings.HasPrefix(name, "var/cache/apt/archives/") {
					addFixture(m, base+"/archives/"+path.Base(name), b, 0644)
				}
			}
		default:
			return nil, errors.New("unexpected download-only argv")
		}
		return nil, nil
	})
	set, _, err := v.refreshPackages(context.Background(), s, receiptSet{})
	if err != nil || len(set.Packages) != 2 {
		t.Fatalf("APT bootstrap failed: %v %+v", err, set.Packages)
	}
	for _, rec := range set.Packages {
		if !rec.Authorized {
			t.Fatal("downloaded APT evidence was not crypto/payload verified")
		}
	}
	if len(commands) != 2 {
		t.Fatalf("expected private index update and one exact download: %+v", commands)
	}
	config := string(m[f.stage+"/apt/apt.conf"].Data)
	for _, required := range []string{"Dir::Etc::parts", "empty-config", "Dir::State::lists", f.stage + "/apt/lists", `APT::Get::AllowUnauthenticated "false"`, `Acquire::AllowInsecureRepositories "false"`, `Acquire::https::Verify-Peer "true"`} {
		if !strings.Contains(config, required) {
			t.Fatalf("private APT policy missing %q", required)
		}
	}
	if strings.Contains(config, "Post-Invoke") || strings.Contains(config, `Dir::State::lists "/var/lib/apt/lists"`) {
		t.Fatal("global native hook/state mutation possible")
	}
	if strings.Contains(string(m[f.stage+"/apt/sources.sources"].Data), "Trusted:") {
		t.Fatal("native source trust was overridden instead of deriving it from Signed-By")
	}
	publishFixtureSet(t, m, set)
	for name := range m {
		if strings.HasPrefix(name, f.stage) {
			delete(m, name)
		}
	}
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance != "verified" {
		t.Fatal("APT boot required download staging")
	}
}

func TestFetchFailsClosedForInsecureTraversalCancellationAndUnavailableBuilds(t *testing.T) {
	t.Run("APT-insecure", func(t *testing.T) {
		m, run, _, _ := dockerAPTFixture(t, false)
		m["etc/apt/sources.list.d/docker.sources"].Data = append(m["etc/apt/sources.list.d/docker.sources"].Data, []byte("Trusted: yes\n")...)
		v := verifier{fsys: m, run: run, store: fixtureStore(m)}
		calls := 0
		f := fixtureFetcher(t, m, &v, func(context.Context, fetchCommand) ([]byte, error) { calls++; return nil, nil })
		owner := inspect(m, "linux", "amd64").owners["engine"]
		if f.fetch(context.Background(), "deb", owner) == nil || calls != 0 {
			t.Fatal("insecure APT config reached a downloader")
		}
	})
	t.Run("cancel-before-native", func(t *testing.T) {
		m := debFixture()
		v := verifier{fsys: m}
		calls := 0
		f := fixtureFetcher(t, m, &v, func(context.Context, fetchCommand) ([]byte, error) { calls++; return nil, nil })
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if f.fetch(ctx, "deb", inspect(m, "linux", "amd64").owners["engine"]) == nil || calls != 0 {
			t.Fatal("canceled fetch executed")
		}
	})
	t.Run("unavailable-version-no-secret-error", func(t *testing.T) {
		m, run, _, _ := dockerAPTFixture(t, false)
		v := verifier{fsys: m, run: run, store: fixtureStore(m)}
		f := fixtureFetcher(t, m, &v, func(_ context.Context, c fetchCommand) ([]byte, error) {
			if containsString(c.Args, "download") {
				return nil, errors.New("https://private:secret@example.invalid package unavailable")
			}
			return nil, nil
		})
		err := f.fetch(context.Background(), "deb", inspect(m, "linux", "amd64").owners["engine"])
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("unavailable version accepted or raw credential leaked")
		}
	})
	t.Run("download-success-is-not-authentication", func(t *testing.T) {
		m, run, _, _ := dockerAPTFixture(t, false)
		for name := range m {
			if strings.HasPrefix(name, "var/cache/apt/archives/") || strings.HasPrefix(name, "var/lib/apt/lists/") {
				delete(m, name)
			}
		}
		v := verifier{fsys: m, run: run, store: fixtureStore(m)}
		fixtureFetcher(t, m, &v, func(context.Context, fetchCommand) ([]byte, error) { return nil, nil })
		set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
		if err != nil {
			t.Fatal(err)
		}
		for _, rec := range set.Packages {
			if rec.Authorized {
				t.Fatal("native exit success authenticated missing/unknown package")
			}
		}
	})
	for _, uri := range []string{"file:///etc/shadow", "https://user:secret@example.invalid/pkg", "https://example.invalid/a/../b", "https://example.invalid/%2e%2e/b", "https://example.invalid/pkg\n--output=/outside"} {
		if safeFetchURL(uri) {
			t.Fatalf("unsafe URL accepted: %q", uri)
		}
	}
	p := nativePackage{name: "docker", version: "1:29.9.1-1", arch: "x86_64"}
	for _, name := range []string{"../docker-29.9.1-1-x86_64.pkg.tar.zst", "docker-1:29.9.1-1-x86_64.pkg.tar.zst/../../outside", "docker-1:29.9.1-1-x86_64.pkg.tar.%2foutside"} {
		if pacmanArchiveName(name, p) {
			t.Fatal("archive path traversal accepted")
		}
	}
	if securePacmanSigLevel([]byte("PackageNever PackageTrustAll")) || securePacmanSigLevel([]byte("PackageOptional PackageTrustedOnly")) {
		t.Fatal("insecure Pacman download authority accepted")
	}
}

func TestCachedVerificationDoesNotInvokeOptionalFetch(t *testing.T) {
	m, home, _ := authenticatedPacmanFixture(t)
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	calls := 0
	fixtureFetcher(t, m, &v, func(context.Context, fetchCommand) ([]byte, error) { calls++; return nil, errFetch })
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || len(set.Packages) != 2 || calls != 0 {
		t.Fatalf("cached verification used network: %v %d", err, calls)
	}
}

func TestHistoricalReceiptRejectsInvalidAndFutureTimestamps(t *testing.T) {
	m, home, _ := authenticatedPacmanFixture(t)
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	set, _, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []time.Time{{}, time.Now().Add(time.Hour), time.Unix(-1, 0)} {
		copy := set
		copy.VerifiedAt = bad
		publishFixtureSet(t, m, copy)
		if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance == "verified" {
			t.Fatal("invalid or future timestamp accepted")
		}
	}
	copy := set
	copy.ValidUntil = set.VerifiedAt.Add(-time.Second)
	publishFixtureSet(t, m, copy)
	if Inspect(runtimeReadFS{m, t}, "linux", "amd64").Packages[0].Provenance == "verified" {
		t.Fatal("invalid legacy timestamp relation accepted")
	}
}

func TestNinetyDayOfflineAuthorizationAdmitsWorkWithoutNetwork(t *testing.T) {
	m, home, _ := authenticatedPacmanFixture(t)
	payload := map[string][]byte{}
	for _, role := range requiredRoles {
		payload["usr/bin/"+role.executable] = []byte("authenticated fixture " + role.executable)
	}
	installSignedFixturePackage(t, m, home, "docker", "1:29.7.2-1", payload)
	v := verifier{fsys: m, run: fixtureRunner(t, m, home), store: fixtureStore(m)}
	set, failed, err := v.refreshPackages(context.Background(), inspect(m, "linux", "amd64"), receiptSet{})
	if err != nil || failed {
		t.Fatalf("full fixture authentication: %v", err)
	}
	set.VerifiedAt = time.Now().Add(-90 * 24 * time.Hour)
	set.ValidUntil = set.VerifiedAt.Add(ReceiptValidity)
	publishFixtureSet(t, m, set)
	for name := range m {
		if strings.HasPrefix(name, "var/cache/") {
			delete(m, name)
		}
	}
	e := goodEngine("29.7.2+dfsg1")
	evaluate := func() Report {
		r := Inspect(runtimeReadFS{m, t}, "linux", "amd64")
		r.Engine = &e
		return Evaluate(r, e.DataRoot)
	}
	r := evaluate()
	if !r.CanMutate || r.RestartPending || r.Engine.Version != "29.7.2+dfsg1" {
		t.Fatalf("offline age/build metadata denied work: %+v", r)
	}
	old := m["usr/bin/dockerd"].Data
	m["usr/bin/dockerd"].Data = []byte("tampered")
	if r = evaluate(); r.CanMutate || !r.CanStop {
		t.Fatal("payload tamper bypassed admission or stranded control")
	}
	m["usr/bin/dockerd"].Data = old
	m["usr/share/pacman/keyrings/fixture-revoked"].Data = []byte(strings.Repeat("A", 40) + "\n")
	if r = evaluate(); r.CanMutate || !r.CanStop {
		t.Fatal("changed public authority retained new-work authorization")
	}
}

func TestPrivateFetchStagingBoundsAndAliases(t *testing.T) {
	dir := t.TempDir()
	stage := "var/lib/omabit/hostcompat/.fetch-limits"
	if err := os.MkdirAll(filepath.Join(dir, stage), 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	r := fixtureRoot{root}
	for i := 0; i < 5; i++ {
		f, err := root.OpenFile(fmt.Sprintf("%s/large-%d", stage, i), os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Truncate(maxArtifact); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
	}
	if stageWithinBounds(r, stage) == nil {
		t.Fatal("aggregate private download storage limit ignored")
	}
	for i := 0; i < 5; i++ {
		root.Remove(fmt.Sprintf("%s/large-%d", stage, i))
	}
	if err := root.Symlink("../../../../outside", stage+"/alias"); err != nil {
		t.Fatal(err)
	}
	if stageWithinBounds(r, stage) == nil {
		t.Fatal("download staging alias accepted")
	}
}

func TestFetchCancellationDuringNativeCommand(t *testing.T) {
	m, run, _, _ := dockerAPTFixture(t, false)
	v := verifier{fsys: m, run: run, store: fixtureStore(m)}
	f := fixtureFetcher(t, m, &v, func(ctx context.Context, _ fetchCommand) ([]byte, error) { <-ctx.Done(); return nil, ctx.Err() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := f.fetch(ctx, "deb", inspect(m, "linux", "amd64").owners["engine"]); err != errFetch {
		t.Fatalf("canceled native command was not sanitized: %v", err)
	}
	if f.aptReady {
		t.Fatal("canceled native index preparation marked ready")
	}
}

func TestNativePacmanConfigurationIncludeClosure(t *testing.T) {
	m := fixtureFS(map[string]string{"etc/pacman.conf": "[options]\nSigLevel = Required DatabaseOptional\n[core]\nInclude = /etc/pacman.d/mirrorlist\n[extra]\nInclude = /etc/pacman.d/mirrorlist\n", "etc/pacman.d/mirrorlist": "Server = https://example.invalid/$repo/os/$arch\n"})
	if evidence, err := pacmanConfigEvidence(m); err != nil || len(evidence) != 2 {
		t.Fatal("normal shared mirrorlist include rejected", err)
	}
	m["etc/pacman.d/mirrorlist"].Data = []byte("Include = /etc/pacman.conf\n")
	if _, err := pacmanConfigEvidence(m); err == nil {
		t.Fatal("recursive native include accepted")
	}
}

type vanishingFetchRoot struct {
	fixtureRoot
	name string
}

func (r vanishingFetchRoot) Lstat(name string) (os.FileInfo, error) {
	if name == r.name {
		r.Root.Remove(name)
	}
	return r.fixtureRoot.Lstat(name)
}

func TestNativePartialFileRenameDoesNotCancelDownload(t *testing.T) {
	dir := t.TempDir()
	stage := "var/lib/omabit/hostcompat/.fetch-race"
	if err := os.MkdirAll(filepath.Join(dir, stage), 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	name := stage + "/partial-index"
	if err := root.WriteFile(name, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := stageWithinBounds(vanishingFetchRoot{fixtureRoot{root}, name}, stage); err != nil {
		t.Fatal("normal native partial-file rename canceled acquisition")
	}
}
