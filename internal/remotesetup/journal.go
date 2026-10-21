package remotesetup

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

type installJournal struct {
	Version      int               `json:"version"`
	ID           string            `json:"id"`
	Fingerprint  string            `json:"fingerprint"`
	Role         string            `json:"role"`
	Owner        ownerhost.Profile `json:"owner"`
	HelperSHA256 string            `json:"helper_sha256"`
	Stage        string            `json:"stage"`
	RunnerID     string            `json:"runner_id,omitempty"`
}

func journalDirectory() (string, error) {
	// Keep private pairing state independent of the public native-evidence tree.
	path := "/var/lib/omabit-remote-setup"
	for _, dir := range []string{"/var", "/var/lib", path} {
		info, err := os.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) {
			if err = os.Mkdir(dir, 0700); err != nil {
				return "", err
			}
			info, err = os.Lstat(dir)
		}
		if err != nil {
			return "", err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || st.Uid != 0 || info.Mode().Perm()&0022 != 0 {
			return "", errors.New("unsafe remote setup journal directory")
		}
	}
	return path, nil
}

func readInstallJournal(path string) (*installJournal, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || st.Uid != 0 || st.Nlink != 1 || info.Mode().Perm() != 0600 || info.Size() > 16384 {
		return nil, errors.New("unsafe remote setup journal")
	}
	var result installJournal
	d := json.NewDecoder(io.LimitReader(f, 16385))
	d.DisallowUnknownFields()
	if err = d.Decode(&result); err != nil {
		return nil, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF || result.Version != 1 || result.Owner.Validate() != nil || (result.Stage != "installing" && result.Stage != "installed") {
		return nil, errors.New("unsupported remote installation journal")
	}
	return &result, nil
}

func saveInstallJournal(path string, j installJournal) error {
	if _, err := readInstallJournal(path); err != nil {
		return err
	}
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".remote-install-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	err = errors.Join(err, f.Chmod(0600), f.Sync(), f.Close())
	if err != nil {
		return err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func matchesJournal(j *installJournal, r Request, fingerprint, helper string) bool {
	return j != nil && j.Version == 1 && j.ID == r.ID && j.Fingerprint == fingerprint && j.Role == r.Role && j.Owner.Validate() == nil &&
		(r.Owner == "" || r.Owner == j.Owner.Name) && (r.RunnerID == "" || j.RunnerID == r.RunnerID) &&
		(j.Stage == "installed" || j.Stage == "installing" && j.HelperSHA256 == helper)
}
