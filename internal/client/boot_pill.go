package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) StageBootPill(ctx context.Context, input io.Reader, meta protocol.BootPill) error {
	if !protocol.ValidBootPill(&meta) {
		return errors.New("invalid pill descriptor")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", "http://runner"+protocol.PillUploadPath, input)
	if err != nil {
		return err
	}
	req.ContentLength = meta.Bytes
	req.GetBody = nil
	req.Header.Set(protocol.Header, strconv.Itoa(protocol.Version))
	req.Header.Set(protocol.PillHashHeader, meta.SHA256)
	req.Header.Set("Content-Type", "application/octet-stream")
	client := *c.httpClient
	client.Timeout = 10 * time.Minute
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("pill upload outcome uncertain; ship registration was not sent")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8193))
	if err != nil || len(data) > 8192 || resp.Header.Get(protocol.Header) != "1" {
		return errors.New("unverified pill upload response; registration was not sent")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e protocol.ErrorResponse
		if json.Unmarshal(data, &e) != nil {
			return errors.New("pill upload refused")
		}
		return &RemoteError{Status: resp.StatusCode, Problem: e.Error}
	}
	var result protocol.BootPill
	if json.Unmarshal(data, &result) != nil || result != meta {
		return errors.New("pill upload acknowledgement mismatch; registration was not sent")
	}
	return nil
}
