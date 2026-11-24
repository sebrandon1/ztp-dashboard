package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

func TestOriginAllowed(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		host    string
		allowed []string
		want    bool
	}{
		{name: "no origin header", origin: "", host: "dashboard.example.com", want: true},
		{name: "same host", origin: "https://dashboard.example.com", host: "dashboard.example.com", want: true},
		{name: "same host different scheme", origin: "http://dashboard.example.com", host: "dashboard.example.com", want: true},
		{name: "allowlisted origin", origin: "http://localhost:5173", host: "localhost:8080", allowed: []string{"http://localhost:5173"}, want: true},
		{name: "allowlist ignores case", origin: "https://Dash.Example.com", host: "dashboard.example.com", allowed: []string{"https://dash.example.com"}, want: true},
		{name: "foreign origin", origin: "https://evil.example", host: "dashboard.example.com", want: false},
		{name: "foreign origin not on allowlist", origin: "https://evil.example", host: "dashboard.example.com", allowed: []string{"https://other.example"}, want: false},
		{name: "malformed origin", origin: "not-a-url", host: "dashboard.example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://"+tt.host+"/api/status", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if got := originAllowed(req, tt.allowed); got != tt.want {
				t.Errorf("originAllowed(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		allowed    []string
		wantStatus int
		wantOrigin string
	}{
		{
			name:       "same-origin preflight",
			origin:     "http://dashboard.example.com",
			wantStatus: http.StatusNoContent,
			wantOrigin: "http://dashboard.example.com",
		},
		{
			name:       "allowlisted dev origin",
			origin:     "http://localhost:5173",
			allowed:    []string{"http://localhost:5173"},
			wantStatus: http.StatusNoContent,
			wantOrigin: "http://localhost:5173",
		},
		{
			name:       "foreign origin rejected",
			origin:     "https://evil.example",
			wantStatus: http.StatusForbidden,
			wantOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := corsMiddleware(tt.allowed)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodOptions, "http://dashboard.example.com/api/clusters/x", nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", http.MethodDelete)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("Allow-Origin = %q, want %q", got, tt.wantOrigin)
			}
			if tt.wantStatus == http.StatusNoContent && rec.Header().Get("Vary") != "Origin" {
				t.Errorf("Vary = %q, want Origin", rec.Header().Get("Vary"))
			}
		})
	}
}

func TestCORSRejectsForeignActualRequest(t *testing.T) {
	handler := corsMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("handler must not run for a foreign origin")
	}))
	req := httptest.NewRequest(http.MethodDelete, "http://dashboard.example.com/api/clusters/x", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want empty", got)
	}
}

func TestWebSocketOriginCheck(t *testing.T) {
	s := newTestServer(t)
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	wsURL := "ws" + srv.URL[len("http"):] + "/ws/watch"

	if _, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": {"https://evil.example"}}); err == nil {
		t.Fatal("dial with foreign origin succeeded, want rejection")
	} else if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Errorf("foreign-origin handshake status = %v, want 403", resp)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": {srv.URL}})
	if err != nil {
		t.Fatalf("dial with same-host origin failed: %v", err)
	}
	_ = conn.Close()
}
