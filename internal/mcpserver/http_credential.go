package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

type httpCredential struct {
	Version   int       `json:"version"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
type HTTPTokenFile struct {
	Path      string    `json:"path"`
	ExpiresAt time.Time `json:"expires_at"`
}
type httpAuthority struct {
	path        string
	node        os.FileInfo
	fileSHA256  [32]byte
	tokenSHA256 [32]byte
	expires     time.Time
}

func openHTTPAuthority(path string) (*httpAuthority, error) {
	raw, node, err := readHTTPPrivateFile(path)
	if err != nil {
		return nil, errors.New("invalid private HTTP credential")
	}
	var c httpCredential
	if json.Unmarshal(raw, &c) != nil || c.Version != 1 || len(c.Token) != 64 || !c.ExpiresAt.After(time.Now()) || c.ExpiresAt.After(time.Now().Add(24*time.Hour)) {
		return nil, errors.New("invalid private HTTP credential")
	}
	token, err := hex.DecodeString(c.Token)
	canonical, _ := json.Marshal(c)
	if err != nil || len(token) != 32 || hex.EncodeToString(token) != c.Token || string(raw) != string(canonical)+"\n" {
		return nil, errors.New("invalid private HTTP credential")
	}
	return &httpAuthority{path: path, node: node, fileSHA256: sha256.Sum256(raw), tokenSHA256: sha256.Sum256([]byte(c.Token)), expires: c.ExpiresAt}, nil
}

func (a *httpAuthority) Verify(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	if ctx.Err() != nil || !a.expires.After(time.Now()) || len(token) != 64 {
		return nil, auth.ErrInvalidToken
	}
	raw, node, err := readHTTPPrivateFile(a.path)
	if err != nil || !os.SameFile(a.node, node) || sha256.Sum256(raw) != a.fileSHA256 {
		return nil, auth.ErrInvalidToken
	}
	digest := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(digest[:], a.tokenSHA256[:]) != 1 {
		return nil, auth.ErrInvalidToken
	}
	return &auth.TokenInfo{UserID: "omabit-local-" + hex.EncodeToString(a.tokenSHA256[:]), Expiration: a.expires}, nil
}

// CreateHTTPToken writes one new owner-only local credential, never overwriting
// a prior credential and never returning/printing its secret. Restart HTTP after
// rotation; changed/deleted credentials immediately refuse existing sessions.
func CreateHTTPToken(path string) (HTTPTokenFile, error) {
	var out HTTPTokenFile
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return out, errors.New("absolute private credential path required")
	}
	parent := filepath.Dir(path)
	actual, err := filepath.EvalSymlinks(parent)
	if err != nil || actual != parent {
		return out, errors.New("physical existing credential parent required")
	}
	info, err := os.Stat(parent)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0022 != 0 {
		return out, errors.New("private credential parent required")
	}
	token := make([]byte, 32)
	if _, err = rand.Read(token); err != nil {
		return out, errors.New("credential generation failed")
	}
	c := httpCredential{Version: 1, Token: hex.EncodeToString(token), ExpiresAt: time.Now().UTC().Add(8 * time.Hour)}
	raw, err := json.Marshal(c)
	if err != nil {
		return out, err
	}
	raw = append(raw, '\n')
	if err = writeHTTPPrivateFile(path, raw); err != nil {
		return out, errors.New("new private credential could not be saved; existing files are never replaced")
	}
	return HTTPTokenFile{Path: path, ExpiresAt: c.ExpiresAt}, nil
}
