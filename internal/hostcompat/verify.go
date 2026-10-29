package hostcompat

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

// commandRunner is injectable to exercise verification without changing the host.
// Production uses fixed root-owned tools, a fixed environment and bounded output.
type commandRunner func(context.Context, string, []string, int64) ([]byte, error)

type verifier struct {
	fsys fs.FS
	run  commandRunner
	// Authentication of a shared artifact is reused only within one refresh.
	authenticated    map[string][]digest
	store            func(context.Context, []byte, string) (string, error)
	sourceByArtifact map[string]string
	pacmanCacheDirs  []string
	pacmanDBDirs     []string
	aptArchivesDir   string
	aptListsDir      string
	fetch            func(context.Context, string, nativePackage) error
	stream           streamRunner
	aptTargets       map[string]bool
	indexes          map[string]verifiedIndex
	payloads         map[string]map[string]string
}

func (v *verifier) verifyPackage(ctx context.Context, provider string, p Package, owner nativePackage) (receipt, error) {
	if v.authenticated == nil {
		v.authenticated = make(map[string][]digest)
	}
	if v.sourceByArtifact == nil {
		v.sourceByArtifact = make(map[string]string)
	}
	r := receipt{Role: p.Role, Name: p.Name, Version: p.Version, PackageIdentity: packageIdentity(owner, p.Executables), Authorized: true}
	metadataBefore, err := digestFiles(v.fsys, owner.metadata, maxMetadata)
	if err != nil {
		return receipt{}, err
	}
	var artifact string
	var evidence []digest
	switch provider {
	case "pacman":
		artifact, evidence, err = v.pacmanArtifact(ctx, owner)
		r.Method = pacmanMethod
	case "deb":
		artifact, evidence, err = v.debArtifact(ctx, owner)
		r.Method = aptMethod
	default:
		err = errEvidence
	}
	if err != nil {
		return receipt{}, err
	}
	paths := make([]string, 0, len(p.Executables))
	for _, exe := range p.Executables {
		paths = append(paths, strings.TrimPrefix(exe, "/"))
	}
	var payload map[string]string
	if v.payloads != nil {
		payload = v.payloads[evidence[0].SHA256]
	}
	for _, name := range paths {
		if payload[name] == "" {
			payload = nil
			break
		}
	}
	if payload == nil {
		wanted := append([]string{}, paths...)
		for name := range owner.files {
			if !containsString(wanted, name) {
				wanted = append(wanted, name)
			}
		}
		sort.Strings(wanted)
		if len(wanted) > 32 {
			return receipt{}, errEvidence
		}
		payload, err = v.packagePayload(ctx, provider, artifact, wanted)
		if err != nil {
			return receipt{}, err
		}
		if v.payloads == nil {
			v.payloads = map[string]map[string]string{}
		}
		v.payloads[evidence[0].SHA256] = payload
	}
	for _, name := range paths {
		h, e := hashTrusted(v.fsys, name, maxBinary)
		if e != nil || payload[name] == "" || h != payload[name] {
			return receipt{}, errors.New("installed executable does not match authenticated package payload")
		}
		r.Files = append(r.Files, digest{name, h})
	}
	r.Metadata, err = digestFiles(v.fsys, owner.metadata, maxMetadata)
	if err != nil || !sameDigests(metadataBefore, r.Metadata) {
		return receipt{}, errEvidence
	}
	if !verifyDigests(v.fsys, evidence, maxArtifact) {
		return receipt{}, errEvidence
	}
	r.Evidence = evidence
	r.SourceID = v.sourceByArtifact[artifact]
	return r, nil
}

// hashTar does not extract files. Only regular payload entries can authenticate an
// executable; links, duplicate members and traversal are rejected.
func hashTar(b []byte, names []string) (map[string]string, error) {
	return hashTarReader(bytes.NewReader(b), names)
}

