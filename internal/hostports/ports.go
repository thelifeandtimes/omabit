package hostports

import (
	"net"
	"strconv"
)

// Available checks the actual bind address used by the runtime. It is admission
// evidence, not a reservation: Docker still decides races at publication time.
func Available(kind string, port uint16) bool {
	if kind == "udp" {
		c, err := net.ListenPacket("udp4", "0.0.0.0:"+strconv.Itoa(int(port)))
		if err != nil {
			return false
		}
		c.Close()
		return true
	}
	c, err := net.Listen("tcp4", "127.0.0.1:"+strconv.Itoa(int(port)))
	if err != nil {
		return false
	}
	c.Close()
	return true
}
