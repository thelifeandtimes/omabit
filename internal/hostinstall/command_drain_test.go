package hostinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// This helper never invokes systemctl, Docker or another installed service.
// Its only descendant is this test binary, with a finite three-second lifetime.
func TestCommandDrainHelper(t *testing.T) {
	mode := os.Getenv("OMABIT_TEST_COMMAND_DRAIN")
	if mode == "" {
		return
	}
	if mode == "hold" {
		time.Sleep(3 * time.Second)
		os.Exit(0)
	}
	if mode == "wait" {
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}
	if mode == "overflow" {
		fmt.Print(strings.Repeat("x", maxOutput+1))
		os.Exit(0)
	}
	if mode != "success" && mode != "failure" {
		os.Exit(91)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestCommandDrainHelper$")
	child.Env = []string{"OMABIT_TEST_COMMAND_DRAIN=hold"}
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	if child.Start() != nil {
		os.Exit(92)
	}
	fmt.Print("inert-complete-line\n")
	if mode == "failure" {
		os.Exit(7)
	}
	os.Exit(0)
}
func TestCommandDrainRequiresSuccessfulUncancelledBoundedProcess(t *testing.T) {
	for _, mode := range []string{"success", "failure", "overflow", "wait"} {
		t.Run(mode, func(t *testing.T) {
			timeout := 10 * time.Second
			if mode == "wait" {
				timeout = 100 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			out, err := runCommand(ctx, os.Args[0], []string{"-test.run=^TestCommandDrainHelper$"}, []string{"OMABIT_TEST_COMMAND_DRAIN=" + mode})
			var typed *successfulCommandOutputDrain
			if err == nil || out != nil || errors.As(err, &typed) != (mode == "success") {
				t.Fatal("incomplete output was accepted or misclassified", err)
			}
			if mode == "success" && (!errors.Is(err, exec.ErrWaitDelay) || acknowledgedManagementStart(ctx, err) != nil) {
				t.Fatal("successful exit acknowledgement lost", err)
			}
			if mode != "success" && acknowledgedManagementStart(ctx, err) == nil {
				t.Fatal("unsuccessful command acknowledged")
			}
		})
	}
}
func TestCommandDrainRefusesBareErrorAndCancellation(t *testing.T) {
	for _, kind := range []string{"bare", "text", "cancelled", "joined-cancelled", "joined-deadline"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var err error = &successfulCommandOutputDrain{}
			switch kind {
			case "bare":
				err = exec.ErrWaitDelay
			case "text":
				err = errors.New(exec.ErrWaitDelay.Error())
			case "cancelled":
				cancel()
			case "joined-cancelled":
				err = errors.Join(err, context.Canceled)
			case "joined-deadline":
				err = errors.Join(err, context.DeadlineExceeded)
			}
			if acknowledgedManagementStart(ctx, err) == nil {
				t.Fatal("unproven successful exit accepted")
			}
		})
	}
}
func TestCommandDrainOrdinaryProbeStillRefusesOutput(t *testing.T) {
	f := newFixture(t)
	f.i.run = func(context.Context, string, []string, []string) ([]byte, error) {
		return []byte(`{"partial":`), &successfulCommandOutputDrain{}
	}
	out, err := f.i.command(context.Background(), "/usr/bin/dockerd", "--version")
	if err == nil || out != nil {
		t.Fatal("ordinary probe accepted incomplete output")
	}
}
func TestCommandDrainActivationStillRequiresIndependentVerification(t *testing.T) {
	for _, verified := range []bool{false, true} {
		t.Run(fmt.Sprint(verified), func(t *testing.T) {
			f, p, probe := ownerFixture(t)
			run := f.i.run
			starts := 0
			drained := false
			checks := 0
			f.i.run = func(ctx context.Context, path string, args, env []string) ([]byte, error) {
				out, err := run(ctx, path, args, env)
				if err == nil && path == "/usr/bin/systemctl" && slices.Contains(args, "urbit-runner.service") && slices.Contains(args, "enable") && slices.Contains(args, "--now") {
					starts++
					drained = true
					return nil, &successfulCommandOutputDrain{}
				}
				return out, err
			}
			rpc := f.i.updateRPC
			f.i.updateRPC = func(ctx context.Context, socket string, uid uint32, pid int, req *protocol.ManagementUpdateRequest) (protocol.ManagementUpdateStatus, error) {
				if drained {
					checks++
					if !verified {
						return protocol.ManagementUpdateStatus{}, errors.New("synthetic unavailable management peer")
					}
				}
				return rpc(ctx, socket, uid, pid, req)
			}
			result, err := f.i.installOwner(context.Background(), "/bundle", hostPublic(1), []byte("runner"), p, probe, nil)
			if starts != 1 || checks == 0 || (err == nil) != verified || result.Ready != verified {
				t.Fatal("service effect replayed or verification bypassed", starts, checks, result, err)
			}
			if !verified && !result.Partial {
				t.Fatal("failed verification lost partial installation")
			}
		})
	}
}
