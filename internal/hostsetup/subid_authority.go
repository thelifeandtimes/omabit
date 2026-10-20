package hostsetup

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
)

const SubIDReservationsPath = "/etc/urbit-runner/subid-reservations.json"

// Fail closed for compat, systemd dynamic users, sss, ldap, winbind, unknown
// plugins, conditional actions, and missing/duplicate entries. getent enumeration
// cannot establish completeness for external/non-enumerating identity domains.
func LocalIdentityAuthority(nss string) bool {
	return nssFiles(nss, "passwd", false) && nssFiles(nss, "group", false)
}

func FileSubIDAuthority(nss string) bool { return nssFiles(nss, "subid", true) }

func nssFiles(text, database string, defaultFiles bool) bool {
	seen := false
	for _, line := range strings.Split(text, "\n") {
		line, _, _ = strings.Cut(line, "#")
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != database {
			continue
		}
		if seen || strings.TrimSpace(value) != "files" {
			return false
		}
		seen = true
	}
	return seen || defaultFiles
}

type ReservedSubIDRange struct {
	Start uint64 `json:"start"`
	Count uint64 `json:"count"`
}

// Authority is the administrator's explicit assertion that these ranges are
// reserved outside ALL identity domains. It is not inferred from NSS enumeration.
type SubIDReservations struct {
	Version   int                `json:"version"`
	Authority string             `json:"authority"`
	UID       ReservedSubIDRange `json:"uid"`
	GID       ReservedSubIDRange `json:"gid"`
}

func ParseSubIDReservations(data []byte) (SubIDReservations, error) {
	var r SubIDReservations
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF || r.Version != 1 || strings.TrimSpace(r.Authority) == "" || len(r.Authority) > 1024 {
		return r, errors.New("require explicit version 1 subordinate-ID reservations and authority")
	}
	for _, span := range []ReservedSubIDRange{r.UID, r.GID} {
		if span.Start == 0 || span.Count < 65536 || span.Start >= 1<<32-1 || span.Count > (1<<32-1)-span.Start {
			return r, errors.New("invalid reserved subordinate-ID range")
		}
	}
	return r, nil
}

func ValidateReservedSubIDs(name, text, uid string, ids []uint64, span ReservedSubIDRange) (string, string) {
	status, detail := ValidateSubIDs(name, text, uid, ids)
	if status != "pass" {
		return status, detail
	}
	for _, line := range strings.Split(text, "\n") {
		f := strings.Split(strings.TrimSpace(line), ":")
		if len(f) != 3 || f[0] != name && f[0] != uid {
			continue
		}
		a, _ := number(f[1])
		n, _ := number(f[2])
		if a < span.Start || a+n > span.Start+span.Count {
			return "blocked", "Configured subordinate IDs fall outside the administrator-reserved identity-domain range."
		}
	}
	return "pass", "Configured subordinate IDs fit the explicit administrator-reserved range and local collision checks."
}

func ValidateSubIDAuthority(nss string, reservations []byte, name, text, uid, kind string, ids []uint64) (string, string) {
	if !FileSubIDAuthority(nss) {
		return "blocked", "Subordinate-ID authority is external; file mappings are not authoritative."
	}
	if LocalIdentityAuthority(nss) {
		return ValidateSubIDs(name, text, uid, ids)
	}
	r, err := ParseSubIDReservations(reservations)
	if err != nil {
		return "blocked", "External or unknown NSS requires explicitly reserved, preconfigured subordinate-ID ranges; getent enumeration cannot prove safety."
	}
	span := r.UID
	if kind == "subgid" {
		span = r.GID
	}
	return ValidateReservedSubIDs(name, text, uid, ids, span)
}

// ValidateOwnerSubIDAuthority permits read-only reuse of existing OS delegation
// for a bound local account or an installed owner profile. This is distinct from
// inventing new ranges: unknown/external identity domains still cannot authorize
// allocation through incomplete enumeration. Privileged callers must recheck the
// root ownership/permissions of these files and the selected account/home binding.
func ValidateOwnerSubIDAuthority(nss string, reservations []byte, reservationErr error, reuse bool, name, text, uid, kind string, ids []uint64) (string, string) {
	if !FileSubIDAuthority(nss) || LocalIdentityAuthority(nss) {
		return ValidateSubIDAuthority(nss, reservations, name, text, uid, kind, ids)
	}
	if reservationErr == nil {
		// An explicit policy remains authoritative, including malformed/conflicting
		// policies. Reuse must not silently bypass it.
		return ValidateSubIDAuthority(nss, reservations, name, text, uid, kind, ids)
	}
	if !errors.Is(reservationErr, os.ErrNotExist) {
		return "blocked", "The configured user-namespace ID policy could not be read; have the host administrator review it."
	}
	if reuse {
		status, detail := ValidateSubIDs(name, text, uid, ids)
		if status == "pass" {
			return "pass", "Existing OS-assigned user-namespace IDs can be reused without allocation. Known local collisions and grant overlaps were checked; this does not claim complete external NSS enumeration."
		}
		if status == "missing" {
			return "blocked", "This account needs at least 65536 OS-assigned user-namespace IDs in both /etc/subuid and /etc/subgid. Existing ranges are preserved; new ranges on this identity configuration require the host administrator's reserved authority."
		}
		return status, detail
	}
	return ValidateSubIDAuthority(nss, reservations, name, text, uid, kind, ids)
}
