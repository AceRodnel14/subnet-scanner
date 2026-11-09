// Command app serves the HTTP API, and the embedded web UI when built from the
// app-web template. Plain HTTP only: a reverse proxy terminates TLS.
package main

import (
	"cmp"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subnet-scanner/internal/scanner"
	"subnet-scanner/internal/server"
)

// static is the embedded web UI; nil for a backend-only app (web.go sets it).
var static fs.FS

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		slog.Error("exit", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := server.LoadConfig(os.Getenv)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: server.New(cfg, static, scanner.ICMPPinger{}),
		// Cancel in-flight scans on SIGTERM so shutdown is not held up by a 15-18 s scan.
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second, // an empty /24 scan takes ~15-18 s
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr, "base_url", cfg.BaseURL)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// healthcheck backs the image's HEALTHCHECK (distroless has no curl).
func healthcheck() int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + cmp.Or(os.Getenv("PORT"), "8080") + "/healthz")
	if err != nil {
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
