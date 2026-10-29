// Qualification-only owner of the actual immutable application receiver adapter.
// This does not issue a session or access a credential store.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
)

func main() {
	var guard map[string]string
	marker, err := os.ReadFile("/lab/marker")
	ns, nerr := os.ReadFile("/lab/host-namespaces.json")
	if err != nil || string(marker) != "private-receiver-v1\n" || nerr != nil || json.Unmarshal(ns, &guard) != nil || os.Getenv("HOME") != "/lab/home" || len(os.Args) != 2 || !regexp.MustCompile(`^(normal|cancel|crash)$`).MatchString(os.Args[1]) {
		fmt.Fprintln(os.Stderr, "private receiver namespace required")
		os.Exit(2)
	}
	for _, name := range []string{"pid", "net", "mnt", "user"} {
		got, err := os.Readlink("/proc/self/ns/" + name)
		if err != nil || guard[name] == "" || got == guard[name] {
			os.Exit(2)
		}
	}
	if _, err := os.Stat("/home"); !os.IsNotExist(err) {
		os.Exit(2)
	}
	if _, err := os.Stat("/run/user"); !os.IsNotExist(err) {
		os.Exit(2)
	}
	root := filepath.Join("/lab/cases", os.Args[1])
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	a, err := applicationauth.OpenReceiver(filepath.Join(root, "artifact"), "receiver.py", []string{"manifest.json", "application-profile.json", "receiver.py"}, map[string]string{"HOME": root, "XDG_STATE_HOME": filepath.Join(root, "state")})
	if err != nil {
		fmt.Fprintln(os.Stderr, "fixture artifact refused")
		os.Exit(3)
	}
	output, err := a.Deliver(ctx, []byte(`{}`))
	clear(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, "receiver operation ended without delivery")
		os.Exit(1)
	}
	fmt.Println("receiver operation completed")
}
