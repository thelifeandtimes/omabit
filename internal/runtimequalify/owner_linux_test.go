//go:build linux && amd64

package runtimequalify

import (
	"os"
	"testing"
)

func TestOwnerRuntimeKeepsPrivateEngineWhileAllowingNormalLoginGroups(t *testing.T) {
	f := fixture(t)
	f.i.ownerUser = true
	f.i.account = "alice"
	f.i.group = "users"
	f.i.groups = []int{f.i.gid, 998, 999}
	stat := f.d.stat
	f.d.stat = func(path string) (node, error) {
		n, e := stat(path)
		if path == f.d.home {
			n.mode = os.ModeDir | 0755
		}
		return n, e
	}
	q := &qualifier{d: f.d}
	if err := q.boundary(); err != nil {
		t.Fatal("owner account refused", err)
	}
	f.i.ownerUser = false
	if err := q.boundary(); err == nil {
		t.Fatal("uninstalled ordinary account accepted as legacy service account")
	}
	f.i.ownerUser = true
	f.i.uid++
	if err := q.boundary(); err == nil {
		t.Fatal("changed owner UID accepted")
	}
}
