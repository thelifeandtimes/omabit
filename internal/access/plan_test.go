package access

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareRestrictedReviewBundle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "review")
	key := []byte(publicKey(t))
	if err := Prepare(dir, key, "operator", 22222); err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile(filepath.Join(dir, "sshd_config"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"ListenAddress 127.0.0.1", "AllowUsers urbit-runner", "UsePAM yes", "DisableForwarding yes", "PermitTTY no", "PermitUserEnvironment no", "PermitUserRC no", "AuthenticationMethods publickey"} {
		if !strings.Contains(string(config), line+"\n") {
			t.Fatalf("missing restriction %s", line)
		}
	}
	if err := Prepare(dir, key, "observer", 22223); err == nil {
		t.Fatal("overwrote existing review")
	}
	line, err := os.ReadFile(filepath.Join(dir, "authorized_keys"))
	if err != nil {
		t.Fatal(err)
	}
	if grant, err := ParseLine(strings.TrimSpace(string(line))); err != nil || grant.Role != "operator" {
		t.Fatalf("grant changed: %+v %v", grant, err)
	}
}
