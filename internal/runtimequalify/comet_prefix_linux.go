//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
)

func isCometPrefixRecovery(kind string) bool {
	return kind == "comet-prefix-recovery-1" || kind == "comet-prefix-recovery-2"
}

func isCometPrefixRuntime(kind string) bool {
	return kind == "comet-prefix-restart-1" || kind == "comet-prefix-restart-2" || kind == "comet-prefix-recovered-1" || kind == "comet-prefix-recovered-2"
}

func (q *qualifier) cometPrefixResumeReady(kind string) error {
	if kind == "comet-resume" {
		if err := q.cometEpochResumeReady(kind); err != nil {
			return err
		}
		if len(q.r.PrefixPrunes) != 2 {
			return errors.New("final comet resume requires both prefix/recovery rounds")
		}
		for _, p := range q.r.PrefixPrunes {
			if p == nil || p.Uncertain || p.FinishedAt == nil || p.PostRestart == nil || p.Recovery == nil || p.Recovery.Uncertain || p.Recovery.FinishedAt == nil || p.Recovery.PostRestart == nil {
				return errors.New("comet prefix/recovery outcome is incomplete")
			}
		}
		return nil
	}
	if isCometEpochRestart(kind) {
		if len(q.r.PrefixPrunes) != 0 {
			return errors.New("epoch preparation cannot resume after prefix deletion")
		}
		return q.cometEpochResumeReady(kind)
	}
	if !isCometPrefixRuntime(kind) {
		return errors.New("unsupported comet prefix restart")
	}
	round := int(kind[len(kind)-1] - '0')
	p, err := q.prefixRecord(round)
	if err != nil {
		return err
	}
	if kind == fmt.Sprintf("comet-prefix-restart-%d", round) {
		if p.PostRestart != nil || p.Recovery != nil {
			return errors.New("comet post-prune restart cannot repeat")
		}
	} else if p.PostRestart == nil || p.Recovery == nil || p.Recovery.Uncertain || p.Recovery.FinishedAt == nil || p.Recovery.PostRestart != nil {
		return errors.New("comet post-recovery restart requires resolved replay")
	}
	return nil
}

func (q *qualifier) cometPrefixFinish(ctx context.Context) error {
	if err := q.cometPrefixResumeReady("comet-resume"); err != nil {
		return err
	}
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return err
	}
	if validEpochSample(inv, q.r.RunID, 2) != nil || !sameEpochNames(q.r.PrefixPrunes[1].Recovery.PostRestart, inv) {
		return errors.New("final comet replacement changed the retained newest-two epoch set")
	}
	q.r.CometFinalEpochs = inv
	return q.save()
}
