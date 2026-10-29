// Package resourcepolicy separates start admission from resource recommendations.
// Values describe currently available host capacity, not container limits.
package resourcepolicy

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const MiB uint64 = 1 << 20
const fileName = "resource-policy.json"

type Capacity struct {
	RAMMiB  uint64 `json:"ram_mib"`
	DiskMiB uint64 `json:"disk_mib"`
}
type Thresholds struct {
	Minimum     Capacity `json:"minimum"`
	Recommended Capacity `json:"recommended"`
}
type Policy struct {
	Version  int        `json:"version"`
	New      Thresholds `json:"new"`
	Existing Thresholds `json:"existing"`
}

func Default() Policy {
	return Policy{Version: 1,
		New:      Thresholds{Capacity{5120, 8192}, Capacity{8192, 16384}},
		Existing: Thresholds{Capacity{1024, 2048}, Capacity{4096, 8192}},
	}
}
func (p Policy) Validate() error {
	if p.Version != 1 {
		return errors.New("unsupported resource policy version")
	}
	for _, t := range []Thresholds{p.New, p.Existing} {
		// Reject zero/overflow and inconsistent recommendations. The owner can tune
		// below the defaults; resource needs vary with the ship and its workloads.
		for _, n := range []uint64{t.Minimum.RAMMiB, t.Minimum.DiskMiB, t.Recommended.RAMMiB, t.Recommended.DiskMiB} {
			if n == 0 || n > 1<<40 {
				return errors.New("resource values must be positive MiB values no greater than 1099511627776")
			}
		}
		if t.Recommended.RAMMiB < t.Minimum.RAMMiB || t.Recommended.DiskMiB < t.Minimum.DiskMiB {
			return errors.New("recommended resources cannot be less than minimum resources")
		}
	}
	return nil
}
func (p Policy) For(initialized bool) Thresholds {
	if initialized {
		return p.Existing
	}
	return p.New
}
func (p Policy) Admit(initialized bool, ram, disk uint64) error {
	t := p.For(initialized)
	if ram/MiB >= t.Minimum.RAMMiB && disk/MiB >= t.Minimum.DiskMiB {
		return nil
	}
	phase := "new ship provisioning"
	if initialized {
		phase = "existing ship start"
	}
	return fmt.Errorf("%s requires at least %d MiB available RAM and %d MiB free disk (available: %d MiB RAM, %d MiB disk); recommended: %d MiB RAM and %d MiB disk; configure with urbit-runner resource-policy set --phase %s", phase, t.Minimum.RAMMiB, t.Minimum.DiskMiB, ram/MiB, disk/MiB, t.Recommended.RAMMiB, t.Recommended.DiskMiB, map[bool]string{false: "new", true: "existing"}[initialized])
}
func Path(home string) string { return filepath.Join(home, ".local/share/urbit-runner", fileName) }

func openDir(home string, create bool) (*os.Root, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return nil, errors.New("resource policy requires an absolute owner home")
	}
	dir := home
	for _, part := range []string{".local", "share", "urbit-runner"} {
		dir = filepath.Join(dir, part)
		if create {
			if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return nil, err
			}
		}
		i, err := os.Lstat(dir)
		if err != nil {
			return nil, err
		}
		s, ok := i.Sys().(*syscall.Stat_t)
		if !ok || !i.IsDir() || s.Uid != uint32(os.Geteuid()) || i.Mode().Perm()&0022 != 0 {
			return nil, errors.New("unsafe resource policy directory")
		}
	}
	return os.OpenRoot(dir)
}
func safeFile(f *os.File) error {
	i, err := f.Stat()
	if err != nil {
		return err
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	if !ok || !i.Mode().IsRegular() || s.Uid != uint32(os.Geteuid()) || s.Nlink != 1 || i.Mode().Perm() != 0600 || i.Size() > 8192 {
		return errors.New("unsafe resource policy file")
	}
	return nil
}
func load(r *os.Root) (Policy, error) {
	f, err := r.OpenFile(fileName, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Policy{}, err
	}
	defer f.Close()
	if err = safeFile(f); err != nil {
		return Policy{}, err
	}
	var p Policy
	d := json.NewDecoder(io.LimitReader(f, 8193))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil {
		return Policy{}, errors.New("invalid resource policy JSON")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return Policy{}, errors.New("invalid resource policy trailing data")
	}
	return p, p.Validate()
}

// Load is read-only. Absence uses defaults; an invalid saved policy fails closed
// for new starts, without disabling observation or Stop.
func Load(home string) (Policy, error) {
	r, err := openDir(home, false)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Policy{}, err
	}
	defer r.Close()
	return load(r)
}

// Update serializes read/modify/write so separate phase edits cannot be lost.
// Rename and fsync make the complete validated policy visible atomically.
func Update(home string, change func(*Policy) error) (Policy, error) {
	r, err := openDir(home, true)
	if err != nil {
		return Policy{}, err
	}
	defer r.Close()
	lock, err := r.OpenFile("resource-policy.lock", os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return Policy{}, err
	}
	defer lock.Close()
	if err = safeFile(lock); err != nil {
		return Policy{}, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return Policy{}, err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	p, err := load(r)
	if err != nil {
		return Policy{}, err
	}
	if err = change(&p); err != nil {
		return Policy{}, err
	}
	if err = p.Validate(); err != nil {
		return Policy{}, err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return Policy{}, err
	}
	b = append(b, '\n')
	var nonce [12]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return Policy{}, err
	}
	name := ".resource-policy-" + hex.EncodeToString(nonce[:])
	defer r.Remove(name)
	f, err := r.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return Policy{}, err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return Policy{}, err
	}
	if err = r.Rename(name, fileName); err != nil {
		return Policy{}, err
	}
	dir, err := r.Open(".")
	if err != nil {
		return Policy{}, err
	}
	defer dir.Close()
	return p, dir.Sync()
}
