package service

import (
	"context"
	"errors"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
)

// RuntimeProfileInput selects an enrolled runner and a finite reviewed artifact.
type RuntimeProfileInput struct {
	Target
	RuntimeProfile string `json:"runtime_profile"`
}

func (in RuntimeProfileInput) Validate() error {
	if in.Target.Validate() != nil || in.RuntimeProfile == "" {
		return invalid()
	}
	if _, err := runtimeprofile.Resolve(in.RuntimeProfile); err != nil && !runtimeprofile.IsLocalID(in.RuntimeProfile) {
		return invalid()
	}
	return nil
}

// LogTailInput opts into the selected operator-private preview. It does not
// grant original-recording access, publish an export or enable following.
type LogTailInput struct {
	Target
	InstanceID            string `json:"instance_id"`
	ConfigurationRevision uint64 `json:"configuration_revision"`
	ContainerID           string `json:"container_id"`
	IncludePrivateData    bool   `json:"include_private_data"`
}

func (in LogTailInput) Validate() error {
	if in.Target.Validate() != nil || !protocol.ValidID(in.InstanceID) || in.ConfigurationRevision == 0 || !protocol.ValidDeskOperationID(in.ContainerID) || in.ContainerID == strings.Repeat("0", 64) || !in.IncludePrivateData {
		return invalid()
	}
	return nil
}

func (s *Service) registerRuntimeReads() error {
	return errors.Join(
		selected(s, description("runtime.profile-preflight.v1", "Verify an immutable runtime profile and its staged image on the selected runner without opening a pier.", nil, []string{"local-metadata-read", "live-probe"}, []string{protocol.RuntimeProfilesFeature}, "operator", "pinned runner and exact reviewed profile metadata", "read-only artifact preflight; never installs, selects or starts a runtime"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RuntimeProfileInput) (protocol.RuntimeProfilePreflight, error) {
			c := e.Client(true)
			defer c.Close()
			return c.RuntimeProfilePreflight(ctx, in.RuntimeProfile)
		}),
		selected(s, description("runner.snapshot.v1", "Read the selected runner fleet snapshot, including bounded current project/storage probes; no lifecycle action or polling.", []string{"urbit-collector snapshot"}, []string{"local-metadata-read", "saved-runner-read", "live-probe"}, []string{"snapshot"}, "authenticated", "explicit enrolled runner name and pinned runner ID/mode, also checked on the snapshot response", "one bounded read; no automatic watch or mutation"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, _ Target) (protocol.Snapshot, error) {
			c := e.Client(true)
			defer c.Close()
			out, err := c.Snapshot(ctx)
			if err == nil {
				err = e.Identity().Check(out.RunnerID, out.Mode)
			}
			if err != nil {
				return protocol.Snapshot{}, err
			}
			return out, nil
		}),
		selected(s, description("runtime.log-tail.v1", "Read one operator-private bounded sanitized log preview after include_private_data=true. May contain private application data. Not a complete recording, durable cursor, export or watch.", nil, []string{"local-metadata-read", "live-probe", "private-runtime-read"}, []string{"runtime-logs"}, "operator", "explicit enrolled runner identity, instance, exact configuration revision and full container ID; changed response is withheld", "one read with no retry; select runner.snapshot.v1 again if the runtime changes"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in LogTailInput) (protocol.RuntimeLogs, error) {
			c := e.Client(true)
			defer c.Close()
			out, err := c.RuntimeLogs(ctx, e.RunnerID, in.InstanceID, in.ConfigurationRevision)
			if err != nil {
				return protocol.RuntimeLogs{}, err
			}
			if out.ContainerID != in.ContainerID {
				return protocol.RuntimeLogs{}, &refusal{"runtime_changed", "The selected container changed while reading logs.", "Refresh the exact snapshot before reading again."}
			}
			return out, nil
		}),
	)
}
