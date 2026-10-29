//go:build linux && amd64

package managed

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectCurrentMemberStopBackend interface {
	OpenCurrentProjectMemberStop(context.Context, runtimequalify.ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopDriver, error)
}
type projectCurrentMemberStopExitBackend interface {
	OpenCurrentProjectMemberStopExit(context.Context, runtimequalify.ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopExitDriver, error)
}
type projectCurrentMemberRetirementBackend interface {
	OpenCurrentProjectMemberRetirement(context.Context, runtimequalify.ProjectCurrentMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error)
}

func (realBackend) OpenCurrentProjectMemberStop(ctx context.Context, s runtimequalify.ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	return runtimequalify.OpenCurrentProjectMemberStop(ctx, s)
}
func (realBackend) OpenCurrentProjectMemberStopExit(ctx context.Context, s runtimequalify.ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	return runtimequalify.OpenCurrentProjectMemberStopExit(ctx, s)
}
func (realBackend) OpenCurrentProjectMemberRetirement(ctx context.Context, s runtimequalify.ProjectCurrentMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	return runtimequalify.OpenCurrentProjectMemberRetirement(ctx, s)
}
