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

// A recovery contract explicitly names successor/retirement authority while its
// original AnchorStart remains unchanged. No ordinary v1 digest or label changes.
// This constructor is side-effect free and grants no Create/Start authority.
func newRecoveryProjectMemberContract(project ProjectPreparationSelection, history projectnetwork.AnchorRecoveryHistory, retired projectnetwork.AnchorRetirement, instance, operation, home, mountSource, imageID string) (projectMemberContract, error) {
	if !reflect.DeepEqual(project.Plan, history.Plan) {
		return projectMemberContract{}, errors.New("replacement project differs from retained history")
	}
	creation, err := projectnetwork.SelectRecoveryMemberCreation(history, retired, instance, operation)
	if err != nil {
		return projectMemberContract{}, err
	}
	c, err := newProjectMemberContract(projectMemberSelection{Project: project, Anchor: creation.OriginalAnchor, InstanceID: creation.InstanceID, RuntimeID: creation.RuntimeID, OperationID: creation.OperationID, ContainerGeneration: creation.Generation, Mode: "resume", Storage: creation.Storage}, home, mountSource, imageID)
	if err != nil {
		return projectMemberContract{}, err
	}
	c.recovery = &creation
	body, err := json.Marshal(struct {
		Domain                 string
		OriginalContractDigest string
		Creation               projectnetwork.RecoveryMemberCreation
		Arguments              []string
	}{"omabit-project-recovered-member-v1", c.digest, creation, c.createArgs()})
	if err != nil {
		return projectMemberContract{}, err
	}
	c.digest = fmt.Sprintf("%x", sha256.Sum256(body))
	return c, nil
}
