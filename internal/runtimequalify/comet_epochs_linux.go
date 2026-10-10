//go:build linux && amd64

package runtimequalify

import (
	"errors"
	"fmt"
)

func isCometRoll(kind string) bool {
	return kind == "comet-roll-1" || kind == "comet-roll-2" || kind == "comet-roll-3" || kind == "comet-roll-4"
}

func isCometEpochRestart(kind string) bool {
	return kind == "comet-epoch-restart-1" || kind == "comet-epoch-restart-2" || kind == "comet-epoch-restart-3" || kind == "comet-epoch-restart-4"
}

func cometEpochProfile(profile string) bool {
	return profile == ProfileCometEpochs || profile == ProfileCometPrefix
}

func cometEpochContainer(profile, kind string) bool {
	return cometEpochProfile(profile) && (isCometRoll(kind) || isCometEpochRestart(kind)) && int(kind[len(kind)-1]-'0') <= epochRollCount(profile)
}

// Epoch evidence keeps the same roll-N names across profiles, while container
// kinds select exact fake/comet command paths and network boundaries.
func epochContainerKind(profile, kind string) string {
	if cometEpochProfile(profile) {
		return "comet-" + kind
	}
	return kind
}

func (q *qualifier) cometEpochResumeReady(kind string) error {
	e := q.r.EpochLifecycle
	if e == nil || len(e.Rolls) == 0 || len(e.Rolls) > epochRollCount(q.d.profile) {
		return errors.New("comet epoch restart requires its roll evidence")
	}
	for i, r := range e.Rolls {
		if r.Kind != fmt.Sprintf("roll-%d", i+1) || r.Uncertain || r.FinishedAt == nil || r.After == nil || (i < len(e.Rolls)-1 && r.PostRestart == nil) {
			return errors.New("comet epoch restart requires resolved preceding rolls")
		}
	}
	last := e.Rolls[len(e.Rolls)-1]
	if kind == "comet-resume" {
		if len(e.Rolls) != epochRollCount(q.d.profile) || last.PostRestart == nil {
			return errors.New("final comet resume requires all planned verified roll/restart pairs")
		}
	} else if kind != fmt.Sprintf("comet-epoch-restart-%d", len(e.Rolls)) || last.PostRestart != nil {
		return errors.New("comet epoch restart is not the next pending step")
	}
	return nil
}

func (q *qualifier) epochPierName() string {
	if q.d.managed {
		return "pier"
	}
	if cometProfile(q.d.profile) {
		return "comet"
	}
	return "fake-zod"
}