func hashTarReader(input io.Reader, names []string) (map[string]string, error) {
	wanted := make(map[string]bool)
	for _, name := range names {
		wanted[name] = true
	}
	result := make(map[string]string)
	r := tar.NewReader(input)
	for count := 0; ; count++ {
		if count > maxEntries {
			return nil, errEvidence
		}
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errEvidence
		}
		name := strings.TrimPrefix(h.Name, "./")
		if name == "" && h.Typeflag == tar.TypeDir {
			continue
		}
		if !fs.ValidPath(strings.TrimSuffix(name, "/")) {
			return nil, errEvidence
		}
		if !wanted[name] {
			continue
		}
		if h.Typeflag != tar.TypeReg || h.Size > maxBinary || h.Size <= 0 || result[name] != "" {
			return nil, errEvidence
		}
		d := sha256.New()
		if _, err := io.Copy(d, r); err != nil {
			return nil, errEvidence
		}
		result[name] = hex.EncodeToString(d.Sum(nil))
	}
	if len(result) != len(wanted) {
		return nil, errEvidence
	}
	return result, nil
}

func (v *verifier) pacmanArtifact(ctx context.Context, p nativePackage) (string, []digest, error) {
	attempts := 0
	dirs := append(append([]string{}, v.pacmanCacheDirs...), "var/cache/pacman/pkg")
	for _, dir := range dirs {
		entries, err := readDir(v.fsys, dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !pacmanArchiveName(ent.Name(), p) {
				continue
			}
			attempts++
			if attempts > 8 {
				break
			}
			artifact := dir + "/" + ent.Name()
			if evidence := v.authenticated[artifact]; evidence != nil {
				return artifact, evidence, nil
			}
			signature := artifact + ".sig"
			if _, err := readTrusted(v.fsys, signature, 128<<10); err != nil {
				signature, err = v.embeddedPacmanSignature(ctx, p)
				if err != nil {
					continue
				}
			}
			evidence, err := digestFiles(v.fsys, []string{artifact, signature}, maxArtifact)
			if err != nil {
				continue
			}
			if err := v.pacmanSignature(ctx, artifact, signature); err != nil {
				continue
			}
			b, err := v.run(ctx, "/usr/bin/bsdtar", []string{"-xOf", "/" + artifact, "--", ".PKGINFO"}, 256<<10)
			if err != nil {
				continue
			}
			m := make(map[string]string)
			valid := true
			for _, line := range strings.Split(string(b), "\n") {
				k, val, ok := strings.Cut(line, " = ")
				if !ok {
					continue
				}
				if k == "pkgname" || k == "pkgver" || k == "arch" {
					if m[k] != "" {
						valid = false
					}
					m[k] = val
				}
			}
			if !valid || m["pkgname"] != p.name || m["pkgver"] != p.version || m["arch"] != p.arch {
				continue
			}
			if !verifyDigests(v.fsys, evidence, maxArtifact) {
				continue
			}
			v.authenticated[artifact] = evidence
			return artifact, evidence, nil
		}
	}
	return "", nil, errors.New("cached package and trusted Pacman signature evidence are unavailable or invalid")
}

