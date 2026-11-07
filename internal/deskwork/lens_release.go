package deskwork

import (
	"errors"
	"fmt"
)

// AppInventory uses Gall liveness before agent scries, and selects the current
// desk only. Missing/changed protocol structures fail rather than imply health.
// Read variant agent responses as nouns, then check their value shape. Asking
// a scry for only one variant of the mark's union can leave Lens waiting.
// Treaty requires its sovereign entry as well as alliance membership: its public
// treaty-map scry does not reliably contain its own published desks.
func AppInventory(ship, desk string) (LensRequest, error) {
	if !validSelection(ship, desk) {
		return LensRequest{}, errors.New("invalid app selection")
	}
	expression := fmt.Sprintf(`=/  ego  (scot %%p %s)
=/  syd  %%%s
=/  wen  (scot %%da now)
=/  pre  /[ego]/[syd]/[wen]
?>  (~(has in .^((set @tas) %%cd /[ego]//[wen])) syd)
=/  pikes  .^((map @tas [sync=(unit [ship=@p desk=@tas]) hash=@uv zest=zest:clay wic=(set weft)]) %%gx /[ego]/hood/[wen]/kiln/pikes/noun)
=/  pike  (~(get by pikes) syd)
=/  zest  ?~(pike 'absent' (scot %%tas zest.u.pike))
=/  origin=(unit [ship=@p desk=@tas])  ?~(pike ~ sync.u.pike)
=/  revision  ud:.^(cass:clay %%cw pre)
=/  expected=(list @tas)
  ?:(.^(? %%cu (weld pre /desk/bill)) .^((list @tas) %%cx (weld pre /desk/bill)) ~)
?>  (lte (lent expected) 256)
=/  agents  ~(tap in .^((set [name=@tas live=?]) %%ge /[ego]/[syd]/[wen]/$))
?>  (lte (lent agents) 256)
=/  treaty-live  .^(? %%gu /[ego]/treaty/[wen]/$)
=/  treaty-hash=@t
  ?.  treaty-live  ''
  =/  alliance  ;;([%%ini entries=(set [@p @tas])] .^(* %%gx /[ego]/treaty/[wen]/alliance/noun))
  ?.  (~(has in entries.alliance) [%s syd])  ''
  =/  state  .^(vase %%gx /[ego]/treaty/[wen]/dbug/state/noun)
  =/  selected  (slap (slop state !>(..zuse)) (ream '=/  row  (~(got by sovereign) %%%s)  ?>  =([%s %%%s] [ship desk]:row)  (scot %%uv hash.row)'))
  ;;(@t q.selected)
=/  docket-live  .^(? %%gu /[ego]/docket/[wen]/$)
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
=/  docket=(unit dock)
  ?.  .^(? %%cu (weld pre /desk/docket-0))  ~
  [~ ;;(dock .^(* %%cx (weld pre /desk/docket-0)))]
=/  frontend
  ?~  docket  'none'
  ?-  -.href.u.docket
    %%site  'site'
    %%glob  ?:(?=(%%ames -.location.href.u.docket) 'glob-ames' 'glob-http')
  ==
=/  charge=(unit [docket=* chad=[tag=@tas data=*]])
  ?.  docket-live  ~
  =/  charges  ;;([%%initial entries=(map @tas [docket=* chad=[tag=@tas data=*]])] .^(* %%gx /[ego]/docket/[wen]/charges/noun))
  ?>  (lte (lent ~(tap by entries.charges)) 8.192)
  (~(get by entries.charges) syd)
=/  frontend-state
  ?.  docket-live  'unavailable'
  ?~(charge 'absent' (scot %%tas tag.chad.u.charge))
=/  values=(list [@t json])
:~  ['zest' [%%s zest]]
    ['source_revision' [%%n (scot %%ud revision)]]
    ['source_ship' [%%s ?~(origin '' (scot %%p ship.u.origin))]]
    ['source_desk' [%%s ?~(origin '' (scot %%tas desk.u.origin))]]
    ['expected_agents' [%%a (turn expected |=(name=@tas [%%s (scot %%tas name)]))]]
    ['agents' [%%a (turn agents |=([name=@tas live=?] [%%o (malt `+"`(list [@t json])`"+`~[['name' [%%s (scot %%tas name)]] ['live' [%%b live]]])]))]]
    ['treaty_available' [%%b treaty-live]]
    ['treaty_listed' [%%b !=('' treaty-hash)]]
    ['treaty_hash' [%%s treaty-hash]]
    ['docket_available' [%%b docket-live]]
    ['docket_hash' [%%s ?~(docket '' (scot %%uv (sham u.docket)))]]
    ['frontend_mode' [%%s frontend]]
    ['frontend_state' [%%s frontend-state]]
    ['frontend_docket_hash' [%%s ?~(charge '' (scot %%uv (sham docket.u.charge)))]]
==
`+"`@ux`(en:json:html [%%o (malt values)])", ship, desk, ship, desk, ship, desk)
	return LensRequest{expression: expression}, nil
}

func releasePrecondition(ship, desk string, baseline Observation) (string, error) {
	if !validSelection(ship, desk) || baseline.Validate() != nil || !baseline.Exists {
		return "", errors.New("invalid reviewed release selection")
	}
	return fmt.Sprintf("=/  pre  /%s/%s/(scot %%da now)\n?>  =(%s let:.^(domo:clay %%cv pre))\n?>  =(%s .^(@uvI %%cz pre))\n", ship, desk, hoonDecimal(baseline.Revision), baseline.ClayHash), nil
}
func PublishSourceAt(ship, desk string, baseline Observation) (LensRequest, error) {
	pre, err := releasePrecondition(ship, desk, baseline)
	if err != nil {
		return LensRequest{}, err
	}
	return LensRequest{expression: pre + fmt.Sprintf("[%%%s / &]", desk), app: "hood", mark: "kiln-permission"}, nil
}
func PublishTreatyAt(ship, desk string, baseline Observation) (LensRequest, error) {
	pre, err := releasePrecondition(ship, desk, baseline)
	if err != nil {
		return LensRequest{}, err
	}
	return LensRequest{expression: pre + fmt.Sprintf("?>  .^(? %%gu /%s/treaty/(scot %%da now)/$)\n?>  .^(? %%cu (weld pre /desk/docket-0))\n[%%add %s %%%s]", ship, ship, desk), app: "treaty", mark: "alliance-update-0"}, nil
}

// ActivateLocal installs only from the same ship/desk. Remote subscriptions need
// their own explicit source selection and receiver policy, never this shortcut.
func ActivateLocal(ship, desk string, baseline Observation) (LensRequest, error) {
	pre, err := releasePrecondition(ship, desk, baseline)
	if err != nil {
		return LensRequest{}, err
	}
	return LensRequest{expression: pre + fmt.Sprintf("[%%%s %s %%%s]", desk, ship, desk), app: "hood", mark: "kiln-install"}, nil
}
