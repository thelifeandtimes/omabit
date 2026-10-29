//go:build linux && amd64

package runtimequalify

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// This separate contract preserves ordinary v1 and coordinated-replacement
// contracts while describing a later ordinary resume in the recovered namespace.
// Its labels/digest retain this retirement and the earlier recovery separately.
func newRecoveredResumeProjectMemberContract(project ProjectPreparationSelection, history projectnetwork.AnchorRecoveryHistory, retired projectnetwork.MemberRetirement, operation, home, mountSource, imageID string) (projectMemberContract, error) {
	if !reflect.DeepEqual(project.Plan, history.Plan) {
		return projectMemberContract{}, errors.New("post-recovery resume plan differs from history")
	}
	creation, err := projectnetwork.SelectRecoveredResumeCreation(history, retired, operation)
	if err != nil {
		return projectMemberContract{}, err
	}
	c, err := newProjectMemberContract(projectMemberSelection{Project: project, Anchor: creation.OriginalAnchor, InstanceID: creation.InstanceID, RuntimeID: creation.RuntimeID, OperationID: creation.OperationID, ContainerGeneration: creation.Generation, Mode: "resume", Storage: creation.Storage}, home, mountSource, imageID)
	if err != nil {
		return projectMemberContract{}, err
	}
	c.postRecovery = &creation
	body, err := json.Marshal(struct {
		Domain                 string
		OriginalContractDigest string
		Creation               projectnetwork.RecoveredResumeCreation
		Arguments              []string
	}{"omabit-project-post-recovery-resume-v1", c.digest, creation, c.createArgs()})
	if err != nil {
		return projectMemberContract{}, err
	}
	c.digest = fmt.Sprintf("%x", sha256.Sum256(body))
	return c, nil
}
