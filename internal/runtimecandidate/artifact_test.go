package runtimecandidate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

const testPill = "small inert pill fixture\x00\xff\n"

type pillRoundTripFunc func(*http.Request) (*http.Response, error)

func (f pillRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type pillReadCloseFunc struct {
	read  func([]byte) (int, error)
	close func() error
}

func (b pillReadCloseFunc) Read(p []byte) (int, error) { return b.read(p) }
func (b pillReadCloseFunc) Close() error               { return b.close() }

type pillWriteFunc func([]byte) (int, error)

func (f pillWriteFunc) Write(p []byte) (int, error) { return f(p) }

func pillHash(content string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(content))) }

func pillTestParent(t *testing.T) string {
	t.Helper()
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	return parent
}

func pillTestClient(t *testing.T, mutate func(*http.Request, *http.Response) (*http.Response, error)) *http.Client {
	t.Helper()
	client := newPillDownloadClient()
	client.Transport = pillRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.String() != PillURL || request.URL.Scheme != "https" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL)
			return nil, errors.New("network forbidden")
		}
		if request.Header.Get("Accept-Encoding") != "identity" || request.Header.Get("User-Agent") != pillUserAgent {
			t.Errorf("unexpected headers: %v", request.Header)
		}
		response := &http.Response{
			StatusCode: http.StatusOK, Header: make(http.Header), Request: request,
			ContentLength: int64(len(testPill)), Body: io.NopCloser(strings.NewReader(testPill)),
		}
		if mutate != nil {
			return mutate(request, response)
		}
		return response, nil
	})
	return client
}

func downloadTestPill(ctx context.Context, destination string, client *http.Client) (Artifact, error) {
	return downloadPill(ctx, destination, client, pillHash(testPill), int64(len(testPill)), 64)
}

func assertPillEntries(t *testing.T, parent string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entries in %s: got %v, want %v", parent, got, want)
	}
}

func assertPillFailure(t *testing.T, artifact Artifact, err error, contains string) {
	t.Helper()
	if err == nil || artifact != (Artifact{}) {
		t.Fatalf("expected error and empty artifact, got %+v, %v", artifact, err)
	}
	if !strings.Contains(err.Error(), contains) {
		t.Fatalf("expected error containing %q, got %v", contains, err)
	}
}

func TestPillSourcePinsAndHTTPPolicy(t *testing.T) {
	if PillURL != "https://bootstrap.urbit.org/urbit-v4.6.pill?generation=1790171686269740" ||
		PillSHA256 != "f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b" ||
		PillBytes != 217421172 || PillName != "urbit-v4.6.pill" || maxPillBytes != 256<<20 {
		t.Fatal("candidate pill pins or cap changed")
	}
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
		t.Setenv(name, "http://proxy.invalid:1234")
	}
	client := newPillDownloadClient()
	defer client.CloseIdleConnections()
	transport := client.Transport.(*http.Transport)
	if client.Timeout != 5*time.Minute || transport.Proxy != nil || !transport.DisableCompression || transport.DialContext == nil || transport.TLSHandshakeTimeout != 10*time.Second {
		t.Fatalf("unsafe download policy: %+v, %+v", client, transport)
	}
	if transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS certificate verification is disabled")
	}
	if client.CheckRedirect == nil || client.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
		t.Fatal("redirects must be refused")
	}
	if !strings.HasPrefix(pillUserAgent, "curl/") || !strings.Contains(pillUserAgent, "omabit") {
		t.Fatal("user agent must identify the project")
	}
	parent := pillTestParent(t)
	// Exercise the production expectations without allocating the real pill or
	// making a network request. Small fixture expectations are private to tests.
	artifact, err := downloadPill(context.Background(), filepath.Join(parent, PillName), pillTestClient(t, nil), PillSHA256, PillBytes, maxPillBytes)
	assertPillFailure(t, artifact, err, "does not match expected size 217421172")
	assertPillEntries(t, parent)
}

func TestDownloadPillPublicPreflight(t *testing.T) {
	parent := pillTestParent(t)
	destination := filepath.Join(parent, PillName)
	if os.Geteuid() == 0 {
		artifact, err := DownloadPill(context.Background(), destination)
		assertPillFailure(t, artifact, err, "must not run as root")
	} else {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		artifact, err := DownloadPill(ctx, destination)
		assertPillFailure(t, artifact, err, "")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled download: %v", err)
		}
		artifact, err = DownloadPill(context.Background(), "relative-pill")
		assertPillFailure(t, artifact, err, "absolute, clean")
	}
	assertPillEntries(t, parent)
}

