package hostcompat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path"
	"strings"
	"time"
)

// retainReceipt copies exactly the bytes authenticated above. A cache rotation
// or mutation during copy cannot publish a receipt for different evidence.
func (v *verifier) retainReceipt(ctx context.Context, rec receipt) (receipt, error) {
	if v.store == nil {
		return receipt{}, errEvidence
	}
	rec.Retained = nil
	for _, item := range rec.Evidence {
		if ctx.Err() != nil {
			return receipt{}, errEvidence
		}
		b, err := readTrusted(v.fsys, item.Path, maxArtifact)
		if err != nil {
			return receipt{}, err
		}
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != item.SHA256 {
			return receipt{}, errEvidence
		}
		name, err := v.store(ctx, b, path.Ext(item.Path))
		if err != nil {
			return receipt{}, err
		}
		rec.Retained = append(rec.Retained, digest{name, item.SHA256})
	}
	if !verifyDigests(v.fsys, rec.Retained, maxArtifact) {
		return receipt{}, errEvidence
	}
	return rec, nil
}

// Reuse never trusts a cache filename or a stale authorization bit. The saved
// payload hashes came from a root-owned receipt; retained signed evidence is
// checked under current native trust, then current installed bytes are checked.
// Only previously authenticated Release expiry may be historical here.
func (v *verifier) reuseReceipt(ctx context.Context, provider string, p Package, owner nativePackage, rec receipt) (receipt, error) {
	if !receiptMatches(p, owner, rec, provider) || len(rec.Evidence) != len(rec.Retained) {
		return receipt{}, errEvidence
	}
	want := 2
	if provider == "deb" {
		want = 3
	}
	if len(rec.Retained) != want {
		return receipt{}, errEvidence
	}
	for i, item := range rec.Retained {
		if !strings.HasPrefix(item.Path, strings.TrimPrefix(VerificationCacheDirectory, "/")+"/") || item.SHA256 != rec.Evidence[i].SHA256 {
			return receipt{}, errEvidence
		}
	}
	if !verifyDigests(v.fsys, rec.Retained, maxArtifact) || !verifyDigests(v.fsys, rec.Files, maxBinary) {
		return receipt{}, errEvidence
	}
	if provider == "pacman" {
		if v.pacmanSignature(ctx, rec.Retained[0].Path, rec.Retained[1].Path) != nil {
			return receipt{}, errEvidence
		}
	} else {
		sources, _, err := aptSources(v.fsys)
		if err != nil {
			return receipt{}, err
		}
		found := false
		for _, source := range sources {
			if source.proofID() == rec.SourceID && source.permitsRetainedIndex(rec.Evidence[2].Path, owner.arch) {
				if _, err := v.signedRelease(ctx, source, rec.Retained[1].Path, true); err == nil {
					found = true
				}
				break
			}
		}
		if !found {
			return receipt{}, errEvidence
		}
	}
	if !verifyDigests(v.fsys, rec.Retained, maxArtifact) || !verifyDigests(v.fsys, rec.Files, maxBinary) {
		return receipt{}, errEvidence
	}
	rec.Authorized = true
	return rec, nil
}

// refreshPackages is independent of real UID and filesystem writes so the
// complete hook workflow is tested against an injected root filesystem.
func (v *verifier) refreshPackages(ctx context.Context, snapshot inspection, old receiptSet) (receiptSet, bool, error) {
	if v.aptTargets == nil {
		v.aptTargets = map[string]bool{}
	}
	for _, owner := range snapshot.owners {
		v.aptTargets[packageTuple(owner.name, owner.version, owner.arch)] = true
	}
	provider := snapshot.report.Distribution.Provider
	set := receiptSet{SchemaVersion: 2, PolicyID: PolicyID, Provider: provider, VerifiedAt: time.Now().UTC()}
	trust, err := trustSnapshot(v.fsys, provider)
	if err != nil {
		return withdrawAuthorization(set, old), true, nil
	}
	public, policy, err := publicTrustSnapshot(v.fsys, provider)
	if err != nil {
		return withdrawAuthorization(set, old), true, nil
	}
	failed := false
	for _, p := range snapshot.report.Packages {
		owner, ok := snapshot.owners[p.Role]
		var previous *receipt
		for i := range old.Packages {
			if old.Packages[i].Role == p.Role {
				previous = &old.Packages[i]
				break
			}
		}
		var rec receipt
		err = errEvidence
		if ok && ctx.Err() == nil && previous != nil {
			rec, err = v.reuseReceipt(ctx, provider, p, owner, *previous)
		}
		if ok && ctx.Err() == nil && err != nil {
			rec, err = v.verifyPackage(ctx, provider, p, owner)
			if err != nil && v.fetch != nil && ctx.Err() == nil {
				if fetchErr := v.fetch(ctx, provider, owner); fetchErr == nil {
					rec, err = v.verifyPackage(ctx, provider, p, owner)
				}
			}
			if err == nil {
				rec, err = v.retainReceipt(ctx, rec)
			}
		}
		if err != nil {
			failed = true
			// Keep bounded recovery proof, but withdraw current authorization.
			if previous != nil {
				rec = *previous
				rec.Authorized = false
				set.Packages = append(set.Packages, rec)
			}
			continue
		}
		set.Packages = append(set.Packages, rec)
	}
	after, err := trustSnapshot(v.fsys, provider)
	if err != nil || !sameDigests(trust, after) {
		return set, true, errEvidence
	}
	publicAfter, policyAfter, err := publicTrustSnapshot(v.fsys, provider)
	if err != nil || !sameDigests(public, publicAfter) || policy != policyAfter || ctx.Err() != nil {
		return set, true, errEvidence
	}
	set.Trust, set.PublicTrust, set.SourcePolicy = after, publicAfter, policyAfter
	// Reinspect semantic ownership/version after all commands, rather than
	// binding every role to the entire mutable dpkg status database.
	current := inspect(v.fsys, "linux", strings.TrimPrefix(snapshot.report.Platform, "linux/"))
	for i, rec := range set.Packages {
		if !rec.Authorized {
			continue
		}
		valid := false
		for _, p := range current.report.Packages {
			if owner, ok := current.owners[p.Role]; ok && receiptMatches(p, owner, rec, provider) && verifyDigests(v.fsys, rec.Files, maxBinary) {
				valid = true
				break
			}
		}
		if !valid {
			set.Packages[i].Authorized = false
			failed = true
		}
	}
	return set, failed, nil
}

func withdrawAuthorization(set, old receiptSet) receiptSet {
	set.Packages = append([]receipt{}, old.Packages...)
	for i := range set.Packages {
		set.Packages[i].Authorized = false
	}
	return set
}
