//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type appRequestLifetimeBackend struct {
	*managedAppNativeFixture
	once    sync.Once
	entered chan context.Context
	release chan struct{}
}

type appRequestLifetimeRuntime struct {
	managedAppNativeRuntime
	backend *appRequestLifetimeBackend
}

func (b *appRequestLifetimeBackend) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	return appRequestLifetimeRuntime{managedAppNativeRuntime{b.managedAppNativeFixture}, b}, nil
}

func (runtime appRequestLifetimeRuntime) GrubberyRead(ctx context.Context, request grubbery.ReadRequest) (grubbery.NativeResponse, error) {
	runtime.backend.once.Do(func() {
		runtime.backend.entered <- ctx
		<-runtime.backend.release
	})
	if ctx.Err() != nil {
		return grubbery.NativeResponse{}, ctx.Err()
	}
	return runtime.managedAppNativeRuntime.GrubberyRead(ctx, request)
}

func TestGrubberyAcceptedPhaseSurvivesCallerCancellationButHonorsRunnerLifetime(t *testing.T) {
	for _, scenario := range []string{"accepted-phase", "read-only-review", "runner-stopping"} {
		t.Run(scenario, func(t *testing.T) {
			r, fixture, plan, _ := managedGrubberyFixture(t)
			path := protocol.GrubberyAppOperationsPath + "/" + plan.RequestID + "/advance"
			var input any = protocol.GrubberyAppPhaseRequest{PlanDigest: plan.Digest, StepID: plan.Steps[0].ID}
			if scenario == "read-only-review" {
				path = protocol.GrubberyAppReviewPath
				input = protocol.GrubberyAppReviewRequest{Selection: protocol.DeskSelection{Target: plan.Binding.Target, RuntimeID: plan.Binding.RuntimeID, ConfigRevision: plan.Binding.ConfigRevision}, FrameworkReceiptID: plan.Binding.FrameworkReceiptID, AppID: plan.Binding.AppID, StageID: plan.StageID, RequestID: plan.RequestID}
			} else {
				appHTTPRecord(t, r, protocol.GrubberyAppOperationsPath, protocol.GrubberyAppBeginRequest{Plan: plan})
			}
			backend := &appRequestLifetimeBackend{managedAppNativeFixture: fixture, entered: make(chan context.Context, 1), release: make(chan struct{})}
			r.b = backend
			data, err := json.Marshal(input)
			g02Check(t, err)
			caller, cancel := context.WithCancel(context.WithValue(t.Context(), peerKey{}, uint32(os.Geteuid())))
			defer cancel()
			request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data)).WithContext(caller)
			request.Header.Set(protocol.Header, "1")
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			done := make(chan struct{})
			go func() { defer close(done); r.ServeHTTP(response, request) }()
			released := false
			defer func() {
				if !released {
					close(backend.release)
				}
			}()
			var native context.Context
			select {
			case native = <-backend.entered:
			case <-done:
				t.Fatal("request ended before selected native phase", response.Code, response.Body.String())
			case <-time.After(15 * time.Second):
				t.Fatal("selected native phase did not enter")
			}
			if scenario == "runner-stopping" {
				r.cancel()
			} else {
				cancel()
			}
			if scenario == "accepted-phase" && native.Err() != nil {
				t.Fatal("client disconnect cancelled runner-owned phase")
			}
			if scenario != "accepted-phase" && native.Err() == nil {
				t.Fatal("read-only caller/runner lifetime was ignored")
			}
			close(backend.release)
			released = true
			select {
			case <-done:
			case <-time.After(15 * time.Second):
				t.Fatal("bounded phase did not finish after releasing its fixture")
			}
			if scenario == "accepted-phase" {
				original, err := (grubberyJournal{r.s}).Load(t.Context(), plan.RequestID)
				g02Check(t, err)
				state, err := original.State(plan.Steps[0].ID)
				if err != nil || state != "observed" || fixture.writes != 1 || response.Code != http.StatusOK {
					t.Fatal("accepted exact phase did not preserve one durable outcome", state, fixture.writes, response.Code, err)
				}
			} else if fixture.writes != 0 {
				t.Fatal("cancelled read/runner dispatched a native effect")
			}
		})
	}
}
