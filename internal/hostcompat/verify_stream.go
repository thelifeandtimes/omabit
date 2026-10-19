package hostcompat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strconv"
)

type streamRunner func(context.Context, string, []string, int64, func(io.Reader) error) error
type byteCounter struct {
	io.Reader
	n int64
}

func (r *byteCounter) Read(p []byte) (int, error) {
	n, e := r.Reader.Read(p)
	r.n += int64(n)
	return n, e
}

func boundCommands(run commandRunner, w *evidenceWork) commandRunner {
	return func(ctx context.Context, tool string, args []string, limit int64) ([]byte, error) {
		remaining := w.limits.commands - w.commands
		if w.exhausted || remaining <= 0 {
			w.exhausted = true
			return nil, errBudget
		}
		if limit > remaining {
			limit = remaining
		}
		b, err := run(ctx, tool, args, limit)
		if int64(len(b)) > limit || w.bytes("command", int64(len(b))) != nil {
			return nil, errBudget
		}
		return b, err
	}
}
func boundStreams(run streamRunner, w *evidenceWork) streamRunner {
	return func(ctx context.Context, tool string, args []string, limit int64, consume func(io.Reader) error) error {
		remaining := w.limits.commands - w.commands
		if w.exhausted || remaining <= 0 {
			w.exhausted = true
			return errBudget
		}
		if limit > remaining {
			limit = remaining
		}
		var count *byteCounter
		err := run(ctx, tool, args, limit, func(r io.Reader) error { count = &byteCounter{Reader: r}; return consume(count) })
		if count != nil && (count.n > limit || w.bytes("command", count.n) != nil) {
			return errBudget
		}
		return err
	}
}

type verifiedIndex struct {
	targets  map[string]bool
	packages []map[string]string
}

func packageTuple(name, version, arch string) string { return name + "\x00" + version + "\x00" + arch }

func (v *verifier) aptIndex(ctx context.Context, name string, index releaseEntry, p nativePackage) ([]map[string]string, []digest, error) {
	proof, err := digestFiles(v.fsys, []string{name}, maxIndex)
	if err != nil {
		return nil, nil, err
	}
	if v.aptTargets == nil {
		v.aptTargets = map[string]bool{}
	}
	target := packageTuple(p.name, p.version, p.arch)
	v.aptTargets[target] = true
	key := name + ":" + proof[0].SHA256 + ":" + index.hash + ":" + strconv.FormatInt(index.size, 10)
	if cached, ok := v.indexes[key]; ok && cached.targets[target] {
		return cached.packages, proof, nil
	}
	targets := map[string]bool{}
	for name := range v.aptTargets {
		targets[name] = true
	}
	var packages []map[string]string
	h := sha256.New()
	var count *byteCounter
	consume := func(r io.Reader) error {
		count = &byteCounter{Reader: io.TeeReader(r, h)}
		return scanControl(count, 262144, packageIndexFields, func(m map[string]string) error {
			if chargeCount(v.fsys, "record", 1) != nil {
				return errBudget
			}
			if targets[packageTuple(m["Package"], m["Version"], m["Architecture"])] {
				packages = append(packages, m)
				if len(packages) > 64 {
					return errEvidence
				}
			}
			return nil
		})
	}
	if v.stream != nil {
		err = v.stream(ctx, "/usr/lib/apt/apt-helper", []string{"cat-file", "/" + name}, maxIndex, consume)
	} else {
		// Injected adapters can keep their existing bounded command seam.
		var b []byte
		b, err = v.run(ctx, "/usr/lib/apt/apt-helper", []string{"cat-file", "/" + name}, maxIndex)
		if err == nil {
			err = consume(bytes.NewReader(b))
		}
	}
	if err != nil || count == nil || count.n != index.size || hex.EncodeToString(h.Sum(nil)) != index.hash || !verifyDigests(v.fsys, proof, maxIndex) {
		return nil, nil, errEvidence
	}
	if v.indexes == nil {
		v.indexes = map[string]verifiedIndex{}
	}
	if len(v.indexes) >= 128 {
		return nil, nil, errEvidence
	}
	v.indexes[key] = verifiedIndex{targets, packages}
	return packages, proof, nil
}
