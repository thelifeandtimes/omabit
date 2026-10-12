// Package pierstore defines the operator-facing pier tree. Runtime artifacts and
// the management journal are separate from these ordinary, movable directories.
package pierstore

import (
	"path"
	"regexp"
	"strings"
	"time"
)

var component = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

var Categories = []string{"fakeships", "livenet/galaxies", "livenet/stars", "livenet/planets", "livenet/moons", "livenet/comets", "archive"}

func Kind(relative string) string {
	p := strings.Split(relative, "/")
	if len(p) < 2 || len(p) > 8 || path.Clean(relative) != relative {
		return ""
	}
	for _, part := range p {
		if !component.MatchString(part) {
			return ""
		}
	}
	if p[0] == "fakeships" {
		return "fake"
	}
	if len(p) < 3 || p[0] != "livenet" {
		return ""
	}
	return map[string]string{"galaxies": "galaxy", "stars": "star", "planets": "planet", "moons": "moon", "comets": "comet"}[p[1]]
}

// Every managed pier is the fixed child of an exclusive permanent mount root.
func ValidPath(relative string) bool {
	return len(relative) <= 384 && path.Clean(relative) == relative && strings.HasSuffix(relative, "/pier") && Kind(MountPath(relative)) != "" && len(strings.Split(relative, "/")) <= 8
}

func MountPath(relative string) string { return path.Dir(relative) }

type DirectoryIdentity struct {
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
	Birth  string `json:"birth"`
}

type Identity struct {
	Device    uint64 `json:"device"`
	Inode     uint64 `json:"inode"`
	UrbDevice uint64 `json:"urb_device"`
	UrbInode  uint64 `json:"urb_inode"`
	Birth     string `json:"birth"`
	UrbBirth  string `json:"urb_birth"`
}

// SameObject reports whether two observations name the same directory object.
// st_dev is deliberately excluded: btrfs subvolumes report a per-boot anonymous
// device number, and dm/loop-backed filesystems can also be renumbered, so a
// device number recorded in one boot is not a durable identity. Inode plus the
// nanosecond birth timestamp still distinguishes a replaced or inode-reused
// directory. Records without a birth timestamp keep the strict comparison.
func (d DirectoryIdentity) SameObject(o DirectoryIdentity) bool {
	if d.Birth == "" || o.Birth == "" {
		return d == o
	}
	return d.Inode == o.Inode && d.Birth == o.Birth
}

// SameObject compares the pier and its .urb child; see DirectoryIdentity.SameObject.
func (i Identity) SameObject(o Identity) bool {
	if i.Birth == "" || o.Birth == "" || i.UrbBirth == "" || o.UrbBirth == "" {
		return i == o
	}
	return i.Inode == o.Inode && i.Birth == o.Birth && i.UrbInode == o.UrbInode && i.UrbBirth == o.UrbBirth
}

type Location struct {
	Path          string            `json:"path"`
	Identity      Identity          `json:"identity"`
	MountIdentity DirectoryIdentity `json:"mount_identity"`
	Uncertain     bool              `json:"uncertain,omitempty"`
}

type Candidate struct {
	Path          string            `json:"path"`
	Kind          string            `json:"kind"`
	Fingerprint   string            `json:"fingerprint"`
	Busy          bool              `json:"busy"`
	Problem       string            `json:"problem,omitempty"`
	EnrolledBy    string            `json:"enrolled_by,omitempty"`
	Moved         bool              `json:"moved,omitempty"`
	Identity      Identity          `json:"-"`
	MountIdentity DirectoryIdentity `json:"-"`
}

type Catalogue struct {
	RunnerID   string      `json:"runner_id"`
	Root       string      `json:"root"`
	ObservedAt time.Time   `json:"observed_at"`
	Candidates []Candidate `json:"candidates"`
	Error      string      `json:"error,omitempty"`
}
