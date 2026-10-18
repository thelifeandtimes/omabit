// Package hostsetup provides read-only preflight evidence and a non-executable
// setup plan. Neither filesystem metadata nor a plan qualifies a runtime.
package hostsetup

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strconv"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

const deployment = "dedicated-account-rootless"
const maxRead = 1 << 20

type Check struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type Report struct {
	SchemaVersion    int     `json:"schema_version"`
	Deployment       string  `json:"deployment"`
	RuntimeQualified bool    `json:"runtime_qualified"`
	Checks           []Check `json:"checks"`
}

// Inspect reads local metadata only. It never executes programs, connects to
// sockets, reads Docker configuration, or treats the desktop user as the target.
func Inspect() Report {
	_, host := os.LookupEnv("DOCKER_HOST")
	_, context := os.LookupEnv("DOCKER_CONTEXT")
	return (probe{os.DirFS("/"), func(path string) (fs.FileInfo, error) {
		return os.Lstat("/" + path)
	}, runtime.GOOS, runtime.GOARCH, host, context}).inspect()
}

// Tests inject all host evidence, including lstat and environment presence.
type probe struct {
	files                     fs.FS
	lstat                     func(string) (fs.FileInfo, error)
	goos, goarch              string
	dockerHost, dockerContext bool
}

func (r *Report) add(id, status, detail string) {
	r.Checks = append(r.Checks, Check{id, status, detail})
}

func unavailable(err error) (string, string) {
	if errors.Is(err, fs.ErrNotExist) {
		return "missing", "Absent; setup is required."
	}
	return "unknown", "Evidence is unreadable, non-regular, oversized, or malformed."
}

// Check type before opening (notably FIFOs/devices), and bound actual reads even
// when a proc/sys file reports size zero. Host evidence is not a race-free audit.
func (p probe) read(path string) (string, error) {
	info, err := fs.Stat(p.files, path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxRead {
		return "", fs.ErrInvalid
	}
	f, err := p.files.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxRead {
		return "", fs.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(f, maxRead+1))
	if err != nil || len(b) > maxRead {
		return "", fs.ErrInvalid
	}
	return string(b), nil
}

func (p probe) metadata(r *Report, id, path string, valid func(fs.FileMode) bool, detail string) {
	info, err := p.lstat(path)
	status := "pass"
	if err != nil {
		status, detail = unavailable(err)
	} else if !valid(info.Mode()) {
		status, detail = "blocked", "Unexpected file type or permissions; authorized review is required."
	}
	r.add(id, status, detail)
}

func (p probe) inspect() Report {
	r := Report{SchemaVersion: 1, Deployment: deployment}
	status := "pass"
	if p.goos != "linux" || p.goarch != "amd64" {
		status = "blocked"
	}
	r.add("platform", status, "Only linux/amd64 is supported; runtime qualification has not been performed.")
	p.metadata(&r, "systemd", "run/systemd/system", fs.FileMode.IsDir, "Systemd runtime directory exists; manager operation is unverified.")
	for _, item := range []struct{ id, path string }{
		{"cgroup-v2", "sys/fs/cgroup/cgroup.controllers"},
		{"userns-clone", "proc/sys/kernel/unprivileged_userns_clone"},
		{"userns-limit", "proc/sys/user/max_user_namespaces"},
	} {
		text, err := p.read(item.path)
		status, detail := "pass", "Kernel prerequisite advertised; dedicated-account operation and delegation are unverified."
		if err != nil {
			status, detail = unavailable(err)
			if item.id == "userns-clone" && errors.Is(err, fs.ErrNotExist) {
				status, detail = "unknown", "Optional kernel control absent; user namespace policy is unverified."
			}
		} else if item.id == "cgroup-v2" {
			fields := " " + strings.Join(strings.Fields(text), " ") + " "
			for _, controller := range []string{"cpu", "memory", "pids"} {
				if !strings.Contains(fields, " "+controller+" ") {
					status, detail = "blocked", "Required CPU, memory, and PID controllers are not all advertised."
				}
			}
		} else if n, ok := number(strings.TrimSpace(text)); !ok || (item.id == "userns-clone" && n > 1) {
			status, detail = "unknown", "Kernel control value is malformed."
		} else if n == 0 {
			status, detail = "blocked", "User namespaces are disabled by this kernel control."
		}
		r.add(item.id, status, detail)
	}
	for _, name := range strings.Fields("docker dockerd rootlesskit slirp4netns newuidmap newgidmap systemctl loginctl runuser useradd getsubids iptables") {
		// Follow package-managed executable symlinks; ownership/setuid/capability
		// decisions require a separate authorized verification, not this probe.
		info, err := fs.Stat(p.files, "usr/bin/"+name)
		status, detail := "pass", "Regular executable metadata present; package integrity and effective privileges unverified."
		if err != nil {
			status, detail = unavailable(err)
		} else if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			status, detail = "blocked", "Path is not a regular executable."
		}
		r.add("binary-"+name, status, detail)
	}
	status, detail := p.engineVersion()
	r.add("docker-version", status, detail)
	status, detail, account := p.account()
	r.add("account", status, detail)
	status, detail, groupIDs := p.groups(account.gid)
	r.add("account-groups", status, detail)
	nss, _ := p.read("etc/nsswitch.conf")
	reservations, _ := p.read(strings.TrimPrefix(SubIDReservationsPath, "/"))
	for _, name := range []string{"subuid", "subgid"} {
		text, err := p.read("etc/" + name)
		status, detail = unavailable(err)
		if err == nil {
			hostIDs := account.hostUIDs
			if name == "subgid" {
				hostIDs = nil
				if account.hostGIDs != nil && groupIDs != nil {
					// Primary GIDs need not have an entry in /etc/group.
					hostIDs = append(groupIDs, account.hostGIDs...)
				}
			}
			status, detail = ValidateSubIDAuthority(nss, []byte(reservations), rootless.AccountName, text, account.uid, name, hostIDs)
		}
		r.add(name, status, detail)
	}
	p.metadata(&r, "linger", "var/lib/systemd/linger/"+rootless.AccountName, fs.FileMode.IsRegular, "Dedicated-account linger marker exists; unattended startup is unverified.")
	if account.uid == "" || account.uid == "0" {
		r.add("account-socket", "unknown", "A non-root dedicated-account UID is not established; no socket was inspected.")
	} else {
		p.metadata(&r, "account-socket", "run/user/"+account.uid+"/"+rootless.SocketName, func(m fs.FileMode) bool {
			return m&fs.ModeSocket != 0
		}, "Expected dedicated-account socket metadata exists; ownership, access and rootless engine operation are unverified.")
	}
	_, err := p.lstat("run/docker.sock")
	detail = "Rootful socket metadata exists; existing system Docker is preserved and is not a fallback."
	if errors.Is(err, fs.ErrNotExist) {
		detail = "Rootful socket absent; existing system Docker is preserved and is not a fallback."
	} else if err != nil {
		detail = "Rootful socket presence unknown (metadata unreadable); existing system Docker is preserved and is not a fallback."
	}
	r.add("rootful-socket", "info", detail)
	r.add("docker-environment", "info", fmt.Sprintf("DOCKER_HOST present: %t; DOCKER_CONTEXT present: %t. Overrides are ignored; values and Docker configuration are not inspected.", p.dockerHost, p.dockerContext))
	r.add("installation-verification", "unknown", "Package integrity, helper privileges, account locking/private home, and root-owned launcher digest and user-unit permissions are unverified, even if paths exist.")
	r.add("runtime-verification", "unknown", "No engine contacted; rootless mode, cgroup limits, private socket access, and unattended startup require explicit verification.")
	return r
}

