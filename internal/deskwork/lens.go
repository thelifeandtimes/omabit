package deskwork

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var shipName = regexp.MustCompile(`^~[a-z]{3,6}(?:-{1,2}[a-z]{6})*$`)
var deskName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func validSelection(ship, desk string) bool {
	return len(ship) <= 128 && shipName.MatchString(ship) && deskName.MatchString(desk)
}

// LensRequest can only be built through typed constructors below. No caller
// supplied Hoon, agent name, mark, URL or filesystem path is accepted.
type LensRequest struct {
	expression, app, mark string
	input                 *string
}

func (r LensRequest) Body() ([]byte, error) {
	if r.expression == "" {
		return nil, errors.New("empty desk request")
	}
	var source any = map[string]any{"dojo": r.expression}
	if r.input != nil {
		source = map[string]any{"hoon": map[string]any{"code": r.expression, "next": map[string]any{"data": *r.input}}}
	}
	if r.mark != "" {
		source = map[string]any{"as": map[string]any{"mark": r.mark, "next": source}}
	}
	sink := map[string]any{"stdout": nil}
	if r.app != "" {
		sink = map[string]any{"app": r.app}
	}
	return json.Marshal(map[string]any{"source": source, "sink": sink})
}
func Inventory(ship, desk string) (LensRequest, error) {
	if !validSelection(ship, desk) {
		return LensRequest{}, errors.New("invalid desk selection")
	}
	// Check existence before querying a desk. An unresolved Clay scry can leave a
	// Lens job waiting; it is not a safe way to discover an absent desk.
	expression := fmt.Sprintf(`=/  ego  (scot %%p %s)
=/  syd  %%%s
=/  wen  (scot %%da now)
=/  pre  /[ego]/[syd]/[wen]
=/  present  (~(has in .^((set @tas) %%cd /[ego]//[wen])) syd)
=/  value=json
  ?.  present  [%%o (malt `+"`(list [@t json])`"+`~[['exists' [%%b %%.n]]])]
  =/  paths  .^((list path) %%ct pre)
  ?>  (lte (lent paths) 8.192)
  =/  private
    %%+  levy  ^-((list path) [~ paths])
    |=  p=path
    =/  permissions  .^([r=dict:clay w=dict:clay] %%cp (weld pre p))
    =([%%white ~ ~] rul.r.permissions)
  =/  revision  let:.^(domo:clay %%cv pre)
  =/  digest  .^(@uvI %%cz pre)
  [%%o (malt `+"`(list [@t json])`"+`~[['exists' [%%b %%.y]] ['revision' [%%n (scot %%ud revision)]] ['clay_hash' [%%s (scot %%uv digest)]] ['publication' [%%s ?:(private 'private' 'published')]]])]
`+"`@ux`(en:json:html value)", ship, desk)
	return LensRequest{expression: expression}, nil
}
func Create(ship, desk string) (LensRequest, error) {
	if !validSelection(ship, desk) {
		return LensRequest{}, errors.New("invalid desk selection")
	}
	expression := fmt.Sprintf(`=/  wen  (scot %%da now)
?>  !(~(has in .^((set @tas) %%cd /%s//[wen])) %%%s)
=/  paths=(list path)  ~[/mar/noun/hoon /mar/hoon/hoon /mar/txt/hoon /mar/kelvin/hoon /sys/kelvin]
=/  files  (turn paths |=(p=path [p [(rear p) .^(* %%cx (weld /%s/base/[wen] p))]]))
(new-desk:cloy %%%s ~ (~(gas by *(map path page:clay)) files))`, ship, desk, ship, desk)
	return LensRequest{expression: expression, app: "hood", mark: "helm-pass"}, nil
}
func Mount(ship, desk, alias string, revision uint64) (LensRequest, error) {
	if !validSelection(ship, desk) || !deskName.MatchString(alias) || revision > 1<<53-1 {
		return LensRequest{}, errors.New("invalid desk mount")
	}
	at := "(scot %da now)"
	if revision != 0 {
		at = hoonDecimal(revision)
	}
	return LensRequest{expression: fmt.Sprintf("+hood/mount /%s/%s/%s %%%s", ship, desk, at, alias), app: "hood"}, nil
}
func Commit(alias string) (LensRequest, error) {
	if !deskName.MatchString(alias) {
		return LensRequest{}, errors.New("invalid desk mount")
	}
	return LensRequest{expression: "[%" + alias + " %.n]", app: "hood", mark: "kiln-commit"}, nil
}
func PublishSource(desk string) (LensRequest, error) {
	if !deskName.MatchString(desk) {
		return LensRequest{}, errors.New("invalid desk")
	}
	return LensRequest{expression: "+hood/public %" + desk, app: "hood"}, nil
}

