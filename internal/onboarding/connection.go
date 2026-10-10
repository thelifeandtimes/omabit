package onboarding

import (
	"context"
	"errors"
	"fmt"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostinstall"
)

func probeConnection(ctx context.Context, c collection.Connection) error {
	cli := c.Client(true)
	defer cli.Close()
	h, err := cli.Hello(ctx)
	if err != nil {
		return err
	}
	if h.AccessRole != "operator" {
		return errors.New("local connection is not an operator")
	}
	return nil
}

func connectLocal(ctx context.Context, config string, ssh client.SSHConfig, host hostinstall.HostResult) (collection.Connection, error) {
	cli := client.NewSSH(ssh, &client.Identity{RunnerID: host.RunnerID, Mode: host.Mode})
	hello, err := cli.Hello(ctx)
	cli.Close()
	if err != nil || hello.AccessRole != "operator" || hello.Actor != "ssh-key:"+host.OperatorFingerprint {
		return collection.Connection{}, errors.New("The local SSH connection did not verify the expected runner and operator grant.")
	}
	r, err := collection.Open(config)
	if err != nil {
		return collection.Connection{}, err
	}
	entries, err := r.List(ctx)
	if err != nil {
		return collection.Connection{}, err
	}
	for _, old := range entries {
		if old.RunnerID != host.RunnerID {
			continue
		}
		if old.Mode != host.Mode {
			return collection.Connection{}, errors.New("The enrolled runner mode differs. Explicitly review its mode in Manage runners.")
		}
		if old.Transport == "ssh" && old.SSH != nil && *old.SSH == ssh {
			return old, nil
		}
		return r.Update(ctx, old.Name, old.RunnerID, old.Revision, collection.Connection{Name: old.Name, Transport: "ssh", SSH: &ssh})
	}
	name := "local"
	for index := 2; ; index++ {
		used := false
		for _, old := range entries {
			if old.Name == name {
				used = true
			}
		}
		if !used {
			break
		}
		name = fmt.Sprintf("local-%d", index)
	}
	return r.EnrollSSHExpected(ctx, name, ssh, client.Identity{RunnerID: host.RunnerID, Mode: host.Mode})
}
