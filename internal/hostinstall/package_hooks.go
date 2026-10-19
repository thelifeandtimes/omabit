package hostinstall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

// Hooks are published only after activation has installed the OS-authorized new
// executable. In particular, an old schema-compatible runner need not implement
// verify-host-packages, and must not be replaced early to make hooks available.
func (i *installer) installPackageHooks(ctx context.Context, binary []byte) error {
	if i.euid != 0 || ctx == nil || i.inspectCompatibility == nil {
		return errors.New("package hooks require OS authorization and current native evidence")
	}
	current, err := i.readInstalled(ctx, access.ShellPath, 128<<20)
	if err != nil || len(binary) == 0 || !bytes.Equal(current, binary) {
		return errors.New("package hooks require the activated, verified current runner generation")
	}
	info, err := i.lstat(i.path(access.ShellPath))
	if err != nil || info.Mode().Perm()&0111 == 0 {
		return errors.New("package hooks require an executable current runner")
	}
	provider := i.inspectCompatibility().Distribution.Provider
	files, err := rootless.PackageHookFiles(provider)
	if err != nil {
		return err
	}
	previous, err := rootless.PreviousPackageHookFiles(provider)
	if err != nil {
		return err
	}
	for _, name := range []string{"id", "stat", "env", "timeout"} {
		if err := i.helper(ctx, "/usr/bin/"+name); err != nil {
			return err
		}
	}
	if err := i.validatePackageHookConfiguration(ctx, provider, false); err != nil {
		return err
	}
	// Check every destination before changing any of them. Prefixes, symlinks,
	// broad permissions, exclusive links and sizes are checked by readInstalled.
	for path, data := range files {
		if err := i.reviewPackageHook(ctx, path, data, previous[path]); err != nil {
			return err
		}
	}
	path := rootless.PacmanPackageHookPath
	if provider == "deb" {
		path = rootless.APTPackageHookPath
	}
	for _, target := range []string{rootless.PackageRefreshPath, path} {
		mode := os.FileMode(0644)
		if target == rootless.PackageRefreshPath {
			mode = 0755
		}
		if err := i.installFile(ctx, target, files[target], mode, true); err != nil {
			return err
		}
	}
	return i.validatePackageHookConfiguration(ctx, provider, true)
}

func (i *installer) reviewPackageHook(ctx context.Context, path string, target []byte, previous [][]byte) error {
	old, err := i.readInstalled(ctx, path, maxOutput)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if bytes.Equal(old, target) {
		return nil
	}
	for _, candidate := range previous {
		if bytes.Equal(old, candidate) {
			return nil
		}
	}
	return fmt.Errorf("%s has local changes; preserving native package hook for review", path)
}

// Use fixed native tools for their own configuration syntax. Tool output is
// bounded data, never commands, executable paths or installation destinations.
func (i *installer) validatePackageHookConfiguration(ctx context.Context, provider string, installed bool) error {
	switch provider {
	case "pacman":
		out, err := i.command(ctx, "/usr/bin/pacman-conf", "HookDir")
		if err != nil {
			return errors.New("cannot establish native Pacman hook configuration")
		}
		found := false
		for _, dir := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			dir = filepath.Clean(strings.TrimSpace(dir))
			switch dir {
			case "/etc/pacman.d/hooks":
				found = true
			case "/usr/share/libalpm/hooks":
			default:
				return errors.New("custom Pacman HookDir needs reviewed transaction-hook integration; configuration was preserved")
			}
		}
		if !found {
			return errors.New("Pacman does not load the standard local hook directory; configuration was preserved")
		}
		cache, err := i.command(ctx, "/usr/bin/pacman-conf", "CacheDir")
		if err != nil {
			return errors.New("cannot establish native Pacman package cache configuration")
		}
		for _, dir := range strings.Split(strings.TrimSpace(string(cache)), "\n") {
			if filepath.Clean(strings.TrimSpace(dir)) != "/var/cache/pacman/pkg" {
				return errors.New("custom Pacman CacheDir needs native verifier support before automatic hook verification")
			}
		}
		for _, dir := range []string{"/usr/share/libalpm/hooks", "/etc/pacman.d/hooks"} {
			if err := i.reviewEarlierPacmanHooks(ctx, dir); err != nil {
				return err
			}
		}
		return nil
	case "deb":
		out, err := i.command(ctx, "/usr/bin/apt-config", "dump")
		if err != nil {
			return errors.New("cannot establish native APT hook configuration")
		}
		return validateAPTConfiguration(out, installed)
	default:
		return errors.New("native transaction hooks are unavailable for this provider")
	}
}

