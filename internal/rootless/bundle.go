package rootless

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	sourceURL        = "https://raw.githubusercontent.com/moby/moby/" + SourceCommit + "/"
	launcherURL      = sourceURL + "contrib/dockerd-rootless.sh"
	licenseSHA256    = "7c87873291f289713ac5df48b1f2010eb6963752bbd6b530416ab99fc37914a8"
	noticeSHA256     = "b40ec5b16182103ef1cd69d42a21c98369e35bd483b3973902da4807c0755446"
	maxDownloadBytes = 1 << 20
)

//go:embed assets/urbit-docker.service assets/check-engine.sh assets/daemon.json
var assets embed.FS

// Manifest describes schema version 1 of a preparation bundle. Files lists every
// artifact except manifest.json itself, with hashes of the exact emitted bytes.
// It is an inventory, not a signature, installation approval, engine validation,
// or claim that any container image has been qualified. UnitPath is the intended
// root-owned global user-unit location, not a path written by Prepare.
type Manifest struct {
	SchemaVersion  int            `json:"schema_version"`
	EngineVersion  string         `json:"engine_version"`
	SourceCommit   string         `json:"source_commit"`
	LauncherURL    string         `json:"launcher_url"`
	LauncherSHA256 string         `json:"launcher_sha256"`
	Account        string         `json:"account"`
	Home           string         `json:"home"`
	InstallDir     string         `json:"install_dir"`
	Unit           string         `json:"unit"`
	UnitPath       string         `json:"unit_path"`
	TrustNote      string         `json:"trust_note"`
	Files          []ManifestFile `json:"files"`
}

// ManifestFile records a basename, lowercase SHA-256, octal preparation mode,
// and (only for unmodified upstream files) immutable source URL. Mode describes
// the bundle, not eventual installation permissions; neither script is executable.
type ManifestFile struct {
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	Mode      string `json:"mode"`
	SourceURL string `json:"source_url,omitempty"`
}

// Prepare downloads and verifies the pinned launcher and legal files, then
// prepares inert review artifacts. It never executes scripts, installs software,
// changes Docker contexts, or interacts with services. Root is refused.
//
// Output must be absolute, clean, and absent (including dangling symlinks). Its
// existing parent must be owned by the effective UID, not group/world writable,
// and have no symlink components; a 0755 repository/build directory is allowed.
// A 0700 destination is exclusively reserved with mkdir before staging in a
// same-parent 0700 temporary directory. The completed bundle is renamed over
// only that reservation, never over an existing caller-owned destination.
// As with the resulting user-owned files, the caller's UID and parent directory
// are trusted during preparation; this is not a privilege boundary against the
// caller. A later privileged installer must use independently trusted source.
func Prepare(ctx context.Context, output string) (Manifest, error) {
	client := newDownloadClient()
	defer client.CloseIdleConnections()
	return prepare(ctx, output, client)
}

func newDownloadClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       30 * time.Second,
			DisableCompression:    true,
		},
	}
}

