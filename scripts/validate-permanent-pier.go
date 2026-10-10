//go:build ignore

// Explicit development acceptance while management is stopped. The temporary
// fake is never enrolled; its stopped pier and receipt are retained in archive.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if os.Geteuid() != 959 || len(os.Args) != 2 || os.Args[1] != "--acknowledge-development" {
		return errors.New("run as urbit-runner with --acknowledge-development")
	}
	host, err := runtimequalify.OpenManagedHost()
	if err != nil {
		return err
	}
	defer host.Close()
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return err
	}
	state := runtimequalify.ManagedState{ID: hex.EncodeToString(nonce[:]), Port: 18092, Fake: &runtimequalify.ManagedFake{Ship: "~zod"}, Storage: &pierstore.Location{Path: "fakeships/layout20-acceptance/pier"}}
	receipt := filepath.Join(host.Directory, "layout20-validation.json")
	f, err := os.OpenFile(receipt, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	f.Close()
	stage := ""
	persist := func(s runtimequalify.ManagedState) error {
		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return err
		}
		if s.Runtime.Stage != stage {
			stage = s.Runtime.Stage
			fmt.Println("Acceptance:", stage)
		}
		return os.WriteFile(receipt, data, 0600)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	state, obs, err := runtimequalify.ApplyManaged(ctx, "start", state, persist)
	if err != nil {
		return err
	}
	if obs.Health != "healthy" || !state.Initialized || state.Storage.Identity.Inode == 0 {
		return errors.New("direct bootstrap did not become healthy")
	}
	var bootID string
	for _, c := range state.Runtime.Containers {
		if c.Kind == "http-boot" && !c.Removed {
			bootID = c.ID
		}
		if c.Kind != "probe" && c.Kind != "http-boot" {
			return errors.New("initial boot created a replacement runtime")
		}
	}
	if bootID == "" {
		return errors.New("initial boot container did not remain running")
	}
	identity := *state.Storage
	for n := 0; n < 3; n++ {
		time.Sleep(2 * time.Second)
		state, obs, err = runtimequalify.ObserveManaged(ctx, state)
		if err != nil || obs.Health != "healthy" || obs.Resources == nil || obs.Resources.ContainerID != bootID {
			return errors.New("initial boot changed its runtime")
		}
	}
	fmt.Println("PASS: first boot is healthy in its permanent directory with the original container", bootID)
	state, _, err = runtimequalify.ApplyManaged(ctx, "stop", state, persist)
	if err != nil {
		return err
	}
	state, obs, err = runtimequalify.ApplyManaged(ctx, "start", state, persist)
	if err != nil {
		return err
	}
	if obs.Health != "healthy" || *state.Storage != identity {
		return errors.New("explicit resume changed the permanent location")
	}
	for _, c := range state.Runtime.Containers {
		if c.Kind == "http-boot" {
			return errors.New("explicit resume repeated bootstrap")
		}
	}
	fmt.Println("PASS: explicit stop/resume preserves the mount root and pier identities")
	state, _, err = runtimequalify.ApplyManaged(ctx, "stop", state, persist)
	if err != nil {
		return err
	}
	for _, c := range state.Runtime.Containers {
		if !c.Removed || c.CreateUncertain {
			return errors.New("acceptance container remains unresolved")
		}
	}
	// The managed driver retains its dedicated bridge for future starts. This
	// temporary acceptance has no future starts: remove only its authenticated bridge.
	n := state.Runtime.Network
	args := []string{"--host", "unix:///run/user/959/urbit-docker.sock", "--config", filepath.Join(state.Runtime.Directory, "docker-config")}
	cmd := exec.CommandContext(ctx, "/usr/bin/docker", append(args, "network", "inspect", n.ID)...)
	cmd.Env = []string{"HOME=" + rootless.HomeDir, "PATH=/usr/bin:/bin"}
	data, err := cmd.Output()
	if err != nil {
		return err
	}
	var networks []struct {
		ID         string `json:"Id"`
		Name       string
		Labels     map[string]string
		Containers map[string]json.RawMessage
	}
	if json.Unmarshal(data, &networks) != nil || len(networks) != 1 || networks[0].ID != n.ID || networks[0].Name != n.Name || networks[0].Labels["org.urbit-runner.managed-fake.run"] != state.ID || len(networks[0].Containers) != 0 {
		return errors.New("acceptance bridge ownership changed")
	}
	cmd = exec.CommandContext(ctx, "/usr/bin/docker", append(args, "network", "rm", n.ID)...)
	cmd.Env = []string{"HOME=" + rootless.HomeDir, "PATH=/usr/bin:/bin"}
	if err = cmd.Run(); err != nil {
		return err
	}
	n.Removed = true
	archive := filepath.Join(pierstore.Root(rootless.HomeDir), "archive/layout20-acceptance/pier")
	if err = pierstore.Move(filepath.Join(pierstore.Root(rootless.HomeDir), state.Storage.Path), archive); err != nil {
		return err
	}
	if err = persist(state); err != nil {
		return err
	}
	fmt.Println("PASS: acceptance container/bridge removed; stopped test pier archived at", archive)
	fmt.Println("Receipt:", receipt)
	return nil
}
