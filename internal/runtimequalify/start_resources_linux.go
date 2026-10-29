//go:build linux && amd64

package runtimequalify

import (
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/resourcepolicy"
)

func (q *qualifier) admitStart(path string, initialized bool, stage string) error {
	policy := resourcepolicy.Default()
	if q.d.resourcePolicy != nil {
		var err error
		policy, err = q.d.resourcePolicy()
		if err != nil {
			return fmt.Errorf("%s: cannot load resource policy: %w", stage, err)
		}
	}
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("%s: %w", stage, err)
	}
	ram, disk, err := q.d.admission(path)
	if err != nil {
		return fmt.Errorf("%s: cannot read available RAM/free disk", stage)
	}
	if err = policy.Admit(initialized, ram, disk); err != nil {
		return fmt.Errorf("%s: %w", stage, err)
	}
	return nil
}
