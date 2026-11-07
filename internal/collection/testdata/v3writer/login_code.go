package collection

import (
	"context"
	"encoding/hex"
	"errors"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Registry) LoginCode(ctx context.Context, name, expectedRunner, instance string, revision uint64, container string) (protocol.LoginCode, error) {
	if !protocol.ValidID(expectedRunner) || !protocol.ValidID(instance) || revision == 0 || len(container) != 64 {
		return protocol.LoginCode{}, errors.New("require the selected runner, instance, revision and full container ID")
	}
	if _, err := hex.DecodeString(container); err != nil {
		return protocol.LoginCode{}, errors.New("invalid container ID")
	}
	entry, err := r.Get(ctx, name)
	if err != nil {
		return protocol.LoginCode{}, err
	}
	if entry.RunnerID != expectedRunner {
		return protocol.LoginCode{}, errors.New("selected runner enrollment changed")
	}
	c := entry.Client(true)
	defer c.Close()
	hello, err := c.Hello(ctx)
	if err != nil {
		return protocol.LoginCode{}, errors.New("runner identity verification failed")
	}
	if hello.AccessRole != "operator" || !slices.Contains(hello.Features, "login-code") {
		return protocol.LoginCode{}, errors.New("runner requires operator access and login-code capability")
	}
	return c.LoginCode(ctx, expectedRunner, instance, protocol.LoginCodeRequest{ExpectedRevision: revision, ContainerID: container})
}
