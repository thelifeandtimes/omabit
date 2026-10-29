// A disposable HTTP identity responder for the no-NIC guest's network tests.
// It contains no Urbit credentials, pier or management operation.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

func main() {
	name := flag.String("name", "", "fixture identity")
	port := flag.Int("port", 0, "internal HTTP port")
	flag.Parse()
	if flag.NArg() != 0 || !regexp.MustCompile(`^[ab]-(one|two)-(initial|recovered)$`).MatchString(*name) || (*port != 8081 && *port != 8082) {
		fmt.Fprintln(os.Stderr, "invalid endpoint fixture binding")
		os.Exit(2)
	}
	listener, err := net.Listen("tcp4", "0.0.0.0:"+strconv.Itoa(*port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "fixture listen failed")
		os.Exit(1)
	}
	server := &http.Server{ReadHeaderTimeout: 3 * time.Second, IdleTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, MaxHeaderBytes: 4096, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/identity" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"fixture": *name, "port": *port})
	})}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		if err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "fixture serve failed")
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdown, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		if err := server.Shutdown(shutdown); err != nil {
			fmt.Fprintln(os.Stderr, "fixture shutdown failed")
			os.Exit(1)
		}
		if err := <-done; err != http.ErrServerClosed {
			os.Exit(1)
		}
	}
}
