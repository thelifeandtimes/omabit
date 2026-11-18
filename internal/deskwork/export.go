package deskwork

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const MaxExportFiles = 64
const MaxExportBytes = 256 << 10

type ExportedFile struct {
	Path          string `json:"path"`
	ContentBase64 string `json:"content_base64"`
	SHA256        string `json:"sha256"`
}

func ValidateExportPaths(paths []string) error {
	if len(paths) == 0 || len(paths) > MaxExportFiles {
		return errors.New("select a bounded nonempty source-file list")
	}
	for i, name := range paths {
		if !ValidPath(name) || i > 0 && paths[i-1] >= name {
			return errors.New("source export requires sorted unique ordinary Clay paths")
		}
	}
	return nil
}

// Decode returns exact binary bytes after checking canonical base64 and its hash.
func (file ExportedFile) Decode() ([]byte, error) {
	if !ValidPath(file.Path) || len(file.ContentBase64) > base64.StdEncoding.EncodedLen(MaxExportBytes) {
		return nil, errors.New("invalid source export encoding")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(file.ContentBase64)
	if err != nil || len(data) > MaxExportBytes || base64.StdEncoding.EncodeToString(data) != file.ContentBase64 {
		return nil, errors.New("invalid source export bytes")
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != file.SHA256 {
		return nil, errors.New("source export hash differs")
	}
	if data == nil {
		data = []byte{}
	}
	return data, nil
}
func ValidateExportedFiles(paths []string, files []ExportedFile) error {
	if ValidateExportPaths(paths) != nil || len(files) != len(paths) {
		return errors.New("source export changed selected paths")
	}
	total := 0
	for i, file := range files {
		data, err := file.Decode()
		total += len(data)
		if err != nil || file.Path != paths[i] || total > MaxExportBytes {
			return errors.New("invalid source export bytes")
		}
	}
	return nil
}

// ExportFiles is a pure historical Clay read. Mark-to-MIME conversion produces
// source bytes, not necessarily the publisher's original spelling. No mount,
// source ownership, current revision or app policy is changed. The caller must
// bracket this read with exact runtime, pier and desk-baseline observations.
func ExportFiles(ship, desk string, revision uint64, paths []string) (LensRequest, error) {
	if !validSelection(ship, desk) || revision == 0 || revision > 1<<53-1 || ValidateExportPaths(paths) != nil {
		return LensRequest{}, errors.New("invalid source export selection")
	}
	rows := []any{}
	for _, name := range paths {
		parts := strings.Split(name, "/")
		last := len(parts) - 1
		dot := strings.LastIndexByte(parts[last], '.')
		base := parts[last]
		parts[last] = base[:dot]
		parts = append(parts, base[dot+1:])
		rows = append(rows, []any{name, parts})
	}
	data, _ := json.Marshal(rows)
	encoded := string(data)
	expression := fmt.Sprintf(`|=  encoded=@t
=/  wen  (scot %%da now)
?>  (~(has in .^((set @tas) %%cd /%s//[wen])) %%%s)
?>  (lte %s let:.^(domo:clay %%cv /%s/%s/[wen]))
=/  root  /%s/%s/%s
=/  rows  ((ar:dejs:format same) (need (de:json:html encoded)))
?>  (lte (lent rows) 64)
=/  files=(list [name=@t data=octs])
  %%+  turn  rows
  |=  row=json
  =/  fields  ((ar:dejs:format same) row)
  ?>  ?=([* * ~] fields)
  =/  name  (so:dejs:format i.fields)
  =/  p=path  ((ar:dejs:format so:dejs:format) i.t.fields)
  ?>  .^(? %%cu (weld root p))
  =/  convert  .^(tube:clay %%cc (weld root ~[(rear p) %%mime]))
  =/  source=mime  !<(mime (convert .^(vase %%cr (weld root p))))
  [name q.source]
?>  (lte (roll files |=([f=[name=@t data=octs] n=@ud] (add p.data.f n))) 262.144)
=/  value=json
  :-  %%a
  %%+  turn  files
  |=  f=[name=@t data=octs]
  [%%o (malt `+"`(list [@t json])`"+`~[['path' [%%s name.f]] ['content' [%%s (en:base64:mimes:html data.f)]]])]
`+"`@ux`"+`(en:json:html value)`, ship, desk, hoonDecimal(revision), ship, desk, ship, desk, hoonDecimal(revision))
	return LensRequest{expression: expression, input: &encoded}, nil
}

func DecodeExportedFiles(output string, paths []string) ([]ExportedFile, error) {
	var rows []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	encoded, _ := json.Marshal(output)
	if ValidateExportPaths(paths) != nil || len(output) > 1<<20 || DecodeJSONOutput(encoded, &rows) != nil || len(rows) != len(paths) {
		return nil, errors.New("invalid source export response")
	}
	files := []ExportedFile{}
	total := 0
	for i, row := range rows {
		if row.Path != paths[i] || len(row.Content) > base64.StdEncoding.EncodedLen(MaxExportBytes) {
			return nil, errors.New("invalid source export row")
		}
		data, err := base64.StdEncoding.Strict().DecodeString(row.Content)
		total += len(data)
		if err != nil || total > MaxExportBytes || base64.StdEncoding.EncodeToString(data) != row.Content {
			return nil, errors.New("invalid or oversized source export bytes")
		}
		if data == nil {
			data = []byte{}
		}
		digest := sha256.Sum256(data)
		files = append(files, ExportedFile{Path: row.Path, ContentBase64: row.Content, SHA256: hex.EncodeToString(digest[:])})
	}
	return files, ValidateExportedFiles(paths, files)
}
