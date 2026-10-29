//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// The rootless API belongs to the same already verified engine owner. Requests
// never accept sockets, addresses or protocols from RPC input.
type ProjectPortSpec struct {
	Proto      string `json:"proto"`
	ParentIP   string `json:"parentIP"`
	ParentPort int    `json:"parentPort"`
	ChildIP    string `json:"childIP"`
	ChildPort  int    `json:"childPort"`
}
type ProjectPort struct {
	ID   int             `json:"id"`
	Spec ProjectPortSpec `json:"spec"`
}

func (q *qualifier) projectPortAPI(ctx context.Context, method, path string, input, output any) error {
	dir := q.d.runtimeDir
	if dir == "" {
		dir = fmt.Sprintf("/run/user/%d", q.uid)
	}
	name := "urbit-rootlesskit"
	if q.d.runtimeDir != "" {
		name = "rootlesskit"
	}
	socket := filepath.Join(dir, name, "api.sock")
	if err := q.owned(dir, os.ModeDir, true); err != nil {
		return err
	}
	if err := q.owned(filepath.Dir(socket), os.ModeDir, false); err != nil {
		return err
	}
	if err := q.owned(socket, os.ModeSocket, false); err != nil {
		return err
	}
	tr := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("rootless API redirects refused") }}
	var data []byte
	var err error
	if input != nil {
		data, err = json.Marshal(input)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://rootless/v1"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return errors.New("rootless project port API unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("rootless project port API returned %d", res.StatusCode)
	}
	if output != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(output)
	}
	return nil
}
func (q *qualifier) checkProjectPorts(ctx context.Context) error {
	var info struct {
		PortDriver *struct {
			Driver string `json:"driver"`
		} `json:"portDriver"`
	}
	if err := q.projectPortAPI(ctx, "GET", "/info", nil, &info); err != nil {
		return err
	}
	if info.PortDriver == nil || info.PortDriver.Driver != "builtin" {
		return errors.New("incremental project members require the rootless builtin port driver")
	}
	return nil
}
func (q *qualifier) addProjectPort(ctx context.Context, spec ProjectPortSpec) (ProjectPort, error) {
	var result ProjectPort
	if spec.Proto != "tcp4" || spec.ParentIP != "127.0.0.1" || spec.ParentPort < 1024 || spec.ChildPort < 8080 || spec.ChildPort > 8335 || net.ParseIP(spec.ChildIP) == nil || !net.ParseIP(spec.ChildIP).IsPrivate() {
		return result, errors.New("invalid project publication")
	}
	err := q.projectPortAPI(ctx, "POST", "/ports", spec, &result)
	if err == nil && (result.ID <= 0 || result.Spec != spec) {
		err = errors.New("project publication receipt changed")
	}
	return result, err
}
func (q *qualifier) verifyProjectPort(ctx context.Context, p ProjectPort) error {
	var all []ProjectPort
	if err := q.projectPortAPI(ctx, "GET", "/ports", nil, &all); err != nil {
		return err
	}
	for _, v := range all {
		if v.ID == p.ID && v.Spec == p.Spec {
			return nil
		}
	}
	return errors.New("project HTTP publication is missing; review the engine restart before reconnecting")
}
