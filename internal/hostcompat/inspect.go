package hostcompat

import (
	"io"
	"io/fs"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"
)

var requiredRoles = []struct{ role, executable string }{
	{"engine", "dockerd"}, {"cli", "docker"}, {"containerd", "containerd"},
	{"oci-runtime", "runc"}, {"rootlesskit", "rootlesskit"}, {"network", "slirp4netns"},
	{"uidmap", "newuidmap"}, {"gidmap", "newgidmap"},
}

type nativePackage struct {
	name, version, arch, assertion string
	files                          map[string]bool
	metadata                       []string
}

type inspection struct {
	report Report
	owners map[string]nativePackage
}

// Inspect reads only bounded native metadata and previously authenticated local
// receipts. It never executes a program, downloads evidence or writes files.
func Inspect(fsys fs.FS, goos, goarch string) Report { return inspect(fsys, goos, goarch).report }

func InspectCurrent() Report {
	root, err := os.OpenRoot("/")
	if err != nil {
		return Report{SchemaVersion: 1, PolicyID: PolicyID, Platform: runtime.GOOS + "/" + runtime.GOARCH, Compatibility: "unknown", Qualification: "unqualified", Provenance: "unknown", ObservedAt: time.Now().UTC()}
	}
	defer root.Close()
	return inspectCurrentFS(currentEvidenceFS(root.FS()), runtime.GOOS, runtime.GOARCH, &currentFileDigests)
}

func inspectCurrentFS(fsys fs.FS, goos, goarch string, cache *fileDigestCache) Report {
	fsys, _ = withWork(fsys, inspectionLimits, cache)
	return inspect(fsys, goos, goarch).report
}

func inspect(fsys fs.FS, goos, goarch string) inspection {
	fsys, work := withWork(fsys, inspectionLimits, nil)
	r := Report{SchemaVersion: 1, PolicyID: PolicyID, Platform: goos + "/" + goarch, Compatibility: "unknown", Qualification: "unqualified", Provenance: "unknown", ObservedAt: time.Now().UTC()}
	result := inspection{owners: make(map[string]nativePackage)}
	r.Distribution = detectDistribution(fsys)
	if r.Distribution.Provider == "deb" {
		if _, _, err := aptSources(fsys); err != nil {
			r.Checks = append(r.Checks, Check{"packages.apt-authority", "unknown", "APT authority is unresolved: effective source-selection overrides, unsupported layouts, inline keys, fingerprints and insecure bypasses are not authenticated; use supported native source files and per-source Signed-By keyrings."})
		}
	}
	var packages []nativePackage
	var err error
	switch r.Distribution.Provider {
	case "pacman":
		packages, err = pacmanPackages(fsys)
	case "deb":
		packages, err = debPackages(fsys)
	default:
		err = errEvidence
	}
	if err != nil {
		r.Checks = append(r.Checks, Check{"packages.database", "unknown", "Trusted supported native package database is unavailable or malformed."})
	}
	for _, role := range requiredRoles {
		p := Package{Role: role.role, Provenance: "unknown", Detail: "No trusted native package owns the required fixed executable."}
		var resolved string
		for _, dir := range []string{"usr/bin", "usr/sbin", "bin", "sbin"} {
			candidate, info, e := resolve(fsys, path.Join(dir, role.executable))
			if e == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 && (strings.HasPrefix(candidate, "usr/bin/") || strings.HasPrefix(candidate, "usr/sbin/") || strings.HasPrefix(candidate, "usr/lib/")) {
				resolved = candidate
				break
			}
		}
		if resolved != "" {
			var owners []nativePackage
			for _, pkg := range packages {
				if pkg.files[resolved] {
					owners = append(owners, pkg)
				}
			}
			if len(owners) == 1 {
				owner := owners[0]
				p.Name, p.Version, p.UpstreamVersion = owner.name, owner.version, upstreamVersion(owner.version)
				p.Executables = []string{"/" + resolved}
				p.Provenance, p.Detail = "metadata-only", "Native ownership and version metadata; publisher and installed payload are not authenticated."
				if owner.assertion == "pgp" {
					p.Detail = "Pacman package-manager assertion: recorded PGP validation; installed payload has not been authenticated."
				}
				result.owners[role.role] = owner
			}
		}
		if role.role == "engine" {
			r.InstalledEngineVersion = p.UpstreamVersion
		}
		r.Packages = append(r.Packages, p)
	}
	applyReceipts(fsys, &r, result.owners)
	if work.exhausted {
		r.Checks = append(r.Checks, Check{"packages.budget", "unknown", "Host evidence exceeded the aggregate metadata, hashing or entry budget; unresolved evidence does not admit new work."})
		for i := range r.Packages {
			if r.Packages[i].Provenance == "verified" {
				r.Packages[i].Provenance = "unknown"
			}
		}
	}
	r = Evaluate(r, "")
	r.Guidance = []string{"Use native OS packages for the rootless Docker stack. Refresh local authenticated package evidence explicitly as root; cached verification does not require network access.", "A supported Docker version is compatibility evidence, not qualification of a new host distribution."}
	result.report = r
	return result
}

