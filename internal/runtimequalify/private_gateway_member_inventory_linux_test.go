//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestPrivateMemberInventoryRejectsIncompleteProjection(t *testing.T) {
	for _, kind := range []string{"missing-mounts", "invalid-mounts", "too-many-mounts", "missing-mode", "wrong-id", "duplicate-id", "duplicate-name", "extra-field", "missing-row", "duplicate-key", "command-error", "cancel-after-read"} {
		t.Run(kind, func(t *testing.T) {
			x := newPrivateMembersFixture(t, "boot", "running", 1)
			r := x.f.o.runtime
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			original := r.q.d.execute
			r.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				out, err := original(ctx, args, env)
				a := args[4:]
				if len(a) < 5 || !reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", privateMemberInventoryFormat}) {
					return out, err
				}
				lines := strings.Split(out, "\n")
				var row, second map[string]any
				if json.Unmarshal([]byte(lines[0]), &row) != nil || json.Unmarshal([]byte(lines[1]), &second) != nil {
					t.Fatal("fixture rows")
				}
				switch kind {
				case "missing-mounts":
					delete(row, "mounts")
				case "invalid-mounts":
					row["mounts"] = map[string]any{"Source": "unclassified"}
				case "too-many-mounts":
					row["mounts"] = make([]mount, 129)
				case "missing-mode":
					delete(row, "network_mode")
				case "wrong-id":
					row["id"] = strings.Repeat("f", 64)
				case "duplicate-id":
					row["id"] = second["id"]
				case "duplicate-name":
					row["name"] = second["name"]
				case "extra-field":
					row["environment"] = []string{"synthetic"}
				case "missing-row":
					return lines[0], nil
				case "duplicate-key":
					return strings.Replace(out, `"id":`, `"id":"invalid","id":`, 1), nil
				case "command-error":
					return out, errors.New("synthetic projection error")
				case "cancel-after-read":
					cancel()
					return out, nil
				}
				raw, _ := json.Marshal(row)
				lines[0] = string(raw)
				return strings.Join(lines, "\n"), nil
			}
			mounts, sharing, err := r.memberInventory(ctx)
			if err == nil || mounts != nil || sharing != nil {
				t.Fatal("partial inventory accepted")
			}
			if x.f.x.networkCalls+x.f.x.containerCalls+x.f.x.startCalls+x.f.x.boot.calls != 0 {
				t.Fatal("inventory mutated engine")
			}
		})
	}
}

func TestPrivateMemberInventoryReturnsCorrespondingSnapshots(t *testing.T) {
	x := newPrivateMembersFixture(t, "boot", "running", 2)
	r := x.f.o.runtime
	mounts, sharing, err := r.memberInventory(context.Background())
	if err != nil || len(mounts) != 3 || len(sharing) != 3 {
		t.Fatal(err)
	}
	for i, row := range sharing {
		if mounts[i].ID != row.ID || !reflect.DeepEqual(mounts[i].Mounts, x.f.x.inventory[row.ID]) || row != x.sharing[row.ID] {
			t.Fatal("mismatched observations")
		}
	}
}

func TestPrivateMemberInventoryRejectsListChangeDuringProjection(t *testing.T) {
	x := newPrivateMembersFixture(t, "boot", "running", 1)
	x.change = func() { delete(x.f.x.inventory, x.members[0].ContainerID) }
	mounts, sharing, err := x.f.o.runtime.memberInventory(context.Background())
	if err == nil || mounts != nil || sharing != nil {
		t.Fatal("changing container set accepted")
	}
}
