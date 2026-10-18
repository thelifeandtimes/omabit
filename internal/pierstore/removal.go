package pierstore

// RemovalTree is a durable, object-bound deletion manifest. Missing entries are
// allowed on continuation; existing entries must still be these exact objects.
// In particular, deleting .urb does not destroy the evidence needed to finish.
type RemovalTree struct {
	Entries []RemovalEntry `json:"entries"`
}

type RemovalEntry struct {
	Path      string            `json:"path"`
	Identity  DirectoryIdentity `json:"identity"`
	Directory bool              `json:"directory"`
	Mode      uint32            `json:"mode"`
	Size      int64             `json:"size"`
	Modified  int64             `json:"modified"`
	Changed   int64             `json:"changed,omitempty"`
	Links     uint64            `json:"links,omitempty"`
	mountID   uint64
}
