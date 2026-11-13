package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func validateDeskStage(out protocol.DeskStage, expected Identity, id string) error {
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return err
	}
	if out.Version != 1 || out.Receipt.Validate() != nil || out.Receipt.Intent.ID != id || out.ReadAt.IsZero() || time.Since(out.ReadAt) > protocol.DeskReadTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) || out.Receipt.CreatedAt.After(out.ReadAt) {
		return errors.New("invalid source stage receipt")
	}
	return nil
}
func (c *Client) StageDesk(ctx context.Context, expected Identity, intent deskwork.StageIntent, input io.Reader) (protocol.DeskStage, error) {
	var out protocol.DeskStage
	if err := intent.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskStageFeature); err != nil {
		return out, err
	}
	ctx, cancel := context.WithTimeout(ctx, protocol.DeskStageTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", "http://runner"+protocol.DeskStagesPath, input)
	if err != nil {
		return out, err
	}
	req.GetBody = nil
	req.Header.Set(protocol.Header, "1")
	req.Header.Set("Content-Type", "application/octet-stream")
	body, _ := json.Marshal(intent)
	req.Header.Set(protocol.DeskStageHeader, base64.RawURLEncoding.EncodeToString(body))
	hc := *c.httpClient
	hc.Timeout = protocol.DeskStageTimeout + time.Second
	resp, err := hc.Do(req)
	if err != nil {
		return out, errors.New("source upload outcome uncertain; inspect the original stage ID")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16385))
	if err != nil || len(data) > 16384 || resp.Header.Get(protocol.Header) != "1" {
		return out, errors.New("unverified source upload response; inspect the original stage ID")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var result protocol.ErrorResponse
		if json.Unmarshal(data, &result) != nil {
			return out, errors.New("source upload refused")
		}
		return out, &RemoteError{Status: resp.StatusCode, Problem: result.Error}
	}
	if json.Unmarshal(data, &out) != nil || validateDeskStage(out, expected, intent.ID) != nil || out.Receipt.Intent != intent {
		return protocol.DeskStage{}, errors.New("source upload receipt mismatch; inspect the original stage ID")
	}
	return out, nil
}
func (c *Client) DeskStage(ctx context.Context, expected Identity, id string) (protocol.DeskStage, error) {
	var out protocol.DeskStage
	if !protocol.ValidDeskOperationID(id) {
		return out, errors.New("invalid source stage ID")
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskStageFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskStageTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, "GET", protocol.DeskStagesPath+"/"+id, nil, &out, true); err != nil {
		return out, err
	}
	if err := validateDeskStage(out, expected, id); err != nil {
		return protocol.DeskStage{}, err
	}
	return out, nil
}

func (c *Client) DiscardDeskStage(ctx context.Context, expected Identity, id, digest string) (protocol.DeskStageDiscard, error) {
	var out protocol.DeskStageDiscard
	if !protocol.ValidDeskOperationID(id) || !protocol.ValidDeskOperationID(digest) {
		return out, errors.New("invalid source discard selection")
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskStageFeature); err != nil {
		return out, err
	}
	if err := c.do(ctx, "DELETE", protocol.DeskStagesPath+"/"+id, protocol.DeskStageDiscardRequest{SourceDigest: digest}, &out, true); err != nil {
		return out, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskStageDiscard{}, err
	}
	if out.Version != 1 || out.ID != id || out.SourceDigest != digest || !out.Discarded {
		return protocol.DeskStageDiscard{}, errors.New("source discard response mismatch; inspect the original stage ID")
	}
	return out, nil
}
