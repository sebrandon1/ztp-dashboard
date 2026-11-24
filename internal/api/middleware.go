package api

import (
	"bufio"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// originAllowed reports whether a request from r may call the API or open the
// WebSocket. Requests without an Origin header (curl, same-origin navigations)
// are allowed, as are origins whose host matches the request host and origins
// listed in the allowlist. The SPA is served from the same origin, so a normal
// deployment needs no allowlist entries.
func originAllowed(r *http.Request, allowed []string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	for _, entry := range allowed {
		if strings.EqualFold(entry, origin) {
			return true
		}
	}
	return false
}

// corsMiddleware answers preflights and sets CORS headers only for allowed
// origins. Cross-origin requests from anywhere else are rejected with 403.
func corsMiddleware(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if !originAllowed(r, allowed) {
					writeError(w, http.StatusForbidden, "origin not allowed")
					return
				}
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// authMiddleware requires a bearer token on API and WebSocket routes when token
// is non-empty. Health probes and the embedded SPA stay public. WebSocket
// clients, which cannot set headers, may pass the token as a query parameter.
func authMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if token == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requiresAuth(r.URL.Path) || tokenMatches(r, token) {
				next.ServeHTTP(w, r)
				return
			}
			writeError(w, http.StatusUnauthorized, "unauthorized")
		})
	}
}

// requiresAuth reports whether path is an API or WebSocket route.
func requiresAuth(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/") ||
		path == "/ws" || strings.HasPrefix(path, "/ws/")
}

// tokenMatches reports whether the request carries the expected token in the
// Authorization header, or, for WebSocket routes, in the token query parameter.
func tokenMatches(r *http.Request, token string) bool {
	const bearer = "Bearer "
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, bearer) {
		if constantTimeEqual(strings.TrimPrefix(header, bearer), token) {
			return true
		}
	}
	if strings.HasPrefix(r.URL.Path, "/ws") {
		if queryToken := r.URL.Query().Get("token"); queryToken != "" {
			return constantTimeEqual(queryToken, token)
		}
	}
	return false
}

func constantTimeEqual(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		slog.Debug("http request", "method", r.Method, "path", r.URL.Path, "status", wrapped.statusCode, "duration", time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err, "method", r.Method, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijack: feature not supported")
	}
	return h.Hijack()
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
