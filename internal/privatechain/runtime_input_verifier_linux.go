//go:build linux

package privatechain

import (
	"context"
	"os"
	"sync"
)

// RuntimeInputVerifier holds one canonical sealed input root. Opening it checks
// the root and review identifiers only; Verify must succeed to qualify contents.
// Every Verify rehashes the full manifest/tree and rechecks the pathname binding.
// Neither opening nor verification grants runtime authority or excludes another
// process with the same UID. There are no execution or repair methods.
type RuntimeInputVerifier struct {
	mu                                       sync.Mutex
	root                                     *os.Root
	identity                                 os.FileInfo
	directory, manifestSHA, foundationDigest string
}

func OpenRuntimeInputVerifier(ctx context.Context, directory, expectedSHA, expectedFoundation string) (*RuntimeInputVerifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !hash(expectedSHA, false) || !hash(expectedFoundation, false) {
		return nil, errRuntimeInputs
	}
	root, err := openInputRoot(directory)
	if err != nil {
		return nil, errRuntimeInputs
	}
	info, err := root.Stat(".")
	if err != nil {
		root.Close()
		return nil, errRuntimeInputs
	}
	v := &RuntimeInputVerifier{root: root, identity: info, directory: directory, manifestSHA: expectedSHA, foundationDigest: expectedFoundation}
	if err = v.currentRoot(); err != nil {
		v.Close()
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		v.Close()
		return nil, err
	}
	return v, nil
}

func (v *RuntimeInputVerifier) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.root == nil {
		return nil
	}
	root := v.root
	v.root = nil
	return root.Close()
}

// currentRoot is called with exclusive ownership of the handle or its mutex.
// Reopening the name is essential: an os.Root can still read an old directory
// after its original pathname has been moved, replaced or made an alias.
func (v *RuntimeInputVerifier) currentRoot() error {
	if v.root == nil || v.identity == nil {
		return errRuntimeInputs
	}
	held, err := v.root.Stat(".")
	if err != nil || held.Mode() != os.ModeDir|0500 || !os.SameFile(v.identity, held) {
		return errRuntimeInputs
	}
	if _, err = inputStat(held); err != nil {
		return errRuntimeInputs
	}
	named, err := openInputRoot(v.directory)
	if err != nil {
		return errRuntimeInputs
	}
	defer named.Close()
	current, err := named.Stat(".")
	if err != nil || !os.SameFile(v.identity, current) {
		return errRuntimeInputs
	}
	return nil
}

func (v *RuntimeInputVerifier) Verify(ctx context.Context) (RuntimeInputInspection, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return RuntimeInputInspection{}, err
	}
	if err := v.currentRoot(); err != nil {
		return RuntimeInputInspection{}, err
	}
	result, err := inspectRuntimeInputRoot(ctx, v.root, v.manifestSHA, v.foundationDigest)
	if err != nil {
		return RuntimeInputInspection{}, err
	}
	if err = v.currentRoot(); err != nil {
		return RuntimeInputInspection{}, err
	}
	if err = ctx.Err(); err != nil {
		return RuntimeInputInspection{}, err
	}
	return result, nil
}
