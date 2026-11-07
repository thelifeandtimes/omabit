package deskwork

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// CanonicalFile records the SHA-256 of the jammed [mark noun] that Clay stores.
// It deliberately is not the raw Unix byte hash, since marks normalize bytes.
type CanonicalFile struct {
	Path []string `json:"path"`
	Hash string   `json:"hash"`
}

func CanonicalManifest(files []CanonicalFile) (Manifest, error) {
	if files == nil || len(files) > MaxFiles {
		return Manifest{}, errors.New("invalid canonical inventory")
	}
	paths := make([][]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	if _, err := UnixPaths(paths); err != nil {
		return Manifest{}, err
	}
	out := make([]File, 0, len(files))
	for _, f := range files {
		hash := strings.TrimPrefix(f.Hash, "0x")
		if hash == f.Hash {
			return Manifest{}, errors.New("invalid canonical hash")
		}
		hash = strings.ReplaceAll(hash, ".", "")
		if len(hash) == 0 || len(hash) > 64 {
			return Manifest{}, errors.New("invalid canonical hash")
		}
		hash = strings.Repeat("0", 64-len(hash)) + hash
		if !digestPattern.MatchString(hash) {
			return Manifest{}, errors.New("invalid canonical hash")
		}
		p := f.Path
		out = append(out, File{Path: strings.Join(p[:len(p)-1], "/") + "." + p[len(p)-1], SHA256: hash})
	}
	return NewManifest(out)
}
func ClayFiles(ship, desk string, revision uint64) (LensRequest, error) {
	if !validSelection(ship, desk) || revision == 0 || revision > 1<<53-1 {
		return LensRequest{}, errors.New("invalid desk revision")
	}
	expr := fmt.Sprintf(`=/  wen  (scot %%da now)
?>  (~(has in .^((set @tas) %%cd /%s//[wen])) %%%s)
?>  (lte %s let:.^(domo:clay %%cv /%s/%s/[wen]))
=/  root  /%s/%s/%s
=/  paths  .^((list path) %%ct root)
?>  (lte (lent paths) 8.192)
=/  value=json
  :-  %%a
  %%+  turn  paths
  |=  p=path
  =/  hash  (shax (jam [(rear p) .^(* %%cx (weld root p))]))
  [%%o (malt `+"`(list [@t json])`"+`~[['path' [%%a (turn p |=(s=@ta [%%s s]))]] ['hash' [%%s (scot %%ux hash)]]])]
`+"`@ux`(en:json:html value)", ship, desk, hoonDecimal(revision), ship, desk, ship, desk, hoonDecimal(revision))
	return LensRequest{expression: expr}, nil
}

type SourceFile struct {
	Path  string
	Bytes []byte
}

// NormalizeFiles uses only the selected historical desk's mark converters.
// New/unsupported marks fail visibly; callers must not substitute raw equality.
// Batches cap working memory and transport size independently of the tree limit.
func NormalizeFiles(ship, desk string, revision uint64, files []SourceFile) (LensRequest, error) {
	if !validSelection(ship, desk) || revision == 0 || revision > 1<<53-1 || files == nil || len(files) > 128 {
		return LensRequest{}, errors.New("invalid normalization batch")
	}
	data := []any{}
	seen := map[string]bool{}
	var total int64
	for _, f := range files {
		if !ValidPath(f.Path) || seen[f.Path] || int64(len(f.Bytes)) > MaxFileBytes {
			return LensRequest{}, errors.New("invalid normalization source")
		}
		seen[f.Path] = true
		total += int64(len(f.Bytes))
		if total > MaxFileBytes {
			return LensRequest{}, errors.New("normalization batch exceeds limit")
		}
		p := strings.Split(f.Path, "/")
		base := p[len(p)-1]
		dot := strings.LastIndexByte(base, '.')
		p[len(p)-1] = base[:dot]
		p = append(p, base[dot+1:])
		data = append(data, []any{p, base64.StdEncoding.EncodeToString(f.Bytes)})
	}
	raw, _ := json.Marshal(data)
	encoded := string(raw)
	expr := fmt.Sprintf(`|=  encoded=@t
=/  wen  (scot %%da now)
?>  (~(has in .^((set @tas) %%cd /%s//[wen])) %%%s)
?>  (lte %s let:.^(domo:clay %%cv /%s/%s/[wen]))
=/  rows  ((ar:dejs:format same) (need (de:json:html encoded)))
?>  (lte (lent rows) 128)
=/  value=json
  :-  %%a
  %%+  turn  rows
  |=  row=json
  =/  fields  ((ar:dejs:format same) row)
  ?>  ?=([* * ~] fields)
  =/  p=path  ((ar:dejs:format so:dejs:format) i.fields)
  =/  raw  (need (de:base64:mimes:html (so:dejs:format i.t.fields)))
  =/  file-mark  (rear p)
  =/  convert  .^(tube:clay %%cc /%s/%s/%s/mime/[file-mark])
  =/  hash  (shax (jam [file-mark q:(convert !>(`+"`mime`"+`[/text/plain raw]))]))
  [%%o (malt `+"`(list [@t json])`"+`~[['path' [%%a (turn p |=(s=@ta [%%s s]))]] ['hash' [%%s (scot %%ux hash)]]])]
`+"`@ux`(en:json:html value)", ship, desk, hoonDecimal(revision), ship, desk, ship, desk, hoonDecimal(revision))
	return LensRequest{expression: expr, input: &encoded}, nil
}
