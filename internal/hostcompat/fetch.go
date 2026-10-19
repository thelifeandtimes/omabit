package hostcompat

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"runtime"
	"sort"
	"strings"
)

var errFetch = errors.New("matching installed native package download is unavailable or failed validation")
var errAPTSandboxUnavailable = errors.New("native non-root _apt download account is unavailable or unsafe")

type fetchCommand struct {
	Tool      string
	Args      []string
	Directory string
	APTConfig string
	MaxFile   int64
}
type fetchRunner func(context.Context, fetchCommand) ([]byte, error)

type nativeFetcher struct {
	v          *verifier
	stage      string // fs-relative, application-owned, private, unique per refresh
	run        fetchRunner
	write      func(string, []byte) error
	mkdir      func(string) error
	attempted  map[string]error
	aptReady   bool
	aptArch    string
	prepareAPT func(string) error
	sealAPT    func(context.Context, string, nativePackage, bool) (string, error)
	lastError  error
}

func (f *nativeFetcher) fetch(ctx context.Context, provider string, p nativePackage) error {
	if ctx.Err() != nil || !fetchPackageIdentity(p) {
		return errFetch
	}
	if f.attempted == nil {
		f.attempted = map[string]error{}
	}
	key := provider + "/" + p.name + "/" + p.version + "/" + p.arch
	if err, ok := f.attempted[key]; ok {
		return err
	}
	f.attempted[key] = errFetch
	var err error
	switch provider {
	case "deb":
		err = f.fetchAPT(ctx, p)
	case "pacman":
		err = f.fetchPacman(ctx, p)
	default:
		err = errFetch
	}
	if err == errAPTSandboxUnavailable {
		f.lastError = err
		f.attempted[key] = err
		return err
	}
	if err != nil || ctx.Err() != nil {
		f.lastError = errFetch
		return errFetch
	}
	f.attempted[key] = nil
	return nil
}

func fetchPackageIdentity(p nativePackage) bool {
	if p.name == "" || len(p.name) > 128 || !token(p.version) || !token(p.arch) {
		return false
	}
	for _, c := range p.name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("+._-", c)) {
			return false
		}
	}
	if !(p.name[0] >= 'a' && p.name[0] <= 'z' || p.name[0] >= '0' && p.name[0] <= '9') {
		return false
	}
	for _, c := range p.arch {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func (f *nativeFetcher) command(ctx context.Context, tool string, args []string, dir, config string, maxFile int64) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, errFetch
	}
	b, err := f.run(ctx, fetchCommand{tool, args, "/" + dir, config, maxFile})
	if err != nil || ctx.Err() != nil {
		return nil, errFetch
	}
	return b, nil
}

func aptPrivateConfig(stage, architecture string) string {
	base := "/" + stage
	// APT_CONFIG is loaded before apt.conf.d/main. Pointing both at private,
	// empty locations excludes global post-update hooks and insecure overrides.
	return fmt.Sprintf(`Dir::Etc::main "%s/no-global-config";
Dir::Etc::parts "%s/empty-config";
Dir::Etc::sourcelist "%s/sources.sources";
Dir::Etc::sourceparts "%s/empty-sources";
Dir::Etc::trusted "%s/no-global-keyring.gpg";
Dir::Etc::trustedparts "%s/empty-trusted";
Dir::Etc::netrc "/etc/apt/auth.conf";
Dir::Etc::netrcparts "/etc/apt/auth.conf.d";
Dir::State "%s/state";
Dir::State::status "/var/lib/dpkg/status";
Dir::State::lists "%s/lists";
Dir::State::extended_states "%s/state/extended_states";
Dir::Cache "%s/cache";
Dir::Cache::archives "%s/archives";
Dir::Cache::pkgcache "";
Dir::Cache::srcpkgcache "";
Dir::Log "%s/log";
APT::Architecture "%s";
APT::Architectures { "%s"; };
APT::Sandbox::User "_apt";
APT::Get::AllowUnauthenticated "false";
APT::Update::Error-Mode "any";
Acquire::AllowInsecureRepositories "false";
Acquire::AllowDowngradeToInsecureRepositories "false";
Acquire::AllowWeakRepositories "false";
Acquire::Check-Date "true";
Acquire::Check-Valid-Until "true";
Acquire::Languages "none";
Acquire::Queue-Mode "access";
Acquire::PDiffs "false";
Acquire::Retries "0";
Acquire::http::Timeout "20";
Acquire::https::Timeout "20";
Acquire::https::Verify-Peer "true";
Acquire::https::Verify-Host "true";
`, base, base, base, base, base, base, base, base, base, base, base, base, architecture, architecture)
}

