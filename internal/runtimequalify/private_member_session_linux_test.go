//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func TestPrivateMemberSessionPureContract(t *testing.T) {
	c := privateMemberContractFixture(t, "sender", "boot")
	s := c.selection
	got, err := PrivateMemberLaunchContract(c.root, s.Plan, s.Workspace, s.Gateway, s.InstanceID, s.RuntimeID, s.OperationID)
	if err != nil || got != c.digest {
		t.Fatal("pure contract differs", err)
	}
	for _, kind := range []string{"gateway", "instance", "runtime", "operation", "directory"} {
		t.Run(kind, func(t *testing.T) {
			s := c.selection
			s.Gateway = s.Gateway.Clone()
			root := c.root
			switch kind {
			case "gateway":
				s.Gateway.Observation = nil
			case "instance":
				s.InstanceID = "unknown"
			case "runtime":
				s.RuntimeID = "../escape"
			case "operation":
				s.OperationID = s.Gateway.Selection.OperationID
			case "directory":
				root = "relative"
			}
			if digest, err := PrivateMemberLaunchContract(root, s.Plan, s.Workspace, s.Gateway, s.InstanceID, s.RuntimeID, s.OperationID); err == nil || digest != "" {
				t.Fatal("invalid pure selection accepted")
			}
		})
	}
}

func TestPrivateMemberSessionRefusesBeforeEngineOrMetadata(t *testing.T) {
	for _, kind := range []string{"legacy", "phase", "digest", "workspace", "directory", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			q, _ := privateClientFixture(t)
			c := privateMemberContractFixture(t, "sender", "boot")
			c, err := newPrivateMemberContract(c.selection, q.base, c.imageID)
			if err != nil {
				t.Fatal(err)
			}
			s, err := privatechain.NewMemberLaunch(c.selection.Plan, c.selection.Workspace, c.selection.Gateway, c.selection.InstanceID, c.selection.RuntimeID, c.selection.OperationID, c.digest, nil)
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadDir(q.base)
			if err != nil {
				t.Fatal(err)
			}
			q.d.execute = func(context.Context, []string, []string) (string, error) {
				t.Fatal("invalid session reached engine")
				return "", nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			root := q.base
			switch kind {
			case "legacy":
				s.Version = 1
			case "phase":
				s.Phase = "artifact-dispatched"
			case "digest":
				s.ContractDigest = strings.Repeat("f", 64)
			case "workspace":
				c.selection.Workspace.Phase = "planned"
			case "directory":
				root += "-different"
			case "cancel":
				cancel()
			}
			if d, err := openManagedPrivateMemberLaunch(ctx, root, c.selection.Plan, c.selection.Workspace, s, q.d); err == nil || d != nil {
				t.Fatal("invalid session opened")
			}
			after, err := os.ReadDir(q.base)
			if err != nil || len(before) != len(after) {
				t.Fatal("refusal wrote metadata")
			}
			for n := range before {
				if before[n].Name() != after[n].Name() {
					t.Fatal("metadata changed")
				}
			}
		})
	}
}

type privateMemberCloseStub struct {
	count int
	err   error
}

func (v *privateMemberCloseStub) Close() error { v.count++; return v.err }
func TestPrivateMemberSessionClosesBothHandlesOnce(t *testing.T) {
	a, b := errors.New("driver"), errors.New("client")
	driver, client := &privateMemberCloseStub{err: a}, &privateMemberCloseStub{err: b}
	s := &privateMemberSessionDriver{driver: driver, client: client}
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Close()
			if !errors.Is(err, a) || !errors.Is(err, b) {
				t.Error("close evidence lost")
			}
		}()
	}
	wg.Wait()
	if !reflect.DeepEqual([]int{driver.count, client.count}, []int{1, 1}) {
		t.Fatal("close repeated")
	}
}
