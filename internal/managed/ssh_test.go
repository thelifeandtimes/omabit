package managed

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func grantLine(t *testing.T, role string) (string, string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 51)
	binary.BigEndian.PutUint32(key, 11)
	copy(key[4:], "ssh-ed25519")
	binary.BigEndian.PutUint32(key[15:], 32)
	copy(key[19:], pub)
	line, err := access.AuthorizedKey([]byte("ssh-ed25519 "+base64.StdEncoding.EncodeToString(key)), role)
	if err != nil {
		t.Fatal(err)
	}
	grant, err := access.ParseLine(line)
	if err != nil {
		t.Fatal(err)
	}
	return line, grant.Fingerprint
}

func TestSSHActorsRolesAndRevocation(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	a, idA := grantLine(t, "operator")
	b, idB := grantLine(t, "operator")
	readOnly, idRead := grantLine(t, "observer")
	r.keys = filepath.Join(privateDir(t), "keys")
	if err := os.WriteFile(r.keys, []byte(a+"\n"+b+"\n"+readOnly+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request := func(id, method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
		req.Header.Set(access.Header, id)
		req.Header.Set(protocol.Header, "1")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	body := `{"id":"alpha","label":"test","request_id":"same-id","config":{"http_port":18080,"ames_port":34343,"loom":31}}`
	first := request(idA, "POST", "/v1/instances", body)
	if first.Code != 201 {
		t.Fatal(first.Body.String())
	}
	var op protocol.Operation
	if err := json.Unmarshal(first.Body.Bytes(), &op); err != nil || op.Actor != "ssh-key:"+idA {
		t.Fatalf("actor: %+v %v", op, err)
	}
	if next := request(idA, "POST", "/v1/instances", body); next.Code != 200 {
		t.Fatal(next.Body.String())
	}
	// The same request ID belongs to a different actor; it must not replay A's op.
	if next := request(idB, "POST", "/v1/instances", body); next.Code != 409 {
		t.Fatal(next.Body.String())
	}
	if next := request(idRead, "GET", "/v1/snapshot", ""); next.Code != 200 {
		t.Fatal(next.Body.String())
	}
	if next := request(idRead, "POST", "/v1/instances", body); next.Code != 403 {
		t.Fatal("observer mutated state")
	}
	if err := os.WriteFile(r.keys, []byte(b+"\n"+readOnly+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if next := request(idA, "GET", "/hello", ""); next.Code != 403 {
		t.Fatal("revoked key remains valid")
	}
	if next := request(idB, "GET", "/hello", ""); next.Code != 200 {
		t.Fatal("revocation affected other key")
	}
}
