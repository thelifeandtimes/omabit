package runtimequalify

// DefaultSnapshotMinutes is the periodic checkpoint interval for newly created
// Vere runtimes. Event-log durability and explicit maintenance are independent.
const DefaultSnapshotMinutes uint32 = 60

// A zero launch value records omission of --snap-time, as in containers created
// before this policy. Official Vere 4.6 defaults that invocation to two minutes.
// Keep the immutable launch contract until an explicit replacement is created.
func (c ContainerRecord) SnapshotIntervalMinutes() uint32 {
	if c.SnapshotMinutes == 0 {
		return 2
	}
	return c.SnapshotMinutes
}
