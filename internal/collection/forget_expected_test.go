package collection

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
)

func TestCredentialForgetPreservesConcurrentRebindAndIsIdempotent(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "registry"))
	if err != nil {
		t.Fatal(err)
	}
	original := Connection{Name: "host", RunnerID: "runner-host", Mode: "development-managed", Revision: 1, EnrolledAt: time.Now().UTC(), Transport: "ssh", SSH: &client.SSHConfig{Host: "dedicated-key", ConfigFile: filepath.Join(t.TempDir(), "ssh_config")}}
	changed := original
	changed.Revision = 2
	changed.SSH = &client.SSHConfig{Host: "another-key", ConfigFile: original.SSH.ConfigFile}
	if err = r.write(registryFile{Version: 1, Connections: []Connection{changed}}); err != nil {
		t.Fatal(err)
	}
	if err = r.ForgetSSHExpected(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	rows, err := r.List(context.Background())
	if err != nil || len(rows) != 1 || rows[0].SSH.Host != "another-key" {
		t.Fatal("lost concurrent credential change", rows, err)
	}
	if err = r.ForgetSSHExpected(context.Background(), changed); err != nil {
		t.Fatal(err)
	}
	if err = r.ForgetSSHExpected(context.Background(), changed); err != nil {
		t.Fatal("repeat", err)
	}
	rows, err = r.List(context.Background())
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
}
