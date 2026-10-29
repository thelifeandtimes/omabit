//go:build linux && amd64

package runtimequalify

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var errHTTPPorts = errors.New("selected runtime HTTP inventory is unavailable or invalid")

// privateHTTPPort recognizes Vere's bounded data file; it never accepts a URL.
// The public port must match the inspected runtime's configured internal port.
func privateHTTPPort(data string, public uint16) (uint16, error) {
	if data == "" || len(data) > 1024 || public < 1024 {
		return 0, errHTTPPorts
	}
	rows := strings.Split(strings.TrimSuffix(data, "\n"), "\n")
	if len(rows) > 4 {
		return 0, errHTTPPorts
	}
	var loopback uint16
	publicSeen := false
	used := map[uint64]bool{}
	kinds := map[string]bool{}
	for _, row := range rows {
		fields := strings.Fields(row)
		if len(fields) != 3 {
			return 0, errHTTPPorts
		}
		port, err := strconv.ParseUint(fields[0], 10, 16)
		if err != nil || port == 0 || strconv.FormatUint(port, 10) != fields[0] || used[port] ||
			(fields[1] != "secure" && fields[1] != "insecure") || (fields[2] != "public" && fields[2] != "loopback") {
			return 0, errHTTPPorts
		}
		key := fields[1] + " " + fields[2]
		if kinds[key] {
			return 0, errHTTPPorts
		}
		used[port], kinds[key] = true, true
		if key == "insecure public" {
			if uint16(port) != public {
				return 0, errHTTPPorts
			}
			publicSeen = true
		}
		if key == "insecure loopback" {
			if port < 1024 {
				return 0, errHTTPPorts
			}
			loopback = uint16(port)
		}
	}
	if loopback == 0 || !publicSeen {
		return 0, errHTTPPorts
	}
	return loopback, nil
}

func (q *qualifier) readHTTPPorts() (string, error) {
	if err := q.pier(); err != nil {
		return "", errHTTPPorts
	}
	return readHTTPPortsAt(q.pierPath())
}

func readHTTPPortsAt(path string) (string, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return "", errHTTPPorts
	}
	defer root.Close()
	// Nonblocking and no-follow: a FIFO/symlink must not turn a bounded read
	// into a wait or access to another filesystem object.
	f, err := root.OpenFile(".http.ports", os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", errHTTPPorts
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", errHTTPPorts
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 || st.Nlink != 1 || st.Uid != uint32(os.Geteuid()) || info.Size() > 1024 {
		return "", errHTTPPorts
	}
	data, err := io.ReadAll(io.LimitReader(f, 1025))
	if err != nil || len(data) > 1024 {
		return "", errHTTPPorts
	}
	return string(data), nil
}
