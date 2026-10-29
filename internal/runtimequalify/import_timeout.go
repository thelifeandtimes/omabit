package runtimequalify

import (
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// Budget five full reads plus the destination write at an aggregate 20 MiB/s,
// with 30 minutes for inspection/sync/enrollment. Large imports must not inherit
// the former fixed 30-minute cap. Parent cancellation still stops work and leaves
// the journaled source/partial target for explicit recovery, never silent retry.
func localImportTimeout(size int64) time.Duration {
	if size < 0 {
		size = 0
	}
	if size > pierstore.MaxLocalPierBytes {
		size = pierstore.MaxLocalPierBytes
	}
	const rate int64 = 20 << 20
	return 30*time.Minute + time.Duration((6*size+rate-1)/rate)*time.Second
}
