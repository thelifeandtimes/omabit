package runtimecandidate

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const (
	maxPillBytes  = protocol.MaxBootPillBytes
	pillUserAgent = "curl/omabit-runtimecandidate (+https://github.com/thelifeandtimes/omabit)"
)

// Artifact describes the verified, inert pill file published by DownloadPill.
type Artifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
	URL    string `json:"url"`
}

// DownloadPill streams the fixed candidate's pill over HTTPS, checks its compiled-in
// size and SHA-256, and atomically publishes a synced, read-only (0400) file. It
// refuses root and never executes the downloaded content. The pin records bytes
// observed at the HTTPS source; it is not independent publisher authentication.
//
// Destination must be absolute, clean, and absent, including dangling symlinks.
// Its existing parent must be owned by the effective UID, have mode 0700, and
// have no symlink components. The caller-created private qualification directory
// and other processes with the caller's UID are trusted throughout this operation;
// these checks do not provide privilege isolation against that UID.
func DownloadPill(ctx context.Context, destination string) (Artifact, error) {
	if os.Geteuid() == 0 {
		return Artifact{}, fmt.Errorf("pill download must not run as root")
	}
	client := newPillDownloadClient()
	defer client.CloseIdleConnections()
	return downloadPill(ctx, destination, client, PillSHA256, PillBytes, maxPillBytes)
}

func newPillDownloadClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Minute,
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

