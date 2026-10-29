package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

func TestPrivateChainRegisterRejectsReviewBeforeRegistry(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run(context.Background(), privateArgs(t), &out, &errs); code != 0 {
		t.Fatal(code, errs.String())
	}
	good := bytes.Clone(out.Bytes())
	var view privatechain.Review
	if err := json.Unmarshal(good, &view); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"duplicate", "case-alias", "secret", "fake-mode", "schema", "authority", "reservation", "trailing", "oversized", "symlink", "fifo", "directory"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "plan.json")
			registry := filepath.Join(dir, "unopened")
			data := bytes.TrimSpace(bytes.Clone(good))
			switch kind {
			case "duplicate":
				data = append(data[:len(data)-1], []byte(`,"review_digest":"changed"}`)...)
			case "case-alias":
				data = append(data[:len(data)-1], []byte(`,"Review_Digest":"changed"}`)...)
			case "secret":
				data = append(data[:len(data)-1], []byte(`,"key":"DO-NOT-REFLECT"}`)...)
			case "fake-mode":
				data = bytes.ReplaceAll(data, []byte(privatechain.Mode), []byte("fake-galaxies-v1"))
			case "schema":
				v := view
				v.SchemaVersion++
				data, _ = json.Marshal(v)
			case "authority":
				v := view
				v.RuntimeAuthorized = true
				data, _ = json.Marshal(v)
			case "reservation":
				v := view
				v.ReservationCheck = "performed"
				data, _ = json.Marshal(v)
			case "trailing":
				data = append(data, []byte(` {}`)...)
			case "oversized":
				data = append(data, bytes.Repeat([]byte(" "), privatechain.MaxFoundationBytes)...)
			}
			switch kind {
			case "symlink":
				target := filepath.Join(dir, "target")
				if err := os.WriteFile(target, data, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, file); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := syscall.Mkfifo(file, 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(file, 0700); err != nil {
					t.Fatal(err)
				}
			default:
				if err := os.WriteFile(file, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var output, failures bytes.Buffer
			code := run(context.Background(), []string{"--config-dir", registry, "--runner", "missing", "--expected-runner-id", "runner-a", "private-chain-register", "--plan", file, "--review-digest", view.ReviewDigest, "--request-id", "register-a", "--confirm-reservation"}, &output, &failures)
			if code != 2 || bytes.Contains(failures.Bytes(), []byte("DO-NOT-REFLECT")) {
				t.Fatal(code, failures.String())
			}
			if _, err := os.Stat(registry); !os.IsNotExist(err) {
				t.Fatal("invalid review opened registry", err)
			}
		})
	}
}
