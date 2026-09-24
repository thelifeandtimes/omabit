package credentials

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func exercise(t *testing.T, s Store) {
	t.Helper()
	ctx := context.Background()
	key := Scope("test-"+t.Name(), "https://example.invalid", time.Now().String())
	value := []byte(`{"fixture":"ephemeral-store-test"}`)
	t.Cleanup(func() {
		if err := s.Delete(context.Background(), key); err != nil {
			t.Error(err)
		}
	})
	if err := s.Save(ctx, key, value); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(ctx, key)
	if err != nil || !bytes.Equal(value, got) {
		t.Fatalf("credential round trip failed: %v", err)
	}
	clear(got)
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if data, err := s.Load(ctx, key); !errors.Is(err, ErrMissing) {
		clear(data)
		t.Fatalf("deleted credential still available: %v", err)
	}
}

func TestFileCredentialsPermissionsAndIsolation(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(parent, "credentials")
	s, err := NewFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	exercise(t, s)
	key := Scope("profile", "http://127.0.0.1:1", "session")
	if key == Scope("profile", "http://127.0.0.1:2", "session") {
		t.Fatal("ports share credential scope")
	}
	if err := s.Save(context.Background(), key, []byte("private-token")); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(context.Background(), key, []byte("overwrite")); err == nil {
		t.Fatal("overwrote existing session")
	}
	if err := os.Chmod(filepath.Join(dir, key), 0644); err != nil {
		t.Fatal(err)
	}
	if data, err := s.Load(context.Background(), key); err == nil {
		clear(data)
		t.Fatal("public credential file accepted")
	}
	if _, err := s.Load(context.Background(), "../../escape"); err == nil {
		t.Fatal("credential path escape accepted")
	}
}

func TestSecretServiceUsesStdinAndRedactsFailures(t *testing.T) {
	secret := []byte("SECRET-ONLY-ON-STDIN")
	key := Scope("profile", "https://example.invalid", "session")
	s := &SecretService{run: func(ctx context.Context, args []string, input []byte) ([]byte, bool, error) {
		if strings.Contains(strings.Join(args, " "), string(secret)) || !bytes.Equal(input, secret) {
			t.Fatal("secret was not confined to stdin")
		}
		return nil, false, errors.New("subprocess diagnostic contains SECRET-ONLY-ON-STDIN")
	}}
	if err := s.Save(context.Background(), key, secret); !errors.Is(err, ErrStore) || strings.Contains(err.Error(), string(secret)) {
		t.Fatalf("diagnostic leaked: %v", err)
	}
}

func TestLiveSecretService(t *testing.T) {
	if os.Getenv("OMABIT_TEST_SECRET_SERVICE") != "1" {
		t.Skip("opt in to ephemeral desktop credential round trip")
	}
	exercise(t, NewSecretService())
	t.Log("PASS: desktop Secret Service store/load/delete; only an ephemeral fixture item was used")
}
