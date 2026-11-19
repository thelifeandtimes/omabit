package recording

import (
	"testing"
	"time"
)

func TestCapacityAdmissionsBoundHostShipJobsAndFreeReserve(t *testing.T) {
	p := DefaultRetentionPolicy()
	c := Capacity{Version: 1, RunnerID: "runner", Jobs: 4, CommittedBytes: 256 << 20, AvailableBytes: 10 << 30, Instances: []InstanceCommitment{{InstanceID: "one", Jobs: 4, CommittedBytes: 256 << 20}}, ReadAt: time.Now()}
	if c.Admit(p, "one") == nil || c.Admit(p, "two") != nil {
		t.Fatal("instance maximum not enforced independently")
	}
	p.Instances = []InstanceRetention{{InstanceID: "one", MaxRawBytes: 512 << 20}}
	if c.Admit(p, "one") != nil {
		t.Fatal("explicit ship override ignored")
	}
	for _, name := range []string{"host", "jobs", "free", "malformed"} {
		t.Run(name, func(t *testing.T) {
			policy := p
			capacity := c
			switch name {
			case "host":
				policy.HostMaxRawBytes = 256 << 20
				policy.Instances = nil
				policy.Instances = []InstanceRetention{}
			case "jobs":
				policy.MaxJobs = 4
			case "free":
				capacity.AvailableBytes = policy.ReserveFreeBytes + uint64(policy.MaxSessionBytes) - 1
			case "malformed":
				capacity.CommittedBytes--
			}
			if capacity.Admit(policy, "two") == nil {
				t.Fatal("exhausted/unknown commitment admitted")
			}
		})
	}
}
