package runtimequalify

import (
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func TestLocalImportBudgetIncludesLargeCopyAndVerification(t *testing.T) {
	if localImportTimeout(0) != 30*time.Minute || localImportTimeout(-1) != 30*time.Minute {
		t.Fatal("invalid minimum budget")
	}
	previous := localImportTimeout(0)
	for _, size := range []int64{100_000_000_001, 512 << 30, pierstore.MaxLocalPierBytes} {
		got := localImportTimeout(size)
		if got <= previous || got < 30*time.Minute+time.Duration(size/(20<<20))*6*time.Second {
			t.Fatal("large import verification budget", size, got)
		}
		previous = got
	}
	if localImportTimeout(1<<62) != previous {
		t.Fatal("unbounded or overflowed duration")
	}
}