func (f *nativeFetcher) fetchAPT(ctx context.Context, p nativePackage) error {
	sources, _, err := aptSources(f.v.fsys)
	if err != nil {
		return errFetch
	}
	arch := p.arch
	if arch == "all" {
		arch = runtime.GOARCH
		if arch == "386" {
			arch = "i386"
		}
	}
	base := f.stage + "/apt"
	config := "/" + base + "/apt.conf"
	if !f.aptReady {
		for _, dir := range []string{base, base + "/state", base + "/cache", base + "/archives", base + "/lists", base + "/lists/partial", base + "/log", base + "/home", base + "/tmp", base + "/empty-config", base + "/empty-sources", base + "/empty-trusted"} {
			if f.mkdir(dir) != nil {
				return errFetch
			}
		}
		var text strings.Builder
		for _, source := range sources {
			if len(source.Architectures) > 0 && !containsString(source.Architectures, arch) {
				continue
			}
			var keys []string
			for _, key := range source.Keyrings {
				if _, err := readTrusted(f.v.fsys, key, 8<<20); err != nil {
					return errFetch
				}
				keys = append(keys, "/"+key)
			}
			// Leave Trusted unset: APT then derives trust from Signed-By and
			// authenticated indexes. Trusted:no would force even valid indexes
			// to remain untrusted; Trusted:yes would bypass authentication.
			fmt.Fprintf(&text, "Types: deb\nURIs: %s\nSuites: %s\nComponents: %s\nArchitectures: %s\nSigned-By: %s\n\n", source.URI, source.Suite, strings.Join(source.Components, " "), arch, strings.Join(keys, " "))
		}
		if text.Len() == 0 || f.write(base+"/sources.sources", []byte(text.String())) != nil || f.write(base+"/apt.conf", []byte(aptPrivateConfig(base, arch))) != nil {
			return errFetch
		}
		if f.prepareAPT != nil {
			if err := f.prepareAPT(base); err != nil {
				return err
			}
		}
		if _, err := f.command(ctx, "/usr/bin/apt-get", []string{"-qq", "update"}, base, config, maxIndex); err != nil {
			return errFetch
		}
		if f.sealAPT != nil {
			sealed, err := f.sealAPT(ctx, base, p, true)
			if err != nil {
				return errFetch
			}
			f.v.aptListsDir = sealed + "/lists"
		}
		f.aptReady = true
		f.aptArch = arch
	}
	if arch != f.aptArch {
		return errFetch
	}
	// Native download only: the exact version includes its epoch and revision.
	if _, err := f.command(ctx, "/usr/bin/apt-get", []string{"-qq", "download", "--", p.name + ":" + p.arch + "=" + p.version}, base+"/archives", config, maxArtifact); err != nil {
		return errFetch
	}
	if f.sealAPT != nil {
		sealed, err := f.sealAPT(ctx, base, p, false)
		if err != nil {
			return errFetch
		}
		f.v.aptArchivesDir = sealed + "/archives"
	} else {
		// Injected adapters already supply trusted fixture files. Production
		// always installs the non-root acquisition/sealing callbacks.
		f.v.aptArchivesDir = base + "/archives"
		f.v.aptListsDir = base + "/lists"
	}
	return nil
}

