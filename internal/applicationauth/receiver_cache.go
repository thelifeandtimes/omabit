package applicationauth

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"golang.org/x/sys/unix"
)

const receiverDescriptor = ".omabit-receiver.json"

type receiverInventory struct {
	Version int      `json:"version"`
	Entry   string   `json:"entry"`
	Files   []string `json:"files"`
}

// Retain durably publishes the nonsecret approved generation before approval is
// recorded. Reopening by digest permits cleanup after a checkout/plugin update.
func (a *Artifact) Retain(ctx context.Context, cache string) (*Artifact, error) {
	if err := a.Verify(ctx); err != nil {
		return nil, err
	}
	if _, err := collection.Open(cache); err != nil {
		return nil, ErrArtifact
	}
	temp, err := os.MkdirTemp(cache, ".receiver-")
	if err != nil {
		return nil, ErrArtifact
	}
	defer os.RemoveAll(temp)
	for _, name := range a.names {
		path := filepath.Join(temp, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, ErrArtifact
		}
		if err := durableArtifactFile(path, a.files[name]); err != nil {
			return nil, err
		}
	}
	metadata, _ := json.Marshal(receiverInventory{1, a.entry, a.names})
	if err := durableArtifactFile(filepath.Join(temp, receiverDescriptor), metadata); err != nil {
		return nil, err
	}
	// Sync all newly created directories before publishing their parent entry.
	if err := filepath.WalkDir(temp, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		return syncArtifactDir(path)
	}); err != nil {
		return nil, ErrArtifact
	}
	dest := filepath.Join(cache, a.target.SHA256)
	if err := unix.Renameat2(unix.AT_FDCWD, temp, unix.AT_FDCWD, dest, unix.RENAME_NOREPLACE); err != nil && !errors.Is(err, unix.EEXIST) {
		return nil, ErrArtifact
	}
	if err := syncArtifactDir(cache); err != nil {
		return nil, ErrArtifact
	}
	environment := map[string]string{}
	for _, entry := range a.environment {
		key, value, _ := strings.Cut(entry, "=")
		if key != "PATH" {
			environment[key] = value
		}
	}
	return LoadReceiver(cache, a.target.SHA256, environment)
}

func durableArtifactFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if err != nil {
		return ErrArtifact
	}
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if errors.Join(writeErr, syncErr, closeErr) != nil {
		return ErrArtifact
	}
	return nil
}
func syncArtifactDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func LoadReceiver(cache, digest string, environment map[string]string) (*Artifact, error) {
	hash, err := hex.DecodeString(digest)
	if err != nil || len(hash) != 32 || digest != strings.ToLower(digest) {
		return nil, ErrArtifact
	}
	if _, err := collection.Open(cache); err != nil {
		return nil, ErrArtifact
	}
	root := filepath.Join(cache, digest)
	reader := &Artifact{root: root}
	data, err := reader.read(receiverDescriptor)
	if err != nil {
		return nil, err
	}
	var inventory receiverInventory
	if json.Unmarshal(data, &inventory) != nil || inventory.Version != 1 {
		return nil, ErrArtifact
	}
	a, err := OpenReceiver(root, inventory.Entry, inventory.Files, environment)
	if err != nil || a.target.SHA256 != digest {
		return nil, ErrArtifact
	}
	return a, nil
}
