//go:build linux && amd64

package managed

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestVMMemberIdentityWarmupRequiresExactBoundedIdentity(t *testing.T) {
	for _, name := range []string{"pass", "wrong-ship", "oversized", "redirect", "unavailable"} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/~/host" || r.Method != "GET" || r.Header.Get("Cookie") != "" {
					t.Error("unexpected fixture request")
				}
				switch name {
				case "wrong-ship":
					fmt.Fprint(w, "~nec")
				case "oversized":
					fmt.Fprint(w, strings.Repeat(" ", 129)+"~zod")
				case "redirect":
					w.Header().Set("Location", "/other")
					w.WriteHeader(302)
				case "unavailable":
					w.WriteHeader(503)
				default:
					fmt.Fprint(w, "~zod")
				}
			}))
			defer server.Close()
			_, port, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
			if err != nil {
				t.Fatal(err)
			}
			n, _ := strconv.Atoi(port)
			var member projectMember
			member.Ship = "~zod"
			member.HTTPPort = uint16(n)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			result := map[string]any{}
			err = vmWaitInitialIdentity(ctx, member, result)
			if name == "pass" {
				if err != nil || result["fixture_identity_ship"] != "~zod" || result["fixture_identity_http_status"] != 200 {
					t.Fatal(result, err)
				}
			} else if err == nil || result["fixture_identity_ship"] != nil {
				t.Fatal("invalid identity accepted", result)
			}
		})
	}
}