func validateAPTConfiguration(data []byte, installed bool) error {
	values := map[string]string{}
	var post []string
	for _, line := range strings.Split(string(data), "\n") {
		key, raw, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		key = strings.ToLower(key) // APT configuration names are case-insensitive.
		// Native RunScripts executes nonempty values of every direct child in
		// tree order, including named children; grandchildren are not invoked.
		child, postChild := strings.CutPrefix(key, "dpkg::post-invoke::")
		postChild = postChild && !strings.Contains(child, "::")
		if key != "dir" && key != "rootdir" && key != "dir::etc" && key != "dir::etc::parts" && key != "dir::cache" && key != "dir::cache::archives" && !postChild {
			continue
		}
		value, err := strconv.Unquote(strings.TrimSuffix(strings.TrimSpace(raw), ";"))
		if err != nil {
			return errors.New("native APT hook configuration is malformed")
		}
		if postChild {
			if value != "" {
				post = append(post, value)
			}
		} else {
			if _, exists := values[key]; exists {
				return errors.New("native APT directory configuration is ambiguous")
			}
			values[key] = value
		}
	}
	if values["rootdir"] != "" && values["rootdir"] != "/" {
		return errors.New("alternate APT root needs reviewed transaction-hook integration")
	}
	root, etc, parts := values["dir"], values["dir::etc"], values["dir::etc::parts"]
	if root != "/" || etc == "" || parts == "" {
		return errors.New("native APT configuration directories are unresolved")
	}
	if !filepath.IsAbs(etc) {
		etc = filepath.Join(root, etc)
	}
	if !filepath.IsAbs(parts) {
		parts = filepath.Join(etc, parts)
	}
	if filepath.Clean(parts) != "/etc/apt/apt.conf.d" {
		return errors.New("custom APT fragment directory needs reviewed transaction-hook integration; configuration was preserved")
	}
	cache, archives := values["dir::cache"], values["dir::cache::archives"]
	if cache == "" || archives == "" {
		return errors.New("native APT archive cache configuration is unresolved")
	}
	if !filepath.IsAbs(cache) {
		cache = filepath.Join(root, cache)
	}
	if !filepath.IsAbs(archives) {
		archives = filepath.Join(cache, archives)
	}
	if filepath.Clean(archives) != "/var/cache/apt/archives" {
		return errors.New("custom APT archive cache needs native verifier support before automatic hook verification")
	}
	if installed && (len(post) == 0 || post[0] != rootless.PackageRefreshPath+" || true") {
		return errors.New("APT does not run the installed refresh first, before possible cache cleanup; preserve hooks and review native ordering")
	}
	return nil
}

func (i *installer) reviewEarlierPacmanHooks(ctx context.Context, dir string) error {
	if err := i.parents(ctx, dir+"/hook", false); err != nil {
		return err
	}
	file, err := os.OpenFile(i.path(dir), os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.New("native hook directory is unavailable")
	}
	defer file.Close()
	entries, err := file.ReadDir(1025)
	if err != nil && !errors.Is(err, io.EOF) || len(entries) > 1024 {
		return errors.New("native hook directory exceeds inspection bounds")
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".hook") || entry.Name() >= filepath.Base(rootless.PacmanPackageHookPath) {
			continue
		}
		data, err := i.readInstalled(ctx, dir+"/"+entry.Name(), 64<<10)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(data), "\n") {
			key, value, _ := strings.Cut(line, "=")
			if strings.TrimSpace(key) == "When" && strings.TrimSpace(value) == "PostTransaction" {
				return errors.New("an earlier Pacman post-transaction hook may remove authentication evidence; review native ordering before installing Omabit hooks")
			}
		}
	}
	return nil
}
