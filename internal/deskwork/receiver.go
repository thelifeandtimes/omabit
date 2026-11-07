package deskwork

import (
	"errors"
	"fmt"
)

// Publisher identifies a continuing Urbit subscription, not an HTTP origin.
// ExpectedHash is the source version to verify after installation. Kiln follows
// the publisher's later releases; its install acknowledgement does not pin a
// version or prove that any source has arrived.
type Publisher struct {
	Ship         string `json:"ship"`
	Desk         string `json:"desk"`
	ExpectedHash string `json:"expected_hash"`
}

func (p Publisher) Validate() error {
	if !validSelection(p.Ship, p.Desk) || (Observation{Exists: true, Revision: 1, ClayHash: p.ExpectedHash, Publication: "published"}).Validate() != nil {
		return errors.New("invalid selected publisher")
	}
	return nil
}

// Subscription is read even when Clay has not received the desk yet. An absent
// Clay desk is insufficient to prove that a previous installation is absent.
type Subscription struct {
	Present bool   `json:"present"`
	Ship    string `json:"ship"`
	Desk    string `json:"desk"`
	Zest    string `json:"zest"`
	Hash    string `json:"hash"`
}

func (s Subscription) Validate() error {
	if !s.Present {
		if s.Ship != "" || s.Desk != "" || s.Zest != "" || s.Hash != "" {
			return errors.New("invalid absent subscription")
		}
		return nil
	}
	if s.Zest != "dead" && s.Zest != "held" && s.Zest != "live" {
		return errors.New("invalid subscription state")
	}
	if (s.Ship == "") != (s.Desk == "") || s.Ship != "" && !validSelection(s.Ship, s.Desk) || (Observation{Exists: true, Revision: 1, ClayHash: s.Hash, Publication: "private"}).Validate() != nil {
		return errors.New("invalid subscription observation")
	}
	return nil
}
func SubscriptionInventory(ship, desk string) (LensRequest, error) {
	if !validSelection(ship, desk) {
		return LensRequest{}, errors.New("invalid subscription selection")
	}
	expression := fmt.Sprintf(`=/  ego  (scot %%p %s)
=/  pikes  .^((map @tas [sync=(unit [ship=@p desk=@tas]) hash=@uv zest=zest:clay wic=(set weft)]) %%gx /[ego]/hood/(scot %%da now)/kiln/pikes/noun)
=/  pike  (~(get by pikes) %%%s)
=/  origin=(unit [ship=@p desk=@tas])  ?~(pike ~ sync.u.pike)
=/  values=(list [@t json])
:~  ['present' [%%b !?=(~ pike)]]
    ['ship' [%%s ?~(origin '' (scot %%p ship.u.origin))]]
    ['desk' [%%s ?~(origin '' (scot %%tas desk.u.origin))]]
    ['zest' [%%s ?~(pike '' (scot %%tas zest.u.pike))]]
    ['hash' [%%s ?~(pike '' (scot %%uv hash.u.pike))]]
==
`+"`@ux`(en:json:html [%%o (malt values)])", ship, desk)
	return LensRequest{expression: expression}, nil
}

// ReceiverSourceReady distinguishes Kiln's reserved, revision-zero Clay desk
// while downloading from an arrived source revision. It does not redefine that
// reservation as absence or permit another install into the occupied name.
func ReceiverSourceReady(ship, desk string) (LensRequest, error) {
	if !validSelection(ship, desk) {
		return LensRequest{}, errors.New("invalid receiver selection")
	}
	expression := fmt.Sprintf(`=/  ego  (scot %%p %s)
=/  wen  (scot %%da now)
=/  present  (~(has in .^((set @tas) %%cd /[ego]//[wen])) %%%s)
=/  ready  ?:(present (gth let:.^(domo:clay %%cv /[ego]/%s/[wen]) 0) %%.n)
`+"`@ux`(en:json:html [%%b ready])", ship, desk, desk)
	return LensRequest{expression: expression}, nil
}

// InstallRemote creates a subscription only in an unused local name. It never
// replaces an existing Clay desk or a pending Kiln source. Callers must enforce
// creation and commit policy, journal once, then observe receipt of source.
func InstallRemote(ship, desk string, p Publisher) (LensRequest, error) {
	if !validSelection(ship, desk) || p.Validate() != nil || ship == p.Ship {
		return LensRequest{}, errors.New("invalid remote installation selection")
	}
	expression := fmt.Sprintf(`=/  ego  (scot %%p %s)
=/  wen  (scot %%da now)
?>  !(~(has in .^((set @tas) %%cd /[ego]//[wen])) %%%s)
=/  pikes  .^((map @tas [sync=(unit [ship=@p desk=@tas]) hash=@uv zest=zest:clay wic=(set weft)]) %%gx /[ego]/hood/[wen]/kiln/pikes/noun)
?>  !(~(has by pikes) %%%s)
[%%%s %s %%%s]`, ship, desk, desk, desk, p.Ship, p.Desk)
	return LensRequest{expression: expression, app: "hood", mark: "kiln-install"}, nil
}

// VerifyReceiver reports the requested source at a live subscription. It proves
// neither compiled-agent revision nor frontend/application health. A newer hash
// is not silently accepted as the reviewed release.
func VerifyReceiver(p Publisher, o Observation, app AppState) error {
	if p.Validate() != nil || o.Validate() != nil || !o.Exists || o.Publication != "private" || app.Validate() != nil || o.ClayHash != p.ExpectedHash || app.SourceRevision != o.Revision || app.SourceShip != p.Ship || app.SourceDesk != p.Desk || app.Zest != "live" {
		return errors.New("selected publisher source has not been verified on receiver")
	}
	return nil
}
