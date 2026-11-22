package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
)

func TestRuntimePreflightAcceptsCatalogueIDsBeforeTargetAuthorization(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent-registry")
	s, err := New(Options{ConfigDir: dir, RequireAllowed: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ profile, code string }{
		{runtimeprofile.OfficialProfile, "target_not_allowed"},
		{runtimeprofile.JSONFixProfile, "target_not_allowed"},
		{"", "invalid_input"},
		{"unreviewed-runtime", "invalid_input"},
		{"vere-4.6-unknown", "invalid_input"},
		{"../image.tar.gz", "invalid_input"},
	} {
		t.Run(tc.profile, func(t *testing.T) {
			input, err := json.Marshal(RuntimeProfileInput{Target: Target{Name: "local", RunnerID: "runner"}, RuntimeProfile: tc.profile})
			if err != nil {
				t.Fatal(err)
			}
			result := s.Call(t.Context(), "runtime.profile-preflight.v1", input)
			if result.Error == nil || result.Error.Code != tc.code || result.Error.Dispatch != "not-dispatched" {
				t.Fatalf("catalogue validation or runner authorization changed: %+v", result)
			}
		})
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("refused preflight created registry", err)
	}
}
