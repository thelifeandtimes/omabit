//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func inputVerifierFixture(t *testing.T) (string, []RuntimeInputFile, *RuntimeInputVerifier) {
	t.Helper()
	directory, files := inputTreeFixture(t)
	// This small tree deliberately has an unreviewed manifest. The tests below
	// exercise held root identity; complete positive inspection uses the separate
	// qualified-generation fixture, never weakened dependency pins.
	v, err := OpenRuntimeInputVerifier(context.Background(), directory, inputSHA([]byte("{}")), strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Dir(directory)
	t.Cleanup(func() {
		v.Close()
		_ = filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
			if err == nil && info.IsDir() {
				_ = os.Chmod(p, 0700)
			}
			return nil
		})
	})
	return directory, files, v
}

func TestRuntimeInputVerifierOpenIsNotContentAuthority(t *testing.T) {
	directory, files, v := inputVerifierFixture(t)
	if err := v.currentRoot(); err != nil {
		t.Fatal(err)
	}
	before := map[string][]byte{}
	for _, name := range []string{"manifest.json", "tools/code", "tools/nested/empty"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		mustInput(t, err)
		before[name] = data
	}
	if inspection, err := v.Verify(context.Background()); err == nil || inspection != (RuntimeInputInspection{}) {
		t.Fatal("opening qualified unreviewed contents", inspection, err)
	}
	if err := verifyInputTree(context.Background(), v.root, files); err != nil {
		t.Fatal("valid synthetic tree changed", err)
	}
	for name, data := range before {
		after, err := os.ReadFile(filepath.Join(directory, name))
		mustInput(t, err)
		if !reflect.DeepEqual(data, after) {
			t.Fatal("read-only handle changed a file")
		}
	}
	for _, name := range []string{"", "tools", "tools/nested"} {
		info, err := os.Stat(filepath.Join(directory, name))
		mustInput(t, err)
		if info.Mode() != os.ModeDir|0500 {
			t.Fatal("directory mode changed")
		}
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(context.Background()); err == nil {
		t.Fatal("closed handle retained inspection access")
	}
}

func TestRuntimeInputVerifierRefusesIdenticalRootReplacement(t *testing.T) {
	directory, files, v := inputVerifierFixture(t)
	retained := directory + "-retained"
	mustInput(t, os.Rename(directory, retained))
	mustInput(t, os.MkdirAll(filepath.Join(directory, "tools", "nested"), 0700))
	for _, name := range []string{"manifest.json", "tools/code", "tools/nested/empty"} {
		source := filepath.Join(retained, name)
		data, err := os.ReadFile(source)
		mustInput(t, err)
		info, err := os.Stat(source)
		mustInput(t, err)
		mustInput(t, os.WriteFile(filepath.Join(directory, name), data, info.Mode()))
	}
	for _, name := range []string{"", "tools", "tools/nested"} {
		mustInput(t, os.Chmod(filepath.Join(directory, name), 0500))
	}
	replacement, err := openInputRoot(directory)
	mustInput(t, err)
	defer replacement.Close()
	if err = verifyInputTree(context.Background(), replacement, files); err != nil {
		t.Fatal("replacement contents must be identical", err)
	}
	if err = v.currentRoot(); err == nil {
		t.Fatal("same-byte root replacement adopted")
	}
	if _, err = v.Verify(context.Background()); err == nil {
		t.Fatal("old descriptor supplied current-path authority")
	}
	// Explicitly opening the new object does not rebind the old handle.
	fresh, err := OpenRuntimeInputVerifier(context.Background(), directory, inputSHA([]byte("{}")), strings.Repeat("b", 64))
	mustInput(t, err)
	defer fresh.Close()
	if err = fresh.currentRoot(); err != nil {
		t.Fatal(err)
	}
	if err = v.currentRoot(); err == nil {
		t.Fatal("fresh open rebound old handle")
	}
}

func TestRuntimeInputVerifierRefusesMissingAliasedOrChangedRoot(t *testing.T) {
	for _, kind := range []string{"missing", "symlink", "ancestor-symlink", "writable", "public"} {
		t.Run(kind, func(t *testing.T) {
			directory, _, v := inputVerifierFixture(t)
			switch kind {
			case "writable":
				mustInput(t, os.Chmod(directory, 0700))
			case "public":
				mustInput(t, os.Chmod(directory, 0555))
			case "missing", "symlink":
				retained := directory + "-retained"
				mustInput(t, os.Rename(directory, retained))
				if kind == "symlink" {
					mustInput(t, os.Symlink(retained, directory))
				}
			case "ancestor-symlink":
				mustInput(t, v.Close())
				parent := filepath.Join(filepath.Dir(directory), "held")
				mustInput(t, os.Mkdir(parent, 0700))
				inside := filepath.Join(parent, "bundle")
				// Moving a directory to a different parent updates its ".."
				// entry, so fixture setup needs write permission temporarily.
				mustInput(t, os.Chmod(directory, 0700))
				mustInput(t, os.Rename(directory, inside))
				mustInput(t, os.Chmod(inside, 0500))
				var err error
				v, err = OpenRuntimeInputVerifier(context.Background(), inside, inputSHA([]byte("{}")), strings.Repeat("b", 64))
				mustInput(t, err)
				defer v.Close()
				mustInput(t, os.Rename(parent, parent+"-retained"))
				mustInput(t, os.Symlink(parent+"-retained", parent))
			}
			if err := v.currentRoot(); err == nil {
				t.Fatal("changed root admitted")
			}
			if got, err := v.Verify(context.Background()); err == nil || got != (RuntimeInputInspection{}) {
				t.Fatal("changed root produced an inspection", got, err)
			}
			if kind == "writable" || kind == "public" {
				info, err := os.Stat(directory)
				mustInput(t, err)
				if info.Mode().Perm() == 0500 {
					t.Fatal("verifier repaired permissions")
				}
			}
		})
	}
}

func TestRuntimeInputVerifierAdmissionAndCancellation(t *testing.T) {
	directory, _, _ := inputVerifierFixture(t)
	for _, pair := range [][2]string{{"", strings.Repeat("b", 64)}, {strings.Repeat("a", 64), ""}, {strings.Repeat("A", 64), strings.Repeat("b", 64)}, {strings.Repeat("a", 64), strings.Repeat("0", 64)}} {
		if v, err := OpenRuntimeInputVerifier(context.Background(), directory, pair[0], pair[1]); err == nil {
			v.Close()
			t.Fatal("invalid review identifiers admitted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if v, err := OpenRuntimeInputVerifier(ctx, "/does-not-exist", strings.Repeat("a", 64), strings.Repeat("b", 64)); !errors.Is(err, context.Canceled) {
		if v != nil {
			v.Close()
		}
		t.Fatal("cancellation did not precede access", err)
	}
	_, _, v := inputVerifierFixture(t)
	if _, err := v.Verify(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("inspection cancellation lost", err)
	}
}

func TestRuntimeInputVerifierConcurrentClose(t *testing.T) {
	_, _, v := inputVerifierFixture(t)
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			if _, err := v.Verify(context.Background()); err == nil {
				t.Error("unreviewed manifest passed")
			}
		})
	}
	group.Go(func() { _ = v.Close() })
	group.Wait()
	if _, err := v.Verify(context.Background()); err == nil {
		t.Fatal("closed handle retained access")
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
}
