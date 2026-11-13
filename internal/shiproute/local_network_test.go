package shiproute

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestLocalTailnetUsesOnlyCollectorStatus(t *testing.T) {
	for _, kind := range []string{"ok", "unavailable", "invalid", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "canceled" {
				cancel()
			}
			calls := 0
			got := localTailnet(ctx, func(ctx context.Context, args ...string) ([]byte, error) {
				calls++
				if !reflect.DeepEqual(args, []string{"status", "--json", "--peers=false"}) {
					t.Fatal(args)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("unbounded read")
				}
				if kind == "unavailable" {
					return nil, errors.New("PRIVATE")
				}
				if kind == "invalid" {
					return []byte(`{"BackendState":"Stopped"}`), nil
				}
				return []byte(serveStatus), nil
			})
			if kind == "ok" && got != "example.ts.net" || kind != "ok" && got != "" || kind == "canceled" && calls != 0 {
				t.Fatal(got, calls)
			}
		})
	}
}