func (v *verifier) pacmanSignature(ctx context.Context, artifact, signature string) error {
	status, err := v.run(ctx, "/usr/bin/gpg", []string{"--no-options", "--homedir", "/etc/pacman.d/gnupg", "--batch", "--no-auto-check-trustdb", "--no-auto-key-retrieve", "--no-auto-key-import", "--auto-key-locate", "clear", "--trust-model", "pgp", "--status-fd", "1", "--verify", "/" + signature, "/" + artifact}, 64<<10)
	if err != nil || !trustedSignature(status) {
		return errEvidence
	}
	entries, err := readDir(v.fsys, "usr/share/pacman/keyrings")
	if err != nil || len(entries) > 128 {
		return errEvidence
	}
	var signers []string
	for _, line := range strings.Split(string(status), "\n") {
		f := strings.Fields(line)
		if len(f) >= 11 && f[0] == "[GNUPG:]" && f[1] == "VALIDSIG" {
			signers = append(signers, strings.ToUpper(f[2]))
			if len(f) > 11 {
				signers = append(signers, strings.ToUpper(f[11]))
			}
		}
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "-revoked") && !strings.HasSuffix(entry.Name(), ".revoked") {
			continue
		}
		b, e := readTrusted(v.fsys, "usr/share/pacman/keyrings/"+entry.Name(), 1<<20)
		if e != nil {
			return errEvidence
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
			if line == "" {
				continue
			}
			if len(line) != 40 && len(line) != 64 {
				return errEvidence
			}
			if _, e := hex.DecodeString(line); e != nil {
				return errEvidence
			}
			if containsString(signers, strings.ToUpper(line)) {
				return errEvidence
			}
		}
	}
	return nil
}

type releaseEntry struct {
	hash, name string
	size       int64
}

func releaseHashes(b []byte) ([]releaseEntry, error) {
	stanzas, err := parseControl(string(b))
	if err != nil || len(stanzas) != 1 {
		return nil, errEvidence
	}
	// InRelease expiration is checked on refresh; offline receipts remain usable
	// while bound package, payload and trust evidence remain unchanged.
	var result []releaseEntry
	for _, line := range strings.Split(stanzas[0]["SHA256"], "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 || len(fields[0]) != 64 || !fs.ValidPath(fields[2]) {
			return nil, errEvidence
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			return nil, errEvidence
		}
		n, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || n <= 0 || n > maxIndex {
			continue
		}
		if strings.HasSuffix(fields[2], "/Packages") {
			result = append(result, releaseEntry{fields[0], fields[2], n})
		}
	}
	if len(result) == 0 || len(result) > 256 {
		return nil, errEvidence
	}
	return result, nil
}

func (v *verifier) debArtifact(ctx context.Context, p nativePackage) (string, []digest, error) {
	archives := v.aptArchivesDir
	if archives == "" {
		archives = "var/cache/apt/archives"
	}
	cache, err := readDir(v.fsys, archives)
	if err != nil {
		return "", nil, err
	}
	var candidates []string
	for _, e := range cache {
		if strings.HasPrefix(e.Name(), p.name+"_") && strings.HasSuffix(e.Name(), "_"+p.arch+".deb") {
			candidates = append(candidates, archives+"/"+e.Name())
		}
	}
	if len(candidates) == 0 || len(candidates) > 8 {
		return "", nil, errEvidence
	}
	for _, name := range candidates {
		if evidence := v.authenticated[name]; evidence != nil {
			return name, evidence, nil
		}
	}
	sources, _, err := aptSources(v.fsys)
	if err != nil {
		return "", nil, err
	}
	for _, source := range sources {
		releaseName := source.releasePath()
		if v.aptListsDir != "" {
			releaseName = v.aptListsDir + "/" + strings.TrimPrefix(releaseName, "var/lib/apt/lists/")
		}
		if _, err := readTrusted(v.fsys, releaseName, 4<<20); err != nil {
			continue
		}
		releaseDigest, err := digestFiles(v.fsys, []string{releaseName}, 4<<20)
		if err != nil {
			continue
		}
		plain, err := v.signedRelease(ctx, source, releaseName, false)
		if err != nil {
			continue
		}
		hashes, err := releaseHashes(plain)
		if err != nil {
			continue
		}
		for _, index := range hashes {
			if !source.permitsIndex(index.name, p.arch) {
				continue
			}
			base := strings.TrimSuffix(releaseName, "_InRelease") + "_" + strings.ReplaceAll(index.name, "/", "_")
			for _, ext := range []string{"", ".lz4", ".xz", ".gz"} {
				indexName := base + ext
				stanzas, indexDigest, err := v.aptIndex(ctx, indexName, index, p)
				if err != nil {
					continue
				}
				for _, m := range stanzas {
					if m["Package"] != p.name || m["Version"] != p.version || m["Architecture"] != p.arch || len(m["SHA256"]) != 64 {
						continue
					}
					for _, name := range candidates {
						d, err := hashTrusted(v.fsys, name, maxArtifact)
						_, info, e := resolve(v.fsys, name)
						if err != nil || e != nil || d != m["SHA256"] || fmt.Sprint(info.Size()) != m["Size"] {
							continue
						}
						fields, err := v.run(ctx, "/usr/bin/dpkg-deb", []string{"--field", "/" + name, "Package", "Version", "Architecture"}, 64<<10)
						control, ce := parseControl(string(fields))
						if err != nil || ce != nil || len(control) != 1 || control[0]["Package"] != p.name || control[0]["Version"] != p.version || control[0]["Architecture"] != p.arch {
							continue
						}
						evidence := append([]digest{{name, d}}, releaseDigest...)
						evidence = append(evidence, indexDigest...)
						if !verifyDigests(v.fsys, evidence, maxArtifact) {
							continue
						}
						v.authenticated[name] = evidence
						v.sourceByArtifact[name] = source.proofID()
						return name, evidence, nil
					}
				}
			}
		}
	}
	return "", nil, errors.New("cached APT signed-index hash chain and matching package are unavailable or invalid")
}