func TestDownloadPillPublishesVerifiedReadOnlyArtifact(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		t.Run(fmt.Sprintf("chunked-%t", chunked), func(t *testing.T) {
			parent := pillTestParent(t)
			destination := filepath.Join(parent, PillName)
			var staged os.FileInfo
			closed := false
			requests := 0
			client := pillTestClient(t, func(_ *http.Request, response *http.Response) (*http.Response, error) {
				requests++
				if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("destination visible before verification: %v", err)
				}
				entries, err := os.ReadDir(parent)
				if err != nil || len(entries) != 1 {
					t.Fatalf("staging entries: %v, %v", entries, err)
				}
				staged, err = entries[0].Info()
				if err != nil || !staged.Mode().IsRegular() || staged.Mode().Perm() != 0600 || staged.Size() != 0 {
					t.Fatalf("staging file: %v, %v", staged, err)
				}
				body := response.Body
				response.Body = pillReadCloseFunc{read: body.Read, close: func() error { closed = true; return body.Close() }}
				response.Header.Set("Content-Encoding", "identity")
				if chunked {
					response.ContentLength = -1
					response.TransferEncoding = []string{"chunked"}
				}
				return response, nil
			})
			artifact, err := downloadTestPill(context.Background(), destination, client)
			if err != nil {
				t.Fatal(err)
			}
			want := Artifact{Path: destination, SHA256: pillHash(testPill), Bytes: int64(len(testPill)), URL: PillURL}
			if artifact != want || !closed || requests != 1 {
				t.Fatalf("download: %+v, closed=%t, requests=%d", artifact, closed, requests)
			}
			data, err := os.ReadFile(destination)
			if err != nil || string(data) != testPill {
				t.Fatalf("published bytes: %q, %v", data, err)
			}
			info, err := os.Lstat(destination)
			if err != nil || !os.SameFile(staged, info) || !info.Mode().IsRegular() || info.Mode().Perm() != 0400 {
				t.Fatalf("published file: %v, %v", info, err)
			}
			encoded, err := json.Marshal(artifact)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded, map[string]any{"path": destination, "sha256": want.SHA256, "bytes": float64(want.Bytes), "url": PillURL}) {
				t.Fatalf("JSON fields: %s", encoded)
			}
			assertPillEntries(t, parent, PillName)
			artifact, err = downloadTestPill(context.Background(), destination, client)
			assertPillFailure(t, artifact, err, "already exists")
			if !errors.Is(err, os.ErrExist) || requests != 1 {
				t.Fatalf("repeat download: requests=%d, err=%v", requests, err)
			}
		})
	}
}

