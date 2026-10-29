//go:build linux && amd64

package runtimequalify

import (
	"errors"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// projectMemberCommand builds only the Vere invocation. It grants no storage or
// engine authority. The future member driver must bind the reserved runtime ID,
// exclusive <instance>/<ship> mount and current anchor namespace before launch.
// It must not use the standalone per-instance bridge/publication/cleanup path.
func projectMemberCommand(plan projectnetwork.Plan, instanceID, mode string) (string, []string, error) {
	if _, err := plan.Digest(); err != nil {
		return "", nil, err
	}
	if mode != "boot" && mode != "resume" {
		return "", nil, errors.New("project member requires an explicit boot or resume contract")
	}
	var member *projectnetwork.Member
	for i := range plan.Members {
		if plan.Members[i].InstanceID == instanceID {
			member = &plan.Members[i]
			break
		}
	}
	if member == nil {
		return "", nil, errors.New("instance is not in the reviewed project plan")
	}
	return projectCommandForMember(*member, mode)
}

func projectCommandForMember(member projectnetwork.Member, mode string) (string, []string, error) {
	ship := strings.TrimPrefix(member.Ship, "~")
	args := []string{"-t", "-L", "--no-conn", "--no-dock", "--loom", "31", "--snap-time", strconv.FormatUint(uint64(DefaultSnapshotMinutes), 10), "-b", "0.0.0.0", "--http-port", strconv.FormatUint(uint64(member.InternalHTTPPort), 10)}
	if mode == "boot" {
		args = append(args, "-F", ship, "-B", "/artifacts/boot.pill", "-c")
	}
	return "/bin/urbit", append(args, "/urbit/"+ship), nil
}