func number(s string) (uint64, bool) {
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 10, 32)
	return n, err == nil && n < 1<<32-1 // Linux reserves (uid_t)-1.
}

// Nil inventories mean the full local database could not be validated.
type accountInfo struct {
	uid, gid           string
	hostUIDs, hostGIDs []uint64
}

func (p probe) account() (status, detail string, account accountInfo) {
	text, err := p.read("etc/passwd")
	if err != nil {
		status, detail = unavailable(err)
		return
	}
	status, detail = "missing", "Dedicated account is absent; authorized setup is required."
	account.hostUIDs, account.hostGIDs = []uint64{0}, []uint64{0}
	owners := make(map[uint64]int)
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) != 7 {
			return "unknown", "Account database is malformed.", accountInfo{}
		}
		u, uok := number(f[2])
		g, gok := number(f[3])
		if f[0] == "" || !uok || !gok || (f[0] == rootless.AccountName && account.uid != "") {
			return "unknown", "Account database is malformed or target identity is ambiguous.", accountInfo{}
		}
		account.hostUIDs = append(account.hostUIDs, u)
		account.hostGIDs = append(account.hostGIDs, g)
		owners[u]++
		if f[0] != rootless.AccountName {
			continue
		}
		account.uid, account.gid = strconv.FormatUint(u, 10), strconv.FormatUint(g, 10)
		status, detail = "pass", "Dedicated account has a non-root UID and expected home; locking and home permissions are unverified."
		if u == 0 || g == 0 || f[5] != rootless.HomeDir {
			status, detail = "blocked", "Dedicated account has a root UID/GID or unexpected home; do not repurpose it automatically."
		}
	}
	if u, ok := number(account.uid); ok && owners[u] > 1 {
		status, detail = "blocked", "Dedicated-account UID is shared with another login; account isolation is not established."
	}
	return
}

func (p probe) groups(gid string) (status, detail string, hostGIDs []uint64) {
	text, err := p.read("etc/group")
	if err != nil {
		status, detail = unavailable(err)
		return
	}
	hostGIDs = []uint64{0}
	forbidden, memberships := make(map[uint64]bool), make(map[uint64]bool)
	if g, ok := number(gid); ok {
		memberships[g] = true
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, ":")
		if len(f) != 4 {
			return "unknown", "Group database is malformed.", nil
		}
		g, ok := number(f[2])
		if f[0] == "" || !ok {
			return "unknown", "Group database is malformed.", nil
		}
		hostGIDs = append(hostGIDs, g)
		if f[0] == "wheel" || f[0] == "docker" {
			forbidden[g] = true
		}
		for _, member := range strings.Split(f[3], ",") {
			if strings.TrimSpace(member) == rootless.AccountName {
				memberships[g] = true
			}
		}
	}
	for g := range forbidden {
		if memberships[g] {
			return "blocked", "Dedicated account has forbidden wheel/docker primary or supplementary membership, including GID aliases.", hostGIDs
		}
	}
	if gid == "" {
		return "unknown", "Dedicated-account primary GID is unknown.", hostGIDs
	}
	return "pass", "No dedicated-account wheel/docker membership found in the local group database; external identity policy is unverified.", hostGIDs
}
