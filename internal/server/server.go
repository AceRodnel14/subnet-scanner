// Package server holds the app's HTTP handlers.
package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"subnet-scanner/internal/scanner"
)

// maxBody caps the /api/scan request body.
const maxBody = 1 << 10

// New returns the app's handler: health endpoints, the API under /api/, the
// favicon, and the web UI from static (nil for a backend-only app).
func New(cfg Config, static fs.FS, pinger scanner.Pinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/info", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"base_url":       cfg.BaseURL,
			"default_subnet": cfg.DefaultSubnet,
			"has_icon":       cfg.Icon != "",
		})
	})
	mux.HandleFunc("POST /api/scan", scanHandler(pinger))
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Icon == "" {
			http.NotFound(w, r)
			return
		}
		if st, err := os.Stat(cfg.Icon); err != nil || st.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, cfg.Icon)
	})
	// Any other method on a known route is 405, and unknown /api/ paths are 404, never
	// the SPA: the "/" catch-all below matches every method and would hide ServeMux's 405.
	mux.HandleFunc("/api/scan", methodNotAllowed(http.MethodPost))
	mux.HandleFunc("/api/info", methodNotAllowed(http.MethodGet))
	mux.HandleFunc("/api/", http.NotFound)
	if static != nil {
		mux.Handle("/", spa(static))
	}
	return mux
}

func scanHandler(pinger scanner.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Subnet string `json:"subnet"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: expected {\"subnet\":\"a.b.c.d/n\"}", http.StatusBadRequest)
			return
		}
		prefix, err := scanner.ParseSubnet(req.Subnet)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		slog.Info("scan start", "subnet", prefix.String(), "hosts", len(scanner.Hosts(prefix)))
		results, err := scanner.Scan(r.Context(), prefix, pinger)
		switch {
		case errors.Is(err, scanner.ErrNotPermitted):
			slog.Error("scan failed", "subnet", prefix.String(), "err", err)
			http.Error(w, scanner.ErrNotPermitted.Error(), http.StatusServiceUnavailable)
			return
		case r.Context().Err() != nil:
			slog.Info("scan canceled", "subnet", prefix.String())
			return
		case err != nil:
			slog.Error("scan failed", "subnet", prefix.String(), "err", err)
			http.Error(w, "scan failed", http.StatusInternalServerError)
			return
		}
		alive := 0
		for _, res := range results {
			if res.Alive {
				alive++
			}
		}
		slog.Info("scan done", "subnet", prefix.String(), "hosts", len(results), "alive", alive)
		writeJSON(w, results)
	}
}

// spa serves the built UI; unknown paths get index.html so client-side routes work.
func spa(static fs.FS) http.Handler {
	files := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "."
		}
		if _, err := fs.Stat(static, name); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}

func methodNotAllowed(allow string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allow)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
