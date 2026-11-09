package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"subnet-scanner/internal/scanner"
)

type fakePinger func(netip.Addr) (bool, error)

func (f fakePinger) Ping(_ context.Context, ip netip.Addr) (bool, error) { return f(ip) }

var allDead = fakePinger(func(netip.Addr) (bool, error) { return false, nil })

func testConfig(t *testing.T, env map[string]string) Config {
	t.Helper()
	cfg, err := LoadConfig(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestLoadConfig(t *testing.T) {
	cfg := testConfig(t, nil)
	if cfg.Port != "8080" || cfg.BaseURL != "http://localhost:8080" ||
		cfg.DefaultSubnet != "192.168.1.0/24" || cfg.Icon != "" {
		t.Errorf("defaults: %+v", cfg)
	}
	cfg = testConfig(t, map[string]string{"PORT": "9000", "DEFAULT_SUBNET": "10.1.2.0/28", "ICON": "/i.ico"})
	if cfg.Port != "9000" || cfg.DefaultSubnet != "10.1.2.0/28" || cfg.Icon != "/i.ico" {
		t.Errorf("custom: %+v", cfg)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"bad subnet":   {"DEFAULT_SUBNET": "nope"},
		"too wide":     {"DEFAULT_SUBNET": "10.0.0.0/16"},
		"ipv6":         {"DEFAULT_SUBNET": "fe80::/64"},
		"bad loglevel": {"LOG_LEVEL": "loud"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadConfig(func(k string) string { return env[k] }); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestHealth(t *testing.T) {
	h := New(testConfig(t, nil), nil, allDead)
	for _, p := range []string{"/healthz", "/readyz"} {
		if rec := do(h, http.MethodGet, p, ""); rec.Code != http.StatusOK {
			t.Errorf("%s: %d", p, rec.Code)
		}
	}
}

func TestInfo(t *testing.T) {
	for name, tt := range map[string]struct {
		env     map[string]string
		subnet  string
		hasIcon bool
	}{
		"default": {nil, "192.168.1.0/24", false},
		"custom":  {map[string]string{"DEFAULT_SUBNET": "10.0.0.0/24", "ICON": "/x.ico"}, "10.0.0.0/24", true},
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(New(testConfig(t, tt.env), nil, allDead), http.MethodGet, "/api/info", "")
			var got map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got["default_subnet"] != tt.subnet || got["has_icon"] != tt.hasIcon || got["base_url"] != "http://localhost:8080" {
				t.Errorf("got %v", got)
			}
		})
	}
}

func TestScan(t *testing.T) {
	h := New(testConfig(t, nil), nil, fakePinger(func(ip netip.Addr) (bool, error) {
		return ip.As4()[3] == 2, nil
	}))
	rec := do(h, http.MethodPost, "/api/scan", `{"subnet":"10.0.0.0/30"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d: %s", rec.Code, rec.Body)
	}
	var got []scanner.Result
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != (scanner.Result{IP: "10.0.0.1"}) || got[1] != (scanner.Result{IP: "10.0.0.2", Alive: true}) {
		t.Errorf("got %+v", got)
	}
}

func TestScanErrors(t *testing.T) {
	h := New(testConfig(t, nil), nil, allDead)
	for name, tt := range map[string]struct {
		body string
		want string
	}{
		"too wide": {`{"subnet":"10.0.0.0/16"}`, "/24"},
		"ipv6":     {`{"subnet":"fe80::/64"}`, "IPv4"},
		"garbage":  {`{"subnet":"x"}`, "IPv4"},
		"not json": {`nope`, "invalid request body"},
		"too big":  {`{"subnet":"` + strings.Repeat("1", 2048) + `"}`, "invalid request body"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(h, http.MethodPost, "/api/scan", tt.body)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("%d %q", rec.Code, rec.Body)
			}
		})
	}
	if rec := do(h, http.MethodGet, "/api/scan", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: %d", rec.Code)
	}
	if rec := do(h, http.MethodPost, "/scan", `{"subnet":"10.0.0.0/30"}`); rec.Code == http.StatusOK {
		t.Error("/scan alias must not exist")
	}
}

func TestScanPingerErrors(t *testing.T) {
	denied := New(testConfig(t, nil), nil, fakePinger(func(netip.Addr) (bool, error) {
		return false, scanner.ErrNotPermitted
	}))
	rec := do(denied, http.MethodPost, "/api/scan", `{"subnet":"10.0.0.0/30"}`)
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "ping_group_range") {
		t.Errorf("503 expected: %d %q", rec.Code, rec.Body)
	}
	other := New(testConfig(t, nil), nil, fakePinger(func(netip.Addr) (bool, error) {
		return false, errors.New("boom")
	}))
	if rec := do(other, http.MethodPost, "/api/scan", `{"subnet":"10.0.0.0/30"}`); rec.Code != http.StatusInternalServerError {
		t.Errorf("500 expected: %d", rec.Code)
	}
}

func TestFavicon(t *testing.T) {
	icon := filepath.Join(t.TempDir(), "icon.ico")
	if err := os.WriteFile(icon, []byte("ICON"), 0o600); err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{"index.html": {Data: []byte("<h1>app</h1>")}}
	for name, tt := range map[string]struct {
		icon string
		code int
		body string
	}{
		"set":     {icon, http.StatusOK, "ICON"},
		"unset":   {"", http.StatusNotFound, ""},
		"missing": {icon + ".gone", http.StatusNotFound, ""},
	} {
		t.Run(name, func(t *testing.T) {
			h := New(testConfig(t, map[string]string{"ICON": tt.icon}), static, allDead)
			rec := do(h, http.MethodGet, "/favicon.ico", "")
			if rec.Code != tt.code || (tt.body != "" && rec.Body.String() != tt.body) {
				t.Errorf("%d %q", rec.Code, rec.Body)
			}
		})
	}
}

func TestSPAFallback(t *testing.T) {
	static := fstest.MapFS{"index.html": {Data: []byte("<h1>app</h1>")}, "app.js": {Data: []byte("js")}}
	h := New(testConfig(t, nil), static, allDead)
	for path, want := range map[string]string{"/": "<h1>app</h1>", "/settings": "<h1>app</h1>", "/app.js": "js"} {
		rec := do(h, http.MethodGet, path, "")
		if rec.Code != http.StatusOK || rec.Body.String() != want {
			t.Errorf("%s: %d %q", path, rec.Code, rec.Body.String())
		}
	}
}

func TestAPIRoutesWithSPA(t *testing.T) {
	static := fstest.MapFS{"index.html": {Data: []byte("<h1>app</h1>")}}
	h := New(testConfig(t, nil), static, allDead)
	for _, tt := range []struct {
		method, path string
		code         int
	}{
		{http.MethodGet, "/api/scan", http.StatusMethodNotAllowed},
		{http.MethodPut, "/api/scan", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/info", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/nope", http.StatusNotFound},
		{http.MethodPost, "/api/scan", http.StatusBadRequest}, // empty body reaches the handler
		{http.MethodGet, "/api/info", http.StatusOK},
	} {
		rec := do(h, tt.method, tt.path, "")
		if rec.Code != tt.code || strings.Contains(rec.Body.String(), "<h1>") {
			t.Errorf("%s %s: %d %q", tt.method, tt.path, rec.Code, rec.Body)
		}
	}
	if rec := do(h, http.MethodGet, "/api/scan", ""); rec.Header().Get("Allow") != "POST" {
		t.Errorf("Allow = %q", rec.Header().Get("Allow"))
	}
}
