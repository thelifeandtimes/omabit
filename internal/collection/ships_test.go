package collection

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestShipEnrollmentAndSessionCAS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			t.Error("enrollment sent credentials")
		}
		fmt.Fprint(w, "~zod")
	}))
	defer server.Close()
	r := registry(t)
	ctx := context.Background()
	entry, err := r.EnrollShip(ctx, "alpha", server.URL, "~zod", "file")
	if err != nil {
		t.Fatal(err)
	}
	if entry.SessionRef != "" {
		t.Fatal("enrollment authenticated implicitly")
	}
	if got, err := r.EnrollShip(ctx, "alpha", server.URL, "~zod", "file"); err != nil || !reflect.DeepEqual(got, entry) {
		t.Fatalf("non-idempotent enrollment: %v", err)
	}
	if _, err := r.EnrollShip(ctx, "alias", server.URL, "~zod", "file"); !errors.Is(err, ErrShipConflict) {
		t.Fatalf("duplicate origin: %v", err)
	}
	ref := NewSessionRef()
	active, err := r.SetShipSession(ctx, entry, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.SetShipSession(ctx, entry, NewSessionRef()); !errors.Is(err, ErrShipConflict) {
		t.Fatal("slow login overwrote new session")
	}
	if _, err := r.SetShipSession(ctx, entry, ""); !errors.Is(err, ErrShipConflict) {
		t.Fatal("slow logout erased new session")
	}
	if err := r.ForgetShip(ctx, entry.Name, entry.Origin); err == nil {
		t.Fatal("forgot referenced credential")
	}
	if _, err := r.SetShipSession(ctx, active, ""); err != nil {
		t.Fatal(err)
	}
	if err := r.ForgetShip(ctx, entry.Name, entry.Origin); err == nil {
		t.Fatal("forgot pending cleanup intent")
	}
	if err := r.CompleteCredentialCleanup(ctx, entry.ID, []string{ref}); err != nil {
		t.Fatal(err)
	}
	if err := r.ForgetShip(ctx, entry.Name, entry.Origin); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(r.dir, "ships.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "cookie") || strings.Contains(string(data), "password") {
		t.Fatal("enrollment contains secret fields")
	}
}
