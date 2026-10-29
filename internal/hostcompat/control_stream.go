package hostcompat

import (
	"bufio"
	"io"
	"strings"
)

// scanControl retains only selected fields of one stanza at a time. Description
// bodies and unrelated repository paragraphs never become a repository-sized
// slice of maps. Duplicate fields/oversized records still fail closed.
func scanControl(r io.Reader, maxRecords int, fields map[string]bool, visit func(map[string]string) error) error {
	s := bufio.NewScanner(r)
	// Ubuntu Jammy includes unrelated Provides lines above 64 KiB. Match the
	// existing 1 MiB field budget without retaining those fields. Security-
	// relevant selected identity/hash/path values remain capped at 4096 bytes.
	s.Buffer(make([]byte, 8192), (1<<20)+2)
	values := map[string]string{}
	seen := map[string]bool{}
	key := ""
	fieldBytes := 0
	records := 0
	flush := func() error {
		if len(seen) == 0 {
			return nil
		}
		records++
		if records > maxRecords {
			return errEvidence
		}
		if err := visit(values); err != nil {
			return err
		}
		values = map[string]string{}
		seen = map[string]bool{}
		key = ""
		fieldBytes = 0
		return nil
	}
	for s.Scan() {
		line := s.Text()
		if len(line) > 1<<20 {
			return errEvidence
		}
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			if key == "" {
				return errEvidence
			}
			// The selected package identity/hash/path fields are single-line
			// tokens. Refuse folds instead of quadratic string concatenation.
			if fields != nil && fields[key] {
				return errEvidence
			}
			fieldBytes += len(line)
			if fieldBytes > 1<<20 {
				return errEvidence
			}
			if fields == nil || fields[key] {
				values[key] += "\n" + strings.TrimSpace(line)
			}
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok || !token(k) || seen[k] || len(seen) >= 128 {
			return errEvidence
		}
		seen[k] = true
		key = k
		fieldBytes = len(v)
		if fields != nil && fields[k] && fieldBytes > 4096 {
			return errEvidence
		}
		if fields == nil || fields[k] {
			values[k] = strings.TrimSpace(v)
		}
	}
	if s.Err() != nil {
		return errEvidence
	}
	return flush()
}

var packageStatusFields = map[string]bool{"Package": true, "Status": true, "Version": true, "Architecture": true}
var packageIndexFields = map[string]bool{"Package": true, "Version": true, "Architecture": true, "Size": true, "SHA256": true, "Filename": true}
