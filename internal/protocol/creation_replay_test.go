package protocol

import "testing"

func TestReplayOnlyEndpointsExcludeUnguardedEffects(t *testing.T) {
	for _, path := range []string{"/v1/instances", "/v1/comets", "/v1/ships", "/v1/projects", PrivateChainProjectsPath, ProjectStopReconcilePath("project-one"), ProjectActionPath("project-one"), ProjectRecoveryActionPath("project-one", "", "start"), ProjectRecoveryActionPath("project-one", "op-parent", "observe"), ProjectRecoveryActionPath("project-one", "op-parent", "continue")} {
		if !ReplayOnlyEndpoint("POST", path) || ReplayOnlyEndpoint("PUT", path) || ReplayOnlyEndpoint("GET", path) {
			t.Fatal("invalid guarded endpoint", path)
		}
	}
	for _, path := range []string{"/hello", "/v1/instances/ship/operations", "/v1/projects/project-one/operations/extra", "/v1/projects//operations", "/v1/projects/project-one/recovery/bad/unknown", "/v1/projects/project-one/recovery//continuations", DeskOperationsPath, DeskStagesPath, AppPolicyPath} {
		if ReplayOnlyEndpoint("POST", path) {
			t.Fatal("unguarded endpoint admitted", path)
		}
	}
}
