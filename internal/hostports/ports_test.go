package hostports

import (
	"net"
	"testing"
)

func TestAvailableRejectsActualRuntimeBindConflicts(t *testing.T) {
	tcp, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer tcp.Close()
	port := uint16(tcp.Addr().(*net.TCPAddr).Port)
	if Available("tcp", port) {
		t.Fatal("occupied loopback TCP port accepted")
	}
	tcp.Close()
	if !Available("tcp", port) {
		t.Fatal("released TCP port refused")
	}
	udp, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	port = uint16(udp.LocalAddr().(*net.UDPAddr).Port)
	if Available("udp", port) {
		t.Fatal("occupied wildcard UDP port accepted")
	}
	udp.Close()
	if !Available("udp", port) {
		t.Fatal("released UDP port refused")
	}
}
