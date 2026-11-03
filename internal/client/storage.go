package client

import (
	"context"
	"errors"
	"fmt"
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
	if err != nil {
		return preview, err
	}
	if preview.RunnerID != runner || preview.Source.Path != path || preview.StorageRoot != root {
		return preview, errors.New("guided import preview does not match the selected source, runner and location; inspect the pier again")
	}
	if !protocol.ValidID(preview.InstanceID) || !protocol.ValidFakeShip(preview.Ship) || !epochs.ValidDigest(preview.Source.Fingerprint) ||
		(preview.Network != "livenet" && preview.Network != "fakenet") ||
		preview.Network == "fakenet" && preview.Kind != "fake" ||
		preview.Network == "livenet" && preview.Kind != protocol.ShipClass(preview.Ship) {
		return preview, errors.New("guided import preview has an invalid identity, network or source fingerprint; inspect the pier again")
	}
	id, _ := pierstore.Split(preview.Destination)
	if id == "" {
		id = "default"
	}
	if !pierstore.ValidPath(preview.Destination) || pierstore.Kind(preview.Destination) != preview.Kind || id != root ||
		filepath.Base(preview.Destination) != strings.TrimPrefix(preview.Ship, "~") ||
		filepath.Base(filepath.Dir(preview.Destination)) != preview.InstanceID || preview.HTTPPort < 1024 {
		return preview, errors.New("guided import destination or port does not match the reviewed identity and storage location; inspect the pier again")
	}
	err = validateLocalImportBounds("import source", preview.Source.Bytes, preview.Source.Files)
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

// Local moves and their restoration use the filesystem import limit. The
// separate protocol.MaxPierBytes limit applies only to cross-host streams.
func validateLocalImportBounds(label string, bytes int64, files int) error {
	if bytes < 0 {
		return fmt.Errorf("%s reports a negative logical byte count", label)
	}
	if bytes > pierstore.MaxLocalPierBytes {
		return fmt.Errorf("%s exceeds the 8 TiB local import limit (%d logical bytes); sparse file lengths count toward this limit", label, bytes)
	}
	if files < 0 || files > 8192 {
		return fmt.Errorf("%s reports an invalid entry count (%d); local imports allow at most 8192 entries", label, files)
	}
	return nil
}
