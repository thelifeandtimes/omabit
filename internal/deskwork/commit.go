package deskwork

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type DeltaFile struct {
	Path   string
	Delete bool
	Bytes  []byte
}

// CommitFiles targets Clay by desk identity instead of trusting a mutable Unix
// mount alias. The reviewed delta preserves every unspecified file and supplies
// MIME pages for Clay's native mark conversion in the resulting tree. Source
// acceptance does not establish application compilation or activation.
// The caller must still enforce fresh policy and journal dispatch before use.
func CommitFiles(ship, desk string, baseline Observation, changes []DeltaFile) (LensRequest, error) {
	if !validSelection(ship, desk) || baseline.Validate() != nil || !baseline.Exists || len(changes) == 0 || len(changes) > MaxFiles {
		return LensRequest{}, errors.New("invalid desk commit delta")
	}
	rows := []any{}
	seen := map[string]bool{}
	var total int64
	for _, f := range changes {
		if !ValidPath(f.Path) || seen[f.Path] || int64(len(f.Bytes)) > MaxFileBytes || f.Delete && len(f.Bytes) != 0 {
			return LensRequest{}, errors.New("invalid desk commit file")
		}
		seen[f.Path] = true
		total += int64(len(f.Bytes))
		if total > MaxFileBytes {
			return LensRequest{}, errors.New("single commit exceeds 32 MiB")
		}
		p := strings.Split(f.Path, "/")
		base := p[len(p)-1]
		dot := strings.LastIndexByte(base, '.')
		p[len(p)-1] = base[:dot]
		p = append(p, base[dot+1:])
		action := "write"
		if f.Delete {
			action = "delete"
		}
		rows = append(rows, []any{p, action, base64.StdEncoding.EncodeToString(f.Bytes)})
	}
	data, _ := json.Marshal(rows)
	input := string(data)
	expr := fmt.Sprintf(`|=  encoded=@t
=/  wen  (scot %%da now)
?>  (~(has in .^((set @tas) %%cd /%s//[wen])) %%%s)
=/  root  /%s/%s/[wen]
?>  =(%s let:.^(domo:clay %%cv root))
?>  =(%s .^(@uvI %%cz root))
=/  rows  ((ar:dejs:format same) (need (de:json:html encoded)))
?>  (lte (lent rows) 8.192)
=/  delta=(list [p=path q=miso:clay])
  %%+  turn  rows
  |=  row=json
  =/  fields  ((ar:dejs:format same) row)
  ?>  ?=([* * * ~] fields)
  =/  p=path  ((ar:dejs:format so:dejs:format) i.fields)
  =/  action  (so:dejs:format i.t.fields)
  ?:  =(action 'delete')  [p %%del ~]
  ?>  =(action 'write')
  =/  raw  (need (de:base64:mimes:html (so:dejs:format i.t.t.fields)))
  =/  page=cage  [%%mime !>(`+"`mime`"+`[/text/plain raw])]
  ?:  .^(? %%cu (weld root p))  [p %%mut page]
  [p %%ins page]
[%%c %%info %%%s [%%& delta]]`, ship, desk, ship, desk, hoonDecimal(baseline.Revision), baseline.ClayHash, desk)
	return LensRequest{expression: expr, app: "hood", mark: "helm-pass", input: &input}, nil
}