// Only tests can substitute a client, content expectation, or smaller byte cap.
// Even this private entry point always requests PillURL. Root refusal belongs to
// the public entry point so filesystem tests also work in root-run test suites.
func downloadPill(ctx context.Context, destination string, client *http.Client, expectedHash string, expectedBytes, byteLimit int64) (artifact Artifact, err error) {
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	if expectedBytes <= 0 || byteLimit <= 0 || byteLimit > maxPillBytes || expectedBytes > byteLimit {
		return Artifact{}, fmt.Errorf("pill expected size must be positive and within the byte limit (at most %d)", maxPillBytes)
	}
	if err := checkPillDestination(destination, os.Geteuid()); err != nil {
		return Artifact{}, err
	}
	stage, err := os.CreateTemp(filepath.Dir(destination), "."+PillName+"-*")
	if err != nil {
		return Artifact{}, fmt.Errorf("create pill staging file: %w", err)
	}
	stageInfo, err := stage.Stat()
	if err != nil {
		return Artifact{}, errors.Join(fmt.Errorf("inspect pill staging file: %w", err), stage.Close(), os.Remove(stage.Name()))
	}
	closed := false
	defer func() {
		if !closed {
			if closeErr := stage.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close pill staging file: %w", closeErr))
			}
		}
		// Never remove the destination, or a replacement of our staging file.
		if cleanupErr := removePillStage(stage.Name(), stageInfo); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("clean up pill staging file: %w", cleanupErr))
		}
		if err != nil {
			artifact = Artifact{}
		}
	}()
	if !stageInfo.Mode().IsRegular() {
		return Artifact{}, fmt.Errorf("pill staging file must be regular")
	}
	// CreateTemp is exclusive and starts with at most 0600; set the exact mode
	// even when the caller has a restrictive umask.
	if err := stage.Chmod(0600); err != nil {
		return Artifact{}, fmt.Errorf("set pill staging mode: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, PillURL, nil)
	if err != nil {
		return Artifact{}, fmt.Errorf("create pill request: %w", err)
	}
	request.Header.Set("User-Agent", pillUserAgent)
	request.Header.Set("Accept-Encoding", "identity")
	response, err := client.Do(request)
	if err != nil {
		return Artifact{}, fmt.Errorf("download pill: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Artifact{}, fmt.Errorf("download pill: expected HTTP 200, got %d", response.StatusCode)
	}
	if response.Uncompressed {
		return Artifact{}, fmt.Errorf("download pill: unexpected content decompression")
	}
	for _, encoding := range response.Header.Values("Content-Encoding") {
		if encoding = strings.TrimSpace(encoding); encoding != "" && !strings.EqualFold(encoding, "identity") {
			return Artifact{}, fmt.Errorf("download pill: unexpected content encoding %q", encoding)
		}
	}
	if response.ContentLength > byteLimit {
		return Artifact{}, fmt.Errorf("download pill: content length exceeds byte limit %d", byteLimit)
	}
	if response.ContentLength >= 0 && response.ContentLength != expectedBytes {
		return Artifact{}, fmt.Errorf("download pill: content length %d does not match expected size %d", response.ContentLength, expectedBytes)
	}
	if err := writeVerifiedPill(ctx, stage, response.Body, expectedHash, expectedBytes); err != nil {
		return Artifact{}, err
	}
	if err := stage.Chmod(0400); err != nil {
		return Artifact{}, fmt.Errorf("make pill read-only: %w", err)
	}
	// Sync after chmod to persist both the verified content and its final mode.
	if err := stage.Sync(); err != nil {
		return Artifact{}, fmt.Errorf("sync pill: %w", err)
	}
	closeErr := stage.Close()
	closed = true
	if closeErr != nil {
		return Artifact{}, fmt.Errorf("close verified pill: %w", closeErr)
	}
	current, err := os.Lstat(stage.Name())
	if err != nil {
		return Artifact{}, fmt.Errorf("inspect verified pill: %w", err)
	}
	if !os.SameFile(stageInfo, current) || !current.Mode().IsRegular() || current.Mode().Perm() != 0400 || current.Size() != expectedBytes {
		return Artifact{}, fmt.Errorf("pill staging file changed; refusing to publish")
	}
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	// Link atomically fails if any destination entry exists, including a symlink
	// or a file created since the initial check. Rename would overwrite it.
	if err := os.Link(stage.Name(), destination); err != nil {
		return Artifact{}, fmt.Errorf("publish pill without overwriting: %w", err)
	}
	// The deferred unlink removes only the staging name. If it fails, report the
	// error but leave the already-published, verified destination in place.
	return Artifact{Path: destination, SHA256: expectedHash, Bytes: expectedBytes, URL: PillURL}, nil
}

func checkPillDestination(destination string, euid int) error {
	if !filepath.IsAbs(destination) || destination != filepath.Clean(destination) {
		return fmt.Errorf("pill destination must be an absolute, clean path")
	}
	parent := filepath.Dir(destination)
	info, err := os.Lstat(parent)
	if err != nil {
		return fmt.Errorf("inspect pill parent: %w", err)
	}
	if info.Mode() != os.ModeDir|0700 {
		return fmt.Errorf("pill parent must be a private 0700 directory, not a symlink")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(euid) {
		return fmt.Errorf("pill parent must be owned by the effective UID")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return fmt.Errorf("resolve pill parent: %w", err)
	}
	if resolved != parent {
		return fmt.Errorf("pill parent must not contain symlink components")
	}
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("pill destination already exists: %w", os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect pill destination: %w", err)
	}
	return nil
}

func removePillStage(path string, owned os.FileInfo) error {
	current, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !os.SameFile(owned, current) {
		return fmt.Errorf("staging path changed; refusing to remove it")
	}
	return os.Remove(path)
}

// At most expectedBytes are written and hashed. One additional byte is probed
// (never written) to reject excess input, including unknown-length/chunked bodies.
// The caller has already checked expectedBytes against the independent byte cap.
func writeVerifiedPill(ctx context.Context, destination io.Writer, body io.Reader, expectedHash string, expectedBytes int64) error {
	reader := pillContextReader{ctx: ctx, reader: body}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(destination, hash), io.LimitReader(reader, expectedBytes))
	if err != nil {
		return fmt.Errorf("stream pill: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if written != expectedBytes {
		return fmt.Errorf("pill size mismatch: got %d bytes, want %d: %w", written, expectedBytes, io.ErrUnexpectedEOF)
	}
	var extra [1]byte
	n, err := io.ReadFull(reader, extra[:])
	if n != 0 {
		return fmt.Errorf("pill exceeds expected size of %d bytes", expectedBytes)
	}
	if err != io.EOF {
		return fmt.Errorf("read end of pill: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if actual := fmt.Sprintf("%x", hash.Sum(nil)); actual != expectedHash {
		return fmt.Errorf("pill SHA-256 mismatch: got %s, want %s", actual, expectedHash)
	}
	return nil
}

type pillContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r pillContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
