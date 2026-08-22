package httpapi

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PooriChaiya/backend-challenge-a1/internal/port"
)

type ctxKey int

const userIDKey ctxKey = 1

// Logging captures method, path, status, elapsed.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Auth verifies a Bearer token and injects the user id in context.
func Auth(v port.TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			const p = "Bearer "
			if !strings.HasPrefix(h, p) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
				return
			}
			uid, err := v.Verify(strings.TrimPrefix(h, p))
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), userIDKey, uid))
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFrom returns the authenticated user id, if any. ponytail: unused by
// handlers today; kept for future ownership checks.
func UserIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}
