//go:build linux && amd64

package runtimequalify

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"sync"
)

// A fleet display reconstructs and brackets the same project several times.
// Read native package evidence once for this read-only response. Each engine,
// container, namespace and storage identity is still freshly checked. The next
// response starts a new scope; mutation and login-code entry points never use
// it. No admission result or timer-based authorization cache survives this call.
func fleetObservationDependencies(d dependencies) dependencies {
	inspect := d.InspectHost
	if inspect == nil {
		inspect = hostcompat.InspectCurrent
	}
	var once sync.Once
	var report hostcompat.Report
	d.InspectHost = func() hostcompat.Report {
		once.Do(func() { report = inspect() })
		// Evaluate copies derived checks and only reads the nested package evidence.
		return report
	}
	return d
}
