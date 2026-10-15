package client

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"time"
)

func (c *Client) storageRequest(ctx context.Context, path string, body, result any) error {
	httpClient := *c.httpClient
	httpClient.Timeout = 45 * time.Second
	scoped := *c
	scoped.httpClient = &httpClient
	return scoped.do(ctx, http.MethodPost, path, body, result, true)
}

func (c *Client) AddStorageRoot(ctx context.Context, req protocol.StorageRootRequest) (protocol.Operation, error) {
	var op protocol.Operation
	err := c.storageRequest(ctx, "/v1/storage-roots", req, &op)
	if err == nil && (op.Kind != "storage-root" || op.RequestID != req.RequestID || op.Status != "succeeded") {
		err = errors.New("storage response does not match request")
	}
	return op, err
}
func (c *Client) PreviewPierImport(ctx context.Context, path string) (pierstore.ExternalSource, error) {
	var source pierstore.ExternalSource
	err := c.storageRequest(ctx, "/v1/pier-import-preview", protocol.ImportPierPreviewRequest{Source: path}, &source)
	if err == nil && source.Path != path {
		err = errors.New("import preview source changed")
	}
	return source, err
}
func (c *Client) ImportPier(ctx context.Context, req protocol.ImportPierRequest) (protocol.Operation, error) {
	var op protocol.Operation
	err := c.storageRequest(ctx, "/v1/pier-import", req, &op)
	if err == nil && (op.Kind != "import-pier" || op.InstanceID != req.ID || op.RequestID != req.RequestID) {
		err = errors.New("import response does not match request")
	}
	return op, err
}
