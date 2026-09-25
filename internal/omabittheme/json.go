package omabittheme

import (
	"bytes"
	"encoding/json"
	"io"
)

// Settings merges must not silently collapse duplicate keys or round unknown
// numeric fields. Preserve json.Number and reject ambiguous/deep input.
func strictJSON(data []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var read func(int) (any, error)
	read = func(depth int) (any, error) {
		if depth > 64 {
			return nil, ErrMalformed
		}
		token, err := d.Token()
		if err != nil {
			return nil, ErrMalformed
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return token, nil
		}
		switch delim {
		case '{':
			m := map[string]any{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, ErrMalformed
				}
				name, ok := key.(string)
				if !ok {
					return nil, ErrMalformed
				}
				if _, ok := m[name]; ok {
					return nil, ErrMalformed
				}
				value, err := read(depth + 1)
				if err != nil {
					return nil, err
				}
				m[name] = value
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return nil, ErrMalformed
			}
			return m, nil
		case '[':
			a := []any{}
			for d.More() {
				value, err := read(depth + 1)
				if err != nil {
					return nil, err
				}
				a = append(a, value)
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return nil, ErrMalformed
			}
			return a, nil
		}
		return nil, ErrMalformed
	}
	v, err := read(0)
	if err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, ErrMalformed
	}
	return v, nil
}
