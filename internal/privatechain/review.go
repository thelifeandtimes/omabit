package privatechain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// Review is the offline planner's saved public document, never runtime authority.
type Review struct {
	SchemaVersion     int    `json:"schema_version"`
	Plan              Plan   `json:"plan"`
	ReviewDigest      string `json:"review_digest"`
	ReservationCheck  string `json:"reservation_check"`
	RuntimeAuthorized bool   `json:"runtime_authorized"`
	Notice            string `json:"notice"`
}

func DecodeReview(data []byte) (Review, error) {
	var r Review
	bad := errors.New("expected a bounded canonical offline private-chain review with no runtime authority")
	if len(data) > MaxFoundationBytes || !uniqueJSONDepth(data, 6) {
		return r, bad
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF {
		return Review{}, bad
	}
	digest, err := r.Plan.Digest()
	if err != nil || r.SchemaVersion != 1 || r.RuntimeAuthorized || r.ReservationCheck != "not-performed" || digest != r.ReviewDigest {
		return Review{}, bad
	}
	return r, nil
}
