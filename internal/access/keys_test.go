package access

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func publicKey(t *testing.T) string {
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
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(key)
}

func TestGrantIdentityRevocationAndPermissions(t *testing.T) {
	public := publicKey(t)
	if _, err := AuthorizedKey([]byte(public+"\n"+public), "operator"); err == nil {
		t.Fatal("multiple public key lines accepted")
	}
	line, err := AuthorizedKey([]byte(public), "observer")
	if err != nil {
		t.Fatal(err)
	}
	grant, err := ParseLine(line)
	if err != nil || grant.Role != "observer" {
		t.Fatalf("grant: %+v %v", grant, err)
	}
	if !grant.Allows("GET", "/v1/snapshot") || grant.Allows("POST", "/v1/instances") {
		t.Fatal("observer permissions")
	}
	path := filepath.Join(t.TempDir(), "keys")
	if err := os.WriteFile(path, []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := Load(path, grant.Fingerprint); err != nil || got != grant {
		t.Fatalf("load: %+v %v", got, err)
	}
	for _, contents := range []string{"", line + "\n" + line, strings.Replace(line, "restrict,", "", 1), strings.Replace(line, "observer", "root", 1), strings.Replace(line, grant.Fingerprint, "SHA256:"+strings.Repeat("a", 43), 1)} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path, grant.Fingerprint); err == nil {
			t.Fatalf("invalid/revoked grant accepted: %q", contents)
		}
	}
	if err := os.WriteFile(path, []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0666); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, grant.Fingerprint); err == nil {
		t.Fatal("writable grants accepted")
	}
}

func TestOwnerForcedCommandPreservesGrantBinding(t *testing.T) {
	public := []byte(publicKey(t))
	for _, role := range []string{"operator", "observer"} {
		line, err := OwnerAuthorizedKey(public, role)
		if err != nil {
			t.Fatal(err)
		}
		grant, err := ParseLine(line)
		if err != nil || grant.Role != role || !strings.HasPrefix(line, `restrict,command="`+ShellPath+` owner-ssh `) {
			t.Fatal(line, grant, err)
		}
		for _, bad := range []string{strings.Replace(line, " owner-ssh ", " serve-development ", 1), strings.Replace(line, ShellPath, "/tmp/urbit-runner", 1), strings.Replace(line, "restrict,", "", 1)} {
			if _, err := ParseLine(bad); err == nil {
				t.Fatal("accepted altered owner command", bad)
			}
		}
	}
}
