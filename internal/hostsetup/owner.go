package hostsetup

import (
	"os"
	"os/user"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
)

func LocalIDs(passwd, groups string) ([]uint64, []uint64) {
	var uids, gids []uint64
	for _, line := range strings.Split(passwd, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) != 7 {
			return nil, nil
		}
		u, a := strconv.ParseUint(f[2], 10, 32)
		g, b := strconv.ParseUint(f[3], 10, 32)
		if a != nil || b != nil {
			return nil, nil
		}
		uids = append(uids, u)
		gids = append(gids, g)
	}
	for _, line := range strings.Split(groups, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) != 4 {
			return nil, nil
		}
		g, e := strconv.ParseUint(f[2], 10, 32)
		if e != nil {
			return nil, nil
		}
		gids = append(gids, g)
	}
	return uids, gids
}

// LocalOwnerBinding identifies an already existing local login account. Consuming
// its OS-delegated file grants does not require Omabit to have installed a profile
// first, nor does it imply that external NSS enumeration is complete.
func LocalOwnerBinding(passwd string, p ownerhost.Profile) bool {
	if p.Validate() != nil {
		return false
	}
	seen := false
	for _, line := range strings.Split(passwd, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) != 7 {
			return false
		}
		if fields[0] != p.Name {
			continue
		}
		uid, uidErr := strconv.ParseUint(fields[2], 10, 32)
		gid, gidErr := strconv.ParseUint(fields[3], 10, 32)
		if seen || uidErr != nil || gidErr != nil || uid != uint64(p.UID) || gid != uint64(p.GID) || fields[5] != p.Home {
			return false
		}
		seen = true
	}
	return seen
}

func InspectCurrentOwner() Report {
	u, err := user.Current()
	if err != nil {
		return Report{Deployment: "owner-user-rootless", Checks: []Check{{"owner-account", "blocked", err.Error()}}}
	}
	return InspectOwner(u.Username)
}

func InspectOwner(name string) Report {
	base := Inspect()
	r := Report{SchemaVersion: 1, Deployment: "owner-user-rootless", Compatibility: base.Compatibility}
	for _, c := range base.Checks {
		if c.ID == "binary-useradd" {
			continue
		}
		c.Detail = strings.ReplaceAll(c.Detail, "dedicated-account", "owner-user")
		if c.ID == "platform" || c.ID == "systemd" || c.ID == "cgroup-v2" || c.ID == "userns-clone" || c.ID == "userns-limit" || c.ID == "docker-version" || strings.HasPrefix(c.ID, "binary-") {
			r.Checks = append(r.Checks, c)
		}
	}
	p, err := ownerhost.ForUser(name)
	if err != nil {
		r.add("owner-account", "blocked", err.Error())
		return r
	}
	r.add("owner-account", "pass", "Runner and rootless containers will run as "+p.Name+" (UID "+strconv.FormatUint(uint64(p.UID), 10)+").")
	installed, err := ownerhost.Load()
	preserve := err == nil && installed != nil && installed.Version == p.Version && installed.Name == p.Name && installed.UID == p.UID && installed.GID == p.GID && installed.Home == p.Home
	if err != nil || installed != nil && (installed.Name != p.Name || installed.UID != p.UID || installed.GID != p.GID || installed.Home != p.Home) {
		r.add("account", "blocked", "Installed owner profile is invalid or belongs to another user; it was preserved.")
	} else if installed != nil {
		r.add("account", "pass", "Owner-user installation profile exists.")
	} else if _, err := os.Lstat("/var/lib/urbit-runner"); err == nil {
		r.add("account", "blocked", "Legacy dedicated-account installation must be explicitly retired or migrated first.")
	} else {
		r.add("account", "missing", "No owner-user installation yet; setup creates its services and user-owned pier root.")
	}
	passwd, e1 := os.ReadFile("/etc/passwd")
	groups, e2 := os.ReadFile("/etc/group")
	nss, _ := os.ReadFile("/etc/nsswitch.conf")
	reservations, reservationErr := os.ReadFile(SubIDReservationsPath)
	uids, gids := LocalIDs(string(passwd), string(groups))
	if e1 != nil || e2 != nil {
		uids, gids = nil, nil
	}
	reuse := preserve || e1 == nil && LocalOwnerBinding(string(passwd), p)
	if reuse && uids != nil && gids != nil {
		// Include the selected owner's known IDs explicitly. An installed owner
		// may be absent from local files; this does not enumerate external NSS.
		uids, gids = append(uids, 0, uint64(p.UID)), append(gids, 0, uint64(p.GID))
	}
	for _, item := range []struct {
		name string
		ids  []uint64
	}{{"subuid", uids}, {"subgid", gids}} {
		data, err := os.ReadFile("/etc/" + item.name)
		status, detail := "unknown", "Cannot read subordinate ID allocation."
		if err == nil {
			status, detail = ValidateOwnerSubIDAuthority(string(nss), reservations, reservationErr, reuse, p.Name, string(data), strconv.FormatUint(uint64(p.UID), 10), item.name, item.ids)
		}
		if status == "missing" && preserve {
			detail = "Installed-owner grants need explicit review; management updates preserve existing ranges and never allocate replacements."
		} else if status == "missing" && LocalIdentityAuthority(string(nss)) {
			detail = "Host setup will allocate a non-overlapping range for this existing user."
		}
		detail = strings.ReplaceAll(strings.ReplaceAll(detail, "Dedicated account", "Selected user"), "dedicated-account", "owner-user")
		r.add(item.name, status, detail)
	}
	return r
}
