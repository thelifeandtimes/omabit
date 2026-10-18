package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
)

func TestRC11StalledLogoutCannotCancelLocalCleanup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/~/logout" {
			fmt.Fprint(w, "~zod")
			return
		}
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx := context.Background()
	r, err := collection.Open(filepath.Join(t.TempDir(), "collector"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := r.EnrollShip(ctx, "fixture", server.URL, "~zod", "file")
	if err != nil {
		t.Fatal(err)
	}
	store, err := storeFor(r, s)
	if err != nil {
		t.Fatal(err)
	}
	ref := collection.NewSessionRef()
	copy := s
	copy.SessionRef = ref
	data, _ := json.Marshal(map[string]any{"origin": s.Origin, "ship": s.Ship, "cookie_name": "urbauth-~zod", "cookie_value": "synthetic-test-token", "expires": time.Now().Add(time.Hour)})
	defer clear(data)
	// An ambiguous Save failure exercises persisted pending intent without a
	// current session; the same cleanup path handles a losing/interrupted login.
	injected := errors.New("interrupted publication")
	if _, err := r.SaveShipSession(ctx, s, ref, func(ctx context.Context) error {
		if err := store.Save(ctx, secretKey(copy), data); err != nil {
			return err
		}
		return injected
	}); !errors.Is(err, injected) {
		t.Fatal(err)
	}
	s, err = r.Ship(ctx, s.Name)
	if err != nil || len(s.Retired) != 1 {
		t.Fatal("pending reference lost", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	confirmed, err := cleanupRetired(canceled, r, s)
	if err != nil || confirmed {
		t.Fatal("stalled remote blocked local cleanup or claimed confirmation", err)
	}
	if err := r.ForgetShip(ctx, s.Name, s.Origin); err != nil {
		t.Fatal("cleanup intent not completed", err)
	}
	files, err := os.ReadDir(r.CredentialsDir())
	if err != nil || len(files) != 0 {
		t.Fatal("credential survived cleanup", err)
	}
}