// DecodeJSONOutput reads JSON rendered as a hex atom, avoiding Hoon display
// quoting, line wrapping, aura guessing and an unbounded general noun decoder.
func DecodeJSONOutput(body []byte, out any) error {
	if len(body) > 2<<20 {
		return errors.New("desk observation exceeds limit")
	}
	var display string
	if err := json.Unmarshal(body, &display); err != nil {
		return errors.New("invalid Lens response")
	}
	display = strings.TrimSpace(display)
	if !strings.HasPrefix(display, "0x") {
		return errors.New("desk observation is not a hex atom")
	}
	digits := strings.TrimPrefix(display, "0x")
	for _, c := range digits {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c == '.') {
			return errors.New("invalid desk observation encoding")
		}
	}
	digits = strings.ReplaceAll(digits, ".", "")
	if len(digits)%2 != 0 {
		digits = "0" + digits
	}
	raw, err := hex.DecodeString(digits)
	if err != nil {
		return err
	}
	for i, j := 0, len(raw)-1; i < j; i, j = i+1, j-1 {
		raw[i], raw[j] = raw[j], raw[i]
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err = d.Decode(out); err != nil {
		return err
	}
	if d.More() {
		return errors.New("trailing desk observation")
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return errors.New("trailing desk observation")
	}
	return nil
}

// Hoon's textual numeric syntax requires grouping beyond three decimal digits.
func hoonDecimal(n uint64) string {
	text := strconv.FormatUint(n, 10)
	first := len(text) % 3
	if first == 0 {
		first = 3
	}
	out := text[:first]
	for i := first; i < len(text); i += 3 {
		out += "." + text[i:i+3]
	}
	return out
}

// MarkEquivalent checks an exact historical Clay revision using that desk's
// mark converter. Source bytes travel as data, never interpolated Hoon code.
// Vere's Unix sync sends every file as text/plain MIME, including binary data.
func MarkEquivalent(ship, desk string, revision uint64, name string, raw []byte) (LensRequest, error) {
	if !validSelection(ship, desk) || revision == 0 || revision > 1<<53-1 || !ValidPath(name) || int64(len(raw)) > MaxFileBytes {
		return LensRequest{}, errors.New("invalid mark comparison")
	}
	parts := strings.Split(name, "/")
	base := parts[len(parts)-1]
	dot := strings.LastIndexByte(base, '.')
	fileMark := base[dot+1:]
	parts[len(parts)-1] = base[:dot]
	parts = append(parts, fileMark)
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = "`@ta`'" + p + "'"
	}
	filePath := "(weld /" + ship + "/" + desk + "/" + hoonDecimal(revision) + " ~[" + strings.Join(quoted, " ") + "])"
	expression := "|=  encoded=@t\n=/  file  " + filePath + "\n?.  .^(? %cu file)  %.n\n=/  raw=octs  ?:(=('' encoded) [0 0] (need (de:base64:mimes:html encoded)))\n=/  convert  .^(tube:clay %cc /" + ship + "/" + desk + "/" + hoonDecimal(revision) + "/mime/" + fileMark + ")\n=(q:(convert !>(`mime`[/text/plain raw])) .^(* %cx file))"
	data := base64.StdEncoding.EncodeToString(raw)
	return LensRequest{expression: expression, input: &data}, nil
}
