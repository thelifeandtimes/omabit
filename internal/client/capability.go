package client

import "git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"

// UnsupportedCapabilityError preserves the actual checked Hello so a caller can
// distinguish missing API support from a policy denial or unavailable adapter.
type UnsupportedCapabilityError struct {
	Feature string
	Hello   protocol.Hello
}

func (e *UnsupportedCapabilityError) Error() string {
	if e.Feature == protocol.AppPolicyFeature {
		return "runner lacks app-development-policy-v2; desk creation, commit and publication unavailable"
	}
	return "runner does not advertise " + e.Feature
}
