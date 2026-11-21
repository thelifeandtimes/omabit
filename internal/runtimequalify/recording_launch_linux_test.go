//go:build linux && amd64

package runtimequalify

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func recordingLaunchFixture(t *testing.T, home string) (ManagedRecordingJob, RecordingExecutable) {
	t.Helper()
	job, _, _, _ := recordingJobFixture(t)
	job.State.Runtime.Directory = filepath.Join(home, ".local/share/urbit-runner/managed", job.State.ID)
	job.State.Runtime.PillPath = filepath.Join(job.State.Runtime.Directory, filepath.Base(job.State.Runtime.PillPath))
	image := RecordingExecutable{Path: filepath.Join(home, ".local/share/urbit-runner/recording-images", job.ExecutableSHA256+".runner"), SHA256: job.ExecutableSHA256}
	return job, image
}

func TestRecordingLaunchRetainsLiteralArgumentsAndExplicitSelectedNamespace(t *testing.T) {
	for _, development := range []string{"", "/srv/dev/urbit", "/srv/dev/urbit-040", "/srv/dev/urbit-review-2"} {
		home := "/fixture/space owner/$literal"
		if development != "" {
			home = filepath.Join(development, "state")
		}
		job, image := recordingLaunchFixture(t, home)
		args, err := recordingWorkerArguments(home, development, job, image)
		if err != nil {
			t.Fatal("valid selected namespace refused", development, err)
		}
		separator := slices.Index(args, "--")
		if separator < 0 || args[separator+1] != image.Path || !slices.Contains(args[:separator], "--expand-environment=no") {
			t.Fatal("literal image argument could expand or split")
		}
		command := args[separator+2:]
		if development != "" {
			if len(command) < 2 || command[0] != "--development-root" || command[1] != development {
				t.Fatal("worker fell back to installed namespace")
			}
			command = command[2:]
		}
		raw, _ := job.Bytes()
		if !slices.Equal(command, []string{"recording-worker", "--job", job.ID, "--sha256", recordingJobSHA(raw)}) {
			t.Fatal("job selection changed")
		}
		for _, required := range []string{"--property=Type=notify", "--property=NotifyAccess=main", "--property=Restart=no", "--property=MemoryMax=256M", "--property=MemorySwapMax=0", "--property=CPUQuota=50%", "--property=TasksMax=64", "--property=StandardInput=null", "--property=StandardOutput=null", "--property=StandardError=null"} {
			if !slices.Contains(args[:separator], required) {
				t.Fatal("lost service readiness/resource/output boundary", required)
			}
		}
		for _, forbidden := range []string{"--scope", "--shell", "--wait", "--pipe", "--pty", "--no-block", "--slice-inherit"} {
			if slices.Contains(args[:separator], forbidden) {
				t.Fatal("launcher regained lifetime/terminal ownership", forbidden)
			}
		}
		for _, arg := range args[:separator] {
			if strings.Contains(arg, "PartOf=") || strings.Contains(arg, "BindsTo=") || strings.Contains(arg, job.Actor) {
				t.Fatal("worker lifetime or private actor attached to manager")
			}
		}
	}
}

func TestRecordingLaunchRefusesDifferentImagesRootsOrManagerNamespace(t *testing.T) {
	for _, scenario := range []string{"image-digest", "image-path", "runtime-path", "development-root", "development-home", "specifier", "control", "noncanonical-home"} {
		t.Run(scenario, func(t *testing.T) {
			home, development := "/fixture/owner", ""
			job, image := recordingLaunchFixture(t, home)
			switch scenario {
			case "image-digest":
				image.SHA256 = strings.Repeat("f", 64)
			case "image-path":
				image.Path = "/other/runner"
			case "runtime-path":
				job.State.Runtime.Directory += "-other"
				job.State.Runtime.PillPath = filepath.Join(job.State.Runtime.Directory, filepath.Base(job.State.Runtime.PillPath))
			case "development-root":
				development = "/tmp/fake-root"
			case "development-home":
				development = "/srv/dev/urbit-040"
			case "specifier":
				home = "/fixture/%h"
				job, image = recordingLaunchFixture(t, home)
			case "control":
				home = "/fixture/owner\nprivate"
			case "noncanonical-home":
				home = "/fixture/a/../owner"
			}
			if _, err := recordingWorkerArguments(home, development, job, image); err == nil {
				t.Fatal("rebound or unsafe launch admitted", scenario)
			}
		})
	}
}