// The client injection is private: callers cannot change pins, URLs, or TLS policy.
func prepare(ctx context.Context, output string, client *http.Client) (Manifest, error) {
	var empty Manifest
	if os.Geteuid() == 0 {
		return empty, fmt.Errorf("rootless preparation must not run as root")
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if !filepath.IsAbs(output) || output != filepath.Clean(output) {
		return empty, fmt.Errorf("output must be an absolute, clean path")
	}
	parent := filepath.Dir(output)
	info, err := os.Lstat(parent)
	if err != nil {
		return empty, fmt.Errorf("inspect output parent: %w", err)
	}
	if !info.IsDir() || info.Mode().Perm()&0022 != 0 {
		return empty, fmt.Errorf("output parent must be a directory, not a symlink, and not group/world writable")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return empty, fmt.Errorf("output parent must be owned by the current user")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil || resolved != parent {
		return empty, fmt.Errorf("output parent must not contain symlinks")
	}
	// mkdir, not a check followed by rename, arbitrates concurrent preparers.
	if err := os.Mkdir(output, 0700); err != nil {
		return empty, fmt.Errorf("reserve output (must not already exist): %w", err)
	}
	reservation, err := os.Lstat(output)
	if err != nil {
		return empty, fmt.Errorf("inspect output reservation: %w", err)
	}
	defer func() {
		if current, err := os.Lstat(output); err == nil && os.SameFile(reservation, current) {
			// Never recursively remove a destination, even one we reserved.
			_ = os.Remove(output)
		}
	}()
	stage, err := os.MkdirTemp(parent, ".urbit-rootless-*")
	if err != nil {
		return empty, fmt.Errorf("create staging directory: %w", err)
	}
	stageInfo, err := os.Lstat(stage)
	if err != nil {
		return empty, fmt.Errorf("inspect staging directory: %w", err)
	}
	defer func() {
		if current, err := os.Lstat(stage); err == nil && os.SameFile(stageInfo, current) {
			_ = os.RemoveAll(stage)
		}
	}()
	// Ensure exact modes even when the preparing user has a restrictive umask.
	if err := os.Chmod(stage, 0700); err != nil {
		return empty, fmt.Errorf("set staging mode: %w", err)
	}

	manifest := Manifest{
		SchemaVersion: 1, EngineVersion: EngineVersion, SourceCommit: SourceCommit,
		LauncherURL: launcherURL, LauncherSHA256: LauncherSHA256,
		Account: AccountName, Home: HomeDir, InstallDir: InstallDir, Unit: UnitName,
		UnitPath:  "/etc/systemd/user/" + UnitName,
		TrustNote: "User-owned preparation output, not independently trusted installer input. Account/home describe legacy template placeholders; install-host binds the verified templates to the explicitly selected existing owner. The OS-authorized installer checks content against compiled-in pins/templates, not this manifest. Preparation performs no installation, runtime validation, or image qualification.",
	}
	type artifact struct {
		name, url, hash string
		data            []byte
	}
	artifacts := []artifact{
		{name: "dockerd-rootless.sh", url: launcherURL, hash: LauncherSHA256},
		{name: "LICENSE", url: sourceURL + "LICENSE", hash: licenseSHA256},
		{name: "NOTICE", url: sourceURL + "NOTICE", hash: noticeSHA256},
		{name: UnitName},
		{name: "check-engine.sh"},
		{name: "daemon.json"},
	}
	// Verify every upstream file before emitting any artifact.
	for i := range artifacts {
		a := &artifacts[i]
		if a.url != "" {
			a.data, err = download(ctx, client, a.url, a.hash)
		} else {
			a.data, err = renderedAsset(a.name)
		}
		if err != nil {
			return empty, fmt.Errorf("prepare %s: %w", a.name, err)
		}
		manifest.Files = append(manifest.Files, ManifestFile{
			Name: a.name, SHA256: fmt.Sprintf("%x", sha256.Sum256(a.data)),
			Mode: "0644", SourceURL: a.url,
		})
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return empty, fmt.Errorf("encode manifest: %w", err)
	}
	artifacts = append(artifacts, artifact{name: "manifest.json", data: append(data, '\n')})
	for _, a := range artifacts {
		if err := ctx.Err(); err != nil {
			return empty, err
		}
		f, err := os.OpenFile(filepath.Join(stage, a.name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return empty, fmt.Errorf("create %s: %w", a.name, err)
		}
		_, writeErr := f.Write(a.data)
		modeErr := f.Chmod(0644)
		closeErr := f.Close()
		for _, err := range []error{writeErr, modeErr, closeErr} {
			if err != nil {
				return empty, fmt.Errorf("write %s: %w", a.name, err)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	current, err := os.Lstat(output)
	if err != nil || !os.SameFile(reservation, current) {
		return empty, fmt.Errorf("output reservation changed; refusing to publish")
	}
	// os.Rename rejects an existing directory, even our empty reservation.
	// Linux rename atomically replaces an empty directory but refuses contents.
	if err := syscall.Rename(stage, output); err != nil {
		return empty, fmt.Errorf("publish bundle: %w", err)
	}
	return manifest, nil
}

func renderedAsset(name string) ([]byte, error) {
	data, err := assets.ReadFile("assets/" + name)
	if err != nil {
		return nil, err
	}
	text := string(data)
	for _, replacement := range [][2]string{
		{"@INSTALL_DIR@", InstallDir}, {"@ENGINE_VERSION@", EngineVersion},
		{"@LAUNCHER_SHA256@", LauncherSHA256}, {"@ACCOUNT@", AccountName},
		{"@HOME@", HomeDir}, {"@SOCKET@", SocketName},
	} {
		text = strings.ReplaceAll(text, replacement[0], replacement[1])
	}
	return []byte(text), nil
}

// VerifiedFiles snapshots the bundle's exact bytes for an authorized installer.
// It trusts compiled-in pins/templates, never hashes or paths in manifest.json.
// The installer must install these returned bytes, not reopen user-owned files.
func VerifiedFiles(dir string) (map[string][]byte, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return nil, fmt.Errorf("bundle path must be absolute and clean")
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil || resolved != dir {
		return nil, fmt.Errorf("bundle path must exist without symlink components")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer root.Close()
	pins := map[string]string{
		"dockerd-rootless.sh": LauncherSHA256, "LICENSE": licenseSHA256, "NOTICE": noticeSHA256,
	}
	for _, name := range []string{UnitName, "check-engine.sh", "daemon.json"} {
		data, err := renderedAsset(name)
		if err != nil {
			return nil, err
		}
		pins[name] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	files := make(map[string][]byte, len(pins))
	for name, hash := range pins {
		info, err := root.Lstat(name)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxDownloadBytes {
			return nil, fmt.Errorf("bundle %s must be a regular file no larger than 1 MiB", name)
		}
		// O_NONBLOCK avoids hanging on a FIFO swapped in after the metadata check.
		f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if err != nil {
			return nil, fmt.Errorf("open bundle %s: %w", name, err)
		}
		info, err = f.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxDownloadBytes {
			f.Close()
			return nil, fmt.Errorf("bundle %s changed or is not a bounded regular file", name)
		}
		data, readErr := io.ReadAll(io.LimitReader(f, maxDownloadBytes+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || len(data) > maxDownloadBytes {
			return nil, fmt.Errorf("read bundle %s failed or exceeded limit", name)
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != hash {
			return nil, fmt.Errorf("bundle %s does not match this binary's reviewed content", name)
		}
		files[name] = data
	}
	return files, nil
}

func download(ctx context.Context, client *http.Client, url, expectedHash string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "identity")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: expected HTTP 200, got %d", url, resp.StatusCode)
	}
	if resp.Header.Get("Content-Encoding") != "" && resp.Header.Get("Content-Encoding") != "identity" {
		return nil, fmt.Errorf("%s: unexpected content encoding", url)
	}
	if resp.ContentLength > maxDownloadBytes {
		return nil, fmt.Errorf("%s: download exceeds 1 MiB", url)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownloadBytes {
		return nil, fmt.Errorf("%s: download exceeds 1 MiB", url)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if actual := fmt.Sprintf("%x", sha256.Sum256(data)); actual != expectedHash {
		return nil, fmt.Errorf("%s: SHA-256 mismatch: got %s, want %s", url, actual, expectedHash)
	}
	return data, nil
}
