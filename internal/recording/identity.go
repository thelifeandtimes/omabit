package recording

import (
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// HostNamespaces identifies the external recorder's host context. It is not a
// claim about a container PID or a container namespace before Start. The runner
// must obtain these values itself, never trust a caller's recording metadata.
type HostNamespaces struct {
	User    uint64 `json:"user"`
	Mount   uint64 `json:"mount"`
	PID     uint64 `json:"pid"`
	Network uint64 `json:"network"`
}

// ProjectNamespace identifies the already verified anchor network namespace at
// attachment. A project's generation and creator are required even before its
// ship process starts. Ordinary containers omit this field; their post-Start
// process/network facts must be recorded independently, not invented here.
type ProjectNamespace struct {
	BootID string `json:"boot_id"`
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
}

type AttachmentIdentity struct {
	Target               apppolicy.Target `json:"target"`
	NativeShip           string           `json:"native_ship"`
	PierContinuitySHA256 string           `json:"pier_continuity_sha256"`
	// Version4 explicitly labels what was knowable before Start. Registered
	// identity is expected intent, not proof that a process answered /~/host.
	// Prepared storage binds the verified exclusive mount/selected leaf contract,
	// not an invented future pier/.urb inode. Older metadata omits both labels.
	NativeShipStatus     string            `json:"native_ship_status,omitempty"`
	PierBinding          string            `json:"pier_binding,omitempty"`
	EngineID             string            `json:"engine_id"`
	EngineEndpointSHA256 string            `json:"engine_endpoint_sha256"`
	HostBootID           string            `json:"host_boot_id"`
	RecorderNamespaces   HostNamespaces    `json:"recorder_namespaces"`
	ProjectNamespace     *ProjectNamespace `json:"project_namespace,omitempty"`
}

// CaptureIdentity refuses to infer a current target/generation from a legacy
// instance/project name. A successful return is metadata validation only, not
// runtime observation or permission to read/export the recording.
func (m Metadata) CaptureIdentity() (AttachmentIdentity, error) {
	if m.Version == 1 {
		return AttachmentIdentity{}, ErrLegacyMetadata
	}
	if m.Validate() != nil || m.Identity == nil {
		return AttachmentIdentity{}, ErrInvalid
	}
	out := *m.Identity
	if out.ProjectNamespace != nil {
		namespace := *out.ProjectNamespace
		out.ProjectNamespace = &namespace
	}
	return out, nil
}

func validBootID(id string) bool {
	return len(id) == 36 && id[8] == '-' && id[13] == '-' && id[18] == '-' && id[23] == '-' && lowerHex(strings.ReplaceAll(id, "-", ""), 32) && strings.ReplaceAll(id, "-", "") != strings.Repeat("0", 32)
}

func nonzeroDigest(id string) bool { return lowerHex(id, 64) && id != strings.Repeat("0", 64) }

func (i AttachmentIdentity) validate(m Metadata) error {
	t := i.Target
	knownShip := protocol.ValidFakeShip(i.NativeShip)
	if m.Version < 4 {
		if i.NativeShipStatus != "" || i.PierBinding != "" {
			return ErrInvalid
		}
	} else {
		if i.PierBinding != "existing-pier" && i.PierBinding != "prepared-storage" {
			return ErrInvalid
		}
		switch i.NativeShipStatus {
		case "registered":
			if !knownShip {
				return ErrInvalid
			}
		case "pending-comet-mining":
			if i.NativeShip != "" || t.ProjectID != "" {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	if t.Validate() != nil || t.InstanceID == "" || t.InstanceID != m.InstanceID || t.ProjectID != m.ProjectID || t.Desk != "" || !knownShip && i.NativeShipStatus != "pending-comet-mining" || !nonzeroDigest(i.PierContinuitySHA256) || !hostcompat.ValidEngineIdentity(i.EngineID) || !nonzeroDigest(i.EngineEndpointSHA256) || !validBootID(i.HostBootID) || i.RecorderNamespaces.User == 0 || i.RecorderNamespaces.Mount == 0 || i.RecorderNamespaces.PID == 0 || i.RecorderNamespaces.Network == 0 {
		return ErrInvalid
	}
	if t.ProjectID == "" {
		if i.ProjectNamespace != nil {
			return ErrInvalid
		}
	} else {
		if t.Generation == 0 || t.Creator == "" || i.ProjectNamespace == nil || i.ProjectNamespace.BootID != i.HostBootID || i.ProjectNamespace.Device == 0 || i.ProjectNamespace.Inode == 0 {
			return ErrInvalid
		}
	}
	return nil
}
