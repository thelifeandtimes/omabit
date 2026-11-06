package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/managed"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

func runIdentityRepair(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("repair-boot-identity", flag.ContinueOnError)
	f.SetOutput(stderr)
	var req managed.IdentityRepairRequest
	f.StringVar(&req.RunnerID, "expected-runner-id", "", "exact enrolled runner identity")
	f.StringVar(&req.InstanceID, "instance", "", "existing stable instance ID")
	f.Uint64Var(&req.Revision, "revision", 0, "current configuration revision")
	f.StringVar(&req.ExpectedShip, "expected-ship", "", "mistyped registered identity")
	f.StringVar(&req.Ship, "ship", "", "confirmed actual identity")
	f.StringVar(&req.ContainerID, "container-id", "", "full retained first-boot container ID")
	f.StringVar(&req.RequestID, "request-id", "", "stable request ID; reuse on interruption")
	confirm := f.Bool("confirm-stop-and-correct", false, "authorize verified graceful stop and same-pier registration correction")
	f.Usage = func() {
		fmt.Fprintln(stdout, "Usage: urbit-runner repair-boot-identity [flags]\nRun locally as the installed pier owner with ONLY the management service stopped.\nKeep the dedicated engine running. Retry the identical command before restarting management.\nSuccessful correction leaves the same pier stopped; use normal collector Start.")
		f.PrintDefaults()
	}
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if !*confirm || f.NArg() != 0 {
		fmt.Fprintln(stderr, "require --confirm-stop-and-correct and all explicit identity bindings")
		return 2
	}
	owner, err := ownerhost.Current()
	if err == nil && owner == nil {
		err = errors.New("installed pier owner profile required")
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	lock, err := managed.LockOfflineState(filepath.Join(owner.Home, ".local/share/urbit-runner/managed"), owner.UID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer lock.Close()
	op, err := lock.RepairBootIdentity(ctx, req)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err = json.NewEncoder(stdout).Encode(op); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
