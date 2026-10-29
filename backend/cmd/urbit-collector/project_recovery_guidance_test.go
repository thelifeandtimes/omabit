package main

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"strings"
	"testing"
)

func TestPublicRecoveryActionLostReplyGuidanceKeepsExactInputs(t *testing.T) {
	for _, kind := range []string{"start", "observe", "continue"} {
		e := &client.OutcomeUncertainError{Method: http.MethodPost, Path: protocol.ProjectRecoveryActionPath("project", "parent", kind), Err: errors.New("lost reply")}
		g := guidanceFor(e, protocol.APIError{})
		if g == nil || g.Kind != "reconcile_project_recovery" || !strings.Contains(g.Summary, "--replay-only") || !strings.Contains(g.Summary, "sequence") || len(g.Next) != 2 || g.Next[0].Name != "project-recover --help" {
			t.Fatal(g)
		}
	}
}
