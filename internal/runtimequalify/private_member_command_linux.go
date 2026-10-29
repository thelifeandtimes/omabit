//go:build linux && amd64

package runtimequalify

import (
	"errors"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

// privateMemberCommand only describes the qualified keyed Vere invocation.
// It does not authorize a launch or open any key, pier, input or engine. A future
// lifecycle driver must prove the retained project/gateway selection and bind
// these fixed container paths to its own verified objects before dispatch.
func privateMemberCommand(plan privatechain.Plan, instanceID, mode string) (string, []string, error) {
	if _, err := plan.Digest(); err != nil {
		return "", nil, err
	}
	if mode != "boot" && mode != "resume" {
		return "", nil, errors.New("private member requires explicit boot or resume")
	}
	var member *privatechain.Member
	for i := range plan.Members {
		if plan.Members[i].InstanceID == instanceID {
			member = &plan.Members[i]
			break
		}
	}
	if member == nil {
		return "", nil, errors.New("instance is not in the reviewed private project")
	}
	ship := strings.TrimPrefix(member.Ship, "~")
	args := []string{"-t", "-L", "--no-conn", "--no-dock", "--loom", "31", "--snap-time", strconv.FormatUint(uint64(DefaultSnapshotMinutes), 10), "-b", "0.0.0.0", "--http-port", strconv.Itoa(int(member.InternalHTTPPort)), "-p", strconv.Itoa(int(member.InternalAmesPort))}
	if mode == "boot" {
		args = append(args, "-H", "galaxies.omabit.invalid", "-w", ship, "-k", "/run/omabit-boot/key", "-e", "http://127.0.0.1:"+strconv.Itoa(int(plan.GatewayPort)), "-B", "/run/omabit-boot/boot.pill", "-c")
	}
	return "/bin/urbit", append(args, "/urbit/"+ship), nil
}
