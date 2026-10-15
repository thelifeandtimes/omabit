// Package access binds restricted OpenSSH forced commands to public-key grants.
package access

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
)

const Header = "Urbit-Runner-SSH-Grant"
const DefaultKeys = "/etc/urbit-runner/authorized_keys"

type Grant struct{ Fingerprint, Role string }

var ErrDenied = errors.New("SSH grant is absent, revoked, or invalid")
var linePattern = regexp.MustCompile(`^restrict,command="(?:urbit-runner-rpc-v1|/usr/local/lib/urbit-runner/current/urbit-runner owner-ssh) (SHA256:[A-Za-z0-9+/]{43}) (operator|observer)" ssh-ed25519 ([A-Za-z0-9+/]+=*)$`)

func OwnerAuthorizedKey(public []byte, role string) (string, error) {
	line, err := AuthorizedKey(public, role)
	return strings.Replace(line, `command="`+client.SSHCommand+` `, `command="`+ShellPath+` owner-ssh `, 1), err
}

func fingerprint(encoded string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 51 || binary.BigEndian.Uint32(key[:4]) != 11 || string(key[4:15]) != "ssh-ed25519" || binary.BigEndian.Uint32(key[15:19]) != 32 {
		return "", errors.New("require a valid Ed25519 SSH public key")
	}
	sum := sha256.Sum256(key)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), nil
}

func AuthorizedKey(public []byte, role string) (string, error) {
	if strings.ContainsAny(strings.TrimSpace(string(public)), "\r\n") {
		return "", errors.New("require exactly one SSH public key line")
	}
	fields := strings.Fields(string(public))
	if len(public) > 8192 || len(fields) < 2 || fields[0] != "ssh-ed25519" || (role != "operator" && role != "observer") {
		return "", errors.New("require an Ed25519 public key and role operator or observer")
	}
	id, err := fingerprint(fields[1])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("restrict,command=\"%s %s %s\" ssh-ed25519 %s", client.SSHCommand, id, role, fields[1]), nil
}

func ParseLine(line string) (Grant, error) {
	m := linePattern.FindStringSubmatch(line)
	if m == nil {
		return Grant{}, ErrDenied
	}
	id, err := fingerprint(m[3])
	if err != nil || id != m[1] {
		return Grant{}, ErrDenied
	}
	return Grant{Fingerprint: id, Role: m[2]}, nil
}

// Load rechecks revocation on every request. The file contains public keys only.
// Root or the service account owns it; group/other writable files are refused.
func Load(path, id string) (Grant, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return Grant{}, ErrDenied
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return Grant{}, ErrDenied
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 || st.Nlink != 1 || (st.Uid != 0 && st.Uid != uint32(os.Geteuid())) {
		return Grant{}, ErrDenied
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(data) > 65536 {
		return Grant{}, ErrDenied
	}
	var found Grant
	seen := map[string]bool{}
	for _, raw := range bytes.Split(data, []byte{'\n'}) {
		line := strings.TrimSpace(string(raw))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		grant, err := ParseLine(line)
		if err != nil || seen[grant.Fingerprint] {
			return Grant{}, ErrDenied
		}
		seen[grant.Fingerprint] = true
		if grant.Fingerprint == id {
			found = grant
		}
	}
	if found.Fingerprint == "" {
		return Grant{}, ErrDenied
	}
	return found, nil
}

func (g Grant) Allows(method, path string) bool {
	return g.Role == "operator" || (g.Role == "observer" && method == "GET" && (path == "/hello" || path == "/v1/snapshot"))
}
