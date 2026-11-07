package deskwork

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// FrontendGlobHash evaluates the same map and hash as Docket without an effect.
// Bytes are input data; neither asset names nor contents become executable Hoon.
func FrontendGlobHash(m FrontendManifest, files []SourceFile) (LensRequest, error) {
	if err := validateFrontendFiles(m, files); err != nil {
		return LensRequest{}, err
	}
	rows := make([]any, 0, len(files))
	for i, f := range files {
		p := strings.Split(f.Path, "/")
		last := p[len(p)-1]
		dot := strings.LastIndexByte(last, '.')
		p[len(p)-1] = last[:dot]
		p = append(p, last[dot+1:])
		// Native multipart +nab retains the complete header in one path atom.
		// Splitting it at '/' serves the same MIME text but hashes a different noun.
		rows = append(rows, []any{p, m.Files[i].ContentType, base64.StdEncoding.EncodeToString(f.Bytes)})
	}
	data, _ := json.Marshal(rows)
	encoded := string(data)
	expression := `|=  encoded=@t
=/  rows  ((ar:dejs:format same) (need (de:json:html encoded)))
?>  (lte (lent rows) 2.048)
=/  entries=(list [path mime])
  %+  turn  rows
  |=  row=json
  =/  fields  ((ar:dejs:format same) row)
  ?>  ?=([* * * ~] fields)
  =/  p=path  ((ar:dejs:format so:dejs:format) i.fields)
  =/  p=path  (turn p |=(s=@ta ` + "`@ta`" + `(crip (en-urlt:html (trip s)))))
  =/  type  (so:dejs:format i.t.fields)
  =/  bytes  (so:dejs:format i.t.t.fields)
  =/  raw=octs  ?:(=('' bytes) [0 0] (need (de:base64:mimes:html bytes)))
  [p [[` + "`@ta`type" + ` ~] raw]]
=/  hash  (sham (~(gas by *(map path mime)) entries))
` + "`@ux`(en:json:html [%s (scot %uv hash)])"
	return LensRequest{expression: expression, input: &encoded}, nil
}

func FrontendInventory(ship, desk string) (LensRequest, error) {
	if !validSelection(ship, desk) {
		return LensRequest{}, errors.New("invalid frontend selection")
	}
	expression := fmt.Sprintf(`=/  ego  (scot %%p %s)
=/  syd  %%%s
=/  wen  (scot %%da now)
=/  pre  /[ego]/[syd]/[wen]
?>  (~(has in .^((set @tas) %%cd /[ego]//[wen])) syd)
=/  revision  ud:.^(cass:clay %%cw pre)
=/  dock
  $:  %%1
      title=@t
      info=@t
      color=@ux
      href=$%%([%%site =path] [%%glob base=@tas hash=@uv location=$%%([%%http url=@t] [%%ames ship=@p])])
      image=(unit @t)
      version=[@ud @ud @ud]
      website=@t
      license=@t
  ==
=/  declared=(unit dock)
  ?.(.^(? %%cu (weld pre /desk/docket-0)) ~ [~ ;;(dock .^(* %%cx (weld pre /desk/docket-0)))])
=/  available  .^(? %%gu /[ego]/docket/[wen]/$)
=/  charge=(unit [docket=* chad=[tag=@tas data=*]])
  ?.  available  ~
  =/  charges  ;;([%%initial entries=(map @tas [docket=* chad=[tag=@tas data=*]])] .^(* %%gx /[ego]/docket/[wen]/charges/noun))
  ?>  (lte (lent ~(tap by entries.charges)) 8.192)
  (~(get by entries.charges) syd)
=/  reference
  |=  selected=(unit dock)
  ^-  json
  =/  mode
    ?~  selected  'none'
    ?-  -.href.u.selected
      %%site  'site'
      %%glob  ?:(?=(%%ames -.location.href.u.selected) 'glob-ames' 'glob-http')
    ==
  =/  base=@t  ?~(selected '' ?.(?=(%%glob -.href.u.selected) '' (scot %%tas base.href.u.selected)))
  =/  ship=@t  ?~(selected '' ?.(?=(%%glob -.href.u.selected) '' ?.(?=(%%ames -.location.href.u.selected) '' (scot %%p ship.location.href.u.selected))))
  =/  hash=@t  ?~(selected '' ?.(?=(%%glob -.href.u.selected) '' (scot %%uv hash.href.u.selected)))
  [%%o (malt `+"`(list [@t json])`"+`~[['mode' [%%s mode]] ['base' [%%s base]] ['ship' [%%s ship]] ['glob_hash' [%%s hash]] ['docket_hash' [%%s ?~(selected '' (scot %%uv (sham u.selected)))]]])]
=/  values=(list [@t json])
:~  ['source_revision' [%%n (scot %%ud revision)]]
    ['docket_available' [%%b available]]
    ['stage' [%%s ?.(available 'unavailable' ?~(charge 'absent' (scot %%tas tag.chad.u.charge)))]]
    ['declared' (reference declared)]
    ['current' (reference ?~(charge ~ [~ ;;(dock docket.u.charge)]))]
==
`+"`@ux`(en:json:html [%%o (malt values)])", ship, desk)
	return LensRequest{expression: expression}, nil
}