func TestDownloadPillFailuresCleanUp(t *testing.T) {
	for _, failure := range []struct {
		name string
		want string
		is   error
	}{
		{"hash", "SHA-256 mismatch", nil},
		{"404", "expected HTTP 200, got 404", nil},
		{"500", "expected HTTP 200, got 500", nil},
		{"partial", "expected HTTP 200, got 206", nil},
		{"redirect", "expected HTTP 200, got 302", nil},
		{"short-header", "content length", nil},
		{"long-header", "content length", nil},
		{"over-cap-header", "exceeds byte limit", nil},
		{"short-body", "size mismatch", io.ErrUnexpectedEOF},
		{"chunked-short", "size mismatch", io.ErrUnexpectedEOF},
		{"long-body", "exceeds expected size", nil},
		{"chunked-oversize", "exceeds expected size", nil},
		{"compressed", "content encoding", nil},
		{"multiple-encodings", "content encoding", nil},
		{"decompressed", "content decompression", nil},
		{"read-error", "stream pill", io.ErrUnexpectedEOF},
		{"trailer-error", "read end of pill", io.ErrUnexpectedEOF},
		{"transport-error", "download pill", io.ErrClosedPipe},
		{"cancel", "", context.Canceled},
		{"cancel-after-data", "", context.Canceled},
		{"cancel-at-eof", "", context.Canceled},
		{"request-timeout", "", context.DeadlineExceeded},
		{"body-timeout", "", context.DeadlineExceeded},
	} {
		t.Run(failure.name, func(t *testing.T) {
			parent := pillTestParent(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			requests, reads := 0, 0
			closed := false
			client := pillTestClient(t, func(request *http.Request, response *http.Response) (*http.Response, error) {
				requests++
				var body io.Reader = strings.NewReader(testPill)
				switch failure.name {
				case "hash":
					body = strings.NewReader(strings.Repeat("x", len(testPill)))
				case "404":
					response.StatusCode = http.StatusNotFound
				case "500":
					response.StatusCode = http.StatusInternalServerError
				case "partial":
					response.StatusCode = http.StatusPartialContent
				case "redirect":
					response.StatusCode = http.StatusFound
					response.Header.Set("Location", "https://other.invalid/pill")
				case "short-header":
					response.ContentLength--
				case "long-header":
					response.ContentLength++
				case "over-cap-header":
					response.ContentLength = 65
				case "short-body", "chunked-short":
					body = strings.NewReader(testPill[:len(testPill)-1])
				case "long-body", "chunked-oversize":
					body = strings.NewReader(testPill + strings.Repeat("x", 128))
				case "compressed":
					response.Header.Set("Content-Encoding", "gzip")
				case "multiple-encodings":
					response.Header.Add("Content-Encoding", "identity")
					response.Header.Add("Content-Encoding", "gzip")
				case "decompressed":
					response.Uncompressed = true
				case "transport-error":
					return nil, io.ErrClosedPipe
				case "cancel":
					cancel()
				case "request-timeout":
					<-request.Context().Done()
					return nil, request.Context().Err()
				}
				if strings.HasPrefix(failure.name, "chunked-") {
					response.ContentLength = -1
					response.TransferEncoding = []string{"chunked"}
				}
				response.Body = pillReadCloseFunc{
					read: func(p []byte) (int, error) {
						reads++
						if failure.name == "read-error" || failure.name == "trailer-error" && reads > 1 {
							return 0, io.ErrUnexpectedEOF
						}
						if failure.name == "body-timeout" {
							<-request.Context().Done()
							return 0, request.Context().Err()
						}
						n, err := body.Read(p)
						if failure.name == "cancel-after-data" || failure.name == "cancel-at-eof" && err == io.EOF {
							cancel()
						}
						return n, err
					},
					close: func() error { closed = true; return nil },
				}
				return response, nil
			})
			if strings.HasSuffix(failure.name, "timeout") {
				client.Timeout = 10 * time.Millisecond
			}
			artifact, err := downloadTestPill(ctx, filepath.Join(parent, PillName), client)
			assertPillFailure(t, artifact, err, failure.want)
			if failure.is != nil && !errors.Is(err, failure.is) {
				t.Fatalf("expected %v, got %v", failure.is, err)
			}
			if requests != 1 {
				t.Fatalf("made %d requests, want 1 (no redirects)", requests)
			}
			if failure.name != "request-timeout" && failure.name != "transport-error" && !closed {
				t.Fatal("response body was not closed")
			}
			if (strings.Contains(failure.want, "HTTP 200") || strings.Contains(failure.want, "content")) && reads != 0 {
				t.Fatal("read a body rejected by status or headers")
			}
			assertPillEntries(t, parent)
		})
	}
}

func TestDownloadPillTinyCapBoundaries(t *testing.T) {
	for _, size := range []int{7, 8, 9, 64} {
		t.Run(fmt.Sprintf("bytes-%d", size), func(t *testing.T) {
			parent := pillTestParent(t)
			readBytes := 0
			body := strings.NewReader(strings.Repeat("x", size))
			client := pillTestClient(t, func(_ *http.Request, response *http.Response) (*http.Response, error) {
				response.ContentLength = -1
				response.Body = pillReadCloseFunc{read: func(p []byte) (int, error) {
					n, err := body.Read(p)
					readBytes += n
					return n, err
				}, close: func() error { return nil }}
				return response, nil
			})
			artifact, err := downloadPill(context.Background(), filepath.Join(parent, PillName), client, pillHash("xxxxxxxx"), 8, 8)
			if size == 8 {
				if err != nil || artifact.Bytes != 8 {
					t.Fatalf("exact cap: %+v, %v", artifact, err)
				}
				assertPillEntries(t, parent, PillName)
			} else {
				assertPillFailure(t, artifact, err, "")
				assertPillEntries(t, parent)
			}
			if readBytes != min(size, 9) {
				t.Fatalf("read %d bytes, want at most cap plus one-byte probe", readBytes)
			}
		})
	}
	for _, limits := range [][2]int64{{9, 8}, {0, 8}, {-1, 8}, {8, 0}, {8, -1}, {8, maxPillBytes + 1}} {
		parent := pillTestParent(t)
		client := pillTestClient(t, func(*http.Request, *http.Response) (*http.Response, error) {
			t.Error("invalid expectation made a request")
			return nil, errors.New("network forbidden")
		})
		artifact, err := downloadPill(context.Background(), filepath.Join(parent, PillName), client, pillHash(testPill), limits[0], limits[1])
		assertPillFailure(t, artifact, err, "byte limit")
		assertPillEntries(t, parent)
	}
}

func TestWriteVerifiedPillBoundsWritesAndHandlesDiskErrors(t *testing.T) {
	for _, kind := range []string{"disk-full", "short-write", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			var written bytes.Buffer
			writer := pillWriteFunc(func(p []byte) (int, error) {
				if kind == "disk-full" {
					return 0, syscall.ENOSPC
				}
				if kind == "short-write" {
					return len(p) - 1, nil
				}
				return written.Write(p)
			})
			err := writeVerifiedPill(context.Background(), writer, strings.NewReader(testPill+"extra"), pillHash(testPill), int64(len(testPill)))
			if err == nil {
				t.Fatal("accepted failed write or oversize input")
			}
			switch kind {
			case "disk-full":
				if !errors.Is(err, syscall.ENOSPC) {
					t.Fatalf("lost disk error: %v", err)
				}
			case "short-write":
				if !errors.Is(err, io.ErrShortWrite) {
					t.Fatalf("lost short write: %v", err)
				}
			case "oversize":
				if written.String() != testPill {
					t.Fatalf("wrote beyond expected byte count: %q", written.String())
				}
			}
		})
	}
}

