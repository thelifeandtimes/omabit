package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) PrepareHandoff(ctx context.Context, id string, req protocol.PrepareHandoffRequest) (protocol.Handoff, error) {
	var h protocol.Handoff
	if err := validateID(id); err != nil {
		return h, err
	}
	err := c.do(ctx, "POST", "/v1/instances/"+id+"/handoff", req, &h, true)
	return h, err
}
func (c *Client) ReceiveHandoff(ctx context.Context, req protocol.ReceiveHandoffRequest) (protocol.Handoff, error) {
	var h protocol.Handoff
	err := c.do(ctx, "POST", "/v1/handoffs", req, &h, true)
	return h, err
}
func (c *Client) Handoff(ctx context.Context, id string) (protocol.Handoff, error) {
	var h protocol.Handoff
	if err := validateID(id); err != nil {
		return h, err
	}
	err := c.do(ctx, "GET", "/v1/handoffs/"+id, nil, &h, true)
	return h, err
}
func (c *Client) FinishHandoff(ctx context.Context, id, verb string, req protocol.FinishHandoffRequest) (protocol.Handoff, error) {
	var h protocol.Handoff
	if err := validateID(id); err != nil {
		return h, err
	}
	if verb != "commit" && verb != "activate" && verb != "retry" {
		return h, errors.New("invalid handoff completion")
	}
	err := c.do(ctx, "POST", "/v1/handoffs/"+id+"/"+verb, req, &h, true)
	return h, err
}

// RelayHandoff streams through the authenticated collector. No pier archive is
// written on the desktop and no SSH shell/SFTP access is granted to management.
func RelayHandoff(ctx context.Context, source, destination *Client, h protocol.Handoff) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	get, _ := http.NewRequestWithContext(ctx, "GET", "http://runner/v1/handoffs/"+h.ID+"/data", nil)
	get.Header.Set(protocol.Header, "1")
	client := *source.httpClient
	client.Timeout = 30 * time.Minute
	response, err := client.Do(get)
	if err != nil {
		return errors.New("source handoff stream unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || response.Header.Get(protocol.Header) != "1" || response.Header.Get("Content-Type") != "application/octet-stream" || response.ContentLength != h.Bytes || response.Header.Get(protocol.HandoffHashHeader) != h.SHA256 {
		return errors.New("source handoff stream does not match its offer")
	}
	put, _ := http.NewRequestWithContext(ctx, "PUT", "http://runner/v1/handoffs/"+h.ID+"/data", response.Body)
	put.ContentLength = h.Bytes
	put.GetBody = nil
	put.Header.Set(protocol.Header, "1")
	put.Header.Set("Content-Type", "application/octet-stream")
	put.Header.Set(protocol.HandoffHashHeader, h.SHA256)
	client = *destination.httpClient
	client.Timeout = 30 * time.Minute
	ack, err := client.Do(put)
	if err != nil {
		return errors.New("transfer outcome uncertain; source remains fenced; inspect handoff status")
	}
	defer ack.Body.Close()
	data, err := io.ReadAll(io.LimitReader(ack.Body, 65537))
	if err != nil || len(data) > 65536 || ack.Header.Get(protocol.Header) != "1" {
		return errors.New("invalid handoff acknowledgement")
	}
	if ack.StatusCode != 202 && ack.StatusCode != 200 {
		var problem protocol.ErrorResponse
		if json.Unmarshal(data, &problem) == nil {
			return &RemoteError{Status: ack.StatusCode, Problem: problem.Error}
		}
		return errors.New("destination refused handoff data")
	}
	var got protocol.Handoff
	if json.Unmarshal(data, &got) != nil || !got.SameTransfer(h) || got.Direction != "in" || got.State != "receiving" {
		return errors.New("handoff acknowledgement identity mismatch")
	}
	return nil
}
