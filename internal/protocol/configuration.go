package protocol

import (
	"bytes"
	"encoding/json"
)

type configurationRequestWire struct {
	ExpectedRevision uint64 `json:"expected_revision"`
	Config           struct {
		Configuration
		NativeControl *bool `json:"native_control,omitempty"`
	} `json:"config"`
}

// MarshalJSON sends an explicit false for an intentional disable. The public
// snapshot's omitempty boolean cannot represent that request-side distinction.
func (r ConfigureRequest) MarshalJSON() ([]byte, error) {
	var wire configurationRequestWire
	wire.ExpectedRevision, wire.Config.Configuration = r.ExpectedRevision, r.Config
	if !r.PreserveNativeControl {
		wire.Config.NativeControl = &r.Config.NativeControl
	}
	return json.Marshal(wire)
}

func (r *ConfigureRequest) UnmarshalJSON(data []byte) error {
	var wire configurationRequestWire
	d := json.NewDecoder(bytes.NewReader(data))
	// Custom decoding must retain the endpoint's unknown-field refusal.
	d.DisallowUnknownFields()
	if err := d.Decode(&wire); err != nil {
		return err
	}
	*r = ConfigureRequest{ExpectedRevision: wire.ExpectedRevision, Config: wire.Config.Configuration, PreserveNativeControl: wire.Config.NativeControl == nil}
	if wire.Config.NativeControl != nil {
		r.Config.NativeControl = *wire.Config.NativeControl
	}
	return nil
}
