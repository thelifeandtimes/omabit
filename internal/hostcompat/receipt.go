package hostcompat

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"sort"
	"strings"
	"time"
)

// VerificationDirectory contains only root-issued host authentication evidence.
// Native post-transaction hooks call VerifyCurrent; readers never run hooks.
const VerificationDirectory = "/var/lib/omabit/hostcompat"
const VerificationReceiptPath = VerificationDirectory + "/receipts-v2.json"
const VerificationCacheDirectory = VerificationDirectory + "/evidence-v2"

// ReceiptValidity is retained as a diagnostic-age threshold for callers of the
// initial API. It is NOT an authorization expiry or a required refresh interval.
const ReceiptValidity = 30 * 24 * time.Hour
const receiptPath = "var/lib/omabit/hostcompat/receipts-v2.json"
const pacmanMethod = "pacman-keyring-gpg-trusted-signature+sha256-installed-payload"
const aptMethod = "apt-archive-keyring-inrelease+sha256-index+sha256-deb+sha256-installed-payload"

type digest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type receipt struct {
	Role     string   `json:"role"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Method   string   `json:"method"`
	Files    []digest `json:"files"`
	Metadata []digest `json:"metadata"`
	Evidence []digest `json:"evidence"`
	// Evidence/Metadata are historical audit hashes, not live-file requirements.
	Retained        []digest `json:"retained"`
	PackageIdentity string   `json:"package_identity"`
	SourceID        string   `json:"source_id,omitempty"`
	Authorized      bool     `json:"authorized"`
}

type receiptSet struct {
	SchemaVersion int       `json:"schema_version"`
	PolicyID      string    `json:"policy_id"`
	Provider      string    `json:"provider"`
	Trust         []digest  `json:"trust"`
	Packages      []receipt `json:"packages"`
	PublicTrust   []digest  `json:"public_trust"`
	SourcePolicy  string    `json:"source_policy"`
	VerifiedAt    time.Time `json:"verified_at"`
	ValidUntil    time.Time `json:"valid_until,omitempty"` // legacy diagnostic field, never an expiry gate
}

func decodeStrict(b []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return errEvidence
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errEvidence
	}
	return nil
}

func digestFiles(fsys fs.FS, names []string, limit int64) ([]digest, error) {
	result := make([]digest, 0, len(names))
	for _, name := range names {
		h, err := hashTrusted(fsys, name, limit)
		if err != nil {
			return nil, err
		}
		result = append(result, digest{name, h})
	}
	return result, nil
}

func sameDigests(a, b []digest) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func verifyDigests(fsys fs.FS, records []digest, limit int64) bool {
	return newDigestChecker(fsys).matches(records, limit)
}

type digestKey struct {
	path  string
	limit int64
}
type digestChecker struct {
	fsys    fs.FS
	checked map[digestKey]checkedDigest
}

type checkedDigest struct {
	hash  string
	state fs.FileInfo
}

func newDigestChecker(fsys fs.FS) *digestChecker {
	return &digestChecker{fsys, make(map[digestKey]checkedDigest)}
}

// Within one read-only inspection, shared package artifacts/databases are hashed
// once even when several required executable roles belong to the same package.
func (c *digestChecker) matches(records []digest, limit int64) bool {
	if len(records) == 0 || len(records) > 32 {
		return false
	}
	seen := make(map[string]bool)
	for _, record := range records {
		if !fs.ValidPath(record.Path) || len(record.SHA256) != 64 || seen[record.Path] {
			return false
		}
		seen[record.Path] = true
		key := digestKey{record.Path, limit}
		cached, seen := c.checked[key]
		if seen {
			_, current, err := resolve(c.fsys, record.Path)
			seen = err == nil && sameFileState(cached.state, current)
		}
		if !seen {
			h, state, _ := hashTrustedState(c.fsys, record.Path, limit)
			cached = checkedDigest{h, state}
			c.checked[key] = cached
		}
		if cached.hash == "" || cached.hash != record.SHA256 {
			return false
		}
	}
	return true
}

// trustSnapshot is root-refresh-only race/audit evidence. Runtime readers use
// publicTrustSnapshot; authorization invalidation is evidence-based, not age-based.
func trustSnapshot(fsys fs.FS, provider string) ([]digest, error) {
	var names []string
	switch provider {
	case "pacman":
		entries, err := readDir(fsys, "etc/pacman.d/gnupg")
		if err != nil || len(entries) > 128 {
			return nil, errEvidence
		}
		hasPublic, hasTrust := false, false
		for _, e := range entries {
			// Secret keys, agents, lock files, random seeds and TOFU state are
			// not inputs to the fixed public-key PGP trust validation command.
			name := e.Name()
			if e.IsDir() || name == "secring.gpg" || !(strings.HasSuffix(name, ".gpg") || strings.HasSuffix(name, ".kbx") || name == "gpg.conf" || name == "common.conf") {
				continue
			}
			names = append(names, "etc/pacman.d/gnupg/"+e.Name())
			if name == "pubring.gpg" || name == "pubring.kbx" {
				hasPublic = true
			}
			if name == "trustdb.gpg" {
				hasTrust = true
			}
		}
		if !hasPublic || !hasTrust {
			return nil, errEvidence
		}
		names = append(names, "usr/bin/gpg", "usr/bin/bsdtar")
	case "deb":
		sources, paths, err := aptSources(fsys)
		if err != nil {
			return nil, err
		}
		configuration, err := aptConfigurationInputs(fsys)
		if err != nil {
			return nil, err
		}
		names = append(names, configuration...)
		names = append(names, paths...)
		for _, source := range sources {
			names = append(names, source.Keyrings...)
		}
		names = append(names, "usr/bin/gpgv", "usr/bin/dpkg-deb", "usr/lib/apt/apt-helper")
		for _, name := range names {
			if strings.HasSuffix(name, ".asc") {
				names = append(names, "usr/bin/gpg")
				break
			}
		}
	default:
		return nil, errEvidence
	}
	sort.Strings(names)
	names = uniqueStrings(names)
	return digestFiles(fsys, names, maxBinary)
}

func uniqueStrings(names []string) []string {
	result := []string{}
	for _, name := range names {
		if len(result) == 0 || result[len(result)-1] != name {
			result = append(result, name)
		}
	}
	return result
}

// publicTrustSnapshot deliberately excludes private Pacman trust databases,
// transient APT lists and OS archive caches. Root refresh validates private
// trust/revocation state and mints an evidence-bound public authorization.
// Native keyring package revocations and APT authority changes invalidate it
// immediately; direct private trust edits must invoke the same root refresh hook.
func publicTrustSnapshot(fsys fs.FS, provider string) ([]digest, string, error) {
	var names []string
	policy := ""
	switch provider {
	case "pacman":
		entries, err := readDir(fsys, "usr/share/pacman/keyrings")
		if err != nil || len(entries) > 128 {
			return nil, "", errEvidence
		}
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasSuffix(name, ".gpg") || strings.HasSuffix(name, ".trusted") || strings.HasSuffix(name, ".revoked") || strings.HasSuffix(name, "-trusted") || strings.HasSuffix(name, "-revoked") {
				names = append(names, "usr/share/pacman/keyrings/"+name)
			}
		}
	case "deb":
		sources, sourceFiles, err := aptSources(fsys)
		if err != nil {
			return nil, "", err
		}
		b, _ := json.Marshal(struct {
			Model       string      `json:"model"`
			SourceList  string      `json:"source_list"`
			SourceParts string      `json:"source_parts"`
			Files       []string    `json:"files"`
			Sources     []aptSource `json:"sources"`
		}{aptAuthorityModel, "/etc/apt/sources.list", "/etc/apt/sources.list.d", sourceFiles, sources})
		h := sha256.Sum256(b)
		policy = hex.EncodeToString(h[:])
		for _, s := range sources {
			names = append(names, s.Keyrings...)
		}
	default:
		return nil, "", errEvidence
	}
	sort.Strings(names)
	names = uniqueStrings(names)
	if len(names) == 0 || len(names) > 32 {
		return nil, "", errEvidence
	}
	d, err := digestFiles(fsys, names, 8<<20)
	return d, policy, err
}

func packageIdentity(owner nativePackage, executables []string) string {
	b, _ := json.Marshal(struct {
		Name, Version, Architecture string
		Executables                 []string
	}{owner.name, owner.version, owner.arch, executables})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func loadReceipts(fsys fs.FS, provider string) (receiptSet, error) {
	b, err := readTrusted(fsys, receiptPath, 1<<20)
	if err != nil {
		return receiptSet{}, err
	}
	var set receiptSet
	if decodeStrict(b, &set) != nil || set.SchemaVersion != 2 || set.PolicyID != PolicyID || set.Provider != provider || len(set.Packages) > len(requiredRoles) || len(set.Trust) > 256 {
		return receiptSet{}, errEvidence
	}
	if !validReceiptTime(set, time.Now()) {
		return receiptSet{}, errEvidence
	}
	roles := make(map[string]bool)
	for _, rec := range set.Packages {
		if roles[rec.Role] || len(rec.Files) == 0 || len(rec.Files) > 8 || len(rec.Retained) > 8 || len(rec.Evidence) > 8 || len(rec.Metadata) > 8 {
			return receiptSet{}, errEvidence
		}
		roles[rec.Role] = true
	}
	return set, nil
}

func receiptMatches(p Package, owner nativePackage, rec receipt, provider string) bool {
	if rec.Role != p.Role || rec.Name != p.Name || rec.Version != p.Version || rec.PackageIdentity != packageIdentity(owner, p.Executables) {
		return false
	}
	if (provider == "pacman" && rec.Method != pacmanMethod) || (provider == "deb" && rec.Method != aptMethod) {
		return false
	}
	if len(rec.Files) != len(p.Executables) {
		return false
	}
	for i, exe := range p.Executables {
		if rec.Files[i].Path != strings.TrimPrefix(exe, "/") {
			return false
		}
	}
	return true
}

func applyReceipts(fsys fs.FS, report *Report, owners map[string]nativePackage) {
	set, err := loadReceipts(fsys, report.Distribution.Provider)
	if err != nil {
		return
	}
	now := time.Now()
	if !validReceiptTime(set, now) {
		report.Checks = append(report.Checks, Check{"packages.receipt", "unknown", "Root-issued package evidence has an invalid or future verification timestamp; refresh local evidence as root."})
		return
	}
	trust, policy, err := publicTrustSnapshot(fsys, set.Provider)
	if err != nil || !sameDigests(trust, set.PublicTrust) || policy != set.SourcePolicy {
		report.Checks = append(report.Checks, Check{"packages.receipt", "unknown", "Readable native trust or APT source authority changed; root verification must refresh package authorization."})
		return
	}
	checked := newDigestChecker(fsys)
	detail := "Root-issued package evidence last authenticated at " + set.VerifiedAt.UTC().Format(time.RFC3339) + "; current package identity, public authority and executable hashes are checked. Age does not expire authorization."
	status := "current"
	if now.Sub(set.VerifiedAt) > ReceiptValidity {
		status = "historical"
		detail = "Historical package evidence last authenticated at " + set.VerifiedAt.UTC().Format(time.RFC3339) + " remains authorized while package, executable and public authority bindings match; native update hooks refresh it locally."
	}
	report.Checks = append(report.Checks, Check{"packages.receipt", status, detail})
	for i := range report.Packages {
		p := &report.Packages[i]
		owner, ok := owners[p.Role]
		if !ok {
			continue
		}
		for _, rec := range set.Packages {
			if !rec.Authorized || !receiptMatches(*p, owner, rec, set.Provider) {
				continue
			}
			if !checked.matches(rec.Files, maxBinary) {
				p.Detail = "Cached authentication receipt is stale or installed bytes differ; explicit verification is required."
				continue
			}
			p.Provenance, p.Detail = "verified", "Authenticated cached evidence; method: "+rec.Method+"."
		}
	}
}

func validReceiptTime(set receiptSet, now time.Time) bool {
	return !set.VerifiedAt.IsZero() && set.VerifiedAt.Unix() > 0 && !set.VerifiedAt.After(now) && (set.ValidUntil.IsZero() || set.ValidUntil.After(set.VerifiedAt))
}
