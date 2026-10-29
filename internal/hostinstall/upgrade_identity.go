package hostinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

// A GUI update may resume while the manager is down, but only when the root-owned
// activation journal already binds this exact owner, runner and executable.
func (i *installer) checkUpgradeIdentity(ctx context.Context, p ownerhost.Profile, expected string, binary []byte, probe hostProbe) error {
	if !strings.HasPrefix(expected, "runner-") || len(expected) > 80 {
		return errors.New("invalid expected runner identity")
	}
	data, err := i.readInstalled(ctx, activationJournalPath, 16384)
	if err == nil {
		var j activationJournal
		if json.Unmarshal(data, &j) != nil || j.Version != 1 || j.Owner != p.Name || j.UID != p.UID || j.RunnerID != expected || !validDigest(j.Token) || !j.Target.valid() || j.Previous != nil && !j.Previous.valid() {
			return errors.New("saved activation belongs to a different runner; preserving it for review")
		}
		if j.Stage != "complete" && j.Stage != "rolled-back" {
			if j.Offline || j.Target.SHA256 != digestBytes(binary) || !validDigest(j.Token) || !j.Target.valid() {
				return errors.New("retry requires the original installer and supported activation journal")
			}
			switch j.Stage {
			case "prepared", "quiesced", "stopping", "starting", "verified", "rollback", "rollback-verified":
				return nil
			}
			return errors.New("unknown activation stage; preserve for review")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	hello, busy, err := probe(ctx, fmt.Sprintf("/run/user/%d/urbit-runner/runner.sock", p.UID))
	if err != nil || busy || hello.RunnerID != expected || hello.Mode != "development-managed" {
		return errors.New("existing runner is unavailable, busy or changed; finish accepted work and check again")
	}
	return nil
}
