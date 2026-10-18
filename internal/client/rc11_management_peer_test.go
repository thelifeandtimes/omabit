package client

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRC11ManagementClientPinsKernelPeer(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "manager.sock")
	l, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(protocol.Header, "1")
		json.NewEncoder(w).Encode(protocol.ManagementUpdateStatus{PID: os.Getpid()})
	})}
	defer srv.Close()
	go srv.Serve(l)
	for _, good := range []bool{true, false} {
		pid := os.Getpid()
		if !good {
			pid++
		}
		c := NewLocalManagement(socket, uint32(os.Geteuid()), pid)
		_, err := c.ManagementUpdate(context.Background(), nil)
		c.Close()
		if good && err != nil || !good && err == nil {
			t.Fatal(good, err)
		}
	}
}
