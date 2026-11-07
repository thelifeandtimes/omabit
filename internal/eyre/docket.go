package eyre

import (
	"context"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

// UploadDocket uses the existing authenticated upload route exactly once. The
// caller owns effect permission, journaling and fresh desk/source checks. Native
// Docket can commit a new docket hash to Clay; this is never just a file copy.
// An acknowledgement is not source, artifact, or HTTP-health verification.
func (c *Client) UploadDocket(ctx context.Context, desk string, artifact deskwork.FrontendManifest, files []deskwork.SourceFile) error {
	body, contentType, err := deskwork.FrontendMultipart(desk, artifact, files)
	if err != nil {
		return err
	}
	defer clear(body)
	if err = c.Check(ctx); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	response, err := c.request(ctx, "POST", "/docket/upload", body, true, http.Header{"Content-Type": {contentType}})
	if err != nil {
		return err
	}
	data, err := readResponse(response)
	defer clear(data)
	if err != nil || response.StatusCode != http.StatusOK || !strings.Contains(string(data), "successfully globbed") {
		return ErrUncertain
	}
	return nil
}