func TestDownloadPillRefusesUnsafeDestinations(t *testing.T) {
	for _, kind := range []string{
		"relative", "unclean", "trailing-slash", "missing-parent", "parent-file",
		"file", "directory", "symlink", "dangling-symlink", "fifo",
		"parent-symlink", "ancestor-symlink", "group-readable", "world-readable", "group-writable", "world-writable", "special-mode",
	} {
		t.Run(kind, func(t *testing.T) {
			parent := pillTestParent(t)
			destination := filepath.Join(parent, PillName)
			var err error
			switch kind {
			case "relative":
				destination = "relative-pill"
			case "unclean":
				destination = parent + "/child/../pill"
			case "trailing-slash":
				destination += "/"
			case "missing-parent":
				destination = filepath.Join(parent, "missing", PillName)
			case "parent-file":
				err = os.WriteFile(destination, []byte("untouched"), 0600)
				destination = filepath.Join(destination, "pill")
			case "file":
				err = os.WriteFile(destination, []byte("untouched"), 0600)
			case "directory":
				err = os.Mkdir(destination, 0700)
			case "fifo":
				err = syscall.Mkfifo(destination, 0600)
			case "symlink":
				err = os.Symlink(parent, destination)
			case "dangling-symlink":
				err = os.Symlink(filepath.Join(parent, "absent"), destination)
			case "parent-symlink", "ancestor-symlink":
				if err := os.Mkdir(filepath.Join(parent, "child"), 0700); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(parent, destination)
				if kind == "parent-symlink" {
					destination = filepath.Join(destination, "pill")
				} else {
					destination = filepath.Join(destination, "child", "pill")
				}
			default:
				mode := map[string]os.FileMode{"group-readable": 0740, "world-readable": 0704, "group-writable": 0720, "world-writable": 0702, "special-mode": 0700 | os.ModeSticky}[kind]
				err = os.Chmod(parent, mode)
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadDir(parent)
			if err != nil {
				t.Fatal(err)
			}
			original, originalErr := os.Lstat(destination)
			client := pillTestClient(t, func(*http.Request, *http.Response) (*http.Response, error) {
				t.Error("unsafe destination made a request")
				return nil, errors.New("network forbidden")
			})
			artifact, err := downloadTestPill(context.Background(), destination, client)
			assertPillFailure(t, artifact, err, "")
			after, err := os.ReadDir(parent)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("parent entries changed: %v, %v, %v", before, after, err)
			}
			if originalErr == nil {
				current, err := os.Lstat(destination)
				if err != nil || !os.SameFile(original, current) || original.Mode() != current.Mode() {
					t.Fatalf("existing destination changed: %v", err)
				}
			}
			if kind == "file" || kind == "parent-file" {
				data, err := os.ReadFile(filepath.Join(parent, PillName))
				if err != nil || string(data) != "untouched" {
					t.Fatalf("existing file modified: %q, %v", data, err)
				}
			}
		})
	}
	parent := pillTestParent(t)
	// Inject only the ownership comparison, without chown or changing process UID.
	if err := checkPillDestination(filepath.Join(parent, PillName), os.Geteuid()+1); err == nil || !strings.Contains(err.Error(), "effective UID") {
		t.Fatalf("foreign-owned parent accepted: %v", err)
	}
	assertPillEntries(t, parent)
}

