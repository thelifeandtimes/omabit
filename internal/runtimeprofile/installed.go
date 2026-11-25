package runtimeprofile

// Local profiles are installed by the Unix owner, never uploaded through the
// runner API. The content-derived ID binds every field, including compatibility
// assertions. A channel is provenance at selection time, not a mutable pin.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var installedStore struct {
	sync.RWMutex
	directory string
}
var localID = regexp.MustCompile(`^local-[0-9a-f]{56}$`)
var hexDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var versionName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,95}$`)

// ConfigureStore is called only after the CLI validates the pier-owning Unix
// identity. An empty directory disables locally installed profiles.
func ConfigureStore(directory string) {
	installedStore.Lock()
	defer installedStore.Unlock()
	installedStore.directory = directory
}
func StoreDirectory() string {
	installedStore.RLock()
	defer installedStore.RUnlock()
	return installedStore.directory
}
func IsLocalID(id string) bool { return localID.MatchString(id) }
func LocalID(p Profile) string {
	fields := []string{strconv.Itoa(p.Format), p.Kind, p.Platform, p.VereVersion, p.Image, p.ManifestDigest, p.ConfigDigest, p.BinarySHA256, p.SourceBase, p.PatchRevision, p.Requirement, p.ReviewURL, p.ArchiveSHA256, strconv.FormatInt(p.ArchiveBytes, 10), p.Channel, strconv.Itoa(int(p.MaxLoom)), p.CompatibleVersions}
	sum := sha256.Sum256([]byte(strings.Join(fields, "\n")))
	return "local-" + hex.EncodeToString(sum[:])[:56]
}
func ValidateLocal(p Profile) error {
	if p.Format != 1 || p.Kind != "local" || p.Platform != Platform || !versionName.MatchString(p.VereVersion) || !hexDigest.MatchString(p.BinarySHA256) || !hexDigest.MatchString(p.ArchiveSHA256) || !hexDigest.MatchString(strings.TrimPrefix(p.ManifestDigest, "sha256:")) || !strings.HasPrefix(p.ManifestDigest, "sha256:") || !hexDigest.MatchString(strings.TrimPrefix(p.ConfigDigest, "sha256:")) || !strings.HasPrefix(p.ConfigDigest, "sha256:") || p.Image != "omabit/vere@"+p.ManifestDigest || p.ArchiveBytes <= 0 || p.ArchiveBytes > 1<<30 || p.MaxLoom < 30 || p.MaxLoom > 40 || p.ID != LocalID(p) {
		return errors.New("invalid content-addressed local runtime profile")
	}
	if p.Channel != "" && p.Channel != "live" && p.Channel != "edge" && p.Channel != "dev" {
		return errors.New("unsupported Vere channel")
	}
	for _, v := range []string{p.SourceBase, p.PatchRevision, p.Requirement, p.ReviewURL, p.CompatibleVersions} {
		if len(v) > 512 || strings.ContainsAny(v, "\x00\n\r") {
			return errors.New("invalid runtime provenance")
		}
	}
	for _, v := range strings.Fields(p.CompatibleVersions) {
		if !versionName.MatchString(v) {
			return errors.New("invalid compatible Vere version")
		}
	}
	return nil
}
func privateFile(path string, directory bool) error {
	s, err := os.Lstat(path)
	if err != nil {
		return err
	}
	st, ok := s.Sys().(*syscall.Stat_t)
	if !ok || int(st.Uid) != os.Geteuid() || s.Mode().Perm()&0077 != 0 || directory && !s.IsDir() || !directory && !s.Mode().IsRegular() {
		return fmt.Errorf("runtime registry path must be owner-private: %s", path)
	}
	return nil
}
func readInstalled(id string) (Profile, error) {
	var p Profile
	base := StoreDirectory()
	if base == "" || !IsLocalID(id) {
		return p, errors.New("local runtime registry is unavailable on this runner")
	}
	dir := filepath.Join(base, id)
	for _, path := range []string{base, dir} {
		if err := privateFile(path, true); err != nil {
			return p, err
		}
	}
	path := filepath.Join(dir, "profile.json")
	if err := privateFile(path, false); err != nil {
		return p, err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return p, err
	}
	defer f.Close()
	if info, err := f.Stat(); err != nil || !info.Mode().IsRegular() || info.Size() > 16384 {
		return p, errors.New("runtime manifest must be a regular file of at most16 KiB")
	}
	dec := json.NewDecoder(io.LimitReader(f, 16385))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&p); err != nil {
		return p, err
	}
	var extra any
	if dec.Decode(&extra) != io.EOF || p.ID != id {
		return Profile{}, errors.New("runtime metadata identity mismatch")
	}
	return p, ValidateLocal(p)
}
func installedCatalogue() []Profile {
	base := StoreDirectory()
	if base == "" || privateFile(base, true) != nil {
		return nil
	}
	dirs, err := os.ReadDir(base)
	if err != nil || len(dirs) > 256 {
		return nil
	}
	out := []Profile{}
	for _, d := range dirs {
		if IsLocalID(d.Name()) {
			if p, err := readInstalled(d.Name()); err == nil {
				out = append(out, p)
			}
		}
	}
	return out
}

// InstallMetadata atomically publishes a validated manifest without replacing
// any existing identity. Archive bytes are independently authenticated at use.
func InstallMetadata(p Profile) error {
	if err := ValidateLocal(p); err != nil {
		return err
	}
	base := StoreDirectory()
	if base == "" {
		return errors.New("owner runtime registry not configured")
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		return err
	}
	if err := privateFile(base, true); err != nil {
		return err
	}
	dir := filepath.Join(base, p.ID)
	if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	if err := privateFile(dir, true); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".profile-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	data, err := json.Marshal(p)
	if err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Link(tmp.Name(), filepath.Join(dir, "profile.json")); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	got, err := readInstalled(p.ID)
	if err != nil {
		return err
	}
	if got != p {
		return errors.New("installed runtime manifest differs")
	}
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// SetChannel affects future choices; enrolled ships retain immutable IDs.
func SetChannel(p Profile) error {
	if p.Channel == "" {
		return nil
	}
	if err := ValidateLocal(p); err != nil {
		return err
	}
	if got, err := readInstalled(p.ID); err != nil || got != p {
		return errors.New("channel requires an installed exact profile")
	}
	base := StoreDirectory()
	tmp, err := os.CreateTemp(base, ".channel-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.WriteString(p.ID + "\n"); err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(tmp.Name(), filepath.Join(base, "channel-"+p.Channel)); err != nil {
		return err
	}
	f, err := os.Open(base)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func Channels() map[string]string {
	out := map[string]string{}
	base := StoreDirectory()
	if base == "" || privateFile(base, true) != nil {
		return out
	}
	for _, channel := range []string{"live", "edge", "dev"} {
		path := filepath.Join(base, "channel-"+channel)
		if privateFile(path, false) != nil {
			continue
		}
		f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, 128))
		f.Close()
		if err != nil {
			continue
		}
		id := strings.TrimSpace(string(data))
		p, err := readInstalled(id)
		if err == nil && p.Channel == channel {
			out[channel] = id
		}
	}
	return out
}
