package hostcompat

import (
	"errors"
	"io/fs"
	"sync"
)

var errBudget = errors.New("host evidence work budget exceeded")

type workLimits struct {
	metadata, reads, hashes, commands int64
	operations, entries, records      int
}

var inspectionLimits = workLimits{96 << 20, 96 << 20, 512 << 20, 0, 262144, 32768, 32768}
var verificationLimits = workLimits{256 << 20, 2 << 30, 2 << 30, 4 << 30, 2097152, 262144, 1048576}

type evidenceWork struct {
	limits                            workLimits
	metadata, reads, hashes, commands int64
	operations, entries, records      int
	exhausted                         bool
	cache                             *fileDigestCache
}

func (w *evidenceWork) bytes(kind string, n int64) error {
	if n < 0 || w.exhausted {
		return errBudget
	}
	switch kind {
	case "metadata":
		if n > w.limits.metadata-w.metadata || n > w.limits.reads-w.reads {
			w.exhausted = true
			return errBudget
		}
		w.metadata += n
		w.reads += n
	case "read":
		if n > w.limits.reads-w.reads {
			w.exhausted = true
			return errBudget
		}
		w.reads += n
	case "hash":
		if n > w.limits.hashes-w.hashes {
			w.exhausted = true
			return errBudget
		}
		w.hashes += n
	case "command":
		if n > w.limits.commands-w.commands {
			w.exhausted = true
			return errBudget
		}
		w.commands += n
	}
	return nil
}
func (w *evidenceWork) count(kind string, n int) error {
	if n < 0 || w.exhausted {
		return errBudget
	}
	var used, limit *int
	switch kind {
	case "operation":
		used, limit = &w.operations, &w.limits.operations
	case "entry":
		used, limit = &w.entries, &w.limits.entries
	case "record":
		used, limit = &w.records, &w.limits.records
	}
	if used == nil || n > *limit-*used {
		w.exhausted = true
		return errBudget
	}
	*used += n
	return nil
}

type workFS struct {
	fs.FS
	work *evidenceWork
}

func (f *workFS) aptEnvironment() string {
	if env, ok := f.FS.(interface{ aptEnvironment() string }); ok {
		return env.aptEnvironment()
	}
	return ""
}

func (f *workFS) Open(name string) (fs.File, error) {
	if f.work.count("operation", 1) != nil {
		return nil, errBudget
	}
	return f.FS.Open(name)
}
func (f *workFS) Stat(name string) (fs.FileInfo, error) {
	if f.work.count("operation", 1) != nil {
		return nil, errBudget
	}
	return fs.Stat(f.FS, name)
}
func (f *workFS) Lstat(name string) (fs.FileInfo, error) {
	if f.work.count("operation", 1) != nil {
		return nil, errBudget
	}
	return fs.Lstat(f.FS, name)
}
func (f *workFS) ReadLink(name string) (string, error) {
	if f.work.count("operation", 1) != nil {
		return "", errBudget
	}
	return fs.ReadLink(f.FS, name)
}
func withWork(fsys fs.FS, limits workLimits, cache *fileDigestCache) (fs.FS, *evidenceWork) {
	if f, ok := fsys.(*workFS); ok {
		return f, f.work
	}
	w := &evidenceWork{limits: limits, cache: cache}
	return &workFS{fsys, w}, w
}
func workFor(fsys fs.FS) *evidenceWork {
	if f, ok := fsys.(*workFS); ok {
		return f.work
	}
	return nil
}
func chargeBytes(fsys fs.FS, kind string, n int64) error {
	if w := workFor(fsys); w != nil {
		return w.bytes(kind, n)
	}
	return nil
}
func chargeCount(fsys fs.FS, kind string, n int) error {
	if w := workFor(fsys); w != nil {
		return w.count(kind, n)
	}
	return nil
}

// The cache stores digests, never authorization decisions or a TTL. Every use
// opens the current trusted object and checks its full kernel identity before and
// after lookup. Unknown/synthetic filesystem identities are not cached.
type fileStamp struct {
	Dev, Ino, Nlink                      uint64
	Size                                 int64
	Mode, UID, GID                       uint32
	MTimeSec, MTimeNS, CTimeSec, CTimeNS int64
}
type fileDigestCache struct {
	mu      sync.Mutex
	entries map[fileStamp]string
}

func (c *fileDigestCache) get(key fileStamp) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.entries[key]
	return v, ok
}
func (c *fileDigestCache) put(key fileStamp, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil || len(c.entries) >= 256 {
		c.entries = make(map[fileStamp]string)
	}
	c.entries[key] = value
}

var currentFileDigests fileDigestCache
