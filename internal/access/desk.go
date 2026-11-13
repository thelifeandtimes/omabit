package access

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func deskStageFraming(req *http.Request) error {
	values := req.Header.Values(protocol.DeskStageHeader)
	if len(values) != 1 || len(values[0]) == 0 || len(values[0]) > 8192 ||
		req.Header.Get("Content-Type") != "application/octet-stream" ||
		req.ContentLength == 0 || req.ContentLength > deskwork.MaxBundleBytes || len(req.Trailer) != 0 {
		return errors.New("invalid source upload framing")
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(values[0])
	if err != nil {
		return errors.New("invalid source upload descriptor")
	}
	var intent deskwork.StageIntent
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&intent) != nil || d.Decode(new(any)) != io.EOF || intent.Validate() != nil {
		return errors.New("invalid source transfer intent")
	}
	return nil
}

// Match the public client's existing bounds. Header/ordinary-body framing keeps
// the short SSH progress deadline; only an authorized, framed desk RPC receives
// its longer application deadline. Other RPCs keep their previous limits.
func deskRequestTimeout(path string) time.Duration {
	if path == protocol.DeskStagesPath || strings.HasPrefix(path, protocol.DeskStagesPath+"/") && protocol.ValidDeskOperationID(strings.TrimPrefix(path, protocol.DeskStagesPath+"/")) {
		return protocol.DeskStageTimeout
	}
	switch path {
	case protocol.DeskObservationPath, protocol.DeskDiscoveryPath, protocol.DeskSourcePath,
		protocol.DeskApplicationPath, protocol.DeskReceiverStatusPath:
		return protocol.DeskReadTimeout
	case protocol.DeskDeploymentReviewPath, protocol.DeskCommitPath, protocol.DeskAdoptionReviewPath,
		protocol.DeskAdoptionPath, protocol.DeskReleaseReviewPath, protocol.DeskReleasePath,
		protocol.DeskFrontendReviewPath, protocol.DeskFrontendsPath, protocol.DeskReceiverReviewPath,
		protocol.DeskReceiversPath:
		return protocol.DeskDeploymentTimeout
	}
	return 0
}