func TestDownloadPillAtomicNoOverwrite(t *testing.T) {
	for _, kind := range []string{"file", "directory", "symlink", "dangling-symlink", "fifo", "competing-download"} {
		t.Run(kind, func(t *testing.T) {
			parent := pillTestParent(t)
			destination := filepath.Join(parent, PillName)
			var original os.FileInfo
			client := pillTestClient(t, func(_ *http.Request, response *http.Response) (*http.Response, error) {
				var err error
				switch kind {
				case "file":
					err = os.WriteFile(destination, []byte("untouched"), 0600)
				case "directory":
					err = os.Mkdir(destination, 0700)
				case "symlink":
					err = os.Symlink(parent, destination)
				case "dangling-symlink":
					err = os.Symlink(filepath.Join(parent, "absent"), destination)
				case "fifo":
					err = syscall.Mkfifo(destination, 0600)
				case "competing-download":
					_, err = downloadTestPill(context.Background(), destination, pillTestClient(t, nil))
				}
				if err != nil {
					t.Fatal(err)
				}
				original, err = os.Lstat(destination)
				if err != nil {
					t.Fatal(err)
				}
				return response, nil
			})
			artifact, err := downloadTestPill(context.Background(), destination, client)
			assertPillFailure(t, artifact, err, "publish pill without overwriting")
			if !errors.Is(err, os.ErrExist) {
				t.Fatalf("expected atomic link conflict: %v", err)
			}
			current, err := os.Lstat(destination)
			if err != nil || !os.SameFile(original, current) || original.Mode() != current.Mode() {
				t.Fatalf("competing destination changed: %v", err)
			}
			if kind == "file" || kind == "competing-download" {
				want := "untouched"
				if kind == "competing-download" {
					want = testPill
				}
				data, err := os.ReadFile(destination)
				if err != nil || string(data) != want {
					t.Fatalf("competing file content changed: %q, %v", data, err)
				}
			}
			assertPillEntries(t, parent, PillName)
		})
	}
}

func TestDownloadPillPreservesReplacedStagingPath(t *testing.T) {
	for _, kind := range []string{"file", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			parent := pillTestParent(t)
			var stagingPath string
			var replacement os.FileInfo
			client := pillTestClient(t, func(_ *http.Request, response *http.Response) (*http.Response, error) {
				entries, err := os.ReadDir(parent)
				if err != nil || len(entries) != 1 {
					t.Fatalf("staging entries: %v, %v", entries, err)
				}
				stagingPath = filepath.Join(parent, entries[0].Name())
				if err := os.Rename(stagingPath, filepath.Join(parent, "moved")); err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "file":
					err = os.WriteFile(stagingPath, []byte("untouched"), 0600)
				case "directory":
					err = os.Mkdir(stagingPath, 0700)
				case "symlink":
					err = os.Symlink("moved", stagingPath)
				}
				if err != nil {
					t.Fatal(err)
				}
				replacement, err = os.Lstat(stagingPath)
				if err != nil {
					t.Fatal(err)
				}
				return response, nil
			})
			artifact, err := downloadTestPill(context.Background(), filepath.Join(parent, PillName), client)
			assertPillFailure(t, artifact, err, "staging file changed")
			current, err := os.Lstat(stagingPath)
			if err != nil || !os.SameFile(replacement, current) {
				t.Fatalf("removed replacement staging path: %v", err)
			}
			if kind == "file" {
				data, err := os.ReadFile(stagingPath)
				if err != nil || string(data) != "untouched" {
					t.Fatalf("replacement content changed: %q, %v", data, err)
				}
			}
			assertPillEntries(t, parent, filepath.Base(stagingPath), "moved")
		})
	}
}