func detectDistribution(fsys fs.FS) Distribution {
	d := Distribution{Provider: "unknown", Profile: "unqualified"}
	b, err := readTrusted(fsys, "etc/os-release", 64<<10)
	if err != nil {
		b, err = readTrusted(fsys, "usr/lib/os-release", 64<<10)
	}
	if err == nil {
		values, e := parseOSRelease(string(b))
		if e == nil {
			d.ID, d.VersionID = values["ID"], values["VERSION_ID"]
			d.IDLike = strings.Fields(values["ID_LIKE"])
		}
	}
	_, pi, pe := resolve(fsys, "var/lib/pacman/local")
	_, di, de := resolve(fsys, "var/lib/dpkg/status")
	if pe == nil && pi.IsDir() && de != nil {
		d.Provider = "pacman"
	}
	if de == nil && di.Mode().IsRegular() && pe != nil {
		d.Provider = "deb"
	}
	return d
}

func parseOSRelease(s string) (map[string]string, error) {
	m := make(map[string]string)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, errEvidence
		}
		if k != "ID" && k != "VERSION_ID" && k != "ID_LIKE" {
			continue
		}
		if _, exists := m[k]; exists {
			return nil, errEvidence
		}
		if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		for _, t := range strings.Fields(v) {
			if !token(t) {
				return nil, errEvidence
			}
		}
		if len(v) > 256 || (k != "ID_LIKE" && strings.ContainsAny(v, " \t")) {
			return nil, errEvidence
		}
		m[k] = v
	}
	return m, nil
}

func upstreamVersion(version string) string {
	v := version
	if before, after, ok := strings.Cut(v, ":"); ok {
		for _, c := range before {
			if c < '0' || c > '9' {
				return ""
			}
		}
		if before == "" {
			return ""
		}
		v = after
	}
	// Native package releases follow the final dash. Standard Debian/Arch
	// revisions begin with a digit; leave a SemVer build identifier containing
	// dashes intact. A prerelease in the upstream portion remains unsupported.
	if dash := strings.LastIndexByte(v, '-'); dash >= 0 {
		revision := v[dash+1:]
		if revision != "" && revision[0] >= '0' && revision[0] <= '9' {
			v = v[:dash]
		} else if !strings.Contains(v, "+") {
			return ""
		}
	}
	core, n, ok := stableEngineVersion(v)
	if !ok || n[0] == 0 {
		return ""
	}
	return core
}

func parsePacman(s string) (map[string][]string, error) {
	m := make(map[string][]string)
	key := ""
	for _, line := range strings.Split(s, "\n") {
		if len(line) > 4096 {
			return nil, errEvidence
		}
		if line == "" {
			key = ""
			continue
		}
		if strings.HasPrefix(line, "%") && strings.HasSuffix(line, "%") {
			key = strings.Trim(line, "%")
			if !token(key) {
				return nil, errEvidence
			}
			if _, exists := m[key]; exists {
				return nil, errEvidence
			}
			m[key] = []string{}
		} else if key != "" {
			m[key] = append(m[key], line)
		} else {
			return nil, errEvidence
		}
	}
	return m, nil
}

func singleton(m map[string][]string, key string) string {
	if len(m[key]) != 1 {
		return ""
	}
	return m[key][0]
}

