package omabittheme

import (
	"context"
	"os"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

func TestLiveDevelopmentTheme(t *testing.T) {
	if os.Getenv("OMABIT_TEST_THEME_FAKE") != "1" {
		t.Skip("opt in to publication on the known panel-check development fake")
	}
	c, err := eyre.New("http://127.0.0.1:18085", "~zod")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := c.Login(ctx, []byte("lidlut-tabwed-pillex-ridrup")); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Logout(context.Background()); err != nil {
			t.Error("development session cleanup failed")
		}
	}()
	p, err := Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cleanup, err := Publish(ctx, c, p); err != nil || !cleanup {
		t.Fatalf("publication/cleanup: %v %t", err, cleanup)
	}
	t.Log("PASS: development fake Talon theme/accent publication, Gall acknowledgements, full readback, and channel cleanup")
}
