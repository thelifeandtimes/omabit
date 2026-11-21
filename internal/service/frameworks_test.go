package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
)

func TestFrameworkSourceCheckNeverConnectsOrCreatesRegistry(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "sys.kelvin"), []byte("[%zuse 410]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := OpenSourceRoot("framework", directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	absent := filepath.Join(t.TempDir(), "must-remain-absent")
	s, err := New(Options{ConfigDir: absent, RequireAllowed: true, Sources: []*SourceRoot{root}})
	if err != nil {
		t.Fatal(err)
	}
	result := s.Call(ctx, "app.framework.v1", json.RawMessage(`{"name":"grubbery-7f7701d-v1"}`))
	if result.Error != nil || result.Data.(appdev.Framework).Qualification != "native-workflow-pending" {
		t.Fatal(result)
	}
	for _, tc := range []struct{ input, code string }{
		{`{"name":"grubbery-7f7701d-v1","source":{"root":"framework","directory":"."}}`, "framework_source_mismatch"},
		{`{"name":"grubbery-7f7701d-v1","source":{"root":"missing","directory":"."}}`, "source_not_allowed"},
		{`{"name":"grubbery-7f7701d-v1","source":{"root":"framework","directory":"../private"}}`, "invalid_input"},
		{`{"name":"grubbery-7f7701d-v1","source":{"root":"framework","directory":"/private"}}`, "invalid_input"},
		{`{"name":"grubbery","source":{"root":"framework","directory":"."}}`, "invalid_input"},
		{`{"name":"grubbery-7f7701d-v1","source":{"root":"framework","directory":"."},"grant":true}`, "invalid_input"},
	} {
		result := s.Call(ctx, "app.verify-framework-source.v1", json.RawMessage(tc.input))
		if result.Error == nil || result.Error.Code != tc.code || result.Error.Dispatch != "not-dispatched" {
			t.Fatal(result)
		}
	}
	if err := os.Symlink(directory, filepath.Join(directory, "linked")); err != nil {
		t.Fatal(err)
	}
	result = s.Call(ctx, "app.verify-framework-source.v1", json.RawMessage(`{"name":"grubbery-7f7701d-v1","source":{"root":"framework","directory":"."}}`))
	if result.Error == nil || result.Error.Code != "framework_source_unavailable" {
		t.Fatal(result)
	}
	if _, err := os.Stat(absent); !os.IsNotExist(err) {
		t.Fatal("offline source check created registry", err)
	}
}
