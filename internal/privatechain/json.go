package privatechain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const MaxFoundationBytes = 64 * 1024

// DecodeFoundation accepts only bounded unambiguous public review metadata. It
// deliberately returns no malformed values or input fragments in its error.
func DecodeFoundation(data []byte) (Foundation, error) {
	bad := errors.New("expected bounded supported public private-chain foundation metadata")
	if len(data) > MaxFoundationBytes || !uniqueJSON(data) {
		return Foundation{}, bad
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var f Foundation
	if decoder.Decode(&f) != nil || decoder.Decode(new(any)) != io.EOF {
		return Foundation{}, bad
	}
	canonical, err := canonicalFoundation(f)
	if err != nil {
		return Foundation{}, bad
	}
	return canonical, nil
}
func uniqueJSON(data []byte) bool { return uniqueJSONDepth(data, 4) }
func uniqueJSONDepth(data []byte, maxDepth int) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	var value func(int) bool
	value = func(depth int) bool {
		if depth > maxDepth {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		delim, container := token.(json.Delim)
		if !container {
			return true
		}
		if delim != '{' && delim != '[' {
			return false
		}
		keys := map[string]bool{}
		for d.More() {
			if delim == '{' {
				token, err := d.Token()
				name, ok := token.(string)
				if err != nil || !ok || name != strings.ToLower(name) || keys[name] {
					return false
				}
				keys[name] = true
			}
			if !value(depth + 1) {
				return false
			}
		}
		end, err := d.Token()
		return err == nil && (delim == '{' && end == json.Delim('}') || delim == '[' && end == json.Delim(']'))
	}
	return value(0) && d.Decode(new(any)) == io.EOF
}