func pacmanPackages(fsys fs.FS) ([]nativePackage, error) {
	wanted := criticalPaths(fsys)
	entries, err := readDir(fsys, "var/lib/pacman/local")
	if err != nil {
		return nil, err
	}
	var result []nativePackage
	total := 0
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		if chargeCount(fsys, "record", 1) != nil {
			return nil, errBudget
		}
		base := "var/lib/pacman/local/" + ent.Name()
		b, e := readTrusted(fsys, base+"/desc", 256<<10)
		if e != nil {
			return nil, errEvidence
		}
		total += len(b)
		if total > maxMetadataTotal {
			return nil, errEvidence
		}
		m, e := parsePacman(string(b))
		if e != nil {
			return nil, e
		}
		p := nativePackage{name: singleton(m, "NAME"), version: singleton(m, "VERSION"), arch: singleton(m, "ARCH"), assertion: singleton(m, "VALIDATION"), metadata: []string{base + "/desc", base + "/files"}, files: make(map[string]bool)}
		if !token(p.name) || !token(p.version) {
			return nil, errEvidence
		}
		b, e = readTrusted(fsys, base+"/files", 8<<20)
		if e != nil {
			return nil, e
		}
		total += len(b)
		if total > maxMetadataTotal {
			return nil, errEvidence
		}
		m, e = parsePacman(string(b))
		if e != nil {
			return nil, e
		}
		for _, f := range m["FILES"] {
			if wanted[f] {
				p.files[f] = true
			}
		}
		if len(p.files) > 0 {
			result = append(result, p)
		}
	}
	return result, nil
}

// parseControl handles Debian RFC822-style control data, including continuation
// lines, without executing shell-like fields or accepting duplicate keys.
func parseControl(s string) ([]map[string]string, error) {
	var result []map[string]string
	m := make(map[string]string)
	key := ""
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if len(line) > 65536 {
			return nil, errEvidence
		}
		if line == "" {
			if len(m) > 0 {
				result = append(result, m)
				if len(result) > 131072 {
					return nil, errEvidence
				}
				m = make(map[string]string)
			}
			key = ""
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			if key == "" || len(m[key])+len(line) > 1<<20 {
				return nil, errEvidence
			}
			m[key] += "\n" + strings.TrimSpace(line)
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok || !token(k) {
			return nil, errEvidence
		}
		if _, exists := m[k]; exists {
			return nil, errEvidence
		}
		key, m[k] = k, strings.TrimSpace(v)
	}
	if len(m) > 0 {
		result = append(result, m)
	}
	return result, nil
}

func debPackages(fsys fs.FS) ([]nativePackage, error) {
	wanted := criticalPaths(fsys)
	f, err := openTrusted(fsys, "var/lib/dpkg/status", maxMetadata)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, errEvidence
	}
	if chargeBytes(fsys, "metadata", before.Size()+1) != nil {
		return nil, errBudget
	}
	var result []nativePackage
	total := int(before.Size())
	err = scanControl(io.LimitReader(f, before.Size()+1), 32768, packageStatusFields, func(m map[string]string) error {
		if chargeCount(fsys, "record", 1) != nil {
			return errBudget
		}
		if m["Status"] != "install ok installed" {
			return nil
		}
		p := nativePackage{name: m["Package"], version: m["Version"], arch: m["Architecture"], files: make(map[string]bool)}
		if !token(p.name) || !token(p.version) || !token(p.arch) {
			return errEvidence
		}
		base := "var/lib/dpkg/info/" + p.name
		b, e := readTrusted(fsys, base+".list", 8<<20)
		if e != nil {
			base += ":" + p.arch
			b, e = readTrusted(fsys, base+".list", 8<<20)
		}
		if e != nil {
			return nil
		}
		total += len(b)
		if total > maxMetadataTotal {
			return errEvidence
		}
		p.metadata = []string{"var/lib/dpkg/status", base + ".list"}
		for _, f := range strings.Split(string(b), "\n") {
			f = strings.TrimPrefix(f, "/")
			if wanted[f] {
				p.files[f] = true
			}
		}
		if len(p.files) > 0 {
			result = append(result, p)
		}
		return nil
	})
	if err != nil || !stableRead(fsys, "var/lib/dpkg/status", f, before) {
		return nil, errEvidence
	}
	sort.Slice(result, func(i, j int) bool { return result[i].name < result[j].name })
	return result, nil
}

func criticalPaths(fsys fs.FS) map[string]bool {
	wanted := map[string]bool{}
	for _, role := range requiredRoles {
		for _, dir := range []string{"usr/bin", "usr/sbin", "bin", "sbin"} {
			name, info, err := resolve(fsys, dir+"/"+role.executable)
			if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
				wanted[name] = true
			}
		}
	}
	return wanted
}
