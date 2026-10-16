package client

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
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
func (c *Client) PreviewGuidedPierImport(ctx context.Context, runner, path, root string) (protocol.PierImportPreview, error) {
	var preview protocol.PierImportPreview
	err := c.storageRequest(ctx, "/v1/pier-import-preview", protocol.ImportPierPreviewRequest{Source: path, StorageRoot: root}, &preview)
	if err == nil && (preview.RunnerID != runner || preview.Source.Path != path || preview.StorageRoot != root || !protocol.ValidID(preview.InstanceID) || !protocol.ValidFakeShip(preview.Ship) || !pierstore.ValidPath(preview.Destination) || pierstore.Kind(preview.Destination) != preview.Kind || preview.HTTPPort < 1024 || (preview.Network != "livenet" && preview.Network != "fakenet")) {
		err = errors.New("guided import preview does not match the selected source, runner and location")
	}
	id, _ := pierstore.Split(preview.Destination)
	if id == "" {
		id = "default"
	}
	if err == nil && (id != root || !epochs.ValidDigest(preview.Source.Fingerprint) || preview.Source.Bytes < 0 || preview.Source.Bytes > protocol.MaxPierBytes || preview.Source.Files < 0 || preview.Source.Files > 8192 || filepath.Base(preview.Destination) != strings.TrimPrefix(preview.Ship, "~") || filepath.Base(filepath.Dir(preview.Destination)) != preview.InstanceID || preview.Network == "fakenet" && preview.Kind != "fake" || preview.Network == "livenet" && preview.Kind != protocol.ShipClass(preview.Ship)) {
		err = errors.New("guided import identity, bounds or destination changed")
	}
	return preview, err
}
func (c *Client) ImportPier(ctx context.Context, req protocol.ImportPierRequest) (protocol.Operation, error) {
	var op protocol.Operation
	err := c.storageRequest(ctx, "/v1/pier-import", req, &op)
	if err == nil && (op.Kind != "import-pier" || op.InstanceID != req.ID || op.RequestID != req.RequestID) {
		err = errors.New("import response does not match request")
	}
	return op, err
}
