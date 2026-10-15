//go:build ignore

// Live opt-in test of the ordinary |mount -> cp/edit -> |commit workflow.
// Targets only an explicitly selected disposable rc3-* instance. It invokes the
// same hood generators through the existing container-private Lens endpoint;
// no login code, cookie, or enabled conn.sock is needed.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func main() {
	config := flag.String("config-dir", "", "isolated collector registry")
	runner := flag.String("runner", "local", "selected runner")
	rid := flag.String("expected-runner-id", "", "pinned runner")
	name := flag.String("instance", "", "disposable rc3-* instance")
	output := flag.String("output", "", "exclusive evidence file")
	ack := flag.Bool("acknowledge-disposable", false, "authorize test desk mutations")
	flag.Parse()
	if !*ack || !strings.HasPrefix(*name, "rc3-") || *rid == "" || os.Geteuid() == 0 || !filepath.IsAbs(*output) {
		panic("explicit non-root disposable qualification selection required")
	}
	f, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(err)
	f.Close()
	report := map[string]any{"complete": false, "uid": os.Geteuid(), "instance": *name, "started_at": time.Now().UTC(), "phases": []string{}}
	save := func() {
		data, _ := json.MarshalIndent(report, "", "  ")
		must(os.WriteFile(*output, append(data, '\n'), 0600))
	}
	phase := func(s string) { report["phases"] = append(report["phases"].([]string), s); save(); fmt.Println(s) }
	defer func() {
		if e := recover(); e != nil {
			report["error"] = fmt.Sprint(e)
			save()
			panic(e)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	registry, err := collection.Open(*config)
	must(err)
	entry, err := registry.Get(ctx, *runner)
	must(err)
	if entry.RunnerID != *rid {
		panic("runner changed")
	}
	client := entry.Client(true)
	defer client.Close()
	snapshot, err := client.Snapshot(ctx)
	must(err)
	var instance protocol.Instance
	for _, i := range snapshot.Instances {
		if i.ID == *name {
			instance = i
		}
	}
	if instance.ObservedState != "running" || instance.Health != "healthy" || instance.RecoveryRequired || instance.RuntimeStatus == nil {
		panic("ship is not healthy")
	}
	rootID, rel := pierstore.Split(instance.PierPath)
	if rootID == "" {
		rootID = "default"
	}
	var root string
	for _, r := range snapshot.Piers.Roots {
		if r.ID == rootID {
			root = r.Path
		}
	}
	if root == "" {
		panic("root unavailable")
	}
	pier := filepath.Join(root, rel)
	info, err := os.Lstat(pier)
	must(err)
	if info.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) {
		panic("pier not owned by this user")
	}
	desk := filepath.Join(pier, "base")
	backup := filepath.Join(pier, "rc3-export-before-remount")
	if _, err := os.Lstat(desk); !errors.Is(err, os.ErrNotExist) {
		panic("base is already mounted; inspect before repeating this test")
	}
	if _, err := os.Lstat(backup); !errors.Is(err, os.ErrNotExist) {
		panic("prior export backup exists")
	}
	report["command_transport"] = "the same hood generators used by |mount/|commit/|unmount, via Vere's existing container-private loopback Lens endpoint"
	poke := func(command string) {
		if command != "|mount %base" && command != "|commit %base" && command != "|unmount %base" {
			panic("unsupported fixture command")
		}
		payload, _ := json.Marshal(map[string]any{"source": map[string]string{"dojo": "+hood/" + strings.TrimPrefix(command, "|")}, "sink": map[string]string{"app": "hood"}})
		request, stop := context.WithTimeout(ctx, 20*time.Second)
		defer stop()
		cmd := exec.CommandContext(request, "/usr/bin/docker", "--host", fmt.Sprintf("unix:///run/user/%d/urbit-docker.sock", os.Geteuid()), "exec", "-i", instance.RuntimeStatus.ContainerID, "/usr/bin/curl", "--disable", "--noproxy", "*", "--proxy", "", "--silent", "--show-error", "--fail", "--max-time", "15", "--max-filesize", "65536", "--header", "Content-Type: application/json", "--data-binary", "@-", "http://127.0.0.1:12321")
		cmd.Stdin = bytes.NewReader(payload)
		response, err := cmd.CombinedOutput()
		if err != nil {
			panic(fmt.Sprintf("%s: %v %s", command, err, response))
		}
		if len(response) > 65536 || !json.Valid(response) {
			panic("invalid Lens command response")
		}
		phase(command + " hood generator completed: " + strings.TrimSpace(string(response)))
	}
	wait := func(check func() bool) {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			if check() {
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
		panic("filesystem outcome did not settle")
	}
	poke("|mount %base")
	wait(func() bool { s, e := os.Stat(desk); return e == nil && s.IsDir() })
	phase("mounted directory visible to ordinary user")
	fixture, err := os.MkdirTemp(filepath.Dir(*output), "desk-input-")
	must(err)
	defer os.RemoveAll(fixture)
	must(os.Mkdir(filepath.Join(fixture, "owner-proof"), 0700))
	text := "omabit owner-user desk round-trip\n"
	must(os.WriteFile(filepath.Join(fixture, "owner-proof", "message.txt"), []byte(text), 0600))
	cmd := exec.CommandContext(ctx, "/usr/bin/cp", "-R", "--", fixture+"/.", desk+"/")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprintf("ordinary cp: %v %s", err, out))
	}
	// An editor-style atomic replacement, also without privilege or ACL repairs.
	file := filepath.Join(desk, "owner-proof", "message.txt")
	must(os.WriteFile(file+".tmp", []byte(text), 0600))
	must(os.Rename(file+".tmp", file))
	for _, path := range []string{desk, filepath.Dir(file), file} {
		s, e := os.Stat(path)
		must(e)
		if s.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) {
			panic("copied file UID differs")
		}
	}
	phase("ordinary cp, mkdir and atomic editor save completed without sudo")
	poke("|commit %base")
	time.Sleep(5 * time.Second)
	poke("|unmount %base")
	time.Sleep(time.Second)
	if _, err := os.Stat(desk); err == nil {
		must(os.Rename(desk, backup))
	} else if !os.IsNotExist(err) {
		panic(err)
	}
	poke("|mount %base")
	wait(func() bool {
		data, e := os.ReadFile(file)
		return e == nil && strings.TrimRight(string(data), "\n") == strings.TrimRight(text, "\n")
	})
	canonical, err := os.ReadFile(file)
	must(err)
	report["canonical_exported_text"] = string(canonical)
	phase("fresh export from Clay contains committed native files")
	poke("|unmount %base")
	time.Sleep(time.Second)
	if _, e := os.Stat(backup); e == nil {
		must(os.RemoveAll(backup))
	}
	report["pier"] = pier
	report["ship"] = instance.Ship
	report["complete"] = true
	report["finished_at"] = time.Now().UTC()
	save()
	fmt.Println("PASS: same-user live Dojo mount/copy/commit/remount readback")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