func (v *verifier) signedRelease(ctx context.Context, source aptSource, name string, historical bool) ([]byte, error) {
	keys, err := v.aptKeyArgs(ctx, source)
	if err != nil {
		return nil, err
	}
	_, before, err := resolve(v.fsys, name)
	if err != nil {
		return nil, err
	}
	raw, err := readTrusted(v.fsys, name, 4<<20)
	if err != nil {
		return nil, err
	}
	args := append(append([]string{}, keys...), "--status-fd", "1", "/"+name)
	status, err := v.run(ctx, "/usr/bin/gpgv", args, 64<<10)
	if ctx.Err() != nil || !signatureExitAllowed(err) || !signatureStatus(status, false) {
		return nil, errEvidence
	}
	plain, err := releaseCleartext(raw)
	_, after, stateErr := resolve(v.fsys, name)
	h := sha256.Sum256(raw)
	if err != nil || ctx.Err() != nil || (!historical && !freshRelease(plain)) || stateErr != nil || !sameFileState(before, after) || !verifyDigests(v.fsys, []digest{{name, hex.EncodeToString(h[:])}}, 4<<20) {
		return nil, errEvidence
	}
	fields, err := parseControl(string(plain))
	if err != nil || len(fields) != 1 || (fields[0]["Suite"] != source.Suite && fields[0]["Codename"] != source.Suite) {
		return nil, errEvidence
	}
	return plain, nil
}

// Pacman's keyring is a web-of-trust store, not a bag of trusted keys. A valid
// signature alone is insufficient: require GnuPG's FULLY/ULTIMATE trust result.
func trustedSignature(b []byte) bool {
	return signatureStatus(b, true)
}

func signatureStatus(b []byte, requireTrust bool) bool {
	if !requireTrust {
		return aptSignatureStatus(b)
	}
	valid, trusted := 0, !requireTrust
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] != "[GNUPG:]" {
			continue
		}
		switch f[1] {
		case "VALIDSIG":
			if !strongValidSignature(f) {
				return false
			}
			valid++
		case "TRUST_FULLY", "TRUST_ULTIMATE":
			trusted = true
		case "BADSIG", "ERRSIG", "EXPSIG", "EXPKEYSIG", "REVKEYSIG", "KEYREVOKED", "KEYEXPIRED", "SIGEXPIRED", "NO_PUBKEY", "NODATA", "FAILURE", "ERROR":
			return false
		}
	}
	if requireTrust {
		return valid == 1 && trusted
	}
	return valid > 0 && valid <= 16
}
