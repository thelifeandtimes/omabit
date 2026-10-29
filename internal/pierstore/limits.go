package pierstore

// MaxLocalPierBytes bounds local import/copy/recovery logical contents at 8 TiB.
// It fits int64 and JavaScript's exact integer range. Source/destination identity,
// entry limits, free-space admission and full content verification still apply.
// Host-handoff streams retain MaxPierBytes for compatibility with older peers.
const MaxLocalPierBytes int64 = 8 << 40
