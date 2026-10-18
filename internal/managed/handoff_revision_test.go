package managed

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestHandoffReusedDestinationRevisionAndInterruptedReceipt(t *testing.T) {
	for _, deletedRevision := range []uint64{1, 11} {
		t.Run(fmt.Sprint(deletedRevision), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			a, b := privateDir(t), privateDir(t)
			src, source, _ := testEndpoint(t, a, &handoffBackend{dir: a})
			backend := &handoffBackend{dir: b}
			dst, destination, closeDestination := testEndpoint(t, b, backend)
			if _, err := dst.s.db.Exec("INSERT INTO deleted_instances VALUES(?,?,?)", "arrived", "old-delete", deletedRevision); err != nil {
				t.Fatal(err)
			}
			if _, err := source.CreateFake(ctx, registration("ship", "register", 18080)); err != nil {
				t.Fatal(err)
			}
			for _, kind := range []string{"start", "stop"} {
				op, err := source.Operate(ctx, "ship", protocol.OperationRequest{Kind: kind, RequestID: kind, ExpectedRevision: 1})
				if err != nil {
					t.Fatal(err)
				}
				waitOperation(t, source, op.ID, "succeeded")
			}
			request := protocol.PrepareHandoffRequest{DestinationRunner: dst.s.runnerID, DestinationInstance: "arrived", ExpectedRevision: 1, RequestID: "move"}
			backend.failReceive.Store(true)
			if _, err := client.MoveStopped(ctx, source, destination, src.s.runnerID, "ship", request, 18081, 0); err == nil || !strings.Contains(err.Error(), "injected interruption") {
				t.Fatal("missing injected receipt interruption", err)
			}
			closeDestination()
			dst, destination, _ = testEndpoint(t, b, backend)
			// Same coordinator request resumes current transfer progress while its
			// original reservation audit receipt and selected revision survive.
			result, err := client.MoveStopped(ctx, source, destination, src.s.runnerID, "ship", request, 18081, 0)
			if err != nil || result.State != "complete" || !result.ValidCommitment(result.Commitment) {
				t.Fatal("reused destination stranded", result, err)
			}
			v := receiptLoad(t, dst, "arrived")
			if v.Public.ConfigRevision != deletedRevision+1 || v.Public.DesiredState != "stopped" || v.Runtime.Handoff.Fenced() {
				t.Fatal("destination revision/intent lost", v)
			}
			for _, id := range []string{"move", "continue-completed"} {
				request.RequestID = id
				if again, err := client.MoveStopped(ctx, source, destination, src.s.runnerID, "ship", request, 18081, 0); err != nil || !again.SameTransfer(result) || again.State != "complete" {
					t.Fatal("completed move continuation failed", again, err)
				}
			}
			if v := receiptLoad(t, src, "ship"); v.Runtime.Handoff.State != "moved" || !v.Runtime.Handoff.Fenced() {
				t.Fatal("source fence lost")
			}
		})
	}
}