// Read the same root-owned configuration closure that pacman-conf will use.
// Includes are data only; pacman-conf never executes XferCommand or hooks.
func pacmanConfigEvidence(fsys fs.FS) ([]digest, error) {
	var paths []string
	seen := map[string]bool{}
	active := map[string]bool{}
	total := 0
	var visit func(string, int) error
	visit = func(name string, depth int) error {
		if depth > 8 || active[name] {
			return errEvidence
		}
		if seen[name] {
			return nil
		}
		if len(seen) >= 128 {
			return errEvidence
		}
		active[name] = true
		defer delete(active, name)
		seen[name] = true
		b, err := readTrusted(fsys, name, 256<<10)
		if err != nil {
			return err
		}
		total += len(b)
		if total > 1<<20 {
			return errEvidence
		}
		paths = append(paths, name)
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
			key, val, ok := strings.Cut(line, "=")
			if !ok || strings.TrimSpace(key) != "Include" {
				continue
			}
			val = strings.TrimSpace(val)
			if !strings.HasPrefix(val, "/") || strings.Contains(val, "..") || len(val) > 4096 {
				return errEvidence
			}
			dir, pattern := strings.TrimPrefix(path.Dir(val), "/"), path.Base(val)
			if strings.ContainsAny(dir, "*?[") {
				return errEvidence
			}
			var matches []string
			if strings.ContainsAny(pattern, "*?[") {
				entries, err := readDir(fsys, dir)
				if err != nil {
					return err
				}
				for _, entry := range entries {
					ok, err := path.Match(pattern, entry.Name())
					if err != nil {
						return errEvidence
					}
					if ok {
						matches = append(matches, dir+"/"+entry.Name())
						if len(matches) > 128 {
							return errEvidence
						}
					}
				}
			} else {
				matches = []string{strings.TrimPrefix(val, "/")}
			}
			if len(matches) == 0 {
				return errEvidence
			}
			for _, match := range matches {
				if err := visit(match, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit("etc/pacman.conf", 0); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return digestFiles(fsys, paths, 256<<10)
}

func securePacmanSigLevel(b []byte) bool {
	required, trusted := false, false
	for _, word := range strings.Fields(string(b)) {
		switch word {
		case "Required", "PackageRequired":
			required = true
		case "TrustedOnly", "PackageTrustedOnly":
			trusted = true
		case "DatabaseRequired", "DatabaseOptional", "DatabaseNever", "DatabaseTrustedOnly":
		default:
			return false
		}
	}
	return required && trusted
}

func safeFetchURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(raw) > 4096 {
		return false
	}
	if strings.ContainsAny(raw, "\r\n\x00\t ") || strings.Contains(u.Path, "..") {
		return false
	}
	return true
}

func (f *nativeFetcher) curl(ctx context.Context, uri, destination string, limit int64) error {
	if !safeFetchURL(uri) || !fs.ValidPath(destination) || !strings.HasPrefix(destination, f.stage+"/") {
		return errFetch
	}
	args := []string{"--disable", "--fail", "--silent", "--show-error", "--location", "--proto", "=http,https", "--proto-redir", "=http,https", "--max-redirs", "3", "--connect-timeout", "10", "--max-time", "60", "--max-filesize", fmt.Sprint(limit), "--output", "/" + destination, "--", uri}
	_, err := f.command(ctx, "/usr/bin/curl", args, f.stage, "", limit)
	if err != nil {
		return errFetch
	}
	if _, err := hashTrusted(f.v.fsys, destination, limit); err != nil {
		return errFetch
	}
	return nil
}

func (f *nativeFetcher) fetchPacman(ctx context.Context, p nativePackage) error {
	config, err := pacmanConfigEvidence(f.v.fsys)
	if err != nil {
		return errFetch
	}
	base := f.stage + "/pacman"
	for _, dir := range []string{base, base + "/sync", base + "/archives"} {
		if f.mkdir(dir) != nil {
			return errFetch
		}
	}
	b, err := f.command(ctx, "/usr/bin/pacman-conf", []string{"--config", "/etc/pacman.conf", "--repo-list"}, f.stage, "", 0)
	if err != nil {
		return errFetch
	}
	repos := strings.Fields(string(b))
	if len(repos) == 0 || len(repos) > 64 {
		return errFetch
	}
	for _, repo := range repos {
		if !token(repo) || strings.ContainsAny(repo, "+:~") {
			return errFetch
		}
		level, err := f.command(ctx, "/usr/bin/pacman-conf", []string{"--config", "/etc/pacman.conf", "--repo", repo, "SigLevel"}, f.stage, "", 0)
		if err == nil && len(strings.Fields(string(level))) == 0 {
			level, err = f.command(ctx, "/usr/bin/pacman-conf", []string{"--config", "/etc/pacman.conf", "SigLevel"}, f.stage, "", 0)
		}
		if err != nil || !securePacmanSigLevel(level) {
			return errFetch
		}
		servers, err := f.command(ctx, "/usr/bin/pacman-conf", []string{"--config", "/etc/pacman.conf", "--repo", repo, "Server"}, f.stage, "", 0)
		if err != nil {
			return errFetch
		}
		mirrors := strings.Fields(string(servers))
		if len(mirrors) > 64 {
			return errFetch
		}
		for n, server := range mirrors {
			if n >= 3 {
				break
			}
			if !safeFetchURL(server) {
				return errFetch
			}
			db := base + "/sync/" + repo + ".db"
			if f.curl(ctx, strings.TrimSuffix(server, "/")+"/"+repo+".db", db, maxIndex) != nil {
				continue
			}
			records, err := f.v.pacmanRepository(ctx, db, p)
			if err != nil {
				continue
			}
			for _, record := range records {
				if record.Size <= 0 || record.Size > maxArtifact || len(record.SHA256) != 64 || record.PGPSig == "" {
					continue
				}
				if _, err := hex.DecodeString(record.SHA256); err != nil {
					continue
				}
				if path.Base(record.Filename) != record.Filename || !pacmanArchiveName(record.Filename, p) {
					continue
				}
				artifact := base + "/archives/" + record.Filename
				uri := strings.TrimSuffix(server, "/") + "/" + url.PathEscape(record.Filename)
				if f.curl(ctx, uri, artifact, maxArtifact) != nil {
					continue
				}
				h, err := hashTrusted(f.v.fsys, artifact, maxArtifact)
				_, info, e := resolve(f.v.fsys, artifact)
				if err != nil || e != nil || h != record.SHA256 || info.Size() != record.Size {
					continue
				}
				if !verifyDigests(f.v.fsys, config, 256<<10) {
					return errFetch
				}
				f.v.pacmanCacheDirs = []string{base + "/archives"}
				f.v.pacmanDBDirs = []string{base + "/sync"}
				return nil
			}
		}
	}
	return errFetch
}
